import { error, type RequestHandler } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';

const base = () => (env.FAJR_API_URL ?? 'http://localhost:8080').replace(/\/$/, '');

// A certificate is checked by whoever was handed one, so the link lives on the
// main site. The page itself is the API's, kept as the one implementation.
export const GET: RequestHandler = async ({ params, request, fetch }) => {
	const serial = encodeURIComponent(params.serial ?? '');
	let response: Response;
	try {
		response = await fetch(`${base()}/verify/${serial}`, {
			headers: { accept: request.headers.get('accept') ?? 'text/html' }
		});
	} catch {
		error(502, 'The certificate service is not answering. Try again in a moment.');
	}

	const body = await response.text();
	return new Response(body, {
		status: response.status,
		headers: {
			'content-type': response.headers.get('content-type') ?? 'text/html; charset=utf-8',
			'cache-control': 'public, max-age=300'
		}
	});
};
