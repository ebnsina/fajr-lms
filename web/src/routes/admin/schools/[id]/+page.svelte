<script lang="ts">
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const school = $derived(data.school);
	const day = new Intl.DateTimeFormat('en', { dateStyle: 'medium' });
	const money = (minor: number, currency: string) =>
		new Intl.NumberFormat('en', { style: 'currency', currency }).format(minor / 100);
	const awaiting = $derived(school.orders.filter((order) => order.status !== 'paid').length);
</script>

<svelte:head><title>{school.tenant.name} · Back office</title></svelte:head>

<nav class="mb-4 text-sm">
	<a class="inline-flex items-center gap-1.5 text-brand-text underline-offset-4 hover:underline" href="/admin/schools">
		<ArrowLeft size={16} aria-hidden="true" /> Schools
	</a>
</nav>

<header class="mb-6">
	<h1 class="text-2xl font-semibold tracking-tight" dir="auto">{school.tenant.name}</h1>
	<p class="mt-1 mb-0 text-sm text-ink-soft">
		<span class="font-mono">/{school.tenant.slug}</span>
		· {school.tenant.kind} · {school.tenant.status} · opened
		{day.format(new Date(school.tenant.created_at))}
		{#if school.tenant.demo}· <span class="chip">demo</span>{/if}
	</p>
</header>

<div class="mb-8 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
	<div class="card"><p class="mb-1 text-sm text-ink-soft">Members</p><p class="mb-0 font-mono text-2xl font-semibold">{school.members.length}</p></div>
	<div class="card"><p class="mb-1 text-sm text-ink-soft">Courses</p><p class="mb-0 font-mono text-2xl font-semibold">{school.courses.length}</p></div>
	<div class="card"><p class="mb-1 text-sm text-ink-soft">Certificates</p><p class="mb-0 font-mono text-2xl font-semibold">{school.certificates}</p></div>
	<div class="card"><p class="mb-1 text-sm text-ink-soft">Orders waiting</p><p class="mb-0 font-mono text-2xl font-semibold">{awaiting}</p></div>
</div>

<section class="card mb-5">
	<h2 class="mb-3 text-sm font-semibold tracking-wide uppercase text-ink-soft">Who is in it</h2>
	{#if school.members.length === 0}
		<p class="mb-0 text-sm text-ink-soft">Nobody yet.</p>
	{:else}
		<ul class="flex list-none flex-col gap-2 p-0 text-sm">
			{#each school.members as member (member.user_id)}
				<li class="flex flex-wrap items-baseline gap-x-3">
					<span class="font-medium" dir="auto">{member.full_name}</span>
					<span class="chip">{member.role}</span>
					<span class="text-ink-soft" dir="ltr">{member.contact}</span>
					<span class="ms-auto text-xs text-ink-faint">{day.format(new Date(member.since))}</span>
				</li>
			{/each}
		</ul>
	{/if}
</section>

<section class="card mb-5">
	<h2 class="mb-3 text-sm font-semibold tracking-wide uppercase text-ink-soft">What it teaches</h2>
	{#if school.courses.length === 0}
		<p class="mb-0 text-sm text-ink-soft">No courses yet.</p>
	{:else}
		<ul class="flex list-none flex-col gap-2 p-0 text-sm">
			{#each school.courses as course (course.title)}
				<li class="flex flex-wrap items-baseline gap-x-3">
					<span dir="auto">{course.title}</span>
					<span class="chip">{course.status}</span>
					<span class="ms-auto font-mono text-ink-soft">{course.learners} enrolled</span>
				</li>
			{/each}
		</ul>
	{/if}
</section>

<section class="card">
	<h2 class="mb-3 text-sm font-semibold tracking-wide uppercase text-ink-soft">Money</h2>
	{#if school.orders.length === 0}
		<p class="mb-0 text-sm text-ink-soft">Nothing has been paid here.</p>
	{:else}
		<ul class="flex list-none flex-col gap-2 p-0 text-sm">
			{#each school.orders.slice(0, 25) as order (order.reference)}
				<li class="flex flex-wrap items-baseline gap-x-3">
					<span class="font-mono">{order.reference}</span>
					<span class="chip">{order.status}</span>
					<span class="text-ink-soft">{order.provider}</span>
					<span class="ms-auto font-mono">{money(order.amount_minor, order.currency)}</span>
					<span class="text-xs text-ink-faint">{day.format(new Date(order.created_at))}</span>
				</li>
			{/each}
		</ul>
	{/if}
</section>
