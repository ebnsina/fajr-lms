<script lang="ts">
	import StatCard from '$lib/components/StatCard.svelte';
	import type { CourseProgress } from '$lib/api';
	import Award from '@lucide/svelte/icons/award';
	import BookOpen from '@lucide/svelte/icons/book-open';
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import GraduationCap from '@lucide/svelte/icons/graduation-cap';

	type Course = {
		enrollment: { id: string; status: string };
		slug: string;
		title: string;
		dir: string;
		progress: CourseProgress | null;
	};

	let { courses, certificates }: { courses: Course[]; certificates: number } = $props();

	// The API stores a per-course direction; anything unexpected falls back to auto.
	const dirOf = (value: string) => (value === 'ltr' || value === 'rtl' ? value : 'auto');

	let done = $derived(courses.filter((row) => row.enrollment.status === 'completed').length);
	let lessonsDone = $derived(courses.reduce((sum, row) => sum + (row.progress?.lessons_done ?? 0), 0));
	let lessonsTotal = $derived(
		courses.reduce((sum, row) => sum + (row.progress?.lessons_total ?? 0), 0)
	);
	let percent = $derived(lessonsTotal === 0 ? 0 : Math.round((lessonsDone / lessonsTotal) * 100));

	let shownPercent = $derived(
		new Intl.NumberFormat(undefined, { style: 'percent' }).format(percent / 100)
	);

	// A ring drawn as one stroked circle: the dash is the part that is finished.
	const radius = 46;
	const circumference = 2 * Math.PI * radius;
	let filled = $derived((percent / 100) * circumference);
</script>

<section class="mb-8">
	<h2 class="mb-3 text-lg font-semibold" dir="auto">Your learning</h2>
	<div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
		<StatCard
			icon={BookOpen}
			label="Courses in progress"
			value={courses.length - done}
			href="/courses"
		/>
		<StatCard icon={CircleCheck} label="Courses finished" value={done} />
		<StatCard icon={GraduationCap} label="Lessons finished" value={lessonsDone} href="/grades" />
		<StatCard icon={Award} label="Certificates" value={certificates} href="/certificates" />
	</div>
</section>

<div class="mb-8 grid gap-4 lg:grid-cols-[auto_1fr]">
	<section class="card flex flex-col items-center justify-center">
		<svg
			class="block"
			width="120"
			height="120"
			viewBox="0 0 120 120"
			role="img"
			aria-label="{shownPercent} of your lessons are finished: {lessonsDone} of {lessonsTotal}."
		>
			<circle
				cx="60"
				cy="60"
				r={radius}
				fill="none"
				stroke="var(--color-sunken)"
				stroke-width="12"
			/>
			{#if percent > 0}
				<circle
					cx="60"
					cy="60"
					r={radius}
					fill="none"
					stroke="var(--color-brand)"
					stroke-width="12"
					stroke-linecap="round"
					stroke-dasharray="{filled} {circumference}"
					transform="rotate(-90 60 60)"
				/>
			{/if}
			<text
				x="60"
				y="60"
				text-anchor="middle"
				dominant-baseline="central"
				font-size="24"
				font-family="var(--font-mono)"
				fill="var(--color-ink)"
			>
				{shownPercent}
			</text>
		</svg>
		<p class="mt-3 mb-0 text-center text-sm text-ink-soft">
			{lessonsDone} of {lessonsTotal} lessons finished
		</p>
	</section>

	<section class="card">
		<h2 class="mb-1 text-base font-semibold" dir="auto">Your courses</h2>
		<p class="mb-4 text-sm text-ink-soft">Where you left off in each one.</p>

		{#if courses.length === 0}
			<p class="mb-0 text-sm text-ink-soft">
				You are not enrolled in anything yet. Once a teacher adds you, or you join a course, it
				appears here.
			</p>
		{:else}
			<ul class="m-0 flex list-none flex-col gap-1 p-0">
				{#each courses as row (row.enrollment.id)}
					{@const pc = row.progress?.percent_complete ?? 0}
					<li>
						<a
							class="block rounded-xl px-3 py-2 transition-colors hover:bg-sunken"
							href="/courses/{row.slug}"
						>
							<span class="flex items-center gap-3">
								<span class="min-w-0 flex-1 truncate text-sm font-medium" dir={dirOf(row.dir)}>
									{row.title}
								</span>
								<span class="shrink-0 font-mono text-xs text-ink-soft">
									{new Intl.NumberFormat(undefined, { style: 'percent' }).format(pc / 100)}
								</span>
							</span>
							<span class="mt-1.5 block h-1.5 overflow-hidden rounded-xl bg-sunken">
								<span class="block h-full rounded-xl bg-brand" style="inline-size: {pc}%"></span>
							</span>
						</a>
					</li>
				{/each}
			</ul>
		{/if}
	</section>
</div>
