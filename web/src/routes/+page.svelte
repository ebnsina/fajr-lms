<script lang="ts">
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	let name = $derived(data.session?.user.full_name ?? '');

	// The API stores a per-course direction; anything unexpected falls back to auto.
	const dirOf = (value: string) => (value === 'ltr' || value === 'rtl' ? value : 'auto');
</script>

<svelte:head><title>Home · Fajr</title></svelte:head>

<h1 class="mb-1 text-2xl font-bold tracking-tight">
	Assalamu alaikum, <span dir="auto">{name}</span>
</h1>
<p class="mb-6 text-sm text-ink-soft">
	{#if data.unread > 0}
		You have {data.unread} unread {data.unread === 1 ? 'message' : 'messages'}.
	{:else}
		Nothing new since you were last here.
	{/if}
</p>

<h2 class="mb-3 text-lg font-semibold">Your courses</h2>

{#if data.enrollments.length === 0}
	<div class="card text-sm text-ink-soft">
		<p class="mb-0">
			You are not enrolled in anything yet. Once a teacher adds you, or you join a course, it
			appears here.
		</p>
	</div>
{:else}
	<ul class="grid list-none gap-3 p-0 sm:grid-cols-2">
		{#each data.enrollments as row (row.enrollment.id)}
			<li>
				<a class="card block transition hover:border-brand" href="/courses/{row.slug}">
					<span class="block font-semibold" dir={dirOf(row.dir)}>
						{row.title}
					</span>
					<span class="mt-1 block text-sm text-ink-soft">
						{row.enrollment.status === 'completed' ? 'Completed' : 'In progress'}
					</span>
				</a>
			</li>
		{/each}
	</ul>
{/if}
