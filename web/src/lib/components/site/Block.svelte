<script lang="ts">
	import type { SiteBlock, SiteCourse } from '$lib/types.site';

	let { block, courses = [], tenant }: { block: SiteBlock; courses?: SiteCourse[]; tenant: string } =
		$props();

	// Blank lines separate paragraphs. Nothing is ever rendered as markup.
	const paragraphs = $derived((block.text ?? '').split(/\n{2,}/).filter(Boolean));

	function price(course: SiteCourse): string {
		if (course.price_minor === 0) return 'Free';
		return new Intl.NumberFormat(undefined, {
			style: 'currency',
			currency: course.currency
		}).format(course.price_minor / 100);
	}
</script>

{#if block.type === 'hero'}
	<section class="border-b border-line bg-raised px-6 py-16 sm:py-24">
		<div class="mx-auto max-w-3xl text-center">
			<h1 class="text-4xl font-semibold sm:text-5xl" dir="auto">{block.heading}</h1>
			{#each paragraphs as line, i (i)}
				<p class="mt-4 text-lg text-ink-soft" dir="auto">{line}</p>
			{/each}
			{#if block.cta_label && block.cta_href}
				<a class="btn mt-8" href={block.cta_href}>{block.cta_label}</a>
			{/if}
		</div>
	</section>
{:else if block.type === 'richtext'}
	<section class="mx-auto max-w-3xl px-6 py-12">
		{#if block.heading}
			<h2 class="mb-4 text-2xl font-semibold" dir="auto">{block.heading}</h2>
		{/if}
		{#each paragraphs as line, i (i)}
			<p class="mb-4 text-ink-soft" dir="auto">{line}</p>
		{/each}
	</section>
{:else if block.type === 'features' || block.type === 'faq'}
	<section class="mx-auto max-w-5xl px-6 py-12">
		{#if block.heading}
			<h2 class="mb-6 text-2xl font-semibold" dir="auto">{block.heading}</h2>
		{/if}
		<div class="grid gap-4 {block.type === 'features' ? 'sm:grid-cols-2 lg:grid-cols-3' : ''}">
			{#each block.items ?? [] as item, i (i)}
				<div class="card">
					<h3 class="mb-1 font-medium" dir="auto">{item.title}</h3>
					{#if item.text}<p class="mb-0 text-sm text-ink-soft" dir="auto">{item.text}</p>{/if}
				</div>
			{/each}
		</div>
	</section>
{:else if block.type === 'stats'}
	<section class="border-y border-line bg-raised px-6 py-12">
		<div class="mx-auto max-w-5xl">
			{#if block.heading}
				<h2 class="mb-6 text-center text-2xl font-semibold" dir="auto">{block.heading}</h2>
			{/if}
			<dl class="grid gap-6 text-center sm:grid-cols-2 lg:grid-cols-4">
				{#each block.items ?? [] as item, i (i)}
					<div>
						<dt class="font-mono text-3xl font-medium tabular-nums" dir="auto">{item.title}</dt>
						{#if item.text}
							<dd class="mt-1 mb-0 text-sm text-ink-soft" dir="auto">{item.text}</dd>
						{/if}
					</div>
				{/each}
			</dl>
		</div>
	</section>
{:else if block.type === 'notices'}
	<section class="mx-auto max-w-3xl px-6 py-12">
		<h2 class="mb-4 text-2xl font-semibold" dir="auto">{block.heading || 'Notices'}</h2>
		<ul class="divide-y divide-line border-y border-line">
			{#each block.items ?? [] as item, i (i)}
				<li class="flex flex-wrap items-baseline gap-x-4 gap-y-1 py-3">
					<span class="flex-1 font-medium" dir="auto">{item.title}</span>
					{#if item.text}
						<span class="font-mono text-sm text-ink-soft" dir="auto">{item.text}</span>
					{/if}
				</li>
			{/each}
		</ul>
	</section>
{:else if block.type === 'courses'}
	<section class="mx-auto max-w-5xl px-6 py-12">
		{#if block.heading}
			<h2 class="mb-6 text-2xl font-semibold" dir="auto">{block.heading}</h2>
		{/if}
		{#if courses.length === 0}
			<p class="text-ink-soft">No courses are open for enrollment just yet.</p>
		{:else}
			<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
				{#each courses.slice(0, block.limit || 6) as course (course.id)}
					<article class="card flex flex-col">
						<h3 class="mb-1 font-medium" dir={course.dir}>{course.title}</h3>
						<p class="mb-4 flex-1 text-sm text-ink-soft" dir={course.dir}>{course.summary}</p>
						<div class="flex items-center justify-between">
							<span class="chip chip-brand font-mono">{price(course)}</span>
							<a class="btn btn-sm btn-quiet" href="/login?school={tenant}">Enroll</a>
						</div>
					</article>
				{/each}
			</div>
		{/if}
	</section>
{:else if block.type === 'cta'}
	<section class="mx-auto max-w-3xl px-6 py-12">
		<div class="card text-center">
			<h2 class="mb-2 text-2xl font-semibold" dir="auto">{block.heading}</h2>
			{#each paragraphs as line, i (i)}
				<p class="mb-4 text-ink-soft" dir="auto">{line}</p>
			{/each}
			{#if block.cta_label && block.cta_href}
				<a class="btn" href={block.cta_href}>{block.cta_label}</a>
			{/if}
		</div>
	</section>
{/if}
