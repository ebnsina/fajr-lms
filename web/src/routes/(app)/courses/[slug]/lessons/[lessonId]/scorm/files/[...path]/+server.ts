import { error, redirect, type RequestHandler } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';

const base = () => (env.FAJR_API_URL ?? 'http://localhost:8080').replace(/\/$/, '');

// The package's own files, passed through this app so the frame playing them
// is same-origin: SCORM's API is found by walking up window.parent.
export const GET: RequestHandler = async ({ params, locals, cookies, fetch }) => {
	const tenant = cookies.get('fajr_tenant');
	if (!locals.token || !tenant) redirect(303, '/login');

	const path = (params.path ?? '')
		.split('/')
		.map((part) => encodeURIComponent(part))
		.join('/');

	let response: Response;
	try {
		response = await fetch(`${base()}/v1/lessons/${params.lessonId}/scorm/files/${path}`, {
			headers: { authorization: `Bearer ${locals.token}`, 'x-fajr-tenant': tenant }
		});
	} catch {
		error(502, 'The package could not be reached.');
	}
	if (response.status === 404) error(404, 'That file is not part of the package.');
	if (!response.ok) error(response.status, 'That file could not be served.');

	return new Response(response.body, {
		status: 200,
		headers: {
			'content-type': response.headers.get('content-type') ?? 'application/octet-stream',
			'cache-control': 'private, max-age=3600'
		}
	});
};
