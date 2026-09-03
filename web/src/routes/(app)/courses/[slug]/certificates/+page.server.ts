import { error, fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';
import type { Course } from '$lib/api';

type RosterRow = {
	enrollment: { id: string; user_id: string; status: string };
	full_name: string;
	percent_complete: number;
};

type Awarded = {
	certificate: {
		id: string;
		serial: string;
		recipient_name: string;
		grade_percent: number | null;
		issued_at: string;
		revoked_at: string | null;
		revoked_reason: string;
	};
	full_name: string;
	verify_url: string;
};

export const load: PageServerLoad = async ({ params, locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');
	if (!['owner', 'admin', 'instructor'].includes(session.tenant.role)) {
		error(403, 'Only staff can award a certificate.');
	}

	const scoped = { token: locals.token, tenant: session.tenant.slug, fetch };
	const { course } = await api<{ course: Course }>(`/v1/courses/${params.slug}`, scoped);
	const { roster } = await api<{ roster: RosterRow[] }>(
		`/v1/courses/${course.id}/roster?limit=200`,
		scoped
	);
	const { certificates } = await api<{ certificates: Awarded[] }>(
		`/v1/courses/${course.id}/certificates?limit=200`,
		scoped
	);

	// Somebody already holding one that still stands is not offered again.
	const held = new Set(
		certificates.filter((row) => !row.certificate.revoked_at).map((row) => row.full_name)
	);
	const candidates = roster.filter((row) => !held.has(row.full_name));

	return { course, candidates, certificates, slug: params.slug };
};

const scoped = (
	locals: App.Locals,
	cookies: { get: (n: string) => string | undefined },
	fetch: typeof globalThis.fetch
) => {
	const tenant = cookies.get('fajr_tenant');
	if (!locals.token || !tenant) redirect(303, '/login');
	return { token: locals.token, tenant, fetch };
};

const failed = (cause: unknown) => {
	if (cause instanceof ApiFailure) return fail(cause.status, { message: cause.error.message });
	throw cause;
};

export const actions: Actions = {
	award: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		const userID = String(form.get('user_id') ?? '');
		if (!userID) return fail(422, { message: 'Choose who is being awarded.' });

		try {
			await api(`/v1/courses/${form.get('course_id')}/certificates`, {
				method: 'POST',
				body: { user_id: userID },
				...scoped(locals, cookies, fetch)
			});
		} catch (cause) {
			return failed(cause);
		}
		return { awarded: true };
	},

	revoke: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		try {
			await api(`/v1/certificates/${form.get('certificate_id')}/revoke`, {
				method: 'POST',
				body: { reason: String(form.get('reason') ?? '').trim() },
				...scoped(locals, cookies, fetch)
			});
		} catch (cause) {
			return failed(cause);
		}
		return { revoked: true };
	}
};
