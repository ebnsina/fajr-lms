import { fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';

type Member = {
	id: string;
	user_id: string;
	role: string;
	status: string;
	full_name: string;
	phone: string | null;
	email: string | null;
};

export const load: PageServerLoad = async ({ locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	const { members, total } = await api<{ members: Member[]; total: number }>(
		'/v1/tenant/members?limit=100',
		{ token: locals.token, tenant: session.tenant.slug, fetch }
	);
	const role = session.tenant.role;
	return { members, total, canManage: role === 'owner' || role === 'admin' };
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

const failed = (cause: unknown) => {
	if (cause instanceof ApiFailure) return fail(cause.status, { message: cause.error.message });
	throw cause;
};

export const actions: Actions = {
	invite: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		const fullName = String(form.get('full_name') ?? '').trim();
		const destination = String(form.get('destination') ?? '').trim();
		if (!fullName) return fail(422, { message: 'What is their name?' });
		if (!destination) return fail(422, { message: 'A phone number or an email address, please.' });

		try {
			await api('/v1/tenant/members', {
				method: 'POST',
				body: { full_name: fullName, destination, role: String(form.get('role') ?? 'student') },
				...scoped(locals, cookies, fetch)
			});
		} catch (cause) {
			return failed(cause);
		}
		return { invited: fullName };
	},

	setRole: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		try {
			await api(`/v1/tenant/members/${form.get('user_id')}/role`, {
				method: 'PUT',
				body: { role: String(form.get('role') ?? 'student') },
				...scoped(locals, cookies, fetch)
			});
		} catch (cause) {
			return failed(cause);
		}
		return { saved: true };
	},

	remove: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		try {
			await api(`/v1/tenant/members/${form.get('user_id')}`, {
				method: 'DELETE',
				...scoped(locals, cookies, fetch)
			});
		} catch (cause) {
			return failed(cause);
		}
		return { removed: true };
	}
};
