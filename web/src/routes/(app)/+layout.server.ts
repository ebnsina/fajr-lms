import type { LayoutServerLoad } from './$types';
import { api } from '$lib/server/api';

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
		return { recentNotifications: [] as NotificationSummary[], unread: 0 };
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
		return { recentNotifications: notifications, unread };
	} catch {
		return { recentNotifications: [] as NotificationSummary[], unread: 0 };
	}
};
