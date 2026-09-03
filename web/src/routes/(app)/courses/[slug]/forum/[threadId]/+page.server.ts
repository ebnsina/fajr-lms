import { error, fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';

type Thread = {
	thread: {
		forum_thread: {
			id: string;
			title: string;
			dir: string;
			pinned: boolean;
			locked: boolean;
			reply_count: number;
		};
		author_name: string | null;
		course_slug: string;
		course_title: string;
	};
	posts: {
		id: string;
		body: string;
		dir: string;
		created_at: string;
		author_id: string | null;
		author_name: string | null;
		removed: boolean;
	}[];
};

export const load: PageServerLoad = async ({ params, locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	let thread: Thread;
	try {
		thread = await api<Thread>(`/v1/threads/${params.threadId}`, {
			token: locals.token,
			tenant: session.tenant.slug,
			fetch
		});
	} catch (cause) {
		if (cause instanceof ApiFailure && cause.status === 404) error(404, 'That discussion is gone.');
		throw cause;
	}

	const role = session.tenant.role;
	return {
		thread: thread.thread,
		posts: thread.posts,
		me: session.user?.id ?? '',
		moderates: ['owner', 'admin', 'instructor', 'assistant'].includes(role)
	};
};

const scoped = (
	locals: App.Locals,
	cookies: { get: (name: string) => string | undefined },
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
	reply: async ({ params, request, locals, cookies, fetch }) => {
		const form = await request.formData();
		const body = String(form.get('body') ?? '').trim();
		if (!body) return fail(422, { message: 'Write something first.' });

		try {
			await api(`/v1/threads/${params.threadId}/posts`, {
				method: 'POST',
				body: { body, dir: 'auto' },
				...scoped(locals, cookies, fetch)
			});
			return { replied: true };
		} catch (cause) {
			return failed(cause);
		}
	},

	remove: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		try {
			await api(`/v1/posts/${form.get('post_id')}`, {
				method: 'DELETE',
				...scoped(locals, cookies, fetch)
			});
			return { removed: true };
		} catch (cause) {
			return failed(cause);
		}
	},

	flags: async ({ params, request, locals, cookies, fetch }) => {
		const form = await request.formData();
		try {
			await api(`/v1/threads/${params.threadId}/flags`, {
				method: 'PUT',
				body: {
					pinned: form.get('pinned') === null ? null : form.get('pinned') === 'true',
					locked: form.get('locked') === null ? null : form.get('locked') === 'true'
				},
				...scoped(locals, cookies, fetch)
			});
			return { saved: true };
		} catch (cause) {
			return failed(cause);
		}
	}
};
