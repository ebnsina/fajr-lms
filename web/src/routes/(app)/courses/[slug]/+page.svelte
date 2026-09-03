<script lang="ts">
	import ProgressBar from '$lib/components/ProgressBar.svelte';
	import { dirOf, duration, meta } from '$lib/api';
	import { lessonIcon } from '$lib/icons';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import Check from '@lucide/svelte/icons/check';
	import Table from '@lucide/svelte/icons/table';
	import CalendarCheck from '@lucide/svelte/icons/calendar-check';
	import PencilLine from '@lucide/svelte/icons/pencil-line';
	import Video from '@lucide/svelte/icons/video';
	import Award from '@lucide/svelte/icons/award';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	let course = $derived(data.outline.course);
	let staff = $derived(
		['owner', 'admin', 'instructor', 'assistant'].includes(data.session?.tenant?.role ?? '')
	);
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

<svelte:head><title>{course.title} · Fajr LMS</title></svelte:head>

<nav class="mb-4 text-sm">
	<a
		class="inline-flex items-center gap-1.5 text-brand-text underline-offset-4 hover:underline"
		href="/"
	>
		<ArrowLeft class="rtl:hidden" size={16} aria-hidden="true" />
		<ArrowRight class="hidden rtl:block" size={16} aria-hidden="true" />
		Back to your courses
	</a>
</nav>

<header class="mb-6">
	<h1 class="text-2xl font-bold tracking-tight" dir={dirOf(course.dir)}>{course.title}</h1>
	{#if course.summary}
		<p class="mt-1 max-w-prose text-ink-soft" dir={dirOf(course.dir)}>{course.summary}</p>
	{/if}
	{#if staff}
		<span class="mt-3 flex flex-wrap gap-2">
			<a class="btn btn-sm btn-quiet" href="/courses/{course.slug}/edit">
				<PencilLine size={15} aria-hidden="true" />
				Build
			</a>
			<a class="btn btn-sm btn-quiet" href="/courses/{course.slug}/gradebook">
				<Table size={15} aria-hidden="true" />
				Gradebook
			</a>
			<a class="btn btn-sm btn-quiet" href="/courses/{course.slug}/attendance">
				<CalendarCheck size={15} aria-hidden="true" />
				Attendance
			</a>
			<a class="btn btn-sm btn-quiet" href="/courses/{course.slug}/classes">
				<Video size={15} aria-hidden="true" />
				Live classes
			</a>
			<a class="btn btn-sm btn-quiet" href="/courses/{course.slug}/certificates">
				<Award size={15} aria-hidden="true" />
				Certificates
			</a>
		</span>
	{/if}
</header>

{#if data.progress}
	<div class="card mb-6">
		<div class="mb-3 flex flex-wrap items-baseline gap-2">
			<span class="font-semibold" dir="auto">Your progress</span>
			<span class="text-sm text-ink-soft" dir="auto">
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
	<p class="banner mb-6 text-sm" dir="auto">
		You are not enrolled in this course yet, so nothing is being recorded.
	</p>
{/if}

<div class="stack space-y-6">
	{#each data.outline.modules as module, index (module.id)}
		<section>
			<h2 class="mb-2 text-lg font-semibold">
				<bdi class="text-ink-soft tabular-nums">{index + 1}.</bdi>
				<bdi dir="auto">{module.title}</bdi>
			</h2>

			{#if module.lessons.length === 0}
				<p class="text-sm text-ink-soft" dir="auto">Nothing in this part yet.</p>
			{:else}
				<ul class="list-none space-y-2 p-0">
					{#each module.lessons as lesson (lesson.id)}
						{@const state = states.get(lesson.id)}
						{@const Icon = lessonIcon(lesson.kind)}
						<li>
							<a
								class="card flex items-center gap-3 p-4 transition hover:border-line-strong"
								href="/courses/{course.slug}/lessons/{lesson.id}"
							>
								<span
									class="flex size-9 shrink-0 items-center justify-center rounded-xl border border-line bg-sunken text-ink-soft"
								>
									<Icon size={17} aria-hidden="true" />
								</span>
								<span class="min-w-0 flex-1">
									<span class="block font-medium" dir={dirOf(lesson.dir)}>{lesson.title}</span>
									<span class="mt-0.5 block text-sm text-ink-soft" dir="auto">
										{meta(lesson.kind, lesson.duration_s && duration(lesson.duration_s, locale))}
									</span>
								</span>
								{#if state === 'completed'}
									<span class="chip chip-brand shrink-0">
										<Check size={13} aria-hidden="true" />
										{marks[state]}
									</span>
								{:else if state}
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
