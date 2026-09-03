<script lang="ts">
	import Users from '@lucide/svelte/icons/users';
	import ProgressBar from '$lib/components/ProgressBar.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	let locale = $derived(data.session?.tenant?.locale ?? 'en');

	const day = (iso: string) =>
		new Intl.DateTimeFormat(locale, { dateStyle: 'medium' }).format(new Date(iso));
	const count = (n: number) => new Intl.NumberFormat(locale).format(n);

	const kinds: Record<string, string> = {
		sabaq: 'New lesson',
		sabqi: 'Recent revision',
		manzil: 'Older revision'
	};
</script>

<svelte:head><title>Your family · Fajr LMS</title></svelte:head>

<header class="mb-6">
	<h1 class="text-2xl font-semibold tracking-tight" dir="auto">Your family</h1>
	<p class="mt-1 text-sm text-ink-soft" dir="auto">
		What the school records for the children you are guardian of.
	</p>
</header>

{#if data.children.length === 0}
	<div class="card flex items-start gap-3 text-sm text-ink-soft" dir="auto">
		<Users class="mt-0.5 shrink-0" size={18} aria-hidden="true" />
		<p class="mb-0">
			Nobody is linked to your account yet. The school office adds that, and it appears here.
		</p>
	</div>
{:else}
	<div class="flex flex-col gap-4">
		{#each data.children as child (child.student_id)}
			<article class="card">
				<header class="mb-4 flex flex-wrap items-center gap-3">
					<div class="min-w-0 flex-1">
						<h2 class="mb-0.5 text-lg font-medium" dir="auto">{child.full_name}</h2>
						<p class="mb-0 text-sm text-ink-soft" dir="auto">
							{child.relation || 'guardian'}{child.class_name
								? ` · ${child.class_name}${child.section_name ? ` ${child.section_name}` : ''}`
								: ''}
						</p>
					</div>
				</header>

				{#if child.hifz && child.hifz.lessons > 0}
					{@const percent = Math.round((child.hifz.ayahs_memorised / child.hifz.total_ayahs) * 100)}
					<div class="mb-4">
						<div class="mb-1.5 flex flex-wrap items-baseline justify-between gap-2 text-sm">
							<span class="font-medium">Hifz</span>
							<span class="font-mono text-ink-soft tabular-nums">
								{count(child.hifz.ayahs_memorised)} of {count(child.hifz.total_ayahs)} ayahs
							</span>
						</div>
						<ProgressBar {percent} label="Ayahs memorised" />
					</div>

					<ul class="list-none space-y-2 p-0">
						{#each child.hifz.entries as row (row.hifz_entry.id)}
							{@const entry = row.hifz_entry}
							<li class="flex flex-wrap items-center gap-3 rounded-xl border border-line bg-raised px-3.5 py-2.5 text-sm">
								<span class="text-ink-soft">{day(entry.on_date)}</span>
								<span class="chip">{kinds[entry.kind] ?? entry.kind}</span>
								<span class="min-w-0 flex-1" dir="auto">
									{row.from_name}
									{entry.from_ayah}
									–
									{row.from_name === row.to_name ? '' : row.to_name + ' '}{entry.to_ayah}
								</span>
								<span class="text-ink-soft">{entry.quality}</span>
								{#if entry.mistakes > 0}
									<span class="font-mono text-ink-faint tabular-nums">
										{entry.mistakes}
										{entry.mistakes === 1 ? 'slip' : 'slips'}
									</span>
								{/if}
							</li>
						{/each}
					</ul>
				{:else}
					<p class="mb-0 text-sm text-ink-soft" dir="auto">
						Nothing recorded for {child.full_name} yet.
					</p>
				{/if}
			</article>
		{/each}
	</div>
{/if}
