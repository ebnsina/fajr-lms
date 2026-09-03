import { fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';

type Section = {
	section: { id: string; class_id: string; name: string };
	class_name: string;
	students: number;
};

export const load: PageServerLoad = async ({ locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	const scoped = { token: locals.token, tenant: session.tenant.slug, fetch };
	const { sections } = await api<{ sections: Section[] }>('/v1/academics/classes', scoped);
	return { sections: sections ?? [] };
};

export const actions: Actions = {
	send: async ({ request, locals, cookies, fetch }) => {
		const tenant = cookies.get('fajr_tenant');
		if (!locals.token || !tenant) redirect(303, '/login');

		const form = await request.formData();
		const audience = String(form.get('audience') ?? 'school');
		const title = String(form.get('title') ?? '').trim();
		const body = String(form.get('body') ?? '').trim();
		if (!title || !body) return fail(422, { message: 'A notice needs something to say.' });

		try {
			const sent = await api<{ sent_to: number }>('/v1/notices', {
				method: 'POST',
				body: {
					audience,
					target_id: audience === 'school' ? '' : String(form.get('target_id') ?? ''),
					to: String(form.get('to') ?? 'guardians'),
					title,
					body
				},
				token: locals.token,
				tenant,
				fetch
			});
			return { sentTo: sent.sent_to };
		} catch (cause) {
			if (cause instanceof ApiFailure) return fail(cause.status, { message: cause.error.message });
			throw cause;
		}
	}
};
