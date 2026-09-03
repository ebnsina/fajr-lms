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

	// The sun rises from the bottom right, off the tile's diagonal, so the mark
	// is never a circle sitting squarely in a square.
	const SUN_AT = { x: 37, y: 42, r: 15 };

	// Fā', the first letter of فجر, drawn as an outline taken from Aref Ruqaa
	// (SIL OFL) rather than set as text: a mark cannot depend on the reader
	// having an Arabic face installed.
	const FA =
		'M27.82 7.72 24.90 5.00Q24.07 6.03 23.21 7.10Q22.34 8.17 21.52 9.24L24.44 11.96Q25.31 10.89 26.15 9.84Q27.00 8.79 27.82 7.72ZM8.31 33.61Q9.67 34.51 11.72 34.82Q13.78 35.13 16.17 34.90Q18.56 34.68 20.90 33.96Q23.25 33.24 25.25 32.11Q27.24 30.97 28.52 29.45Q29.92 27.80 30.82 25.89Q31.73 23.98 31.87 21.98Q32.02 19.98 31.07 18.17Q30.58 17.14 29.82 16.38Q29.05 15.62 28.27 15.48Q27.49 15.33 26.79 16.28Q25.97 17.43 25.33 18.75Q24.69 20.07 23.99 21.34Q23.70 21.92 23.72 22.76Q23.74 23.61 24.32 23.89Q25.31 24.39 26.36 24.06Q27.41 23.73 28.07 23.19Q28.23 22.99 28.46 23.36Q28.68 23.73 28.89 24.30Q29.10 24.88 29.24 25.33Q29.38 25.79 29.38 25.79Q28.77 26.40 27.28 27.10Q25.80 27.80 23.83 28.38Q21.85 28.96 19.67 29.35Q17.49 29.74 15.43 29.76Q13.37 29.78 11.72 29.35Q10.08 28.92 9.15 27.89Q8.23 26.86 8.39 25.09Q8.51 23.44 9.13 22.12Q9.75 20.81 10.37 19.69Q10.04 19.78 9.87 20.02Q9.71 20.27 9.54 20.52Q7.20 24.10 6.50 26.67Q5.80 29.24 6.35 30.95Q6.91 32.66 8.31 33.61Z';

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
				<stop offset="0.4" stop-color={SUN} stop-opacity="0.94" />
				<stop offset="0.7" stop-color={SUN} stop-opacity="0.34" />
				<stop offset="1" stop-color={SUN} stop-opacity="0" />
			</radialGradient>
		</defs>
		<path d={TILE} fill="url(#fajr-sky-{uid})" />
		<clipPath id="fajr-tile-{uid}"><path d={TILE} /></clipPath>
		<g clip-path="url(#fajr-tile-{uid})">
			<circle cx={SUN_AT.x} cy={SUN_AT.y} r={SUN_AT.r * 1.6} fill="url(#fajr-sun-{uid})" />
		</g>
	</svg>
	<canvas bind:this={canvas} class:live></canvas>
	<!-- The letter rides above the sky, whether the sky is drawn or painted. -->
	<svg class="letter" viewBox="0 0 48 48" fill="none" aria-hidden="true">
		<path d={FA} fill="#fdf6e6" fill-opacity="0.96" />
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
