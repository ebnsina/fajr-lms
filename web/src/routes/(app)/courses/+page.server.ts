import { fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';
import type { Course } from '$lib/api';

export const load: PageServerLoad = async ({ locals, parent, url, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	const status = url.searchParams.get('status');
	const query = status ? `?status=${encodeURIComponent(status)}&limit=50` : '?limit=50';

	const { courses, total } = await api<{ courses: Course[]; total: number }>(
		`/v1/courses${query}`,
		{ token: locals.token, tenant: session.tenant.slug, fetch }
	);
	const role = session.tenant.role;
	return { courses, total, status, teaches: ['owner', 'admin', 'instructor'].includes(role) };
};

export const actions: Actions = {
	create: async ({ request, locals, cookies, fetch }) => {
		const tenant = cookies.get('fajr_tenant');
		if (!locals.token || !tenant) redirect(303, '/login');

		const form = await request.formData();
		const title = String(form.get('title') ?? '').trim();
		if (!title) return fail(422, { message: 'Give the course a title.' });

		let course: Course;
		try {
			course = await api<Course>('/v1/courses', {
				method: 'POST',
				token: locals.token,
				tenant,
				body: {
					title,
					summary: String(form.get('summary') ?? '').trim(),
					dir: String(form.get('dir') ?? 'auto'),
					visibility: String(form.get('visibility') ?? 'private'),
					price_minor: Math.round(Number(form.get('price') ?? 0) * 100)
				},
				fetch
			});
		} catch (cause) {
			if (cause instanceof ApiFailure) return fail(cause.status, { message: cause.error.message });
			throw cause;
		}
		redirect(303, `/courses/${course.slug}/edit`);
	}
};
