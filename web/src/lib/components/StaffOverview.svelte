<script lang="ts">
	import StatCard from '$lib/components/StatCard.svelte';
	import { relativeTime } from '$lib/time';
	import ClipboardCheck from '@lucide/svelte/icons/clipboard-check';
	import Inbox from '@lucide/svelte/icons/inbox';
	import Library from '@lucide/svelte/icons/library';
	import Receipt from '@lucide/svelte/icons/receipt';

	type Attempt = {
		quiz_attempt: { id: string; submitted_at: string | null };
		full_name: string;
		quiz_title: string;
	};
	type Submission = {
		submission: { id: string; is_late: boolean; submitted_at: string | null };
		full_name: string;
		assignment_title: string;
	};

	let {
		attempts,
		submissions,
		courseCount,
		reviewCount,
		isAdmin
	}: {
		attempts: Attempt[];
		submissions: Submission[];
		courseCount: number;
		reviewCount: number;
		isAdmin: boolean;
	} = $props();

	// Everything waiting, newest first, whichever queue it came from.
	let waiting = $derived(
		[
			...attempts.map((row) => ({
				id: row.quiz_attempt.id,
				at: row.quiz_attempt.submitted_at,
				who: row.full_name,
				what: row.quiz_title,
				href: `/grading/${row.quiz_attempt.id}`,
				kind: 'Quiz',
				late: false
			})),
			...submissions.map((row) => ({
				id: row.submission.id,
				at: row.submission.submitted_at,
				who: row.full_name,
				what: row.assignment_title,
				href: `/submissions/${row.submission.id}`,
				kind: 'Assignment',
				late: row.submission.is_late
			}))
		]
			.filter((row) => row.at !== null)
			.sort((a, b) => Date.parse(b.at as string) - Date.parse(a.at as string))
	);

	let weekday = new Intl.DateTimeFormat(undefined, { weekday: 'short' });

	// Seven buckets ending today, counted from when the work actually arrived.
	let days = $derived.by(() => {
		const start = new Date();
		start.setHours(0, 0, 0, 0);
		return Array.from({ length: 7 }, (_, i) => {
			const from = new Date(start);
			from.setDate(from.getDate() - 6 + i);
			const to = new Date(from);
			to.setDate(to.getDate() + 1);
			const count = waiting.filter((row) => {
				const at = Date.parse(row.at as string);
				return at >= from.getTime() && at < to.getTime();
			}).length;
			return { key: from.toISOString(), label: weekday.format(from), count };
		});
	});

	let peak = $derived(Math.max(1, ...days.map((d) => d.count)));
	let chartLabel = $derived(
		new Intl.ListFormat(undefined, { style: 'long', type: 'conjunction' }).format(
			days.map((d) => `${d.label}: ${d.count}`)
		)
	);
</script>

<section class="mb-8">
	<h2 class="mb-3 text-lg font-semibold" dir="auto">Your school</h2>
	<div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
		<StatCard
			icon={ClipboardCheck}
			label="Quizzes to grade"
			value={attempts.length}
			href="/grading"
		/>
		<StatCard
			icon={Inbox}
			label="Assignments to grade"
			value={submissions.length}
			hint={submissions.some((row) => row.submission.is_late) ? 'Some arrived late' : undefined}
			href="/submissions"
		/>
		<StatCard icon={Library} label="Courses" value={courseCount} href="/courses" />
		{#if isAdmin}
			<StatCard icon={Receipt} label="Payments to review" value={reviewCount} href="/payments" />
		{/if}
	</div>
</section>

<div class="mb-8 grid gap-4 lg:grid-cols-2">
	<section class="card">
		<h2 class="mb-1 text-base font-semibold" dir="auto">Work waiting, by the day it arrived</h2>
		<p class="mb-4 text-sm text-ink-soft">Everything still in the two grading queues.</p>

		<svg class="block w-full" viewBox="0 0 322 120" role="img" aria-label={chartLabel}>
			<line
				x1="0"
				y1="96"
				x2="322"
				y2="96"
				stroke="var(--color-line-strong)"
				stroke-width="1"
			/>
			{#each days as day, i (day.key)}
				{@const height = Math.round((day.count / peak) * 80)}
				<rect
					x={i * 46 + 9}
					y={96 - Math.max(height, 2)}
					width="28"
					height={Math.max(height, 2)}
					rx="4"
					fill={day.count > 0 ? 'var(--color-brand)' : 'var(--color-line-strong)'}
				/>
				<text
					x={i * 46 + 23}
					y={96 - Math.max(height, 2) - 6}
					text-anchor="middle"
					font-size="11"
					fill="var(--color-ink-soft)"
				>
					{day.count > 0 ? day.count : ''}
				</text>
				<text
					x={i * 46 + 23}
					y="113"
					text-anchor="middle"
					font-size="11"
					fill="var(--color-ink-faint)"
				>
					{day.label}
				</text>
			{/each}
		</svg>
	</section>

	<section class="card">
		<h2 class="mb-1 text-base font-semibold" dir="auto">Waiting on you</h2>
		<p class="mb-4 text-sm text-ink-soft">The most recent arrivals in either queue.</p>

		{#if waiting.length === 0}
			<p class="mb-0 text-sm text-ink-soft">Nothing is waiting to be graded.</p>
		{:else}
			<ul class="m-0 flex list-none flex-col gap-1 p-0">
				{#each waiting.slice(0, 5) as row (row.id)}
					<li>
						<a
							class="flex items-center gap-3 rounded-xl px-3 py-2 transition-colors hover:bg-sunken"
							href={row.href}
						>
							<span class="min-w-0 flex-1">
								<span class="block truncate text-sm font-medium" dir="auto">{row.what}</span>
								<span class="block truncate text-xs text-ink-soft" dir="auto">{row.who}</span>
							</span>
							{#if row.late}
								<span class="chip shrink-0">Late</span>
							{/if}
							<span class="shrink-0 text-xs text-ink-faint">
								{relativeTime(row.at as string)}
							</span>
						</a>
					</li>
				{/each}
			</ul>
		{/if}
	</section>
</div>
