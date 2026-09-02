<script lang="ts">
	import { enhance } from '$app/forms';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
</script>

<svelte:head><title>Choose a place · Fajr</title></svelte:head>

{#if data.tenants.length === 0}
	<h1 class="mb-1 text-2xl font-bold tracking-tight">You are not part of a school yet</h1>
	<p class="mb-6 max-w-prose text-sm text-ink-soft">
		Someone from your madrasa, school or training provider needs to add you. Once they do, it
		appears here. If you were expecting access, give them the number you signed in with.
	</p>
	<form method="POST" action="/login?/logout">
		<button class="btn btn-quiet" type="submit">Sign out</button>
	</form>
{:else}
<h1 class="mb-1 text-2xl font-bold tracking-tight">Where are you working today?</h1>
<p class="mb-6 text-sm text-ink-soft">You belong to more than one place.</p>

{#if form?.message}
	<p class="banner-bad mb-4 text-sm">{form.message}</p>
{/if}

<div class="grid gap-3 sm:grid-cols-2">
	{#each data.tenants as tenant (tenant.id)}
		<form method="POST" use:enhance>
			<input type="hidden" name="slug" value={tenant.slug} />
			<button
				class="card w-full cursor-pointer text-start transition hover:border-line-strong"
				class:border-brand={tenant.slug === data.current}
				type="submit"
			>
				<span class="block font-semibold" dir="auto">{tenant.name}</span>
				<span class="mt-1 block text-sm text-ink-soft">{tenant.role} · {tenant.kind}</span>
				{#if tenant.slug === data.current}
					<span class="chip mt-2 inline-block">Current</span>
				{/if}
			</button>
		</form>
	{/each}
</div>
{/if}
