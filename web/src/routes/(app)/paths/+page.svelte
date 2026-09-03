<script lang="ts">
	import { enhance } from '$app/forms';
	import Route from '@lucide/svelte/icons/route';
	import Package from '@lucide/svelte/icons/package';
	import { money } from '$lib/api';
	import type { PageProps } from './$types';
	import type { Collection } from './+page.server';

	let { data, form }: PageProps = $props();
	let locale = $derived(data.session?.tenant?.locale ?? 'en');
	let making = $state(false);
	let kind = $state('path');

	const paths = $derived(data.collections.filter((row) => row.kind === 'path'));
	const bundles = $derived(data.collections.filter((row) => row.kind === 'bundle'));
</script>

<svelte:head><title>Paths and bundles · Fajr LMS</title></svelte:head>

<header class="mb-6 flex flex-wrap items-start justify-between gap-3">
	<div>
		<h1 class="text-2xl font-semibold tracking-tight" dir="auto">Paths and bundles</h1>
		<p class="mt-1 mb-0 text-sm text-ink-soft" dir="auto">
			A path is several courses worked through in order. A bundle is several bought together.
		</p>
	</div>
	{#if data.teaches}
		<button class="btn btn-sm" type="button" onclick={() => (making = !making)}>
			{making ? 'Cancel' : 'Make one'}
		</button>
	{/if}
</header>

{#if form?.message}
	<p class="banner-bad mb-5 text-sm" role="alert">{form.message}</p>
{/if}

{#if making}
	<form method="POST" action="?/create" use:enhance class="card mb-6 flex flex-col gap-4">
		<div>
			<span class="mb-1.5 block text-sm font-medium">Which</span>
			<div class="flex flex-wrap gap-4 py-2">
				{#each [['path', 'A path to work through'], ['bundle', 'A bundle to buy']] as [value, label] (value)}
					<label class="flex items-center gap-2 text-sm">
						<input
							class="choice choice-round"
							type="radio"
							name="kind"
							{value}
							checked={kind === value}
							onchange={() => (kind = value)}
						/>
						{label}
					</label>
				{/each}
			</div>
		</div>
		<div>
			<label class="mb-1.5 block text-sm font-medium" for="collection-title">Title</label>
			<input class="field" id="collection-title" name="title" dir="auto" required />
		</div>
		<div>
			<label class="mb-1.5 block text-sm font-medium" for="collection-summary">
				Summary <span class="font-normal text-ink-soft">· one line</span>
			</label>
			<input class="field" id="collection-summary" name="summary" dir="auto" />
		</div>
		{#if kind === 'bundle'}
			<div class="w-40">
				<label class="mb-1.5 block text-sm font-medium" for="collection-price">Price</label>
				<input
					class="field font-mono"
					id="collection-price"
					name="price"
					type="number"
					min="0"
					value="0"
					dir="ltr"
				/>
			</div>
		{/if}
		<div class="flex justify-end">
			<button class="btn" type="submit">Make it</button>
		</div>
	</form>
{/if}

{#snippet list(rows: Collection[])}
	<div class="flex flex-col gap-3">
		{#each rows as row (row.id)}
			<a
				class="card flex flex-wrap items-center gap-3 transition hover:border-line-strong"
				href="/paths/{row.slug}"
			>
				{#if row.kind === 'bundle'}
					<Package class="shrink-0 text-ink-soft" size={18} aria-hidden="true" />
				{:else}
					<Route class="shrink-0 text-ink-soft" size={18} aria-hidden="true" />
				{/if}
				<span class="min-w-0 flex-1">
					<span class="block font-medium" dir="auto">{row.title}</span>
					{#if row.summary}
						<span class="mt-0.5 block text-sm text-ink-soft" dir="auto">{row.summary}</span>
					{/if}
				</span>
				<span class="text-sm text-ink-soft">
					{row.courses}
					{row.courses === 1 ? 'course' : 'courses'}
				</span>
				{#if row.kind === 'bundle' && row.price_minor > 0}
					<span class="font-medium">{money(row.price_minor, row.currency, locale)}</span>
				{/if}
				<span class="chip" class:chip-brand={row.status === 'published'}>
					{row.status === 'published' ? 'Live' : 'Draft'}
				</span>
			</a>
		{/each}
	</div>
{/snippet}

{#if paths.length > 0}
	<section class="mb-6">
		<h2 class="mb-3 text-sm font-semibold tracking-wide uppercase text-ink-soft">Paths</h2>
		{@render list(paths)}
	</section>
{/if}

{#if bundles.length > 0}
	<section class="mb-6">
		<h2 class="mb-3 text-sm font-semibold tracking-wide uppercase text-ink-soft">Bundles</h2>
		{@render list(bundles)}
	</section>
{/if}

{#if data.collections.length === 0}
	<div class="card flex items-start gap-3 text-sm text-ink-soft" dir="auto">
		<Route class="mt-0.5 shrink-0" size={18} aria-hidden="true" />
		<p class="mb-0">
			Nothing here yet. A path is the answer to "where do I start?", asked once and answered for
			everybody.
		</p>
	</div>
{/if}
