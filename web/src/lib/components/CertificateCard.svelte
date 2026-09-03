<script lang="ts">
	type Certificate = {
		serial: string;
		recipient_name: string;
		course_title: string;
		issuer_name: string;
		grade_percent: number | null;
		issued_at: string;
		revoked_at: string | null;
	};

	let { certificate }: { certificate: Certificate } = $props();

	let issued = $derived(
		new Intl.DateTimeFormat(undefined, { dateStyle: 'long' }).format(
			new Date(certificate.issued_at)
		)
	);
	let grade = $derived(
		certificate.grade_percent === null
			? null
			: new Intl.NumberFormat(undefined, { style: 'percent' }).format(
					certificate.grade_percent / 100
				)
	);
</script>

<!-- The public page at full size, shrunk to a card: same paper, same rules. -->
<article class="sheet" class:void={certificate.revoked_at}>
	{#if certificate.revoked_at}
		<span class="stamp" aria-hidden="true">REVOKED</span>
	{/if}

	<svg
		class="crest"
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="1.5"
		stroke-linecap="round"
		stroke-linejoin="round"
		aria-hidden="true"
	>
		<circle cx="12" cy="8" r="6" />
		<path d="M8.2 13.4 7 22l5-3 5 3-1.2-8.6" />
	</svg>

	<p class="label">
		{certificate.revoked_at ? 'Revoked certificate' : 'Certificate of completion'}
	</p>
	<p class="issuer" dir="auto">{certificate.issuer_name}</p>

	<p class="label mt">This is to certify that</p>
	<p class="name" dir="auto">{certificate.recipient_name}</p>
	<hr class="rule" />
	<p class="label">has completed the course</p>
	<p class="course" dir="auto">{certificate.course_title}</p>
	{#if grade}
		<p class="grade">Final grade {grade}</p>
	{/if}

	<dl class="meta">
		<div>
			<dt>Issued on</dt>
			<dd>{issued}</dd>
		</div>
		<div>
			<dt>Serial</dt>
			<dd class="serial">{certificate.serial}</dd>
		</div>
	</dl>
</article>

<style>
	/* A certificate is a printed thing, so it stays paper in both themes. The
	   values are the product's own light tokens, kept in step with them by hand. */
	.sheet {
		--paper: #fffdf8;
		--ink: #14171a;
		--ink-soft: #676f7b;
		--ink-faint: #9aa1ac;
		--brand-text: #046b4e;
		--brand-soft: #e7f6f0;
		--brand-line: #bfe6d6;
		--danger: #a03528;
		--danger-line: #f0d3ce;

		position: relative;
		overflow: hidden;
		background: var(--paper);
		color: var(--ink);
		border: 1px solid var(--color-line-strong);
		border-radius: var(--radius-card);
		padding: 1.75rem 1.5rem;
		text-align: center;
	}

	.sheet::before {
		content: '';
		position: absolute;
		inset: 0.5rem;
		border: 1px solid var(--brand-line);
		border-radius: 1rem;
		pointer-events: none;
	}

	.sheet > * {
		position: relative;
	}

	.crest {
		inline-size: 1.75rem;
		block-size: 1.75rem;
		margin-inline: auto;
		color: var(--brand-text);
	}

	.label {
		margin: 0.5rem 0 0;
		font-size: 0.625rem;
		font-weight: 600;
		letter-spacing: 0.14em;
		text-transform: uppercase;
		color: var(--ink-soft);
	}

	.issuer {
		margin: 0.2rem 0 0;
		font-size: 0.8125rem;
		font-weight: 600;
	}

	.mt {
		margin-top: 1.1rem;
	}

	.name {
		margin: 0.35rem 0 0.6rem;
		font-size: 1.5rem;
		font-weight: 700;
		letter-spacing: -0.02em;
		line-height: 1.3;
	}

	.rule {
		inline-size: 4rem;
		block-size: 1px;
		margin: 0 auto 0.9rem;
		border: 0;
		background: var(--brand-line);
	}

	.course {
		margin: 0.25rem 0 0;
		font-size: 1rem;
		font-weight: 600;
	}

	.grade {
		display: inline-block;
		margin: 0.75rem 0 0;
		padding: 0.15rem 0.6rem;
		border: 1px solid var(--brand-line);
		border-radius: var(--radius-control);
		background: var(--brand-soft);
		color: var(--brand-text);
		font-family: var(--font-mono);
		font-size: 0.75rem;
	}

	.meta {
		display: flex;
		flex-wrap: wrap;
		justify-content: center;
		gap: 0.75rem 1.5rem;
		margin: 1.5rem 0 0;
		padding-top: 1rem;
		border-top: 1px solid var(--brand-line);
	}

	.meta dt {
		margin: 0 0 0.15rem;
		font-size: 0.5625rem;
		font-weight: 600;
		letter-spacing: 0.12em;
		text-transform: uppercase;
		color: var(--ink-faint);
	}

	.meta dd {
		margin: 0;
		font-size: 0.8125rem;
	}

	.serial {
		font-family: var(--font-mono);
		letter-spacing: 0.04em;
	}

	.void::before {
		border-color: var(--danger-line);
	}

	.void .name,
	.void .course {
		color: var(--ink-soft);
	}

	.stamp {
		position: absolute;
		inset-block-start: 50%;
		inset-inline-start: 50%;
		transform: translate(-50%, -50%) rotate(-18deg);
		font-size: 2.5rem;
		font-weight: 700;
		letter-spacing: 0.15em;
		color: var(--danger);
		opacity: 0.14;
		white-space: nowrap;
		pointer-events: none;
	}
</style>
