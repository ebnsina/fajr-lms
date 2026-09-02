<script lang="ts">
	import { enhance } from '$app/forms';
	import ThemeToggle from '$lib/components/ThemeToggle.svelte';
	import type { LayoutProps } from './$types';

	let { data, children }: LayoutProps = $props();
	let session = $derived(data.session);
</script>

{#if session}
	<header class="border-b border-line bg-surface/80 backdrop-blur">
		<div class="mx-auto flex max-w-4xl flex-wrap items-center gap-3 px-4 py-3">
			<span class="text-lg font-semibold tracking-tight">Fajr</span>
			{#if session.tenant}
				<span class="chip" dir="auto">{session.tenant.name}</span>
				<span class="text-sm text-ink-soft">{session.tenant.role}</span>
			{/if}
			<span class="ms-auto text-sm text-ink-soft" dir="auto">{session.user.full_name}</span>
			{#if session.memberships.length > 1}
				<a class="text-sm text-brand-text underline-offset-4 hover:underline" href="/tenant">Switch</a>
			{/if}
			<ThemeToggle theme={data.theme} />
			<form method="POST" action="/login?/logout" use:enhance>
				<button class="btn btn-sm btn-quiet" type="submit">Sign out</button>
			</form>
		</div>
	</header>
{/if}

<main class="mx-auto max-w-4xl px-4 pt-6 pb-16">
	{@render children()}
</main>
