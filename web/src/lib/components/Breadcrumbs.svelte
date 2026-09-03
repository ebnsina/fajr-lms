<script lang="ts">
	import { dismissible } from '$lib/actions/dismiss';
	import type { Crumb } from '$lib/breadcrumbs';
	import MoreHorizontal from '@lucide/svelte/icons/more-horizontal';

	let { crumbs }: { crumbs: Crumb[] } = $props();

	let open = $state(false);
	let trigger = $state<HTMLButtonElement | null>(null);
	let menu = $state<HTMLDivElement | null>(null);

	// Past three levels the middle is folded away: the first crumb and the last
	// two are what people navigate by, the rest is depth they can still reach.
	let hidden = $derived(crumbs.length > 3 ? crumbs.slice(1, -2) : []);
	let tail = $derived(crumbs.length > 3 ? crumbs.slice(-2) : crumbs.slice(1));

	function close() {
		open = false;
		trigger?.focus({ preventScroll: true });
	}

	$effect(() => {
		if (!open || !menu) return;
		menu.querySelector<HTMLElement>('a')?.focus({ preventScroll: true });
	});
</script>

{#snippet crumb(item: Crumb, last: boolean)}
	<li class="min-w-0 truncate">
		{#if last}
			<span class="font-medium text-ink" dir="auto" aria-current="page">{item.label}</span>
		{:else}
			<a class="text-ink-soft transition-colors hover:text-ink" dir="auto" href={item.href}>
				{item.label}
			</a>
		{/if}
	</li>
{/snippet}

{#snippet divider()}
	<li class="text-ink-faint" aria-hidden="true">/</li>
{/snippet}

<nav aria-label="Breadcrumb" class="min-w-0">
	<ol class="m-0 flex list-none items-center gap-1.5 p-0 text-sm">
		{@render crumb(crumbs[0], crumbs.length === 1)}

		{#if hidden.length > 0}
			{@render divider()}
			<li class="relative shrink-0">
				<button
					bind:this={trigger}
					class="flex items-center rounded-xl px-1 py-0.5 text-ink-soft transition-colors hover:bg-sunken hover:text-ink"
					type="button"
					aria-haspopup="menu"
					aria-expanded={open}
					aria-controls="breadcrumb-more"
					aria-label="Show the {hidden.length} steps in between"
					onclick={() => (open = !open)}
				>
					<MoreHorizontal size={16} aria-hidden="true" />
				</button>

				{#if open}
					<div
						bind:this={menu}
						id="breadcrumb-more"
						class="absolute start-0 top-full z-50 mt-1.5 min-w-48 overflow-hidden rounded-xl border border-line-strong bg-surface p-1"
						role="menu"
						aria-label="Steps in between"
						use:dismissible={close}
					>
						{#each hidden as item (item.href)}
							<a
								class="block truncate rounded-xl px-3 py-2 text-sm text-ink transition-colors hover:bg-sunken"
								href={item.href}
								role="menuitem"
								dir="auto"
								onclick={() => (open = false)}
							>
								{item.label}
							</a>
						{/each}
					</div>
				{/if}
			</li>
		{/if}

		{#each tail as item, i (item.href)}
			{@render divider()}
			{@render crumb(item, i === tail.length - 1)}
		{/each}
	</ol>
</nav>
