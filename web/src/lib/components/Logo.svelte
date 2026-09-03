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

	// Fā', the first letter of فجر, drawn as an outline taken from Jomhuria
	// (SIL OFL) rather than set as text: a mark cannot depend on the reader
	// having an Arabic face installed.
	const FA =
		'M25.59 11.18H26.12L28.56 8.83V8.33L26.12 6.00H25.59L23.25 8.33V8.83ZM11.24 34.00H27.99Q28.90 34.00 29.63 33.69Q30.36 33.38 30.81 32.89Q31.26 32.40 31.58 31.75Q31.89 31.10 32.01 30.48Q32.14 29.85 32.14 29.23V19.92Q32.14 16.88 30.40 14.90Q28.67 12.92 26.07 12.92Q23.40 12.92 22.19 14.33Q20.98 15.74 20.98 18.03V21.98Q20.98 23.44 22.12 24.37Q22.95 25.05 24.02 25.05H28.42Q28.67 25.05 28.84 25.22Q29.02 25.38 29.02 25.63Q29.02 25.88 28.84 26.06Q28.67 26.24 28.42 26.24H11.49Q9.69 26.24 9.58 24.83Q9.58 24.74 9.58 24.65Q9.58 23.94 9.98 23.28Q10.38 22.63 10.78 22.32L11.19 22.02L9.28 20.56Q8.57 20.95 8.00 21.52Q7.43 22.09 7.08 22.66Q6.74 23.23 6.48 23.92Q6.22 24.60 6.10 25.08Q5.99 25.56 5.93 26.11Q5.86 26.65 5.86 26.79Q5.86 26.93 5.86 27.06V28.62Q5.86 30.85 7.44 32.42Q9.01 34.00 11.24 34.00ZM23.24 17.89Q23.24 16.32 24.68 16.32Q25.12 16.32 25.68 16.64Q26.24 16.95 26.74 17.38Q27.24 17.82 27.68 18.26Q28.11 18.69 28.36 19.01L28.61 19.31H24.91Q24.32 19.31 23.78 18.96Q23.24 18.60 23.24 17.89Z';

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
