import { fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';
import type { Playback } from '$lib/api';

type Row = {
	order: {
		id: string;
		reference: string;
		amount_minor: number;
		currency: string;
		provider: string;
		provider_ref: string;
		note: string;
		proof_media_id: string | null;
		created_at: string;
	};
	title: string;
	full_name: string;
};

export const load: PageServerLoad = async ({ locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	const scoped = { token: locals.token, tenant: session.tenant.slug, fetch };
	const { orders } = await api<{ orders: Row[] }>('/v1/orders/review?limit=50', scoped);

	// The slip is the whole point of the review, so resolve a link for each one.
	const withProof = await Promise.all(
		(orders ?? []).map(async (row) => {
			if (!row.order.proof_media_id) return { ...row, proof: null as Playback | null };
			try {
				const proof = await api<Playback>(`/v1/media/${row.order.proof_media_id}/playback`, scoped);
				return { ...row, proof };
			} catch {
				return { ...row, proof: null as Playback | null };
			}
		})
	);
	return { orders: withProof };
};

export const actions: Actions = {
	review: async ({ request, locals, cookies, fetch }) => {
		const tenant = cookies.get('fajr_tenant');
		if (!locals.token || !tenant) redirect(303, '/login');

		const form = await request.formData();
		const decision = String(form.get('decision') ?? '');
		if (decision !== 'approve' && decision !== 'reject') {
			return fail(422, { message: 'Choose approve or reject.' });
		}

		try {
			await api(`/v1/orders/${form.get('order_id')}/review`, {
				method: 'POST',
				body: { decision, note: String(form.get('note') ?? '') },
				token: locals.token,
				tenant,
				fetch
			});
		} catch (failure) {
			if (failure instanceof ApiFailure) return fail(failure.status, { message: failure.error.message });
			throw failure;
		}
		return { decided: decision };
	}
};
