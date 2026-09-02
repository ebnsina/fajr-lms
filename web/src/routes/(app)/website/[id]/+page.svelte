<script lang="ts">
	import { enhance } from '$app/forms';
	import Plus from '@lucide/svelte/icons/plus';
	import Trash from '@lucide/svelte/icons/trash-2';
	import ArrowUp from '@lucide/svelte/icons/arrow-up';
	import ArrowDown from '@lucide/svelte/icons/arrow-down';
	import ExternalLink from '@lucide/svelte/icons/external-link';
	import Block from '$lib/components/site/Block.svelte';
	import type { SiteBlock } from '$lib/types.site';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	// Seeded once from the loaded page: editing is local until the form is saved.
	// svelte-ignore state_referenced_locally
	let blocks = $state<SiteBlock[]>(structuredClone($state.snapshot(data.page.blocks) ?? []));
	// svelte-ignore state_referenced_locally
	let title = $state(data.page.title);
	let preview = $state(false);

	const kinds: { value: SiteBlock['type']; label: string; hint: string }[] = [
		{ value: 'hero', label: 'Headline', hint: 'The opening of a page' },
		{ value: 'richtext', label: 'Words', hint: 'A heading and paragraphs' },
		{ value: 'features', label: 'Cards', hint: 'A grid of short points' },
		{ value: 'faq', label: 'Questions', hint: 'Question and answer pairs' },
		{ value: 'courses', label: 'Courses', hint: 'What you are teaching' },
		{ value: 'cta', label: 'Invitation', hint: 'A closing call to act' }
	];
	const listed = (type: SiteBlock['type']) => type === 'features' || type === 'faq';
	const name = (type: SiteBlock['type']) => kinds.find((k) => k.value === type)?.label ?? type;

	function add(type: SiteBlock['type']) {
		blocks = [...blocks, listed(type) ? { type, items: [{ title: '' }] } : { type }];
	}

	function move(index: number, by: number) {
		const to = index + by;
		if (to < 0 || to >= blocks.length) return;
		const next = [...blocks];
		[next[index], next[to]] = [next[to], next[index]];
		blocks = next;
	}

	const remove = (index: number) => (blocks = blocks.filter((_, i) => i !== index));
	const addItem = (block: SiteBlock) => (block.items = [...(block.items ?? []), { title: '' }]);
	const removeItem = (block: SiteBlock, i: number) =>
		(block.items = (block.items ?? []).filter((_, index) => index !== i));

	const published = $derived(data.page.status === 'published');
	const address = $derived(`/site/${data.tenantSlug}${data.page.slug ? '/' + data.page.slug : ''}`);
</script>

<div class="mb-6 flex flex-wrap items-center justify-between gap-3">
	<div>
		<a class="text-sm text-brand-text" href="/website">← All pages</a>
		<h1 class="mt-1 text-2xl font-semibold" dir="auto">{title}</h1>
		<p class="mb-0 font-mono text-sm text-ink-soft">{address}</p>
	</div>
	<div class="flex items-center gap-2">
		<a class="btn btn-sm btn-quiet" href={address} target="_blank" rel="noreferrer">
			<ExternalLink size={16} aria-hidden="true" /> View
		</a>
		<button class="btn btn-sm btn-quiet" type="button" onclick={() => (preview = !preview)}>
			{preview ? 'Edit' : 'Preview'}
		</button>
		<form method="POST" action="?/status" use:enhance>
			<input type="hidden" name="status" value={published ? 'draft' : 'published'} />
			<button class="btn btn-sm" class:btn-quiet={published} type="submit">
				{published ? 'Unpublish' : 'Publish'}
			</button>
		</form>
	</div>
</div>

{#if form?.message}
	<p class="banner banner-bad mb-4" role="alert">{form.message}</p>
{:else if form?.saved}
	<p class="banner mb-4" role="status">Saved.</p>
{/if}

{#if preview}
	<div class="card overflow-hidden p-0">
		{#each blocks as block, i (i)}
			<Block {block} tenant={data.tenantSlug} />
		{/each}
		{#if blocks.length === 0}
			<p class="p-8 text-center text-ink-soft">Nothing to preview yet.</p>
		{/if}
	</div>
{:else}
	<form method="POST" action="?/save" use:enhance>
		<input type="hidden" name="blocks" value={JSON.stringify(blocks)} />

		<div class="card mb-6 grid gap-4 sm:grid-cols-2">
			<div>
				<label class="mb-1.5 block text-sm font-medium" for="title">Title</label>
				<input class="field" id="title" name="title" bind:value={title} dir="auto" required />
			</div>
			<div>
				<label class="mb-1.5 block text-sm font-medium" for="description">
					Description <span class="font-normal text-ink-soft">· shown in search results</span>
				</label>
				<input
					class="field"
					id="description"
					name="description"
					value={data.page.description}
					dir="auto"
				/>
			</div>
			<div>
				<label class="mb-1.5 block text-sm font-medium" for="nav_label">
					Menu label <span class="font-normal text-ink-soft">· empty keeps it off the menu</span>
				</label>
				<input class="field" id="nav_label" name="nav_label" value={data.page.nav_label} dir="auto" />
			</div>
			<div class="w-32">
				<label class="mb-1.5 block text-sm font-medium" for="nav_order">Menu order</label>
				<input
					class="field font-mono"
					id="nav_order"
					name="nav_order"
					type="number"
					value={data.page.nav_order}
					dir="ltr"
				/>
			</div>
			<input type="hidden" name="dir" value={data.page.dir} />
		</div>

		<div class="flex flex-col gap-4">
			{#each blocks as block, index (index)}
				<section class="card">
					<header class="mb-4 flex items-center justify-between gap-2">
						<h2 class="mb-0 text-sm font-semibold tracking-wide uppercase text-ink-soft">
							{name(block.type)}
						</h2>
						<div class="flex items-center gap-1">
							<button
								class="btn btn-sm btn-quiet"
								type="button"
								onclick={() => move(index, -1)}
								disabled={index === 0}
								aria-label="Move {name(block.type)} up"
							>
								<ArrowUp size={16} aria-hidden="true" />
							</button>
							<button
								class="btn btn-sm btn-quiet"
								type="button"
								onclick={() => move(index, 1)}
								disabled={index === blocks.length - 1}
								aria-label="Move {name(block.type)} down"
							>
								<ArrowDown size={16} aria-hidden="true" />
							</button>
							<button
								class="btn btn-sm btn-quiet"
								type="button"
								onclick={() => remove(index)}
								aria-label="Remove {name(block.type)}"
							>
								<Trash size={16} aria-hidden="true" />
							</button>
						</div>
					</header>

					<div class="grid gap-4">
						<div>
							<label class="mb-1.5 block text-sm font-medium" for="heading-{index}">Heading</label>
							<input class="field" id="heading-{index}" bind:value={block.heading} dir="auto" />
						</div>

						{#if !listed(block.type) && block.type !== 'courses'}
							<div>
								<label class="mb-1.5 block text-sm font-medium" for="text-{index}">
									Text <span class="font-normal text-ink-soft">· a blank line starts a paragraph</span>
								</label>
								<textarea
									class="field h-32 py-2"
									id="text-{index}"
									bind:value={block.text}
									dir="auto"
								></textarea>
							</div>
						{/if}

						{#if block.type === 'courses'}
							<div class="w-40">
								<label class="mb-1.5 block text-sm font-medium" for="limit-{index}">
									How many
								</label>
								<input
									class="field font-mono"
									id="limit-{index}"
									type="number"
									min="1"
									max="24"
									bind:value={block.limit}
									dir="ltr"
								/>
							</div>
						{/if}

						{#if block.type === 'hero' || block.type === 'cta'}
							<div class="grid gap-4 sm:grid-cols-2">
								<div>
									<label class="mb-1.5 block text-sm font-medium" for="cta-label-{index}">
										Button
									</label>
									<input
										class="field"
										id="cta-label-{index}"
										bind:value={block.cta_label}
										dir="auto"
									/>
								</div>
								<div>
									<label class="mb-1.5 block text-sm font-medium" for="cta-href-{index}">
										Button goes to
									</label>
									<input
										class="field font-mono"
										id="cta-href-{index}"
										bind:value={block.cta_href}
										placeholder="/courses"
										dir="ltr"
									/>
								</div>
							</div>
						{/if}

						{#if listed(block.type)}
							<div class="flex flex-col gap-3">
								{#each block.items ?? [] as item, i (i)}
									<div class="flex items-start gap-2">
										<div class="grid flex-1 gap-2 sm:grid-cols-2">
											<input
												class="field"
												bind:value={item.title}
												placeholder={block.type === 'faq' ? 'the question' : 'the point'}
												aria-label="Entry {i + 1} title"
												dir="auto"
											/>
											<input
												class="field"
												bind:value={item.text}
												placeholder="a sentence about it"
												aria-label="Entry {i + 1} text"
												dir="auto"
											/>
										</div>
										<button
											class="btn btn-sm btn-quiet"
											type="button"
											onclick={() => removeItem(block, i)}
											aria-label="Remove entry {i + 1}"
										>
											<Trash size={16} aria-hidden="true" />
										</button>
									</div>
								{/each}
								<button
									class="btn btn-sm btn-quiet self-start"
									type="button"
									onclick={() => addItem(block)}
								>
									<Plus size={16} aria-hidden="true" /> Add an entry
								</button>
							</div>
						{/if}
					</div>
				</section>
			{/each}
		</div>

		<div class="mt-6 flex flex-wrap items-center justify-end gap-3">
			<div class="me-auto flex flex-wrap items-center gap-2">
				<span class="text-sm text-ink-soft">Add a section:</span>
				{#each kinds as kind (kind.value)}
					<button
						class="btn btn-sm btn-quiet"
						type="button"
						onclick={() => add(kind.value)}
						title={kind.hint}
					>
						<Plus size={14} aria-hidden="true" />
						{kind.label}
					</button>
				{/each}
			</div>
			<button class="btn" type="submit">Save the page</button>
		</div>
	</form>
{/if}
