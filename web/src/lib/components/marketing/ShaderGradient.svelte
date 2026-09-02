<script lang="ts">
	import { onMount } from 'svelte';

	let { light = '#eef0ff', dark = '#171a2b' }: { light?: string; dark?: string } = $props();

	let canvas = $state<HTMLCanvasElement | null>(null);
	let failed = $state(false);

	// Two triangles and a fragment shader: cheaper than an animated image and it
	// never blocks the text above it.
	const vertex = `
		attribute vec2 position;
		void main() { gl_Position = vec4(position, 0.0, 1.0); }
	`;

	const fragment = `
		precision mediump float;
		uniform vec2 size;
		uniform float time;
		uniform vec3 base;
		uniform vec3 warm;
		uniform vec3 cool;

		// Cheap value noise: enough to make the bands drift without looking like a loop.
		float hash(vec2 p) { return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453); }
		float noise(vec2 p) {
			vec2 i = floor(p), f = fract(p);
			vec2 u = f * f * (3.0 - 2.0 * f);
			return mix(mix(hash(i), hash(i + vec2(1.0, 0.0)), u.x),
			           mix(hash(i + vec2(0.0, 1.0)), hash(i + vec2(1.0, 1.0)), u.x), u.y);
		}

		void main() {
			vec2 uv = gl_FragCoord.xy / size;
			float t = time * 0.06;
			float n = noise(uv * 2.2 + vec2(t, t * 0.6));
			n += 0.5 * noise(uv * 4.5 - vec2(t * 0.8, t));
			float band = smoothstep(0.2, 1.1, n + uv.y * 0.5);
			vec3 color = mix(base, cool, band);
			color = mix(color, warm, smoothstep(0.55, 1.25, n * uv.x + 0.25));
			// Fade out at the bottom so the section below joins without a seam.
			color = mix(color, base, smoothstep(0.55, 1.0, uv.y * -1.0 + 1.0) * 0.35);
			gl_FragColor = vec4(color, 1.0);
		}
	`;

	function rgb(hex: string): [number, number, number] {
		const value = parseInt(hex.replace('#', ''), 16);
		return [((value >> 16) & 255) / 255, ((value >> 8) & 255) / 255, (value & 255) / 255];
	}

	function compile(gl: WebGLRenderingContext, kind: number, source: string) {
		const shader = gl.createShader(kind);
		if (!shader) return null;
		gl.shaderSource(shader, source);
		gl.compileShader(shader);
		if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) return null;
		return shader;
	}

	onMount(() => {
		if (!canvas) return;
		const context = canvas.getContext('webgl', { antialias: false, alpha: false });
		if (!context) {
			failed = true;
			return;
		}
		const gl: WebGLRenderingContext = context;
		const surface = canvas;

		const vs = compile(gl, gl.VERTEX_SHADER, vertex);
		const fs = compile(gl, gl.FRAGMENT_SHADER, fragment);
		const program = gl.createProgram();
		if (!vs || !fs || !program) {
			failed = true;
			return;
		}
		gl.attachShader(program, vs);
		gl.attachShader(program, fs);
		gl.linkProgram(program);
		if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
			failed = true;
			return;
		}
		gl.useProgram(program);

		const buffer = gl.createBuffer();
		gl.bindBuffer(gl.ARRAY_BUFFER, buffer);
		gl.bufferData(gl.ARRAY_BUFFER, new Float32Array([-1, -1, 3, -1, -1, 3]), gl.STATIC_DRAW);
		const position = gl.getAttribLocation(program, 'position');
		gl.enableVertexAttribArray(position);
		gl.vertexAttribPointer(position, 2, gl.FLOAT, false, 0, 0);

		const uSize = gl.getUniformLocation(program, 'size');
		const uTime = gl.getUniformLocation(program, 'time');
		const uBase = gl.getUniformLocation(program, 'base');
		const uWarm = gl.getUniformLocation(program, 'warm');
		const uCool = gl.getUniformLocation(program, 'cool');

		const still = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
		const media = window.matchMedia('(prefers-color-scheme: dark)');

		function palette() {
			const stamped = document.documentElement.dataset.theme;
			const isDark = stamped === 'dark' || (stamped !== 'light' && media.matches);
			gl.uniform3fv(uBase, rgb(isDark ? dark : light));
			gl.uniform3fv(uWarm, rgb(isDark ? '#3b2a4d' : '#ffe9d0'));
			gl.uniform3fv(uCool, rgb(isDark ? '#2a2c63' : '#dcdcfb'));
		}

		function resize() {
			const ratio = Math.min(window.devicePixelRatio || 1, 1.5);
			surface.width = Math.floor(surface.clientWidth * ratio);
			surface.height = Math.floor(surface.clientHeight * ratio);
			gl.viewport(0, 0, surface.width, surface.height);
			gl.uniform2f(uSize, surface.width, surface.height);
		}

		let frame = 0;
		const start = performance.now();
		function draw() {
			gl.uniform1f(uTime, still ? 8 : (performance.now() - start) / 1000);
			gl.drawArrays(gl.TRIANGLES, 0, 3);
			if (!still) frame = requestAnimationFrame(draw);
		}

		palette();
		resize();
		draw();

		// The theme can change under us: from the system, or from the toggle in
		// the footer, which stamps the root element.
		const stamped = new MutationObserver(() => {
			palette();
			if (still) draw();
		});
		stamped.observe(document.documentElement, { attributeFilter: ['data-theme'] });

		window.addEventListener('resize', resize);
		media.addEventListener('change', palette);
		return () => {
			cancelAnimationFrame(frame);
			stamped.disconnect();
			window.removeEventListener('resize', resize);
			media.removeEventListener('change', palette);
		};
	});
</script>

<canvas class="gradient" class:hidden={failed} bind:this={canvas} aria-hidden="true"></canvas>

<style>
	.gradient {
		position: absolute;
		inset: 0;
		inline-size: 100%;
		block-size: 100%;
		display: block;
	}
</style>
