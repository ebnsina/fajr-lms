import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { api } from '$lib/server/api';

type Row = {
	order: {
		id: string;
		reference: string;
		amount_minor: number;
		currency: string;
		provider: string;
		note: string;
	};
	title: string;
	full_name: string;
};

export const load: PageServerLoad = async ({ locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	const { orders } = await api<{ orders: Row[] }>('/v1/orders/review?limit=50', {
		token: locals.token,
		tenant: session.tenant.slug,
		fetch
	});
	return { orders };
};
