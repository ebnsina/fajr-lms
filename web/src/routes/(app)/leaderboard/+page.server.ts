import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';

type Standing = { user_id: string; full_name: string; points: number };

type Mine = {
	on: boolean;
	points: number;
	badges: { id: string; name: string; description: string; emoji: string; awarded_at: string }[];
};

export const load: PageServerLoad = async ({ url, locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	const scoped = { token: locals.token, tenant: session.tenant.slug, fetch };
	const window = url.searchParams.get('window') === 'all' ? 'all' : 'month';
	const mine = await api<Mine>('/v1/points/me', scoped);

	// A school with the board switched off still shows a person their own
	// points and badges, so the page is never empty for them.
	let standings: Standing[] = [];
	if (mine.on) {
		try {
			const board = await api<{ standings: Standing[] }>(
				`/v1/points/board?window=${window}&limit=25`,
				scoped
			);
			standings = board.standings ?? [];
		} catch (cause) {
			if (!(cause instanceof ApiFailure)) throw cause;
		}
	}
	return { mine, standings, window, me: session.user?.id ?? '' };
};
