<script lang="ts">
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const count = new Intl.NumberFormat('en');
	const day = new Intl.DateTimeFormat('en', { dateStyle: 'medium' });
	const ago = new Intl.RelativeTimeFormat('en', { numeric: 'auto' });

	// Days is close enough for a list you scan; the school's own page has dates.
	function since(when: string) {
		const days = Math.round((Date.now() - new Date(when).getTime()) / 86_400_000);
		return days < 1 ? 'today' : ago.format(-days, 'day');
	}
</script>

<svelte:head><title>Schools · Back office</title></svelte:head>

<h1 class="mb-6 text-2xl font-semibold tracking-tight">Schools</h1>

<form class="card mb-5 flex flex-wrap items-end gap-3" method="GET">
	<div class="min-w-48 flex-1">
		<label class="mb-1.5 block text-sm font-medium" for="q">Search</label>
		<input class="field" id="q" name="q" value={data.query} placeholder="name or address" />
	</div>
	<button class="btn btn-sm" type="submit">Filter</button>
</form>

<div class="card overflow-x-auto">
	<table class="w-full min-w-3xl border-collapse text-sm">
		<thead>
			<tr class="border-b border-line text-start text-ink-soft">
				<th class="py-2 pe-3 text-start font-medium">School</th>
				<th class="py-2 pe-3 text-start font-medium">Opened</th>
				<th class="py-2 pe-3 text-end font-medium">Members</th>
				<th class="py-2 pe-3 text-end font-medium">Courses</th>
				<th class="py-2 pe-3 text-end font-medium">Learners</th>
				<th class="py-2 pe-3 text-end font-medium">Certificates</th>
				<th class="py-2 text-end font-medium">Last seen</th>
			</tr>
		</thead>
		<tbody>
			{#each data.schools as school (school.id)}
				<tr class="border-b border-line last:border-0">
					<td class="py-2.5 pe-3">
						<a class="font-medium text-brand-text underline-offset-4 hover:underline" href="/admin/schools/{school.id}" dir="auto">
							{school.name}
						</a>
						<span class="ms-2 font-mono text-xs text-ink-faint">/{school.slug}</span>
						{#if school.demo}<span class="chip ms-2">demo</span>{/if}
						{#if school.status !== 'active'}<span class="chip ms-2">{school.status}</span>{/if}
					</td>
					<td class="py-2.5 pe-3 text-ink-soft">{day.format(new Date(school.created_at))}</td>
					<td class="py-2.5 pe-3 text-end font-mono">{count.format(school.members)}</td>
					<td class="py-2.5 pe-3 text-end font-mono">{count.format(school.courses)}</td>
					<td class="py-2.5 pe-3 text-end font-mono">{count.format(school.learners)}</td>
					<td class="py-2.5 pe-3 text-end font-mono">{count.format(school.certificates)}</td>
					<td class="py-2.5 text-end text-ink-soft">{since(school.last_activity)}</td>
				</tr>
			{/each}
		</tbody>
	</table>

	{#if data.schools.length === 0}
		<p class="mb-0 py-4 text-sm text-ink-soft">No school matches that.</p>
	{/if}
</div>
