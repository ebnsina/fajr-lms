<script lang="ts">
	import { enhance } from '$app/forms';
	import { page } from '$app/state';
	import Sidebar from '$lib/components/Sidebar.svelte';
	import ThemeToggle from '$lib/components/ThemeToggle.svelte';
	import ArrowLeftRight from '@lucide/svelte/icons/arrow-left-right';
	import LogOut from '@lucide/svelte/icons/log-out';
	import Menu from '@lucide/svelte/icons/menu';
	import X from '@lucide/svelte/icons/x';
	import type { LayoutProps } from './$types';

	let { data, children }: LayoutProps = $props();
	let session = $derived(data.session);
	let open = $state(false);

	// Close on navigation, so the drawer never covers the page it just opened.
	$effect(() => {
		page.url.pathname;
		open = false;
	});
</script>

<!-- Inset: the navigation sits on the page ground and the work sits on a panel
     lifted above it, so the two never read as one continuous surface. -->
<div class="flex min-h-dvh bg-ground">
	{#if open}
		<button
			class="fixed inset-0 z-30 bg-black/40 lg:hidden"
			type="button"
			aria-label="Close the menu"
			onclick={() => (open = false)}
		></button>
	{/if}

	<aside
		class="fixed inset-y-0 z-40 flex w-64 shrink-0 flex-col bg-ground transition-transform lg:static lg:translate-x-0"
		class:-translate-x-full={!open}
		class:rtl:translate-x-full={!open}
		class:translate-x-0={open}
	>
		<div class="flex h-16 items-center gap-2 px-6">
			<a class="text-lg font-semibold tracking-tight" href="/">Fajr</a>
			<button
				class="btn btn-sm btn-quiet ms-auto lg:hidden"
				type="button"
				aria-label="Close the menu"
				onclick={() => (open = false)}
			>
				<X size={16} aria-hidden="true" />
			</button>
		</div>
		<Sidebar role={session?.tenant?.role} onNavigate={() => (open = false)} />
	</aside>

	<div class="flex min-w-0 flex-1 flex-col gap-3 p-3 lg:ps-0">
		<main class="card min-w-0 flex-1 overflow-hidden p-0">
			<header
				class="flex h-16 items-center gap-3 border-b border-line px-4 sm:px-6"
			>
				<button
					class="btn btn-sm btn-quiet lg:hidden"
					type="button"
					aria-label="Open the menu"
					onclick={() => (open = true)}
				>
					<Menu size={16} aria-hidden="true" />
				</button>

				{#if session?.tenant}
					<span class="chip max-w-[12rem] truncate" dir="auto">{session.tenant.name}</span>
					<span class="hidden text-sm text-ink-soft sm:inline" dir="auto">
						{session.tenant.role}
					</span>
				{/if}

				<div class="ms-auto flex items-center gap-2">
					<span class="hidden text-sm text-ink-soft sm:inline" dir="auto">
						{session?.user.full_name ?? ''}
					</span>
					{#if (session?.memberships.length ?? 0) > 1}
						<a class="btn btn-sm btn-quiet" href="/tenant" title="Switch school">
							<ArrowLeftRight size={15} aria-hidden="true" />
							<span class="hidden sm:inline">Switch</span>
						</a>
					{/if}
					<ThemeToggle theme={data.theme} />
					<form method="POST" action="/login?/logout" use:enhance>
						<button class="btn btn-sm btn-quiet" type="submit" title="Sign out">
							<LogOut size={15} aria-hidden="true" />
							<span class="hidden sm:inline">Sign out</span>
						</button>
					</form>
				</div>
			</header>

			<div class="p-4 sm:p-6 lg:p-8">
				<div class="mx-auto max-w-4xl">
					{@render children()}
				</div>
			</div>
		</main>
	</div>
</div>
