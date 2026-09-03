<script lang="ts">
	import { enhance } from '$app/forms';
	import ThemeChoice from '$lib/components/ThemeChoice.svelte';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	let session = $derived(data.session);
	let provider = $derived(data.provider);
	let roles = ['student', 'instructor', 'admin'];
</script>

<svelte:head><title>Settings · Fajr LMS</title></svelte:head>

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

	{#if provider}
		<section class="card">
			<h2 class="mb-1 text-base font-semibold" dir="auto">Signing in with a school account</h2>
			<p class="mb-4 text-sm text-ink-soft" dir="auto">
				Point this at your Google Workspace or Microsoft account, and people sign in with the
				account they already have instead of a code. The link to give them is
				<span class="font-mono" dir="ltr">/login?school={session?.tenant?.slug}</span>.
			</p>

			{#if form?.message}
				<p class="banner-bad mb-4 text-sm" role="alert">{form.message}</p>
			{/if}

			<form method="POST" action="?/sso" use:enhance class="flex flex-col gap-4">
				<div>
					<label class="mb-1.5 block text-sm font-medium" for="issuer">Issuer</label>
					<input
						class="field font-mono"
						id="issuer"
						name="issuer"
						value={provider.issuer ?? ''}
						placeholder="https://accounts.google.com"
						dir="ltr"
						required
					/>
				</div>
				<div>
					<label class="mb-1.5 block text-sm font-medium" for="client_id">Client id</label>
					<input
						class="field font-mono"
						id="client_id"
						name="client_id"
						value={provider.client_id ?? ''}
						dir="ltr"
						required
					/>
				</div>
				<div>
					<label class="mb-1.5 block text-sm font-medium" for="client_secret">
						Client secret
						{#if provider.has_secret}
							<span class="font-normal text-ink-soft">· stored; leave empty to keep it</span>
						{/if}
					</label>
					<input
						class="field font-mono"
						id="client_secret"
						name="client_secret"
						type="password"
						autocomplete="off"
						dir="ltr"
					/>
				</div>
				<div>
					<label class="mb-1.5 block text-sm font-medium" for="allowed_domains">
						Addresses allowed
						<span class="font-normal text-ink-soft">· one domain per line, empty means any</span>
					</label>
					<textarea
						class="field h-auto min-h-20 py-2.5 font-mono"
						id="allowed_domains"
						name="allowed_domains"
						dir="ltr"
						placeholder="school.edu.bd">{(provider.allowed_domains ?? []).join('\n')}</textarea>
				</div>
				<div>
					<label class="mb-1.5 block text-sm font-medium" for="label">Words on the button</label>
					<input
						class="field"
						id="label"
						name="label"
						value={provider.label ?? 'Your school account'}
						dir="auto"
					/>
				</div>
				<div>
					<label class="mb-1.5 block text-sm font-medium" for="join_role">
						A new person joins as
					</label>
					<select class="field" id="join_role" name="join_role">
						{#each roles as role (role)}
							<option value={role} selected={provider.join_role === role}>{role}</option>
						{/each}
					</select>
				</div>
				<label class="flex items-start gap-2.5 text-sm">
					<input
						class="choice mt-0.5"
						type="checkbox"
						name="auto_join"
						checked={provider.auto_join ?? true}
					/>
					<span>Let anybody the provider vouches for join this school</span>
				</label>
				<label class="flex items-center gap-2.5 text-sm">
					<input
						class="choice"
						type="checkbox"
						name="enabled"
						checked={provider.enabled ?? true}
					/>
					Offer this on the sign-in page
				</label>

				<div class="flex justify-end gap-2">
					{#if provider.configured}
						<button class="btn btn-quiet" type="submit" formaction="?/removeSso">Remove</button>
					{/if}
					<button class="btn" type="submit">Save</button>
				</div>
			</form>
		</section>
	{/if}
</div>