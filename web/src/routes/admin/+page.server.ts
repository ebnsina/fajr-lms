import type { PageServerLoad } from './$types';
import { admin } from '$lib/server/admin';

export type Overview = {
	schools: number;
	demo_schools: number;
	people: number;
	leads: number;
	leads_this_week: number;
	leads_won: number;
	leads_converted: number;
	courses: number;
	enrollments: number;
	certificates: number;
	paid_orders: number;
	paid_minor: number;
};

export const load: PageServerLoad = async ({ cookies, fetch }) => ({
	numbers: await admin<Overview>('/v1/admin/overview', cookies, fetch)
});
