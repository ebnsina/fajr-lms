<script lang="ts">
	import Monitor from '@lucide/svelte/icons/monitor';
	import Moon from '@lucide/svelte/icons/moon';
	import Sun from '@lucide/svelte/icons/sun';
	import type { Component } from 'svelte';

	type Theme = 'light' | 'dark' | 'system';

	let { theme }: { theme: Theme } = $props();
	// Held locally so the control reacts at once, seeded from what the server stamped.
	let chosen = $state<Theme | null>(null);
	let current = $derived(chosen ?? theme);

	const options: { value: Theme; label: string; icon: Component }[] = [
		{ value: 'system', label: 'Auto', icon: Monitor },
		{ value: 'light', label: 'Light', icon: Sun },
		{ value: 'dark', label: 'Dark', icon: Moon }
	];

	async function choose(next: Theme) {
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

<div class="flex flex-wrap gap-2" role="radiogroup" aria-label="Appearance">
	{#each options as option (option.value)}
		<button
			class="btn btn-sm"
			class:btn-quiet={current !== option.value}
			type="button"
			role="radio"
			aria-checked={current === option.value}
			onclick={() => choose(option.value)}
		>
			<option.icon size={16} aria-hidden="true" />
			{option.label}
		</button>
	{/each}
</div>
