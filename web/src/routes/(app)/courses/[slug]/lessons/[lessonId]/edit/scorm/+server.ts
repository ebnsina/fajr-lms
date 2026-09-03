import { json, redirect, type RequestHandler } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';

const base = () => (env.FAJR_API_URL ?? 'http://localhost:8080').replace(/\/$/, '');

// The zip goes straight through to the API: it is read once, unpacked there,
// and never held by this app.
export const POST: RequestHandler = async ({ params, request, locals, cookies, fetch }) => {
	const tenant = cookies.get('fajr_tenant');
	if (!locals.token || !tenant) redirect(303, '/login');

	let response: Response;
	try {
		response = await fetch(`${base()}/v1/lessons/${params.lessonId}/scorm`, {
			method: 'POST',
			headers: {
				authorization: `Bearer ${locals.token}`,
				'x-fajr-tenant': tenant,
				'content-type': request.headers.get('content-type') ?? 'multipart/form-data'
			},
			body: await request.arrayBuffer()
		});
	} catch {
		return json({ message: 'The package could not be sent. Try again.' }, { status: 502 });
	}

	const body = await response.text();
	return new Response(body, {
		status: response.status,
		headers: { 'content-type': 'application/json' }
	});
};

export const DELETE: RequestHandler = async ({ params, locals, cookies, fetch }) => {
	const tenant = cookies.get('fajr_tenant');
	if (!locals.token || !tenant) redirect(303, '/login');

	const response = await fetch(`${base()}/v1/lessons/${params.lessonId}/scorm`, {
		method: 'DELETE',
		headers: { authorization: `Bearer ${locals.token}`, 'x-fajr-tenant': tenant }
	});
	return new Response(null, { status: response.status });
};
