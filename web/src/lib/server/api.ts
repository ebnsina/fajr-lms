import { env } from '$env/dynamic/private';
import type { ApiError } from '$lib/types';

const base = () => (env.FAJR_API_URL ?? 'http://localhost:8080').replace(/\/$/, '');

export class ApiFailure extends Error {
	constructor(
		readonly status: number,
		readonly error: ApiError
	) {
		super(error.message);
	}
}

type Options = {
	method?: string;
	body?: unknown;
	token?: string;
	tenant?: string;
	fetch?: typeof globalThis.fetch;
};

/** Calls the API and turns its error shape into something a page can render. */
export async function api<T>(path: string, options: Options = {}): Promise<T> {
	const headers: Record<string, string> = { accept: 'application/json' };
	if (options.token) headers.authorization = `Bearer ${options.token}`;
	if (options.tenant) headers['x-fajr-tenant'] = options.tenant;
	if (options.body !== undefined) headers['content-type'] = 'application/json';

	const doFetch = options.fetch ?? globalThis.fetch;
	let response: Response;
	try {
		response = await doFetch(`${base()}${path}`, {
			method: options.method ?? 'GET',
			headers,
			body: options.body === undefined ? undefined : JSON.stringify(options.body)
		});
	} catch (cause) {
		throw new ApiFailure(503, {
			code: 'api_unreachable',
			message: 'Could not reach the server. Check your connection and try again.'
		});
	}

	if (response.status === 204) return undefined as T;

	const text = await response.text();
	const parsed = text ? safeParse(text) : null;

	if (!response.ok) {
		const error = (parsed as { error?: ApiError } | null)?.error;
		throw new ApiFailure(
			response.status,
			error ?? {
				code: 'unexpected',
				message: 'Something went wrong. Please try again.'
			}
		);
	}
	return parsed as T;
}

function safeParse(text: string): unknown {
	try {
		return JSON.parse(text);
	} catch {
		return null;
	}
}
