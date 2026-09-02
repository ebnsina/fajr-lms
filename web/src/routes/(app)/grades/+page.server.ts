import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { api } from '$lib/server/api';
import type { CourseProgress } from '$lib/api';

type Row = {
	enrollment: { id: string; status: string; course_id: string };
	slug: string;
	title: string;
	dir: string;
};

export const load: PageServerLoad = async ({ locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	const scoped = { token: locals.token, tenant: session.tenant.slug, fetch };
	const { enrollments } = await api<{ enrollments: Row[] }>('/v1/enrollments?limit=50', scoped);

	// One request per course, but only for the handful somebody is enrolled in.
	const courses = await Promise.all(
		(enrollments ?? []).map(async (row) => {
			try {
				const grades = await api<{
					items: unknown[];
					grades: {
						percent: number;
						items_graded: number;
						items_total: number;
					};
				}>(`/v1/courses/${row.enrollment.course_id}/grades`, scoped);
				return { ...row, grades: grades.grades };
			} catch {
				return { ...row, grades: null };
			}
		})
	);
	return { courses };
};
