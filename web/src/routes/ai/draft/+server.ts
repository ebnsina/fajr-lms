import { json, type RequestHandler } from '@sveltejs/kit';
import { chat } from '@tanstack/ai';
import { z } from 'zod';
import { api, ApiFailure } from '$lib/server/api';
import { aiAdapter, NOT_CONFIGURED } from '$lib/server/ai';

const Draft = z.object({
	questions: z
		.array(
			z.object({
				kind: z.enum(['mcq_single', 'mcq_multi', 'true_false']),
				prompt: z.string().min(1).max(500),
				points: z.number().int().min(1).max(5),
				explanation: z.string().max(400),
				options: z
					.array(z.object({ label: z.string().min(1).max(300), is_correct: z.boolean() }))
					.min(2)
					.max(6)
			})
		)
		.min(1)
		.max(20)
});

type Draft = z.infer<typeof Draft>;
type Question = Draft['questions'][number];

const SYSTEM = [
	'You write quiz questions for a school, from the lesson a teacher has written.',
	'Ask only about what the lesson itself says; never add facts of your own.',
	'Write in the language the lesson is written in.',
	'Every question must be answerable by somebody who read the lesson, and unanswerable by somebody who did not.',
	'mcq_single has exactly one correct option and at least three options.',
	'mcq_multi has at least two correct options and at least three options.',
	'true_false has exactly two options, labelled for true and false in the lesson’s language, one correct.',
	'The explanation says why the answer is right, in one sentence.'
].join(' ');

// A draft that could not be graded is worse than no draft, so anything the
// quiz builder would refuse is dropped here rather than shown to a teacher.
function usable(question: Question): boolean {
	const correct = question.options.filter((option) => option.is_correct).length;
	if (question.kind === 'true_false') return question.options.length === 2 && correct === 1;
	if (question.kind === 'mcq_single') return question.options.length >= 3 && correct === 1;
	return question.options.length >= 3 && correct >= 2;
}

// Every refusal comes back as JSON with a message: the caller is a script on
// the page, never a browser asking for a page.
const no = (status: number, message: string) => json({ message }, { status });

export const POST: RequestHandler = async ({ request, locals, cookies, fetch }) => {
	const tenant = cookies.get('fajr_tenant');
	if (!locals.token || !tenant) return no(401, 'Sign in first.');

	const adapter = aiAdapter();
	if (!adapter) return no(501, NOT_CONFIGURED);

	const { lesson_id: lessonID, count } = await request.json();
	const wanted = Math.min(Math.max(Number(count) || 5, 1), 20);

	let lesson: { title: string; body: string };
	try {
		lesson = await api<{ title: string; body: string }>(`/v1/lessons/${lessonID}`, {
			token: locals.token,
			tenant,
			fetch
		});
	} catch (cause) {
		if (cause instanceof ApiFailure) return no(cause.status, cause.error.message);
		throw cause;
	}
	if ((lesson.body ?? '').trim().length < 200) {
		return no(409, 'There is not enough written in this lesson to ask questions about yet.');
	}

	let drafted: Draft;
	try {
		drafted = (await chat({
			adapter,
			messages: [
				{
					role: 'user',
					content: `Write ${wanted} questions on this lesson.\n\nTitle: ${lesson.title}\n\n${lesson.body}`
				}
			],
			systemPrompts: [SYSTEM],
			outputSchema: Draft
		})) as Draft;
	} catch {
		return no(502, 'Fajr AI could not draft anything just now. Try again.');
	}

	const kept = drafted.questions.filter(usable);
	if (kept.length === 0) {
		return no(502, 'The draft came back in a shape we could not use. Try again.');
	}
	return json({ questions: kept });
};
