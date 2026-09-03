<script lang="ts">
	import { enhance } from '$app/forms';
	import Megaphone from '@lucide/svelte/icons/megaphone';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	let audience = $state('school');

	// Sections are named by their class, since "A" alone says nothing.
	const sections = $derived(
		data.sections.map((row) => ({
			id: row.section.id,
			label: `${row.class_name} · ${row.section.name}`,
			students: row.students
		}))
	);
	const classes = $derived([
		...new Map(data.sections.map((row) => [row.section.class_id, row.class_name])).entries()
	]);
</script>

<svelte:head><title>Notices · Fajr LMS</title></svelte:head>

<header class="mb-6">
	<h1 class="text-2xl font-semibold tracking-tight" dir="auto">Notices</h1>
	<p class="mt-1 text-sm text-ink-soft" dir="auto">
		Tell a section, a class or the whole school something. It reaches guardians here and by text
		message where this school has an SMS gateway configured.
	</p>
</header>

{#if form?.message}
	<p class="banner-bad mb-5 text-sm" role="alert">{form.message}</p>
{:else if form?.sentTo}
	<p class="banner mb-5 flex items-center gap-2.5 text-sm" role="status">
		<Megaphone size={16} aria-hidden="true" />
		Sent to {form.sentTo}
		{form.sentTo === 1 ? 'person' : 'people'}.
	</p>
{/if}

<form method="POST" action="?/send" use:enhance class="card flex flex-col gap-4 lg:max-w-2xl">
	<div>
		<span class="mb-1.5 block text-sm font-medium">Who it is for</span>
		<div class="flex flex-wrap gap-4 py-2">
			{#each [['school', 'The whole school'], ['class', 'One class'], ['section', 'One section']] as [value, label] (value)}
				<label class="flex items-center gap-2 text-sm">
					<input
						class="choice choice-round"
						type="radio"
						name="audience"
						{value}
						checked={audience === value}
						onchange={() => (audience = value)}
					/>
					{label}
				</label>
			{/each}
		</div>
	</div>

	{#if audience === 'class'}
		<div>
			<label class="mb-1.5 block text-sm font-medium" for="target-class">Class</label>
			<select class="field" id="target-class" name="target_id" required>
				{#each classes as [id, name] (id)}
					<option value={id}>{name}</option>
				{/each}
			</select>
		</div>
	{:else if audience === 'section'}
		<div>
			<label class="mb-1.5 block text-sm font-medium" for="target-section">Section</label>
			<select class="field" id="target-section" name="target_id" required>
				{#each sections as section (section.id)}
					<option value={section.id}>
						{section.label} · {section.students}
						{section.students === 1 ? 'student' : 'students'}
					</option>
				{/each}
			</select>
		</div>
	{/if}

	<div>
		<span class="mb-1.5 block text-sm font-medium">Sent to</span>
		<div class="flex flex-wrap gap-4 py-2">
			{#each [['guardians', 'Guardians'], ['students', 'Students'], ['both', 'Both']] as [value, label] (value)}
				<label class="flex items-center gap-2 text-sm">
					<input
						class="choice choice-round"
						type="radio"
						name="to"
						{value}
						checked={value === 'guardians'}
					/>
					{label}
				</label>
			{/each}
		</div>
	</div>

	<div>
		<label class="mb-1.5 block text-sm font-medium" for="notice-title">Title</label>
		<input class="field" id="notice-title" name="title" dir="auto" required />
	</div>

	<div>
		<label class="mb-1.5 block text-sm font-medium" for="notice-body">What it says</label>
		<textarea class="field h-auto min-h-28 py-2.5" id="notice-body" name="body" dir="auto" required
		></textarea>
		<p class="mt-1.5 mb-0 text-sm text-ink-soft">
			Keep it short. A text message is read on a small screen, often standing up.
		</p>
	</div>

	<div class="flex justify-end">
		<button class="btn" type="submit">Send the notice</button>
	</div>
</form>
