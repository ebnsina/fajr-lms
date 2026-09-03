import type { Reroute } from '@sveltejs/kit';

// One answer per host is enough for the life of the process, so navigation
// does not pay for a lookup every time.
const known = new Map<string, string | null>();

// A school on its own domain sees its site at the root. Everything else, our
// own address included, is left alone.
export const reroute: Reroute = async ({ url, fetch }) => {
	if (url.pathname === '/__host' || url.pathname.startsWith('/site/')) return;

	let slug = known.get(url.host);
	if (slug === undefined) {
		const response = await fetch(`/__host?host=${encodeURIComponent(url.host)}`);
		if (!response.ok) return;
		slug = ((await response.json()) as { slug: string | null }).slug;
		known.set(url.host, slug);
	}
	if (!slug) return;

	const rest = url.pathname === '/' ? '' : url.pathname;
	return `/site/${slug}${rest}`;
};
