import type { PageServerLoad } from './$types';
import { admin } from '$lib/server/admin';

export type School = {
	id: string;
	slug: string;
	name: string;
	kind: string;
	status: string;
	demo: boolean;
	created_at: string;
	members: number;
	courses: number;
	learners: number;
	certificates: number;
	orders: number;
	last_activity: string;
};

export const load: PageServerLoad = async ({ cookies, fetch, url }) => {
	const query = url.searchParams.get('q') ?? '';
	const search = new URLSearchParams({ limit: '200' });
	if (query) search.set('q', query);
	const { schools } = await admin<{ schools: School[] }>(
		`/v1/admin/schools?${search}`,
		cookies,
		fetch
	);
	return { schools, query };
};
