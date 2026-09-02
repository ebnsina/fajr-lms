<script lang="ts">
	import { enhance } from '$app/forms';
	import { invalidateAll } from '$app/navigation';
	import QuizTimer from '$lib/components/QuizTimer.svelte';
	import { dirOf } from '$lib/api';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import Check from '@lucide/svelte/icons/check';
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import CircleAlert from '@lucide/svelte/icons/circle-alert';
	import Hourglass from '@lucide/svelte/icons/hourglass';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	let quiz = $derived(data.quiz);
	let live = $derived(data.live);
	let result = $derived(form && 'result' in form ? (form.result as Result) : null);

	type Result = {
		attempt: { points_awarded: number; points_possible: number; state: string };
		percent: number;
		passed: boolean;
		awaiting_marking: boolean;
		breakdown?: { question_id: string; points_awarded: number; awaiting_marking: boolean; explanation?: string }[];
	};

	let used = $derived(data.attempts.length);
	let left = $derived(Math.max(0, quiz.max_attempts - used));
	let best = $derived(
		data.attempts
			.filter((a) => a.state === 'graded')
			.reduce((top, a) => Math.max(top, a.points_awarded), 0)
	);

	// Which answers are already stored, so a resumed paper comes back filled in.
	let saved = $derived(new Map((live?.answers ?? []).map((a) => [a.question_id, a])));
	let justSaved = $state<string | null>(null);

	function saving(questionId: string) {
		return () =>
			async ({ update }: { update: (o?: { reset?: boolean }) => Promise<void> }) => {
				await update({ reset: false });
				justSaved = questionId;
				setTimeout(() => (justSaved = null), 1500);
			};
	}
</script>

<svelte:head><title>{quiz.title} · Fajr</title></svelte:head>

<nav class="mb-4 text-sm">
	<a
		class="inline-flex items-center gap-1.5 text-brand-text underline-offset-4 hover:underline"
		href="/courses/{data.slug}/lessons/{data.lessonId}"
	>
		<ArrowLeft class="rtl:hidden" size={16} aria-hidden="true" />
		<ArrowRight class="hidden rtl:block" size={16} aria-hidden="true" />
		Back to the lesson
	</a>
</nav>

<header class="mb-6">
	<h1 class="text-2xl font-semibold tracking-tight" dir={dirOf(quiz.dir)}>{quiz.title}</h1>
	{#if quiz.instructions}
		<p class="mt-1 max-w-prose text-ink-soft" dir={dirOf(quiz.dir)}>{quiz.instructions}</p>
	{/if}
</header>

{#if form?.message}
	<p class="banner-bad mb-5 text-sm" dir="auto">{form.message}</p>
{/if}

{#if result}
	<!-- Result: what was scored, and whether anything is still with a teacher. -->
	<div class="card mb-5">
		<div class="mb-3 flex items-center gap-3">
			{#if result.awaiting_marking}
				<Hourglass size={20} aria-hidden="true" />
			{:else if result.passed}
				<CircleCheck class="text-brand-text" size={20} aria-hidden="true" />
			{:else}
				<CircleAlert class="text-danger" size={20} aria-hidden="true" />
			{/if}
			<h2 class="text-lg font-semibold" dir="auto">
				{#if result.awaiting_marking}
					Handed in
				{:else if result.passed}
					Passed
				{:else}
					Not passed this time
				{/if}
			</h2>
			<span class="chip ms-auto font-mono tabular-nums">
				{result.attempt.points_awarded} / {result.attempt.points_possible}
			</span>
		</div>

		<p class="mb-0 text-sm text-ink-soft" dir="auto">
			{#if result.awaiting_marking}
				Your written answers are with a teacher. The rest is already marked, so your final
				score may go up.
			{:else}
				You scored {result.percent}%. The pass mark is {quiz.pass_percent}%.
			{/if}
		</p>
	</div>

	{#if result.breakdown?.length}
		<ol class="mb-5 list-none space-y-2 p-0">
			{#each result.breakdown as row, index (row.question_id)}
				{@const question = data.questions.find((q) => q.id === row.question_id)}
				<li class="card p-4">
					<div class="flex items-start gap-3">
						<span class="font-mono text-sm text-ink-faint tabular-nums">{index + 1}</span>
						<span class="min-w-0 flex-1">
							<span class="block" dir={dirOf(question?.dir ?? 'auto')}>{question?.prompt}</span>
							{#if row.explanation}
								<span class="mt-1 block text-sm text-ink-soft" dir="auto">{row.explanation}</span>
							{/if}
						</span>
						<span class="chip shrink-0" class:chip-brand={row.points_awarded > 0}>
							{#if row.awaiting_marking}
								with a teacher
							{:else}
								{row.points_awarded} / {question?.points ?? 0}
							{/if}
						</span>
					</div>
				</li>
			{/each}
		</ol>
	{/if}

	<a class="btn" href="/courses/{data.slug}/lessons/{data.lessonId}">Back to the lesson</a>
{:else if live}
	<!-- Sitting the paper. Each answer saves on its own, so nothing is lost. -->
	<!-- Sticky: the time left is the one thing worth seeing on every question,
	     and it is useless once it has scrolled away. The negative margins let the
	     bar span the panel so nothing shows through behind it. -->
	<div
		class="sticky top-0 z-10 -mx-4 -mt-4 mb-5 flex flex-wrap items-center gap-3 border-b border-line bg-surface px-4 pt-4 pb-3 sm:-mx-6 sm:-mt-6 sm:px-6 sm:pt-6 lg:-mx-8 lg:-mt-8 lg:px-8 lg:pt-8"
	>
		<span class="chip" dir="auto">Attempt {live.attempt.attempt_no} of {quiz.max_attempts}</span>
		<span class="ms-auto text-sm text-ink-soft" dir="auto">Answers save as you go.</span>
		{#if quiz.time_limit_s > 0}
			<QuizTimer seconds={live.expires_in_s} />
		{/if}
	</div>

	<ol class="mb-6 list-none space-y-3 p-0">
		{#each data.questions as question, index (question.id)}
			{@const answer = saved.get(question.id)}
			<li class="card">
				<div class="mb-4 flex items-start gap-3">
					<span class="font-mono text-sm text-ink-faint tabular-nums">{index + 1}</span>
					<span class="min-w-0 flex-1 font-medium" dir={dirOf(question.dir)}>{question.prompt}</span>
					<span class="chip shrink-0" dir="auto">
						{question.points}
						{question.points === 1 ? 'point' : 'points'}
					</span>
				</div>

				<form
					method="POST"
					action="?/answer"
					use:enhance={saving(question.id)}
					class="ps-8"
				>
					<input type="hidden" name="attempt_id" value={live.attempt.id} />
					<input type="hidden" name="question_id" value={question.id} />

					{#if question.kind === 'short_answer' || question.kind === 'essay'}
						<textarea
							class="field h-auto min-h-28 py-2.5"
							name="text"
							rows={question.kind === 'essay' ? 6 : 3}
							dir="auto"
							placeholder="type your answer"
							value={answer?.text ?? ''}
							onchange={(e) => e.currentTarget.form?.requestSubmit()}
						></textarea>
					{:else}
						{@const multi = question.kind === 'mcq_multi'}
						<fieldset class="space-y-2">
							<legend class="sr-only">
								{multi ? 'Choose every correct answer' : 'Choose one answer'}
							</legend>
							{#each question.options as option (option.id)}
								<label
									class="flex cursor-pointer items-center gap-3 rounded-xl border border-line bg-raised px-3.5 py-2.5 transition hover:border-line-strong"
								>
									<input
										class="choice {multi ? '' : 'choice-round'}"
										type={multi ? 'checkbox' : 'radio'}
										name="option_ids"
										value={option.id}
										checked={answer?.option_ids.includes(option.id)}
										onchange={(e) => e.currentTarget.form?.requestSubmit()}
									/>
									<span class="min-w-0 flex-1" dir="auto">{option.label}</span>
								</label>
							{/each}
						</fieldset>
					{/if}

					{#if justSaved === question.id}
						<p class="mt-2 flex items-center gap-1.5 text-sm text-brand-text" dir="auto">
							<Check size={14} aria-hidden="true" />
							Saved
						</p>
					{/if}
				</form>
			</li>
		{/each}
	</ol>

	<form
		method="POST"
		action="?/submit"
		use:enhance={() => async ({ update }) => {
			await update({ reset: false });
			await invalidateAll();
		}}
	>
		<input type="hidden" name="attempt_id" value={live.attempt.id} />
		<div class="flex justify-end">
			<button class="btn" type="submit">Hand in this attempt</button>
		</div>
		<p class="mt-2 text-end text-sm text-ink-soft" dir="auto">
			Anything left blank scores nothing. You cannot change it after handing in.
		</p>
	</form>
{:else}
	<!-- Before starting: what the rules are, and what has happened before. -->
	<div class="card mb-5">
		<dl class="grid gap-4 sm:grid-cols-3">
			<div>
				<dt class="text-sm text-ink-soft" dir="auto">Attempts left</dt>
				<dd class="mt-0.5 font-medium tabular-nums" dir="auto">{left} of {quiz.max_attempts}</dd>
			</div>
			<div>
				<dt class="text-sm text-ink-soft" dir="auto">Time limit</dt>
				<dd class="mt-0.5 font-medium" dir="auto">
					{quiz.time_limit_s > 0 ? `${Math.round(quiz.time_limit_s / 60)} min` : 'None'}
				</dd>
			</div>
			<div>
				<dt class="text-sm text-ink-soft" dir="auto">Pass mark</dt>
				<dd class="mt-0.5 font-medium tabular-nums" dir="auto">{quiz.pass_percent}%</dd>
			</div>
		</dl>
	</div>

	{#if data.attempts.length > 0}
		<div class="card mb-5">
			<h2 class="mb-3 text-sm font-medium text-ink-soft" dir="auto">Earlier attempts</h2>
			<ul class="list-none space-y-2 p-0">
				{#each data.attempts as attempt (attempt.id)}
					<li class="flex items-center gap-3 text-sm">
						<span dir="auto">Attempt {attempt.attempt_no}</span>
						<span class="ms-auto font-mono tabular-nums text-ink-soft">
							{#if attempt.state === 'graded'}
								{attempt.points_awarded} / {attempt.points_possible}
							{:else}
								{attempt.state.replace('_', ' ')}
							{/if}
						</span>
					</li>
				{/each}
			</ul>
			{#if best > 0}
				<p class="mt-3 mb-0 text-sm text-ink-soft" dir="auto">
					Your best attempt counts, not your last.
				</p>
			{/if}
		</div>
	{/if}

	{#if left > 0}
		<form method="POST" action="?/start" use:enhance>
			<input type="hidden" name="quiz_id" value={quiz.id} />
			<button class="btn" type="submit">
				{used === 0 ? 'Start the quiz' : 'Start another attempt'}
			</button>
		</form>
	{:else}
		<p class="banner text-sm" dir="auto">
			You have used all {quiz.max_attempts}
			{quiz.max_attempts === 1 ? 'attempt' : 'attempts'} at this quiz.
		</p>
	{/if}
{/if}
