import { error, redirect } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';
import type { Cookies } from '@sveltejs/kit';
import { clearStaff, STAFF_COOKIE } from '$lib/server/session';

const base = () => (env.FAJR_API_URL ?? 'http://localhost:8080').replace(/\/$/, '');

/** Every back-office call, with the staff token and none of the tenant header. */
export async function admin<T>(
	path: string,
	cookies: Cookies,
	fetcher: typeof globalThis.fetch,
	init: { method?: string; body?: unknown } = {}
): Promise<T> {
	const token = cookies.get(STAFF_COOKIE);
	if (!token) redirect(303, '/admin/login');

	let response: Response;
	try {
		response = await fetcher(`${base()}${path}`, {
			method: init.method ?? 'GET',
			headers: {
				accept: 'application/json',
				authorization: `Bearer ${token}`,
				...(init.body === undefined ? {} : { 'content-type': 'application/json' })
			},
			body: init.body === undefined ? undefined : JSON.stringify(init.body)
		});
	} catch {
		error(502, 'The API is not answering.');
	}

	// A dead session means sign in again; a 404 is a missing row, and says so.
	if (response.status === 401) {
		clearStaff(cookies);
		redirect(303, '/admin/login');
	}
	if (!response.ok) {
		const failed = await response.json().catch(() => null);
		error(response.status, failed?.error?.message ?? 'That did not work.');
	}
	return (await response.json()) as T;
}
