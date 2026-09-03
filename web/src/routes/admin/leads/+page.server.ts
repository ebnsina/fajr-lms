import { fail } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { admin } from '$lib/server/admin';

export type Lead = {
	id: string;
	full_name: string;
	email: string;
	phone: string;
	organisation: string;
	role: string;
	learners: string;
	runs: string;
	note: string;
	state: string;
	worked_note: string;
	created_at: string;
};

export const load: PageServerLoad = async ({ cookies, fetch, url }) => {
	const state = url.searchParams.get('state') ?? '';
	const query = url.searchParams.get('q') ?? '';
	const search = new URLSearchParams({ limit: '200' });
	if (state) search.set('state', state);
	if (query) search.set('q', query);

	const { leads } = await admin<{ leads: Lead[] }>(`/v1/admin/leads?${search}`, cookies, fetch);
	return { leads, state, query };
};

export const actions: Actions = {
	work: async ({ request, cookies, fetch }) => {
		const form = await request.formData();
		const id = String(form.get('id') ?? '');
		try {
			await admin(`/v1/admin/leads/${id}`, cookies, fetch, {
				method: 'PUT',
				body: { state: String(form.get('state') ?? ''), note: String(form.get('note') ?? '') }
			});
		} catch (cause) {
			if (cause instanceof Error) return fail(422, { message: cause.message });
			throw cause;
		}
		return { worked: true };
	}
};
