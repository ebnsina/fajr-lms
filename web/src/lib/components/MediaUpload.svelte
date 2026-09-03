<script lang="ts">
	import Upload from '@lucide/svelte/icons/upload';
	import { invalidateAll } from '$app/navigation';

	let {
		lessonId,
		kind,
		title,
		endpoint
	}: { lessonId: string; kind: string; title: string; endpoint: string } = $props();

	let file = $state<File | null>(null);
	let percent = $state(0);
	let busy = $state(false);
	let problem = $state('');

	async function ask(body: unknown) {
		const response = await fetch(endpoint, {
			method: 'POST',
			headers: { 'content-type': 'application/json' },
			body: JSON.stringify(body)
		});
		if (!response.ok) {
			throw new Error((await response.text()) || 'The upload could not be prepared.');
		}
		return response.json();
	}

	// A plain XHR rather than fetch: it is the only way to report progress, and
	// somebody on a slow line needs to see that something is happening.
	function put(url: string, method: string, headers: Record<string, string>, blob: File) {
		return new Promise<void>((resolve, reject) => {
			const request = new XMLHttpRequest();
			request.open(method, url);
			for (const [name, value] of Object.entries(headers ?? {})) {
				request.setRequestHeader(name, value);
			}
			request.upload.onprogress = (event) => {
				if (event.lengthComputable) percent = Math.round((event.loaded / event.total) * 100);
			};
			request.onload = () =>
				request.status < 300
					? resolve()
					: reject(new Error(`Storage refused the file (${request.status}).`));
			request.onerror = () => reject(new Error('The connection dropped during the upload.'));
			request.send(blob);
		});
	}

	async function send() {
		if (!file) return;
		busy = true;
		problem = '';
		percent = 0;

		try {
			const prepared = await ask({
				step: 'prepare',
				filename: file.name,
				content_type: file.type || 'application/octet-stream',
				byte_size: file.size,
				title,
				kind
			});
			await put(
				prepared.upload.url,
				prepared.upload.method ?? 'PUT',
				prepared.upload.headers ?? {},
				file
			);
			await ask({ step: 'finish', media_id: prepared.id, lesson_id: lessonId });
			file = null;
			await invalidateAll();
		} catch (cause) {
			problem = cause instanceof Error ? cause.message : 'The upload failed.';
		} finally {
			busy = false;
		}
	}

	const size = (bytes: number) =>
		new Intl.NumberFormat(undefined, {
			style: 'unit',
			unit: 'megabyte',
			maximumFractionDigits: 1
		}).format(bytes / 1_000_000);
</script>

<div class="flex flex-wrap items-center gap-3">
	<label class="btn btn-sm btn-quiet cursor-pointer">
		<Upload size={16} aria-hidden="true" />
		{file ? 'Choose another' : 'Choose a file'}
		<input
			class="sr-only"
			type="file"
			accept={kind === 'audio' ? 'audio/*' : kind === 'pdf' ? 'application/pdf' : 'video/*'}
			onchange={(event) => (file = event.currentTarget.files?.[0] ?? null)}
		/>
	</label>

	{#if file}
		<span class="text-sm text-ink-soft" dir="auto">{file.name} · {size(file.size)}</span>
		<button class="btn btn-sm" type="button" onclick={send} disabled={busy}>
			{busy ? `Uploading ${percent}%` : 'Upload it'}
		</button>
	{/if}
</div>

{#if busy}
	<div class="mt-3 h-1.5 overflow-hidden rounded-full bg-sunken">
		<div class="h-full rounded-full bg-brand transition-[width]" style:width="{percent}%"></div>
	</div>
{/if}

{#if problem}
	<p class="banner banner-bad mt-3 text-sm" role="alert">{problem}</p>
{/if}
