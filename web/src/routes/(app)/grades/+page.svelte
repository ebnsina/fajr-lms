<script lang="ts">
	import ProgressBar from '$lib/components/ProgressBar.svelte';
	import GraduationCap from '@lucide/svelte/icons/graduation-cap';
	import { dirOf } from '$lib/api';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<svelte:head><title>My grades · Fajr LMS</title></svelte:head>

<header class="mb-6">
	<h1 class="text-2xl font-semibold tracking-tight" dir="auto">My grades</h1>
	<p class="mt-1 text-sm text-ink-soft" dir="auto">
		Only work that has been marked counts towards these.
	</p>
</header>

{#if data.courses.length === 0}
	<div class="card flex items-start gap-3 text-sm text-ink-soft" dir="auto">
		<GraduationCap class="mt-0.5 shrink-0" size={18} aria-hidden="true" />
		<p class="mb-0">Nothing graded yet. Grades appear once you are enrolled and marked.</p>
	</div>
{:else}
	<ul class="list-none space-y-3 p-0">
		{#each data.courses as row (row.enrollment.id)}
			<li class="card">
				<div class="mb-3 flex flex-wrap items-baseline gap-2">
					<a class="font-medium hover:underline" href="/courses/{row.slug}" dir={dirOf(row.dir)}>
						{row.title}
					</a>
					{#if row.grades}
						<span class="text-sm text-ink-soft" dir="auto">
							{row.grades.items_graded} of {row.grades.items_total} marked
						</span>
					{/if}
				</div>
				{#if row.grades && row.grades.items_graded > 0}
					<ProgressBar percent={row.grades.percent} label="Grade for {row.title}" />
				{:else}
					<p class="mb-0 text-sm text-ink-faint" dir="auto">Nothing marked yet.</p>
				{/if}
			</li>
		{/each}
	</ul>
{/if}
