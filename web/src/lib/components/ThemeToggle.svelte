<script lang="ts">
	import Monitor from '@lucide/svelte/icons/monitor';
	import Sun from '@lucide/svelte/icons/sun';
	import Moon from '@lucide/svelte/icons/moon';

	type Theme = 'light' | 'dark' | 'system';

	let { theme }: { theme: Theme } = $props();

	// One compact control for the entry pages, where a full radio group is more
	// furniture than the screen can justify. Settings has the expanded version.
	let chosen = $state<Theme | null>(null);
	let current = $derived(chosen ?? theme);

	const order: Theme[] = ['system', 'light', 'dark'];
	const labels: Record<Theme, string> = { system: 'Auto', light: 'Light', dark: 'Dark' };
	const icons = { system: Monitor, light: Sun, dark: Moon };
	let Icon = $derived(icons[current]);

	async function cycle() {
		const next = order[(order.indexOf(current) + 1) % order.length];
		chosen = next;

		const root = document.documentElement;
		if (next === 'system') root.removeAttribute('data-theme');
		else root.setAttribute('data-theme', next);

		await fetch('/theme', {
			method: 'POST',
			headers: { 'content-type': 'application/json' },
			body: JSON.stringify({ theme: next })
		}).catch(() => {});
	}
</script>

<button
	class="btn btn-sm btn-quiet"
	type="button"
	onclick={cycle}
	aria-label="Appearance: {labels[current]}. Change it."
	title="Appearance: {labels[current]}"
>
	<Icon size={15} strokeWidth={2} aria-hidden="true" />
	{labels[current]}
</button>
