import type { LayoutServerLoad } from './$types';
import { THEME_COOKIE } from '$lib/server/session';

export const load: LayoutServerLoad = ({ cookies }) => {
	const theme = cookies.get(THEME_COOKIE);
	return { theme: theme === 'light' || theme === 'dark' ? theme : 'system' };
};
