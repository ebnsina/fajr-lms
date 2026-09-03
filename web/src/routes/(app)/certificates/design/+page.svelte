<script lang="ts">
	import { enhance } from '$app/forms';
	import { invalidateAll } from '$app/navigation';
	import { untrack } from 'svelte';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import Trash from '@lucide/svelte/icons/trash-2';
	import type { PageProps } from './$types';
	import type { CertField } from './+page.server';

	let { data, form }: PageProps = $props();

	// The words each token stands in for while the layout is being drawn.
	const samples: Record<string, string> = $derived({
		recipient: 'Fatima Rahman',
		course: 'Foundations of Arabic Grammar',
		issuer: data.school,
		date: '3 September 2026',
		serial: 'FJR-7HQE-8YDW-Q64T',
		grade: '92%'
	});
	const names: Record<string, string> = {
		recipient: "The learner's name",
		course: 'The course',
		issuer: 'The school',
		date: 'The date',
		serial: 'The serial',
		grade: 'The grade',
		text: 'A line of your own'
	};

	let fields = $state<CertField[]>(untrack(() => structuredClone(data.fields)));
	let picked = $state<number | null>(null);
	let paper = $state<HTMLDivElement | null>(null);
	let dragging: number | null = null;
	let uploading = $state(false);
	let problem = $state('');

	const text = (field: CertField) =>
		field.token === 'text' ? field.label || 'Your words here' : (samples[field.token] ?? '');

	function add(token: string) {
		fields = [
			...fields,
			{
				token,
				x: 50,
				y: 50,
				size: token === 'recipient' ? 3 : 1.2,
				align: 'center',
				bold: token === 'recipient',
				color: '',
				label: token === 'text' ? 'Certificate of completion' : ''
			}
		];
		picked = fields.length - 1;
	}

	// Dragging works in percentages of the paper, so a layout drawn on a laptop
	// prints the same from a phone.
	function grab(index: number, event: PointerEvent) {
		dragging = index;
		picked = index;
		(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId);
	}

	function move(event: PointerEvent) {
		if (dragging === null || !paper) return;
		const box = paper.getBoundingClientRect();
		const x = Math.min(Math.max(((event.clientX - box.left) / box.width) * 100, 0), 100);
		const y = Math.min(Math.max(((event.clientY - box.top) / box.height) * 100, 0), 100);
		fields = fields.map((field, at) =>
			at === dragging ? { ...field, x: Math.round(x * 10) / 10, y: Math.round(y * 10) / 10 } : field
		);
	}

	const drop = () => (dragging = null);

	// The same move from the keyboard: a field is a button, so it can be picked
	// and nudged without a pointer.
	function nudge(index: number, event: KeyboardEvent) {
		const step = event.shiftKey ? 0.2 : 1;
		const by: Record<string, [number, number]> = {
			ArrowLeft: [-step, 0],
			ArrowRight: [step, 0],
			ArrowUp: [0, -step],
			ArrowDown: [0, step]
		};
		const shift = by[event.key];
		if (!shift) return;
		event.preventDefault();
		picked = index;
		fields = fields.map((field, at) =>
			at === index
				? {
						...field,
						x: Math.round(Math.min(Math.max(field.x + shift[0], 0), 100) * 10) / 10,
						y: Math.round(Math.min(Math.max(field.y + shift[1], 0), 100) * 10) / 10
					}
				: field
		);
	}

	async function sendBackground(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		if (!file) return;
		uploading = true;
		problem = '';
		try {
			const body = new FormData();
			body.set('file', file);
			const response = await fetch('/certificates/design/background', { method: 'PUT', body });
			if (!response.ok) {
				const answer = await response.json().catch(() => null);
				problem = answer?.message ?? 'That image could not be used.';
				return;
			}
			await invalidateAll();
		} catch {
			problem = 'The image could not be sent.';
		} finally {
			uploading = false;
			input.value = '';
		}
	}

	async function removeBackground() {
		uploading = true;
		try {
			await fetch('/certificates/design/background', { method: 'DELETE' });
			await invalidateAll();
		} finally {
			uploading = false;
		}
	}
</script>

<svelte:head><title>Certificate design · Fajr LMS</title></svelte:head>

<nav class="mb-4 text-sm">
	<a
		class="inline-flex items-center gap-1.5 text-brand-text underline-offset-4 hover:underline"
		href="/certificates"
	>
		<ArrowLeft class="rtl:hidden" size={16} aria-hidden="true" />
		<ArrowRight class="hidden rtl:block" size={16} aria-hidden="true" />
		Certificates
	</a>
</nav>

<header class="mb-6">
	<h1 class="text-2xl font-semibold tracking-tight" dir="auto">Certificate design</h1>
	<p class="mt-1 text-sm text-ink-soft" dir="auto">
		Drag each line where you want it, or pick one and nudge it with the arrow keys. Leave this
		empty and certificates keep the design we ship.
	</p>
</header>

{#if form?.message || problem}
	<p class="banner-bad mb-5 text-sm" role="alert">{form?.message ?? problem}</p>
{:else if form?.saved}
	<p class="banner mb-5 text-sm" role="status">Saved. New certificates print this way.</p>
{/if}

<div class="mb-5 flex flex-wrap items-center gap-2">
	{#each Object.entries(names) as [token, name] (token)}
		<button class="btn btn-sm btn-quiet" type="button" onclick={() => add(token)}>+ {name}</button>
	{/each}
	<label class="btn btn-sm btn-quiet ms-auto" class:opacity-60={uploading}>
		{uploading ? 'Sending…' : data.hasBackground ? 'Replace the paper' : 'Add paper'}
		<input class="sr-only" type="file" accept="image/*" onchange={sendBackground} />
	</label>
	{#if data.hasBackground}
		<button class="btn btn-sm btn-quiet" type="button" onclick={removeBackground} disabled={uploading}>
			Remove the paper
		</button>
	{/if}
</div>

<!-- The paper, at the shape a certificate is printed on. -->
<div
	class="relative mb-5 w-full overflow-hidden rounded-3xl border border-line bg-paper"
	style="aspect-ratio: 297 / 210; {data.hasBackground
		? `background: url('/certificates/design/background') center / cover no-repeat`
		: ''}"
	bind:this={paper}
	role="application"
	aria-label="The certificate, with each line where you put it"
	onpointermove={move}
	onpointerup={drop}
	onpointerleave={drop}
>
	{#each fields as field, index (index)}
		<button
			class="absolute -translate-x-1/2 -translate-y-1/2 cursor-move px-2 py-1 text-ink"
			class:ring-2={picked === index}
			class:ring-brand={picked === index}
			style="inset-inline-start: {field.x}%; inset-block-start: {field.y}%;
			       font-size: {field.size}rem; text-align: {field.align};
			       {field.bold ? 'font-weight:700;' : ''} {field.color ? `color:${field.color};` : ''}"
			type="button"
			dir="auto"
			onpointerdown={(event) => grab(index, event)}
			onkeydown={(event) => nudge(index, event)}
		>
			{text(field)}
		</button>
	{/each}

	{#if fields.length === 0}
		<p class="absolute inset-0 flex items-center justify-center text-sm text-ink-faint">
			Add a line above, then drag it into place.
		</p>
	{/if}
</div>

{#if picked !== null && fields[picked]}
	{@const field = fields[picked]}
	<div class="card mb-5 flex flex-wrap items-end gap-3">
		<span class="min-w-32 text-sm font-medium">{names[field.token]}</span>

		{#if field.token === 'text'}
			<div class="min-w-48 flex-1">
				<label class="mb-1.5 block text-sm font-medium" for="field-label">Words</label>
				<input class="field" id="field-label" bind:value={field.label} dir="auto" />
			</div>
		{/if}

		<div class="w-28">
			<label class="mb-1.5 block text-sm font-medium" for="field-size">Size</label>
			<input
				class="field font-mono"
				id="field-size"
				type="number"
				min="0.5"
				max="12"
				step="0.1"
				bind:value={field.size}
				dir="ltr"
			/>
		</div>

		<div class="w-36">
			<label class="mb-1.5 block text-sm font-medium" for="field-align">Aligned</label>
			<select class="field" id="field-align" bind:value={field.align}>
				<option value="start">Left</option>
				<option value="center">Centre</option>
				<option value="end">Right</option>
			</select>
		</div>

		<div class="w-28">
			<label class="mb-1.5 block text-sm font-medium" for="field-color">Colour</label>
			<input class="field" id="field-color" type="color" bind:value={field.color} />
		</div>

		<label class="flex items-center gap-2 self-end pb-3 text-sm">
			<input class="choice" type="checkbox" bind:checked={field.bold} />
			Bold
		</label>

		<button
			class="btn btn-sm btn-quiet"
			type="button"
			onclick={() => {
				fields = fields.filter((_, at) => at !== picked);
				picked = null;
			}}
		>
			<Trash size={16} aria-hidden="true" />
			Remove
		</button>
	</div>
{/if}

<form method="POST" action="?/save" use:enhance class="flex justify-end gap-2">
	<input type="hidden" name="fields" value={JSON.stringify(fields)} />
	<button class="btn" type="submit">Save the design</button>
</form>
