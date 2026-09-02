<script lang="ts">
	import Receipt from '@lucide/svelte/icons/receipt';
	import { money } from '$lib/api';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	let locale = $derived(data.session?.tenant?.locale ?? 'en');
</script>

<svelte:head><title>Payments · Fajr</title></svelte:head>

<header class="mb-6">
	<h1 class="text-2xl font-semibold tracking-tight" dir="auto">Payments to review</h1>
	<p class="mt-1 text-sm text-ink-soft" dir="auto">
		Bank transfers and wallet payments waiting for someone to confirm them.
	</p>
</header>

{#if data.orders.length === 0}
	<div class="card flex items-start gap-3 text-sm text-ink-soft" dir="auto">
		<Receipt class="mt-0.5 shrink-0" size={18} aria-hidden="true" />
		<p class="mb-0">Nothing waiting. Card and wallet payments confirm themselves.</p>
	</div>
{:else}
	<ul class="list-none space-y-2 p-0">
		{#each data.orders as row (row.order.id)}
			<li class="card flex flex-wrap items-center gap-3 p-4">
				<span class="min-w-0 flex-1">
					<span class="block font-medium" dir="auto">{row.full_name}</span>
					<span class="mt-0.5 block text-sm text-ink-soft" dir="auto">{row.title}</span>
					{#if row.order.note}
						<span class="mt-1 block text-sm text-ink-faint" dir="auto">{row.order.note}</span>
					{/if}
				</span>
				<span class="text-end">
					<span class="block font-medium" dir="auto">
						{money(row.order.amount_minor, row.order.currency, locale)}
					</span>
					<span class="mt-0.5 block font-mono text-sm text-ink-faint" dir="ltr">
						{row.order.reference}
					</span>
				</span>
			</li>
		{/each}
	</ul>
{/if}
