<script lang="ts">
	// A scratch page for consolidating the word and the sun. Delete this route
	// once one is picked.
	const SKY = '#0f4c81';
	const HORIZON = '#57c7dd';
	const SUN = '#ffd27a';
	const INK = '#fdf6e6';

	const TILE =
		'M48 24C48 46.06 46.06 48 24 48C1.94 48 0 46.06 0 24C0 1.94 1.94 0 24 0C46.06 0 48 1.94 48 24Z';

	// فجر in Gulzar, shaped and joined, with the sun placed differently in each.
	const MARKS = [
		{ label: "Bled off the edge", note: "The word at three times the tile, cropped by it \u2014 the stroke runs off two sides.", d: "M2.91 62.00 2.77 61.20Q5.04 60.02 6.93 58.39Q8.81 56.76 10.09 55.35Q11.36 53.93 11.79 53.41Q13.11 51.76 14.43 49.57Q15.75 47.37 17.05 45.13Q18.34 42.89 19.59 41.03Q20.84 39.16 21.98 38.13Q22.68 37.89 23.51 38.60Q24.34 39.31 24.76 39.97Q23.25 40.96 22.17 42.09Q21.08 43.22 20.09 44.68Q19.10 46.15 17.99 48.06Q16.88 49.97 15.32 52.47Q14.71 53.46 13.63 54.90Q12.54 56.34 10.96 57.82Q9.38 59.31 7.37 60.44Q5.37 61.58 2.91 62.00ZM29.43 58.70Q29.24 58.70 29.03 58.67Q28.82 58.65 28.58 58.60Q26.69 58.08 25.77 57.02Q24.85 55.96 24.85 54.97Q24.85 54.12 25.30 53.44Q25.75 52.75 26.08 52.33L27.83 50.06H28.06Q28.77 51.29 29.76 52.04Q30.75 52.80 32.31 53.13Q33.16 53.27 33.16 53.84Q33.16 54.17 32.78 54.69L30.42 58.13Q30.19 58.46 29.97 58.58Q29.76 58.70 29.43 58.70ZM17.49 44.83Q18.82 42.14 19.83 40.56Q20.84 38.98 21.88 38.10Q22.92 37.23 24.24 36.71Q25.56 36.19 27.45 35.63Q28.35 35.34 29.64 34.94Q30.94 34.54 32.21 34.04Q33.49 33.55 34.34 32.98Q35.19 32.42 35.23 31.85Q33.91 31.43 31.98 31.24Q30.04 31.05 28.04 31.07Q26.03 31.10 24.43 31.38Q24.38 30.01 24.81 29.30Q25.23 28.60 25.94 28.26Q27.92 27.65 31.08 27.49Q34.24 27.32 37.40 27.70Q38.25 27.79 38.51 28.17Q38.77 28.55 38.70 28.97Q38.63 29.40 38.54 29.77Q38.40 30.25 37.92 31.12Q37.45 31.99 36.82 32.96Q36.18 33.93 35.54 34.80Q34.90 35.67 34.43 36.14Q32.55 38.03 30.54 38.93Q28.53 39.82 26.60 40.39Q24.76 40.96 23.18 41.74Q21.60 42.51 20.37 44.40ZM24.43 31.38Q24.10 31.19 24.10 30.67Q24.10 29.30 25.02 27.65Q25.94 26.00 27.83 24.49Q29.34 23.31 31.39 22.30Q33.44 21.28 36.04 20.01Q37.26 19.39 38.14 18.73Q39.01 18.07 39.01 17.46Q39.01 16.80 38.44 16.28Q37.83 16.85 37.05 17.06Q36.27 17.27 35.56 17.27Q34.15 17.27 33.16 16.40Q32.17 15.53 32.17 13.59L34.72 6.37Q35.00 6.14 35.61 5.85Q36.23 5.57 36.98 5.57Q38.44 5.57 39.39 6.66Q40.33 7.74 40.78 9.37Q41.23 11.00 41.23 12.69Q41.23 16.28 39.98 18.78Q38.73 21.28 36.84 22.56Q35.66 23.31 33.98 24.02Q32.31 24.73 30.80 25.48Q29.10 26.28 27.33 27.53Q25.56 28.78 24.43 31.38Z", sun: { cx: 34.0, cy: 12.0, r: 2.6 }, bloom: 2.6 },
		{ label: "One gesture", note: "Not the word: the single sweeping stroke out of it, cropped, with the sun above.", d: "M4.20 48.00 3.94 46.54Q8.07 44.38 11.51 41.41Q14.96 38.44 17.28 35.86Q19.61 33.28 20.38 32.33Q22.79 29.32 25.21 25.31Q27.62 21.31 29.98 17.22Q32.35 13.13 34.63 9.73Q36.92 6.32 38.98 4.43Q40.27 4.00 41.78 5.29Q43.29 6.58 44.06 7.79Q41.31 9.60 39.33 11.66Q37.35 13.73 35.54 16.40Q33.73 19.07 31.71 22.56Q29.68 26.04 26.84 30.61Q25.72 32.41 23.74 35.04Q21.76 37.67 18.88 40.38Q15.99 43.09 12.33 45.16Q8.67 47.23 4.20 48.00Z", sun: { cx: 33.0, cy: 13.0, r: 2.8 }, bloom: 2.8 },
		{ label: "Drawn long", note: "Kashida: the joins stretched into one long line, the way a calligrapher fills a space.", d: "M13.99 25.12V28.32H12.37Q11.22 28.32 10.71 28.10Q10.77 29.43 10.36 30.56Q9.96 31.69 9.14 32.53Q8.33 33.38 7.17 33.83Q6.01 34.28 4.62 34.28V36.00H4.53Q4.32 35.37 3.81 34.88Q3.30 34.40 2.39 34.46V31.30Q5.32 31.21 7.00 30.18Q8.69 29.10 8.96 27.05Q8.48 26.60 8.22 25.77Q7.97 24.94 8.03 24.28Q8.03 23.98 8.09 23.43L8.60 22.95Q9.11 24.13 9.85 24.62Q10.59 25.12 11.70 25.12ZM18.21 28.32H13.48Q12.82 28.32 12.35 27.85Q11.88 27.38 11.88 26.72Q11.88 26.05 12.35 25.59Q12.82 25.12 13.48 25.12H18.21ZM23.10 31.63Q22.82 32.02 22.46 32.47Q22.10 32.93 21.53 33.47Q20.29 32.23 19.60 31.84Q20.87 30.39 21.20 29.97Q21.71 30.24 22.27 30.73Q22.82 31.21 23.10 31.63ZM34.97 25.12V28.32H33.04Q31.56 28.32 30.52 27.95Q29.48 27.59 28.97 26.39Q28.67 25.75 28.67 25.00Q28.67 24.58 28.76 24.13Q27.95 24.40 27.38 24.82L26.80 25.24Q25.69 26.18 24.45 26.90Q22.01 28.32 19.51 28.32H17.70Q17.04 28.32 16.57 27.85Q16.10 27.38 16.10 26.75Q16.10 26.08 16.57 25.60Q17.04 25.12 17.70 25.12H19.45Q21.14 25.12 22.64 24.67Q23.46 24.43 24.23 24.02Q24.99 23.61 25.12 23.55L25.96 23.13L21.62 22.41L21.38 22.38Q20.93 22.38 20.75 22.74V23.49L20.41 23.64Q20.08 23.10 19.68 22.83Q19.27 22.56 18.97 22.48Q18.67 22.41 18.58 22.41Q18.64 22.26 18.86 21.65Q19.09 21.05 19.45 20.33Q19.63 20.03 20.49 19.76Q21.35 19.48 21.95 19.48Q22.55 19.48 23.04 19.61Q23.52 19.73 24.36 20.00Q24.57 20.06 25.08 20.24Q25.60 20.42 26.11 20.54Q27.38 20.87 28.42 21.01Q29.45 21.14 31.02 21.20L32.32 21.23L30.87 23.92Q30.57 23.85 30.39 23.85Q30.21 23.85 29.85 23.92Q30.09 24.52 30.78 24.82Q31.47 25.12 33.07 25.12ZM39.19 28.32H34.46Q33.79 28.32 33.33 27.85Q32.86 27.38 32.86 26.72Q32.86 26.05 33.33 25.59Q33.79 25.12 34.46 25.12H39.19ZM44.76 25.99Q44.79 26.24 44.79 26.72Q44.79 27.29 44.72 27.74Q44.64 28.19 44.61 28.32Q44.52 27.41 44.16 26.81Q43.50 27.56 42.41 27.94Q41.33 28.32 40.39 28.32H38.68Q38.01 28.32 37.55 27.85Q37.08 27.38 37.08 26.72Q37.08 26.05 37.55 25.59Q38.01 25.12 38.68 25.12H40.82Q41.45 25.09 41.96 25.05Q42.47 25.00 42.65 24.97Q42.17 24.34 41.09 23.92Q40.58 23.70 39.91 23.60Q39.25 23.49 39.04 23.46V21.32Q39.04 20.00 39.84 19.26Q40.64 18.52 41.93 18.52Q43.56 18.52 44.52 19.55Q45.61 20.69 45.61 22.71Q45.61 24.52 44.76 25.99ZM44.40 23.13Q44.40 22.35 43.95 21.81Q43.44 21.20 42.53 21.20Q41.66 21.20 40.91 21.81Q42.05 21.99 42.96 22.68Q43.83 23.37 44.22 24.19Q44.40 23.82 44.40 23.13Z", sun: { cx: 42.14, cy: 15.75, r: 2.2 }, bloom: 2.6 },
		{ label: "The bowl, low", note: "The stroke as a horizon across the bottom, with the sun coming up out of it.", d: "M10.50 50.00 10.32 49.00Q13.14 47.53 15.49 45.51Q17.84 43.48 19.42 41.72Q21.01 39.96 21.53 39.32Q23.18 37.26 24.82 34.53Q26.47 31.80 28.08 29.01Q29.69 26.22 31.25 23.90Q32.81 21.59 34.22 20.29Q35.10 20.00 36.12 20.88Q37.15 21.76 37.68 22.58Q35.80 23.82 34.45 25.23Q33.10 26.63 31.87 28.45Q30.63 30.27 29.25 32.65Q27.87 35.03 25.94 38.14Q25.17 39.37 23.82 41.16Q22.47 42.95 20.51 44.80Q18.54 46.65 16.05 48.06Q13.55 49.47 10.50 50.00Z", sun: { cx: 26.0, cy: 20.0, r: 5.5 }, bloom: 2.2 }
	];
</script>

<svelte:head><title>Word and sun · Fajr LMS</title></svelte:head>

{#snippet tile(mark: (typeof MARKS)[number], px: number, id: string)}
	<svg width={px} height={px} viewBox="0 0 48 48" fill="none" aria-hidden="true">
		<defs>
			<linearGradient id="sky-{id}" x1="0" y1="0" x2="1" y2="1">
				<stop offset="0" stop-color={SKY} />
				<stop offset="1" stop-color={HORIZON} />
			</linearGradient>
			<radialGradient id="sun-{id}" cx="0.5" cy="0.5" r="0.5">
				<stop offset="0" stop-color={SUN} stop-opacity="1" />
				<stop offset="0.34" stop-color={SUN} stop-opacity="0.95" />
				<stop offset="0.62" stop-color={SUN} stop-opacity="0.42" />
				<stop offset="1" stop-color={SUN} stop-opacity="0" />
			</radialGradient>
			<clipPath id="clip-{id}"><path d={TILE} /></clipPath>
		</defs>
		<path d={TILE} fill="url(#sky-{id})" />
		<g clip-path="url(#clip-{id})">
			<circle
				cx={mark.sun.cx}
				cy={mark.sun.cy}
				r={mark.sun.r * mark.bloom}
				fill="url(#sun-{id})"
			/>
			<path d={mark.d} fill={INK} fill-opacity="0.97" />
		</g>
	</svg>
{/snippet}

<div class="wrap">
	<header>
		<p class="eyebrow">Fajr LMS · the word and the light</p>
		<h1>Composed, not typed</h1>
		<p class="lede">
			The word treated as a drawing rather than a setting: blown up and cropped by the tile, reduced
			to the one stroke that carries it, or stretched along its joins the way a calligrapher fills a
			space. The sun stands in for the dot throughout.
		</p>
	</header>

	<div class="stack">
		{#each MARKS as mark, index (mark.label)}
			<article class="card">
				<div class="head">
					<h2>{mark.label}</h2>
					<span class="note">{mark.note}</span>
				</div>
				<div class="big">{@render tile(mark, 240, `a-${index}`)}</div>
				<div class="small">
					{@render tile(mark, 64, `b-${index}`)}
					{@render tile(mark, 44, `c-${index}`)}
					{@render tile(mark, 28, `d-${index}`)}
					{@render tile(mark, 16, `e-${index}`)}
				</div>
			</article>
		{/each}
	</div>
</div>

<style>
	:global(body) {
		margin: 0;
		background: #0d1012;
		color: #eef2f0;
		font: 16px/1.6 'Cabin', system-ui, sans-serif;
	}

	.wrap { max-width: 940px; margin: 0 auto; padding: 3.5rem 1.25rem 5rem; }
	.eyebrow {
		font: 500 0.7rem/1 'Geist Mono', ui-monospace, monospace;
		letter-spacing: 0.18em;
		text-transform: uppercase;
		color: #57c7dd;
		margin: 0 0 0.9rem;
	}
	h1 { font-size: clamp(1.9rem, 5vw, 2.6rem); letter-spacing: -0.025em; margin: 0 0 0.7rem; }
	.lede { max-width: 64ch; color: #9aa6a1; margin: 0 0 2.5rem; }

	.stack { display: flex; flex-direction: column; gap: 1.25rem; }
	.card {
		border: 1px solid #232a2e;
		border-radius: 1.25rem;
		background: #14181b;
		padding: 1.5rem;
		display: grid;
		gap: 1.5rem;
		grid-template-columns: auto 1fr;
		align-items: center;
	}
	.head { grid-column: 1 / -1; }
	.head h2 { margin: 0 0 0.2rem; font-size: 1.05rem; font-weight: 600; }
	.note { color: #6b7773; font-size: 0.87rem; }
	.small { display: flex; align-items: flex-end; gap: 1.5rem; }

	@media (max-width: 720px) {
		.card { grid-template-columns: 1fr; }
		.small { flex-wrap: wrap; }
	}
</style>
