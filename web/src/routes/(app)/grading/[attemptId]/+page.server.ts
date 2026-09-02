import { error, fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';

type Option = { id: string; label: string; is_correct: boolean };

type MarkerQuestion = {
	id: string;
	kind: 'mcq_single' | 'mcq_multi' | 'true_false' | 'short_answer' | 'essay';
	prompt: string;
	dir: 'auto' | 'ltr' | 'rtl';
	points: number;
	explanation: string;
	options: Option[];
	correct_option_ids: string[];
	answer_option_ids: string[];
	text_answer: string;
	points_awarded: number | null;
	needs_grading: boolean;
	feedback: string;
};

type Sheet = {
	attempt: {
		id: string;
		attempt_no: number;
		state: string;
		points_awarded: number;
		points_possible: number;
	};
	questions: MarkerQuestion[];
	pending: number;
};

export const load: PageServerLoad = async ({ params, locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	try {
		const sheet = await api<Sheet>(`/v1/attempts/${params.attemptId}/sheet`, {
			token: locals.token,
			tenant: session.tenant.slug,
			fetch
		});
		return { ...sheet, attemptId: params.attemptId };
	} catch (failure) {
		if (failure instanceof ApiFailure && failure.status === 404) {
			error(404, 'That attempt does not exist here.');
		}
		if (failure instanceof ApiFailure && failure.status === 403) {
			error(403, 'Only staff can grade work.');
		}
		throw failure;
	}
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

export const actions: Actions = {
	mark: async ({ params, request, locals, cookies, fetch }) => {
		const form = await request.formData();
		const points = Number(form.get('points_awarded'));
		if (!Number.isFinite(points) || points < 0) {
			return fail(422, { message: 'Give the points this answer earned.' });
		}

		try {
			await api(`/v1/attempts/${params.attemptId}/questions/${form.get('question_id')}/grade`, {
				method: 'PUT',
				body: {
					points_awarded: Math.trunc(points),
					feedback: String(form.get('feedback') ?? '')
				},
				...scoped(locals, cookies, fetch)
			});
		} catch (failure) {
			if (failure instanceof ApiFailure)
				return fail(failure.status, { message: failure.error.message });
			throw failure;
		}
		return { marked: String(form.get('question_id')) };
	},

	release: async ({ params, locals, cookies, fetch }) => {
		try {
			await api(`/v1/attempts/${params.attemptId}/release`, {
				method: 'POST',
				...scoped(locals, cookies, fetch)
			});
		} catch (failure) {
			if (failure instanceof ApiFailure)
				return fail(failure.status, { message: failure.error.message });
			throw failure;
		}
		redirect(303, '/grading');
	}
};
