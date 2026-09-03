import { error, fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';
import type { Lesson } from '$lib/api';

type Quiz = {
	id: string;
	title: string;
	instructions: string;
	time_limit_s: number;
	max_attempts: number;
	pass_percent: number;
	shuffle: boolean;
	draw_count: number | null;
};

type Question = {
	id: string;
	kind: string;
	prompt: string;
	points: number;
	explanation: string;
	options: { id: string; label: string; is_correct: boolean }[];
};

type Assignment = {
	id: string;
	title: string;
	instructions: string;
	points: number;
	due_at: string | null;
	allow_late: boolean;
	late_penalty: number;
	max_files: number;
};

export const load: PageServerLoad = async ({ params, locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');
	if (!['owner', 'admin', 'instructor'].includes(session.tenant.role)) {
		error(403, 'Only staff can set a quiz or an assignment.');
	}

	const scoped = { token: locals.token, tenant: session.tenant.slug, fetch };
	const outline = await api<{ modules: { lessons: Lesson[] }[] }>(
		`/v1/courses/${params.slug}`,
		scoped
	);
	const lesson = outline.modules
		.flatMap((module) => module.lessons)
		.find((row) => row.id === params.lessonId);
	if (!lesson) error(404, 'That lesson is not part of this course.');

	// A lesson has a quiz or an assignment, never both, and often neither yet.
	let quiz: Quiz | null = null;
	let questions: Question[] = [];
	let assignment: Assignment | null = null;

	if (lesson.kind === 'quiz') {
		try {
			const learner = await api<{ quiz: Quiz }>(`/v1/lessons/${lesson.id}/quiz`, scoped);
			const sheet = await api<{ quiz: Quiz; questions: Question[] }>(
				`/v1/quizzes/${learner.quiz.id}`,
				scoped
			);
			quiz = sheet.quiz;
			questions = sheet.questions;
		} catch (cause) {
			if (!(cause instanceof ApiFailure && cause.status === 404)) throw cause;
		}
	} else if (lesson.kind === 'assignment') {
		try {
			const found = await api<{ assignment: Assignment }>(
				`/v1/lessons/${lesson.id}/assignment`,
				scoped
			);
			assignment = found.assignment;
		} catch (cause) {
			if (!(cause instanceof ApiFailure && cause.status === 404)) throw cause;
		}
	}

	return { lesson, quiz, questions, assignment, slug: params.slug };
};

const scoped = (
	locals: App.Locals,
	cookies: { get: (n: string) => string | undefined },
	fetch: typeof globalThis.fetch
) => {
	const tenant = cookies.get('fajr_tenant');
	if (!locals.token || !tenant) redirect(303, '/login');
	return { token: locals.token, tenant, fetch };
};

const failed = (cause: unknown) => {
	if (cause instanceof ApiFailure) return fail(cause.status, { message: cause.error.message });
	throw cause;
};

export const actions: Actions = {
	createQuiz: async ({ params, request, locals, cookies, fetch }) => {
		const form = await request.formData();
		const title = String(form.get('title') ?? '').trim();
		if (!title) return fail(422, { message: 'Give the quiz a title.' });
		try {
			await api(`/v1/lessons/${params.lessonId}/quiz`, {
				method: 'POST',
				body: {
					title,
					instructions: String(form.get('instructions') ?? '').trim(),
					dir: 'auto',
					time_limit_s: Math.round(Number(form.get('minutes') ?? 0) * 60),
					max_attempts: Number(form.get('max_attempts') ?? 1),
					pass_percent: Number(form.get('pass_percent') ?? 50),
					shuffle: form.get('shuffle') === 'on',
					draw_count: Number(form.get('draw_count') ?? 0)
				},
				...scoped(locals, cookies, fetch)
			});
		} catch (cause) {
			return failed(cause);
		}
		return { saved: true };
	},

	addQuestion: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		const kind = String(form.get('kind') ?? 'mcq_single');
		const prompt = String(form.get('prompt') ?? '').trim();
		if (!prompt) return fail(422, { message: 'Ask the question.' });

		// Choices arrive as parallel fields: the label, and whether it is correct.
		const labels = form.getAll('label').map(String);
		const correct = new Set(form.getAll('correct').map(String));
		const options = labels
			.map((label, index) => ({ label: label.trim(), is_correct: correct.has(String(index)) }))
			.filter((option) => option.label !== '');

		try {
			await api(`/v1/quizzes/${form.get('quiz_id')}/questions`, {
				method: 'POST',
				body: {
					kind,
					prompt,
					dir: 'auto',
					points: Number(form.get('points') ?? 1),
					explanation: String(form.get('explanation') ?? '').trim(),
					options
				},
				...scoped(locals, cookies, fetch)
			});
		} catch (cause) {
			return failed(cause);
		}
		return { added: true };
	},

	removeQuestion: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		try {
			await api(`/v1/questions/${form.get('question_id')}`, {
				method: 'DELETE',
				...scoped(locals, cookies, fetch)
			});
		} catch (cause) {
			return failed(cause);
		}
		return { removed: true };
	},

	saveAssignment: async ({ params, request, locals, cookies, fetch }) => {
		const form = await request.formData();
		const title = String(form.get('title') ?? '').trim();
		if (!title) return fail(422, { message: 'Give the assignment a title.' });
		const due = String(form.get('due_at') ?? '');
		// One already set is edited; the first one is created on the lesson.
		const existing = String(form.get('assignment_id') ?? '');

		try {
			await api(
				existing ? `/v1/assignments/${existing}` : `/v1/lessons/${params.lessonId}/assignment`,
				{
					method: existing ? 'PATCH' : 'POST',
					body: {
						title,
						instructions: String(form.get('instructions') ?? '').trim(),
						dir: 'auto',
						points: Number(form.get('points') ?? 100),
						due_at: due ? new Date(due).toISOString() : null,
						allow_late: form.get('allow_late') === 'on',
						late_penalty: Number(form.get('late_penalty') ?? 0),
						max_files: Number(form.get('max_files') ?? 3)
					},
					...scoped(locals, cookies, fetch)
				}
			);
		} catch (cause) {
			return failed(cause);
		}
		return { saved: true };
	}
};
