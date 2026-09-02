import { fail, redirect } from '@sveltejs/kit';
import { dev } from '$app/environment';
import type { Actions, PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';
import { saveTenant } from '$lib/server/session';

type TenantRow = {
	id: string;
	slug: string;
	name: string;
	kind: string;
	role: string;
	default_dir: string;
};

export const load: PageServerLoad = async ({ locals, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { tenants } = await api<{ tenants: TenantRow[] }>('/v1/tenants', {
		token: locals.token,
		fetch
	});
	return { tenants, current: locals.tenantSlug ?? null };
};

export const actions: Actions = {
	default: async ({ request, cookies, locals, fetch }) => {
		const slug = String((await request.formData()).get('slug') ?? '').trim();
		if (!slug) return fail(422, { message: 'Choose a place to work in.' });

		try {
			// Prove the membership before trusting the choice.
			await api('/v1/tenant', { token: locals.token, tenant: slug, fetch });
		} catch (error) {
			if (error instanceof ApiFailure) return fail(error.status, { message: error.error.message });
			throw error;
		}

		saveTenant(cookies, slug, !dev);
		redirect(303, '/');
	}
};
