<script lang="ts">
	import { dismissible } from '$lib/actions/dismiss';
	import Bell from '@lucide/svelte/icons/bell';

	type Notification = {
		id: string;
		title: string;
		body: string;
		read_at: string | null;
		created_at: string;
	};

	let { notifications, unread }: { notifications: Notification[]; unread: number } = $props();

	let open = $state(false);
	let trigger: HTMLButtonElement | undefined = $state();
	let menu: HTMLDivElement | undefined = $state();

	function close() {
		if (!open) return;
		open = false;
		trigger?.focus();
	}

	$effect(() => {
		if (open) menu?.querySelector<HTMLElement>('[role="menuitem"]')?.focus();
	});
</script>

<div class="relative">
	<button
		bind:this={trigger}
		class="btn btn-sm btn-quiet relative"
		type="button"
		aria-haspopup="menu"
		aria-expanded={open}
		aria-controls="notification-menu"
		aria-label={unread > 0 ? `Notifications, ${unread} unread` : 'Notifications'}
		onclick={() => (open = !open)}
	>
		<Bell size={16} aria-hidden="true" />
		{#if unread > 0}
			<span
				class="absolute -top-1 -end-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-danger px-1 font-mono text-[0.65rem] text-white"
				aria-hidden="true"
			>
				{unread > 9 ? '9+' : unread}
			</span>
		{/if}
	</button>

	{#if open}
		<div
			bind:this={menu}
			id="notification-menu"
			class="absolute end-0 top-full z-50 mt-2 w-80 max-w-[calc(100vw-2rem)] origin-top-right overflow-hidden rounded-xl border border-line-strong bg-surface p-1 transition-[opacity,transform] motion-reduce:transition-none"
			role="menu"
			aria-label="Notifications"
			use:dismissible={close}
		>
			{#if notifications.length === 0}
				<p class="px-3 py-4 text-center text-sm text-ink-soft" dir="auto">Nothing yet.</p>
			{:else}
				<ul class="list-none space-y-0.5 p-0">
					{#each notifications as item (item.id)}
						<li>
							<a
								class="block rounded-xl px-3 py-2 text-sm transition-colors hover:bg-sunken"
								class:bg-brand-soft={!item.read_at}
								href="/notifications"
								role="menuitem"
								onclick={close}
							>
								<span class="block font-medium" dir="auto">{item.title}</span>
								{#if item.body}
									<span class="mt-0.5 block truncate text-xs text-ink-soft" dir="auto">
										{item.body}
									</span>
								{/if}
							</a>
						</li>
					{/each}
				</ul>
			{/if}
			<a
				class="mt-1 block rounded-xl px-3 py-2 text-center text-sm font-medium text-brand-text transition-colors hover:bg-sunken"
				href="/notifications"
				role="menuitem"
				onclick={close}
			>
				View all
			</a>
		</div>
	{/if}
</div>
