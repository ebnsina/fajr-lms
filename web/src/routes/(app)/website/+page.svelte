<script lang="ts">
	import { enhance } from '$app/forms';
	import Plus from '@lucide/svelte/icons/plus';
	import ExternalLink from '@lucide/svelte/icons/external-link';
	import Trash from '@lucide/svelte/icons/trash-2';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	let adding = $state(false);

	const themes = [
		{ value: 'plain', label: 'Plain' },
		{ value: 'gulf', label: 'Gulf' },
		{ value: 'bengal', label: 'Bengal' }
	];

	const when = (iso: string) =>
		new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(
			new Date(iso)
		);
	const address = (slug: string) => `/site/${data.tenantSlug}${slug ? '/' + slug : ''}`;
</script>

<div class="mb-6 flex flex-wrap items-start justify-between gap-3">
	<div>
		<h1 class="text-2xl font-semibold">Website</h1>
		<p class="mb-0 text-ink-soft">
			Your public pages. A page appears on the menu once it has a menu label, and to the world once
			it is published.
		</p>
	</div>
	<div class="flex items-center gap-2">
		<a class="btn btn-sm btn-quiet" href="/site/{data.tenantSlug}" target="_blank" rel="noreferrer">
			<ExternalLink size={16} aria-hidden="true" /> Visit the site
		</a>
		<button class="btn btn-sm" type="button" onclick={() => (adding = !adding)}>
			<Plus size={16} aria-hidden="true" /> New page
		</button>
	</div>
</div>

{#if form?.message}
	<p class="banner banner-bad mb-4" role="alert">{form.message}</p>
{/if}

{#if adding}
	<form class="card mb-6 flex flex-wrap items-end gap-3" method="POST" action="?/create" use:enhance>
		<div class="min-w-56 flex-1">
			<label class="mb-1.5 block text-sm font-medium" for="title">Title</label>
			<input class="field" id="title" name="title" dir="auto" required />
		</div>
		<div class="w-56">
			<label class="mb-1.5 block text-sm font-medium" for="slug">
				Address <span class="font-normal text-ink-soft">· empty for the front page</span>
			</label>
			<input class="field font-mono" id="slug" name="slug" placeholder="about" dir="ltr" />
		</div>
		<button class="btn" type="submit">Create</button>
	</form>
{/if}

<form class="card mb-6" method="POST" action="?/theme" use:enhance>
	<h2 class="mb-1 text-sm font-semibold tracking-wide uppercase text-ink-soft">Look</h2>
	<p class="mb-4 text-sm text-ink-soft">
		How the public site is dressed. The Gulf setting reads right to left and sets Arabic larger;
		Bengal sets Bengali and runs a little tighter.
	</p>
	<div class="flex flex-wrap items-center gap-4">
		{#each themes as choice (choice.value)}
			<label class="flex items-center gap-2 text-sm">
				<input
					class="choice choice-round"
					type="radio"
					name="theme"
					value={choice.value}
					checked={data.theme === choice.value}
				/>
				{choice.label}
			</label>
		{/each}
		<button class="btn btn-sm ms-auto" type="submit">Save the look</button>
	</div>
</form>

{#if data.pages.length === 0}
	<div class="card text-ink-soft">
		<p class="mb-0">
			No pages yet. Start with a front page: leave the address empty and it becomes the site's home.
		</p>
	</div>
{:else}
	<div class="flex flex-col gap-3">
		{#each data.pages as page (page.id)}
			<article class="card flex flex-wrap items-center gap-4">
				<div class="min-w-56 flex-1">
					<a class="font-medium hover:underline" href="/website/{page.id}" dir="auto">
						{page.title}
					</a>
					<p class="mb-0 font-mono text-sm text-ink-soft">{address(page.slug)}</p>
				</div>
				<span class="chip" class:chip-brand={page.status === 'published'}>
					{page.status === 'published' ? 'Live' : 'Draft'}
				</span>
				{#if page.nav_label}
					<span class="chip" dir="auto">Menu · {page.nav_label}</span>
				{/if}
				<span class="text-sm text-ink-soft">{when(page.updated_at)}</span>
				<form method="POST" action="?/remove" use:enhance>
					<input type="hidden" name="id" value={page.id} />
					<button class="btn btn-sm btn-quiet" type="submit" aria-label="Delete {page.title}">
						<Trash size={16} aria-hidden="true" />
					</button>
				</form>
			</article>
		{/each}
	</div>
{/if}
