import { error, fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';
import type { Outline } from '$lib/api';

type Provider = {
	name: string;
	label: string;
	needs_proof: boolean;
};

type Instruction = {
	kind: string;
	reference: string;
	needs_proof: boolean;
	redirect_url?: string;
	fields?: Record<string, string>;
};

type Order = {
	id: string;
	reference: string;
	status: string;
	amount_minor: number;
	currency: string;
	part_no: number | null;
	instruction?: Instruction;
};

type Plan = {
	payment_plan: {
		id: string;
		total_minor: number;
		currency: string;
		parts: number;
		paid_parts: number;
		next_due_on: string | null;
	};
	title: string;
	slug: string;
};

export const load: PageServerLoad = async ({ params, locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	const scoped = { token: locals.token, tenant: session.tenant.slug, fetch };
	let outline: Outline;
	try {
		outline = await api<Outline>(`/v1/courses/${params.slug}`, scoped);
	} catch (cause) {
		if (cause instanceof ApiFailure && cause.status === 404) {
			error(404, 'That course does not exist here.');
		}
		throw cause;
	}
	if (outline.course.price_minor <= 0) redirect(303, `/courses/${params.slug}`);

	const [providers, orders, plans] = await Promise.all([
		api<{ providers: Provider[] }>('/v1/payment/providers', scoped),
		api<{ orders: Order[] }>('/v1/orders', scoped),
		api<{ plans: Plan[] }>('/v1/plans', scoped)
	]);

	const open = (orders.orders ?? []).find(
		(order) => order.status === 'pending' || order.status === 'awaiting_review'
	);
	return {
		course: outline.course,
		providers: providers.providers ?? [],
		open: open ?? null,
		plan: (plans.plans ?? []).find((row) => row.slug === params.slug) ?? null,
		slug: params.slug
	};
};

const scoped = (
	locals: App.Locals,
	cookies: { get: (name: string) => string | undefined },
	fetch: typeof globalThis.fetch
) => {
	const tenant = cookies.get('fajr_tenant');
	if (!locals.token || !tenant) redirect(303, '/login');
	return { token: locals.token, tenant, fetch };
};

export const actions: Actions = {
	order: async ({ params, request, locals, cookies, fetch }) => {
		const form = await request.formData();
		const call = scoped(locals, cookies, fetch);
		let courseID: string;
		try {
			const outline = await api<Outline>(`/v1/courses/${params.slug}`, call);
			courseID = outline.course.id;
		} catch (cause) {
			if (cause instanceof ApiFailure) return fail(cause.status, { message: cause.error.message });
			throw cause;
		}

		try {
			const order = await api<Order>(`/v1/courses/${courseID}/orders`, {
				method: 'POST',
				body: {
					provider: String(form.get('provider') ?? ''),
					coupon: String(form.get('coupon') ?? '').trim(),
					in_parts: form.get('in_parts') === 'on'
				},
				...call
			});
			// A gateway takes over from here; a bank transfer stays on this page.
			if (order.instruction?.redirect_url) redirect(303, order.instruction.redirect_url);
			return { ordered: true };
		} catch (cause) {
			if (cause instanceof ApiFailure) return fail(cause.status, { message: cause.error.message });
			throw cause;
		}
	},

	proof: async ({ request, locals, cookies, fetch }) => {
		const form = await request.formData();
		const reference = String(form.get('provider_ref') ?? '').trim();
		const media = String(form.get('media_id') ?? '').trim();
		if (!reference && !media) {
			return fail(422, { message: 'Give the transaction id, or attach the slip.' });
		}

		try {
			await api(`/v1/orders/${form.get('order_id')}/proof`, {
				method: 'POST',
				body: {
					provider_ref: reference,
					media_id: media,
					note: String(form.get('note') ?? '').trim()
				},
				...scoped(locals, cookies, fetch)
			});
			return { sent: true };
		} catch (cause) {
			if (cause instanceof ApiFailure) return fail(cause.status, { message: cause.error.message });
			throw cause;
		}
	}
};
