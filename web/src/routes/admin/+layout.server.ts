import { redirect } from '@sveltejs/kit';
import type { LayoutServerLoad } from './$types';
import { STAFF_COOKIE } from '$lib/server/session';

// Everything under /admin needs the back office's own sign-in. The login page
// is the one exception, and it says so itself.
export const load: LayoutServerLoad = ({ cookies, url }) => {
	const token = cookies.get(STAFF_COOKIE);
	if (!token && url.pathname !== '/admin/login') {
		redirect(303, '/admin/login');
	}
	return { staff: Boolean(token) };
};
