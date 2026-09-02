<script lang="ts">
	import SevenSegment from '$lib/components/SevenSegment.svelte';

	let { seconds }: { seconds: number } = $props();

	// Counted from a fixed end time, so a backgrounded tab does not drift away
	// from what the server will enforce. Display only.
	let endsAt = $derived(Date.now() + seconds * 1000);
	let now = $state(Date.now());

	$effect(() => {
		const id = setInterval(() => (now = Date.now()), 200);
		return () => clearInterval(id);
	});

	let left = $derived(Math.max(0, Math.round((endsAt - now) / 1000)));
	let minutes = $derived(String(Math.floor(left / 60)).padStart(2, '0'));
	let secs = $derived(String(left % 60).padStart(2, '0'));

	let urgent = $derived(left > 0 && left <= 60);
	let warning = $derived(left > 60 && left <= 300);
	let done = $derived(left === 0);
	let blink = $derived(Math.floor(now / 500) % 2 === 0);

	let spoken = $derived(
		done ? 'Time is up' : urgent ? `${left} seconds left` : `${Math.ceil(left / 60)} minutes left`
	);
</script>

<!-- The case, the display, the glow: it should read as the clock on a desk. -->
<div class="inline-flex flex-col items-center gap-1.5 rounded-2xl border border-line bg-raised p-2.5">
	<div
		class="flex items-center gap-1.5 rounded-xl px-3 py-2"
		class:clock-face={true}
		class:text-accent={warning}
		class:text-danger={urgent || done}
		class:text-ink={!warning && !urgent && !done}
	>
		<SevenSegment digit={minutes[0]} />
		<SevenSegment digit={minutes[1]} />

		<span
			class="flex h-11 flex-col justify-center gap-2 px-0.5 transition-opacity duration-150"
			class:opacity-20={!blink && !done}
			aria-hidden="true"
		>
			<span class="block size-1.5 rounded-full bg-current"></span>
			<span class="block size-1.5 rounded-full bg-current"></span>
		</span>

		<SevenSegment digit={secs[0]} />
		<SevenSegment digit={secs[1]} />
	</div>

	<span
		class="font-mono text-[0.6rem] tracking-[0.2em] uppercase"
		class:text-ink-faint={!urgent && !done}
		class:text-danger={urgent || done}
		aria-hidden="true"
	>
		{done ? 'time up' : 'time left'}
	</span>

	<span class="sr-only" role="timer" aria-live={urgent ? 'assertive' : 'off'}>{spoken}</span>
</div>

<style>
	/* An inset, near-black face so the lit segments read as glowing rather than
	   printed. Kept out of the token set on purpose: this is a physical object. */
	.clock-face {
		background: #0c0e11;
		box-shadow: inset 0 1px 3px rgb(0 0 0 / 0.6);
		filter: drop-shadow(0 0 3px currentColor);
	}

	@media (prefers-reduced-motion: reduce) {
		.clock-face :global(*) {
			transition: none !important;
		}
	}
</style>
