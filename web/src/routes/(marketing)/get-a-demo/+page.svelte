<script lang="ts">
	import { enhance } from '$app/forms';
	import { untrack } from 'svelte';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	let runs = $state(untrack(() => form?.runs ?? ''));
	let learners = $state(untrack(() => form?.learners ?? '100-500'));
	let sealing = $state(false);

	// The label the API sends names a kind; the sentence needs it to read as one
	// clause, so the phrasing lives here.
	const phrases: Record<string, string> = {
		madrasah: 'a madrasah',
		school: 'a school or college',
		coaching: 'a coaching centre',
		creator: 'a teacher on my own',
		corporate: 'a company'
	};

	const sizes = [
		{ value: 'under-100', label: 'under a hundred' },
		{ value: '100-500', label: 'a hundred or so' },
		{ value: '500-2000', label: 'several hundred' },
		{ value: 'over-2000', label: 'a few thousand' }
	];

	// The letter folds, seals and goes; then the answer is applied and the demo
	// opens. Under reduced motion it simply goes.
	const still = () =>
		typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches;
</script>

<svelte:head>
	<title>Get a demo · Fajr LMS</title>
	<meta
		name="description"
		content="Tell us who you are in a sentence, and a Fajr LMS school with a term's work already in it opens straight away."
	/>
</svelte:head>

<section class="mx-auto max-w-4xl px-6 pt-40 pb-24">
	<h1 class="font-display text-4xl font-bold">Write us a line</h1>
	<p class="mt-2 mb-8 text-ink-soft">
		Fill in the blanks and a school opens with a term already behind it — courses, learners part
		way through, marks, the register taken, a certificate somebody earned.
	</p>

	{#if form?.message}
		<p class="banner-bad mb-6" role="alert">{form.message}</p>
	{/if}

	<form
		class="envelope"
		class:sealing
		method="POST"
		use:enhance={() => {
			sealing = true;
			return async ({ update }) => {
				if (!still()) await new Promise((done) => setTimeout(done, 1150));
				await update();
				sealing = false;
			};
		}}
	>
		<!-- The flap folds down over the letter when it is sent. -->
		<div class="flap" aria-hidden="true"></div>

		<div class="letter">
			<p class="line">
				<span>I am</span>
				<input
					class="blank"
					name="full_name"
					value={form?.full_name ?? ''}
					placeholder="your name"
					autocomplete="name"
					aria-label="Your name"
					dir="auto"
					required
				/><span>,</span>
				<span>and I look after</span>
				<input
					class="blank"
					name="role"
					value={form?.role ?? ''}
					placeholder="the office"
					autocomplete="organization-title"
					aria-label="Your part in it"
					dir="auto"
				/>
				<span>at</span>
				<input
					class="blank"
					name="organisation"
					value={form?.organisation ?? ''}
					placeholder="where you teach"
					autocomplete="organization"
					aria-label="Where you teach"
					dir="auto"
				/><span>.</span>
			</p>

			<p class="line">
				<span>We are</span>
				<select class="blank" name="runs" bind:value={runs} aria-label="What you run" required>
					<option value="" disabled>a madrasah…</option>
					{#each data.kinds as kind (kind.slug)}
						<option value={kind.slug}>{phrases[kind.slug] ?? kind.label}</option>
					{/each}
				</select>
				<span>teaching</span>
				<select class="blank" name="learners" bind:value={learners} aria-label="How many learners">
					{#each sizes as size (size.value)}
						<option value={size.value}>{size.label}</option>
					{/each}
				</select>
				<span>learners.</span>
			</p>

			<p class="line">
				<span>Write to me at</span>
				<input
					class="blank"
					name="email"
					type="email"
					value={form?.email ?? ''}
					placeholder="you@school"
					autocomplete="email"
					aria-label="Work email"
					dir="ltr"
					required
				/><span>,</span>
				<span>or ring</span>
				<input
					class="blank mono"
					name="phone"
					type="tel"
					value={form?.phone ?? ''}
					placeholder="+8801…"
					autocomplete="tel"
					aria-label="Phone, if you would rather we called"
					dir="ltr"
				/><span>.</span>
			</p>

			<p class="line">
				<span>Show me</span>
				<input
					class="blank wide"
					name="note"
					value={form?.note ?? ''}
					placeholder="what you would most like to see"
					aria-label="What you would like to see"
					dir="auto"
				/><span>.</span>
			</p>

			<div class="sign">
				<button class="btn" type="submit" disabled={sealing || data.kinds.length === 0}>
					{sealing ? 'Sealing…' : 'Seal it and send'}
				</button>
				<p class="mb-0 text-xs text-ink-faint">
					The demo school is shared and read-only, so nothing you click changes it.
				</p>
			</div>
		</div>

		{#if data.kinds.length === 0}
			<p class="mt-4 text-sm text-ink-soft">
				The demo is not answering just now. Try again in a moment, or
				<a class="underline" href="/start">open a school of your own</a>.
			</p>
		{/if}
	</form>
</section>

<style>
	/* A letter on the page: the writing is the form, and the flap folds over it
	   on the way out. */
	.envelope {
		perspective: 1200px;
		position: relative;
	}

	.letter {
		background: var(--color-surface);
		border: 1px solid var(--color-line);
		border-radius: var(--radius-card);
		padding: 2.5rem 2.75rem 1.75rem;
		box-shadow: 0 1px 2px rgb(0 0 0 / 0.04);
		transform-origin: 50% 0;
	}

	/* The flap is folded up and out of sight until the letter is sent. */
	.flap {
		position: absolute;
		inset-inline: 0;
		inset-block-start: 0;
		block-size: 3.5rem;
		border-start-start-radius: var(--radius-card);
		border-start-end-radius: var(--radius-card);
		background: linear-gradient(var(--color-brand-soft), var(--color-surface));
		border: 1px solid var(--color-brand-line);
		border-block-end: 0;
		transform-origin: 50% 0;
		transform: rotateX(-180deg);
		opacity: 0;
		z-index: 1;
	}

	.line {
		margin: 0 0 1.1rem;
		font-size: 1.25rem;
		line-height: 2.4;
		color: var(--color-ink);
	}

	.line span {
		color: var(--color-ink-soft);
	}

	/* Every blank is written on, not typed into: no box, just the rule under it. */
	.blank {
		font: inherit;
		color: var(--color-brand-text);
		font-weight: 500;
		background: transparent;
		border: 0;
		border-block-end: 2px solid var(--color-line-strong);
		border-radius: 0;
		padding: 0 0.15rem 0.15rem;
		inline-size: 12ch;
		min-inline-size: 8ch;
		max-inline-size: 100%;
		transition: border-color 0.15s;
	}

	.blank.wide {
		inline-size: 24ch;
	}

	/* Where the browser can size a field to what is written in it, the blank
	   grows with the answer instead of clipping it. */
	@supports (field-sizing: content) {
		.blank,
		.blank.wide {
			inline-size: auto;
			field-sizing: content;
		}
	}

	.blank.mono {
		font-family: var(--font-mono, ui-monospace, monospace);
	}

	.blank::placeholder {
		color: var(--color-ink-faint);
		font-weight: 400;
	}

	.blank:focus {
		outline: 0;
		border-block-end-color: var(--color-brand);
		background: var(--color-brand-soft);
	}

	select.blank {
		appearance: none;
		cursor: pointer;
		inline-size: auto;
		padding-inline-end: 0.4rem;
	}

	.sign {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		justify-content: space-between;
		gap: 0.75rem;
		margin-block-start: 1.5rem;
		padding-block-start: 1.25rem;
		border-block-start: 1px dashed var(--color-line-strong);
	}

	/* Sent: the flap comes down, then the whole thing goes. */
	.sealing .flap {
		animation: fold 0.45s ease-in forwards;
	}

	.sealing .letter {
		animation: post 0.7s 0.45s cubic-bezier(0.4, 0, 0.9, 0.4) forwards;
	}

	@keyframes fold {
		from {
			transform: rotateX(-180deg);
			opacity: 0;
		}
		10% {
			opacity: 1;
		}
		to {
			transform: rotateX(0deg);
			opacity: 1;
		}
	}

	@keyframes post {
		to {
			transform: translateY(-60vh) scale(0.7) rotate(-4deg);
			opacity: 0;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.sealing .flap,
		.sealing .letter {
			animation: none;
		}
	}
</style>
