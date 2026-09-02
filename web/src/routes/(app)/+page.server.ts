import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { api } from '$lib/server/api';

type Enrollment = {
	order: never;
	enrollment: { id: string; status: string };
	slug: string;
	title: string;
	dir: string;
	course_status: string;
};

export const load: PageServerLoad = async ({ locals, parent, fetch }) => {
	// A visitor lands on the marketing site; a member lands on their work.
	if (!locals.token) redirect(303, '/welcome');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	const scoped = { token: locals.token, tenant: session.tenant.slug, fetch };
	const [enrollments, notifications] = await Promise.all([
		api<{ enrollments: Enrollment[] }>('/v1/enrollments?limit=6', scoped),
		api<{ unread: number }>('/v1/notifications?limit=1', scoped)
	]);

	return {
		enrollments: enrollments.enrollments ?? [],
		unread: notifications.unread
	};
};
