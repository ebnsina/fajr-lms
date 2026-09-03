import { error, redirect, type RequestHandler } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';
import { STAFF_COOKIE } from '$lib/server/session';

const base = () => (env.FAJR_API_URL ?? 'http://localhost:8080').replace(/\/$/, '');

// The browser cannot send a bearer token from a link, so the file comes
// through here.
export const GET: RequestHandler = async ({ cookies, fetch, url }) => {
	const token = cookies.get(STAFF_COOKIE);
	if (!token) redirect(303, '/admin/login');

	const search = new URLSearchParams({ limit: '500' });
	const state = url.searchParams.get('state');
	const query = url.searchParams.get('q');
	if (state) search.set('state', state);
	if (query) search.set('q', query);

	let response: Response;
	try {
		response = await fetch(`${base()}/v1/admin/leads.csv?${search}`, {
			headers: { authorization: `Bearer ${token}` }
		});
	} catch {
		error(502, 'The API is not answering.');
	}
	if (response.status === 401) redirect(303, '/admin/login');
	if (!response.ok) error(response.status, 'The file could not be made.');

	return new Response(response.body, {
		status: 200,
		headers: {
			'content-type': 'text/csv; charset=utf-8',
			'content-disposition': 'attachment; filename="fajr-leads.csv"',
			'cache-control': 'private, no-store'
		}
	});
};
