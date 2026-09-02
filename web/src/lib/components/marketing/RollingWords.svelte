<script lang="ts">
	import { onMount } from 'svelte';

	let { words, every = 2200 }: { words: string[]; every?: number } = $props();

	let index = $state(0);
	let paused = $state(false);

	onMount(() => {
		if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
			paused = true;
			return;
		}
		const timer = setInterval(() => (index = (index + 1) % words.length), every);
		return () => clearInterval(timer);
	});

	// The widest word holds the line, so the headline never reflows mid-roll.
	const widest = $derived(words.reduce((a, b) => (b.length > a.length ? b : a), ''));
</script>

<span class="roll" aria-label={words[0]}>
	<span class="ghost" aria-hidden="true">{widest}</span>
	{#each words as word, i (word)}
		<span
			class="word"
			class:in={i === index}
			class:out={i !== index}
			aria-hidden={i === index ? undefined : 'true'}
		>
			{word}
		</span>
	{/each}
</span>

<style>
	.roll {
		position: relative;
		display: inline-block;
		block-size: 1.15em;
		line-height: 1.15;
		vertical-align: -0.24em;
		overflow: hidden;
		text-align: center;
	}

	.ghost {
		visibility: hidden;
		display: block;
	}

	.word {
		position: absolute;
		inset-block-start: 0;
		inset-inline: 0;
		white-space: nowrap;
		color: var(--color-brand-text);
		transition:
			transform 520ms cubic-bezier(0.22, 1, 0.36, 1),
			opacity 320ms ease;
	}

	.in {
		transform: translateY(0);
		opacity: 1;
	}

	.out {
		transform: translateY(-0.9em);
		opacity: 0;
	}

	@media (prefers-reduced-motion: reduce) {
		.word {
			transition: none;
		}
	}
</style>
