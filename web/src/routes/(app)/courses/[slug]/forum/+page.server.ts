import { error, fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';
import type { Outline } from '$lib/api';

type ThreadRow = {
	forum_thread: {
		id: string;
		title: string;
		dir: string;
		pinned: boolean;
		locked: boolean;
		reply_count: number;
		last_post_at: string;
	};
	author_name: string | null;
};

export const load: PageServerLoad = async ({ params, locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	const scoped = { token: locals.token, tenant: session.tenant.slug, fetch };
	let outline: Outline;
	try {
		outline = await api<Outline>(`/v1/courses/${params.slug}`, scoped);
	} catch (cause) {
		if (cause instanceof ApiFailure && cause.status === 404)
			error(404, 'That course does not exist here.');
		throw cause;
	}

	const { threads } = await api<{ threads: ThreadRow[] }>(
		`/v1/courses/${outline.course.id}/threads?limit=50`,
		scoped
	);
	const role = session.tenant.role;
	return {
		course: outline.course,
		threads: threads ?? [],
		moderates: ['owner', 'admin', 'instructor', 'assistant'].includes(role)
	};
};

export const actions: Actions = {
	start: async ({ request, locals, cookies, fetch }) => {
		const tenant = cookies.get('fajr_tenant');
		if (!locals.token || !tenant) redirect(303, '/login');

		const form = await request.formData();
		const title = String(form.get('title') ?? '').trim();
		const body = String(form.get('body') ?? '').trim();
		if (!title || !body) return fail(422, { message: 'A question needs a title and a question.' });

		try {
			await api(`/v1/courses/${form.get('course_id')}/threads`, {
				method: 'POST',
				body: { title, body, dir: 'auto' },
				token: locals.token,
				tenant,
				fetch
			});
			return { started: true };
		} catch (cause) {
			if (cause instanceof ApiFailure) return fail(cause.status, { message: cause.error.message });
			throw cause;
		}
	}
};
