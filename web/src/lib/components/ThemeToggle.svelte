<script lang="ts">
	type Theme = 'light' | 'dark' | 'system';

	let { theme }: { theme: Theme } = $props();
	// Held locally so the button reacts at once, seeded from what the server stamped.
	let chosen = $state<Theme | null>(null);
	let current = $derived(chosen ?? theme);

	const order: Theme[] = ['system', 'light', 'dark'];
	const labels: Record<Theme, string> = { system: 'Auto', light: 'Light', dark: 'Dark' };

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
	class="btn btn-quiet px-3 py-1.5 text-sm"
	type="button"
	onclick={cycle}
	aria-label="Appearance: {labels[current]}. Change it."
>
	{labels[current]}
</button>
