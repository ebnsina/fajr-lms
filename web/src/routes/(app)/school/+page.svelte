<script lang="ts">
	import { enhance } from '$app/forms';
	import Trash from '@lucide/svelte/icons/trash-2';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	let locale = $derived(data.session?.tenant?.locale ?? 'en');

	let currentYear = $derived(data.years.find((year) => year.is_current) ?? null);
	let addingClass = $state(false);
	let openClass = $state<string | null>(null);

	const day = (iso: string) =>
		new Intl.DateTimeFormat(locale, { dateStyle: 'medium' }).format(new Date(iso));

	const sectionsOf = (classID: string) =>
		data.sections.filter((row) => row.section.class_id === classID);
	const subjectsOf = (classID: string) =>
		data.subjects.filter((row) => row.subject.class_id === classID);
</script>

<svelte:head><title>The school · Fajr LMS</title></svelte:head>

<header class="mb-6">
	<h1 class="text-2xl font-semibold tracking-tight" dir="auto">The school</h1>
	<p class="mt-1 text-sm text-ink-soft" dir="auto">
		The year you are teaching in, and the classes, sections and subjects everything else is
		arranged by. Name them the way your school does.
	</p>
</header>

{#if form?.message}
	<p class="banner-bad mb-5 text-sm" role="alert">{form.message}</p>
{/if}

<section class="card mb-6">
	<h2 class="mb-1 text-sm font-semibold tracking-wide uppercase text-ink-soft">Academic year</h2>
	<p class="mb-4 text-sm text-ink-soft" dir="auto">
		One year is the one being taught in. Everything dated reads from it.
	</p>

	{#if data.years.length > 0}
		<ul class="mb-4 list-none space-y-2 p-0">
			{#each data.years as year (year.id)}
				<li class="flex flex-wrap items-center gap-3 rounded-xl border border-line bg-raised px-3.5 py-2.5">
					<span class="min-w-0 flex-1">
						<span class="font-medium" dir="auto">{year.name}</span>
						<span class="ms-2 text-sm text-ink-soft">
							{day(year.starts_on)} – {day(year.ends_on)}
						</span>
					</span>
					{#if year.is_current}
						<span class="chip chip-brand">Now</span>
					{:else}
						<form method="POST" action="?/currentYear" use:enhance>
							<input type="hidden" name="id" value={year.id} />
							<button class="btn btn-sm btn-quiet" type="submit">Teach in this year</button>
						</form>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}

	<form method="POST" action="?/addYear" use:enhance class="flex flex-wrap items-end gap-3">
		<div class="min-w-40 flex-1">
			<label class="mb-1.5 block text-sm font-medium" for="year-name">Name</label>
			<input class="field" id="year-name" name="name" placeholder="2026" dir="auto" required />
		</div>
		<div>
			<label class="mb-1.5 block text-sm font-medium" for="year-starts">Starts</label>
			<input class="field font-mono" id="year-starts" name="starts_on" type="date" required />
		</div>
		<div>
			<label class="mb-1.5 block text-sm font-medium" for="year-ends">Ends</label>
			<input class="field font-mono" id="year-ends" name="ends_on" type="date" required />
		</div>
		<button class="btn" type="submit">Add the year</button>
	</form>
</section>

{#if currentYear}
	<section class="card mb-6">
		<h2 class="mb-1 text-sm font-semibold tracking-wide uppercase text-ink-soft">
			Terms in {currentYear.name}
		</h2>
		<p class="mb-4 text-sm text-ink-soft" dir="auto">
			What your school calls the parts of the year: terms, semesters, or nothing at all.
		</p>

		{#if data.terms.length > 0}
			<ul class="mb-4 list-none space-y-2 p-0">
				{#each data.terms as term (term.id)}
					<li class="flex flex-wrap items-center gap-3 rounded-xl border border-line bg-raised px-3.5 py-2.5">
						<span class="min-w-0 flex-1">
							<span class="font-medium" dir="auto">{term.name}</span>
							<span class="ms-2 text-sm text-ink-soft">
								{day(term.starts_on)} – {day(term.ends_on)}
							</span>
						</span>
						{#if term.is_current}
							<span class="chip chip-brand">Now</span>
						{:else}
							<form method="POST" action="?/currentTerm" use:enhance>
								<input type="hidden" name="id" value={term.id} />
								<button class="btn btn-sm btn-quiet" type="submit">This is the term</button>
							</form>
						{/if}
					</li>
				{/each}
			</ul>
		{/if}

		<form method="POST" action="?/addTerm" use:enhance class="flex flex-wrap items-end gap-3">
			<input type="hidden" name="year_id" value={currentYear.id} />
			<div class="min-w-40 flex-1">
				<label class="mb-1.5 block text-sm font-medium" for="term-name">Name</label>
				<input class="field" id="term-name" name="name" placeholder="First term" dir="auto" required />
			</div>
			<div>
				<label class="mb-1.5 block text-sm font-medium" for="term-starts">Starts</label>
				<input class="field font-mono" id="term-starts" name="starts_on" type="date" required />
			</div>
			<div>
				<label class="mb-1.5 block text-sm font-medium" for="term-ends">Ends</label>
				<input class="field font-mono" id="term-ends" name="ends_on" type="date" required />
			</div>
			<button class="btn" type="submit">Add the term</button>
		</form>
	</section>
{/if}

<section class="mb-6">
	<div class="mb-4 flex flex-wrap items-center justify-between gap-3">
		<div>
			<h2 class="text-lg font-semibold tracking-tight" dir="auto">Classes</h2>
			<p class="mb-0 text-sm text-ink-soft" dir="auto">
				Whatever your school calls them: Class Six, Ibtidaiyyah, HSC first year.
			</p>
		</div>
		<button class="btn btn-sm" type="button" onclick={() => (addingClass = !addingClass)}>
			{addingClass ? 'Cancel' : 'Add a class'}
		</button>
	</div>

	{#if addingClass}
		<form method="POST" action="?/addClass" use:enhance class="card mb-4 flex flex-wrap items-end gap-3">
			<div class="min-w-48 flex-1">
				<label class="mb-1.5 block text-sm font-medium" for="class-name">Name</label>
				<input class="field" id="class-name" name="name" dir="auto" required />
			</div>
			<div class="w-28">
				<label class="mb-1.5 block text-sm font-medium" for="class-rank">
					Order <span class="font-normal text-ink-soft">· low first</span>
				</label>
				<input class="field font-mono" id="class-rank" name="rank" type="number" value="0" dir="ltr" />
			</div>
			<button class="btn" type="submit">Add</button>
		</form>
	{/if}

	{#if data.classes.length === 0}
		<div class="card text-sm text-ink-soft">
			<p class="mb-0">No classes yet. Add the first one and its sections follow.</p>
		</div>
	{:else}
		<div class="flex flex-col gap-3">
			{#each data.classes as klass (klass.id)}
				{@const sections = sectionsOf(klass.id)}
				<article class="card">
					<header class="mb-3 flex flex-wrap items-center gap-3">
						<h3 class="mb-0 min-w-0 flex-1 font-medium" dir="auto">{klass.name}</h3>
						<span class="text-sm text-ink-soft">
							{sections.length}
							{sections.length === 1 ? 'section' : 'sections'}
						</span>
						<button
							class="btn btn-sm btn-quiet"
							type="button"
							onclick={() => (openClass = openClass === klass.id ? null : klass.id)}
						>
							{openClass === klass.id ? 'Done' : 'Sections and subjects'}
						</button>
						<form method="POST" action="?/removeClass" use:enhance>
							<input type="hidden" name="id" value={klass.id} />
							<button class="btn btn-sm btn-quiet" type="submit" aria-label="Remove {klass.name}">
								<Trash size={16} aria-hidden="true" />
							</button>
						</form>
					</header>

					{#if sections.length > 0}
						<ul class="list-none space-y-2 p-0">
							{#each sections as row (row.section.id)}
								<li class="flex flex-wrap items-center gap-3 rounded-xl border border-line bg-raised px-3.5 py-2.5 text-sm">
									<span class="min-w-0 flex-1" dir="auto">{row.section.name}</span>
									<span class="text-ink-soft">
										{row.students}
										{row.students === 1 ? 'student' : 'students'}{row.section.capacity
											? ` of ${row.section.capacity}`
											: ''}
									</span>
									{#if row.teacher_name}
										<span class="chip" dir="auto">{row.teacher_name}</span>
									{/if}
									<form method="POST" action="?/removeSection" use:enhance>
										<input type="hidden" name="id" value={row.section.id} />
										<button
											class="btn btn-sm btn-quiet"
											type="submit"
											aria-label="Remove section {row.section.name}"
										>
											<Trash size={14} aria-hidden="true" />
										</button>
									</form>
								</li>
							{/each}
						</ul>
					{/if}

					{#if openClass === klass.id}
						<div class="mt-4 flex flex-col gap-4 border-t border-line pt-4">
							<form method="POST" action="?/addSection" use:enhance class="flex flex-wrap items-end gap-3">
								<input type="hidden" name="class_id" value={klass.id} />
								<div class="min-w-40 flex-1">
									<label class="mb-1.5 block text-sm font-medium" for="section-{klass.id}">
										New section
									</label>
									<input class="field" id="section-{klass.id}" name="name" placeholder="A" dir="auto" required />
								</div>
								<div class="w-28">
									<label class="mb-1.5 block text-sm font-medium" for="capacity-{klass.id}">
										Seats
									</label>
									<input
										class="field font-mono"
										id="capacity-{klass.id}"
										name="capacity"
										type="number"
										min="1"
										placeholder="any"
										dir="ltr"
									/>
								</div>
								<button class="btn btn-sm" type="submit">Add the section</button>
							</form>

							<div>
								<span class="mb-2 block text-sm font-medium">Subjects</span>
								{#if subjectsOf(klass.id).length > 0}
									<ul class="mb-3 flex flex-wrap gap-2 p-0">
										{#each subjectsOf(klass.id) as row (row.subject.id)}
											<li class="flex list-none items-center gap-2 rounded-xl border border-line bg-raised px-3 py-1.5 text-sm">
												<span dir="auto">{row.subject.name}</span>
												{#if row.subject.code}
													<span class="font-mono text-ink-faint">{row.subject.code}</span>
												{/if}
												<form method="POST" action="?/removeSubject" use:enhance>
													<input type="hidden" name="id" value={row.subject.id} />
													<button
														class="text-ink-soft hover:text-ink"
														type="submit"
														aria-label="Remove {row.subject.name}"
													>
														<Trash size={13} aria-hidden="true" />
													</button>
												</form>
											</li>
										{/each}
									</ul>
								{/if}

								<form method="POST" action="?/addSubject" use:enhance class="flex flex-wrap items-end gap-3">
									<input type="hidden" name="class_id" value={klass.id} />
									<div class="min-w-40 flex-1">
										<label class="mb-1.5 block text-sm font-medium" for="subject-{klass.id}">
											New subject
										</label>
										<input class="field" id="subject-{klass.id}" name="name" dir="auto" required />
									</div>
									<div class="w-28">
										<label class="mb-1.5 block text-sm font-medium" for="code-{klass.id}">Code</label>
										<input class="field font-mono" id="code-{klass.id}" name="code" dir="ltr" />
									</div>
									<button class="btn btn-sm" type="submit">Add the subject</button>
								</form>
							</div>
						</div>
					{/if}
				</article>
			{/each}
		</div>
	{/if}
</section>
