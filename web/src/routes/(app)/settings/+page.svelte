<script lang="ts">
	import ThemeChoice from '$lib/components/ThemeChoice.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	let session = $derived(data.session);
</script>

<svelte:head><title>Settings · Fajr</title></svelte:head>

<header class="mb-6">
	<h1 class="text-2xl font-semibold tracking-tight" dir="auto">Settings</h1>
	<p class="mt-1 text-sm text-ink-soft" dir="auto">Your appearance and account details.</p>
</header>

<div class="grid gap-4 lg:max-w-2xl">
	<section class="card">
		<h2 class="mb-1 text-base font-semibold" dir="auto">Appearance</h2>
		<p class="mb-4 text-sm text-ink-soft" dir="auto">
			Auto follows this device's system setting.
		</p>
		<ThemeChoice theme={data.theme} />
	</section>

	{#if session?.user}
		<section class="card">
			<h2 class="mb-4 text-base font-semibold" dir="auto">Profile</h2>
			<dl class="m-0 grid grid-cols-[auto_1fr] gap-x-4 gap-y-3 text-sm">
				<dt class="text-ink-soft">Name</dt>
				<dd class="m-0" dir="auto">{session.user.full_name}</dd>

				{#if session.tenant}
					<dt class="text-ink-soft">School</dt>
					<dd class="m-0" dir="auto">{session.tenant.name}</dd>

					<dt class="text-ink-soft">Role</dt>
					<dd class="m-0" dir="auto">{session.tenant.role}</dd>
				{/if}
			</dl>
		</section>
	{/if}
</div>
