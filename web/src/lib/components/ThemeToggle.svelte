<script lang="ts">
	import Monitor from '@lucide/svelte/icons/monitor';
	import Sun from '@lucide/svelte/icons/sun';
	import Moon from '@lucide/svelte/icons/moon';

	type Theme = 'light' | 'dark' | 'system';

	let { theme }: { theme: Theme } = $props();
	// Held locally so the button reacts at once, seeded from what the server stamped.
	let chosen = $state<Theme | null>(null);
	let current = $derived(chosen ?? theme);

	const order: Theme[] = ['system', 'light', 'dark'];
	const labels: Record<Theme, string> = { system: 'Auto', light: 'Light', dark: 'Dark' };
	const icons = { system: Monitor, light: Sun, dark: Moon };
	let Icon = $derived(icons[current]);

	async function cycle() {
		const next = order[(order.indexOf(current) + 1) % order.length];
		chosen = next;
		apply(next);
		// Stored server side, so the next full load is stamped before it paints.
		await fetch('/theme', {
			method: 'POST',
			headers: { 'content-type': 'application/json' },
			body: JSON.stringify({ theme: next })
		}).catch(() => {});
	}

	function apply(next: Theme) {
		const root = document.documentElement;
		if (next === 'system') root.removeAttribute('data-theme');
		else root.setAttribute('data-theme', next);
	}
</script>

<button
	class="btn btn-sm btn-quiet"
	type="button"
	onclick={cycle}
	aria-label="Appearance: {labels[current]}. Change it."
	title="Appearance: {labels[current]}"
>
	<Icon size={16} strokeWidth={2} aria-hidden="true" />
	{labels[current]}
</button>
