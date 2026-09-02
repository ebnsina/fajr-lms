import { error, fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';
import type { SitePage } from '$lib/types.site';

export const load: PageServerLoad = async ({ params, locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	try {
		const page = await api<SitePage>(`/v1/site/pages/${params.id}`, {
			token: locals.token,
			tenant: session.tenant.slug,
			fetch
		});
		return { page, tenantSlug: session.tenant.slug };
	} catch (cause) {
		if (cause instanceof ApiFailure && cause.status === 404) error(404, 'That page is gone.');
		if (cause instanceof ApiFailure && cause.status === 403) {
			error(403, 'Only an owner or admin can build the website.');
		}
		throw cause;
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
	save: async ({ request, params, locals, cookies, fetch }) => {
		const form = await request.formData();
		let blocks: unknown;
		try {
			blocks = JSON.parse(String(form.get('blocks') ?? '[]'));
		} catch {
			return fail(422, {
				message: 'The sections could not be read. Reload and try again.'
			});
		}

		try {
			await api(`/v1/site/pages/${params.id}`, {
				method: 'PATCH',
				body: {
					title: String(form.get('title') ?? '').trim(),
					description: String(form.get('description') ?? '').trim(),
					nav_label: String(form.get('nav_label') ?? '').trim(),
					nav_order: Number(form.get('nav_order') ?? 0),
					dir: String(form.get('dir') ?? 'auto'),
					blocks
				},
				...scoped(locals, cookies, fetch)
			});
		} catch (cause) {
			if (cause instanceof ApiFailure) return fail(cause.status, { message: cause.error.message });
			throw cause;
		}
		return { saved: true };
	},

	status: async ({ request, params, locals, cookies, fetch }) => {
		const form = await request.formData();
		try {
			await api(`/v1/site/pages/${params.id}/status`, {
				method: 'PUT',
				body: { status: String(form.get('status') ?? 'draft') },
				...scoped(locals, cookies, fetch)
			});
		} catch (cause) {
			if (cause instanceof ApiFailure) return fail(cause.status, { message: cause.error.message });
			throw cause;
		}
		return { saved: true };
	}
};
