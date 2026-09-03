<script lang="ts">
	import { enhance } from '$app/forms';
	import Users from '@lucide/svelte/icons/users';
	import UserPlus from '@lucide/svelte/icons/user-plus';
	import Trash from '@lucide/svelte/icons/trash-2';
	import Select from '$lib/components/Select.svelte';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	let inviting = $state(false);
	let role = $state('student');

	// One pending role and one form per row, so changing a role submits itself.
	let pending = $state<Record<string, string>>({});
	let forms = $state<Record<string, HTMLFormElement | null>>({});

	function changeRole(userID: string, next: string) {
		pending[userID] = next;
		queueMicrotask(() => forms[userID]?.requestSubmit());
	}

	// The roles a school actually hands out, in the order it hands them out.
	const roles = [
		{ value: 'student', label: 'Student', hint: 'Takes courses' },
		{ value: 'parent', label: 'Parent or guardian', hint: 'Sees a learner’s attendance and results' },
		{ value: 'instructor', label: 'Teacher', hint: 'Builds courses and grades work' },
		{ value: 'assistant', label: 'Assistant', hint: 'Helps with grading and the register' },
		{ value: 'admin', label: 'Admin', hint: 'Runs the school and its fees' },
		{ value: 'owner', label: 'Owner', hint: 'Everything, including other owners' }
	];
	const roleName = (value: string) => roles.find((r) => r.value === value)?.label ?? value;
</script>

<svelte:head><title>Members · Fajr LMS</title></svelte:head>

<header class="mb-6 flex flex-wrap items-start justify-between gap-3">
	<div>
		<h1 class="text-2xl font-semibold tracking-tight" dir="auto">Members</h1>
		<p class="mt-1 mb-0 text-sm text-ink-soft" dir="auto">
			{data.total}
			{data.total === 1 ? 'person' : 'people'} in this school.
		</p>
	</div>
	{#if data.canManage}
		<button class="btn btn-sm" type="button" onclick={() => (inviting = !inviting)}>
			<UserPlus size={16} aria-hidden="true" /> Invite somebody
		</button>
	{/if}
</header>

{#if form?.message}
	<p class="banner banner-bad mb-4" role="alert">{form.message}</p>
{:else if form?.invited}
	<p class="banner mb-4" role="status">
		{form.invited} is in the school. They sign in with the number or address you gave.
	</p>
{/if}

{#if inviting}
	<form class="card mb-6 grid gap-4 sm:grid-cols-3" method="POST" action="?/invite" use:enhance>
		<div>
			<label class="mb-1.5 block text-sm font-medium" for="full_name">Their name</label>
			<input class="field" id="full_name" name="full_name" dir="auto" required />
		</div>
		<div>
			<label class="mb-1.5 block text-sm font-medium" for="destination">
				Phone or email <span class="font-normal text-ink-soft">· how they sign in</span>
			</label>
			<input
				class="field font-mono"
				id="destination"
				name="destination"
				placeholder="+8801XXXXXXXXX"
				dir="ltr"
				required
			/>
		</div>
		<div>
			<span class="mb-1.5 block text-sm font-medium">Role</span>
			<input type="hidden" name="role" value={role} />
			<Select id="role" label="Role" bind:value={role} options={roles} />
		</div>
		<div class="flex justify-end sm:col-span-3">
			<button class="btn" type="submit">Add them</button>
		</div>
	</form>
{/if}

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
					{#if data.canManage}
						<th class="px-5 py-3 text-end font-medium">&nbsp;</th>
					{/if}
				</tr>
			</thead>
			<tbody>
				{#each data.members as row (row.id)}
					<tr class="border-b border-line last:border-0">
						<td class="px-5 py-3 font-medium" dir="auto">{row.full_name}</td>
						<td class="px-5 py-3 text-ink-soft" dir="auto">
							{#if data.canManage}
								<form
									class="w-52"
									method="POST"
									action="?/setRole"
									bind:this={forms[row.user_id]}
									use:enhance
								>
									<input type="hidden" name="user_id" value={row.user_id} />
									<input type="hidden" name="role" value={pending[row.user_id] ?? row.role} />
									<Select
										id="role-{row.user_id}"
										label="Role for {row.full_name}"
										value={pending[row.user_id] ?? row.role}
										options={roles}
										onchange={(next) => changeRole(row.user_id, next)}
									/>
								</form>
							{:else}
								{roleName(row.role)}
							{/if}
						</td>
						<td class="px-5 py-3 font-mono text-ink-soft" dir="ltr">
							{row.phone ?? row.email ?? '—'}
						</td>
						{#if data.canManage}
							<td class="px-5 py-3 text-end">
								<form method="POST" action="?/remove" use:enhance>
									<input type="hidden" name="user_id" value={row.user_id} />
									<button
										class="btn btn-sm btn-quiet"
										type="submit"
										aria-label="Remove {row.full_name} from this school"
									>
										<Trash size={16} aria-hidden="true" />
									</button>
								</form>
							</td>
						{/if}
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}
