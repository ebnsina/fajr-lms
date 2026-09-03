<script lang="ts">
	import { enhance } from '$app/forms';
	import Download from '@lucide/svelte/icons/download';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const states = ['new', 'contacted', 'qualified', 'won', 'lost'];
	const when = new Intl.DateTimeFormat('en', { dateStyle: 'medium', timeStyle: 'short' });

	let open = $state<string | null>(null);
	const exportHref = $derived(
		`/admin/leads/export?${new URLSearchParams({ state: data.state, q: data.query })}`
	);
</script>

<svelte:head><title>Leads · Back office</title></svelte:head>

<header class="mb-6 flex flex-wrap items-center justify-between gap-3">
	<h1 class="text-2xl font-semibold tracking-tight">Leads</h1>
	<a class="btn btn-sm btn-quiet" href={exportHref} download>
		<Download size={16} aria-hidden="true" /> Take them as a file
	</a>
</header>

{#if form?.message}
	<p class="banner-bad mb-4 text-sm" role="alert">{form.message}</p>
{:else if form?.worked}
	<p class="banner mb-4 text-sm" role="status">Saved.</p>
{/if}

<form class="card mb-5 flex flex-wrap items-end gap-3" method="GET">
	<div class="min-w-48 flex-1">
		<label class="mb-1.5 block text-sm font-medium" for="q">Search</label>
		<input class="field" id="q" name="q" value={data.query} placeholder="name, address, school" />
	</div>
	<div class="w-44">
		<label class="mb-1.5 block text-sm font-medium" for="state">State</label>
		<select class="field" id="state" name="state" value={data.state}>
			<option value="">Any</option>
			{#each states as state (state)}
				<option value={state}>{state}</option>
			{/each}
		</select>
	</div>
	<button class="btn btn-sm" type="submit">Filter</button>
</form>

{#if data.leads.length === 0}
	<p class="card mb-0 text-sm text-ink-soft">Nothing here yet.</p>
{:else}
	<ul class="flex list-none flex-col gap-3 p-0">
		{#each data.leads as lead (lead.id)}
			<li class="card">
				<div class="flex flex-wrap items-baseline gap-x-3 gap-y-1">
					<span class="font-medium" dir="auto">{lead.full_name}</span>
					<span class="chip">{lead.state}</span>
					<span class="text-sm text-ink-soft" dir="ltr">{lead.email}</span>
					{#if lead.phone}
						<span class="font-mono text-sm text-ink-soft" dir="ltr">{lead.phone}</span>
					{/if}
					<span class="ms-auto font-mono text-xs text-ink-faint">
						{when.format(new Date(lead.created_at))}
					</span>
				</div>

				<p class="mt-1.5 mb-0 text-sm text-ink-soft" dir="auto">
					{lead.organisation || 'No school given'}{lead.role ? ` · ${lead.role}` : ''}{lead.learners
						? ` · ${lead.learners} learners`
						: ''} · runs a {lead.runs}
				</p>

				{#if lead.note}
					<p class="mt-2 mb-0 border-s-2 border-line ps-3 text-sm" dir="auto">{lead.note}</p>
				{/if}
				{#if lead.worked_note}
					<p class="mt-2 mb-0 text-sm text-ink-soft" dir="auto">Note: {lead.worked_note}</p>
				{/if}

				<button
					class="btn btn-sm btn-quiet mt-3"
					type="button"
					onclick={() => (open = open === lead.id ? null : lead.id)}
					aria-expanded={open === lead.id}
				>
					{open === lead.id ? 'Close' : 'Work this lead'}
				</button>

				{#if open === lead.id}
					<form class="mt-3 flex flex-wrap items-end gap-3" method="POST" action="?/work" use:enhance>
						<input type="hidden" name="id" value={lead.id} />
						<div class="w-44">
							<label class="mb-1.5 block text-sm font-medium" for="state-{lead.id}">State</label>
							<select class="field" id="state-{lead.id}" name="state" value={lead.state}>
								{#each states as state (state)}
									<option value={state}>{state}</option>
								{/each}
							</select>
						</div>
						<div class="min-w-48 flex-1">
							<label class="mb-1.5 block text-sm font-medium" for="note-{lead.id}">Note</label>
							<input
								class="field"
								id="note-{lead.id}"
								name="note"
								value={lead.worked_note}
								dir="auto"
							/>
						</div>
						<button class="btn btn-sm" type="submit">Save</button>
					</form>
				{/if}
			</li>
		{/each}
	</ul>
{/if}
