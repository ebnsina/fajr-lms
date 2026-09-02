<script lang="ts">
	import ProgressBar from '$lib/components/ProgressBar.svelte';
	import { dirOf, duration } from '$lib/api';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	let course = $derived(data.outline.course);
	let locale = $derived(data.session?.tenant?.locale ?? 'en');

	// One lookup, so a long outline does not scan the list per lesson.
	let states = $derived(
		new Map((data.progress?.lessons ?? []).map((row) => [row.lesson_id, row.state]))
	);

	let firstUnfinished = $derived(
		data.outline.modules
			.flatMap((module) => module.lessons)
			.find((lesson) => states.get(lesson.id) !== 'completed')
	);

	const marks: Record<string, string> = {
		completed: 'Done',
		in_progress: 'Started'
	};
</script>

<svelte:head><title>{course.title} · Fajr</title></svelte:head>

<nav class="mb-4 text-sm">
	<a class="text-brand underline-offset-4 hover:underline" href="/">Back to your courses</a>
</nav>

<header class="mb-6">
	<h1 class="text-2xl font-bold tracking-tight" dir={dirOf(course.dir)}>{course.title}</h1>
	{#if course.summary}
		<p class="mt-1 max-w-prose text-ink-soft" dir={dirOf(course.dir)}>{course.summary}</p>
	{/if}
</header>

{#if data.progress}
	<div class="card mb-6">
		<div class="mb-3 flex flex-wrap items-baseline gap-2">
			<span class="font-semibold">Your progress</span>
			<span class="text-sm text-ink-soft">
				{data.progress.lessons_done} of {data.progress.lessons_total} lessons
			</span>
			{#if data.progress.enrollment.status === 'completed'}
				<span class="chip ms-auto">Completed</span>
			{/if}
		</div>
		<ProgressBar percent={data.progress.percent_complete} label="Course progress" />

		{#if firstUnfinished}
			<a class="btn mt-4" href="/courses/{course.slug}/lessons/{firstUnfinished.id}">
				{data.progress.lessons_done === 0 ? 'Start the course' : 'Continue'}
			</a>
		{/if}
	</div>
{:else}
	<p class="banner mb-6 text-sm">
		You are not enrolled in this course yet, so nothing is being recorded.
	</p>
{/if}

<div class="stack space-y-6">
	{#each data.outline.modules as module, index (module.id)}
		<section>
			<h2 class="mb-2 text-lg font-semibold" dir="auto">
				<span class="text-ink-soft tabular-nums">{index + 1}.</span>
				{module.title}
			</h2>

			{#if module.lessons.length === 0}
				<p class="text-sm text-ink-soft">Nothing in this part yet.</p>
			{:else}
				<ul class="list-none space-y-2 p-0">
					{#each module.lessons as lesson (lesson.id)}
						{@const state = states.get(lesson.id)}
						<li>
							<a
								class="card flex items-center gap-3 transition hover:border-brand"
								href="/courses/{course.slug}/lessons/{lesson.id}"
							>
								<span class="min-w-0 flex-1">
									<span class="block font-medium" dir={dirOf(lesson.dir)}>{lesson.title}</span>
									<span class="mt-0.5 block text-sm text-ink-soft">
										{lesson.kind}{#if lesson.duration_s}
											· {duration(lesson.duration_s, locale)}{/if}
									</span>
								</span>
								{#if state}
									<span class="chip shrink-0">{marks[state] ?? state}</span>
								{/if}
							</a>
						</li>
					{/each}
				</ul>
			{/if}
		</section>
	{/each}
</div>
