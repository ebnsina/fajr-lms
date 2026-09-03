import { env } from '$env/dynamic/private';
import { createAnthropicChat } from '@tanstack/ai-anthropic';
import { createOpenaiChat } from '@tanstack/ai-openai';
import { createGeminiChat } from '@tanstack/ai-gemini';
import { createOllamaChat } from '@tanstack/ai-ollama';

// Fajr AI, kept behind one seam. The provider is a setting, not a decision
// baked into the code: every adapter here comes from TanStack AI and answers
// the same way, so swapping one for another changes an environment variable
// and nothing else.
export type Provider = 'anthropic' | 'openai' | 'gemini' | 'ollama';

const DEFAULT_MODEL: Record<Provider, string> = {
	anthropic: 'claude-sonnet-5',
	openai: 'gpt-5.2',
	gemini: 'gemini-3.1-pro-preview',
	ollama: 'llama3.1'
};

// The words a person sees when nobody has switched it on. One sentence, and
// it says who can fix it.
export const NOT_CONFIGURED =
	'Fajr AI is not configured. Ask whoever runs this installation to turn it on.';

function chosen(): Provider {
	const named = (env.FAJR_AI_PROVIDER ?? '').trim().toLowerCase();
	if (named === 'openai' || named === 'gemini' || named === 'ollama' || named === 'anthropic') {
		return named;
	}
	// Ollama runs on the school's own machine and needs no key, so it is only
	// used when asked for by name.
	return 'anthropic';
}

export function aiModel(): string {
	return (env.FAJR_AI_MODEL ?? '').trim() || DEFAULT_MODEL[chosen()];
}

/** Whether Fajr AI can answer at all: a key, or a local model to talk to. */
export function aiEnabled(): boolean {
	if (chosen() === 'ollama') return Boolean((env.FAJR_AI_BASE_URL ?? '').trim());
	return Boolean((env.FAJR_AI_API_KEY ?? '').trim());
}

export function aiStatus(): { enabled: boolean; provider: Provider; model: string } {
	return { enabled: aiEnabled(), provider: chosen(), model: aiModel() };
}

/** The adapter to answer with, or null when nothing is configured. */
export function aiAdapter() {
	if (!aiEnabled()) return null;

	const model = aiModel();
	const key = (env.FAJR_AI_API_KEY ?? '').trim();
	const baseURL = (env.FAJR_AI_BASE_URL ?? '').trim();

	// Each adapter validates its own model name at request time; the cast is
	// what lets an installation name a model this build has never heard of.
	switch (chosen()) {
		case 'openai':
			return createOpenaiChat(model as Parameters<typeof createOpenaiChat>[0], key);
		case 'gemini':
			return createGeminiChat(model as Parameters<typeof createGeminiChat>[0], key);
		case 'ollama':
			// Ollama takes the host it runs on rather than a key.
			return createOllamaChat(model, baseURL);
		default:
			return createAnthropicChat(model as Parameters<typeof createAnthropicChat>[0], key);
	}
}
