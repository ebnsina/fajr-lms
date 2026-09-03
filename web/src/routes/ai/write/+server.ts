import { error, type RequestHandler } from '@sveltejs/kit';
import { chat, toServerSentEventsResponse } from '@tanstack/ai';
import { aiAdapter, NOT_CONFIGURED } from '$lib/server/ai';

// One assistant for every place a teacher writes. What is being written comes
// in the message; the model answers with the text itself and nothing else, so
// what streams back can go straight into the field.
const SYSTEM = [
	'You help a teacher write course material for a school.',
	'Answer with the requested text only: no preamble, no sign-off, no markdown fences, no explanation of what you did.',
	'Write in the language the teacher wrote to you in.',
	'Keep to what the teacher gave you. Never invent facts, prices, dates or credentials.',
	'Plain, clear language a learner can read.'
].join(' ');

export const POST: RequestHandler = async ({ request, locals }) => {
	if (!locals.token) error(401, 'Sign in first.');

	const adapter = aiAdapter();
	if (!adapter) error(501, NOT_CONFIGURED);

	const { messages } = await request.json();
	if (!Array.isArray(messages) || messages.length === 0) {
		error(422, 'Say what you would like written.');
	}

	// A learner navigating away should not leave a request running.
	const abortController = new AbortController();
	request.signal.addEventListener('abort', () => abortController.abort());

	const stream = chat({
		adapter,
		messages,
		systemPrompts: [SYSTEM],
		abortController,
		modelOptions: { temperature: 0.5, max_tokens: 1500 }
	});
	return toServerSentEventsResponse(stream);
};
