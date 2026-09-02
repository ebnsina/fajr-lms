import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';
import type { SitePage, SiteNavItem, SiteCourse } from '$lib/types.site';

type SiteResponse = {
	page: SitePage;
	nav: SiteNavItem[];
	courses?: SiteCourse[];
	theme?: string;
};

export const load: PageServerLoad = async ({ params, fetch }) => {
	const path = params.slug ? `/site/${params.tenant}/${params.slug}` : `/site/${params.tenant}`;
	try {
		const { page, nav, courses, theme } = await api<SiteResponse>(path, {
			fetch
		});
		return {
			page,
			nav: nav ?? [],
			courses: courses ?? [],
			theme: theme ?? 'plain',
			tenant: params.tenant
		};
	} catch (cause) {
		if (cause instanceof ApiFailure && cause.status === 404) {
			error(404, 'This page is not part of the site.');
		}
		if (cause instanceof ApiFailure) error(cause.status, cause.error.message);
		throw cause;
	}
};
