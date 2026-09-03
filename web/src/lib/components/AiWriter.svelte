<script lang="ts">
	import { createChat, fetchServerSentEvents } from '@tanstack/ai-svelte';
	import FluidOrb from '$lib/components/FluidOrb.svelte';

	// A drafting hand for whatever a teacher is writing. It never saves: the
	// text arrives in the field, and the teacher decides what to keep.
	let {
		what,
		context = '',
		enabled = true,
		onText
	}: {
		what: string;
		context?: string;
		enabled?: boolean;
		onText: (text: string) => void;
	} = $props();

	let asking = $state(false);
	let instruction = $state('');
	let problem = $state('');

	// The chat is made when the panel is opened, and only in a browser: it
	// holds a live connection, which a server render has no use for.
	type Chat = ReturnType<typeof createChat>;
	let chat = $state<Chat | null>(null);

	function open() {
		asking = true;
		chat ??= createChat({
			connection: fetchServerSentEvents('/ai/write'),
			onError: () => {
				problem = 'Fajr AI could not finish that. Try again.';
			}
		});
	}

	// The last thing the assistant said, streamed in as it arrives.
	const answer = $derived(
		(chat?.messages ?? [])
			.filter((message) => message.role === 'assistant')
			.at(-1)
			?.parts.filter((part) => part.type === 'text')
			.map((part) => part.content)
			.join('') ?? ''
	);

	$effect(() => {
		if (answer) onText(answer);
	});

	async function ask() {
		if (!instruction.trim() || !chat) return;
		problem = '';
		const brief = context.trim()
			? `${what}. ${instruction.trim()}\n\nWhat is written so far:\n${context.trim()}`
			: `${what}. ${instruction.trim()}`;
		await chat.sendMessage(brief);
	}
</script>

<div class="mt-2">
	{#if !asking}
		<button
			class="inline-flex items-center gap-2 text-sm text-ink-soft hover:text-ink"
			type="button"
			onclick={open}
		>
			<FluidOrb size={18} label="" />
			{enabled ? 'Ask Fajr AI to draft this' : 'Fajr AI'}
		</button>
	{:else}
		<div class="rounded-xl border border-line bg-raised p-3">
			<div class="mb-2 flex items-center gap-2 text-sm font-medium">
				<FluidOrb size={18} label="" />
				Fajr AI
				<button
					class="ms-auto text-sm font-normal text-ink-soft hover:text-ink"
					type="button"
					onclick={() => (asking = false)}
				>
					Close
				</button>
			</div>

			{#if !enabled}
				<p class="mb-0 text-sm text-ink-soft">
					Fajr AI is not configured. Ask whoever runs this installation to turn it on.
				</p>
			{:else}
				<div class="flex flex-wrap items-end gap-2">
					<input
						class="field min-w-48 flex-1"
						bind:value={instruction}
						dir="auto"
						placeholder="say what you want, e.g. three paragraphs for beginners"
						onkeydown={(event) => {
							if (event.key === 'Enter') {
								event.preventDefault();
								ask();
							}
						}}
					/>
					{#if chat?.isLoading}
						<button class="btn btn-sm btn-quiet" type="button" onclick={() => chat?.stop()}>
							Stop
						</button>
					{:else}
						<button class="btn btn-sm" type="button" onclick={ask}>Draft</button>
					{/if}
				</div>

				{#if problem}
					<p class="banner-bad mt-3 mb-0 text-sm" role="alert">{problem}</p>
				{:else if chat?.isLoading}
					<p class="mt-3 mb-0 text-sm text-ink-soft" aria-live="polite">Writing…</p>
				{:else if answer}
					<p class="mt-3 mb-0 text-sm text-ink-soft">
						Written into the field above. Change anything you like before saving.
					</p>
				{/if}
			{/if}
		</div>
	{/if}
</div>
