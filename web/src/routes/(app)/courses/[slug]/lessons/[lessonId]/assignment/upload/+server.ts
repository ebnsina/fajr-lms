import { json, redirect } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { api, ApiFailure } from '$lib/server/api';

export const POST: RequestHandler = async ({ request, locals, cookies, fetch }) => {
	const tenant = cookies.get('fajr_tenant');
	if (!locals.token || !tenant) redirect(303, '/login');

	const payload = await request.json().catch(() => null);
	if (!payload || typeof payload !== 'object') {
		return json({ error: 'Send a JSON body.' }, { status: 422 });
	}

	const scoped = { token: locals.token, tenant, fetch };
	try {
		// step: "sign" reserves an upload target, "complete" confirms the bytes landed.
		if (payload.step === 'complete') {
			return json(
				await api(`/v1/media/${payload.media_id}/complete`, {
					method: 'POST',
					...scoped
				})
			);
		}
		return json(
			await api('/v1/media', {
				method: 'POST',
				body: {
					provider: 'file',
					filename: payload.filename,
					content_type: payload.content_type,
					byte_size: payload.byte_size,
					kind: 'pdf'
				},
				...scoped
			})
		);
	} catch (failure) {
		if (failure instanceof ApiFailure) {
			return json({ error: failure.error.message }, { status: failure.status });
		}
		throw failure;
	}
};
