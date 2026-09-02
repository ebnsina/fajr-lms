import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { api } from '$lib/server/api';

type Row = {
	submission: { id: string; is_late: boolean; submitted_at: string | null };
	full_name: string;
	assignment_title: string;
	points: number;
	due_at: string | null;
};

export const load: PageServerLoad = async ({ locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	const { submissions } = await api<{ submissions: Row[] }>('/v1/submissions?limit=50', {
		token: locals.token,
		tenant: session.tenant.slug,
		fetch
	});
	return { submissions };
};
