<script lang="ts">
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const count = new Intl.NumberFormat('en');
	const money = new Intl.NumberFormat('en', { style: 'currency', currency: 'BDT' });

	const groups = $derived([
		{
			title: 'The business',
			tiles: [
				{ label: 'Schools', value: count.format(data.numbers.schools) },
				{ label: 'People', value: count.format(data.numbers.people) },
				{ label: 'Paid orders', value: count.format(data.numbers.paid_orders) },
				{ label: 'Collected', value: money.format(data.numbers.paid_minor / 100) }
			]
		},
		{
			title: 'The demo',
			tiles: [
				{ label: 'Leads', value: count.format(data.numbers.leads) },
				{ label: 'This week', value: count.format(data.numbers.leads_this_week) },
				{ label: 'Opened a school', value: count.format(data.numbers.leads_converted) },
				{ label: 'Marked won', value: count.format(data.numbers.leads_won) }
			]
		},
		{
			title: 'Teaching',
			tiles: [
				{ label: 'Courses', value: count.format(data.numbers.courses) },
				{ label: 'Enrollments', value: count.format(data.numbers.enrollments) },
				{ label: 'Certificates', value: count.format(data.numbers.certificates) },
				{ label: 'Demo schools', value: count.format(data.numbers.demo_schools) }
			]
		}
	]);
</script>

<svelte:head><title>Overview · Back office</title></svelte:head>

<h1 class="mb-6 text-2xl font-semibold tracking-tight">Overview</h1>

{#each groups as group (group.title)}
	<h2 class="mb-3 text-sm font-semibold tracking-wide uppercase text-ink-soft">{group.title}</h2>
	<div class="mb-8 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
		{#each group.tiles as tile (tile.label)}
			<div class="card">
				<p class="mb-1 text-sm text-ink-soft">{tile.label}</p>
				<p class="mb-0 font-mono text-2xl font-semibold">{tile.value}</p>
			</div>
		{/each}
	</div>
{/each}
