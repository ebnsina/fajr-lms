import { fail, redirect } from '@sveltejs/kit';
import { dev } from '$app/environment';
import type { Actions, PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';
import { saveSession, saveTenant } from '$lib/server/session';

type Kind = { slug: string; label: string; name: string };

export const load: PageServerLoad = async ({ fetch }) => {
	// The kinds come from the API so the form can never offer a school that
	// has no demo behind it.
	try {
		const { kinds } = await api<{ kinds: Kind[] }>('/v1/demo/kinds', { fetch });
		return { kinds };
	} catch {
		return { kinds: [] as Kind[] };
	}
};

export const actions: Actions = {
	default: async ({ request, cookies, fetch }) => {
		const form = await request.formData();
		const entered = {
			full_name: String(form.get('full_name') ?? '').trim(),
			email: String(form.get('email') ?? '').trim(),
			phone: String(form.get('phone') ?? '').trim(),
			organisation: String(form.get('organisation') ?? '').trim(),
			role: String(form.get('role') ?? '').trim(),
			learners: String(form.get('learners') ?? '').trim(),
			runs: String(form.get('runs') ?? '').trim(),
			note: String(form.get('note') ?? '').trim()
		};

		if (!entered.full_name) return fail(422, { ...entered, message: 'What should we call you?' });
		if (!entered.email) return fail(422, { ...entered, message: 'Where can we reach you?' });
		if (!entered.runs) return fail(422, { ...entered, message: 'Tell us what you run.' });

		let opened: { token: string; tenant: string };
		try {
			opened = await api<{ token: string; tenant: string }>('/v1/demo', {
				method: 'POST',
				body: entered,
				fetch
			});
		} catch (cause) {
			if (cause instanceof ApiFailure) {
				return fail(cause.status, { ...entered, message: cause.error.message });
			}
			throw cause;
		}

		saveSession(cookies, opened.token, !dev);
		saveTenant(cookies, opened.tenant, !dev);
		redirect(303, '/');
	}
};
