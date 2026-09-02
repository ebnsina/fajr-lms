import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { api } from '$lib/server/api';
import type { Course } from '$lib/api';

export const load: PageServerLoad = async ({ locals, parent, url, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	const status = url.searchParams.get('status');
	const query = status ? `?status=${encodeURIComponent(status)}&limit=50` : '?limit=50';

	const { courses, total } = await api<{ courses: Course[]; total: number }>(
		`/v1/courses${query}`,
		{ token: locals.token, tenant: session.tenant.slug, fetch }
	);
	return { courses, total, status };
};
