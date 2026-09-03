<script lang="ts">
	import { enhance } from '$app/forms';
	import Plus from '@lucide/svelte/icons/plus';
	import Trash from '@lucide/svelte/icons/trash-2';
	import Eye from '@lucide/svelte/icons/eye';
	import Settings from '@lucide/svelte/icons/settings';
	import Pencil from '@lucide/svelte/icons/pencil';
	import ArrowUp from '@lucide/svelte/icons/arrow-up';
	import ArrowDown from '@lucide/svelte/icons/arrow-down';
	import MediaUpload from '$lib/components/MediaUpload.svelte';
	import { toMajor } from '$lib/api';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	let openModule = $state<string | null>(null);
	let addingModule = $state(false);
	let editingCourse = $state(false);
	let renamingModule = $state<string | null>(null);
	let uploadingTo = $state<string | null>(null);

	// An input keeps whatever it was created with, so the settings are held in
	// state and resynced when the page reloads after a save.
	let settings = $state({
		title: '',
		summary: '',
		price: 0,
		visibility: 'private',
		installments: 1,
		gapDays: 30
	});
	$effect(() => {
		settings = {
			title: data.outline.course.title,
			summary: data.outline.course.summary,
			price: toMajor(data.outline.course.price_minor, data.outline.course.currency),
			visibility: data.outline.course.visibility,
			installments: data.outline.course.installments,
			gapDays: data.outline.course.installment_gap_days
		};
	});

	const course = $derived(data.outline.course);
	const published = $derived(course.status === 'published');

	const kinds = [
		{ value: 'text', label: 'Reading' },
		{ value: 'video', label: 'Video' },
		{ value: 'audio', label: 'Audio' },
		{ value: 'pdf', label: 'PDF' },
		{ value: 'link', label: 'Link' },
		{ value: 'live', label: 'Live class' },
		{ value: 'quiz', label: 'Quiz' },
		{ value: 'assignment', label: 'Assignment' }
	];
	const kindName = (kind: string) => kinds.find((k) => k.value === kind)?.label ?? kind;

	// Positions are fractional, so a move writes one row: the new position is the
	// midpoint between the two lessons it lands between.
	function between(lessons: { position: number }[], from: number, to: number): number {
		const target = lessons[to].position;
		const beyond = to < from ? lessons[to - 1]?.position : lessons[to + 1]?.position;
		if (beyond === undefined) return to < from ? target / 2 : target + 1;
		return (target + beyond) / 2;
	}
	const takesLink = (kind: string) => ['video', 'audio', 'pdf', 'link'].includes(kind);

	let kind = $state('text');
</script>

<svelte:head><title>Building {course.title} · Fajr LMS</title></svelte:head>

<div class="mb-6 flex flex-wrap items-start justify-between gap-3">
	<div>
		<a class="text-sm text-brand-text" href="/courses/{data.slug}">← Back to the course</a>
		<h1 class="mt-1 text-2xl font-semibold tracking-tight" dir={course.dir}>{course.title}</h1>
		<p class="mb-0 text-sm text-ink-soft">
			{published ? 'Anyone enrolled can see it.' : 'A draft. Only staff can see it.'}
		</p>
	</div>
	<div class="flex items-center gap-2">
		<button
			class="btn btn-sm btn-quiet"
			type="button"
			onclick={() => (editingCourse = !editingCourse)}
		>
			<Settings size={16} aria-hidden="true" /> Settings
		</button>
		<a class="btn btn-sm btn-quiet" href="/courses/{data.slug}">
			<Eye size={16} aria-hidden="true" /> View
		</a>
		<form method="POST" action="?/setCourseStatus" use:enhance>
			<input type="hidden" name="course_id" value={course.id} />
			<input type="hidden" name="status" value={published ? 'draft' : 'published'} />
			<button class="btn btn-sm" class:btn-quiet={published} type="submit">
				{published ? 'Unpublish' : 'Publish the course'}
			</button>
		</form>
	</div>
</div>

{#if form?.message}
	<p class="banner-bad mb-4" role="alert">{form.message}</p>
{/if}

{#if editingCourse}
	<form class="card mb-6 grid gap-4 sm:grid-cols-2" method="POST" action="?/saveCourse" use:enhance>
		<input type="hidden" name="course_id" value={course.id} />
		<div class="sm:col-span-2">
			<label class="mb-1.5 block text-sm font-medium" for="course-title">Title</label>
			<input
				class="field"
				id="course-title"
				name="title"
				bind:value={settings.title}
				dir="auto"
				required
			/>
		</div>
		<div class="sm:col-span-2">
			<label class="mb-1.5 block text-sm font-medium" for="course-summary">
				Summary <span class="font-normal text-ink-soft">· a line for the catalog</span>
			</label>
			<input class="field" id="course-summary" name="summary" bind:value={settings.summary} dir="auto" />
		</div>
		<div>
			<label class="mb-1.5 block text-sm font-medium" for="course-price">
				Fee <span class="font-normal text-ink-soft">· 0 is free</span>
			</label>
			<input type="hidden" name="currency" value={course.currency} />
			<input
				class="field font-mono"
				id="course-price"
				name="price"
				type="number"
				min="0"
				step="1"
				bind:value={settings.price}
				dir="ltr"
			/>
		</div>
		<div>
			<label class="mb-1.5 block text-sm font-medium" for="course-parts">
				Payments <span class="font-normal text-ink-soft">· 1 is paid in full</span>
			</label>
			<input
				class="field font-mono"
				id="course-parts"
				name="installments"
				type="number"
				min="1"
				max="24"
				bind:value={settings.installments}
				dir="ltr"
			/>
		</div>
		{#if settings.installments > 1}
			<div>
				<label class="mb-1.5 block text-sm font-medium" for="course-gap">
					Days between payments
				</label>
				<input
					class="field font-mono"
					id="course-gap"
					name="installment_gap_days"
					type="number"
					min="1"
					max="365"
					bind:value={settings.gapDays}
					dir="ltr"
				/>
			</div>
		{:else}
			<input type="hidden" name="installment_gap_days" value={settings.gapDays} />
		{/if}
		<div>
			<span class="mb-1.5 block text-sm font-medium">Who may see it</span>
			<div class="flex flex-wrap gap-4 py-2.5">
				<label class="flex items-center gap-2 text-sm">
					<input
						class="choice choice-round"
						type="radio"
						name="visibility"
						value="private"
						bind:group={settings.visibility}
					/>
					Only this school
				</label>
				<label class="flex items-center gap-2 text-sm">
					<input
						class="choice choice-round"
						type="radio"
						name="visibility"
						value="public"
						bind:group={settings.visibility}
					/>
					Anyone
				</label>
			</div>
		</div>
		<div class="flex justify-end sm:col-span-2">
			<button class="btn" type="submit">Save the course</button>
		</div>
	</form>
{/if}

<div class="flex flex-col gap-4">
	{#each data.outline.modules as module, moduleIndex (module.id)}
		<section class="card">
			<header class="mb-4 flex flex-wrap items-center justify-between gap-2">
				{#if renamingModule === module.id}
					<form
						class="flex flex-1 flex-wrap items-center gap-2"
						method="POST"
						action="?/renameModule"
						use:enhance={() => async ({ update }) => {
							renamingModule = null;
							await update();
						}}
					>
						<input type="hidden" name="module_id" value={module.id} />
						<input
							class="field field-sm min-w-48 flex-1"
							name="title"
							value={module.title}
							aria-label="Section name"
							dir="auto"
						/>
						<button class="btn btn-sm" type="submit">Save</button>
						<button
							class="btn btn-sm btn-quiet"
							type="button"
							onclick={() => (renamingModule = null)}
						>
							Cancel
						</button>
					</form>
				{:else}
					<h2 class="mb-0 text-lg font-medium" dir="auto">{module.title}</h2>
					<div class="flex flex-wrap items-center gap-2">
						<button
							class="btn btn-sm btn-quiet"
							type="button"
							onclick={() => (renamingModule = module.id)}
							aria-label="Rename {module.title}"
						>
							<Pencil size={16} aria-hidden="true" />
						</button>
						<form method="POST" action="?/moveModule" use:enhance>
							<input type="hidden" name="module_id" value={module.id} />
							<input
								type="hidden"
								name="position"
								value={moduleIndex > 0
									? between(data.outline.modules, moduleIndex, moduleIndex - 1)
									: 0}
							/>
							<button
								class="btn btn-sm btn-quiet"
								type="submit"
								disabled={moduleIndex === 0}
								aria-label="Move {module.title} up"
							>
								<ArrowUp size={16} aria-hidden="true" />
							</button>
						</form>
						<form method="POST" action="?/moveModule" use:enhance>
							<input type="hidden" name="module_id" value={module.id} />
							<input
								type="hidden"
								name="position"
								value={moduleIndex < data.outline.modules.length - 1
									? between(data.outline.modules, moduleIndex, moduleIndex + 1)
									: 0}
							/>
							<button
								class="btn btn-sm btn-quiet"
								type="submit"
								disabled={moduleIndex === data.outline.modules.length - 1}
								aria-label="Move {module.title} down"
							>
								<ArrowDown size={16} aria-hidden="true" />
							</button>
						</form>
						<form method="POST" action="?/removeModule" use:enhance>
							<input type="hidden" name="module_id" value={module.id} />
							<button
								class="btn btn-sm btn-quiet"
								type="submit"
								aria-label="Delete {module.title} and its lessons"
							>
								<Trash size={16} aria-hidden="true" />
							</button>
						</form>
						<button
							class="btn btn-sm btn-quiet"
							type="button"
							onclick={() => (openModule = openModule === module.id ? null : module.id)}
						>
							<Plus size={16} aria-hidden="true" /> Add a lesson
						</button>
					</div>
				{/if}
			</header>

			{#if module.lessons.length === 0}
				<p class="mb-0 text-sm text-ink-soft">Nothing in this section yet.</p>
			{:else}
				<ol class="flex flex-col gap-2">
					{#each module.lessons as lesson, index (lesson.id)}
						<li class="flex flex-wrap items-center gap-3 rounded-xl bg-sunken px-4 py-3">
							<span class="min-w-40 flex-1 font-medium" dir={lesson.dir}>{lesson.title}</span>
							<span class="chip">{kindName(lesson.kind)}</span>
							{#if takesLink(lesson.kind)}
								<button
									class="btn btn-sm btn-quiet"
									type="button"
									onclick={() => (uploadingTo = uploadingTo === lesson.id ? null : lesson.id)}
								>
									{lesson.media_id ? 'Replace the file' : 'Upload a file'}
								</button>
							{/if}
							{#if lesson.kind === 'quiz' || lesson.kind === 'assignment'}
								<a
									class="btn btn-sm btn-quiet"
									href="/courses/{data.slug}/lessons/{lesson.id}/edit"
								>
									Set the {lesson.kind}
								</a>
							{/if}
							<form method="POST" action="?/moveLesson" use:enhance>
								<input type="hidden" name="lesson_id" value={lesson.id} />
								<input type="hidden" name="module_id" value={module.id} />
								<input
									type="hidden"
									name="position"
									value={index > 0 ? between(module.lessons, index, index - 1) : 0}
								/>
								<button
									class="btn btn-sm btn-quiet"
									type="submit"
									disabled={index === 0}
									aria-label="Move {lesson.title} up"
								>
									<ArrowUp size={16} aria-hidden="true" />
								</button>
							</form>
							<form method="POST" action="?/moveLesson" use:enhance>
								<input type="hidden" name="lesson_id" value={lesson.id} />
								<input type="hidden" name="module_id" value={module.id} />
								<input
									type="hidden"
									name="position"
									value={index < module.lessons.length - 1
										? between(module.lessons, index, index + 1)
										: 0}
								/>
								<button
									class="btn btn-sm btn-quiet"
									type="submit"
									disabled={index === module.lessons.length - 1}
									aria-label="Move {lesson.title} down"
								>
									<ArrowDown size={16} aria-hidden="true" />
								</button>
							</form>
							<form method="POST" action="?/setLessonStatus" use:enhance>
								<input type="hidden" name="lesson_id" value={lesson.id} />
								<input
									type="hidden"
									name="status"
									value={lesson.status === 'published' ? 'draft' : 'published'}
								/>
								<button class="btn btn-sm btn-quiet" type="submit">
									{lesson.status === 'published' ? 'Live' : 'Draft'}
								</button>
							</form>
							<form method="POST" action="?/removeLesson" use:enhance>
								<input type="hidden" name="lesson_id" value={lesson.id} />
								<button
									class="btn btn-sm btn-quiet"
									type="submit"
									aria-label="Delete {lesson.title}"
								>
									<Trash size={16} aria-hidden="true" />
								</button>
							</form>
							{#if uploadingTo === lesson.id}
								<div class="w-full border-t border-line pt-3">
									<MediaUpload
										lessonId={lesson.id}
										kind={lesson.kind}
										title={lesson.title}
										endpoint="/courses/{data.slug}/edit/upload"
									/>
								</div>
							{/if}
						</li>
					{/each}
				</ol>
			{/if}

			{#if openModule === module.id}
				<form class="mt-4 grid gap-4 border-t border-line pt-4" method="POST" action="?/addLesson" use:enhance>
					<input type="hidden" name="module_id" value={module.id} />
					<div class="grid gap-4 sm:grid-cols-2">
						<div>
							<label class="mb-1.5 block text-sm font-medium" for="lesson-title-{module.id}">
								Title
							</label>
							<input
								class="field"
								id="lesson-title-{module.id}"
								name="title"
								dir="auto"
								required
							/>
						</div>
						<div>
							<span class="mb-1.5 block text-sm font-medium">What kind</span>
							<div class="flex flex-wrap gap-3 py-2.5">
								{#each kinds as choice (choice.value)}
									<label class="flex items-center gap-1.5 text-sm">
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
					</div>

					{#if takesLink(kind)}
						<div>
							<label class="mb-1.5 block text-sm font-medium" for="link-{module.id}">
								Link <span class="font-normal text-ink-soft">· YouTube, Vimeo or a file</span>
							</label>
							<input
								class="field font-mono"
								id="link-{module.id}"
								name="link"
								type="url"
								placeholder="https://youtu.be/..."
								dir="ltr"
							/>
						</div>
					{:else}
						<div>
							<label class="mb-1.5 block text-sm font-medium" for="body-{module.id}">
								What the learner reads
							</label>
							<textarea class="field h-32 py-2" id="body-{module.id}" name="body" dir="auto"
							></textarea>
						</div>
					{/if}

					<div class="flex justify-end gap-2">
						<button
							class="btn btn-quiet"
							type="button"
							onclick={() => (openModule = null)}
						>
							Cancel
						</button>
						<button class="btn" type="submit">Add the lesson</button>
					</div>
				</form>
			{/if}
		</section>
	{/each}
</div>

<div class="mt-4">
	{#if addingModule}
		<form class="card flex flex-wrap items-end gap-3" method="POST" action="?/addModule" use:enhance>
			<input type="hidden" name="course_id" value={course.id} />
			<div class="min-w-56 flex-1">
				<label class="mb-1.5 block text-sm font-medium" for="module-title">Section name</label>
				<input class="field" id="module-title" name="title" placeholder="Week one" dir="auto" required />
			</div>
			<button class="btn" type="submit">Add the section</button>
		</form>
	{:else}
		<button class="btn btn-quiet" type="button" onclick={() => (addingModule = true)}>
			<Plus size={16} aria-hidden="true" /> Add a section
		</button>
	{/if}
</div>
