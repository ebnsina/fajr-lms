<script lang="ts">
	import { enhance } from '$app/forms';
	import Receipt from '@lucide/svelte/icons/receipt';
	import Paperclip from '@lucide/svelte/icons/paperclip';
	import ExternalLink from '@lucide/svelte/icons/external-link';
	import Check from '@lucide/svelte/icons/check';
	import X from '@lucide/svelte/icons/x';
	import Undo2 from '@lucide/svelte/icons/undo-2';
	import { money } from '$lib/api';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	let locale = $derived(data.session?.tenant?.locale ?? 'en');

	function when(iso: string): string {
		return new Intl.DateTimeFormat(locale, { dateStyle: 'medium' }).format(new Date(iso));
	}

	let refunding = $state<string | null>(null);

	// What is left, written the way the payer would type it.
	function major(minor: number, currency: string): string {
		const digits =
			new Intl.NumberFormat('en', { style: 'currency', currency }).resolvedOptions()
				.maximumFractionDigits ?? 2;
		return (minor / 10 ** digits).toFixed(digits);
	}
</script>

<svelte:head><title>Payments · Fajr LMS</title></svelte:head>

<header class="mb-6">
	<h1 class="text-2xl font-semibold tracking-tight" dir="auto">Payments to review</h1>
	<p class="mt-1 text-sm text-ink-soft" dir="auto">
		Bank transfers and wallet payments waiting for someone to confirm them. Approving enrolls the
		learner straight away.
	</p>
</header>

{#if form?.message}
	<p class="banner-bad mb-5 text-sm" dir="auto">{form.message}</p>
{/if}

{#if data.orders.length === 0}
	<div class="card flex items-start gap-3 text-sm text-ink-soft" dir="auto">
		<Receipt class="mt-0.5 shrink-0" size={18} aria-hidden="true" />
		<p class="mb-0">Nothing waiting. Card and wallet payments confirm themselves.</p>
	</div>
{:else}
	<ul class="list-none space-y-3 p-0">
		{#each data.orders as row (row.order.id)}
			<li class="card">
				<div class="mb-4 flex flex-wrap items-start gap-3">
					<span class="min-w-0 flex-1">
						<span class="block font-medium" dir="auto">{row.full_name}</span>
						<span class="mt-0.5 block text-sm text-ink-soft" dir="auto">
							{row.title} · asked {when(row.order.created_at)}
						</span>
					</span>
					<span class="text-end">
						<span class="block font-medium" dir="auto">
							{money(row.order.amount_minor, row.order.currency, locale)}
						</span>
						<span class="mt-0.5 block font-mono text-sm text-ink-faint" dir="ltr">
							{row.order.reference}
						</span>
					</span>
				</div>

				{#if row.order.provider_ref || row.order.note}
					<dl class="mb-4 grid gap-2 text-sm sm:grid-cols-2">
						{#if row.order.provider_ref}
							<div>
								<dt class="text-ink-soft" dir="auto">Transaction id</dt>
								<dd class="mt-0.5 font-mono" dir="ltr">{row.order.provider_ref}</dd>
							</div>
						{/if}
						{#if row.order.note}
							<div>
								<dt class="text-ink-soft" dir="auto">What they said</dt>
								<dd class="mt-0.5" dir="auto">{row.order.note}</dd>
							</div>
						{/if}
					</dl>
				{/if}

				{#if row.proof?.url}
					<a
						class="mb-4 flex items-center gap-2.5 rounded-xl border border-line bg-raised px-3.5 py-2.5 text-sm transition hover:border-line-strong"
						href={row.proof.url}
						target="_blank"
						rel="noopener"
					>
						<Paperclip size={15} aria-hidden="true" />
						<span class="min-w-0 flex-1" dir="auto">Open the deposit slip</span>
						<ExternalLink class="shrink-0 text-ink-faint" size={14} aria-hidden="true" />
						<span class="sr-only">opens in a new tab</span>
					</a>
				{:else}
					<p class="banner mb-4 text-sm" dir="auto">
						No slip was attached. Check the transaction id against the account before approving.
					</p>
				{/if}

				<form method="POST" action="?/review" use:enhance class="flex flex-wrap items-end gap-3">
					<input type="hidden" name="order_id" value={row.order.id} />
					<div class="min-w-48 flex-1">
						<label class="mb-1.5 block text-sm font-medium" for="note-{row.order.id}">
							Note
							<span class="font-normal text-ink-soft">· optional, kept with the record</span>
						</label>
						<input
							class="field"
							id="note-{row.order.id}"
							name="note"
							dir="auto"
							placeholder="checked against the statement"
						/>
					</div>
					<button class="btn btn-quiet" type="submit" name="decision" value="reject">
						<X size={16} aria-hidden="true" />
						Reject
					</button>
					<button class="btn" type="submit" name="decision" value="approve">
						<Check size={16} aria-hidden="true" />
						Approve
					</button>
				</form>
			</li>
		{/each}
	</ul>
{/if}

<section class="mt-10">
	<header class="mb-4">
		<h2 class="text-lg font-semibold tracking-tight" dir="auto">Money taken</h2>
		<p class="mt-1 text-sm text-ink-soft" dir="auto">
			Payments that went through. Handing money back here records the refund and closes the
			enrolment it paid for; moving the money itself is done at the bank or the gateway.
		</p>
	</header>

	{#if data.paid.length === 0}
		<div class="card flex items-start gap-3 text-sm text-ink-soft" dir="auto">
			<Receipt class="mt-0.5 shrink-0" size={18} aria-hidden="true" />
			<p class="mb-0">No payments yet.</p>
		</div>
	{:else}
		<ul class="list-none space-y-3 p-0">
			{#each data.paid as row (row.order.id)}
				{@const left = row.order.amount_minor - row.order.refunded_minor}
				<li class="card">
					<div class="flex flex-wrap items-start gap-3">
						<span class="min-w-0 flex-1">
							<span class="block font-medium" dir="auto">{row.full_name}</span>
							<span class="mt-0.5 block text-sm text-ink-soft" dir="auto">
								{row.title}{row.order.paid_at ? ` · paid ${when(row.order.paid_at)}` : ''}
							</span>
						</span>
						<span class="text-end">
							<span class="block font-medium" dir="auto">
								{money(row.order.amount_minor, row.order.currency, locale)}
							</span>
							<span class="mt-0.5 block font-mono text-sm text-ink-faint" dir="ltr">
								{row.order.reference}
							</span>
						</span>
					</div>

					{#if row.order.refunded_minor > 0}
						<p class="mt-3 mb-0 text-sm text-ink-soft" dir="auto">
							{money(row.order.refunded_minor, row.order.currency, locale)} handed back{row.order
								.refund_reason
								? ` · ${row.order.refund_reason}`
								: ''}
						</p>
					{/if}

					{#if left > 0}
						{#if refunding === row.order.id}
							<form
								method="POST"
								action="?/refund"
								use:enhance
								class="mt-4 flex flex-wrap items-end gap-3 border-t border-line pt-4"
							>
								<input type="hidden" name="order_id" value={row.order.id} />
								<input type="hidden" name="currency" value={row.order.currency} />
								<div class="w-36">
									<label class="mb-1.5 block text-sm font-medium" for="amount-{row.order.id}">
										Amount
									</label>
									<input
										class="field font-mono"
										id="amount-{row.order.id}"
										name="amount"
										type="number"
										min="0"
										step="any"
										placeholder={major(left, row.order.currency)}
										dir="ltr"
									/>
								</div>
								<div class="min-w-48 flex-1">
									<label class="mb-1.5 block text-sm font-medium" for="reason-{row.order.id}">
										Reason
									</label>
									<input
										class="field"
										id="reason-{row.order.id}"
										name="reason"
										dir="auto"
										placeholder="withdrew before the course began"
									/>
								</div>
								<label class="flex items-center gap-2 self-end pb-3 text-sm">
									<input class="choice" type="checkbox" name="keep_access" />
									Keep their access
								</label>
								<button
									class="btn btn-quiet"
									type="button"
									onclick={() => (refunding = null)}
								>
									Cancel
								</button>
								<button class="btn" type="submit">Record the refund</button>
							</form>
						{:else}
							<div class="mt-4 flex justify-end">
								<button
									class="btn btn-quiet"
									type="button"
									onclick={() => (refunding = row.order.id)}
								>
									<Undo2 size={16} aria-hidden="true" />
									Refund
								</button>
							</div>
						{/if}
					{/if}
				</li>
			{/each}
		</ul>
	{/if}
</section>
