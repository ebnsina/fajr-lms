<script lang="ts">
	import { enhance } from '$app/forms';
	import { dismissible } from '$lib/actions/dismiss';
	import ChevronsUpDown from '@lucide/svelte/icons/chevrons-up-down';
	import LogOut from '@lucide/svelte/icons/log-out';
	import Settings from '@lucide/svelte/icons/settings';

	let { fullName, onNavigate }: { fullName: string; onNavigate?: () => void } = $props();

	let open = $state(false);
	let trigger: HTMLButtonElement | undefined = $state();
	let menu: HTMLDivElement | undefined = $state();

	let initials = $derived(
		fullName
			.trim()
			.split(/\s+/)
			.slice(0, 2)
			.map((part) => part[0]?.toUpperCase() ?? '')
			.join('') || '?'
	);

	function close() {
		if (!open) return;
		open = false;
		trigger?.focus();
	}

	function handleClose() {
		close();
		onNavigate?.();
	}

	// Focus moves into the menu once it opens, and Escape (handled by the
	// dismissible action) brings it back to the trigger.
	$effect(() => {
		if (open) menu?.querySelector<HTMLElement>('[role="menuitem"]')?.focus();
	});
</script>

<div class="relative w-full">
	<button
		bind:this={trigger}
		class="flex w-full items-center gap-2.5 rounded-xl border border-line bg-surface px-3 py-2 text-start text-sm transition-colors hover:border-line-strong hover:bg-raised"
		type="button"
		aria-haspopup="menu"
		aria-expanded={open}
		aria-controls="account-menu"
		onclick={() => (open = !open)}
	>
		<span
			class="flex size-8 shrink-0 items-center justify-center rounded-xl bg-brand-soft font-mono text-xs font-medium text-brand-text"
			aria-hidden="true"
		>
			{initials}
		</span>
		<span class="min-w-0 flex-1 truncate font-medium" dir="auto">{fullName}</span>
		<ChevronsUpDown class="shrink-0 text-ink-faint" size={15} aria-hidden="true" />
	</button>

	{#if open}
		<div
			bind:this={menu}
			id="account-menu"
			class="absolute inset-x-0 bottom-full z-50 mb-2 origin-bottom overflow-hidden rounded-xl border border-line-strong bg-surface p-1 transition-[opacity,transform] motion-reduce:transition-none"
			role="menu"
			aria-label="Account"
			use:dismissible={close}
		>
			<a
				class="flex items-center gap-2.5 rounded-xl px-3 py-2 text-sm text-ink transition-colors hover:bg-sunken"
				href="/settings"
				role="menuitem"
				onclick={handleClose}
			>
				<Settings size={16} aria-hidden="true" />
				Settings
			</a>
			<form method="POST" action="/login?/logout" use:enhance>
				<button
					class="flex w-full items-center gap-2.5 rounded-xl px-3 py-2 text-start text-sm text-danger transition-colors hover:bg-danger-soft"
					type="submit"
					role="menuitem"
				>
					<LogOut size={16} aria-hidden="true" />
					Sign out
				</button>
			</form>
		</div>
	{/if}
</div>
