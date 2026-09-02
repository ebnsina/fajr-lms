import { error, fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';

type Assignment = {
	id: string;
	title: string;
	instructions: string;
	dir: 'auto' | 'ltr' | 'rtl';
	points: number;
	due_at: string | null;
	allow_late: boolean;
	late_penalty: number;
	max_files: number;
};

type Submission = {
	id: string;
	state: 'draft' | 'submitted' | 'returned';
	body: string;
	media_ids: string[];
	is_late: boolean;
	submitted_at: string | null;
	points_awarded: number | null;
	feedback: string;
};

export const load: PageServerLoad = async ({ params, locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	try {
		const data = await api<{
			assignment: Assignment;
			submission: Submission | null;
		}>(`/v1/lessons/${params.lessonId}/assignment`, {
			token: locals.token,
			tenant: session.tenant.slug,
			fetch
		});
		return { ...data, slug: params.slug, lessonId: params.lessonId };
	} catch (failure) {
		if (failure instanceof ApiFailure && failure.status === 404) {
			error(404, 'This lesson has no assignment.');
		}
		throw failure;
	}
};

export const actions: Actions = {
	save: async ({ request, locals, cookies, fetch }) => {
		const tenant = cookies.get('fajr_tenant');
		if (!locals.token || !tenant) redirect(303, '/login');

		const form = await request.formData();
		const body = {
			body: String(form.get('body') ?? ''),
			media_ids: form.getAll('media_ids').map(String).filter(Boolean),
			submit: form.get('submit') === 'true'
		};

		try {
			await api(`/v1/assignments/${form.get('assignment_id')}/submission`, {
				method: 'PUT',
				body,
				token: locals.token,
				tenant,
				fetch
			});
		} catch (failure) {
			if (failure instanceof ApiFailure) {
				return fail(failure.status, { message: failure.error.message });
			}
			throw failure;
		}
		return { saved: true, handedIn: body.submit };
	}
};
