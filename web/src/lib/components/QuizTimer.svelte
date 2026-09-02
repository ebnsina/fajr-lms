<script lang="ts">
	let { seconds }: { seconds: number } = $props();

	// Counted from a fixed end time, so a backgrounded tab does not drift away
	// from what the server will enforce. Display only.
	let endsAt = $derived(Date.now() + seconds * 1000);
	let now = $state(Date.now());

	$effect(() => {
		const id = setInterval(() => (now = Date.now()), 250);
		return () => clearInterval(id);
	});

	let left = $derived(Math.max(0, Math.round((endsAt - now) / 1000)));
	let minutes = $derived(String(Math.floor(left / 60)).padStart(2, '0'));
	let secs = $derived(String(left % 60).padStart(2, '0'));
	let digits = $derived([minutes[0], minutes[1], secs[0], secs[1]]);

	let urgent = $derived(left > 0 && left <= 60);
	let warning = $derived(left > 60 && left <= 300);
	let done = $derived(left === 0);

	// Read out sparingly: every second would be unusable, the last minute matters.
	let spoken = $derived(
		done
			? 'Time is up'
			: urgent
				? `${left} seconds left`
				: `${Math.ceil(left / 60)} minutes left`
	);
</script>

<div
	class="inline-flex items-center gap-2 rounded-xl border px-3 py-2 transition-colors"
	class:border-line={!urgent && !done}
	class:bg-sunken={!urgent && !done}
	class:border-danger={urgent || done}
	class:bg-danger-soft={urgent || done}
>
	<div class="flex items-center gap-1" aria-hidden="true">
		{#each digits as digit, index (index)}
			{#if index === 2}
				<span
					class="px-0.5 pb-0.5 font-mono text-xl leading-none font-semibold"
					class:text-ink-faint={!urgent && !done}
					class:text-danger={urgent || done}
					class:animate-pulse={!done}
				>
					:
				</span>
			{/if}
			<span
				class="grid size-8 place-content-center rounded-lg border font-mono text-lg leading-none font-semibold tabular-nums transition-colors"
				class:border-line={!urgent && !done}
				class:bg-surface={!urgent && !done}
				class:text-ink={!urgent && !done && !warning}
				class:text-accent={warning}
				class:border-danger-line={urgent || done}
				class:bg-danger-soft={urgent || done}
				class:text-danger={urgent || done}
			>
				{digit}
			</span>
		{/each}
	</div>

	<span class="sr-only" role="timer" aria-live={urgent ? 'assertive' : 'off'}>{spoken}</span>
	<span
		class="text-xs font-medium tracking-[0.08em] uppercase"
		class:text-ink-faint={!urgent && !done}
		class:text-danger={urgent || done}
		aria-hidden="true"
	>
		{done ? 'time up' : 'left'}
	</span>
</div>

<style>
	@media (prefers-reduced-motion: reduce) {
		:global(.animate-pulse) {
			animation: none;
		}
	}
</style>
