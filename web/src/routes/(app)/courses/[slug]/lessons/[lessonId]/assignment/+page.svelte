<script lang="ts">
	import { enhance } from '$app/forms';
	import { dirOf } from '$lib/api';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import Paperclip from '@lucide/svelte/icons/paperclip';
	import X from '@lucide/svelte/icons/x';
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	let assignment = $derived(data.assignment);
	let submission = $derived(data.submission);
	let locale = $derived(data.session?.tenant?.locale ?? 'en');

	let marked = $derived(submission?.state === 'returned');
	let handedIn = $derived(submission?.state === 'submitted' || marked);
	let overdue = $derived(
		!!assignment.due_at && new Date(assignment.due_at).getTime() < Date.now()
	);
	let closed = $derived(overdue && !assignment.allow_late);

	let attachments = $state<{ id: string; name: string }[]>([]);
	let uploading = $state(false);
	let uploadError = $state<string | null>(null);

	// Existing attachments have no filename stored on this side, so they show as
	// numbered files rather than a made up name.
	$effect(() => {
		attachments = (submission?.media_ids ?? []).map((id, i) => ({ id, name: `Attachment ${i + 1}` }));
	});

	function due(): string {
		if (!assignment.due_at) return 'No deadline';
		return new Intl.DateTimeFormat(locale, { dateStyle: 'medium', timeStyle: 'short' }).format(
			new Date(assignment.due_at)
		);
	}

	async function attach(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		const files = [...(input.files ?? [])];
		input.value = '';
		if (files.length === 0) return;

		uploading = true;
		uploadError = null;
		try {
			for (const file of files) {
				if (attachments.length >= assignment.max_files) {
					uploadError = `This assignment takes at most ${assignment.max_files} files.`;
					break;
				}

				const signed = await post({
					step: 'sign',
					filename: file.name,
					content_type: file.type || 'application/octet-stream',
					byte_size: file.size
				});

				// The bytes go straight to storage; the server only signs and confirms.
				const put = await fetch(signed.upload.url, {
					method: signed.upload.method,
					headers: signed.upload.headers,
					body: file
				});
				if (!put.ok) throw new Error('The upload did not finish. Try again.');

				await post({ step: 'complete', media_id: signed.id });
				attachments = [...attachments, { id: signed.id, name: file.name }];
			}
		} catch (error) {
			uploadError = error instanceof Error ? error.message : 'That file could not be uploaded.';
		} finally {
			uploading = false;
		}
	}

	async function post(body: unknown) {
		const response = await fetch('./assignment/upload', {
			method: 'POST',
			headers: { 'content-type': 'application/json' },
			body: JSON.stringify(body)
		});
		const result = await response.json();
		if (!response.ok) throw new Error(result.error ?? 'That did not work.');
		return result;
	}
</script>

<svelte:head><title>{assignment.title} · Fajr LMS</title></svelte:head>

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

<header class="mb-5">
	<h1 class="text-2xl font-semibold tracking-tight" dir={dirOf(assignment.dir)}>
		{assignment.title}
	</h1>
	<p class="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-sm text-ink-soft" dir="auto">
		<span>Out of {assignment.points}</span>
		<span aria-hidden="true">·</span>
		<span>Due {due()}</span>
		{#if assignment.late_penalty > 0 && assignment.allow_late}
			<span aria-hidden="true">·</span>
			<span>{assignment.late_penalty}% off if late</span>
		{/if}
	</p>
</header>

{#if assignment.instructions}
	<article class="card mb-5 max-w-prose whitespace-pre-wrap" dir={dirOf(assignment.dir)}>
		{assignment.instructions}
	</article>
{/if}

{#if form?.message}
	<p class="banner-bad mb-5 text-sm" dir="auto">{form.message}</p>
{/if}

{#if marked}
	<div class="card mb-5">
		<div class="mb-2 flex items-center gap-3">
			<CircleCheck class="text-brand-text" size={20} aria-hidden="true" />
			<h2 class="text-lg font-semibold" dir="auto">Marked</h2>
			<span class="chip chip-brand ms-auto font-mono tabular-nums">
				{submission?.points_awarded} / {assignment.points}
			</span>
		</div>
		{#if submission?.feedback}
			<p class="mb-0 whitespace-pre-wrap text-ink-soft" dir="auto">{submission.feedback}</p>
		{:else}
			<p class="mb-0 text-sm text-ink-soft" dir="auto">Your teacher left no comment.</p>
		{/if}
	</div>
{:else if handedIn}
	<p class="banner mb-5 text-sm" dir="auto">
		Handed in{submission?.is_late ? ' late' : ''}. You cannot change it now, but you can still
		see what you sent.
	</p>
{:else if closed}
	<p class="banner-bad mb-5 flex items-start gap-2 text-sm" dir="auto">
		<TriangleAlert class="mt-0.5 shrink-0" size={16} aria-hidden="true" />
		<span>
			The deadline has passed and this assignment does not take late work. You can still save a
			draft, so nothing you have written is lost.
		</span>
	</p>
{:else if overdue}
	<p class="banner-bad mb-5 flex items-start gap-2 text-sm" dir="auto">
		<TriangleAlert class="mt-0.5 shrink-0" size={16} aria-hidden="true" />
		<span>
			This is past its deadline. You can still hand it in, with {assignment.late_penalty}% taken
			off the grade.
		</span>
	</p>
{/if}

<form method="POST" action="?/save" use:enhance class="space-y-5">
	<input type="hidden" name="assignment_id" value={assignment.id} />
	{#each attachments as file (file.id)}
		<input type="hidden" name="media_ids" value={file.id} />
	{/each}

	<div>
		<label class="mb-1.5 block text-sm font-medium" for="body">Your answer</label>
		<textarea
			class="field h-auto min-h-40 py-2.5"
			id="body"
			name="body"
			rows="8"
			dir="auto"
			placeholder="write your answer here"
			readonly={handedIn}
			value={submission?.body ?? ''}
		></textarea>
	</div>

	{#if assignment.max_files > 0}
		<div>
			<p class="mb-1.5 text-sm font-medium" dir="auto">
				Attachments
				<span class="font-normal text-ink-soft">
					· up to {assignment.max_files}, photos of written work are fine
				</span>
			</p>

			{#if attachments.length > 0}
				<ul class="mb-2 list-none space-y-2 p-0">
					{#each attachments as file (file.id)}
						<li
							class="flex items-center gap-2.5 rounded-xl border border-line bg-raised px-3.5 py-2.5 text-sm"
						>
							<Paperclip size={15} aria-hidden="true" />
							<span class="min-w-0 flex-1 truncate" dir="auto">{file.name}</span>
							{#if !handedIn}
								<button
									class="btn btn-sm btn-quiet"
									type="button"
									aria-label="Remove {file.name}"
									onclick={() => (attachments = attachments.filter((a) => a.id !== file.id))}
								>
									<X size={14} aria-hidden="true" />
								</button>
							{/if}
						</li>
					{/each}
				</ul>
			{/if}

			{#if !handedIn}
				<label class="btn btn-quiet cursor-pointer">
					<Paperclip size={15} aria-hidden="true" />
					{uploading ? 'Uploading…' : 'Add a file'}
					<input
						class="sr-only"
						type="file"
						multiple
						accept="image/*,application/pdf"
						disabled={uploading}
						onchange={attach}
					/>
				</label>
			{/if}

			{#if uploadError}
				<p class="mt-2 text-sm text-danger" dir="auto">{uploadError}</p>
			{/if}
		</div>
	{/if}

	{#if !handedIn}
		<div class="flex flex-wrap justify-end gap-3">
			<button class="btn btn-quiet" type="submit" name="submit" value="false">Save a draft</button>
			{#if !closed}
				<button class="btn" type="submit" name="submit" value="true">Hand in</button>
			{/if}
		</div>
		<p class="text-end text-sm text-ink-soft" dir="auto">
			A draft is only visible to you. Handing in is final.
		</p>
	{/if}
</form>
