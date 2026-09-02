<script lang="ts">
	import { enhance } from '$app/forms';
	import AuthLayout from '$lib/components/AuthLayout.svelte';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	let none = $derived(data.tenants.length === 0);
</script>

<svelte:head><title>Choose a place · Fajr</title></svelte:head>

<AuthLayout
	theme={data.theme}
	heading={none ? 'You are not part of a school yet' : 'Where are you working today?'}
	subheading={none
		? 'Someone at your madrasa, school or training provider has to add you first.'
		: 'You belong to more than one place. Pick the one you want to work in.'}
>
	{#if form?.message}
		<p class="banner-bad mb-5 text-sm" dir="auto">{form.message}</p>
	{/if}

	{#if none}
		<p class="mb-6 text-ink-soft" dir="auto">
			If you were expecting access, give them the number you signed in with. Once they add
			you, it appears here.
		</p>
		<form method="POST" action="/login?/logout" use:enhance>
			<button class="btn btn-quiet" type="submit">Sign out</button>
		</form>
	{:else}
		<div class="space-y-3">
			{#each data.tenants as tenant (tenant.id)}
				<form method="POST" use:enhance>
					<input type="hidden" name="slug" value={tenant.slug} />
					<button
						class="card w-full cursor-pointer p-4 text-start transition hover:border-line-strong"
						class:border-brand={tenant.slug === data.current}
						type="submit"
					>
						<span class="flex items-center gap-3">
							<span class="min-w-0 flex-1">
								<span class="block font-medium" dir="auto">{tenant.name}</span>
								<span class="mt-0.5 block text-sm text-ink-soft" dir="auto">
									{tenant.role} · {tenant.kind}
								</span>
							</span>
							{#if tenant.slug === data.current}
								<span class="chip chip-brand shrink-0">Current</span>
							{/if}
						</span>
					</button>
				</form>
			{/each}
		</div>
	{/if}

	{#snippet footer()}
		Signed in as <span class="font-mono" dir="auto">{data.session?.user.full_name ?? ''}</span>
	{/snippet}
</AuthLayout>
