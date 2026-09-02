<script lang="ts">
	import { enhance } from '$app/forms';
	import Select from '$lib/components/Select.svelte';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	let name = $state('');
	// The address is suggested from the name until somebody types their own.
	let slug = $state('');
	let touched = $state(false);
	const suggestion = $derived(
		slug || (touched ? slug : name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, ''))
	);

	const kinds = [
		{ value: 'institution', label: 'A school or madrasah' },
		{ value: 'creator', label: 'A teacher on my own' },
		{ value: 'corporate', label: 'A company training its staff' }
	];
	const directions = [
		{ value: 'auto', label: 'Mixed or English' },
		{ value: 'rtl', label: 'Arabic or Urdu' },
		{ value: 'ltr', label: 'Bengali' }
	];
	const currencies = [
		{ value: 'BDT', label: 'BDT', hint: 'Bangladeshi taka' },
		{ value: 'SAR', label: 'SAR', hint: 'Saudi riyal' },
		{ value: 'AED', label: 'AED', hint: 'UAE dirham' },
		{ value: 'PKR', label: 'PKR', hint: 'Pakistani rupee' },
		{ value: 'INR', label: 'INR', hint: 'Indian rupee' },
		{ value: 'USD', label: 'USD', hint: 'US dollar' }
	];
	let currency = $state('BDT');
</script>

<svelte:head><title>Open a school · Fajr LMS</title></svelte:head>

<section class="mx-auto max-w-xl px-6 pt-40 pb-24">
	<h1 class="font-display text-4xl font-extrabold tracking-tight">Open a school</h1>
	<p class="mt-2 text-ink-soft">
		This takes a minute. You become its owner, and you can invite teachers straight after.
	</p>

	{#if !data.signedIn}
		<p class="banner mt-6">
			You need an account first. <a class="font-medium underline" href="/login?next=/start">
				Sign in or create one
			</a>, then come back here.
		</p>
	{/if}

	{#if form?.message}
		<p class="banner banner-bad mt-6" role="alert">{form.message}</p>
	{/if}

	<form class="card mt-6 flex flex-col gap-4" method="POST" use:enhance>
		<div>
			<label class="mb-1.5 block text-sm font-medium" for="name">What is it called?</label>
			<input
				class="field"
				id="name"
				name="name"
				bind:value={name}
				placeholder="Greenfield Academy"
				dir="auto"
				required
			/>
		</div>

		<div>
			<label class="mb-1.5 block text-sm font-medium" for="slug">
				Its address <span class="font-normal text-ink-soft">· where your site will live</span>
			</label>
			<div class="flex items-center gap-2">
				<span class="font-mono text-sm text-ink-soft">/site/</span>
				<input
					class="field font-mono"
					id="slug"
					name="slug"
					value={suggestion}
					oninput={(event) => {
						touched = true;
						slug = event.currentTarget.value;
					}}
					placeholder="greenfield"
					dir="ltr"
				/>
			</div>
		</div>

		<fieldset>
			<legend class="mb-1.5 text-sm font-medium">Who is teaching?</legend>
			<div class="flex flex-col gap-2">
				{#each kinds as kind, i (kind.value)}
					<label class="flex items-center gap-2 text-sm">
						<input
							class="choice choice-round"
							type="radio"
							name="kind"
							value={kind.value}
							checked={i === 0}
						/>
						{kind.label}
					</label>
				{/each}
			</div>
		</fieldset>

		<fieldset>
			<legend class="mb-1.5 text-sm font-medium">What will you teach in?</legend>
			<div class="flex flex-wrap gap-4">
				{#each directions as choice, i (choice.value)}
					<label class="flex items-center gap-2 text-sm">
						<input
							class="choice choice-round"
							type="radio"
							name="dir"
							value={choice.value}
							checked={i === 0}
						/>
						{choice.label}
					</label>
				{/each}
			</div>
		</fieldset>

		<div class="w-40">
			<span class="mb-1.5 block text-sm font-medium">Fees in</span>
			<input type="hidden" name="currency" value={currency} />
			<Select id="currency" label="Fees in" bind:value={currency} options={currencies} />
		</div>

		<div class="flex justify-end">
			<button class="btn" type="submit" disabled={!data.signedIn}>Open the school</button>
		</div>
	</form>
</section>
