<script lang="ts">
	import { enhance } from '$app/forms';
	import Plus from '@lucide/svelte/icons/plus';
	import Trash from '@lucide/svelte/icons/trash-2';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const kinds = [
		{ value: 'mcq_single', label: 'One right answer' },
		{ value: 'mcq_multi', label: 'Several right answers' },
		{ value: 'true_false', label: 'True or false' },
		{ value: 'short_answer', label: 'A word or two' },
		{ value: 'essay', label: 'Written, graded by hand' }
	];
	const kindName = (kind: string) => kinds.find((k) => k.value === kind)?.label ?? kind;
	const needsOptions = (kind: string) => kind === 'mcq_single' || kind === 'mcq_multi';

	let kind = $state('mcq_single');
	let choices = $state(['', '', '', '']);
	let adding = $state(false);

	const addChoice = () => (choices = [...choices, '']);
	const dropChoice = (index: number) => (choices = choices.filter((_, i) => i !== index));

	// A textarea keeps whatever it was created with, so the brief is held in
	// state and resynced when the page reloads after a save.
	let brief = $state('');
	$effect(() => {
		brief = data.assignment?.instructions ?? '';
	});

	// A local datetime input needs the value without a timezone suffix.
	const localValue = (iso: string | null) => (iso ? iso.slice(0, 16) : '');
</script>

<svelte:head><title>Setting {data.lesson.title} · Fajr LMS</title></svelte:head>

<div class="mb-6">
	<a class="text-sm text-brand-text" href="/courses/{data.slug}/edit">← Back to the course</a>
	<h1 class="mt-1 text-2xl font-semibold tracking-tight" dir={data.lesson.dir}>
		{data.lesson.title}
	</h1>
	<p class="mb-0 text-sm text-ink-soft">
		{data.lesson.kind === 'quiz' ? 'A quiz' : 'An assignment'} on this lesson.
	</p>
</div>

{#if form?.message}
	<p class="banner banner-bad mb-4" role="alert">{form.message}</p>
{:else if form?.saved}
	<p class="banner mb-4" role="status">Saved.</p>
{/if}

{#if data.lesson.kind === 'quiz'}
	{#if !data.quiz}
		<form class="card grid gap-4 sm:grid-cols-2" method="POST" action="?/createQuiz" use:enhance>
			<div class="sm:col-span-2">
				<label class="mb-1.5 block text-sm font-medium" for="title">Title</label>
				<input class="field" id="title" name="title" dir="auto" required />
			</div>
			<div class="sm:col-span-2">
				<label class="mb-1.5 block text-sm font-medium" for="instructions">
					What the learner should know before starting
				</label>
				<textarea class="field h-24 py-2" id="instructions" name="instructions" dir="auto"
				></textarea>
			</div>
			<div>
				<label class="mb-1.5 block text-sm font-medium" for="minutes">
					Time limit <span class="font-normal text-ink-soft">· minutes, 0 for none</span>
				</label>
				<input
					class="field font-mono"
					id="minutes"
					name="minutes"
					type="number"
					min="0"
					value="0"
					dir="ltr"
				/>
			</div>
			<div>
				<label class="mb-1.5 block text-sm font-medium" for="max_attempts">Attempts allowed</label>
				<input
					class="field font-mono"
					id="max_attempts"
					name="max_attempts"
					type="number"
					min="1"
					value="1"
					dir="ltr"
				/>
			</div>
			<div>
				<label class="mb-1.5 block text-sm font-medium" for="pass_percent">Pass at</label>
				<input
					class="field font-mono"
					id="pass_percent"
					name="pass_percent"
					type="number"
					min="0"
					max="100"
					value="50"
					dir="ltr"
				/>
			</div>
			<label class="flex items-center gap-2 self-end pb-3 text-sm">
				<input class="choice" type="checkbox" name="shuffle" />
				Shuffle the questions
			</label>
			<div class="flex justify-end sm:col-span-2">
				<button class="btn" type="submit">Create the quiz</button>
			</div>
		</form>
	{:else}
		<div class="card mb-4">
			<h2 class="mb-1 text-lg font-medium" dir="auto">{data.quiz.title}</h2>
			<p class="mb-0 text-sm text-ink-soft">
				Pass at {data.quiz.pass_percent}% · {data.quiz.max_attempts}
				{data.quiz.max_attempts === 1 ? 'attempt' : 'attempts'} ·
				{data.quiz.time_limit_s ? `${data.quiz.time_limit_s / 60} minutes` : 'no time limit'}
			</p>
		</div>

		<div class="mb-4 flex flex-col gap-3">
			{#each data.questions as question, index (question.id)}
				<article class="card">
					<header class="mb-2 flex flex-wrap items-start justify-between gap-2">
						<h3 class="mb-0 font-medium" dir="auto">
							{index + 1}. {question.prompt}
						</h3>
						<div class="flex items-center gap-2">
							<span class="chip">{kindName(question.kind)}</span>
							<span class="chip font-mono">{question.points} pts</span>
							<form method="POST" action="?/removeQuestion" use:enhance>
								<input type="hidden" name="question_id" value={question.id} />
								<button
									class="btn btn-sm btn-quiet"
									type="submit"
									aria-label="Remove question {index + 1}"
								>
									<Trash size={16} aria-hidden="true" />
								</button>
							</form>
						</div>
					</header>
					{#if question.options.length > 0}
						<ul class="flex flex-col gap-1 text-sm">
							{#each question.options as option (option.id)}
								<li class:text-ink-soft={!option.is_correct} dir="auto">
									{option.is_correct ? '✓' : '·'}
									{option.label}
								</li>
							{/each}
						</ul>
					{/if}
					{#if question.explanation}
						<p class="mt-2 mb-0 text-sm text-ink-soft" dir="auto">{question.explanation}</p>
					{/if}
				</article>
			{/each}
		</div>

		{#if adding}
			<form class="card grid gap-4" method="POST" action="?/addQuestion" use:enhance>
				<input type="hidden" name="quiz_id" value={data.quiz.id} />
				<div>
					<label class="mb-1.5 block text-sm font-medium" for="prompt">The question</label>
					<textarea class="field h-20 py-2" id="prompt" name="prompt" dir="auto" required
					></textarea>
				</div>
				<div class="grid gap-4 sm:grid-cols-2">
					<div>
						<span class="mb-1.5 block text-sm font-medium">How it is answered</span>
						<div class="flex flex-col gap-2">
							{#each kinds as choice (choice.value)}
								<label class="flex items-center gap-2 text-sm">
									<input
										class="choice choice-round"
										type="radio"
										name="kind"
										value={choice.value}
										bind:group={kind}
									/>
									{choice.label}
								</label>
							{/each}
						</div>
					</div>
					<div>
						<label class="mb-1.5 block text-sm font-medium" for="points">Worth</label>
						<input
							class="field font-mono"
							id="points"
							name="points"
							type="number"
							min="1"
							value="1"
							dir="ltr"
						/>
					</div>
				</div>

				{#if needsOptions(kind)}
					<div>
						<span class="mb-1.5 block text-sm font-medium">
							The choices <span class="font-normal text-ink-soft">· tick the right ones</span>
						</span>
						<div class="flex flex-col gap-2">
							{#each choices as choice, index (index)}
								<div class="flex items-center gap-2">
									<input
										class="choice"
										type="checkbox"
										name="correct"
										value={index}
										aria-label="Choice {index + 1} is correct"
									/>
									<input
										class="field"
										name="label"
										bind:value={choices[index]}
										placeholder="choice {index + 1}"
										aria-label="Choice {index + 1}"
										dir="auto"
									/>
									<button
										class="btn btn-sm btn-quiet"
										type="button"
										onclick={() => dropChoice(index)}
										aria-label="Remove choice {index + 1}"
									>
										<Trash size={16} aria-hidden="true" />
									</button>
								</div>
							{/each}
							<button class="btn btn-sm btn-quiet self-start" type="button" onclick={addChoice}>
								<Plus size={16} aria-hidden="true" /> Another choice
							</button>
						</div>
					</div>
				{/if}

				<div>
					<label class="mb-1.5 block text-sm font-medium" for="explanation">
						Why <span class="font-normal text-ink-soft">· shown after the result</span>
					</label>
					<input class="field" id="explanation" name="explanation" dir="auto" />
				</div>

				<div class="flex justify-end gap-2">
					<button class="btn btn-quiet" type="button" onclick={() => (adding = false)}>
						Cancel
					</button>
					<button class="btn" type="submit">Add the question</button>
				</div>
			</form>
		{:else}
			<button class="btn btn-quiet" type="button" onclick={() => (adding = true)}>
				<Plus size={16} aria-hidden="true" /> Add a question
			</button>
		{/if}
	{/if}
{:else if data.lesson.kind === 'assignment'}
	<form class="card grid gap-4 sm:grid-cols-2" method="POST" action="?/saveAssignment" use:enhance>
		<input type="hidden" name="assignment_id" value={data.assignment?.id ?? ''} />
		<div class="sm:col-span-2">
			<label class="mb-1.5 block text-sm font-medium" for="title">Title</label>
			<input class="field" id="title" name="title" value={data.assignment?.title ?? ''} dir="auto" required />
		</div>
		<div class="sm:col-span-2">
			<label class="mb-1.5 block text-sm font-medium" for="instructions">The brief</label>
			<textarea
				class="field h-32 py-2"
				id="instructions"
				name="instructions"
				dir="auto"
				bind:value={brief}
			></textarea>
		</div>
		<div>
			<label class="mb-1.5 block text-sm font-medium" for="points">Out of</label>
			<input
				class="field font-mono"
				id="points"
				name="points"
				type="number"
				min="1"
				value={data.assignment?.points ?? 100}
				dir="ltr"
			/>
		</div>
		<div>
			<label class="mb-1.5 block text-sm font-medium" for="due_at">Due</label>
			<input
				class="field font-mono"
				id="due_at"
				name="due_at"
				type="datetime-local"
				value={localValue(data.assignment?.due_at ?? null)}
			/>
		</div>
		<div>
			<label class="mb-1.5 block text-sm font-medium" for="late_penalty">
				Late penalty <span class="font-normal text-ink-soft">· percent</span>
			</label>
			<input
				class="field font-mono"
				id="late_penalty"
				name="late_penalty"
				type="number"
				min="0"
				max="100"
				value={data.assignment?.late_penalty ?? 0}
				dir="ltr"
			/>
		</div>
		<div>
			<label class="mb-1.5 block text-sm font-medium" for="max_files">Files allowed</label>
			<input
				class="field font-mono"
				id="max_files"
				name="max_files"
				type="number"
				min="0"
				max="10"
				value={data.assignment?.max_files ?? 3}
				dir="ltr"
			/>
		</div>
		<label class="flex items-center gap-2 self-end pb-3 text-sm sm:col-span-2">
			<input class="choice" type="checkbox" name="allow_late" checked={data.assignment?.allow_late ?? true} />
			Accept work after the due date
		</label>
		<div class="flex justify-end sm:col-span-2">
			<button class="btn" type="submit">
				{data.assignment ? 'Save the assignment' : 'Create the assignment'}
			</button>
		</div>
	</form>
{:else}
	<div class="card text-ink-soft">
		<p class="mb-0">
			This lesson is a {data.lesson.kind}, so there is nothing to set here. Only quiz and assignment
			lessons carry one.
		</p>
	</div>
{/if}
