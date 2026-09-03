<script lang="ts">
	import { enhance } from '$app/forms';
	import { invalidateAll } from '$app/navigation';
	import ScormPlayer from '$lib/components/ScormPlayer.svelte';
	import MediaPlayer from '$lib/components/MediaPlayer.svelte';
	import { dirOf, duration, meta } from '$lib/api';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import Check from '@lucide/svelte/icons/check';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	let locale = $derived(data.session?.tenant?.locale ?? 'en');
	let done = $derived(form?.completed ?? data.state === 'completed');
	let saving = $state(false);
</script>

<svelte:head><title>{data.lesson.title} · Fajr LMS</title></svelte:head>

<nav class="mb-4 text-sm">
	<a
		class="inline-flex items-center gap-1.5 text-brand-text underline-offset-4 hover:underline"
		href="/courses/{data.course.slug}"
	>
		<ArrowLeft class="rtl:hidden" size={16} aria-hidden="true" />
		<ArrowRight class="hidden rtl:block" size={16} aria-hidden="true" />
		<span dir={dirOf(data.course.dir)}>{data.course.title}</span>
	</a>
</nav>

<header class="mb-4">
	<h1 class="text-2xl font-bold tracking-tight" dir={dirOf(data.lesson.dir)}>
		{data.lesson.title}
	</h1>
	<p class="mt-1 text-sm text-ink-soft" dir="auto">
		{meta(data.lesson.kind, data.lesson.duration_s && duration(data.lesson.duration_s, locale))}
		{#if done}<span class="text-brand-text"> · Completed</span>{/if}
	</p>
</header>

{#if data.lesson.kind === 'video' || data.lesson.kind === 'audio'}
	<div class="mb-5">
		<MediaPlayer playback={data.playback} title={data.lesson.title} />
	</div>
{/if}

{#if data.lesson.body}
	<article class="card mb-5 max-w-prose whitespace-pre-wrap" dir={dirOf(data.lesson.dir)}>
		{data.lesson.body}
	</article>
{/if}

{#if data.scorm}
	<div class="mb-5">
		<ScormPlayer
			base="/courses/{data.course.slug}/lessons/{data.lesson.id}/scorm"
			entry={data.scorm.package.entry_href}
			progress={data.scorm.state}
			onSaved={() => invalidateAll()}
		/>
	</div>
{/if}

{#if data.lesson.kind === 'quiz'}
	<a class="btn mb-5" href="/courses/{data.course.slug}/lessons/{data.lesson.id}/quiz">
		Open the quiz
	</a>
{:else if data.lesson.kind === 'assignment'}
	<a class="btn mb-5" href="/courses/{data.course.slug}/lessons/{data.lesson.id}/assignment">
		Open the assignment
	</a>
{/if}

{#if form?.message}
	<p class="banner-bad mb-4 text-sm">{form.message}</p>
{/if}

<div class="card">
	{#if !data.enrolled}
		<p class="mb-0 text-sm text-ink-soft" dir="auto">
			You are not enrolled, so your progress through this lesson is not being kept.
		</p>
	{:else}
		<form
			method="POST"
			action="?/progress"
			use:enhance={() => {
				saving = true;
				return async ({ update }) => {
					await update({ reset: false });
					saving = false;
				};
			}}
		>
			<input type="hidden" name="position_s" value={data.lesson.duration_s || data.resumeAt} />
			<input type="hidden" name="completed" value={done ? 'false' : 'true'} />
			<button class="btn" class:btn-quiet={done} type="submit" disabled={saving}>
				{#if saving}
					Saving…
				{:else if done}
					<RotateCcw size={16} aria-hidden="true" />
					Mark as not finished
				{:else}
					<Check size={16} aria-hidden="true" />
					Mark as finished
				{/if}
			</button>
		</form>
	{/if}
</div>

<nav class="mt-6 flex flex-wrap items-center gap-3">
	{#if data.previous}
		<a class="btn btn-quiet" href="/courses/{data.course.slug}/lessons/{data.previous.id}">
			<ArrowLeft class="rtl:hidden" size={16} aria-hidden="true" />
			<ArrowRight class="hidden rtl:block" size={16} aria-hidden="true" />
			Previous
		</a>
	{/if}
	{#if data.next}
		<a class="btn ms-auto" href="/courses/{data.course.slug}/lessons/{data.next.id}">
			Next
			<ArrowRight class="rtl:hidden" size={16} aria-hidden="true" />
			<ArrowLeft class="hidden rtl:block" size={16} aria-hidden="true" />
		</a>
	{/if}
</nav>
