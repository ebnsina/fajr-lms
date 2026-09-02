<script lang="ts">
	import { enhance } from '$app/forms';
	import { dirOf } from '$lib/api';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import Plus from '@lucide/svelte/icons/plus';
	import PencilLine from '@lucide/svelte/icons/pencil-line';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	let adding = $state(false);

	let scoreOf = $derived((learner: (typeof data.learners)[number], itemId: string) =>
		learner.scores.find((s) => s.item_id === itemId)
	);
</script>

<svelte:head><title>Gradebook · {data.course.title} · Fajr LMS</title></svelte:head>

<nav class="mb-4 text-sm">
	<a
		class="inline-flex items-center gap-1.5 text-brand-text underline-offset-4 hover:underline"
		href="/courses/{data.slug}"
	>
		<ArrowLeft class="rtl:hidden" size={16} aria-hidden="true" />
		<ArrowRight class="hidden rtl:block" size={16} aria-hidden="true" />
		<span dir={dirOf(data.course.dir)}>{data.course.title}</span>
	</a>
</nav>

<header class="mb-5 flex flex-wrap items-center gap-3">
	<div>
		<h1 class="text-2xl font-semibold tracking-tight" dir="auto">Gradebook</h1>
		<p class="mt-1 text-sm text-ink-soft" dir="auto">
			Quiz and assignment scores fill in on their own. Type in a box to override one.
		</p>
	</div>
	<button class="btn btn-sm btn-quiet ms-auto" type="button" onclick={() => (adding = !adding)}>
		<Plus size={15} aria-hidden="true" />
		Add an item
	</button>
</header>

{#if form?.message}
	<p class="banner-bad mb-5 text-sm" dir="auto">{form.message}</p>
{/if}

{#if adding}
	<form
		method="POST"
		action="?/addItem"
		use:enhance={() => async ({ update }) => {
			await update();
			adding = false;
		}}
		class="card mb-5 flex flex-wrap items-end gap-3"
	>
		<input type="hidden" name="course_id" value={data.course.id} />
		<div class="min-w-48 flex-1">
			<label class="mb-1.5 block text-sm font-medium" for="title">Name</label>
			<input class="field" id="title" name="title" dir="auto" placeholder="oral exam" required />
		</div>
		<div class="w-28">
			<label class="mb-1.5 block text-sm font-medium" for="points_possible">Out of</label>
			<input
				class="field font-mono"
				id="points_possible"
				name="points_possible"
				type="number"
				min="1"
				value="20"
				dir="ltr"
				required
			/>
		</div>
		<div class="w-28">
			<label class="mb-1.5 block text-sm font-medium" for="weight">Weight</label>
			<input
				class="field font-mono"
				id="weight"
				name="weight"
				type="number"
				min="0"
				value="100"
				dir="ltr"
			/>
		</div>
		<button class="btn" type="submit">Add</button>
	</form>
{/if}

{#if data.learners.length === 0}
	<div class="card text-sm text-ink-soft" dir="auto">
		<p class="mb-0">Nobody is enrolled yet, so there is nothing to grade.</p>
	</div>
{:else}
	<!-- Wide by nature: the table scrolls inside its own panel rather than
	     pushing the page sideways. -->
	<div class="card overflow-x-auto p-0">
		<table class="w-full border-collapse text-sm">
			<thead>
				<tr class="border-b border-line">
					<th
						class="sticky start-0 bg-surface px-5 py-3 text-start font-medium text-ink-soft"
						scope="col"
					>
						Learner
					</th>
					{#each data.items as item (item.id)}
						<th class="px-4 py-3 text-start font-medium" scope="col">
							<span class="block max-w-40 truncate" dir="auto">{item.title}</span>
							<span class="mt-0.5 block font-mono text-xs font-normal text-ink-faint">
								/{item.points_possible} · ×{item.weight}
							</span>
						</th>
					{/each}
					<th class="px-5 py-3 text-end font-medium text-ink-soft" scope="col">Course</th>
				</tr>
			</thead>
			<tbody>
				{#each data.learners as learner (learner.enrollment_id)}
					<tr class="border-b border-line last:border-0">
						<th
							class="sticky start-0 bg-surface px-5 py-2 text-start font-medium"
							scope="row"
							dir="auto"
						>
							{learner.full_name}
						</th>

						{#each data.items as item (item.id)}
							{@const score = scoreOf(learner, item.id)}
							<td class="px-4 py-2">
								<form method="POST" action="?/setGrade" use:enhance class="flex items-center gap-1.5">
									<input type="hidden" name="item_id" value={item.id} />
									<input type="hidden" name="enrollment_id" value={learner.enrollment_id} />
									<input
										class="field w-20 px-2 text-center font-mono tabular-nums"
										class:border-brand-line={score?.overridden}
										name="points"
										type="number"
										min="0"
										max={item.points_possible}
										value={score?.points ?? ''}
										placeholder="–"
										dir="ltr"
										aria-label="{item.title} for {learner.full_name}"
										title={score?.overridden ? score.note || 'Entered by a teacher' : undefined}
										onchange={(e) => e.currentTarget.form?.requestSubmit()}
									/>
									{#if score?.overridden}
										<PencilLine
											class="shrink-0 text-brand-text"
											size={13}
											aria-label="Entered by a teacher"
										/>
									{/if}
								</form>
							</td>
						{/each}

						<td class="px-5 py-2 text-end">
							<span class="font-mono font-medium tabular-nums">{learner.percent}%</span>
							<span class="mt-0.5 block text-xs text-ink-faint">
								{learner.items_graded}/{learner.items_total}
							</span>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>

	<p class="mt-3 text-sm text-ink-soft" dir="auto">
		Empty a box to remove a teacher's score and let the marked one apply again. An item nobody has
		sat is left out of the course percentage rather than counted as zero.
	</p>
{/if}
