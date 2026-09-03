<script lang="ts">
	import { enhance } from '$app/forms';
	import Plus from '@lucide/svelte/icons/plus';
	import ExternalLink from '@lucide/svelte/icons/external-link';
	import Trash from '@lucide/svelte/icons/trash-2';
	import Globe from '@lucide/svelte/icons/globe';
	import Copy from '@lucide/svelte/icons/copy';
	import { templates, regions, kindNames } from '$lib/site-templates';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	let adding = $state(false);

	let region = $state<'bengal' | 'gulf'>('bengal');
	let browsing = $state(false);
	const shown = $derived(templates.filter((row) => row.region === region));

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

	let copied = $state('');
	async function copy(value: string) {
		try {
			await navigator.clipboard.writeText(value);
			copied = value;
		} catch {
			copied = '';
		}
	}
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
		{#if data.pages.length > 0}
			<button class="btn btn-sm btn-quiet" type="button" onclick={() => (browsing = !browsing)}>
				Templates
			</button>
		{/if}
		<button class="btn btn-sm" type="button" onclick={() => (adding = !adding)}>
			<Plus size={16} aria-hidden="true" /> New page
		</button>
	</div>
</div>

{#if form?.message}
	<p class="banner-bad mb-4" role="alert">{form.message}</p>
{:else if form?.made}
	<p class="banner mb-4" role="status">
		{form.made}
		{form.made === 1 ? 'page' : 'pages'} added as drafts. Read them, then publish.
	</p>
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

{#if data.pages.length === 0 || browsing}
	<section class="card mb-6">
		<header class="mb-4 flex flex-wrap items-center justify-between gap-3">
			<div>
				<h2 class="mb-1 text-sm font-semibold tracking-wide uppercase text-ink-soft">
					Start from a template
				</h2>
				<p class="mb-0 text-sm text-ink-soft">
					Written for the kind of institution you are, in the language it teaches in. Everything
					lands as a draft you can rewrite.
				</p>
			</div>
			{#if data.pages.length > 0}
				<button class="btn btn-sm btn-quiet" type="button" onclick={() => (browsing = false)}>
					Close
				</button>
			{/if}
		</header>

		<div class="mb-4 flex flex-wrap gap-2">
			{#each regions as choice (choice.value)}
				<button
					class="btn btn-sm"
					class:btn-quiet={region !== choice.value}
					type="button"
					onclick={() => (region = choice.value)}
					aria-pressed={region === choice.value}
				>
					{choice.label}
				</button>
			{/each}
		</div>

		<div class="grid gap-4 sm:grid-cols-2">
			{#each shown as template (template.id)}
				<form
					class="flex flex-col rounded-3xl border border-line bg-raised p-5"
					method="POST"
					action="?/template"
					use:enhance
				>
					<input type="hidden" name="template" value={template.id} />
					<header class="mb-2 flex items-start justify-between gap-2">
						<h3 class="mb-0 font-medium">{template.name}</h3>
						<span class="chip">{kindNames[template.kind]}</span>
					</header>
					<p class="mb-4 flex-1 text-sm text-ink-soft">{template.summary}</p>
					<div class="mb-4 flex flex-wrap gap-1.5">
						{#each template.pages as page (page.slug)}
							<span class="chip font-mono">/{page.slug}</span>
						{/each}
					</div>
					<button class="btn btn-sm self-end" type="submit">Use this template</button>
				</form>
			{/each}
		</div>
	</section>
{/if}

<section class="card mb-6">
	<h2 class="mb-1 text-sm font-semibold tracking-wide uppercase text-ink-soft">Your own domain</h2>
	<p class="mb-4 text-sm text-ink-soft">
		Serve the site at your school's own address. Point the domain at us, add the record below, and
		the site answers there.
	</p>

	{#if data.domain.domain}
		<div class="mb-4 flex flex-wrap items-center gap-3">
			<Globe class="text-ink-soft" size={18} aria-hidden="true" />
			<span class="font-mono" dir="ltr">{data.domain.domain}</span>
			<span class="chip" class:chip-brand={data.domain.verified}>
				{data.domain.verified ? 'Live' : 'Waiting on DNS'}
			</span>
			<form class="ms-auto flex gap-2" method="POST" action="?/domain" use:enhance>
				{#if !data.domain.verified}
					<button class="btn btn-sm" type="submit" name="intent" value="verify">
						Check the record
					</button>
				{/if}
				<button class="btn btn-sm btn-quiet" type="submit" name="intent" value="clear">
					Remove
				</button>
			</form>
		</div>

		{#if data.domain.record && !data.domain.verified}
			<dl class="grid gap-3 rounded-xl border border-line bg-raised p-4 text-sm sm:grid-cols-[auto_1fr_1fr]">
				<div>
					<dt class="text-ink-soft">Type</dt>
					<dd class="mt-0.5 font-mono" dir="ltr">{data.domain.record.type}</dd>
				</div>
				<div class="min-w-0">
					<dt class="text-ink-soft">Name</dt>
					<dd class="mt-0.5 flex items-center gap-2">
						<span class="truncate font-mono" dir="ltr">{data.domain.record.name}</span>
						<button
							class="btn btn-sm btn-quiet shrink-0"
							type="button"
							onclick={() => copy(data.domain.record?.name ?? '')}
							aria-label="Copy the record name"
						>
							<Copy size={14} aria-hidden="true" />
						</button>
					</dd>
				</div>
				<div class="min-w-0">
					<dt class="text-ink-soft">Value</dt>
					<dd class="mt-0.5 flex items-center gap-2">
						<span class="truncate font-mono" dir="ltr">{data.domain.record.value}</span>
						<button
							class="btn btn-sm btn-quiet shrink-0"
							type="button"
							onclick={() => copy(data.domain.record?.value ?? '')}
							aria-label="Copy the record value"
						>
							<Copy size={14} aria-hidden="true" />
						</button>
					</dd>
				</div>
			</dl>
			{#if copied}
				<p class="mt-2 mb-0 text-sm text-ink-soft" aria-live="polite">Copied.</p>
			{/if}
			<p class="mt-3 mb-0 text-sm text-ink-soft">
				DNS can take a few minutes to spread, sometimes a few hours. Check again after adding it.
			</p>
		{/if}
	{:else}
		<form class="flex flex-wrap items-end gap-3" method="POST" action="?/domain" use:enhance>
			<div class="min-w-56 flex-1">
				<label class="mb-1.5 block text-sm font-medium" for="domain">Domain</label>
				<input
					class="field font-mono"
					id="domain"
					name="domain"
					placeholder="school.edu.bd"
					dir="ltr"
				/>
			</div>
			<button class="btn" type="submit" name="intent" value="set">Add the domain</button>
		</form>
	{/if}
</section>

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
