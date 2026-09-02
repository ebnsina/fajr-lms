import { error, fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';
import type { Course } from '$lib/api';

type Session = {
	id: string;
	title: string;
	location: string;
	starts_at: string;
	ends_at: string | null;
};

type RollRow = {
	enrollment_id: string;
	user_id: string;
	full_name: string;
	status: 'present' | 'late' | 'absent' | 'excused' | null;
	note: string | null;
};

export const load: PageServerLoad = async ({ params, url, locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	const scoped = { token: locals.token, tenant: session.tenant.slug, fetch };
	const outline = await api<{ course: Course }>(`/v1/courses/${params.slug}`, scoped);
	const course = outline.course;

	let sessions: Session[] = [];
	try {
		({ sessions } = await api<{ sessions: Session[] }>(
			`/v1/courses/${course.id}/sessions?limit=50`,
			scoped
		));
	} catch (failure) {
		if (failure instanceof ApiFailure && failure.status === 403) {
			error(403, 'Only staff can take the register.');
		}
		throw failure;
	}

	// The chosen class, or the most recent one, so the page opens on the work.
	const chosen = url.searchParams.get('session') ?? sessions[0]?.id ?? null;
	let roll: RollRow[] = [];
	if (chosen) {
		try {
			({ roll } = await api<{ roll: RollRow[] }>(`/v1/sessions/${chosen}/roll`, scoped));
		} catch {
			roll = [];
		}
	}

	return { course, sessions, roll, chosen, slug: params.slug };
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

export const actions: Actions = {
	createSession: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		const title = String(form.get('title') ?? '').trim();
		const startsAt = String(form.get('starts_at') ?? '');
		if (!title) return fail(422, { message: 'Give the class a name.' });
		if (!startsAt) return fail(422, { message: 'Say when it starts.' });

		try {
			await api(`/v1/courses/${form.get('course_id')}/sessions`, {
				method: 'POST',
				body: {
					title,
					location: String(form.get('location') ?? ''),
					starts_at: new Date(startsAt).toISOString()
				},
				...scoped(locals, cookies, fetch)
			});
		} catch (failure) {
			if (failure instanceof ApiFailure)
				return fail(failure.status, { message: failure.error.message });
			throw failure;
		}
		return { created: true };
	},

	// The whole register goes in one request, which is how a teacher works.
	takeRoll: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		const sessionID = String(form.get('session_id') ?? '');
		const entries = form
			.getAll('entry')
			.map(String)
			.map((raw) => {
				const [enrollment_id, status] = raw.split(':');
				return { enrollment_id, status };
			})
			.filter((entry) => entry.enrollment_id && entry.status);

		if (entries.length === 0) return fail(422, { message: 'Mark at least one learner.' });

		try {
			await api(`/v1/sessions/${sessionID}/roll`, {
				method: 'PUT',
				body: { entries },
				...scoped(locals, cookies, fetch)
			});
		} catch (failure) {
			if (failure instanceof ApiFailure)
				return fail(failure.status, { message: failure.error.message });
			throw failure;
		}
		return { marked: entries.length };
	}
};
