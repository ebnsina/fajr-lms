import { fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';
import type { Course } from '$lib/api';

type ClassSession = {
	id: string;
	title: string;
	location: string;
	starts_at: string;
	ends_at: string | null;
	join_url: string;
	host_url: string;
	provider: string;
	recording_media_id: string | null;
};

export const load: PageServerLoad = async ({ params, locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	const scoped = { token: locals.token, tenant: session.tenant.slug, fetch };
	const { course } = await api<{ course: Course }>(`/v1/courses/${params.slug}`, scoped);
	const { sessions } = await api<{ sessions: ClassSession[] }>(
		`/v1/courses/${course.id}/sessions?limit=50`,
		scoped
	);

	const staff = ['owner', 'admin', 'instructor', 'assistant'].includes(session.tenant.role);
	return { course, sessions, staff, slug: params.slug };
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
	setLink: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		const joinURL = String(form.get('join_url') ?? '').trim();
		if (!joinURL) return fail(422, { message: 'Paste the meeting link.' });
		try {
			await api(`/v1/sessions/${form.get('session_id')}/link`, {
				method: 'PUT',
				body: { join_url: joinURL, host_url: String(form.get('host_url') ?? '').trim() },
				...scoped(locals, cookies, fetch)
			});
		} catch (cause) {
			return failed(cause);
		}
		return { saved: true };
	},

	// A recording is media like any other, so a pasted link becomes one first.
	attachRecording: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		const link = String(form.get('recording') ?? '').trim();
		if (!link) return fail(422, { message: 'Paste the link to the recording.' });

		const call = scoped(locals, cookies, fetch);
		try {
			const media = await api<{ id: string }>('/v1/media', {
				method: 'POST',
				body: { url: link, kind: 'video', title: String(form.get('title') ?? 'Class recording') },
				...call
			});
			await api(`/v1/sessions/${form.get('session_id')}/recording`, {
				method: 'PUT',
				body: { media_id: media.id },
				...call
			});
		} catch (cause) {
			return failed(cause);
		}
		return { saved: true };
	},

	join: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		try {
			const { join_url } = await api<{ join_url: string }>(
				`/v1/sessions/${form.get('session_id')}/join`,
				scoped(locals, cookies, fetch)
			);
			return { joinURL: join_url };
		} catch (cause) {
			return failed(cause);
		}
	}
};
