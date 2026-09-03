import { error, fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';
import type { Course } from '$lib/api';

type CollectionView = {
	collection: {
		id: string;
		kind: 'path' | 'bundle';
		slug: string;
		title: string;
		summary: string;
		status: string;
		price_minor: number;
		currency: string;
	};
	courses: { course: Course; position: number }[];
	courses_done: number;
};

export const load: PageServerLoad = async ({ params, locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	const scoped = { token: locals.token, tenant: session.tenant.slug, fetch };
	let view: CollectionView;
	try {
		view = await api<CollectionView>(`/v1/collections/${params.slug}`, scoped);
	} catch (cause) {
		if (cause instanceof ApiFailure && cause.status === 404) error(404, 'No path or bundle there.');
		throw cause;
	}

	const role = session.tenant.role;
	const teaches = ['owner', 'admin', 'instructor'].includes(role);
	// The courses that could still be added, for the staff picker.
	let available: Course[] = [];
	if (teaches) {
		try {
			const { courses } = await api<{ courses: Course[] }>('/v1/courses?limit=100', scoped);
			const inside = new Set(view.courses.map((row) => row.course.id));
			available = (courses ?? []).filter((course) => !inside.has(course.id));
		} catch {
			available = [];
		}
	}
	return { ...view, teaches, available };
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
	add: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		try {
			await api(`/v1/collections/${form.get('collection_id')}/courses`, {
				method: 'POST',
				body: { course_id: String(form.get('course_id') ?? '') },
				...scoped(locals, cookies, fetch)
			});
			return { added: true };
		} catch (cause) {
			return failed(cause);
		}
	},

	remove: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		try {
			await api(`/v1/collections/${form.get('collection_id')}/courses/${form.get('course_id')}`, {
				method: 'DELETE',
				...scoped(locals, cookies, fetch)
			});
			return { removed: true };
		} catch (cause) {
			return failed(cause);
		}
	},

	publish: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		try {
			await api(`/v1/collections/${form.get('collection_id')}`, {
				method: 'PATCH',
				body: { status: String(form.get('status') ?? 'published') },
				...scoped(locals, cookies, fetch)
			});
			return { saved: true };
		} catch (cause) {
			return failed(cause);
		}
	}
};
