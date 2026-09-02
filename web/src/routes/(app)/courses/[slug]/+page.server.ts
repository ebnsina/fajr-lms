import { error, redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';
import type { CourseProgress, Outline } from '$lib/api';

export const load: PageServerLoad = async ({ params, locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	const scoped = { token: locals.token, tenant: session.tenant.slug, fetch };
	let outline: Outline;
	try {
		outline = await api<Outline>(`/v1/courses/${params.slug}`, scoped);
	} catch (failure) {
		if (failure instanceof ApiFailure && failure.status === 404) {
			error(404, 'That course does not exist here.');
		}
		throw failure;
	}

	// Not being enrolled is normal, not an error: the outline still reads.
	let progress: CourseProgress | null = null;
	try {
		progress = await api<CourseProgress>(`/v1/courses/${outline.course.id}/progress`, scoped);
	} catch {
		progress = null;
	}

	return { outline, progress };
};
