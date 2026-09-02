import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { api } from '$lib/server/api';

type Member = {
	id: string;
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
	return { members, total };
};
