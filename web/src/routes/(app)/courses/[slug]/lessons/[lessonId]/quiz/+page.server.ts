import { error, fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';

type LearnerQuestion = {
	id: string;
	kind: 'mcq_single' | 'mcq_multi' | 'true_false' | 'short_answer' | 'essay';
	prompt: string;
	dir: 'auto' | 'ltr' | 'rtl';
	points: number;
	options: { id: string; label: string }[];
};

type Attempt = {
	id: string;
	attempt_no: number;
	state: 'in_progress' | 'submitted' | 'graded' | 'expired';
	points_awarded: number;
	points_possible: number;
};

type Quiz = {
	id: string;
	title: string;
	instructions: string;
	dir: 'auto' | 'ltr' | 'rtl';
	time_limit_s: number;
	max_attempts: number;
	pass_percent: number;
};

export const load: PageServerLoad = async ({ params, locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	const scoped = { token: locals.token, tenant: session.tenant.slug, fetch };
	let paper: { quiz: Quiz; questions: LearnerQuestion[]; attempts: Attempt[] };
	try {
		paper = await api(`/v1/lessons/${params.lessonId}/quiz`, scoped);
	} catch (failure) {
		if (failure instanceof ApiFailure && failure.status === 404) {
			error(404, 'This lesson has no quiz.');
		}
		throw failure;
	}

	// An attempt already open is resumed rather than started again, so a dropped
	// connection never costs somebody one of their tries.
	const open = paper.attempts.find((a) => a.state === 'in_progress');
	let live: { attempt: Attempt; questions: LearnerQuestion[]; expires_in_s: number; answers: { question_id: string; option_ids: string[]; text: string }[] } | null =
		null;
	if (open) {
		try {
			live = await api(`/v1/attempts/${open.id}`, scoped);
		} catch {
			live = null;
		}
	}

	return {
		quiz: paper.quiz,
		questions: paper.questions,
		attempts: paper.attempts,
		live,
		slug: params.slug,
		lessonId: params.lessonId
	};
};

const scopedFrom = (locals: App.Locals, cookies: { get: (n: string) => string | undefined }, fetch: typeof globalThis.fetch) => {
	const tenant = cookies.get('fajr_tenant');
	if (!locals.token || !tenant) redirect(303, '/login');
	return { token: locals.token, tenant, fetch };
};

export const actions: Actions = {
	start: async ({ request, locals, cookies, fetch }) => {
		const quizID = String((await request.formData()).get('quiz_id') ?? '');
		try {
			await api(`/v1/quizzes/${quizID}/attempts`, {
				method: 'POST',
				...scopedFrom(locals, cookies, fetch)
			});
		} catch (failure) {
			if (failure instanceof ApiFailure) return fail(failure.status, { message: failure.error.message });
			throw failure;
		}
		return { started: true };
	},

	answer: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		const attemptID = String(form.get('attempt_id') ?? '');
		const body = {
			question_id: String(form.get('question_id') ?? ''),
			option_ids: form.getAll('option_ids').map(String).filter(Boolean),
			text: String(form.get('text') ?? '')
		};
		try {
			await api(`/v1/attempts/${attemptID}/answers`, {
				method: 'PUT',
				body,
				...scopedFrom(locals, cookies, fetch)
			});
		} catch (failure) {
			if (failure instanceof ApiFailure) return fail(failure.status, { message: failure.error.message });
			throw failure;
		}
		return { saved: body.question_id };
	},

	submit: async ({ request, locals, cookies, fetch }) => {
		const attemptID = String((await request.formData()).get('attempt_id') ?? '');
		try {
			const result = await api(`/v1/attempts/${attemptID}/submit`, {
				method: 'POST',
				...scopedFrom(locals, cookies, fetch)
			});
			return { result };
		} catch (failure) {
			if (failure instanceof ApiFailure) return fail(failure.status, { message: failure.error.message });
			throw failure;
		}
	}
};
