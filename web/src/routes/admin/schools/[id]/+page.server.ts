import type { PageServerLoad } from './$types';
import { admin } from '$lib/server/admin';

type Detail = {
	tenant: {
		id: string;
		slug: string;
		name: string;
		kind: string;
		status: string;
		demo: boolean;
		locale: string;
		currency: string;
		created_at: string;
	};
	members: { user_id: string; full_name: string; contact: string; role: string; since: string }[];
	courses: { title: string; status: string; learners: number }[];
	orders: {
		reference: string;
		status: string;
		provider: string;
		amount_minor: number;
		currency: string;
		created_at: string;
	}[];
	certificates: number;
};

export const load: PageServerLoad = async ({ cookies, fetch, params }) => ({
	school: await admin<Detail>(`/v1/admin/schools/${params.id}`, cookies, fetch)
});
