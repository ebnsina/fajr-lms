<script lang="ts">
	import { onMount } from 'svelte';

	// A sphere of drifting colour, in the spirit of the FluidOrb on rareui.com,
	// written for Svelte rather than pulling in a React component.
	// The middle and bottom bands are derived from one colour — a pale tint and
	// the full thing — while the top of the sphere stays white.
	let {
		size = 320,
		color = '#059669',
		label = ''
	}: { size?: number; color?: string; label?: string } = $props();

	let canvas = $state<HTMLCanvasElement | null>(null);
	let failed = $state(false);

	const vertex = `
		attribute vec2 position;
		void main() { gl_Position = vec4(position, 0.0, 1.0); }
	`;

	const fragment = `
		precision mediump float;
		uniform vec2 size;
		uniform float time;
		uniform vec3 tint;
		uniform vec3 full;

		float hash(vec2 p) { return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453); }

		float noise(vec2 p) {
			vec2 i = floor(p), f = fract(p);
			vec2 u = f * f * (3.0 - 2.0 * f);
			return mix(mix(hash(i), hash(i + vec2(1.0, 0.0)), u.x),
			           mix(hash(i + vec2(0.0, 1.0)), hash(i + vec2(1.0, 1.0)), u.x), u.y);
		}

		// Noise warped by itself, so the bands wander and reform rather than slide.
		float fluid(vec2 p, float t) {
			vec2 warp = vec2(noise(p * 1.3 + t * 0.11), noise(p.yx * 1.1 - t * 0.08));
			float n = noise(p * 1.7 + warp * 1.6 + vec2(t * 0.07, -t * 0.05));
			n += 0.5 * noise(p * 3.3 - warp * 1.1 + vec2(-t * 0.05, t * 0.09));
			return n / 1.5;
		}

		void main() {
			vec2 uv = (gl_FragCoord.xy / size) * 2.0 - 1.0;
			float r = length(uv);
			if (r > 1.0) discard;

			// Bend the sampling toward the rim so the bands wrap like a sphere.
			float z = sqrt(max(0.0, 1.0 - r * r));
			vec2 p = uv / (0.6 + z * 0.7);

			// Height down the sphere, pushed about by the fluid.
			float band = (1.0 - (uv.y * 0.5 + 0.5)) + (fluid(p, time) - 0.5) * 0.55;

			vec3 color = vec3(1.0);
			color = mix(color, tint, smoothstep(0.34, 0.62, band));
			color = mix(color, full, smoothstep(0.62, 0.92, band));

			// The rim turns away from the light rather than ending in a hard line.
			color *= 0.82 + 0.18 * pow(max(0.0, z), 0.6);
			float edge = smoothstep(1.0, 0.965, r);
			gl_FragColor = vec4(color, edge);
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
		const context = canvas.getContext('webgl', { antialias: false, premultipliedAlpha: false });
		if (!context) {
			failed = true;
			return;
		}
		const gl: WebGLRenderingContext = context;

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
		gl.enable(gl.BLEND);
		gl.blendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA);

		const buffer = gl.createBuffer();
		gl.bindBuffer(gl.ARRAY_BUFFER, buffer);
		gl.bufferData(gl.ARRAY_BUFFER, new Float32Array([-1, -1, 3, -1, -1, 3]), gl.STATIC_DRAW);
		const position = gl.getAttribLocation(program, 'position');
		gl.enableVertexAttribArray(position);
		gl.vertexAttribPointer(position, 2, gl.FLOAT, false, 0, 0);

		const uSize = gl.getUniformLocation(program, 'size');
		const uTime = gl.getUniformLocation(program, 'time');
		const uTint = gl.getUniformLocation(program, 'tint');
		const uFull = gl.getUniformLocation(program, 'full');

		const still = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
		const ratio = Math.min(window.devicePixelRatio || 1, 2);
		canvas.width = size * ratio;
		canvas.height = size * ratio;
		gl.viewport(0, 0, canvas.width, canvas.height);
		gl.uniform2f(uSize, canvas.width, canvas.height);

		const [r, g, b] = rgb(color);
		gl.uniform3fv(uTint, [r + (1 - r) * 0.72, g + (1 - g) * 0.72, b + (1 - b) * 0.72]);
		gl.uniform3fv(uFull, [r, g, b]);

		let frame = 0;
		const start = performance.now();
		function draw() {
			gl.uniform1f(uTime, still ? 12 : (performance.now() - start) / 1000);
			gl.clear(gl.COLOR_BUFFER_BIT);
			gl.drawArrays(gl.TRIANGLES, 0, 3);
			if (!still) frame = requestAnimationFrame(draw);
		}
		draw();

		return () => cancelAnimationFrame(frame);
	});
</script>

<div class="orb" style:inline-size="{size}px" style:block-size="{size}px" role="img" aria-label={label}>
	<canvas class:hidden={failed} bind:this={canvas}></canvas>
</div>

<style>
	.orb {
		position: relative;
		display: block;
		max-inline-size: 100%;
	}

	canvas {
		position: relative;
		z-index: 1;
		inline-size: 100%;
		block-size: 100%;
		border-radius: 999px;
	}

</style>
