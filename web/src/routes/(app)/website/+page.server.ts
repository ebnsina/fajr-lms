import { redirect, fail } from '@sveltejs/kit';
import type { PageServerLoad, Actions } from './$types';
import { api, ApiFailure } from '$lib/server/api';
import type { SitePage } from '$lib/types.site';

export const load: PageServerLoad = async ({ locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	const { pages } = await api<{ pages: SitePage[] }>('/v1/site/pages', {
		token: locals.token,
		tenant: session.tenant.slug,
		fetch
	});
	return { pages, tenantSlug: session.tenant.slug };
};

export const actions: Actions = {
	create: async ({ request, locals, fetch, cookies }) => {
		const form = await request.formData();
		const title = String(form.get('title') ?? '').trim();
		const slug = String(form.get('slug') ?? '').trim();
		const tenant = cookies.get('fajr_tenant') ?? '';
		if (!title) return fail(422, { message: 'Give the page a title.' });

		try {
			const page = await api<SitePage>('/v1/site/pages', {
				method: 'POST',
				token: locals.token,
				tenant,
				body: { title, slug, blocks: [] },
				fetch
			});
			redirect(303, `/website/${page.id}`);
		} catch (cause) {
			if (cause instanceof ApiFailure) return fail(cause.status, { message: cause.error.message });
			throw cause;
		}
	},

	remove: async ({ request, locals, fetch, cookies }) => {
		const form = await request.formData();
		const id = String(form.get('id') ?? '');
		try {
			await api(`/v1/site/pages/${id}`, {
				method: 'DELETE',
				token: locals.token,
				tenant: cookies.get('fajr_tenant') ?? '',
				fetch
			});
			return { removed: true };
		} catch (cause) {
			if (cause instanceof ApiFailure) return fail(cause.status, { message: cause.error.message });
			throw cause;
		}
	}
};
