import { error, fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';

export type CertField = {
	token: string;
	x: number;
	y: number;
	size: number;
	align: 'start' | 'center' | 'end';
	bold: boolean;
	color: string;
	label: string;
};

export const load: PageServerLoad = async ({ locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');
	if (!['owner', 'admin'].includes(session.tenant.role)) {
		error(403, 'Only the office can lay out the certificate.');
	}

	const layout = await api<{ fields: CertField[]; has_background: boolean }>(
		'/v1/certificates/layout',
		{ token: locals.token, tenant: session.tenant.slug, fetch }
	);
	return {
		fields: layout.fields ?? [],
		hasBackground: layout.has_background,
		school: session.tenant.name
	};
};

export const actions: Actions = {
	save: async ({ request, locals, cookies, fetch }) => {
		const tenant = cookies.get('fajr_tenant');
		if (!locals.token || !tenant) redirect(303, '/login');

		const form = await request.formData();
		let fields: CertField[];
		try {
			fields = JSON.parse(String(form.get('fields') ?? '[]')) as CertField[];
		} catch {
			return fail(422, { message: 'That layout could not be read. Try again.' });
		}

		try {
			await api('/v1/certificates/layout', {
				method: 'PUT',
				body: { fields },
				token: locals.token,
				tenant,
				fetch
			});
			return { saved: true };
		} catch (cause) {
			if (cause instanceof ApiFailure) return fail(cause.status, { message: cause.error.message });
			throw cause;
		}
	}
};
