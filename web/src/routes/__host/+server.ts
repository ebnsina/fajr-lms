import { json, type RequestHandler } from '@sveltejs/kit';
import { api, ApiFailure } from '$lib/server/api';

type Resolved = { slug: string; name: string };

const cache = new Map<string, { slug: string | null; until: number }>();
const TTL_MS = 5 * 60 * 1000;

// Which school a hostname belongs to. Answers null for our own address and for
// anything not pointed at a verified domain.
export const GET: RequestHandler = async ({ url, request, fetch }) => {
	const host = (url.searchParams.get('host') ?? request.headers.get('host') ?? '')
		.toLowerCase()
		.replace(/:\d+$/, '');

	const hit = cache.get(host);
	if (hit && hit.until > Date.now()) return json({ slug: hit.slug });

	let slug: string | null = null;
	if (host) {
		try {
			slug = (await api<Resolved>(`/site/resolve?host=${encodeURIComponent(host)}`, { fetch }))
				.slug;
		} catch (cause) {
			if (!(cause instanceof ApiFailure)) throw cause;
		}
	}
	cache.set(host, { slug, until: Date.now() + TTL_MS });
	return json({ slug });
};
