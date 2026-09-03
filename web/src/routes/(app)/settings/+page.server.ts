import { fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';

type Provider = {
	configured: boolean;
	label?: string;
	issuer?: string;
	client_id?: string;
	has_secret?: boolean;
	allowed_domains?: string[];
	join_role?: string;
	auto_join?: boolean;
	enabled?: boolean;
};

const runsTheSchool = (role: string) => role === 'owner' || role === 'admin';

export const load: PageServerLoad = async ({ locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	// Sign-in settings belong to whoever runs the school; everybody else gets
	// the page without them.
	if (!runsTheSchool(session.tenant.role)) return { provider: null };
	try {
		const provider = await api<Provider>('/v1/sso', {
			token: locals.token,
			tenant: session.tenant.slug,
			fetch
		});
		return { provider };
	} catch {
		return { provider: null };
	}
};

const scoped = (
	locals: App.Locals,
	cookies: { get: (name: string) => string | undefined },
	fetch: typeof globalThis.fetch
) => {
	const tenant = cookies.get('fajr_tenant');
	if (!locals.token || !tenant) redirect(303, '/login');
	return { token: locals.token, tenant, fetch };
};

export const actions: Actions = {
	sso: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		const domains = String(form.get('allowed_domains') ?? '')
			.split(/[\s,]+/)
			.map((entry) => entry.trim())
			.filter(Boolean);

		try {
			await api('/v1/sso', {
				method: 'PUT',
				body: {
					label: String(form.get('label') ?? '').trim(),
					issuer: String(form.get('issuer') ?? '').trim(),
					client_id: String(form.get('client_id') ?? '').trim(),
					client_secret: String(form.get('client_secret') ?? '').trim(),
					allowed_domains: domains,
					join_role: String(form.get('join_role') ?? 'student'),
					auto_join: form.get('auto_join') === 'on',
					enabled: form.get('enabled') === 'on'
				},
				...scoped(locals, cookies, fetch)
			});
			return { saved: true };
		} catch (cause) {
			if (cause instanceof ApiFailure) return fail(cause.status, { message: cause.error.message });
			throw cause;
		}
	},

	removeSso: async ({ locals, cookies, fetch }) => {
		try {
			await api('/v1/sso', { method: 'DELETE', ...scoped(locals, cookies, fetch) });
			return { removed: true };
		} catch (cause) {
			if (cause instanceof ApiFailure) return fail(cause.status, { message: cause.error.message });
			throw cause;
		}
	}
};
