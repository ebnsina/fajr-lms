import { fail, redirect } from '@sveltejs/kit';
import { dev } from '$app/environment';
import type { Actions, PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';
import { saveTenant } from '$lib/server/session';

export const load: PageServerLoad = ({ locals }) => {
	// Opening a school needs an account, so sign in first and come straight back.
	return { signedIn: Boolean(locals.token) };
};

export const actions: Actions = {
	default: async ({ request, locals, cookies, fetch }) => {
		if (!locals.token) redirect(303, '/login?next=/start');

		const form = await request.formData();
		const name = String(form.get('name') ?? '').trim();
		const slug = String(form.get('slug') ?? '').trim();
		const kind = String(form.get('kind') ?? 'institution');
		const dir = String(form.get('dir') ?? 'auto');
		const currency = String(form.get('currency') ?? 'BDT');
		if (!name) return fail(422, { name, message: 'What is the school called?' });

		let created: { slug: string };
		try {
			created = await api<{ slug: string }>('/v1/tenants', {
				method: 'POST',
				token: locals.token,
				body: {
					name,
					slug,
					kind,
					dir,
					currency,
					locale: dir === 'rtl' ? 'ar' : 'en'
				},
				fetch
			});
		} catch (cause) {
			if (cause instanceof ApiFailure) {
				return fail(cause.status, { name, slug, message: cause.error.message });
			}
			throw cause;
		}

		saveTenant(cookies, created.slug, !dev);
		redirect(303, '/');
	}
};
