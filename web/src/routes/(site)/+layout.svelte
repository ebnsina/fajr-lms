<script lang="ts">
	import { page } from '$app/state';
	import type { Snippet } from 'svelte';
	import type { SiteNavItem } from '$lib/types.site';

	let { children }: { children: Snippet } = $props();

	const site = $derived(
		page.data as {
			tenant?: string;
			nav?: SiteNavItem[];
			theme?: string;
			page?: { tenant_name?: string; dir?: 'auto' | 'ltr' | 'rtl' };
		}
	);
	const home = $derived(`/site/${site.tenant ?? ''}`);
</script>

<div
	class="flex min-h-dvh flex-col bg-ground"
	dir={site.theme === 'gulf' ? 'rtl' : (site.page?.dir ?? 'auto')}
	data-site-theme={site.theme ?? 'plain'}
>
	<header class="border-b border-line bg-surface">
		<nav class="mx-auto flex max-w-5xl flex-wrap items-center gap-x-6 gap-y-2 px-6 py-4">
			<a class="font-semibold" href={home} dir="auto">{site.page?.tenant_name ?? 'Home'}</a>
			<div class="flex flex-1 flex-wrap items-center gap-4 text-sm">
				{#each site.nav ?? [] as item (item.slug)}
					<a class="text-ink-soft hover:text-ink" href="{home}/{item.slug}" dir="auto">
						{item.nav_label}
					</a>
				{/each}
			</div>
			<a class="btn btn-sm" href="/login?school={site.tenant ?? ''}">Sign in</a>
		</nav>
	</header>

	<main class="flex-1">
		{@render children()}
	</main>

	<footer class="border-t border-line px-6 py-8 text-center text-sm text-ink-soft">
		<p class="mb-0" dir="auto">{site.page?.tenant_name ?? ''} · powered by Fajr LMS</p>
	</footer>
</div>
