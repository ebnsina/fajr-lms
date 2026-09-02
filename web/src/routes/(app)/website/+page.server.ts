import { redirect, fail } from '@sveltejs/kit';
import type { PageServerLoad, Actions } from './$types';
import { api, ApiFailure } from '$lib/server/api';
import type { SitePage } from '$lib/types.site';
import { templates } from '$lib/site-templates';

export const load: PageServerLoad = async ({ locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	const { pages } = await api<{ pages: SitePage[] }>('/v1/site/pages', {
		token: locals.token,
		tenant: session.tenant.slug,
		fetch
	});
	return {
		pages,
		tenantSlug: session.tenant.slug,
		theme: session.tenant.site_theme ?? 'plain'
	};
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

	// A template writes its pages as drafts and dresses the site to match, so
	// nothing goes public until somebody has read it.
	template: async ({ request, locals, fetch, cookies }) => {
		const form = await request.formData();
		const chosen = templates.find((row) => row.id === String(form.get('template') ?? ''));
		if (!chosen) return fail(422, { message: 'That template is not one of ours.' });

		const call = { token: locals.token, tenant: cookies.get('fajr_tenant') ?? '', fetch };
		let made = 0;
		for (const page of chosen.pages) {
			try {
				await api<SitePage>('/v1/site/pages', {
					method: 'POST',
					body: {
						slug: page.slug,
						title: page.title,
						description: page.description,
						nav_label: page.nav_label,
						nav_order: page.nav_order,
						dir: chosen.region === 'gulf' ? 'rtl' : 'auto',
						blocks: page.blocks
					},
					...call
				});
				made++;
			} catch (cause) {
				// An address already taken is the common case; the rest still apply.
				if (cause instanceof ApiFailure && cause.status === 409) continue;
				if (cause instanceof ApiFailure)
					return fail(cause.status, { message: cause.error.message });
				throw cause;
			}
		}
		if (made === 0) {
			return fail(409, {
				message:
					'Those addresses are already taken. Delete the pages first, or pick another template.'
			});
		}

		try {
			await api('/v1/site/theme', { method: 'PUT', body: { theme: chosen.theme }, ...call });
		} catch (cause) {
			if (cause instanceof ApiFailure) return fail(cause.status, { message: cause.error.message });
			throw cause;
		}
		return { made };
	},

	theme: async ({ request, locals, fetch, cookies }) => {
		const form = await request.formData();
		try {
			await api('/v1/site/theme', {
				method: 'PUT',
				token: locals.token,
				tenant: cookies.get('fajr_tenant') ?? '',
				body: { theme: String(form.get('theme') ?? 'plain') },
				fetch
			});
			return { saved: true };
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
