<script lang="ts">
	import { page } from '$app/state';
	import { isCurrent, navFor } from '$lib/nav';

	let {
		role,
		isGuardian = false,
		onNavigate
	}: { role: string | undefined; isGuardian?: boolean; onNavigate?: () => void } = $props();
	let groups = $derived(navFor(role, isGuardian));
</script>

<nav class="flex h-full flex-col gap-6 overflow-y-auto p-4" aria-label="Sections">
	{#each groups as group (group.title)}
		<div>
			<p
				class="mb-2 px-3 text-[0.7rem] font-medium tracking-[0.08em] text-ink-faint uppercase"
				dir="auto"
			>
				{group.title}
			</p>
			<ul class="list-none space-y-0.5 p-0">
				{#each group.items as item (item.href)}
					{@const current = isCurrent(item.href, page.url.pathname)}
					<li>
						<a
							class="flex items-center gap-2.5 rounded-xl px-3 py-2 text-sm transition-colors"
							class:bg-brand-soft={current}
							class:text-brand-text={current}
							class:font-medium={current}
							class:text-ink-soft={!current}
							class:hover:bg-sunken={!current}
							href={item.href}
							aria-current={current ? 'page' : undefined}
							onclick={onNavigate}
						>
							<item.icon size={17} aria-hidden="true" />
							<span dir="auto">{item.label}</span>
						</a>
					</li>
				{/each}
			</ul>
		</div>
	{/each}
</nav>
