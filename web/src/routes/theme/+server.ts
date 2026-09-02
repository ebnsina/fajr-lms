import { json } from '@sveltejs/kit';
import { dev } from '$app/environment';
import type { RequestHandler } from './$types';
import { saveTheme } from '$lib/server/session';

const allowed = new Set(['light', 'dark', 'system']);

/** Stores the choice server side so the next page is stamped before it paints. */
export const POST: RequestHandler = async ({ request, cookies }) => {
	const { theme } = await request.json().catch(() => ({ theme: null }));
	if (typeof theme !== 'string' || !allowed.has(theme)) {
		return json({ error: 'Unknown theme.' }, { status: 422 });
	}
	saveTheme(cookies, theme, !dev);
	return json({ theme });
};
