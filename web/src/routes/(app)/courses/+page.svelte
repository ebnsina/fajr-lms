<script lang="ts">
	import { dirOf, money } from '$lib/api';
	import Library from '@lucide/svelte/icons/library';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	let locale = $derived(data.session?.tenant?.locale ?? 'en');

	const filters = [
		{ label: 'All', value: null },
		{ label: 'Published', value: 'published' },
		{ label: 'Draft', value: 'draft' }
	];
</script>

<svelte:head><title>Courses · Fajr</title></svelte:head>

<header class="mb-6">
	<h1 class="text-2xl font-semibold tracking-tight" dir="auto">Courses</h1>
	<p class="mt-1 text-sm text-ink-soft" dir="auto">
		{data.total}
		{data.total === 1 ? 'course' : 'courses'} in this school.
	</p>
</header>

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
