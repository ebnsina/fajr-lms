<script lang="ts">
	import { page } from '$app/state';
	import ThemeToggle from '$lib/components/ThemeToggle.svelte';
	import type { Snippet } from 'svelte';

	let { data, children }: { data: { theme: 'light' | 'dark' | 'system' }; children: Snippet } =
		$props();

	const links = [
		{ href: '/welcome', label: 'Product' },
		{ href: '/pricing', label: 'Pricing' }
	];
	const here = (href: string) => page.url.pathname === href;
</script>

<div class="font-body flex min-h-dvh flex-col bg-ground" dir="ltr">
	<!-- No rule under the header: it sits on the hero rather than above it. -->
	<header class="absolute inset-x-0 top-0 z-20">
		<nav class="mx-auto flex max-w-6xl items-center px-6 py-6">
			<a class="font-display text-lg font-bold" href="/welcome">Fajr LMS</a>

			<div class="flex flex-1 items-center justify-center gap-8 text-sm">
				{#each links as link (link.href)}
					<a
						class="transition-colors hover:text-ink"
						class:text-ink-soft={!here(link.href)}
						href={link.href}
						aria-current={here(link.href) ? 'page' : undefined}
					>
						{link.label}
					</a>
				{/each}
			</div>

			<div class="flex items-center gap-2">
				<a class="btn btn-sm btn-quiet" href="/login">Sign in</a>
				<a class="btn btn-sm" href="/start">Open a school</a>
			</div>
		</nav>
	</header>

	<main class="flex-1">
		{@render children()}
	</main>

	<footer class="border-t border-line px-6 py-14">
		<div class="mx-auto grid max-w-6xl gap-10 sm:grid-cols-3">
			<div>
				<p class="mb-1 font-display text-lg font-bold">Fajr LMS</p>
				<p class="mb-0 max-w-xs text-sm text-ink-soft">
					Teaching, grading and fees for schools in South Asia and the Gulf.
				</p>
			</div>
			<div>
				<h2 class="mb-3 text-sm font-medium">The product</h2>
				<ul class="flex flex-col gap-2 text-sm text-ink-soft">
					<li><a class="transition-colors hover:text-ink" href="/welcome">What it does</a></li>
					<li><a class="transition-colors hover:text-ink" href="/pricing">Pricing</a></li>
					<li><a class="transition-colors hover:text-ink" href="/start">Open a school</a></li>
				</ul>
			</div>
			<div>
				<h2 class="mb-3 text-sm font-medium">Already with us</h2>
				<ul class="flex flex-col gap-2 text-sm text-ink-soft">
					<li><a class="transition-colors hover:text-ink" href="/login">Sign in</a></li>
					<li><a class="transition-colors hover:text-ink" href="/tenant">Switch school</a></li>
				</ul>
			</div>
		</div>
		<div class="mx-auto mt-10 flex max-w-6xl flex-wrap items-center gap-4 text-sm text-ink-faint">
			<span>Built on open-source parts. Your data stays yours.</span>
			<span class="ms-auto"><ThemeToggle theme={data.theme} /></span>
		</div>
	</footer>
</div>
