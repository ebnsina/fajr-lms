import type { LayoutServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';
import { clearSession } from '$lib/server/session';
import type { Session, Tenant, User, Membership } from '$lib/types';

export const load: LayoutServerLoad = async ({ locals, cookies, fetch }) => {
	const empty: Session = { user: { id: '', full_name: '' }, memberships: [], tenant: null };
	const theme = locals.theme ?? 'system';
	if (!locals.token) return { session: null, theme };

	try {
		const me = await api<{ user: User; memberships: Membership[] }>('/v1/me', {
			token: locals.token,
			fetch
		});

		let tenant: Tenant | null = null;
		if (locals.tenantSlug) {
			try {
				tenant = await api<Tenant>('/v1/tenant', {
					token: locals.token,
					tenant: locals.tenantSlug,
					fetch
				});
			} catch {
				// The membership may have been removed; fall back to picking again.
				tenant = null;
			}
		}

		locals.dir = tenant?.default_dir === 'rtl' ? 'rtl' : 'ltr';
		locals.lang = tenant?.locale ?? 'en';
		return { session: { ...empty, user: me.user, memberships: me.memberships, tenant }, theme };
	} catch (error) {
		if (error instanceof ApiFailure && error.status === 401) clearSession(cookies);
		return { session: null, theme };
	}
};
