import { json, redirect, type RequestHandler } from '@sveltejs/kit';
import { api, ApiFailure } from '$lib/server/api';

// What the package reports, passed on to the API with the session's token.
export const POST: RequestHandler = async ({ params, request, locals, cookies, fetch }) => {
	const tenant = cookies.get('fajr_tenant');
	if (!locals.token || !tenant) redirect(303, '/login');

	const body = await request.json();
	try {
		const saved = await api(`/v1/lessons/${params.lessonId}/scorm/state`, {
			method: 'PUT',
			body,
			token: locals.token,
			tenant,
			fetch
		});
		return json(saved);
	} catch (cause) {
		if (cause instanceof ApiFailure) {
			return json({ message: cause.error.message }, { status: cause.status });
		}
		throw cause;
	}
};
