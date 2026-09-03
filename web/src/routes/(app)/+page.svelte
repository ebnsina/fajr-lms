<script lang="ts">
	import Clock from '$lib/components/Clock.svelte';
	import LearnerOverview from '$lib/components/LearnerOverview.svelte';
	import StaffOverview from '$lib/components/StaffOverview.svelte';
	import { relativeTime } from '$lib/time';
	import BellOff from '@lucide/svelte/icons/bell-off';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	let name = $derived(data.session?.user.full_name ?? '');
	let locale = $derived(data.session?.tenant?.locale ?? undefined);
</script>

<svelte:head><title>Overview · Fajr LMS</title></svelte:head>

<header class="mb-8 flex flex-wrap items-start justify-between gap-4">
	<h1 class="mb-0 text-2xl font-bold tracking-tight" dir="auto">
		Assalamu alaikum, <span dir="auto">{name}</span>
	</h1>
	<Clock />
</header>

{#if data.isStaff}
	<StaffOverview
		attempts={data.attempts}
		submissions={data.submissions}
		courseCount={data.courseCount}
		reviewCount={data.review.length}
		isAdmin={data.isAdmin}
	/>
{/if}

<LearnerOverview courses={data.courses} certificates={data.certificates} />

<section class="card">
	<h2 class="mb-1 text-base font-semibold" dir="auto">Recent activity</h2>
	<p class="mb-4 text-sm text-ink-soft">The last few things the school told you about.</p>

	{#if data.recentNotifications.length === 0}
		<p class="mb-0 flex items-center gap-2 text-sm text-ink-soft">
			<BellOff size={15} aria-hidden="true" />
			Nothing new since you were last here.
		</p>
	{:else}
		<ul class="m-0 flex list-none flex-col gap-1 p-0">
			{#each data.recentNotifications as note (note.id)}
				<li class="flex items-start gap-3 rounded-xl px-3 py-2" class:bg-sunken={!note.read_at}>
					<span class="min-w-0 flex-1">
						<span class="block text-sm font-medium" dir="auto">{note.title}</span>
						<span class="block text-sm text-ink-soft" dir="auto">{note.body}</span>
					</span>
					<span class="shrink-0 text-xs text-ink-faint">
						{relativeTime(note.created_at, locale)}
					</span>
				</li>
			{/each}
		</ul>
		<a class="btn btn-sm btn-quiet mt-4" href="/notifications">All notifications</a>
	{/if}
</section>
