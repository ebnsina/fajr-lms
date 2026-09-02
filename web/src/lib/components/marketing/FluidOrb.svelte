<script lang="ts">
	import { onMount } from 'svelte';

	// A sphere of drifting colour, in the spirit of the FluidOrb on rareui.com,
	// written for Svelte rather than pulling in a React component.
	let { size = 320, label = '' }: { size?: number; label?: string } = $props();

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
		uniform vec3 deep;
		uniform vec3 mid;
		uniform vec3 warm;

		float hash(vec2 p) { return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453); }

		float noise(vec2 p) {
			vec2 i = floor(p), f = fract(p);
			vec2 u = f * f * (3.0 - 2.0 * f);
			return mix(mix(hash(i), hash(i + vec2(1.0, 0.0)), u.x),
			           mix(hash(i + vec2(0.0, 1.0)), hash(i + vec2(1.0, 1.0)), u.x), u.y);
		}

		// Layered noise, warped by itself: patches that drift and reform instead
		// of sliding past one another.
		float fluid(vec2 p, float t) {
			vec2 warp = vec2(noise(p + t * 0.12), noise(p.yx - t * 0.09));
			float n = noise(p * 1.6 + warp * 1.9 + vec2(t * 0.08, -t * 0.05));
			n += 0.5 * noise(p * 3.1 - warp * 1.2 - vec2(t * 0.06, t * 0.11));
			return n / 1.5;
		}

		void main() {
			vec2 uv = (gl_FragCoord.xy / size) * 2.0 - 1.0;
			float r = length(uv);
			if (r > 1.0) discard;

			// Bend the sampling toward the edges so the colour wraps like a sphere.
			float z = sqrt(max(0.0, 1.0 - r * r));
			vec2 p = uv / (0.55 + z * 0.75);

			float a = fluid(p + 0.0, time);
			float b = fluid(p * 1.4 - 3.7, time * 1.3);

			vec3 color = mix(deep, mid, smoothstep(0.25, 0.85, a));
			color = mix(color, warm, smoothstep(0.55, 1.0, b) * 0.7);

			// A little light from above, and a soft edge rather than a cut one.
			color += vec3(0.10) * pow(max(0.0, z), 2.5);
			float edge = smoothstep(1.0, 0.86, r);
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
		const uDeep = gl.getUniformLocation(program, 'deep');
		const uMid = gl.getUniformLocation(program, 'mid');
		const uWarm = gl.getUniformLocation(program, 'warm');

		const still = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
		const ratio = Math.min(window.devicePixelRatio || 1, 2);
		canvas.width = size * ratio;
		canvas.height = size * ratio;
		gl.viewport(0, 0, canvas.width, canvas.height);
		gl.uniform2f(uSize, canvas.width, canvas.height);

		// Dawn: the deep of the night, the green of the brand, the first light.
		gl.uniform3fv(uDeep, rgb('#052e21'));
		gl.uniform3fv(uMid, rgb('#0f9b6c'));
		gl.uniform3fv(uWarm, rgb('#f2c078'));

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
	<span class="halo" aria-hidden="true"></span>
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

	/* The light it throws on whatever it is sitting on. */
	.halo {
		position: absolute;
		inset: -22%;
		border-radius: 999px;
		background: radial-gradient(closest-side, var(--color-brand-soft), transparent 70%);
		opacity: 0.75;
	}
</style>
