<script lang="ts">
	import { enhance } from '$app/forms';
	import Plus from '@lucide/svelte/icons/plus';
	import Trash from '@lucide/svelte/icons/trash-2';
	import Eye from '@lucide/svelte/icons/eye';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	let openModule = $state<string | null>(null);
	let addingModule = $state(false);

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
	<p class="banner banner-bad mb-4" role="alert">{form.message}</p>
{/if}

<div class="flex flex-col gap-4">
	{#each data.outline.modules as module (module.id)}
		<section class="card">
			<header class="mb-4 flex flex-wrap items-center justify-between gap-2">
				<h2 class="mb-0 text-lg font-medium" dir="auto">{module.title}</h2>
				<button
					class="btn btn-sm btn-quiet"
					type="button"
					onclick={() => (openModule = openModule === module.id ? null : module.id)}
				>
					<Plus size={16} aria-hidden="true" /> Add a lesson
				</button>
			</header>

			{#if module.lessons.length === 0}
				<p class="mb-0 text-sm text-ink-soft">Nothing in this section yet.</p>
			{:else}
				<ol class="flex flex-col gap-2">
					{#each module.lessons as lesson (lesson.id)}
						<li class="flex flex-wrap items-center gap-3 rounded-xl bg-sunken px-4 py-3">
							<span class="min-w-40 flex-1 font-medium" dir={lesson.dir}>{lesson.title}</span>
							<span class="chip">{kindName(lesson.kind)}</span>
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
