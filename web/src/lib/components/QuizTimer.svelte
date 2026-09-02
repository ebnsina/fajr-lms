<script lang="ts">
	import Clock from '@lucide/svelte/icons/clock';

	let { seconds }: { seconds: number } = $props();

	// Display only. The server decides what has expired, so a paused tab or a
	// wrong clock changes nothing about the mark.
	// Counted from a fixed end time, so a slow tab does not drift the display
	// away from what the server will enforce.
	let endsAt = $derived(Date.now() + seconds * 1000);
	let now = $state(Date.now());

	$effect(() => {
		const id = setInterval(() => (now = Date.now()), 1000);
		return () => clearInterval(id);
	});

	let left = $derived(Math.max(0, Math.round((endsAt - now) / 1000)));

	let label = $derived(
		`${Math.floor(left / 60)}:${String(left % 60).padStart(2, '0')}`
	);
	let low = $derived(left > 0 && left < 60);
</script>

<span
	class="chip font-mono tabular-nums"
	class:chip-brand={!low}
	class:text-danger={low}
	aria-live={low ? 'polite' : 'off'}
>
	<Clock size={13} aria-hidden="true" />
	{left > 0 ? label : 'Time is up'}
</span>
