<script lang="ts">
	import { onMount } from 'svelte';

	let { size = 28, label = '' }: { size?: number; label?: string } = $props();

	const uid = $props.id();

	// The squircle the orb draws in its shader, |x|^4 + |y|^4 = 1, fitted to four
	// cubics: each handle lands at 0.919 of the half width.
	const TILE =
		'M48 24C48 46.06 46.06 48 24 48C1.94 48 0 46.06 0 24C0 1.94 1.94 0 24 0C46.06 0 48 1.94 48 24Z';

	// A twin ridge: the tall peak bites the sun on the left, the shoulder on the
	// right. The same five points drive the shader below.
	const RIDGE = 'M-6 54 16 23 24 33 32 27 54 54 54 64 -6 64Z';

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
		uniform vec3 u_brand;

		const vec2 A = vec2(-6.0, 54.0);
		const vec2 B = vec2(16.0, 23.0);
		const vec2 C = vec2(24.0, 33.0);
		const vec2 D = vec2(32.0, 27.0);
		const vec2 E = vec2(54.0, 54.0);

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

		float seg(float x, vec2 p, vec2 q) {
			return p.y + (q.y - p.y) * (x - p.x) / (q.x - p.x);
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

			vec3 white = vec3(0.99, 1.0, 1.0);
			vec3 sky = mix(u_brand, u_brand * 0.84, smoothstep(0.32, 0.76, f));
			vec3 sun = mix(white, mix(white, u_brand, 0.4), smoothstep(0.3, 0.74, f));
			vec3 rock = u_brand * 0.44;

			float aa = 48.0 / u_resolution.x * 1.2;
			float disc = smoothstep(13.0 + aa, 13.0 - aa, distance(q, vec2(25.0, 18.5)));

			float y = seg(q.x, A, B);
			y = mix(y, seg(q.x, B, C), step(B.x, q.x));
			y = mix(y, seg(q.x, C, D), step(C.x, q.x));
			y = mix(y, seg(q.x, D, E), step(D.x, q.x));
			float ridge = smoothstep(y - aa, y + aa, q.y);

			vec3 col = mix(mix(sky, sun, disc), rock, ridge);

			vec2 c = (uv - 0.5) * 2.0;
			float squircle = pow(abs(c.x), 4.0) + pow(abs(c.y), 4.0);
			float edge = smoothstep(1.0, 0.9, squircle);

			gl_FragColor = vec4(col * edge, edge);
		}
	`;

	function brandRgb(): [number, number, number] {
		const raw = getComputedStyle(document.documentElement).getPropertyValue('--color-brand').trim();
		let h = raw.replace('#', '');
		if (h.length === 3) h = h[0] + h[0] + h[1] + h[1] + h[2] + h[2];
		const n = parseInt(h, 16);
		if (h.length !== 6 || Number.isNaN(n)) return [0.016, 0.47, 0.34];
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

		const context = canvas.getContext('webgl', { antialias: true, alpha: true });
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

		const uResolution = gl.getUniformLocation(program, 'u_resolution');
		const uTime = gl.getUniformLocation(program, 'u_time');
		const uBrand = gl.getUniformLocation(program, 'u_brand');

		const dpr = Math.min(window.devicePixelRatio || 1, 2);
		const px = Math.round(size * dpr);
		canvas.width = px;
		canvas.height = px;
		gl.viewport(0, 0, px, px);
		gl.uniform2f(uResolution, px, px);
		gl.uniform3f(uBrand, ...brandRgb());

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

		const repaint = () => gl.uniform3f(uBrand, ...brandRgb());
		const scheme = window.matchMedia('(prefers-color-scheme: dark)');
		scheme.addEventListener('change', repaint);
		const themed = new MutationObserver(repaint);
		themed.observe(document.documentElement, { attributeFilter: ['data-theme'] });

		return () => {
			cancelAnimationFrame(raf);
			scheme.removeEventListener('change', repaint);
			themed.disconnect();
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
		<path d={TILE} fill="var(--color-brand)" />
		<clipPath id="fajr-tile-{uid}"><path d={TILE} /></clipPath>
		<g clip-path="url(#fajr-tile-{uid})">
			<circle cx="25" cy="18.5" r="13" fill="var(--color-brand-ink)" />
			<path d={RIDGE} fill="color-mix(in oklab, var(--color-brand) 46%, black)" />
		</g>
	</svg>
	<canvas bind:this={canvas} class:live></canvas>
</div>

<style>
	.mark {
		position: relative;
		max-inline-size: 100%;
	}

	svg,
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
</style>
