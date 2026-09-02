<script lang="ts">
	import { enhance } from '$app/forms';
	import Award from '@lucide/svelte/icons/award';
	import ExternalLink from '@lucide/svelte/icons/external-link';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const issued = (iso: string) =>
		new Intl.DateTimeFormat(undefined, { dateStyle: 'long' }).format(new Date(iso));
</script>

<svelte:head><title>Certificates · Fajr LMS</title></svelte:head>

<header class="mb-6">
	<h1 class="text-2xl font-semibold tracking-tight">Certificates</h1>
	<p class="mt-1 mb-0 text-sm text-ink-soft">
		Every certificate carries a serial anyone can check, without an account.
	</p>
</header>

{#if form?.message}
	<p class="banner banner-bad mb-4" role="alert">{form.message}</p>
{:else if form?.claimed}
	<p class="banner mb-4" role="status">Awarded. It is listed below.</p>
{/if}

{#if data.claimable.length > 0}
	<div class="card mb-6">
		<h2 class="mb-3 text-sm font-semibold tracking-wide uppercase text-ink-soft">
			Ready to claim
		</h2>
		<div class="flex flex-col gap-2">
			{#each data.claimable as row (row.enrollment.id)}
				<form
					class="flex flex-wrap items-center gap-3 rounded-xl bg-sunken px-4 py-3"
					method="POST"
					action="?/claim"
					use:enhance
				>
					<input type="hidden" name="course_id" value={row.enrollment.course_id} />
					<span class="min-w-40 flex-1 font-medium" dir="auto">{row.title}</span>
					<button class="btn btn-sm" type="submit">Claim the certificate</button>
				</form>
			{/each}
		</div>
	</div>
{/if}

{#if data.certificates.length === 0}
	<div class="card flex items-start gap-3 text-sm text-ink-soft">
		<Award class="mt-0.5 shrink-0" size={18} aria-hidden="true" />
		<p class="mb-0">
			Nothing yet. Finish a course and its certificate appears here, ready to share.
		</p>
	</div>
{:else}
	<div class="grid gap-4 sm:grid-cols-2">
		{#each data.certificates as row (row.certificate.id)}
			<article class="card" class:opacity-60={row.certificate.revoked_at}>
				<header class="mb-3 flex items-start justify-between gap-2">
					<h2 class="mb-0 font-medium" dir="auto">{row.certificate.course_title}</h2>
					{#if row.certificate.revoked_at}
						<span class="chip">Revoked</span>
					{:else if row.certificate.grade_percent !== null}
						<span class="chip chip-brand font-mono">{row.certificate.grade_percent}%</span>
					{/if}
				</header>
				<dl class="grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
					<dt class="text-ink-soft">Awarded to</dt>
					<dd class="mb-0" dir="auto">{row.certificate.recipient_name}</dd>
					<dt class="text-ink-soft">By</dt>
					<dd class="mb-0" dir="auto">{row.certificate.issuer_name}</dd>
					<dt class="text-ink-soft">On</dt>
					<dd class="mb-0">{issued(row.certificate.issued_at)}</dd>
					<dt class="text-ink-soft">Serial</dt>
					<dd class="mb-0 font-mono text-xs">{row.certificate.serial}</dd>
				</dl>
				<a
					class="btn btn-sm btn-quiet mt-4"
					href={row.verify_url}
					target="_blank"
					rel="noreferrer"
				>
					<ExternalLink size={16} aria-hidden="true" /> Open the public page
				</a>
			</article>
		{/each}
	</div>
{/if}
