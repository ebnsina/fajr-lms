import { fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';

export type Collection = {
	id: string;
	kind: 'path' | 'bundle';
	slug: string;
	title: string;
	summary: string;
	status: string;
	price_minor: number;
	currency: string;
	courses: number;
};

export const load: PageServerLoad = async ({ locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	const { collections } = await api<{ collections: Collection[] }>('/v1/collections', {
		token: locals.token,
		tenant: session.tenant.slug,
		fetch
	});
	const role = session.tenant.role;
	return {
		collections: collections ?? [],
		teaches: ['owner', 'admin', 'instructor'].includes(role)
	};
};

export const actions: Actions = {
	create: async ({ request, locals, cookies, fetch }) => {
		const tenant = cookies.get('fajr_tenant');
		if (!locals.token || !tenant) redirect(303, '/login');

		const form = await request.formData();
		const title = String(form.get('title') ?? '').trim();
		if (!title) return fail(422, { message: 'Give it a title.' });
		const kind = String(form.get('kind') ?? 'path');
		const price = Number(form.get('price') ?? 0);

		try {
			const made = await api<Collection>('/v1/collections', {
				method: 'POST',
				body: {
					kind,
					title,
					summary: String(form.get('summary') ?? '').trim(),
					dir: 'auto',
					price_minor: kind === 'bundle' ? Math.round(price * 100) : 0
				},
				token: locals.token,
				tenant,
				fetch
			});
			redirect(303, `/paths/${made.slug}`);
		} catch (cause) {
			if (cause instanceof ApiFailure) return fail(cause.status, { message: cause.error.message });
			throw cause;
		}
	}
};
