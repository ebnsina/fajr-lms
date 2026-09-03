<script lang="ts">
	import { enhance } from '$app/forms';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import Check from '@lucide/svelte/icons/check';
	import Trash from '@lucide/svelte/icons/trash-2';
	import ProgressBar from '$lib/components/ProgressBar.svelte';
	import { money } from '$lib/api';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	let locale = $derived(data.session?.tenant?.locale ?? 'en');
	let collection = $derived(data.collection);
	let percent = $derived(
		data.courses.length === 0 ? 0 : Math.round((data.courses_done / data.courses.length) * 100)
	);
</script>

<svelte:head><title>{collection.title} · Fajr LMS</title></svelte:head>

<nav class="mb-4 text-sm">
	<a
		class="inline-flex items-center gap-1.5 text-brand-text underline-offset-4 hover:underline"
		href="/paths"
	>
		<ArrowLeft class="rtl:hidden" size={16} aria-hidden="true" />
		<ArrowRight class="hidden rtl:block" size={16} aria-hidden="true" />
		Paths and bundles
	</a>
</nav>

<header class="mb-6 flex flex-wrap items-start justify-between gap-3">
	<div class="min-w-0 flex-1">
		<h1 class="text-2xl font-semibold tracking-tight" dir="auto">{collection.title}</h1>
		{#if collection.summary}
			<p class="mt-1 mb-0 text-ink-soft" dir="auto">{collection.summary}</p>
		{/if}
		{#if collection.kind === 'bundle' && collection.price_minor > 0}
			<p class="mt-1 mb-0 font-medium">
				{money(collection.price_minor, collection.currency, locale)} for all of it
			</p>
		{/if}
	</div>

	{#if data.teaches}
		<form method="POST" action="?/publish" use:enhance>
			<input type="hidden" name="collection_id" value={collection.id} />
			<input
				type="hidden"
				name="status"
				value={collection.status === 'published' ? 'draft' : 'published'}
			/>
			<button class="btn btn-sm btn-quiet" type="submit">
				{collection.status === 'published' ? 'Unpublish' : 'Publish'}
			</button>
		</form>
	{/if}
</header>

{#if form?.message}
	<p class="banner-bad mb-5 text-sm" role="alert">{form.message}</p>
{/if}

{#if collection.kind === 'path' && data.courses.length > 0}
	<div class="card mb-5">
		<div class="mb-1.5 flex flex-wrap items-baseline justify-between gap-2 text-sm">
			<span class="font-medium">Your way through</span>
			<span class="text-ink-soft">{data.courses_done} of {data.courses.length} finished</span>
		</div>
		<ProgressBar {percent} label="Courses finished on this path" />
	</div>
{/if}

{#if data.courses.length === 0}
	<div class="card mb-5 text-sm text-ink-soft">
		<p class="mb-0">Nothing in it yet.</p>
	</div>
{:else}
	<ol class="mb-5 list-none space-y-2 p-0">
		{#each data.courses as row, index (row.course.id)}
			<li class="card flex flex-wrap items-center gap-3">
				<span class="font-mono text-sm text-ink-faint tabular-nums">{index + 1}</span>
				<a class="min-w-0 flex-1 font-medium hover:underline" href="/courses/{row.course.slug}" dir="auto">
					{row.course.title}
				</a>
				{#if row.course.status !== 'published'}
					<span class="chip">Draft</span>
				{/if}
				{#if data.teaches}
					<form method="POST" action="?/remove" use:enhance>
						<input type="hidden" name="collection_id" value={collection.id} />
						<input type="hidden" name="course_id" value={row.course.id} />
						<button class="btn btn-sm btn-quiet" type="submit" aria-label="Take out {row.course.title}">
							<Trash size={16} aria-hidden="true" />
						</button>
					</form>
				{/if}
			</li>
		{/each}
	</ol>
{/if}

{#if data.teaches && data.available.length > 0}
	<form method="POST" action="?/add" use:enhance class="card flex flex-wrap items-end gap-3">
		<input type="hidden" name="collection_id" value={collection.id} />
		<div class="min-w-56 flex-1">
			<label class="mb-1.5 block text-sm font-medium" for="course">Add a course</label>
			<select class="field" id="course" name="course_id" required>
				{#each data.available as course (course.id)}
					<option value={course.id}>{course.title}</option>
				{/each}
			</select>
		</div>
		<button class="btn" type="submit">
			<Check size={16} aria-hidden="true" />
			Add
		</button>
	</form>
{/if}
