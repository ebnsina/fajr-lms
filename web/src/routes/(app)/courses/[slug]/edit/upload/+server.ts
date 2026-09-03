import { json, error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { api, ApiFailure } from '$lib/server/api';

type Prepared = {
	id: string;
	upload?: { url: string; method: string; headers?: Record<string, string>; max_bytes: number };
};

/** The bytes go straight from the browser to storage; only the signature comes
    through here, because the API token never leaves the server. */
export const POST: RequestHandler = async ({ request, locals, cookies, fetch }) => {
	const tenant = cookies.get('fajr_tenant');
	if (!locals.token || !tenant) error(401, 'Sign in first.');
	const scoped = { token: locals.token, tenant, fetch };

	const body = await request.json();

	try {
		if (body.step === 'prepare') {
			const prepared = await api<Prepared>('/v1/media', {
				method: 'POST',
				body: {
					provider: 'file',
					filename: body.filename,
					content_type: body.content_type,
					byte_size: body.byte_size,
					title: body.title,
					kind: body.kind
				},
				...scoped
			});
			if (!prepared.upload) {
				error(503, 'This school has no file storage configured yet.');
			}
			return json(prepared);
		}

		if (body.step === 'finish') {
			await api(`/v1/media/${body.media_id}/complete`, { method: 'POST', ...scoped });
			await api(`/v1/lessons/${body.lesson_id}/media`, {
				method: 'PUT',
				body: { media_id: body.media_id },
				...scoped
			});
			return json({ attached: true });
		}
	} catch (cause) {
		if (cause instanceof ApiFailure) error(cause.status, cause.error.message);
		throw cause;
	}

	error(422, 'Unknown step.');
};
