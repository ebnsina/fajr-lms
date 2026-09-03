<script lang="ts">
	import { enhance } from '$app/forms';
	import type { PageProps } from './$types';

	let { form }: PageProps = $props();
</script>

<svelte:head><title>Back office · Fajr LMS</title></svelte:head>

<section class="mx-auto max-w-sm pt-24">
	<h1 class="text-2xl font-semibold tracking-tight">Back office</h1>
	<p class="mt-1 text-sm text-ink-soft">This one is not for schools.</p>

	{#if form?.message}
		<p class="banner-bad mt-5 text-sm" role="alert">{form.message}</p>
	{/if}

	<form class="card mt-5 flex flex-col gap-4" method="POST" action="?/in" use:enhance>
		<div>
			<label class="mb-1.5 block text-sm font-medium" for="email">Address</label>
			<input
				class="field"
				id="email"
				name="email"
				type="email"
				value={form?.email ?? ''}
				autocomplete="username"
				dir="ltr"
				required
			/>
		</div>
		<div>
			<label class="mb-1.5 block text-sm font-medium" for="password">Password</label>
			<input
				class="field"
				id="password"
				name="password"
				type="password"
				autocomplete="current-password"
				dir="ltr"
				required
			/>
		</div>
		<div class="flex justify-end">
			<button class="btn" type="submit">Sign in</button>
		</div>
	</form>
</section>
