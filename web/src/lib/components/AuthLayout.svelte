<script lang="ts">
	import ThemeToggle from '$lib/components/ThemeToggle.svelte';

	let {
		theme,
		heading,
		subheading,
		children,
		footer
	}: {
		theme: 'light' | 'dark' | 'system';
		heading: string;
		subheading: string;
		children: any;
		footer?: any;
	} = $props();

	// Written as content rather than decoration: these are the things a school
	// actually asks about before it will move.
	const points = [
		{ title: 'Runs on the phone people already have', body: 'Nothing loads until it is asked for, and a lesson opens on a slow connection without eating the day’s data.' },
		{ title: 'Arabic, Bangla and English side by side', body: 'Names, titles and lessons render correctly in any script, in the same list, without a separate build.' },
		{ title: 'Fees the way they are actually paid', body: 'bKash, card, or a bank slip a member of staff approves. No card required to enrol.' }
	];
</script>

<div class="grid min-h-dvh lg:grid-cols-2">
	<!-- The form comes first in the source, so a screen reader and a narrow
	     screen both reach it without wading through the pitch. -->
	<main class="flex flex-col px-6 py-8 sm:px-10 lg:px-16">
		<div class="flex items-center gap-3">
			<span class="text-lg font-semibold tracking-tight">Fajr</span>
			<span class="ms-auto"><ThemeToggle {theme} /></span>
		</div>

		<div class="flex flex-1 items-center py-10">
			<div class="w-full max-w-sm">
				<h1 class="text-3xl font-semibold tracking-tight" dir="auto">{heading}</h1>
				<p class="mt-2 mb-8 text-ink-soft" dir="auto">{subheading}</p>
				{@render children()}
			</div>
		</div>

		{#if footer}
			<div class="text-sm text-ink-soft" dir="auto">{@render footer()}</div>
		{/if}
	</main>

	<!-- Hidden below lg rather than stacked: on a phone it would push the form
	     under a screenful of marketing. -->
	<aside class="hidden border-s border-line bg-sunken p-16 lg:flex lg:flex-col lg:justify-center">
		<div class="max-w-md">
			<p class="mb-10 text-2xl leading-snug font-medium tracking-tight" dir="auto">
				One place for the whole term: lessons, marking, attendance, fees and the
				message home.
			</p>

			<ul class="list-none space-y-6 p-0">
				{#each points as point (point.title)}
					<li class="border-s-2 border-brand-line ps-4">
						<p class="mb-1 font-medium" dir="auto">{point.title}</p>
						<p class="mb-0 text-sm text-ink-soft" dir="auto">{point.body}</p>
					</li>
				{/each}
			</ul>
		</div>
	</aside>
</div>
