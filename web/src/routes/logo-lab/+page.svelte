<script lang="ts">
	// A scratch page for choosing the mark. Delete this route once one is picked.

	// One WebGL program draws every mark; a uniform picks the composition, so the
	// six variations share the same light, noise and drift.
	const VARIANTS = [
		{
			key: 'Dawn',
			id: 0,
			note: 'rising behind',
			say: 'The sun low behind the twin ridge, half of it still to come up. The most literal reading of fajr, and the warmest.'
		},
		{
			key: 'Horizon',
			id: 1,
			note: 'half risen',
			say: 'Cut by the horizon, mid-rise. Calmer and more geometric — it sits quietly beside a wordmark.'
		},
		{
			key: 'Aperture',
			id: 2,
			note: 'a ring of light',
			say: 'The sun as a ring rather than a disc. Holds its shape at 16 pixels better than anything else here.'
		},
		{
			key: 'Notch',
			id: 3,
			note: 'light through the gap',
			say: 'No disc at all: the light is what comes through the notch between the two peaks. The quietest, and the most confident.'
		},
		{
			key: 'Crescent',
			id: 4,
			note: 'a thin crescent',
			say: 'A crescent instead of a disc — the dawn prayer said without a mosque or a star on it.'
		},
		{
			key: 'Inverse',
			id: 5,
			note: 'dark paper',
			say: 'Ink tile, emerald sun, pale ridge. Reads as a seal rather than an app icon — good on a certificate.'
		}
	];

	const VERT = `
		attribute vec2 a_pos;
		void main() { gl_Position = vec4(a_pos, 0.0, 1.0); }
	`;

	const FRAG = `
		precision mediump float;
		uniform vec2 u_resolution;
		uniform float u_time;
		uniform float u_variant;
		uniform vec3 u_brand;
		uniform vec3 u_sun;

		const vec2 A = vec2(-6.0, 54.0);
		const vec2 B = vec2(16.0, 23.0);
		const vec2 C = vec2(24.0, 33.0);
		const vec2 D = vec2(32.0, 27.0);
		const vec2 E = vec2(54.0, 54.0);

		float hash(vec2 p) { return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453123); }

		float noise(vec2 p) {
			vec2 i = floor(p), f = fract(p);
			vec2 u = f * f * (3.0 - 2.0 * f);
			return mix(mix(hash(i), hash(i + vec2(1.0, 0.0)), u.x),
			           mix(hash(i + vec2(0.0, 1.0)), hash(i + vec2(1.0, 1.0)), u.x), u.y);
		}

		float fbm(vec2 p) {
			float v = 0.0, a = 0.6;
			for (int i = 0; i < 4; i++) { v += a * noise(p); p *= 2.0; a *= 0.5; }
			return v;
		}

		float seg(float x, vec2 p, vec2 q) { return p.y + (q.y - p.y) * (x - p.x) / (q.x - p.x); }

		void main() {
			vec2 uv = gl_FragCoord.xy / u_resolution.xy;
			vec2 q = vec2(uv.x, 1.0 - uv.y) * 48.0;
			float aa = 48.0 / u_resolution.x * 1.2;

			float t = u_time * 0.2;
			vec2 drift = vec2(sin(t) + 0.6 * sin(t * 1.7 + 1.3),
			                  cos(t * 0.8) + 0.6 * cos(t * 1.3 + 2.1));
			vec2 p = vec2(uv.x * 1.6, uv.y) * 1.4 + drift * 0.6;
			vec2 g = vec2(fbm(p + drift), fbm(p + vec2(3.2, 1.5) - drift));
			float f = fbm(p + 1.5 * g);

			vec3 white = u_sun;
			vec3 sky  = mix(u_brand * 1.08, u_brand * 0.72, smoothstep(0.26, 0.82, f));
			vec3 sun  = mix(white, mix(white, u_brand, 0.5), smoothstep(0.24, 0.8, f));
			vec3 rock = u_brand * 0.42;

			// The ridge, shared by every variation that has one.
			float y = seg(q.x, A, B);
			y = mix(y, seg(q.x, B, C), step(B.x, q.x));
			y = mix(y, seg(q.x, C, D), step(C.x, q.x));
			y = mix(y, seg(q.x, D, E), step(D.x, q.x));
			float ridge = smoothstep(y - aa, y + aa, q.y);

			float dist = distance(q, vec2(25.0, 18.5));
			float light = 0.0;
			int v = int(u_variant + 0.5);

			if (v == 0) {
				light = smoothstep(13.5 + aa, 13.5 - aa, distance(q, vec2(25.0, 27.0)));
			} else if (v == 1) {
				// Half risen: the disc, cut flat where it meets the horizon.
				light = smoothstep(14.5 + aa, 14.5 - aa, dist) * smoothstep(27.0 + aa, 27.0 - aa, q.y);
			} else if (v == 2) {
				// A ring: the disc with its middle taken back out.
				light = smoothstep(14.0 + aa, 14.0 - aa, dist) * smoothstep(8.4 - aa, 8.4 + aa, dist);
			} else if (v == 3) {
				// No disc: a soft glow that only shows through the notch.
				light = smoothstep(13.0, 1.0, distance(q, vec2(24.0, 29.0)));
			} else if (v == 4) {
				// A crescent: the disc, less an offset disc.
				float bite = smoothstep(12.2 + aa, 12.2 - aa, distance(q, vec2(29.5, 15.0)));
				light = max(smoothstep(13.5 + aa, 13.5 - aa, dist) - bite, 0.0);
			} else {
				light = smoothstep(10.0 + aa, 10.0 - aa, distance(q, vec2(25.0, 15.0)));
			}

			vec3 col;
			if (v == 5) {
				// Ink tile, emerald sun, pale ridge.
				vec3 ink = vec3(0.055, 0.075, 0.08);
				vec3 glow = mix(u_sun * 1.15, u_sun * 0.85, smoothstep(0.24, 0.8, f));
				col = mix(ink, glow, light);
				col = mix(col, mix(white, u_brand * 1.3, 0.25), ridge);
			} else if (v == 3) {
				col = mix(sky, sun, light * (1.0 - ridge));
				col = mix(col, rock, ridge);
			} else {
				col = mix(mix(sky, sun, light), rock, ridge);
			}

			vec2 c = (uv - 0.5) * 2.0;
			float squircle = pow(abs(c.x), 4.0) + pow(abs(c.y), 4.0);
			float edge = smoothstep(1.0, 0.9, squircle);
			gl_FragColor = vec4(col * edge, edge);
		}
	`;

	const TILE =
		'M48 24C48 46.06 46.06 48 24 48C1.94 48 0 46.06 0 24C0 1.94 1.94 0 24 0C46.06 0 48 1.94 48 24Z';
	const RIDGE = 'M-6 54 16 23 24 33 32 27 54 54 54 64 -6 64Z';

	// The still mark, which is what a favicon and a reduced-motion viewer get.
	// The sun in both colours, compared live: white for contrast, or the dawn
	// amber the product already uses for progress.
	const WHITE = '#ffffff';
	const AMBER = '#e8a33d';
	let amber = $state(false);
	const sunHex = $derived(amber ? AMBER : WHITE);

	function stillMark(variant: (typeof VARIANTS)[number], px: number) {
		const emerald = '#047857';
		const ink = '#0e1214';
		const tile = variant.id === 5 ? ink : emerald;
		const rock = variant.id === 5 ? '#dfeee7' : '#0b3f2e';
		const sunFill = variant.id === 5 ? (amber ? AMBER : '#12b981') : sunHex;

		let sun = '';
		if (variant.id === 0) sun = `<circle cx="25" cy="27" r="13.5" fill="${sunFill}"/>`;
		else if (variant.id === 1)
			sun = `<path d="M12 27a13 13 0 0 1 26 0z" fill="${sunFill}"/>`;
		else if (variant.id === 2)
			sun = `<path d="M25 4.5a14 14 0 1 1 0 28 14 14 0 0 1 0-28zm0 5.6a8.4 8.4 0 1 0 0 16.8 8.4 8.4 0 0 0 0-16.8z" fill="${sunFill}"/>`;
		else if (variant.id === 3)
			sun = `<circle cx="24" cy="29" r="9" fill="${sunFill}"/>`;
		else if (variant.id === 4)
			sun = `<path d="M25 5a13.5 13.5 0 1 0 0 27 13.5 13.5 0 0 0 8.6-3.1 12.2 12.2 0 1 1 0-20.8A13.5 13.5 0 0 0 25 5z" fill="${sunFill}"/>`;
		else sun = `<circle cx="25" cy="15" r="10" fill="${sunFill}"/>`;

		return `<svg width="${px}" height="${px}" viewBox="0 0 48 48" fill="none" aria-hidden="true">
			<path d="${TILE}" fill="${tile}"/>
			<clipPath id="clip-${variant.key}-${px}"><path d="${TILE}"/></clipPath>
			<g clip-path="url(#clip-${variant.key}-${px})">${sun}<path d="${RIDGE}" fill="${rock}"/></g>
		</svg>`;
	}

	function compile(gl: WebGLRenderingContext, type: number, src: string) {
		const shader = gl.createShader(type);
		if (!shader) return null;
		gl.shaderSource(shader, src);
		gl.compileShader(shader);
		if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) return null;
		return shader;
	}

	const drawn: { gl: WebGLRenderingContext; uSun: WebGLUniformLocation | null }[] = [];

	function rgb(hex: string): [number, number, number] {
		const n = parseInt(hex.replace('#', ''), 16);
		return [((n >> 16) & 255) / 255, ((n >> 8) & 255) / 255, (n & 255) / 255];
	}

	$effect(() => {
		const colour = rgb(sunHex);
		for (const one of drawn) {
			one.gl.uniform3f(one.uSun, ...colour);
		}
	});

	function animate(canvas: HTMLCanvasElement, variant: (typeof VARIANTS)[number]) {
		const context = canvas.getContext('webgl', { antialias: true, alpha: true });
		if (!context) return false;
		const gl: WebGLRenderingContext = context;

		const program = gl.createProgram();
		const vert = compile(gl, gl.VERTEX_SHADER, VERT);
		const frag = compile(gl, gl.FRAGMENT_SHADER, FRAG);
		if (!program || !vert || !frag) return false;
		gl.attachShader(program, vert);
		gl.attachShader(program, frag);
		gl.linkProgram(program);
		if (!gl.getProgramParameter(program, gl.LINK_STATUS)) return false;
		gl.useProgram(program);

		const buffer = gl.createBuffer();
		gl.bindBuffer(gl.ARRAY_BUFFER, buffer);
		gl.bufferData(
			gl.ARRAY_BUFFER,
			new Float32Array([-1, -1, 1, -1, -1, 1, -1, 1, 1, -1, 1, 1]),
			gl.STATIC_DRAW
		);
		const aPos = gl.getAttribLocation(program, 'a_pos');
		gl.enableVertexAttribArray(aPos);
		gl.vertexAttribPointer(aPos, 2, gl.FLOAT, false, 0, 0);

		const dpr = Math.min(window.devicePixelRatio || 1, 2);
		const px = Math.round(140 * dpr);
		canvas.width = px;
		canvas.height = px;
		canvas.style.width = '140px';
		canvas.style.height = '140px';
		gl.viewport(0, 0, px, px);
		gl.uniform2f(gl.getUniformLocation(program, 'u_resolution'), px, px);
		gl.uniform1f(gl.getUniformLocation(program, 'u_variant'), variant.id);
		gl.uniform3f(gl.getUniformLocation(program, 'u_brand'), 0.016, 0.47, 0.34);
		const uSun = gl.getUniformLocation(program, 'u_sun');
		gl.uniform3f(uSun, ...rgb(sunHex));
		drawn.push({ gl, uSun });
		const uTime = gl.getUniformLocation(program, 'u_time');

		const still = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
		const start = performance.now();
		function draw(now: number) {
			if (!document.hidden) {
				gl.uniform1f(uTime, still ? 4 : (now - start) / 1000);
				gl.drawArrays(gl.TRIANGLES, 0, 6);
			}
			if (!still) requestAnimationFrame(draw);
		}
		draw(start);
		return true;
	}

	// Each card's canvas starts drawing as soon as it is in the page.
	function live(canvas: HTMLCanvasElement, variant: (typeof VARIANTS)[number]) {
		animate(canvas, variant);
		return {};
	}
</script>

<svelte:head><title>Mark studio · Fajr LMS</title></svelte:head>

<div class="wrap">
	<header class="top">
		<p class="eyebrow">Fajr LMS · the mark</p>
		<h1>Six ways to draw the dawn</h1>
		<p class="lede">
			Same idea throughout: <b>the sun coming up behind a ridge</b>, in the squircle the fluid orb
			uses, moving the way the orb moves. What differs is the composition — how much sun, how the
			light and the rock are weighted, and what carries the motion. Each is live here; the small
			sizes beside it are the still version a favicon uses.
		</p>
	</header>

	<div class="switch">
		<button class="pick" class:on={!amber} type="button" onclick={() => (amber = false)}>
			White sun
		</button>
		<button class="pick" class:on={amber} type="button" onclick={() => (amber = true)}>
			Dawn amber
		</button>
		<span class="hint">
			Both are live. The mark has to stand alone, so judge it at 16 and 28 pixels as much as large.
		</span>
	</div>

	<div class="grid">
		{#each VARIANTS as variant (variant.key)}
			<article class="mark-card">
				<div class="stage">
					<span class="under">{@html stillMark(variant, 140)}</span>
					<canvas use:live={variant}></canvas>
				</div>
				<div class="sizes">
					<span class="chip">{@html stillMark(variant, 44)} 44px</span>
					<span class="chip">{@html stillMark(variant, 28)} 28px</span>
					<span class="chip">{@html stillMark(variant, 16)} 16px</span>
				</div>
				<div class="on-paper">
					{@html stillMark(variant, 26)}
					<span class="name">Fajr LMS</span>
					<span class="note">{variant.note}</span>
				</div>
				<div class="say">
					<h2><span class="key">{variant.key}</span></h2>
					<p>{variant.say}</p>
				</div>
			</article>
		{/each}
	</div>

	<section class="tail">
		<h3>Pick one</h3>
		<p>
			Reply with the name — <code>Dawn</code>, <code>Horizon</code>, <code>Aperture</code>,
			<code>Notch</code>, <code>Crescent</code> or <code>Inverse</code> — and I will put it in the
			sidebar, the tab icon and the home-screen icon. Mixing is fine too: "Aperture, but the ridge
			from Dawn".
		</p>

		<div class="q">
			<b>Two things worth deciding while you look</b>
			<span>
				Should the mark ever appear without the words "Fajr LMS" beside it — on a certificate, say,
				or a printed report? And should the sun stay white, or take the amber that the product
				already uses for progress?
			</span>
		</div>
	</section>
</div>


<style>

	:root {
		--ground: #0d1012;
		--raised: #14181b;
		--line: #232a2e;
		--line-strong: #333d43;
		--ink: #eef2f0;
		--ink-soft: #9aa6a1;
		--ink-faint: #6b7773;
		--brand: #047857;
		--brand-bright: #10b981;
		--paper: #f6f4ee;
		--paper-ink: #14171a;
		--sans: 'Cabin', system-ui, -apple-system, sans-serif;
		--mono: 'Geist Mono', ui-monospace, 'SF Mono', Menlo, monospace;
	}

	* { box-sizing: border-box; }

	body {
		margin: 0;
		background: var(--ground);
		color: var(--ink);
		font: 16px/1.6 var(--sans);
		-webkit-font-smoothing: antialiased;
	}

	.wrap { max-width: 1080px; margin: 0 auto; padding: 3.5rem 1.25rem 5rem; }

	header.top { margin-bottom: 3rem; }
	.eyebrow {
		font: 500 0.7rem/1 var(--mono);
		letter-spacing: 0.18em;
		text-transform: uppercase;
		color: var(--brand-bright);
		margin: 0 0 0.9rem;
	}
	h1 {
		font-size: clamp(2rem, 5vw, 2.9rem);
		font-weight: 700;
		letter-spacing: -0.025em;
		line-height: 1.1;
		margin: 0 0 0.75rem;
		text-wrap: balance;
	}
	.lede { max-width: 62ch; color: var(--ink-soft); margin: 0; }
	.lede b { color: var(--ink); font-weight: 600; }

	.switch { display: flex; flex-wrap: wrap; align-items: center; gap: 0.6rem; margin-bottom: 1.5rem; }
	.pick {
		font: 500 0.85rem/1 var(--sans);
		color: var(--ink-soft);
		background: var(--raised);
		border: 1px solid var(--line);
		border-radius: 0.6rem;
		padding: 0.55rem 0.9rem;
		cursor: pointer;
	}
	.pick.on { color: var(--ink); border-color: var(--brand); background: rgba(4, 120, 87, 0.16); }
	.switch .hint { font-size: 0.8rem; color: var(--ink-faint); margin-inline-start: 0.4rem; }

	.grid {
		display: grid;
		gap: 1.25rem;
		grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
	}

	.mark-card {
		border: 1px solid var(--line);
		border-radius: 1.25rem;
		background: var(--raised);
		overflow: hidden;
		display: flex;
		flex-direction: column;
	}

	.stage {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 2.25rem 1.5rem 1.75rem;
		background:
			radial-gradient(120% 90% at 50% 0%, rgba(4, 120, 87, 0.16), transparent 70%),
			var(--raised);
	}
	.stage { position: relative; }
	/* The still mark sits under the canvas, so the stage is never empty. */
	.stage .under { position: absolute; inset-block-start: 2.25rem; line-height: 0; }
	.stage canvas { display: block; position: relative; border-radius: 30%; }

	.sizes {
		display: flex;
		align-items: center;
		gap: 1.1rem;
		padding: 0 1.5rem 1.4rem;
		justify-content: center;
	}
	.sizes .chip {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font: 400 0.7rem/1 var(--mono);
		color: var(--ink-faint);
	}

	.on-paper {
		display: flex;
		align-items: center;
		gap: 0.7rem;
		padding: 0.85rem 1.5rem;
		background: var(--paper);
		color: var(--paper-ink);
		border-block: 1px solid var(--line);
	}
	.on-paper .name { font-weight: 600; font-size: 0.95rem; letter-spacing: -0.01em; }
	.on-paper .note { margin-inline-start: auto; font: 400 0.7rem/1 var(--mono); color: #6c7570; }

	.say { padding: 1.15rem 1.5rem 1.4rem; }
	.say h2 {
		margin: 0 0 0.35rem;
		font-size: 1.05rem;
		font-weight: 600;
		letter-spacing: -0.01em;
		display: flex;
		align-items: baseline;
		gap: 0.6rem;
	}
	.say h2 .key {
		font: 500 0.7rem/1 var(--mono);
		color: var(--brand-bright);
		border: 1px solid rgba(16, 185, 129, 0.35);
		border-radius: 0.4rem;
		padding: 0.2rem 0.4rem;
	}
	.say p { margin: 0; color: var(--ink-soft); font-size: 0.9rem; }

	.tail { margin-top: 3rem; border-top: 1px solid var(--line); padding-top: 1.75rem; }
	.tail h3 { margin: 0 0 0.5rem; font-size: 1rem; font-weight: 600; }
	.tail p { color: var(--ink-soft); margin: 0 0 0.75rem; max-width: 68ch; font-size: 0.92rem; }
	.tail code {
		font: 400 0.85rem/1.4 var(--mono);
		background: var(--raised);
		border: 1px solid var(--line);
		border-radius: 0.4rem;
		padding: 0.1rem 0.35rem;
	}
	.q { border-inline-start: 2px solid var(--brand); padding-inline-start: 0.9rem; margin: 1.1rem 0; }
	.q b { display: block; font-weight: 600; }
	.q span { color: var(--ink-soft); font-size: 0.9rem; }

	@media (prefers-reduced-motion: reduce) {
		.stage canvas { animation: none; }
	}
</style>
