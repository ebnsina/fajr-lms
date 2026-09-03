<script lang="ts">
	import type { Component } from 'svelte';

	let {
		label,
		value,
		hint,
		href,
		icon: Icon
	}: {
		label: string;
		value: number;
		hint?: string;
		href?: string;
		icon: Component;
	} = $props();

	let shown = $derived(new Intl.NumberFormat().format(value));
</script>

{#snippet body()}
	<span class="icon-tile shrink-0">
		<Icon size={18} aria-hidden="true" />
	</span>
	<span class="min-w-0">
		<span class="block font-mono text-2xl leading-tight font-medium">{shown}</span>
		<span class="block text-sm text-ink-soft" dir="auto">{label}</span>
		{#if hint}
			<span class="mt-0.5 block text-xs text-ink-faint" dir="auto">{hint}</span>
		{/if}
	</span>
{/snippet}

{#if href}
	<a class="card flex items-start gap-3 p-4 transition-colors hover:border-line-strong" {href}>
		{@render body()}
	</a>
{:else}
	<div class="card flex items-start gap-3 p-4">{@render body()}</div>
{/if}
