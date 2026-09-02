<script lang="ts">
	import { enhance } from '$app/forms';
	import { dirOf } from '$lib/api';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import Clock from '@lucide/svelte/icons/clock';
	import Paperclip from '@lucide/svelte/icons/paperclip';
	import ExternalLink from '@lucide/svelte/icons/external-link';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	let marked = $derived(data.submission.state === 'returned');
	let locale = $derived(data.session?.tenant?.locale ?? 'en');

	// What the learner will actually receive once the penalty is applied.
	let entered = $state<number | null>(null);
	let afterPenalty = $derived(
		entered !== null && data.submission.is_late && data.assignment.late_penalty > 0
			? Math.floor((entered * (100 - data.assignment.late_penalty)) / 100)
			: entered
	);

	function when(iso: string | null): string {
		if (!iso) return '';
		return new Intl.DateTimeFormat(locale, { dateStyle: 'medium', timeStyle: 'short' }).format(
			new Date(iso)
		);
	}
</script>

<svelte:head><title>{data.assignment.title} · Fajr</title></svelte:head>

<nav class="mb-4 text-sm">
	<a
		class="inline-flex items-center gap-1.5 text-brand-text underline-offset-4 hover:underline"
		href="/submissions"
	>
		<ArrowLeft class="rtl:hidden" size={16} aria-hidden="true" />
		<ArrowRight class="hidden rtl:block" size={16} aria-hidden="true" />
		Back to submissions
	</a>
</nav>

<header class="mb-5">
	<h1 class="text-2xl font-semibold tracking-tight" dir="auto">{data.full_name}</h1>
	<p class="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-sm text-ink-soft" dir="auto">
		<span dir={dirOf(data.assignment.dir)}>{data.assignment.title}</span>
		<span aria-hidden="true">·</span>
		<span>out of {data.assignment.points}</span>
		{#if data.submission.submitted_at}
			<span aria-hidden="true">·</span>
			<span>handed in {when(data.submission.submitted_at)}</span>
		{/if}
	</p>
	{#if data.submission.is_late}
		<p class="mt-2 inline-flex items-center gap-1.5 chip" dir="auto">
			<Clock size={13} aria-hidden="true" />
			Late · {data.assignment.late_penalty}% comes off automatically
		</p>
	{/if}
</header>

{#if data.assignment.instructions}
	<details class="card mb-5">
		<summary class="cursor-pointer text-sm font-medium" dir="auto">What was asked</summary>
		<p class="mt-3 mb-0 max-w-prose whitespace-pre-wrap" dir={dirOf(data.assignment.dir)}>
			{data.assignment.instructions}
		</p>
	</details>
{/if}

<section class="card mb-5">
	<h2 class="mb-3 text-sm font-medium text-ink-soft" dir="auto">The work</h2>
	{#if data.submission.body}
		<p class="mb-0 max-w-prose whitespace-pre-wrap" dir="auto">{data.submission.body}</p>
	{:else}
		<p class="mb-0 text-sm text-ink-faint" dir="auto">Nothing written; see the attachments.</p>
	{/if}

	{#if data.attachments.length > 0}
		<ul class="mt-4 list-none space-y-2 p-0">
			{#each data.attachments as file (file.media_id)}
				<li>
					{#if file.playback?.url}
						<a
							class="flex items-center gap-2.5 rounded-xl border border-line bg-raised px-3.5 py-2.5 text-sm transition hover:border-line-strong"
							href={file.playback.url}
							target="_blank"
							rel="noopener"
						>
							<Paperclip size={15} aria-hidden="true" />
							<span class="min-w-0 flex-1 truncate" dir="auto">
								{file.title || 'Attachment'}
							</span>
							<ExternalLink class="shrink-0 text-ink-faint" size={14} aria-hidden="true" />
							<span class="sr-only">opens in a new tab</span>
						</a>
					{:else}
						<span
							class="flex items-center gap-2.5 rounded-xl border border-line bg-raised px-3.5 py-2.5 text-sm text-ink-faint"
						>
							<Paperclip size={15} aria-hidden="true" />
							<span class="min-w-0 flex-1 truncate" dir="auto">
								{file.title || 'Attachment'} · not ready
							</span>
						</span>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}
</section>

{#if form?.message}
	<p class="banner-bad mb-5 text-sm" dir="auto">{form.message}</p>
{/if}

{#if marked}
	<div class="card">
		<div class="mb-2 flex items-center gap-3">
			<h2 class="text-lg font-semibold" dir="auto">Marked</h2>
			<span class="chip chip-brand ms-auto font-mono tabular-nums">
				{data.submission.points_awarded} / {data.assignment.points}
			</span>
		</div>
		<p class="mb-0 whitespace-pre-wrap text-ink-soft" dir="auto">
			{data.submission.feedback || 'No comment was left.'}
		</p>
	</div>
{:else}
	<form method="POST" action="?/grade" use:enhance class="card space-y-4">
		<h2 class="text-sm font-medium text-ink-soft" dir="auto">Your mark</h2>

		<div class="flex flex-wrap items-end gap-3">
			<div class="w-32">
				<label class="mb-1.5 block text-sm font-medium" for="points_awarded">Points</label>
				<input
					class="field font-mono"
					id="points_awarded"
					name="points_awarded"
					type="number"
					min="0"
					max={data.assignment.points}
					step="1"
					required
					dir="ltr"
					placeholder="0"
					oninput={(e) => (entered = e.currentTarget.value === '' ? null : Number(e.currentTarget.value))}
				/>
			</div>
			{#if afterPenalty !== null && afterPenalty !== entered}
				<p class="pb-2.5 text-sm text-ink-soft" dir="auto">
					The learner receives <span class="font-mono font-medium">{afterPenalty}</span> after the
					late penalty.
				</p>
			{/if}
		</div>

		<div>
			<label class="mb-1.5 block text-sm font-medium" for="feedback">Comment</label>
			<textarea
				class="field h-auto min-h-28 py-2.5"
				id="feedback"
				name="feedback"
				rows="4"
				dir="auto"
				placeholder="what was good, and what would make it better"
			></textarea>
		</div>

		<div>
			<button class="btn" type="submit">Return the mark</button>
			<p class="mt-2 mb-0 text-sm text-ink-soft" dir="auto">
				The learner is told, and this cannot be changed afterwards.
			</p>
		</div>
	</form>
{/if}
