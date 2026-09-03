<script lang="ts">
	import { enhance } from '$app/forms';
	import Award from '@lucide/svelte/icons/award';
	import ExternalLink from '@lucide/svelte/icons/external-link';
	import Select from '$lib/components/Select.svelte';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	let recipient = $state('');
	let revoking = $state<string | null>(null);

	const issued = (iso: string) =>
		new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' }).format(new Date(iso));

	// Somebody who has finished is offered first; the rest are still allowed,
	// because a school knows things the progress bar does not.
	const candidates = $derived(
		[...data.candidates]
			.sort((a, b) => b.percent_complete - a.percent_complete)
			.map((row) => ({
				value: row.enrollment.user_id,
				label: row.full_name,
				hint: `${row.percent_complete}% through`
			}))
	);
</script>

<svelte:head><title>Certificates · {data.course.title} · Fajr LMS</title></svelte:head>

<div class="mb-6">
	<a class="text-sm text-brand-text" href="/courses/{data.slug}">← Back to the course</a>
	<h1 class="mt-1 text-2xl font-semibold tracking-tight" dir="auto">Certificates</h1>
	<p class="mb-0 text-sm text-ink-soft">
		Every certificate carries a serial anybody can check, without an account.
	</p>
</div>

{#if form?.message}
	<p class="banner banner-bad mb-4" role="alert">{form.message}</p>
{:else if form?.awarded}
	<p class="banner mb-4" role="status">Awarded.</p>
{:else if form?.revoked}
	<p class="banner mb-4" role="status">Revoked. The public page now says so.</p>
{/if}

<form class="card mb-6 flex flex-wrap items-end gap-3" method="POST" action="?/award" use:enhance>
	<input type="hidden" name="course_id" value={data.course.id} />
	<input type="hidden" name="user_id" value={recipient} />
	<div class="min-w-64 flex-1">
		<span class="mb-1.5 block text-sm font-medium">Award to</span>
		{#if candidates.length === 0}
			<p class="mb-0 py-2.5 text-sm text-ink-soft">
				Everybody enrolled already holds one.
			</p>
		{:else}
			<Select
				id="recipient"
				label="Award to"
				placeholder="Choose a learner"
				bind:value={recipient}
				options={candidates}
			/>
		{/if}
	</div>
	<button class="btn" type="submit" disabled={!recipient}>Award a certificate</button>
</form>

{#if data.certificates.length === 0}
	<div class="card flex items-start gap-3 text-sm text-ink-soft">
		<Award class="mt-0.5 shrink-0" size={18} aria-hidden="true" />
		<p class="mb-0">Nothing awarded on this course yet.</p>
	</div>
{:else}
	<div class="flex flex-col gap-3">
		{#each data.certificates as row (row.certificate.id)}
			<article class="card" class:opacity-70={row.certificate.revoked_at}>
				<div class="flex flex-wrap items-center gap-4">
					<div class="min-w-48 flex-1">
						<p class="mb-0 font-medium" dir="auto">{row.full_name}</p>
						<p class="mb-0 font-mono text-xs text-ink-soft">{row.certificate.serial}</p>
					</div>
					<span class="text-sm text-ink-soft">{issued(row.certificate.issued_at)}</span>
					{#if row.certificate.grade_percent !== null}
						<span class="chip chip-brand font-mono">{row.certificate.grade_percent}%</span>
					{/if}
					{#if row.certificate.revoked_at}
						<span class="chip">Revoked</span>
					{:else}
						<button
							class="btn btn-sm btn-quiet"
							type="button"
							onclick={() =>
								(revoking = revoking === row.certificate.id ? null : row.certificate.id)}
						>
							Revoke
						</button>
					{/if}
					<a
						class="btn btn-sm btn-quiet"
						href={row.verify_url}
						target="_blank"
						rel="noreferrer"
						aria-label="Open the public page for {row.full_name}"
					>
						<ExternalLink size={16} aria-hidden="true" />
					</a>
				</div>

				{#if row.certificate.revoked_reason}
					<p class="mt-3 mb-0 text-sm text-ink-soft" dir="auto">
						{row.certificate.revoked_reason}
					</p>
				{/if}

				{#if revoking === row.certificate.id}
					<form
						class="mt-4 flex flex-wrap items-end gap-3 border-t border-line pt-4"
						method="POST"
						action="?/revoke"
						use:enhance={() => async ({ update }) => {
							revoking = null;
							await update();
						}}
					>
						<input type="hidden" name="certificate_id" value={row.certificate.id} />
						<div class="min-w-56 flex-1">
							<label class="mb-1.5 block text-sm font-medium" for="reason-{row.certificate.id}">
								Why <span class="font-normal text-ink-soft">· shown on the public page</span>
							</label>
							<input class="field" id="reason-{row.certificate.id}" name="reason" dir="auto" />
						</div>
						<button class="btn" type="submit">Revoke it</button>
						<button class="btn btn-quiet" type="button" onclick={() => (revoking = null)}>
							Cancel
						</button>
					</form>
				{/if}
			</article>
		{/each}
	</div>
{/if}
