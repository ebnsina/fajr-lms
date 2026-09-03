import type { Cookies } from '@sveltejs/kit';

export const TOKEN_COOKIE = 'fajr_session';
export const TENANT_COOKIE = 'fajr_tenant';
export const THEME_COOKIE = 'fajr_theme';
// The back office signs in separately, so opening a school does not sign you
// out of it and neither does the other way round.
export const STAFF_COOKIE = 'fajr_staff';

const year = 60 * 60 * 24 * 365;

/** The token never reaches client JavaScript, so a script injection cannot read it. */
export function saveSession(cookies: Cookies, token: string, secure: boolean) {
	cookies.set(TOKEN_COOKIE, token, {
		path: '/',
		httpOnly: true,
		sameSite: 'lax',
		secure,
		maxAge: 30 * 24 * 60 * 60
	});
}

export function saveTenant(cookies: Cookies, slug: string, secure: boolean) {
	cookies.set(TENANT_COOKIE, slug, {
		path: '/',
		httpOnly: true,
		sameSite: 'lax',
		secure,
		maxAge: year
	});
}

export function saveStaff(cookies: Cookies, token: string, secure: boolean) {
	cookies.set(STAFF_COOKIE, token, {
		path: '/admin',
		httpOnly: true,
		sameSite: 'lax',
		secure,
		maxAge: 12 * 60 * 60
	});
}

export function clearStaff(cookies: Cookies) {
	cookies.delete(STAFF_COOKIE, { path: '/admin' });
}

export function saveTheme(cookies: Cookies, theme: string, secure: boolean) {
	cookies.set(THEME_COOKIE, theme, {
		path: '/',
		sameSite: 'lax',
		secure,
		maxAge: year
	});
}

export function clearSession(cookies: Cookies) {
	cookies.delete(TOKEN_COOKIE, { path: '/' });
	cookies.delete(TENANT_COOKIE, { path: '/' });
}
