<script lang="ts">
	import { onMount } from 'svelte';

	let { words, every = 2200 }: { words: string[]; every?: number } = $props();

	let index = $state(0);
	let widths = $state<number[]>([]);
	let sizer = $state<HTMLElement | null>(null);

	// The box follows the word it is showing, so a short word does not leave a
	// hole before the next one. Widths are measured rather than guessed, and
	// measured again once the display face has actually loaded.
	function measure() {
		if (!sizer) return;
		widths = [...sizer.children].map((child) => child.getBoundingClientRect().width);
	}

	onMount(() => {
		measure();
		document.fonts?.ready.then(measure);
		window.addEventListener('resize', measure);

		if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
			return () => window.removeEventListener('resize', measure);
		}
		const timer = setInterval(() => (index = (index + 1) % words.length), every);
		return () => {
			clearInterval(timer);
			window.removeEventListener('resize', measure);
		};
	});

	const width = $derived(widths[index]);
</script>

<span class="roll" style:inline-size={width ? `${width}px` : undefined} aria-label={words[0]}>
	<!-- Off-screen copies, only ever read for their width. -->
	<span class="sizer" bind:this={sizer} aria-hidden="true">
		{#each words as word (word)}<span>{word}</span>{/each}
	</span>

	{#if width === undefined}
		<span class="lead">{words[index]}</span>
	{/if}

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
		transition: inline-size 420ms cubic-bezier(0.22, 1, 0.36, 1);
	}

	.sizer {
		position: absolute;
		visibility: hidden;
		pointer-events: none;
		white-space: nowrap;
	}

	.sizer span {
		display: block;
		inline-size: max-content;
	}

	/* Before the measurements land, one word holds the line open. */
	.lead {
		visibility: hidden;
		white-space: nowrap;
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
		.roll,
		.word {
			transition: none;
		}
	}
</style>
