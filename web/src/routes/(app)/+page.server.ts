import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { api } from '$lib/server/api';
import type { CourseProgress } from '$lib/api';

const staffRoles = ['owner', 'admin', 'instructor', 'assistant'];
const adminRoles = ['owner', 'admin'];

type Enrollment = {
	enrollment: { id: string; status: string; course_id: string };
	slug: string;
	title: string;
	dir: string;
	course_status: string;
};

type Certificate = { certificate: { id: string; revoked_at: string | null } };

type Attempt = {
	quiz_attempt: { id: string; submitted_at: string | null };
	full_name: string;
	quiz_title: string;
	pending: number;
};

type Submission = {
	submission: { id: string; is_late: boolean; submitted_at: string | null };
	full_name: string;
	assignment_title: string;
};

type ReviewOrder = {
	order: { id: string; created_at: string; amount_minor: number; currency: string };
	title: string;
	full_name: string;
};

/** An overview panel that fails should go quiet, not take the page down. */
async function orElse<T>(work: Promise<T>, fallback: T): Promise<T> {
	try {
		return await work;
	} catch {
		return fallback;
	}
}

export const load: PageServerLoad = async ({ locals, parent, fetch }) => {
	// A visitor lands on the marketing site; a member lands on their work.
	if (!locals.token) redirect(303, '/welcome');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	const role = session.tenant.role;
	const isStaff = staffRoles.includes(role);
	const isAdmin = adminRoles.includes(role);
	const scoped = { token: locals.token, tenant: session.tenant.slug, fetch };

	const [enrolled, certificates, attempts, submissions, catalog, review] = await Promise.all([
		api<{ enrollments: Enrollment[] }>('/v1/enrollments?limit=8', scoped),
		orElse(api<{ certificates: Certificate[] }>('/v1/certificates', scoped), { certificates: [] }),
		isStaff
			? orElse(api<{ attempts: Attempt[] }>('/v1/grading?limit=50', scoped), { attempts: [] })
			: { attempts: [] as Attempt[] },
		isStaff
			? orElse(api<{ submissions: Submission[] }>('/v1/submissions?limit=50', scoped), {
					submissions: []
				})
			: { submissions: [] as Submission[] },
		isStaff
			? orElse(api<{ total: number }>('/v1/courses?limit=1', scoped), { total: 0 })
			: { total: 0 },
		isAdmin
			? orElse(api<{ orders: ReviewOrder[] }>('/v1/orders/review?limit=50', scoped), { orders: [] })
			: { orders: [] as ReviewOrder[] }
	]);

	const enrollments = enrolled.enrollments ?? [];

	// One request per enrolled course, the same handful the grades page walks.
	const courses = await Promise.all(
		enrollments.map(async (row) => {
			const progress = await orElse(
				api<CourseProgress>(`/v1/courses/${row.enrollment.course_id}/progress`, scoped),
				null as CourseProgress | null
			);
			return { ...row, progress };
		})
	);

	return {
		role,
		isStaff,
		isAdmin,
		courses,
		certificates: (certificates.certificates ?? []).filter((row) => !row.certificate.revoked_at)
			.length,
		attempts: attempts.attempts ?? [],
		submissions: submissions.submissions ?? [],
		courseCount: catalog.total ?? 0,
		review: review.orders ?? []
	};
};
