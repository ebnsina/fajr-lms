<script lang="ts">
	import { enhance } from '$app/forms';
	import Video from '@lucide/svelte/icons/video';
	import ExternalLink from '@lucide/svelte/icons/external-link';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	let editing = $state<string | null>(null);

	const when = (iso: string) =>
		new Intl.DateTimeFormat(undefined, { dateStyle: 'full', timeStyle: 'short' }).format(
			new Date(iso)
		);
	const relative = (iso: string) => {
		const minutes = Math.round((new Date(iso).getTime() - Date.now()) / 60000);
		const rtf = new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' });
		if (Math.abs(minutes) < 60) return rtf.format(minutes, 'minute');
		if (Math.abs(minutes) < 60 * 24) return rtf.format(Math.round(minutes / 60), 'hour');
		return rtf.format(Math.round(minutes / 1440), 'day');
	};
</script>

<svelte:head><title>Classes · {data.course.title} · Fajr LMS</title></svelte:head>

<div class="mb-6">
	<a class="text-sm text-brand-text" href="/courses/{data.slug}">← Back to the course</a>
	<h1 class="mt-1 text-2xl font-semibold tracking-tight" dir="auto">Live classes</h1>
	<p class="mb-0 text-sm text-ink-soft">
		Meetings are held in Google Meet or Zoom for now; paste the link and everyone enrolled can join
		from here.
	</p>
</div>

{#if form?.message}
	<p class="banner-bad mb-4" role="alert">{form.message}</p>
{:else if form?.saved}
	<p class="banner mb-4" role="status">Saved.</p>
{:else if form?.joinURL}
	<p class="banner mb-4">
		The class is open:
		<a class="font-medium underline" href={form.joinURL} target="_blank" rel="noreferrer">
			open the meeting
		</a>
	</p>
{/if}

{#if data.sessions.length === 0}
	<div class="card text-ink-soft">
		<p class="mb-0">
			No classes are scheduled. Staff add them on the
			<a class="text-brand-text underline" href="/courses/{data.slug}/attendance">register</a>.
		</p>
	</div>
{:else}
	<div class="flex flex-col gap-3">
		{#each data.sessions as session (session.id)}
			<article class="card">
				<header class="flex flex-wrap items-start justify-between gap-3">
					<div>
						<h2 class="mb-1 text-lg font-medium" dir="auto">{session.title}</h2>
						<p class="mb-0 text-sm text-ink-soft">
							{when(session.starts_at)} · {relative(session.starts_at)}
							{#if session.location}· {session.location}{/if}
						</p>
					</div>
					<div class="flex flex-wrap items-center gap-2">
						{#if session.join_url}
							<form method="POST" action="?/join" use:enhance>
								<input type="hidden" name="session_id" value={session.id} />
								<button class="btn btn-sm" type="submit">
									<Video size={16} aria-hidden="true" /> Join
								</button>
							</form>
						{:else}
							<span class="chip">No link yet</span>
						{/if}
						{#if data.staff}
							<button
								class="btn btn-sm btn-quiet"
								type="button"
								onclick={() => (editing = editing === session.id ? null : session.id)}
							>
								{session.join_url ? 'Change the link' : 'Add the link'}
							</button>
						{/if}
						{#if session.recording_media_id}
							<span class="chip chip-brand">Recorded</span>
						{/if}
					</div>
				</header>

				{#if data.staff && editing === session.id}
					<div class="mt-4 grid gap-4 border-t border-line pt-4">
						<form class="grid gap-4 sm:grid-cols-2" method="POST" action="?/setLink" use:enhance>
							<input type="hidden" name="session_id" value={session.id} />
							<div>
								<label class="mb-1.5 block text-sm font-medium" for="join-{session.id}">
									Link for the learners
								</label>
								<input
									class="field font-mono"
									id="join-{session.id}"
									name="join_url"
									type="url"
									value={session.join_url}
									placeholder="https://meet.google.com/..."
									dir="ltr"
								/>
							</div>
							<div>
								<label class="mb-1.5 block text-sm font-medium" for="host-{session.id}">
									Host link <span class="font-normal text-ink-soft">· staff only</span>
								</label>
								<input
									class="field font-mono"
									id="host-{session.id}"
									name="host_url"
									type="url"
									value={session.host_url}
									dir="ltr"
								/>
							</div>
							<div class="flex justify-end sm:col-span-2">
								<button class="btn btn-sm" type="submit">Save the link</button>
							</div>
						</form>

						<form
							class="flex flex-wrap items-end gap-3 border-t border-line pt-4"
							method="POST"
							action="?/attachRecording"
							use:enhance
						>
							<input type="hidden" name="session_id" value={session.id} />
							<input type="hidden" name="title" value={session.title} />
							<div class="min-w-56 flex-1">
								<label class="mb-1.5 block text-sm font-medium" for="rec-{session.id}">
									Recording <span class="font-normal text-ink-soft">· paste the link after</span>
								</label>
								<input
									class="field font-mono"
									id="rec-{session.id}"
									name="recording"
									type="url"
									placeholder="https://youtu.be/..."
									dir="ltr"
								/>
							</div>
							<button class="btn btn-sm btn-quiet" type="submit">
								<ExternalLink size={16} aria-hidden="true" /> Attach it
							</button>
						</form>
					</div>
				{/if}
			</article>
		{/each}
	</div>
{/if}
