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

<div class="flex min-h-dvh flex-col bg-ground">
	<header class="border-b border-line bg-surface">
		<nav class="mx-auto flex max-w-5xl items-center gap-6 px-6 py-4">
			<a class="font-semibold tracking-tight" href="/welcome">Fajr LMS</a>
			<div class="flex flex-1 items-center gap-5 text-sm">
				{#each links as link (link.href)}
					<a
						class="hover:text-ink"
						class:text-ink-soft={!here(link.href)}
						href={link.href}
						aria-current={here(link.href) ? 'page' : undefined}
					>
						{link.label}
					</a>
				{/each}
			</div>
			<ThemeToggle theme={data.theme} />
			<a class="btn btn-sm btn-quiet" href="/login">Sign in</a>
			<a class="btn btn-sm" href="/start">Open a school</a>
		</nav>
	</header>

	<main class="flex-1">
		{@render children()}
	</main>

	<footer class="border-t border-line px-6 py-12">
		<div class="mx-auto grid max-w-5xl gap-8 sm:grid-cols-3">
			<div>
				<p class="mb-1 font-semibold tracking-tight">Fajr LMS</p>
				<p class="mb-0 text-sm text-ink-soft">
					Teaching, grading and fees for schools in South Asia and the Gulf.
				</p>
			</div>
			<div>
				<h2 class="mb-2 text-sm font-medium">The product</h2>
				<ul class="flex flex-col gap-1.5 text-sm text-ink-soft">
					<li><a class="hover:text-ink" href="/welcome">What it does</a></li>
					<li><a class="hover:text-ink" href="/pricing">Pricing</a></li>
					<li><a class="hover:text-ink" href="/start">Open a school</a></li>
				</ul>
			</div>
			<div>
				<h2 class="mb-2 text-sm font-medium">Already with us</h2>
				<ul class="flex flex-col gap-1.5 text-sm text-ink-soft">
					<li><a class="hover:text-ink" href="/login">Sign in</a></li>
					<li><a class="hover:text-ink" href="/tenant">Switch school</a></li>
				</ul>
			</div>
		</div>
		<p class="mx-auto mt-8 max-w-5xl text-sm text-ink-faint">
			Built on open-source parts. Your data stays yours.
		</p>
	</footer>
</div>
