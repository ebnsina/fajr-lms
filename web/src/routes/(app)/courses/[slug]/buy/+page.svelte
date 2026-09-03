<script lang="ts">
	import { enhance } from '$app/forms';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import Receipt from '@lucide/svelte/icons/receipt';
	import { money } from '$lib/api';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	let locale = $derived(data.session?.tenant?.locale ?? 'en');
	let course = $derived(data.course);
	let plan = $derived(data.plan?.payment_plan ?? null);

	let parts = $derived(course.installments > 1 ? course.installments : 0);
	let perPart = $derived(parts ? Math.floor(course.price_minor / parts) : 0);

	const when = (iso: string) =>
		new Intl.DateTimeFormat(locale, { dateStyle: 'medium' }).format(new Date(iso));
</script>

<svelte:head><title>Pay for {course.title} · Fajr LMS</title></svelte:head>

<nav class="mb-4 text-sm">
	<a
		class="inline-flex items-center gap-1.5 text-brand-text underline-offset-4 hover:underline"
		href="/courses/{data.slug}"
	>
		<ArrowLeft class="rtl:hidden" size={16} aria-hidden="true" />
		<ArrowRight class="hidden rtl:block" size={16} aria-hidden="true" />
		Back to the course
	</a>
</nav>

<header class="mb-6">
	<h1 class="text-2xl font-semibold tracking-tight" dir="auto">{course.title}</h1>
	<p class="mt-1 text-ink-soft" dir="auto">
		{money(course.price_minor, course.currency, locale)}{parts
			? ` · or ${parts} payments of about ${money(perPart, course.currency, locale)}`
			: ''}
	</p>
</header>

{#if form?.message}
	<p class="banner-bad mb-5 text-sm" role="alert">{form.message}</p>
{/if}

{#if plan}
	<section class="card mb-6">
		<h2 class="mb-1 text-sm font-semibold tracking-wide uppercase text-ink-soft">Your plan</h2>
		<p class="mb-0 text-sm text-ink-soft" dir="auto">
			{plan.paid_parts} of {plan.parts} payments made on {money(
				plan.total_minor,
				plan.currency,
				locale
			)}{plan.next_due_on ? ` · next due ${when(plan.next_due_on)}` : ' · paid off'}
		</p>
	</section>
{/if}

{#if data.open}
	<section class="card mb-6">
		<h2 class="mb-1 text-lg font-medium" dir="auto">
			{money(data.open.amount_minor, data.open.currency, locale)} to pay
		</h2>
		<p class="mb-4 text-sm text-ink-soft" dir="auto">
			Quote this reference so the office can find your payment:
			<span class="font-mono" dir="ltr">{data.open.reference}</span>
		</p>

		{#if data.open.instruction?.fields}
			<dl class="mb-4 grid gap-3 rounded-xl border border-line bg-raised p-4 text-sm sm:grid-cols-2">
				{#each Object.entries(data.open.instruction.fields) as [name, value] (name)}
					<div class="min-w-0">
						<dt class="text-ink-soft">{name.replace(/_/g, ' ')}</dt>
						<dd class="mt-0.5 truncate font-mono" dir="ltr">{value}</dd>
					</div>
				{/each}
			</dl>
		{/if}

		{#if data.open.status === 'awaiting_review'}
			<p class="banner mb-0 flex items-center gap-2.5 text-sm" dir="auto">
				<Receipt size={16} aria-hidden="true" />
				Sent. The office will confirm it, and you are enrolled as soon as they do.
			</p>
		{:else}
			<form method="POST" action="?/proof" use:enhance class="flex flex-col gap-4">
				<input type="hidden" name="order_id" value={data.open.id} />
				<div>
					<label class="mb-1.5 block text-sm font-medium" for="provider_ref">Transaction id</label>
					<input class="field font-mono" id="provider_ref" name="provider_ref" dir="ltr" />
				</div>
				<div>
					<label class="mb-1.5 block text-sm font-medium" for="note">
						Note
						<span class="font-normal text-ink-soft">· where and when you paid</span>
					</label>
					<input
						class="field"
						id="note"
						name="note"
						dir="auto"
						placeholder="paid at the Gulshan branch"
					/>
				</div>
				<div class="flex justify-end">
					<button class="btn" type="submit">Tell the office</button>
				</div>
			</form>
		{/if}
	</section>
{:else}
	<form method="POST" action="?/order" use:enhance class="card flex flex-col gap-4">
		<div>
			<label class="mb-1.5 block text-sm font-medium" for="provider">How you are paying</label>
			<select class="field" id="provider" name="provider">
				{#each data.providers as provider (provider.name)}
					<option value={provider.name}>{provider.label}</option>
				{/each}
			</select>
		</div>

		<div>
			<label class="mb-1.5 block text-sm font-medium" for="coupon">
				Discount code
				<span class="font-normal text-ink-soft">· if you have one</span>
			</label>
			<input class="field font-mono" id="coupon" name="coupon" dir="ltr" />
		</div>

		{#if parts && !plan}
			<label class="flex items-start gap-2.5 text-sm">
				<input class="choice mt-0.5" type="checkbox" name="in_parts" />
				<span>
					Pay in {parts} parts, about {money(perPart, course.currency, locale)} each, roughly
					{course.installment_gap_days} days apart. You are enrolled from the first payment.
				</span>
			</label>
		{/if}

		<div class="flex justify-end">
			<button class="btn" type="submit">
				{plan ? 'Pay the next part' : 'Continue'}
			</button>
		</div>
	</form>
{/if}
