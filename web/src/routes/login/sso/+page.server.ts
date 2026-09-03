import { redirect } from '@sveltejs/kit';
import { dev } from '$app/environment';
import type { PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';
import { saveSession, saveTenant } from '$lib/server/session';
import type { Membership, User } from '$lib/types';

type SessionResponse = { token: string; user: User; memberships: Membership[] };

// Where the school's provider sends the browser back to.
export const load: PageServerLoad = async ({ url, cookies, fetch }) => {
	const refusal = url.searchParams.get('error_description') ?? url.searchParams.get('error');
	if (refusal) return { problem: refusal };

	const code = url.searchParams.get('code') ?? '';
	const state = url.searchParams.get('state') ?? '';
	if (!code || !state) return { problem: 'That sign-in did not come back complete.' };

	let session: SessionResponse;
	try {
		session = await api<SessionResponse>('/v1/auth/sso/finish', {
			method: 'POST',
			body: { code, state },
			fetch
		});
	} catch (cause) {
		if (cause instanceof ApiFailure) return { problem: cause.error.message };
		throw cause;
	}

	saveSession(cookies, session.token, !dev);
	if (session.memberships.length === 1) {
		try {
			const { tenants } = await api<{ tenants: { id: string; slug: string }[] }>('/v1/tenants', {
				token: session.token,
				fetch
			});
			const slug = tenants.find((row) => row.id === session.memberships[0].tenant_id)?.slug;
			if (slug) saveTenant(cookies, slug, !dev);
		} catch {
			// Choosing the school by hand still works.
		}
	}
	redirect(303, session.memberships.length === 1 ? '/' : '/tenant');
};
