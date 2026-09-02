import { error, fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';
import type { Playback } from '$lib/api';

type Sheet = {
	submission: {
		id: string;
		state: 'draft' | 'submitted' | 'returned';
		body: string;
		is_late: boolean;
		submitted_at: string | null;
		points_awarded: number | null;
		feedback: string;
	};
	assignment: {
		id: string;
		title: string;
		instructions: string;
		dir: 'auto' | 'ltr' | 'rtl';
		points: number;
		due_at: string | null;
		late_penalty: number;
	};
	full_name: string;
	attachments: { media_id: string; kind: string; title: string; state: string; playback?: Playback }[];
};

export const load: PageServerLoad = async ({ params, locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	try {
		const sheet = await api<Sheet>(`/v1/submissions/${params.id}`, {
			token: locals.token,
			tenant: session.tenant.slug,
			fetch
		});
		return sheet;
	} catch (failure) {
		if (failure instanceof ApiFailure && failure.status === 404) {
			error(404, 'That submission does not exist here.');
		}
		if (failure instanceof ApiFailure && failure.status === 403) {
			error(403, 'Only staff can mark work.');
		}
		throw failure;
	}
};

export const actions: Actions = {
	grade: async ({ params, request, locals, cookies, fetch }) => {
		const tenant = cookies.get('fajr_tenant');
		if (!locals.token || !tenant) redirect(303, '/login');

		const form = await request.formData();
		const points = Number(form.get('points_awarded'));
		if (!Number.isFinite(points) || points < 0) {
			return fail(422, { message: 'Give the points this work earned.' });
		}

		try {
			await api(`/v1/submissions/${params.id}/grade`, {
				method: 'POST',
				body: {
					points_awarded: Math.trunc(points),
					feedback: String(form.get('feedback') ?? '')
				},
				token: locals.token,
				tenant,
				fetch
			});
		} catch (failure) {
			if (failure instanceof ApiFailure) return fail(failure.status, { message: failure.error.message });
			throw failure;
		}
		redirect(303, '/submissions');
	}
};
