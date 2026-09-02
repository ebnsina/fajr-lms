<script lang="ts">
	import { enhance } from '$app/forms';
	import { dirOf, money } from '$lib/api';
	import Library from '@lucide/svelte/icons/library';
	import Plus from '@lucide/svelte/icons/plus';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	let adding = $state(false);
	let locale = $derived(data.session?.tenant?.locale ?? 'en');

	const filters = [
		{ label: 'All', value: null },
		{ label: 'Published', value: 'published' },
		{ label: 'Draft', value: 'draft' }
	];
</script>

<svelte:head><title>Courses · Fajr LMS</title></svelte:head>

<header class="mb-6 flex flex-wrap items-start justify-between gap-3">
	<div>
		<h1 class="text-2xl font-semibold tracking-tight" dir="auto">Courses</h1>
		<p class="mt-1 mb-0 text-sm text-ink-soft" dir="auto">
			{data.total}
			{data.total === 1 ? 'course' : 'courses'} in this school.
		</p>
	</div>
	{#if data.teaches}
		<button class="btn btn-sm" type="button" onclick={() => (adding = !adding)}>
			<Plus size={16} aria-hidden="true" /> New course
		</button>
	{/if}
</header>

{#if form?.message}
	<p class="banner banner-bad mb-4" role="alert">{form.message}</p>
{/if}

{#if adding}
	<form class="card mb-6 grid gap-4 sm:grid-cols-2" method="POST" action="?/create" use:enhance>
		<div class="sm:col-span-2">
			<label class="mb-1.5 block text-sm font-medium" for="title">Title</label>
			<input class="field" id="title" name="title" dir="auto" required />
		</div>
		<div class="sm:col-span-2">
			<label class="mb-1.5 block text-sm font-medium" for="summary">
				Summary <span class="font-normal text-ink-soft">· a line for the catalog</span>
			</label>
			<input class="field" id="summary" name="summary" dir="auto" />
		</div>
		<div>
			<label class="mb-1.5 block text-sm font-medium" for="price">
				Fee <span class="font-normal text-ink-soft">· 0 is free</span>
			</label>
			<input
				class="field font-mono"
				id="price"
				name="price"
				type="number"
				min="0"
				step="1"
				value="0"
				dir="ltr"
			/>
		</div>
		<div>
			<span class="mb-1.5 block text-sm font-medium">Who may see it</span>
			<div class="flex flex-wrap gap-4 py-2.5">
				<label class="flex items-center gap-2 text-sm">
					<input class="choice choice-round" type="radio" name="visibility" value="private" checked />
					Only this school
				</label>
				<label class="flex items-center gap-2 text-sm">
					<input class="choice choice-round" type="radio" name="visibility" value="public" />
					Anyone
				</label>
			</div>
		</div>
		<input type="hidden" name="dir" value="auto" />
		<div class="flex justify-end sm:col-span-2">
			<button class="btn" type="submit">Create the course</button>
		</div>
	</form>
{/if}

<div class="mb-5 flex flex-wrap gap-2">
	{#each filters as filter (filter.label)}
		<a
			class="btn btn-sm"
			class:btn-quiet={data.status !== filter.value}
			href={filter.value ? `/courses?status=${filter.value}` : '/courses'}
		>
			{filter.label}
		</a>
	{/each}
</div>

{#if data.courses.length === 0}
	<div class="card flex items-start gap-3 text-sm text-ink-soft" dir="auto">
		<Library class="mt-0.5 shrink-0" size={18} aria-hidden="true" />
		<p class="mb-0">Nothing here yet. Courses appear once a teacher creates one.</p>
	</div>
{:else}
	<ul class="grid list-none gap-3 p-0 sm:grid-cols-2">
		{#each data.courses as course (course.id)}
			<li>
				<a class="card block h-full transition hover:border-line-strong" href="/courses/{course.slug}">
					<span class="flex items-start gap-3">
						<span class="min-w-0 flex-1">
							<span class="block font-medium" dir={dirOf(course.dir)}>{course.title}</span>
							{#if course.summary}
								<span class="mt-1 line-clamp-2 block text-sm text-ink-soft" dir={dirOf(course.dir)}>
									{course.summary}
								</span>
							{/if}
						</span>
						{#if course.status !== 'published'}
							<span class="chip shrink-0">{course.status}</span>
						{/if}
					</span>
					{#if course.price_minor > 0}
						<span class="mt-3 block text-sm font-medium" dir="auto">
							{money(course.price_minor, course.currency, locale)}
						</span>
					{/if}
				</a>
			</li>
		{/each}
	</ul>
{/if}
