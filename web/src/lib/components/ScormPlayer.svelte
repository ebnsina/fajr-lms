<script lang="ts">
	import { onMount, untrack } from 'svelte';

	// SCORM 1.2. The package looks for window.API by walking up from its own
	// frame, so the adapter has to live here, in the parent, and the frame has
	// to be same-origin. That is why the files are served through this app.
	let {
		base,
		entry,
		progress,
		onSaved
	}: {
		base: string;
		entry: string;
		progress: {
			lesson_status: string;
			suspend_data: string;
			location: string;
			total_time_s: number;
			score_raw: string | number | null;
		};
		onSaved?: (status: string) => void;
	} = $props();

	// The starting point only; from then on the package says what the status is.
	let status = $state(untrack(() => progress.lesson_status));
	let problem = $state('');
	let saving = $state(false);
	// The frame is not created until the adapter is in place: a package looks
	// for window.API as it loads, and finding nothing it gives up for good.
	let ready = $state(false);

	const NO_ERROR = '0';
	const NOT_INITIALISED = '301';
	const NOT_FOUND = '401';

	onMount(() => {
		const data: Record<string, string> = {
			'cmi.core.lesson_status': progress.lesson_status === 'not attempted' ? 'not attempted' : progress.lesson_status,
			'cmi.core.lesson_location': progress.location,
			'cmi.suspend_data': progress.suspend_data,
			'cmi.core.score.raw': progress.score_raw == null ? '' : String(progress.score_raw),
			'cmi.core.credit': 'credit',
			'cmi.core.entry': progress.lesson_status === 'not attempted' ? 'ab-initio' : 'resume',
			'cmi.core.total_time': seconds(progress.total_time_s)
		};

		let started = false;
		let error = NO_ERROR;
		const openedAt = Date.now();

		function seconds(total: number): string {
			const hours = Math.floor(total / 3600);
			const minutes = Math.floor((total % 3600) / 60);
			const rest = total % 60;
			const pad = (n: number) => String(n).padStart(2, '0');
			return `${pad(hours)}:${pad(minutes)}:${pad(rest)}.00`;
		}

		// A package sends time as HHHH:MM:SS.SS; only the seconds matter here.
		function toSeconds(value: string): number {
			const parts = value.split(':').map((part) => Number.parseFloat(part));
			if (parts.length !== 3 || parts.some((part) => Number.isNaN(part))) return 0;
			return Math.round(parts[0] * 3600 + parts[1] * 60 + parts[2]);
		}

		async function send() {
			saving = true;
			const raw = Number.parseFloat(data['cmi.core.score.raw']);
			try {
				const response = await fetch(`${base}/state`, {
					method: 'POST',
					headers: { 'content-type': 'application/json' },
					body: JSON.stringify({
						lesson_status: data['cmi.core.lesson_status'] || 'incomplete',
						score_raw: Number.isFinite(raw) ? raw : null,
						suspend_data: data['cmi.suspend_data'] ?? '',
						location: data['cmi.core.lesson_location'] ?? '',
						// The package's own reckoning if it keeps one, ours otherwise.
						total_time_s:
							toSeconds(data['cmi.core.total_time'] ?? '') +
							(data['cmi.core.session_time']
								? toSeconds(data['cmi.core.session_time'])
								: Math.round((Date.now() - openedAt) / 1000)),
						cmi: data
					})
				});
				if (!response.ok) {
					problem = 'What you did was not saved. Check your connection and try again.';
					return false;
				}
				problem = '';
				status = data['cmi.core.lesson_status'];
				onSaved?.(status);
				return true;
			} catch {
				problem = 'What you did was not saved. Check your connection and try again.';
				return false;
			} finally {
				saving = false;
			}
		}

		const api = {
			LMSInitialize() {
				started = true;
				error = NO_ERROR;
				return 'true';
			},
			LMSFinish() {
				if (!started) {
					error = NOT_INITIALISED;
					return 'false';
				}
				started = false;
				error = NO_ERROR;
				send();
				return 'true';
			},
			LMSGetValue(name: string) {
				if (!started) {
					error = NOT_INITIALISED;
					return '';
				}
				if (!(name in data)) {
					error = NOT_FOUND;
					return '';
				}
				error = NO_ERROR;
				return data[name];
			},
			LMSSetValue(name: string, value: string) {
				if (!started) {
					error = NOT_INITIALISED;
					return 'false';
				}
				data[name] = String(value ?? '');
				error = NO_ERROR;
				return 'true';
			},
			LMSCommit() {
				if (!started) {
					error = NOT_INITIALISED;
					return 'false';
				}
				error = NO_ERROR;
				send();
				return 'true';
			},
			LMSGetLastError: () => error,
			LMSGetErrorString: (code: string) =>
				code === NO_ERROR ? 'No error' : 'The package asked for something unavailable',
			LMSGetDiagnostic: (code: string) => code
		};

		(window as unknown as { API: typeof api }).API = api;
		ready = true;

		// A learner closing the tab mid-package should not lose the attempt.
		const leaving = () => {
			if (started) send();
		};
		window.addEventListener('pagehide', leaving);
		return () => {
			window.removeEventListener('pagehide', leaving);
			delete (window as unknown as { API?: unknown }).API;
		};
	});
</script>

<div class="flex flex-col gap-3">
	<div class="flex flex-wrap items-center gap-3 text-sm">
		<span class="chip" class:chip-brand={status === 'passed' || status === 'completed'}>
			{status}
		</span>
		{#if saving}
			<span class="text-ink-soft" aria-live="polite">saving…</span>
		{/if}
		{#if problem}
			<span class="banner-bad px-3 py-1.5" role="alert">{problem}</span>
		{/if}
	</div>

	{#if ready}
		<iframe
			class="aspect-video w-full rounded-xl border border-line bg-raised"
			src="{base}/files/{entry}"
			title="Course package"
			allow="fullscreen"
		></iframe>
	{:else}
		<div class="aspect-video w-full rounded-xl border border-line bg-raised"></div>
	{/if}
</div>
