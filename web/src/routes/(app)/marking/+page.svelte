<script lang="ts">
	import ClipboardCheck from '@lucide/svelte/icons/clipboard-check';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<svelte:head><title>Marking · Fajr</title></svelte:head>

<header class="mb-6">
	<h1 class="text-2xl font-semibold tracking-tight" dir="auto">Marking</h1>
	<p class="mt-1 text-sm text-ink-soft" dir="auto">
		Written answers waiting on a teacher. Choice questions are already marked.
	</p>
</header>

{#if data.attempts.length === 0}
	<div class="card flex items-start gap-3 text-sm text-ink-soft" dir="auto">
		<ClipboardCheck class="mt-0.5 shrink-0" size={18} aria-hidden="true" />
		<p class="mb-0">Nothing to mark. Everything handed in has been dealt with.</p>
	</div>
{:else}
	<ul class="list-none space-y-2 p-0">
		{#each data.attempts as row (row.quiz_attempt.id)}
			<li class="card flex items-center gap-3 p-4">
				<span class="min-w-0 flex-1">
					<span class="block font-medium" dir="auto">{row.full_name}</span>
					<span class="mt-0.5 block text-sm text-ink-soft" dir="auto">
						{row.quiz_title} · {row.lesson_title}
					</span>
				</span>
				<span class="chip chip-brand shrink-0" dir="auto">
					{row.pending} to mark
				</span>
			</li>
		{/each}
	</ul>
{/if}
