import { error, type RequestHandler } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';

const base = () => (env.FAJR_API_URL ?? 'http://localhost:8080').replace(/\/$/, '');

// The paper a laid-out certificate is printed on, passed through with the page
// itself so the public link is one origin.
export const GET: RequestHandler = async ({ params, fetch }) => {
	const serial = encodeURIComponent(params.serial ?? '');
	let response: Response;
	try {
		response = await fetch(`${base()}/verify/${serial}/background`);
	} catch {
		error(502, 'The certificate service is not answering. Try again in a moment.');
	}
	if (!response.ok) error(response.status === 404 ? 404 : 502, 'No background for that serial.');

	return new Response(response.body, {
		status: 200,
		headers: {
			'content-type': response.headers.get('content-type') ?? 'application/octet-stream',
			'x-content-type-options': 'nosniff',
			'content-security-policy': "sandbox; default-src 'none'; style-src 'unsafe-inline'",
			'cache-control': 'public, max-age=3600'
		}
	});
};
