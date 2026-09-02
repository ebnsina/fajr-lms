<script lang="ts">
	import { enhance } from '$app/forms';
	import { dirOf } from '$lib/api';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import CalendarPlus from '@lucide/svelte/icons/calendar-plus';
	import CalendarCheck from '@lucide/svelte/icons/calendar-check';
	import Select from '$lib/components/Select.svelte';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	let adding = $state(false);
	let locale = $derived(data.session?.tenant?.locale ?? 'en');

	type Status = 'present' | 'late' | 'absent' | 'excused';
	const options: { value: Status; label: string }[] = [
		{ value: 'present', label: 'Present' },
		{ value: 'late', label: 'Late' },
		{ value: 'absent', label: 'Absent' },
		{ value: 'excused', label: 'Excused' }
	];

	// Held here so a teacher can mark the whole class before sending anything.
	let marks = $state<Record<string, Status>>({});
	$effect(() => {
		marks = Object.fromEntries(
			data.roll.map((row) => [row.enrollment_id, (row.status ?? 'present') as Status])
		);
	});

	let unmarked = $derived(data.roll.filter((row) => row.status === null).length);

	function when(iso: string): string {
		return new Intl.DateTimeFormat(locale, { dateStyle: 'medium', timeStyle: 'short' }).format(
			new Date(iso)
		);
	}

	function markAll(status: Status) {
		marks = Object.fromEntries(data.roll.map((row) => [row.enrollment_id, status]));
	}
</script>

<svelte:head><title>Attendance · {data.course.title} · Fajr</title></svelte:head>

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
		<h1 class="text-2xl font-semibold tracking-tight" dir="auto">Attendance</h1>
		<p class="mt-1 text-sm text-ink-soft" dir="auto">
			Marking somebody absent tells them and anyone listed as their guardian.
		</p>
	</div>
	<button class="btn btn-sm btn-quiet ms-auto" type="button" onclick={() => (adding = !adding)}>
		<CalendarPlus size={15} aria-hidden="true" />
		New class
	</button>
</header>

{#if form?.message}
	<p class="banner-bad mb-5 text-sm" dir="auto">{form.message}</p>
{/if}

{#if adding}
	<form
		method="POST"
		action="?/createSession"
		use:enhance={() => async ({ update }) => {
			await update();
			adding = false;
		}}
		class="card mb-5 flex flex-wrap items-end gap-3"
	>
		<input type="hidden" name="course_id" value={data.course.id} />
		<div class="min-w-48 flex-1">
			<label class="mb-1.5 block text-sm font-medium" for="title">Class</label>
			<input class="field" id="title" name="title" dir="auto" placeholder="Monday morning" required />
		</div>
		<div class="w-40">
			<label class="mb-1.5 block text-sm font-medium" for="location">Room</label>
			<input class="field" id="location" name="location" dir="auto" placeholder="Room 2" />
		</div>
		<div class="w-56">
			<label class="mb-1.5 block text-sm font-medium" for="starts_at">Starts</label>
			<input class="field font-mono" id="starts_at" name="starts_at" type="datetime-local" required />
		</div>
		<button class="btn" type="submit">Add</button>
	</form>
{/if}

{#if data.sessions.length === 0}
	<div class="card flex items-start gap-3 text-sm text-ink-soft" dir="auto">
		<CalendarCheck class="mt-0.5 shrink-0" size={18} aria-hidden="true" />
		<p class="mb-0">No classes scheduled yet. Add one to start taking the register.</p>
	</div>
{:else}
	<div class="mb-5 max-w-md">
		<span class="mb-1.5 block text-sm font-medium" id="session-label">Class</span>
		<Select
			id="session"
			label="Class"
			value={data.chosen}
			options={data.sessions.map((s) => ({
				value: s.id,
				label: s.title,
				hint: when(s.starts_at)
			}))}
			onchange={(id) => (window.location.search = `?session=${id}`)}
		/>
	</div>

	{#if data.roll.length === 0}
		<div class="card text-sm text-ink-soft" dir="auto">
			<p class="mb-0">Nobody is enrolled in this course yet.</p>
		</div>
	{:else}
		<form method="POST" action="?/takeRoll" use:enhance>
			<input type="hidden" name="session_id" value={data.chosen} />

			<div class="mb-3 flex flex-wrap items-center gap-2">
				<span class="text-sm text-ink-soft" dir="auto">
					{data.roll.length}
					{data.roll.length === 1 ? 'learner' : 'learners'}{unmarked > 0
						? `, ${unmarked} not yet marked`
						: ', all marked'}
				</span>
				<span class="ms-auto flex gap-2">
					{#each options.slice(0, 3) as option (option.value)}
						<button
							class="btn btn-sm btn-quiet"
							type="button"
							onclick={() => markAll(option.value)}
						>
							All {option.label.toLowerCase()}
						</button>
					{/each}
				</span>
			</div>

			<ul class="mb-5 list-none space-y-2 p-0">
				{#each data.roll as row (row.enrollment_id)}
					<li class="card flex flex-wrap items-center gap-3 p-3">
						<span class="min-w-0 flex-1 font-medium" dir="auto">{row.full_name}</span>
						<input type="hidden" name="entry" value="{row.enrollment_id}:{marks[row.enrollment_id]}" />

						<fieldset class="flex flex-wrap gap-1.5">
							<legend class="sr-only">Attendance for {row.full_name}</legend>
							{#each options as option (option.value)}
								{@const active = marks[row.enrollment_id] === option.value}
								<label
									class="cursor-pointer rounded-xl border px-3 py-1.5 text-sm transition-colors"
									class:border-brand={active}
									class:bg-brand-soft={active}
									class:text-brand-text={active}
									class:border-line={!active}
									class:text-ink-soft={!active}
									class:hover:border-line-strong={!active}
								>
									<input
										class="sr-only"
										type="radio"
										name="status-{row.enrollment_id}"
										value={option.value}
										checked={active}
										onchange={() => (marks[row.enrollment_id] = option.value)}
									/>
									{option.label}
								</label>
							{/each}
						</fieldset>
					</li>
				{/each}
			</ul>

			<div class="flex justify-end">
				<button class="btn" type="submit">Save the register</button>
			</div>
		</form>
	{/if}
{/if}
