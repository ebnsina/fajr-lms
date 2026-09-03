<script lang="ts">
	import { page } from '$app/state';
	import type { LayoutProps } from './$types';

	let { data, children }: LayoutProps = $props();

	const links = [
		{ href: '/admin', label: 'Overview' },
		{ href: '/admin/leads', label: 'Leads' },
		{ href: '/admin/schools', label: 'Schools' }
	];
	const here = (href: string) =>
		href === '/admin' ? page.url.pathname === '/admin' : page.url.pathname.startsWith(href);
</script>

<div class="min-h-dvh bg-ground">
	{#if data.staff}
		<header class="border-b border-line bg-surface">
			<nav class="mx-auto flex max-w-6xl flex-wrap items-center gap-x-6 gap-y-2 px-6 py-4">
				<span class="font-semibold tracking-tight">Fajr back office</span>
				<div class="flex items-center gap-5 text-sm">
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
				<form class="ms-auto" method="POST" action="/admin/login?/out">
					<button class="btn btn-sm btn-quiet" type="submit">Sign out</button>
				</form>
			</nav>
		</header>
	{/if}

	<main class="mx-auto max-w-6xl px-6 py-8">
		{@render children()}
	</main>
</div>
