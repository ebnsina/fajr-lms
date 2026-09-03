import { fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';

type Certificate = {
	certificate: {
		id: string;
		serial: string;
		recipient_name: string;
		course_title: string;
		issuer_name: string;
		grade_percent: number | null;
		issued_at: string;
		revoked_at: string | null;
	};
	course_slug: string;
	verify_url: string;
};

type Enrollment = {
	enrollment: { id: string; status: string; course_id: string };
	slug: string;
	title: string;
};

export const load: PageServerLoad = async ({ locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	const scoped = { token: locals.token, tenant: session.tenant.slug, fetch };
	const { certificates } = await api<{ certificates: Certificate[] }>('/v1/certificates', scoped);
	const { enrollments } = await api<{ enrollments: Enrollment[] }>('/v1/enrollments', scoped);

	// A course is claimable once it is finished and has no certificate yet.
	const awarded = new Set(certificates.map((row) => row.course_slug));
	const claimable = enrollments.filter(
		(row) => row.enrollment.status === 'completed' && !awarded.has(row.slug)
	);
	const runsSchool = ['owner', 'admin'].includes(session.tenant.role);
	return { certificates, claimable, runsSchool };
};

export const actions: Actions = {
	claim: async ({ request, locals, cookies, fetch }) => {
		const tenant = cookies.get('fajr_tenant');
		if (!locals.token || !tenant) redirect(303, '/login');

		const form = await request.formData();
		try {
			await api(`/v1/courses/${form.get('course_id')}/certificates`, {
				method: 'POST',
				token: locals.token,
				tenant,
				body: {},
				fetch
			});
		} catch (cause) {
			if (cause instanceof ApiFailure) return fail(cause.status, { message: cause.error.message });
			throw cause;
		}
		return { claimed: true };
	}
};
