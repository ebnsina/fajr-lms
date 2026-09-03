<script lang="ts">
	import { enhance } from '$app/forms';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import MessagesSquare from '@lucide/svelte/icons/messages-square';
	import Pin from '@lucide/svelte/icons/pin';
	import Lock from '@lucide/svelte/icons/lock';
	import { dirOf } from '$lib/api';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	let locale = $derived(data.session?.tenant?.locale ?? 'en');
	let asking = $state(false);

	// "2 days ago" reads better than a date on a discussion.
	function since(iso: string): string {
		const seconds = Math.round((Date.now() - new Date(iso).getTime()) / 1000);
		const units: [Intl.RelativeTimeFormatUnit, number][] = [
			['year', 31536000],
			['month', 2592000],
			['day', 86400],
			['hour', 3600],
			['minute', 60]
		];
		const format = new Intl.RelativeTimeFormat(locale, { numeric: 'auto' });
		for (const [unit, size] of units) {
			if (seconds >= size) return format.format(-Math.floor(seconds / size), unit);
		}
		return format.format(0, 'minute');
	}
</script>

<svelte:head><title>Discussion · {data.course.title} · Fajr LMS</title></svelte:head>

<nav class="mb-4 text-sm">
	<a
		class="inline-flex items-center gap-1.5 text-brand-text underline-offset-4 hover:underline"
		href="/courses/{data.course.slug}"
	>
		<ArrowLeft class="rtl:hidden" size={16} aria-hidden="true" />
		<ArrowRight class="hidden rtl:block" size={16} aria-hidden="true" />
		{data.course.title}
	</a>
</nav>

<header class="mb-6 flex flex-wrap items-start justify-between gap-3">
	<div>
		<h1 class="text-2xl font-semibold tracking-tight" dir="auto">Discussion</h1>
		<p class="mt-1 mb-0 text-sm text-ink-soft" dir="auto">
			Ask here rather than in a message, and the next person to wonder finds the answer.
		</p>
	</div>
	<button class="btn btn-sm" type="button" onclick={() => (asking = !asking)}>
		{asking ? 'Cancel' : 'Ask a question'}
	</button>
</header>

{#if form?.message}
	<p class="banner-bad mb-5 text-sm" role="alert">{form.message}</p>
{/if}

{#if asking}
	<form method="POST" action="?/start" use:enhance class="card mb-5 flex flex-col gap-4">
		<input type="hidden" name="course_id" value={data.course.id} />
		<div>
			<label class="mb-1.5 block text-sm font-medium" for="thread-title">Your question</label>
			<input class="field" id="thread-title" name="title" dir="auto" required />
		</div>
		<div>
			<label class="mb-1.5 block text-sm font-medium" for="thread-body">More about it</label>
			<textarea class="field h-auto min-h-28 py-2.5" id="thread-body" name="body" dir="auto" required
			></textarea>
		</div>
		<div class="flex justify-end">
			<button class="btn" type="submit">Ask</button>
		</div>
	</form>
{/if}

{#if data.threads.length === 0}
	<div class="card flex items-start gap-3 text-sm text-ink-soft" dir="auto">
		<MessagesSquare class="mt-0.5 shrink-0" size={18} aria-hidden="true" />
		<p class="mb-0">Nothing asked yet. The first question is the hardest.</p>
	</div>
{:else}
	<ul class="list-none space-y-2 p-0">
		{#each data.threads as row (row.forum_thread.id)}
			{@const thread = row.forum_thread}
			<li>
				<a
					class="card flex flex-wrap items-center gap-3 transition hover:border-line-strong"
					href="/courses/{data.course.slug}/forum/{thread.id}"
				>
					<span class="min-w-0 flex-1">
						<span class="flex items-center gap-2">
							{#if thread.pinned}
								<Pin class="shrink-0 text-ink-soft" size={14} aria-label="Pinned" />
							{/if}
							{#if thread.locked}
								<Lock class="shrink-0 text-ink-soft" size={14} aria-label="Closed" />
							{/if}
							<span class="font-medium" dir={dirOf(thread.dir)}>{thread.title}</span>
						</span>
						<span class="mt-0.5 block text-sm text-ink-soft" dir="auto">
							{row.author_name ?? 'Somebody'} · {since(thread.last_post_at)}
						</span>
					</span>
					<span class="text-sm text-ink-soft">
						{thread.reply_count}
						{thread.reply_count === 1 ? 'reply' : 'replies'}
					</span>
				</a>
			</li>
		{/each}
	</ul>
{/if}
