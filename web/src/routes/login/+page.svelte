<script lang="ts">
	import { enhance } from '$app/forms';
	import AuthLayout from '$lib/components/AuthLayout.svelte';
	import OtpInput from '$lib/components/OtpInput.svelte';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	let sending = $state(false);

	let submitting = () => {
		sending = true;
		return async ({ update }: { update: (o?: { reset?: boolean }) => Promise<void> }) => {
			await update({ reset: false });
			sending = false;
		};
	};
</script>

<svelte:head><title>Sign in · Fajr LMS</title></svelte:head>

<AuthLayout
	theme={data.theme}
	heading={form?.sent ? 'Check your messages' : 'Sign in to Fajr LMS'}
	subheading={form?.sent
		? 'We sent a six digit code. It is good for ten minutes.'
		: 'One code by SMS. No password to remember, and none to lose.'}
>
	{#if form?.message}
		<p class="banner-bad mb-5 text-sm" dir="auto">{form.message}</p>
	{/if}

	{#if form?.sent}
		<form method="POST" action="?/verify" use:enhance={submitting} class="space-y-5">
			<input type="hidden" name="destination" value={form.destination} />

			<p class="banner text-sm" dir="auto">
				Sent to <span class="font-mono" dir="ltr">{form.destination}</span>
			</p>

			<div>
				<label class="mb-1.5 block text-sm font-medium" for="code">Six digit code</label>
				<OtpInput />
			</div>

			<div>
				<label class="mb-1.5 block text-sm font-medium" for="full_name">Your name</label>
				<input
					class="field"
					id="full_name"
					name="full_name"
					dir="auto"
					autocomplete="name"
					placeholder="only the first time"
				/>
				<p class="mt-1.5 text-sm text-ink-soft" dir="auto">
					Write it as it should appear on your certificate.
				</p>
			</div>

			<button class="btn w-full" type="submit" disabled={sending}>
				{sending ? 'Checking…' : 'Continue'}
			</button>
		</form>

		<form method="POST" action="?/request" use:enhance class="mt-3">
			<input type="hidden" name="destination" value={form.destination} />
			<button class="btn btn-quiet w-full" type="submit">Send another code</button>
		</form>
	{:else}
		<form method="POST" action="?/request" use:enhance={submitting} class="space-y-5">
			<div>
				<label class="mb-1.5 block text-sm font-medium" for="destination">
					Phone number or email
				</label>
				<input
					class="field font-mono"
					id="destination"
					name="destination"
					value={form?.destination ?? ''}
					dir="ltr"
					autocomplete="username"
					placeholder="+8801XXXXXXXXX"
					required
				/>
				<p class="mt-1.5 text-sm text-ink-soft" dir="auto">Include the country code.</p>
			</div>

			<button class="btn w-full" type="submit" disabled={sending}>
				{sending ? 'Sending…' : 'Send code'}
			</button>
		</form>

		{#if data.sso}
			<div class="my-5 flex items-center gap-3 text-sm text-ink-faint">
				<span class="h-px flex-1 bg-line"></span>
				or
				<span class="h-px flex-1 bg-line"></span>
			</div>
			<form method="POST" action="?/sso">
				<input type="hidden" name="school" value={data.school} />
				<button class="btn btn-quiet w-full" type="submit">
					Continue with {data.sso.label}
				</button>
			</form>
		{/if}
	{/if}

	{#snippet footer()}
		New here? Signing in with a number your school has on file creates your account.
	{/snippet}
</AuthLayout>
