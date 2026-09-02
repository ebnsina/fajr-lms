<script lang="ts">
	import Users from '@lucide/svelte/icons/users';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<svelte:head><title>Members · Fajr LMS</title></svelte:head>

<header class="mb-6">
	<h1 class="text-2xl font-semibold tracking-tight" dir="auto">Members</h1>
	<p class="mt-1 text-sm text-ink-soft" dir="auto">
		{data.total}
		{data.total === 1 ? 'person' : 'people'} in this school.
	</p>
</header>

{#if data.members.length === 0}
	<div class="card flex items-start gap-3 text-sm text-ink-soft" dir="auto">
		<Users class="mt-0.5 shrink-0" size={18} aria-hidden="true" />
		<p class="mb-0">Nobody here yet.</p>
	</div>
{:else}
	<div class="card overflow-x-auto p-0">
		<table class="w-full text-sm">
			<thead class="border-b border-line text-ink-soft">
				<tr>
					<th class="px-5 py-3 text-start font-medium">Name</th>
					<th class="px-5 py-3 text-start font-medium">Role</th>
					<th class="px-5 py-3 text-start font-medium">Reachable at</th>
				</tr>
			</thead>
			<tbody>
				{#each data.members as row (row.id)}
					<tr class="border-b border-line last:border-0">
						<td class="px-5 py-3 font-medium" dir="auto">{row.full_name}</td>
						<td class="px-5 py-3 text-ink-soft" dir="auto">{row.role}</td>
						<td class="px-5 py-3 font-mono text-ink-soft" dir="ltr">
							{row.phone ?? row.email ?? '—'}
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}
