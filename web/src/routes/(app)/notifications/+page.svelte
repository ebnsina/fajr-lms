<script lang="ts">
	import { enhance } from '$app/forms';
	import BellOff from '@lucide/svelte/icons/bell-off';
	import CheckCheck from '@lucide/svelte/icons/check-check';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	let locale = $derived(data.session?.tenant?.locale ?? 'en');

	// Relative where it helps, absolute once it stops helping.
	function when(iso: string): string {
		const then = new Date(iso);
		const days = Math.round((then.getTime() - Date.now()) / 86_400_000);
		if (Math.abs(days) < 7) {
			return new Intl.RelativeTimeFormat(locale, { numeric: 'auto' }).format(days, 'day');
		}
		return new Intl.DateTimeFormat(locale, { dateStyle: 'medium' }).format(then);
	}
</script>

<svelte:head><title>Notifications · Fajr LMS</title></svelte:head>

<header class="mb-6 flex flex-wrap items-center gap-3">
	<div>
		<h1 class="text-2xl font-semibold tracking-tight" dir="auto">Notifications</h1>
		<p class="mt-1 text-sm text-ink-soft" dir="auto">
			{data.unread === 0 ? 'Nothing unread.' : `${data.unread} unread.`}
		</p>
	</div>
	{#if data.unread > 0}
		<form class="ms-auto" method="POST" action="?/readAll" use:enhance>
			<button class="btn btn-sm btn-quiet" type="submit">
				<CheckCheck size={15} aria-hidden="true" />
				Mark all read
			</button>
		</form>
	{/if}
</header>

{#if data.notifications.length === 0}
	<div class="card flex items-start gap-3 text-sm text-ink-soft" dir="auto">
		<BellOff class="mt-0.5 shrink-0" size={18} aria-hidden="true" />
		<p class="mb-0">Nothing yet. Results, payments and absences appear here.</p>
	</div>
{:else}
	<ul class="list-none space-y-2 p-0">
		{#each data.notifications as item (item.id)}
			<li class="card p-4" class:border-brand-line={!item.read_at}>
				<div class="flex items-start gap-3">
					<span class="min-w-0 flex-1">
						<span class="block font-medium" dir="auto">{item.title}</span>
						{#if item.body}
							<span class="mt-0.5 block text-sm text-ink-soft" dir="auto">{item.body}</span>
						{/if}
					</span>
					<span class="shrink-0 text-sm text-ink-faint" dir="auto">{when(item.created_at)}</span>
				</div>
			</li>
		{/each}
	</ul>
{/if}
