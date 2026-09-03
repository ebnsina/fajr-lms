import { fail, redirect } from '@sveltejs/kit';
import { dev } from '$app/environment';
import type { Actions } from './$types';
import { api, ApiFailure } from '$lib/server/api';
import { clearStaff, saveStaff } from '$lib/server/session';

export const actions: Actions = {
	in: async ({ request, cookies, fetch }) => {
		const form = await request.formData();
		const email = String(form.get('email') ?? '').trim();
		const password = String(form.get('password') ?? '');
		if (!email || !password) {
			return fail(422, { email, message: 'Both, please.' });
		}

		let signed: { token: string };
		try {
			signed = await api<{ token: string }>('/v1/admin/login', {
				method: 'POST',
				body: { email, password },
				fetch
			});
		} catch (cause) {
			if (cause instanceof ApiFailure)
				return fail(cause.status, { email, message: cause.error.message });
			throw cause;
		}

		saveStaff(cookies, signed.token, !dev);
		redirect(303, '/admin');
	},

	out: async ({ cookies }) => {
		clearStaff(cookies);
		redirect(303, '/admin/login');
	}
};
