import { fail, redirect } from '@sveltejs/kit';
import { dev } from '$app/environment';
import type { Actions, PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';
import { clearSession, saveSession, saveTenant } from '$lib/server/session';
import type { Membership, User } from '$lib/types';

export const load: PageServerLoad = ({ locals }) => {
	if (locals.token) redirect(303, '/');
	return {};
};

type VerifyResponse = { token: string; user: User; memberships: Membership[] };

export const actions: Actions = {
	// Step one: ask for a code. The answer never says whether the account exists.
	request: async ({ request, fetch }) => {
		const form = await request.formData();
		const destination = String(form.get('destination') ?? '').trim();
		if (!destination) {
			return fail(422, {
				destination,
				message: 'Enter your phone number or email address.'
			});
		}

		try {
			await api('/v1/auth/otp', {
				method: 'POST',
				body: { destination },
				fetch
			});
		} catch (error) {
			if (error instanceof ApiFailure) {
				return fail(error.status, {
					destination,
					message: error.error.message
				});
			}
			throw error;
		}
		return { sent: true, destination };
	},

	verify: async ({ request, cookies, fetch }) => {
		const form = await request.formData();
		const destination = String(form.get('destination') ?? '').trim();
		const code = String(form.get('code') ?? '').trim();
		const fullName = String(form.get('full_name') ?? '').trim();

		if (code.length !== 6) {
			return fail(422, {
				sent: true,
				destination,
				message: 'The code is six digits.'
			});
		}

		let session: VerifyResponse;
		try {
			session = await api<VerifyResponse>('/v1/auth/otp/verify', {
				method: 'POST',
				body: { destination, code, full_name: fullName },
				fetch
			});
		} catch (error) {
			if (error instanceof ApiFailure) {
				return fail(error.status, {
					sent: true,
					destination,
					message: error.error.message
				});
			}
			throw error;
		}

		saveSession(cookies, session.token, !dev);
		// One membership needs no choosing; more than one does.
		if (session.memberships.length === 1) {
			const slug = await slugFor(session.token, session.memberships[0].tenant_id, fetch);
			if (slug) saveTenant(cookies, slug, !dev);
		}
		redirect(303, session.memberships.length === 1 ? '/' : '/tenant');
	},

	logout: async ({ cookies, locals, fetch }) => {
		if (locals.token) {
			try {
				await api('/v1/auth/logout', {
					method: 'POST',
					token: locals.token,
					fetch
				});
			} catch {
				// The cookie is cleared regardless, so a failed call cannot strand a session.
			}
		}
		clearSession(cookies);
		redirect(303, '/login');
	}
};

/** Resolves a tenant id to the slug every other request is scoped by. */
async function slugFor(token: string, tenantID: string, fetch: typeof globalThis.fetch) {
	try {
		const { tenants } = await api<{ tenants: { id: string; slug: string }[] }>('/v1/tenants', {
			token,
			fetch
		});
		return tenants.find((t) => t.id === tenantID)?.slug ?? null;
	} catch {
		return null;
	}
}
