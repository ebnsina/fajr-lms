<script lang="ts">
	import { page } from '$app/state';
	import ThemeToggle from '$lib/components/ThemeToggle.svelte';
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import type { Snippet } from 'svelte';

	let { data, children }: { data: { theme: 'light' | 'dark' | 'system' }; children: Snippet } =
		$props();

	// Section anchors on the product page, so the menu works from anywhere.
	const links = [
		{ href: '/welcome#why', label: 'Why Fajr' },
		{ href: '/welcome#how', label: 'How it works' },
		{ href: '/welcome#features', label: 'Features' },
		{ href: '/welcome#website', label: 'Website' },
		{ href: '/welcome#compare', label: 'Compare' },
		{ href: '/pricing', label: 'Pricing' },
		{ href: '/welcome#faq', label: 'FAQ' }
	];
	const here = (href: string) => !href.includes('#') && page.url.pathname === href;

	const year = new Date().getFullYear();
</script>

<div class="font-body flex min-h-dvh flex-col bg-ground" dir="ltr">
	<!-- No rule under the header: it sits on the hero rather than above it. -->
	<header class="absolute inset-x-0 top-0 z-20">
		<nav class="mx-auto flex max-w-6xl items-center px-6 py-6">
			<a class="font-display text-lg font-bold" href="/welcome">Fajr LMS</a>

			<div class="hidden flex-1 items-center justify-center gap-7 text-sm lg:flex">
				{#each links as link (link.href)}
					<a
						class="text-ink-soft transition-colors hover:text-ink"
						class:text-ink={here(link.href)}
						class:font-medium={here(link.href)}
						href={link.href}
						aria-current={here(link.href) ? 'page' : undefined}
					>
						{link.label}
					</a>
				{/each}
			</div>

			<div class="ms-auto flex items-center gap-2 lg:ms-0">
				<a class="btn btn-sm btn-quiet" href="/get-a-demo">See a demo</a>
				<a class="btn btn-sm" href="/start">
					Get started
					<ArrowRight size={15} aria-hidden="true" />
				</a>
			</div>
		</nav>
	</header>

	<main class="flex-1">
		{@render children()}
	</main>

	<div class="relative -mt-12">
		<footer class="rounded-t-4xl border border-b-0 border-line bg-surface px-6 pt-16 pb-8">
		<div class="mx-auto grid max-w-6xl gap-12 lg:grid-cols-5">
			<div class="lg:col-span-2">
				<p class="mb-3 font-display text-xl font-bold">Fajr LMS</p>
				<p class="mb-6 max-w-sm text-ink-soft">
					Everything it takes to teach, in one place. Build the course, set the work, grade it,
					collect the fee, and keep the people at home in the loop — without stitching five tools
					together or waiting on a developer.
				</p>
				<a class="btn btn-sm" href="/start">
					Get started
					<ArrowRight size={15} aria-hidden="true" />
				</a>
			</div>

			<div>
				<h2 class="mb-4 text-sm font-medium">Product</h2>
				<ul class="flex flex-col gap-2.5 text-sm text-ink-soft">
					<li><a class="transition-colors hover:text-ink" href="/welcome#why">Why Fajr</a></li>
					<li><a class="transition-colors hover:text-ink" href="/welcome#how">How it works</a></li>
					<li><a class="transition-colors hover:text-ink" href="/welcome#features">Features</a></li>
					<li><a class="transition-colors hover:text-ink" href="/welcome#website">Your website</a></li>
					<li><a class="transition-colors hover:text-ink" href="/welcome#compare">Compare</a></li>
					<li><a class="transition-colors hover:text-ink" href="/pricing">Pricing</a></li>
					<li><a class="transition-colors hover:text-ink" href="/get-a-demo">See a demo</a></li>
				</ul>
			</div>

			<div>
				<h2 class="mb-4 text-sm font-medium">Who it is for</h2>
				<ul class="flex flex-col gap-2.5 text-sm text-ink-soft">
					<li><a class="transition-colors hover:text-ink" href="/welcome#who">Schools</a></li>
					<li><a class="transition-colors hover:text-ink" href="/welcome#who">Madrasahs</a></li>
					<li><a class="transition-colors hover:text-ink" href="/welcome#who">Colleges</a></li>
					<li><a class="transition-colors hover:text-ink" href="/welcome#who">Universities</a></li>
					<li><a class="transition-colors hover:text-ink" href="/welcome#who">Teachers</a></li>
				</ul>
			</div>

			<div>
				<h2 class="mb-4 text-sm font-medium">Your account</h2>
				<ul class="flex flex-col gap-2.5 text-sm text-ink-soft">
					<li><a class="transition-colors hover:text-ink" href="/login">Sign in</a></li>
					<li><a class="transition-colors hover:text-ink" href="/start">Open a school</a></li>
					<li><a class="transition-colors hover:text-ink" href="/tenant">Switch school</a></li>
					<li><a class="transition-colors hover:text-ink" href="/welcome#faq">Questions</a></li>
				</ul>
			</div>
		</div>

		<div
			class="mx-auto mt-14 flex max-w-6xl flex-wrap items-center gap-x-3 gap-y-4 border-t border-line pt-6 text-sm text-ink-faint"
		>
			<span>© {year} Fajr LMS. Your data stays yours.</span>
			<span class="ms-auto"><ThemeToggle theme={data.theme} /></span>
			<span aria-hidden="true">·</span>
			<span>A product of Fajr Labs <span aria-label="Bangladesh">🇧🇩</span></span>
		</div>
		</footer>
	</div>
</div>
