import { error, fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';
import type { Outline } from '$lib/api';

export const load: PageServerLoad = async ({ params, locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');
	if (!['owner', 'admin', 'instructor'].includes(session.tenant.role)) {
		error(403, 'Only staff can build a course.');
	}

	try {
		const outline = await api<Outline>(`/v1/courses/${params.slug}`, {
			token: locals.token,
			tenant: session.tenant.slug,
			fetch
		});
		return { outline, slug: params.slug };
	} catch (cause) {
		if (cause instanceof ApiFailure && cause.status === 404) {
			error(404, 'That course does not exist here.');
		}
		throw cause;
	}
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
	addModule: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		const title = String(form.get('title') ?? '').trim();
		if (!title) return fail(422, { message: 'Give the section a name.' });
		try {
			await api(`/v1/courses/${form.get('course_id')}/modules`, {
				method: 'POST',
				body: { title },
				...scoped(locals, cookies, fetch)
			});
		} catch (cause) {
			return failed(cause);
		}
		return { added: true };
	},

	addLesson: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		const title = String(form.get('title') ?? '').trim();
		const kind = String(form.get('kind') ?? 'text');
		const link = String(form.get('link') ?? '').trim();
		if (!title) return fail(422, { message: 'Give the lesson a name.' });

		const call = scoped(locals, cookies, fetch);
		let lesson: { id: string };
		try {
			lesson = await api<{ id: string }>(`/v1/modules/${form.get('module_id')}/lessons`, {
				method: 'POST',
				body: { title, kind, body: String(form.get('body') ?? '').trim(), dir: 'auto' },
				...call
			});
		} catch (cause) {
			return failed(cause);
		}

		// A pasted link becomes media on the lesson, which is how most teaching
		// video reaches us today.
		if (link) {
			try {
				const media = await api<{ id: string }>('/v1/media', {
					method: 'POST',
					body: { url: link, kind: kind === 'audio' ? 'audio' : 'video', title },
					...call
				});
				await api(`/v1/lessons/${lesson.id}/media`, {
					method: 'PUT',
					body: { media_id: media.id },
					...call
				});
			} catch (cause) {
				if (cause instanceof ApiFailure) {
					return fail(cause.status, {
						message: `The lesson was created, but the link was refused: ${cause.error.message}`
					});
				}
				throw cause;
			}
		}
		return { added: true };
	},

	setLessonStatus: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		try {
			await api(`/v1/lessons/${form.get('lesson_id')}`, {
				method: 'PATCH',
				body: { status: String(form.get('status') ?? 'draft') },
				...scoped(locals, cookies, fetch)
			});
		} catch (cause) {
			return failed(cause);
		}
		return { saved: true };
	},

	removeLesson: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		try {
			await api(`/v1/lessons/${form.get('lesson_id')}`, {
				method: 'DELETE',
				...scoped(locals, cookies, fetch)
			});
		} catch (cause) {
			return failed(cause);
		}
		return { removed: true };
	},

	setCourseStatus: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		try {
			await api(`/v1/courses/${form.get('course_id')}/status`, {
				method: 'PUT',
				body: { status: String(form.get('status') ?? 'draft') },
				...scoped(locals, cookies, fetch)
			});
		} catch (cause) {
			return failed(cause);
		}
		return { saved: true };
	}
};
