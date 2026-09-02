import { fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';

type Notification = {
	id: string;
	kind: string;
	title: string;
	body: string;
	read_at: string | null;
	created_at: string;
};

export const load: PageServerLoad = async ({ locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	const { notifications, unread } = await api<{ notifications: Notification[]; unread: number }>(
		'/v1/notifications?limit=50',
		{ token: locals.token, tenant: session.tenant.slug, fetch }
	);
	return { notifications, unread };
};

export const actions: Actions = {
	readAll: async ({ locals, cookies, fetch }) => {
		const tenant = cookies.get('fajr_tenant');
		if (!locals.token || !tenant) redirect(303, '/login');
		try {
			await api('/v1/notifications/read', { method: 'POST', token: locals.token, tenant, fetch });
		} catch (error) {
			if (error instanceof ApiFailure) return fail(error.status, { message: error.error.message });
			throw error;
		}
		return { cleared: true };
	}
};
