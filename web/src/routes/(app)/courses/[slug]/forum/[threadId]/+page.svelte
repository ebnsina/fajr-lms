<script lang="ts">
	import { enhance } from '$app/forms';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import Pin from '@lucide/svelte/icons/pin';
	import Lock from '@lucide/svelte/icons/lock';
	import Trash from '@lucide/svelte/icons/trash-2';
	import { dirOf } from '$lib/api';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	let locale = $derived(data.session?.tenant?.locale ?? 'en');
	let thread = $derived(data.thread.forum_thread);

	const when = (iso: string) =>
		new Intl.DateTimeFormat(locale, { dateStyle: 'medium', timeStyle: 'short' }).format(
			new Date(iso)
		);
</script>

<svelte:head><title>{thread.title} · Fajr LMS</title></svelte:head>

<nav class="mb-4 text-sm">
	<a
		class="inline-flex items-center gap-1.5 text-brand-text underline-offset-4 hover:underline"
		href="/courses/{data.thread.course_slug}/forum"
	>
		<ArrowLeft class="rtl:hidden" size={16} aria-hidden="true" />
		<ArrowRight class="hidden rtl:block" size={16} aria-hidden="true" />
		Discussion
	</a>
</nav>

<header class="mb-6 flex flex-wrap items-start justify-between gap-3">
	<div class="min-w-0 flex-1">
		<h1 class="flex flex-wrap items-center gap-2 text-2xl font-semibold tracking-tight" dir={dirOf(thread.dir)}>
			{#if thread.pinned}
				<Pin class="shrink-0 text-ink-soft" size={18} aria-label="Pinned" />
			{/if}
			{#if thread.locked}
				<Lock class="shrink-0 text-ink-soft" size={18} aria-label="Closed" />
			{/if}
			{thread.title}
		</h1>
		<p class="mt-1 mb-0 text-sm text-ink-soft" dir="auto">
			Asked by {data.thread.author_name ?? 'somebody'} on {data.thread.course_title}
		</p>
	</div>

	{#if data.moderates}
		<div class="flex gap-2">
			<form method="POST" action="?/flags" use:enhance>
				<input type="hidden" name="pinned" value={String(!thread.pinned)} />
				<button class="btn btn-sm btn-quiet" type="submit">
					{thread.pinned ? 'Unpin' : 'Pin'}
				</button>
			</form>
			<form method="POST" action="?/flags" use:enhance>
				<input type="hidden" name="locked" value={String(!thread.locked)} />
				<button class="btn btn-sm btn-quiet" type="submit">
					{thread.locked ? 'Reopen' : 'Close'}
				</button>
			</form>
		</div>
	{/if}
</header>

{#if form?.message}
	<p class="banner-bad mb-5 text-sm" role="alert">{form.message}</p>
{/if}

<ol class="mb-6 list-none space-y-3 p-0">
	{#each data.posts as post (post.id)}
		<li class="card">
			<header class="mb-2 flex flex-wrap items-center gap-3 text-sm">
				<span class="font-medium" dir="auto">{post.author_name ?? 'Somebody'}</span>
				<span class="text-ink-soft">{when(post.created_at)}</span>
				{#if !post.removed && (data.moderates || post.author_id === data.me)}
					<form class="ms-auto" method="POST" action="?/remove" use:enhance>
						<input type="hidden" name="post_id" value={post.id} />
						<button class="btn btn-sm btn-quiet" type="submit" aria-label="Remove this post">
							<Trash size={14} aria-hidden="true" />
						</button>
					</form>
				{/if}
			</header>
			{#if post.removed}
				<p class="mb-0 text-sm text-ink-faint italic">This post was removed.</p>
			{:else}
				<p class="mb-0 whitespace-pre-wrap" dir={dirOf(post.dir)}>{post.body}</p>
			{/if}
		</li>
	{/each}
</ol>

{#if thread.locked}
	<p class="banner text-sm" dir="auto">This thread is closed. Ask a new question instead.</p>
{:else}
	<form method="POST" action="?/reply" use:enhance class="card flex flex-col gap-3">
		<label class="text-sm font-medium" for="reply">Your reply</label>
		<textarea class="field h-auto min-h-24 py-2.5" id="reply" name="body" dir="auto" required
		></textarea>
		<div class="flex justify-end">
			<button class="btn" type="submit">Reply</button>
		</div>
	</form>
{/if}
