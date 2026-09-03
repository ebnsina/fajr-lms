<script lang="ts">
	import { onMount } from 'svelte';

	let { size = 28, label = '' }: { size?: number; label?: string } = $props();

	const uid = $props.id();

	// The hour the mark is named for, said as simply as it can be: a sky, and
	// the sun coming up out of the corner. Its own palette rather than the
	// product's emerald, because the mark stands on its own.
	const SKY = '#0f4c81';
	const HORIZON = '#57c7dd';
	const SUN = '#ffd27a';

	// Where the fā's dot would sit. Arabic writing puts it exactly where a
	// rising sun belongs, so the sun takes its place.
	const SUN_AT = { x: 29.09, y: 10.56, r: 1.72 };

	// فجر in Gulzar, shaped and joined, as an outline: a mark cannot depend on
	// the reader having an Arabic face installed. The word is drawn without the
	// dot above its fā', because the sun stands in for that dot.
	const WORD =
		'M17.11 35.00 17.06 34.71Q17.87 34.28 18.56 33.70Q19.24 33.11 19.70 32.60Q20.16 32.09 20.31 31.90Q20.79 31.30 21.27 30.51Q21.74 29.72 22.21 28.91Q22.68 28.10 23.13 27.43Q23.58 26.75 23.99 26.38Q24.25 26.29 24.55 26.55Q24.84 26.80 25.00 27.04Q24.45 27.40 24.06 27.81Q23.67 28.22 23.31 28.75Q22.95 29.28 22.55 29.97Q22.15 30.66 21.59 31.56Q21.37 31.92 20.98 32.44Q20.58 32.96 20.01 33.49Q19.44 34.03 18.72 34.44Q17.99 34.85 17.11 35.00ZM26.68 33.81Q26.62 33.81 26.54 33.80Q26.46 33.79 26.38 33.77Q25.70 33.59 25.36 33.20Q25.03 32.82 25.03 32.46Q25.03 32.15 25.19 31.91Q25.35 31.66 25.47 31.51L26.10 30.69H26.19Q26.44 31.13 26.80 31.40Q27.16 31.68 27.72 31.80Q28.03 31.85 28.03 32.05Q28.03 32.17 27.89 32.36L27.04 33.60Q26.96 33.72 26.88 33.76Q26.80 33.81 26.68 33.81ZM22.37 28.80Q22.85 27.83 23.22 27.26Q23.58 26.69 23.96 26.37Q24.33 26.06 24.81 25.87Q25.29 25.68 25.97 25.48Q26.29 25.37 26.76 25.23Q27.23 25.08 27.69 24.90Q28.15 24.73 28.46 24.52Q28.76 24.32 28.78 24.11Q28.30 23.96 27.60 23.89Q26.90 23.82 26.18 23.83Q25.46 23.84 24.88 23.94Q24.86 23.45 25.01 23.19Q25.17 22.94 25.42 22.82Q26.14 22.60 27.28 22.54Q28.42 22.48 29.56 22.61Q29.87 22.65 29.96 22.78Q30.06 22.92 30.03 23.07Q30.01 23.23 29.97 23.36Q29.92 23.53 29.75 23.85Q29.58 24.16 29.35 24.51Q29.12 24.86 28.89 25.18Q28.66 25.49 28.49 25.66Q27.81 26.34 27.08 26.67Q26.36 26.99 25.66 27.20Q25.00 27.40 24.43 27.68Q23.86 27.96 23.41 28.64ZM24.88 23.94Q24.76 23.87 24.76 23.69Q24.76 23.19 25.09 22.60Q25.42 22.00 26.10 21.45Q26.65 21.03 27.39 20.66Q28.13 20.30 29.07 19.84Q29.51 19.61 29.83 19.38Q30.14 19.14 30.14 18.92Q30.14 18.68 29.94 18.49Q29.72 18.69 29.44 18.77Q29.15 18.85 28.90 18.85Q28.39 18.85 28.03 18.53Q27.67 18.22 27.67 17.52L28.59 14.91Q28.69 14.83 28.92 14.72Q29.14 14.62 29.41 14.62Q29.94 14.62 30.28 15.01Q30.62 15.41 30.78 15.99Q30.94 16.58 30.94 17.20Q30.94 18.49 30.49 19.39Q30.04 20.30 29.36 20.76Q28.93 21.03 28.33 21.28Q27.72 21.54 27.18 21.81Q26.56 22.10 25.93 22.55Q25.29 23.01 24.88 23.94Z';

	// The squircle the orb draws in its shader, |x|^4 + |y|^4 = 1, fitted to four
	// cubics: each handle lands at 0.919 of the half width.
	const TILE =
		'M48 24C48 46.06 46.06 48 24 48C1.94 48 0 46.06 0 24C0 1.94 1.94 0 24 0C46.06 0 48 1.94 48 24Z';

	let canvas = $state<HTMLCanvasElement | null>(null);
	let live = $state(false);

	const VERT = `
		attribute vec2 a_pos;
		void main() { gl_Position = vec4(a_pos, 0.0, 1.0); }
	`;

	const FRAG = `
		precision mediump float;

		uniform vec2 u_resolution;
		uniform float u_time;
		uniform vec3 u_sky;
		uniform vec3 u_horizon;
		uniform vec3 u_sun;

		float hash(vec2 p) {
			return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453123);
		}

		float noise(vec2 p) {
			vec2 i = floor(p);
			vec2 f = fract(p);
			vec2 u = f * f * (3.0 - 2.0 * f);
			return mix(
				mix(hash(i + vec2(0.0, 0.0)), hash(i + vec2(1.0, 0.0)), u.x),
				mix(hash(i + vec2(0.0, 1.0)), hash(i + vec2(1.0, 1.0)), u.x),
				u.y
			);
		}

		float fbm(vec2 p) {
			float v = 0.0;
			float a = 0.6;
			for (int i = 0; i < 3; i++) {
				v += a * noise(p);
				p *= 2.0;
				a *= 0.5;
			}
			return v;
		}

		void main() {
			vec2 uv = gl_FragCoord.xy / u_resolution.xy;
			vec2 q = vec2(uv.x, 1.0 - uv.y) * 48.0;

			float t = u_time * 0.16;
			vec2 drift = vec2(
				sin(t) + 0.6 * sin(t * 1.7 + 1.3),
				cos(t * 0.8) + 0.6 * cos(t * 1.3 + 2.1)
			);
			vec2 p = vec2(uv.x * 1.6, uv.y) + drift * 0.5;
			vec2 g = vec2(fbm(p + drift), fbm(p + vec2(3.2, 1.5) - drift));
			float f = fbm(p + 1.2 * g);

			// Cooler overhead, warmer at the horizon, with the sand drifting
			// through both.
			vec3 band = mix(u_sky, u_horizon, smoothstep(0.95, 0.3, uv.y));
			vec3 sky = mix(band * 1.1, band * 0.82, smoothstep(0.3, 0.8, f));

			// No rim on the sun: a soft core bleeding into a wide bloom, which is
			// how light actually comes up over a ridge.
			float d = distance(q, vec2(25.0, 27.0));
			float core = smoothstep(13.5, 6.5, d);
			float bloom = exp(-max(d - 6.0, 0.0) * 0.13);
			float breathe = 0.9 + 0.1 * sin(u_time * 0.5);

			vec3 lit = mix(sky, u_sun, clamp(bloom * 0.7 * breathe, 0.0, 1.0));
			lit = mix(lit, u_sun, core * (0.82 + 0.18 * (1.0 - smoothstep(0.3, 0.8, f))));

			float aa = 48.0 / u_resolution.x * 1.2;
			float y = seg(q.x, A, B);
			y = mix(y, seg(q.x, B, C), step(B.x, q.x));
			y = mix(y, seg(q.x, C, D), step(C.x, q.x));
			y = mix(y, seg(q.x, D, E), step(D.x, q.x));
			float ridge = smoothstep(y - aa, y + aa, q.y);

			vec3 col = mix(lit, u_rock, ridge);

			vec2 c = (uv - 0.5) * 2.0;
			float squircle = pow(abs(c.x), 4.0) + pow(abs(c.y), 4.0);
			float edge = smoothstep(1.0, 0.9, squircle);

			gl_FragColor = vec4(col * edge, edge);
		}
	`;

	function rgb(hex: string): [number, number, number] {
		const n = parseInt(hex.replace('#', ''), 16);
		return [((n >> 16) & 255) / 255, ((n >> 8) & 255) / 255, (n & 255) / 255];
	}

	function compile(gl: WebGLRenderingContext, type: number, src: string) {
		const shader = gl.createShader(type);
		if (!shader) return null;
		gl.shaderSource(shader, src);
		gl.compileShader(shader);
		if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) {
			gl.deleteShader(shader);
			return null;
		}
		return shader;
	}

	onMount(() => {
		if (!canvas) return;
		const reduce = window.matchMedia('(prefers-reduced-motion: reduce)');
		if (reduce.matches) return;

		// The buffer is kept so a compositor frame the shader did not draw still
		// shows the mark.
		const context = canvas.getContext('webgl', {
			antialias: true,
			alpha: true,
			preserveDrawingBuffer: true
		});
		if (!context) return;
		const gl: WebGLRenderingContext = context;

		const program = gl.createProgram();
		const vert = compile(gl, gl.VERTEX_SHADER, VERT);
		const frag = compile(gl, gl.FRAGMENT_SHADER, FRAG);
		if (!program || !vert || !frag) return;

		gl.attachShader(program, vert);
		gl.attachShader(program, frag);
		gl.linkProgram(program);
		if (!gl.getProgramParameter(program, gl.LINK_STATUS)) return;
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
		const px = Math.round(size * dpr);
		canvas.width = px;
		canvas.height = px;
		gl.viewport(0, 0, px, px);
		gl.uniform2f(gl.getUniformLocation(program, 'u_resolution'), px, px);
		gl.uniform3f(gl.getUniformLocation(program, 'u_sky'), ...rgb(SKY));
		gl.uniform3f(gl.getUniformLocation(program, 'u_horizon'), ...rgb(HORIZON));
		gl.uniform3f(gl.getUniformLocation(program, 'u_sun'), ...rgb(SUN));
		const uTime = gl.getUniformLocation(program, 'u_time');

		const start = performance.now();
		let raf = 0;

		function render(now: number) {
			// A hidden tab has no frames, so the still mark under the canvas stands in.
			if (!document.hidden) {
				gl.uniform1f(uTime, (now - start) / 1000);
				gl.drawArrays(gl.TRIANGLES, 0, 6);
			}
			raf = requestAnimationFrame(render);
		}
		render(start);
		live = true;

		return () => {
			cancelAnimationFrame(raf);
			gl.deleteProgram(program);
			gl.deleteShader(vert);
			gl.deleteShader(frag);
			gl.deleteBuffer(buffer);
		};
	});
</script>

<div
	class="mark"
	style:inline-size="{size}px"
	style:block-size="{size}px"
	role={label ? 'img' : 'presentation'}
	aria-label={label || undefined}
	aria-hidden={label ? undefined : 'true'}
>
	<svg viewBox="0 0 48 48" fill="none" aria-hidden="true">
		<defs>
			<!-- Graded along the diagonal the sun comes up on. -->
			<linearGradient id="fajr-sky-{uid}" x1="0" y1="0" x2="1" y2="1">
				<stop offset="0" stop-color={SKY} />
				<stop offset="1" stop-color={HORIZON} />
			</linearGradient>
			<!-- The sun has no rim: a bloom that fades out into the sky. -->
			<radialGradient id="fajr-sun-{uid}" cx="0.5" cy="0.5" r="0.5">
				<stop offset="0" stop-color={SUN} stop-opacity="1" />
				<stop offset="0.34" stop-color={SUN} stop-opacity="0.95" />
				<stop offset="0.62" stop-color={SUN} stop-opacity="0.42" />
				<stop offset="1" stop-color={SUN} stop-opacity="0" />
			</radialGradient>
		</defs>
		<path d={TILE} fill="url(#fajr-sky-{uid})" />
		<clipPath id="fajr-tile-{uid}"><path d={TILE} /></clipPath>
		<g clip-path="url(#fajr-tile-{uid})">
			<circle cx={SUN_AT.x} cy={SUN_AT.y} r={SUN_AT.r * 2.4} fill="url(#fajr-sun-{uid})" />
		</g>
	</svg>
	<canvas bind:this={canvas} class:live></canvas>
	<!-- The word rides above the sky, whether the sky is drawn or painted. -->
	<svg class="letter" viewBox="0 0 48 48" fill="none" aria-hidden="true">
		<path
			d={WORD}
			fill="#fdf6e6"
			fill-opacity="0.97"
			stroke="#fdf6e6"
			stroke-opacity="0.97"
			stroke-width="0.55"
			stroke-linejoin="round"
			stroke-linecap="round"
		/>
	</svg>
</div>

<style>
	.mark {
		position: relative;
		max-inline-size: 100%;
	}

	svg,
	.letter,
	canvas {
		position: absolute;
		inset: 0;
		inline-size: 100%;
		block-size: 100%;
		display: block;
	}

	canvas {
		opacity: 0;
	}

	canvas.live {
		opacity: 1;
	}

	.letter {
		pointer-events: none;
	}
</style>
