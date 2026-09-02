import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { api } from '$lib/server/api';

type Attempt = {
	quiz_attempt: { id: string; submitted_at: string | null };
	full_name: string;
	quiz_title: string;
	lesson_title: string;
	pending: number;
};

export const load: PageServerLoad = async ({ locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	const { attempts } = await api<{ attempts: Attempt[] }>('/v1/grading?limit=50', {
		token: locals.token,
		tenant: session.tenant.slug,
		fetch
	});
	return { attempts };
};
