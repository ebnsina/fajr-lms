<script lang="ts">
	import { enhance } from '$app/forms';
	import { dirOf } from '$lib/api';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import Check from '@lucide/svelte/icons/check';
	import X from '@lucide/svelte/icons/x';
	import Send from '@lucide/svelte/icons/send';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	let pending = $derived(data.pending);
	let done = $derived(data.attempt.state === 'graded');
	let awarded = $derived(
		data.questions.reduce((sum, q) => sum + (q.points_awarded ?? 0), 0)
	);

	function chosen(question: (typeof data.questions)[number], optionId: string) {
		return question.answer_option_ids.includes(optionId);
	}
</script>

<svelte:head><title>Grading · Fajr LMS</title></svelte:head>

<nav class="mb-4 text-sm">
	<a
		class="inline-flex items-center gap-1.5 text-brand-text underline-offset-4 hover:underline"
		href="/grading"
	>
		<ArrowLeft class="rtl:hidden" size={16} aria-hidden="true" />
		<ArrowRight class="hidden rtl:block" size={16} aria-hidden="true" />
		Back to grading
	</a>
</nav>

<header class="mb-5 flex flex-wrap items-center gap-3">
	<div>
		<h1 class="text-2xl font-semibold tracking-tight" dir="auto">
			Attempt {data.attempt.attempt_no}
		</h1>
		<p class="mt-1 text-sm text-ink-soft" dir="auto">
			{#if done}
				Released. This attempt can no longer be changed.
			{:else if pending === 0}
				Everything is marked. Release it to give the learner their result.
			{:else}
				{pending}
				{pending === 1 ? 'answer' : 'answers'} still to mark.
			{/if}
		</p>
	</div>
	<span class="chip ms-auto font-mono tabular-nums" class:chip-brand={pending === 0}>
		{awarded} / {data.attempt.points_possible}
	</span>
</header>

{#if form?.message}
	<p class="banner-bad mb-5 text-sm" dir="auto">{form.message}</p>
{/if}

<ol class="mb-6 list-none space-y-3 p-0">
	{#each data.questions as question, index (question.id)}
		<li class="card">
			<div class="mb-4 flex items-start gap-3">
				<span class="font-mono text-sm text-ink-faint tabular-nums">{index + 1}</span>
				<span class="min-w-0 flex-1 font-medium" dir={dirOf(question.dir)}>{question.prompt}</span>
				<span
					class="chip shrink-0 font-mono tabular-nums"
					class:chip-brand={(question.points_awarded ?? 0) > 0}
				>
					{question.points_awarded ?? '–'} / {question.points}
				</span>
			</div>

			<div class="ps-8">
				{#if question.options.length > 0}
					<!-- The key and the answer side by side, so a marker reads one row. -->
					<ul class="list-none space-y-1.5 p-0">
						{#each question.options as option (option.id)}
							<li
								class="flex items-center gap-2.5 rounded-xl border px-3.5 py-2 text-sm"
								class:border-brand-line={option.is_correct}
								class:bg-brand-soft={option.is_correct}
								class:border-line={!option.is_correct}
							>
								{#if chosen(question, option.id)}
									{#if option.is_correct}
										<Check class="shrink-0 text-brand-text" size={15} aria-hidden="true" />
									{:else}
										<X class="shrink-0 text-danger" size={15} aria-hidden="true" />
									{/if}
								{:else}
									<span class="size-[15px] shrink-0" aria-hidden="true"></span>
								{/if}
								<span class="min-w-0 flex-1" dir="auto">{option.label}</span>
								{#if chosen(question, option.id)}
									<span class="chip shrink-0">chosen</span>
								{:else if option.is_correct}
									<span class="chip shrink-0">correct</span>
								{/if}
							</li>
						{/each}
					</ul>
				{:else if question.text_answer}
					<blockquote
						class="rounded-xl border border-line bg-raised px-4 py-3 whitespace-pre-wrap"
						dir="auto"
					>
						{question.text_answer}
					</blockquote>
				{:else}
					<p class="text-sm text-ink-faint" dir="auto">Left blank.</p>
				{/if}

				{#if question.explanation}
					<p class="mt-3 text-sm text-ink-soft" dir="auto">
						<span class="font-medium">Expected:</span>
						{question.explanation}
					</p>
				{/if}

				{#if question.needs_grading && !done}
					<form method="POST" action="?/mark" use:enhance class="mt-4 space-y-3">
						<input type="hidden" name="question_id" value={question.id} />
						<div class="flex flex-wrap items-end gap-3">
							<div class="w-32">
								<label class="mb-1.5 block text-sm font-medium" for="points-{question.id}">
									Points
								</label>
								<input
									class="field font-mono"
									id="points-{question.id}"
									name="points_awarded"
									type="number"
									min="0"
									max={question.points}
									step="1"
									required
									dir="ltr"
									placeholder="0"
								/>
							</div>
							<div class="min-w-48 flex-1">
								<label class="mb-1.5 block text-sm font-medium" for="feedback-{question.id}">
									Comment
									<span class="font-normal text-ink-soft">· optional</span>
								</label>
								<input
									class="field"
									id="feedback-{question.id}"
									name="feedback"
									dir="auto"
									placeholder="what would make this better"
								/>
							</div>
							<button class="btn" type="submit">Save the grade</button>
						</div>
					</form>
				{:else if question.feedback}
					<p class="mt-3 rounded-xl border border-line bg-raised px-4 py-2.5 text-sm" dir="auto">
						{question.feedback}
					</p>
				{/if}
			</div>
		</li>
	{/each}
</ol>

{#if !done}
	<form method="POST" action="?/release" use:enhance class="flex flex-col items-end">
		<button class="btn" type="submit" disabled={pending > 0}>
			<Send size={16} aria-hidden="true" />
			Release the result
		</button>
		<p class="mt-2 text-end text-sm text-ink-soft" dir="auto">
			{#if pending > 0}
				Mark every written answer first.
			{:else}
				The learner is told, and the mark cannot be changed afterwards.
			{/if}
		</p>
	</form>
{/if}
