import type { LayoutServerLoad } from './$types';
import { api } from '$lib/server/api';
import { aiEnabled } from '$lib/server/ai';

type Child = { student_id: string };

type TenantSummary = {
	id: string;
	slug: string;
	name: string;
	role: string;
	kind: string;
};

type NotificationSummary = {
	id: string;
	title: string;
	body: string;
	read_at: string | null;
	created_at: string;
};

// Feeds the header bell on every authenticated page. Failures here are
// cosmetic (the bell just shows nothing), so they never block the route.
export const load: LayoutServerLoad = async ({ locals, parent, fetch }) => {
	const { session } = await parent();
	if (!locals.token || !session?.tenant) {
		return {
			recentNotifications: [] as NotificationSummary[],
			unread: 0,
			schools: [] as TenantSummary[],
			ai: aiEnabled(),
			isGuardian: false
		};
	}

	// The switcher lists every school this person belongs to, so it needs names
	// rather than the ids /v1/me returns.
	let schools: TenantSummary[] = [];
	try {
		const listed = await api<{ tenants: TenantSummary[] }>('/v1/tenants', {
			token: locals.token,
			fetch
		});
		schools = listed.tenants ?? [];
	} catch {
		schools = [];
	}

	try {
		const { notifications, unread } = await api<{
			notifications: NotificationSummary[];
			unread: number;
		}>('/v1/notifications?limit=5', {
			token: locals.token,
			tenant: session.tenant.slug,
			fetch
		});
		// Being a guardian is what puts the family page on the menu.
		let isGuardian = false;
		try {
			const { children } = await api<{ children: Child[] }>('/v1/children', {
				token: locals.token,
				tenant: session.tenant.slug,
				fetch
			});
			isGuardian = (children ?? []).length > 0;
		} catch {
			isGuardian = false;
		}

		return { recentNotifications: notifications, unread, schools, ai: aiEnabled(), isGuardian };
	} catch {
		return {
			recentNotifications: [] as NotificationSummary[],
			unread: 0,
			schools: [] as TenantSummary[],
			ai: aiEnabled(),
			isGuardian: false
		};
	}
};
