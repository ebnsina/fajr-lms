<script lang="ts">
	import { enhance } from '$app/forms';
	import type { ActionData } from './$types';

	let { form }: { form: ActionData } = $props();
	let sending = $state(false);
</script>

<svelte:head><title>Sign in · Fajr</title></svelte:head>

<div class="mx-auto max-w-sm pt-10">
	<h1 class="mb-1 text-2xl font-bold tracking-tight">Sign in to Fajr</h1>
	<p class="mb-6 text-sm text-ink-soft">
		We send a six digit code. No password to remember or lose.
	</p>

	<div class="card">
		{#if form?.message}
			<p class="banner-bad mb-4 text-sm">{form.message}</p>
		{/if}

		{#if form?.sent}
			<form
				method="POST"
				action="?/verify"
				use:enhance={() => {
					sending = true;
					return async ({ update }) => {
						await update();
						sending = false;
					};
				}}
			>
				<input type="hidden" name="destination" value={form.destination} />

				<p class="banner mb-4 text-sm">
					Code sent to <span dir="auto" class="font-semibold">{form.destination}</span>
				</p>

				<label class="mb-1 block font-semibold" for="code">Six digit code</label>
				<input
					class="field mb-4 text-center font-mono text-lg tracking-[0.4em]"
					id="code"
					name="code"
					inputmode="numeric"
					autocomplete="one-time-code"
					maxlength="6"
					required
					dir="ltr"
				/>

				<label class="mb-1 block font-semibold" for="full_name">Your name</label>
				<input
					class="field mb-1"
					id="full_name"
					name="full_name"
					dir="auto"
					autocomplete="name"
					placeholder="Only needed the first time"
				/>
				<p class="mb-4 text-sm text-ink-soft">Write it as you want it on your certificate.</p>

				<button class="btn w-full justify-center" type="submit" disabled={sending}>
					{sending ? 'Checking…' : 'Continue'}
				</button>
			</form>

			<form method="POST" action="?/request" use:enhance class="mt-3">
				<input type="hidden" name="destination" value={form.destination} />
				<button class="btn btn-quiet w-full justify-center text-sm" type="submit">
					Send another code
				</button>
			</form>
		{:else}
			<form
				method="POST"
				action="?/request"
				use:enhance={() => {
					sending = true;
					return async ({ update }) => {
						await update();
						sending = false;
					};
				}}
			>
				<label class="mb-1 block font-semibold" for="destination">Phone number or email</label>
				<input
					class="field mb-1"
					id="destination"
					name="destination"
					value={form?.destination ?? ''}
					dir="ltr"
					autocomplete="username"
					placeholder="+8801XXXXXXXXX"
					required
				/>
				<p class="mb-4 text-sm text-ink-soft">Include the country code.</p>

				<button class="btn w-full justify-center" type="submit" disabled={sending}>
					{sending ? 'Sending…' : 'Send code'}
				</button>
			</form>
		{/if}
	</div>
</div>
