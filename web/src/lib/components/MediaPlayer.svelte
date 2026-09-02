<script lang="ts">
	import Play from '@lucide/svelte/icons/play';
	import CloudOff from '@lucide/svelte/icons/cloud-off';
	import type { Playback } from '$lib/api';

	let { playback, title }: { playback: Playback | null; title: string } = $props();

	// Nothing loads until asked. On a metered connection an autoloaded embed can
	// cost a learner more than the lesson is worth.
	let started = $state(false);
</script>

{#if !playback || playback.kind === 'not_ready'}
	<div class="card flex items-center gap-3 text-sm text-ink-soft" dir="auto">
		<CloudOff size={18} aria-hidden="true" />
		<p class="mb-0">The video for this lesson is not ready yet.</p>
	</div>
{:else if started}
	{#if playback.kind === 'embed'}
		<div class="aspect-video w-full overflow-hidden rounded-3xl border border-line bg-sunken">
			<iframe
				class="h-full w-full"
				src={playback.url}
				{title}
				loading="lazy"
				allow="accelerometer; encrypted-media; picture-in-picture; fullscreen"
				allowfullscreen
				sandbox="allow-scripts allow-same-origin allow-presentation"
			></iframe>
		</div>
	{:else}
		<!-- svelte-ignore a11y_media_has_caption -->
		<video class="w-full rounded-3xl border border-line bg-sunken" src={playback.url} controls
		></video>
	{/if}
{:else}
	<button
		class="flex aspect-video w-full cursor-pointer flex-col items-center justify-center gap-2 rounded-3xl border border-line bg-raised text-ink-soft transition hover:border-line-strong"
		type="button"
		onclick={() => (started = true)}
	>
		<span
			class="flex size-14 items-center justify-center rounded-full border border-line bg-surface text-brand-text"
		>
			<Play size={22} fill="currentColor" aria-hidden="true" />
		</span>
		<span class="font-medium text-ink">Play this lesson</span>
		<span class="text-sm">Nothing downloads until you press play</span>
	</button>
{/if}
