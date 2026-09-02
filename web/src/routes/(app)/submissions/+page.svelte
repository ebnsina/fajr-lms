<script lang="ts">
	import Inbox from '@lucide/svelte/icons/inbox';
	import Clock from '@lucide/svelte/icons/clock';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<svelte:head><title>Submissions · Fajr</title></svelte:head>

<header class="mb-6">
	<h1 class="text-2xl font-semibold tracking-tight" dir="auto">Submissions</h1>
	<p class="mt-1 text-sm text-ink-soft" dir="auto">Work handed in and waiting for a mark.</p>
</header>

{#if data.submissions.length === 0}
	<div class="card flex items-start gap-3 text-sm text-ink-soft" dir="auto">
		<Inbox class="mt-0.5 shrink-0" size={18} aria-hidden="true" />
		<p class="mb-0">Nothing waiting. Everything handed in has been marked.</p>
	</div>
{:else}
	<ul class="list-none space-y-2 p-0">
		{#each data.submissions as row (row.submission.id)}
			<li class="card flex items-center gap-3 p-4">
				<span class="min-w-0 flex-1">
					<span class="block font-medium" dir="auto">{row.full_name}</span>
					<span class="mt-0.5 block text-sm text-ink-soft" dir="auto">
						{row.assignment_title} · out of {row.points}
					</span>
				</span>
				{#if row.submission.is_late}
					<span class="chip shrink-0">
						<Clock size={13} aria-hidden="true" />
						Late
					</span>
				{/if}
			</li>
		{/each}
	</ul>
{/if}
