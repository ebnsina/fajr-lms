<script lang="ts">
	import Trophy from '@lucide/svelte/icons/trophy';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	let locale = $derived(data.session?.tenant?.locale ?? 'en');
	const count = (n: number) => new Intl.NumberFormat(locale).format(n);
</script>

<svelte:head><title>Standing · Fajr LMS</title></svelte:head>

<header class="mb-6">
	<h1 class="text-2xl font-semibold tracking-tight" dir="auto">Standing</h1>
	<p class="mt-1 text-sm text-ink-soft" dir="auto">
		Points come from finishing lessons, passing quizzes and completing courses.
	</p>
</header>

<section class="card mb-6">
	<div class="flex flex-wrap items-center gap-4">
		<div class="min-w-0 flex-1">
			<p class="mb-0.5 text-sm text-ink-soft">Your points</p>
			<p class="mb-0 font-mono text-2xl font-semibold tabular-nums">{count(data.mine.points)}</p>
		</div>
		{#if data.mine.badges.length > 0}
			<ul class="flex list-none flex-wrap gap-2 p-0">
				{#each data.mine.badges as badge (badge.id)}
					<li class="flex items-center gap-2 rounded-xl border border-line bg-raised px-3 py-1.5 text-sm">
						{#if badge.emoji}<span aria-hidden="true">{badge.emoji}</span>{/if}
						<span dir="auto">{badge.name}</span>
					</li>
				{/each}
			</ul>
		{/if}
	</div>
</section>

{#if !data.mine.on}
	<div class="card flex items-start gap-3 text-sm text-ink-soft" dir="auto">
		<Trophy class="mt-0.5 shrink-0" size={18} aria-hidden="true" />
		<p class="mb-0">This school does not keep a leaderboard.</p>
	</div>
{:else}
	<div class="mb-4 flex flex-wrap items-center justify-between gap-3">
		<h2 class="mb-0 text-lg font-semibold tracking-tight">
			{data.window === 'all' ? 'Since the beginning' : 'This month'}
		</h2>
		<a
			class="btn btn-sm btn-quiet"
			href="/leaderboard?window={data.window === 'all' ? 'month' : 'all'}"
		>
			{data.window === 'all' ? 'Show this month' : 'Show all time'}
		</a>
	</div>

	{#if data.standings.length === 0}
		<div class="card text-sm text-ink-soft">
			<p class="mb-0">Nobody has earned anything yet.</p>
		</div>
	{:else}
		<ol class="list-none space-y-2 p-0">
			{#each data.standings as row, index (row.user_id)}
				<li
					class="card flex items-center gap-4"
					class:border-brand-line={row.user_id === data.me}
				>
					<span class="w-8 font-mono text-sm text-ink-faint tabular-nums">{index + 1}</span>
					<span class="min-w-0 flex-1 font-medium" dir="auto">
						{row.full_name}{row.user_id === data.me ? ' · you' : ''}
					</span>
					<span class="font-mono tabular-nums">{count(row.points)}</span>
				</li>
			{/each}
		</ol>
	{/if}
{/if}
