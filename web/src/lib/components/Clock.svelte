<script lang="ts">
	import SevenSegment from '$lib/components/SevenSegment.svelte';

	let now = $state(new Date());

	$effect(() => {
		const id = setInterval(() => (now = new Date()), 1000);
		return () => clearInterval(id);
	});

	// Every part comes from Intl, so a 12 hour locale gets its day period and a
	// 24 hour one gets none, without this file deciding which is which.
	let parts = $derived(
		new Intl.DateTimeFormat(undefined, { hour: '2-digit', minute: '2-digit' }).formatToParts(now)
	);
	let hours = $derived((parts.find((p) => p.type === 'hour')?.value ?? '00').padStart(2, '0'));
	let minutes = $derived((parts.find((p) => p.type === 'minute')?.value ?? '00').padStart(2, '0'));
	let period = $derived(parts.find((p) => p.type === 'dayPeriod')?.value ?? '');

	let today = $derived(
		new Intl.DateTimeFormat(undefined, {
			weekday: 'long',
			day: 'numeric',
			month: 'long'
		}).format(now)
	);
	let spoken = $derived(
		new Intl.DateTimeFormat(undefined, { dateStyle: 'full', timeStyle: 'short' }).format(now)
	);
	let blink = $derived(now.getSeconds() % 2 === 0);
</script>

<div class="flex shrink-0 flex-col items-end gap-1">
	<div class="flex items-center gap-1 text-ink" aria-hidden="true">
		<SevenSegment digit={hours[0]} />
		<SevenSegment digit={hours[1]} />
		<span
			class="flex h-11 flex-col justify-center gap-2 px-0.5 transition-opacity duration-150 motion-reduce:transition-none"
			class:opacity-20={!blink}
		>
			<span class="block size-1.5 rounded-full bg-current"></span>
			<span class="block size-1.5 rounded-full bg-current"></span>
		</span>
		<SevenSegment digit={minutes[0]} />
		<SevenSegment digit={minutes[1]} />
		{#if period}
			<span class="ms-1 self-end pb-1 font-mono text-xs text-ink-soft">{period}</span>
		{/if}
	</div>
	<span class="text-xs text-ink-soft" aria-hidden="true">{today}</span>
	<span class="sr-only">{spoken}</span>
</div>
