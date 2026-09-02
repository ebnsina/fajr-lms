<script lang="ts">
	import Block from '$lib/components/site/Block.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<svelte:head>
	<title>
		{data.page.title === data.page.tenant_name
			? data.page.title
			: `${data.page.title} · ${data.page.tenant_name}`}
	</title>
	{#if data.page.description}<meta name="description" content={data.page.description} />{/if}
</svelte:head>

{#each data.page.blocks ?? [] as block, i (i)}
	<Block {block} courses={data.courses} tenant={data.tenant} />
{/each}

{#if (data.page.blocks ?? []).length === 0}
	<section class="mx-auto max-w-3xl px-6 py-24 text-center">
		<h1 class="text-3xl font-semibold" dir="auto">{data.page.title}</h1>
		<p class="mt-2 text-ink-soft">This page has no sections yet.</p>
	</section>
{/if}
