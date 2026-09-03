import { error, json, type RequestHandler } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';

const base = () => (env.FAJR_API_URL ?? 'http://localhost:8080').replace(/\/$/, '');
const path = '/v1/certificates/layout/background';

// The paper the school is drawing on. The bytes come through this app because
// the API token never leaves the server.
const scoped = (locals: App.Locals, cookies: { get(name: string): string | undefined }) => {
	const tenant = cookies.get('fajr_tenant');
	if (!locals.token || !tenant) error(401, 'Sign in first.');
	return { authorization: `Bearer ${locals.token}`, 'x-fajr-tenant': tenant };
};

export const GET: RequestHandler = async ({ locals, cookies, fetch }) => {
	const headers = scoped(locals, cookies);
	let response: Response;
	try {
		response = await fetch(`${base()}${path}`, { headers });
	} catch {
		error(502, 'The background could not be reached.');
	}
	if (response.status === 404) error(404, 'No background yet.');
	if (!response.ok) error(response.status, 'That background could not be served.');

	return new Response(response.body, {
		status: 200,
		headers: {
			'content-type': response.headers.get('content-type') ?? 'application/octet-stream',
			'x-content-type-options': 'nosniff',
			'content-security-policy': "sandbox; default-src 'none'; style-src 'unsafe-inline'",
			'cache-control': 'private, no-store'
		}
	});
};

export const PUT: RequestHandler = async ({ request, locals, cookies, fetch }) => {
	const headers = scoped(locals, cookies);
	let response: Response;
	try {
		response = await fetch(`${base()}${path}`, {
			method: 'PUT',
			headers: { ...headers, 'content-type': request.headers.get('content-type') ?? '' },
			body: await request.arrayBuffer()
		});
	} catch {
		error(502, 'The image could not be sent.');
	}

	const text = await response.text();
	const parsed = text ? (JSON.parse(text) as { error?: { message: string } }) : null;
	if (!response.ok) {
		return json(
			{ message: parsed?.error?.message ?? 'That image could not be used.' },
			{
				status: response.status
			}
		);
	}
	return json({ has_background: true });
};

export const DELETE: RequestHandler = async ({ locals, cookies, fetch }) => {
	const headers = scoped(locals, cookies);
	try {
		const response = await fetch(`${base()}${path}`, { method: 'DELETE', headers });
		if (!response.ok && response.status !== 204) {
			error(response.status, 'The background could not be removed.');
		}
	} catch {
		error(502, 'The background could not be removed.');
	}
	return new Response(null, { status: 204 });
};
