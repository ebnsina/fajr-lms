<script lang="ts">
	import { enhance } from '$app/forms';
	import { dismissible } from '$lib/actions/dismiss';
	import Check from '@lucide/svelte/icons/check';
	import ChevronsUpDown from '@lucide/svelte/icons/chevrons-up-down';

	type School = { id: string; slug: string; name: string; role: string };

	let {
		schools,
		current
	}: { schools: School[]; current: { slug: string; name: string; role: string } | null } = $props();

	let open = $state(false);
	let trigger = $state<HTMLButtonElement | null>(null);
	let menu = $state<HTMLDivElement | null>(null);

	// A list of every school, not a link away to choose one.
	function close(returnFocus = true) {
		open = false;
		if (returnFocus) trigger?.focus();
	}

	$effect(() => {
		if (open && menu) menu.querySelector<HTMLElement>('[role="menuitemradio"]')?.focus();
	});
</script>

<div class="relative w-full">
	<button
		bind:this={trigger}
		class="flex w-full items-center gap-2 rounded-xl border border-line bg-surface px-3 py-2 text-start transition-colors hover:border-line-strong hover:bg-raised"
		type="button"
		aria-haspopup="menu"
		aria-expanded={open}
		aria-controls="school-menu"
		disabled={schools.length < 2}
		onclick={() => (open = !open)}
	>
		<span class="min-w-0 flex-1">
			<span class="block truncate text-sm font-semibold tracking-tight" dir="auto">
				{current?.name ?? 'Fajr LMS'}
			</span>
			<span class="block truncate text-xs text-ink-soft" dir="auto">
				{schools.length > 1 ? 'Switch school' : (current?.role ?? '')}
			</span>
		</span>
		{#if schools.length > 1}
			<ChevronsUpDown class="shrink-0 text-ink-faint" size={15} aria-hidden="true" />
		{/if}
	</button>

	{#if open}
		<div
			bind:this={menu}
			id="school-menu"
			class="absolute inset-x-0 top-full z-50 mt-1.5 overflow-hidden rounded-xl border border-line-strong bg-surface p-1"
			role="menu"
			aria-label="Your schools"
			use:dismissible={close}
		>
			{#each schools as school (school.id)}
				{@const isCurrent = school.slug === current?.slug}
				<form method="POST" action="/tenant" use:enhance={() => () => close(false)}>
					<input type="hidden" name="slug" value={school.slug} />
					<button
						class="flex w-full items-center gap-2 rounded-xl px-3 py-2.5 text-start text-sm transition-colors hover:bg-sunken"
						type="submit"
						role="menuitemradio"
						aria-checked={isCurrent}
					>
						<span class="min-w-0 flex-1">
							<span class="block truncate" dir="auto">{school.name}</span>
							<span class="block truncate text-xs text-ink-soft" dir="auto">{school.role}</span>
						</span>
						{#if isCurrent}
							<Check class="shrink-0 text-brand-text" size={15} aria-hidden="true" />
						{/if}
					</button>
				</form>
			{/each}
		</div>
	{/if}
</div>
