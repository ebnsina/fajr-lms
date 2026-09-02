<script lang="ts">
	import Check from '@lucide/svelte/icons/check';

	// Priced in taka, the currency most of the first schools actually hold.
	const plans = [
		{
			name: 'Starter',
			price: 0,
			line: 'Up to 50 active learners',
			for: 'A teacher or a new coaching center finding its feet.',
			has: ['Every teaching feature', 'A public website', 'Bank transfer and wallet payments'],
			cta: 'Open a school',
			quiet: true
		},
		{
			name: 'School',
			price: 4500,
			line: 'Up to 600 active learners',
			for: 'A madrasah or school running a full timetable.',
			has: [
				'Everything in Starter',
				'Attendance with guardian alerts',
				'Certificates with verification',
				'SMS notifications'
			],
			cta: 'Open a school',
			quiet: false
		},
		{
			name: 'Institution',
			price: 12000,
			line: 'Unlimited learners',
			for: 'Several branches, or a corporate training arm.',
			has: [
				'Everything in School',
				'Your own media and transcoder',
				'Data residency in your country',
				'Priority support in Bangla, Arabic or English'
			],
			cta: 'Talk to us',
			quiet: true
		}
	];

	const money = new Intl.NumberFormat(undefined, {
		style: 'currency',
		currency: 'BDT',
		maximumFractionDigits: 0
	});

	const faq = [
		{
			q: 'What counts as an active learner?',
			a: 'Somebody enrolled in at least one course during the month. A learner who finished last term costs nothing.'
		},
		{
			q: 'How do we pay?',
			a: 'By bank transfer or bKash, monthly or for the year. We do not ask for a card.'
		},
		{
			q: 'What happens if we outgrow a plan?',
			a: 'Nothing stops. We tell you, and you move up when the term allows.'
		},
		{
			q: 'Can we take our data out?',
			a: 'Yes, at any time, in full. It is your school.'
		}
	];
</script>

<svelte:head>
	<title>Pricing · Fajr LMS</title>
	<meta
		name="description"
		content="Fajr LMS pricing: every teaching feature on every plan, priced by how many learners you teach."
	/>
</svelte:head>

<section class="px-6 py-16">
	<div class="mx-auto max-w-3xl text-center">
		<h1 class="text-4xl font-semibold">Priced by learners, not by features</h1>
		<p class="mt-3 text-ink-soft">
			Every plan teaches, grades and collects fees. What grows is how many people you teach.
		</p>
	</div>

	<div class="mx-auto mt-12 grid max-w-5xl gap-4 lg:grid-cols-3">
		{#each plans as plan (plan.name)}
			<article class="card flex flex-col" class:border-brand-line={!plan.quiet}>
				<h2 class="text-lg font-semibold">{plan.name}</h2>
				<p class="mt-2 mb-0 font-mono text-3xl">
					{plan.price === 0 ? 'Free' : money.format(plan.price)}
					{#if plan.price > 0}<span class="font-sans text-sm text-ink-soft">/month</span>{/if}
				</p>
				<p class="mt-1 mb-4 text-sm text-ink-soft">{plan.line}</p>
				<p class="mb-4 text-sm">{plan.for}</p>
				<ul class="mb-6 flex flex-1 flex-col gap-2">
					{#each plan.has as item (item)}
						<li class="flex items-start gap-2 text-sm">
							<Check class="mt-0.5 shrink-0 text-brand-text" size={16} aria-hidden="true" />
							{item}
						</li>
					{/each}
				</ul>
				<a class="btn" class:btn-quiet={plan.quiet} href="/start">{plan.cta}</a>
			</article>
		{/each}
	</div>
</section>

<section class="border-t border-line px-6 py-16">
	<div class="mx-auto max-w-3xl">
		<h2 class="mb-6 text-2xl font-semibold">Questions we are asked</h2>
		<div class="flex flex-col gap-3">
			{#each faq as item (item.q)}
				<div class="card">
					<h3 class="mb-1 font-medium">{item.q}</h3>
					<p class="mb-0 text-sm text-ink-soft">{item.a}</p>
				</div>
			{/each}
		</div>
	</div>
</section>
