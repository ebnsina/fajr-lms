import type { Handle } from '@sveltejs/kit';
import { TENANT_COOKIE, TOKEN_COOKIE } from '$lib/server/session';

export const handle: Handle = async ({ event, resolve }) => {
	event.locals.token = event.cookies.get(TOKEN_COOKIE);
	event.locals.tenantSlug = event.cookies.get(TENANT_COOKIE);

	// The document direction is decided per request, so a right-to-left tenant
	// gets a correct first paint rather than a flip after hydration.
	return resolve(event, {
		transformPageChunk: ({ html }) =>
			html.replace('%fajr.dir%', event.locals.dir ?? 'ltr').replace('%fajr.lang%', event.locals.lang ?? 'en')
	});
};
