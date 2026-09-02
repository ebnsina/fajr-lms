<script lang="ts">
	import { page } from '$app/state';
	import AccountMenu from '$lib/components/AccountMenu.svelte';
	import NotificationBell from '$lib/components/NotificationBell.svelte';
	import Sidebar from '$lib/components/Sidebar.svelte';
	import { breadcrumbs } from '$lib/breadcrumbs';
	import ArrowLeftRight from '@lucide/svelte/icons/arrow-left-right';
	import Menu from '@lucide/svelte/icons/menu';
	import Search from '@lucide/svelte/icons/search';
	import X from '@lucide/svelte/icons/x';
	import type { LayoutProps } from './$types';

	let { data, children }: LayoutProps = $props();
	let session = $derived(data.session);
	let open = $state(false);
	let crumbs = $derived(breadcrumbs(page.url.pathname, page.data));

	// Close on navigation, so the drawer never covers the page it just opened.
	$effect(() => {
		page.url.pathname;
		open = false;
	});
</script>

<div class="min-h-dvh bg-ground">
	{#if open}
		<button
			class="fixed inset-0 z-30 bg-black/40 transition-opacity motion-reduce:transition-none lg:hidden"
			type="button"
			aria-label="Close the menu"
			onclick={() => (open = false)}
		></button>
	{/if}

	<!-- The sidebar is always position: fixed, so it never scrolls with the
	     content column; only its mobile visibility is toggled by translate. -->
	<aside
		class="fixed inset-y-0 start-0 z-40 flex w-64 shrink-0 flex-col border-e border-line bg-ground transition-transform duration-200 motion-reduce:transition-none"
		class:max-lg:-translate-x-full={!open}
		class:max-lg:rtl:translate-x-full={!open}
	>
		<div class="flex h-16 shrink-0 items-center gap-2 px-4">
			{#if session?.tenant}
				<div class="min-w-0 flex-1">
					<p class="truncate text-sm font-semibold tracking-tight" dir="auto">
						{session.tenant.name}
					</p>
					{#if (session.memberships?.length ?? 0) > 1}
						<a
							class="inline-flex items-center gap-1 text-xs text-ink-soft transition-colors hover:text-ink"
							href="/tenant"
						>
							<ArrowLeftRight size={12} aria-hidden="true" />
							Switch school
						</a>
					{/if}
				</div>
			{:else}
				<a class="text-lg font-semibold tracking-tight" href="/">Fajr</a>
			{/if}
			<button
				class="btn btn-sm btn-quiet lg:hidden"
				type="button"
				aria-label="Close the menu"
				onclick={() => (open = false)}
			>
				<X size={16} aria-hidden="true" />
			</button>
		</div>

		<div class="min-h-0 flex-1 overflow-y-auto">
			<Sidebar role={session?.tenant?.role} onNavigate={() => (open = false)} />
		</div>

		{#if session?.user}
			<div class="shrink-0 border-t border-line p-3">
				<AccountMenu fullName={session.user.full_name} onNavigate={() => (open = false)} />
			</div>
		{/if}
	</aside>

	<div class="flex min-w-0 flex-col lg:ms-64">
		<header
			class="sticky top-0 z-20 flex h-16 shrink-0 items-center gap-3 border-b border-line bg-ground/95 px-4 backdrop-blur sm:px-6"
		>
			<button
				class="btn btn-sm btn-quiet lg:hidden"
				type="button"
				aria-label="Open the menu"
				onclick={() => (open = true)}
			>
				<Menu size={16} aria-hidden="true" />
			</button>

			<nav aria-label="Breadcrumb" class="min-w-0">
				<ol class="m-0 flex list-none items-center gap-1.5 overflow-hidden p-0 text-sm">
					{#each crumbs as crumb, i (crumb.href)}
						{#if i > 0}
							<li class="text-ink-faint" aria-hidden="true">/</li>
						{/if}
						<li class="min-w-0 truncate">
							{#if i === crumbs.length - 1}
								<span class="font-medium text-ink" dir="auto" aria-current="page">
									{crumb.label}
								</span>
							{:else}
								<a
									class="text-ink-soft transition-colors hover:text-ink"
									dir="auto"
									href={crumb.href}
								>
									{crumb.label}
								</a>
							{/if}
						</li>
					{/each}
				</ol>
			</nav>

			<div class="ms-auto flex items-center gap-2">
				<div class="hidden sm:block">
					<label class="sr-only" for="content-search">Search</label>
					<div class="relative">
						<Search
							class="pointer-events-none absolute inset-y-0 start-3 my-auto text-ink-faint"
							size={15}
							aria-hidden="true"
						/>
						<input
							id="content-search"
							class="field h-9 w-56 ps-9"
							type="search"
							placeholder="Search"
							title="Search is not ready yet"
							readonly
							aria-disabled="true"
						/>
					</div>
				</div>
				<NotificationBell notifications={data.recentNotifications} unread={data.unread} />
			</div>
		</header>

		<main class="min-w-0 p-4 sm:p-6 lg:p-8">
			{@render children()}
		</main>
	</div>
</div>
