<script lang="ts">
	import { onMount } from 'svelte';

	let canvas = $state<HTMLCanvasElement | null>(null);
	let failed = $state(false);

	const vertex = `
		attribute vec2 position;
		void main() { gl_Position = vec4(position, 0.0, 1.0); }
	`;

	// Aurora: a few wide bands of light bending across the frame, with grain over
	// the top so the gradient never bands on a cheap screen.
	const fragment = `
		precision mediump float;
		uniform vec2 size;
		uniform float time;
		uniform vec3 base;
		uniform vec3 glow;
		uniform vec3 deep;
		uniform float grain;

		float hash(vec2 p) { return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453); }

		float noise(vec2 p) {
			vec2 i = floor(p), f = fract(p);
			vec2 u = f * f * (3.0 - 2.0 * f);
			return mix(mix(hash(i), hash(i + vec2(1.0, 0.0)), u.x),
			           mix(hash(i + vec2(0.0, 1.0)), hash(i + vec2(1.0, 1.0)), u.x), u.y);
		}

		// One ribbon: a soft line that waves across the frame and fades at the edges.
		float ribbon(vec2 uv, float offset, float speed, float thickness) {
			float t = time * speed;
			float wave = sin(uv.x * 2.1 + t) * 0.11
			           + sin(uv.x * 3.7 - t * 1.3) * 0.06
			           + noise(vec2(uv.x * 1.5, t * 0.35)) * 0.10;
			float d = abs(uv.y - (offset + wave));
			return smoothstep(thickness, 0.0, d);
		}

		void main() {
			vec2 uv = gl_FragCoord.xy / size;
			float aspect = size.x / size.y;
			vec2 p = vec2(uv.x * aspect, uv.y);

			vec3 color = mix(base, deep, smoothstep(0.1, 1.6, uv.y) * 0.5);

			// The bands sweep the top and the floor; the middle stays quiet so the
			// headline sits on nearly flat colour.
			color = mix(color, glow, ribbon(uv, 0.93, 0.09, 0.10) * 0.55);
			color = mix(color, deep, ribbon(uv, 1.02, 0.06, 0.09) * 0.45);
			color = mix(color, glow, ribbon(uv, 0.02, 0.12, 0.09) * 0.40);

			// A wide, low pool of light behind the middle of the text.
			// One corner catches the light, the way a low sun would.
			float corner = smoothstep(0.55, 0.0, length(p - vec2(aspect * 0.9, 0.92)));
			color = mix(color, glow, corner * 0.35);

			color += (hash(gl_FragCoord.xy * 0.7) - 0.5) * grain;
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
		const uGlow = gl.getUniformLocation(program, 'glow');
		const uDeep = gl.getUniformLocation(program, 'deep');
		const uGrain = gl.getUniformLocation(program, 'grain');

		const still = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
		const media = window.matchMedia('(prefers-color-scheme: dark)');

		function palette() {
			const stamped = document.documentElement.dataset.theme;
			const isDark = stamped === 'dark' || (stamped !== 'light' && media.matches);
			gl.uniform3fv(uBase, rgb(isDark ? '#0f1113' : '#f6f7f8'));
			gl.uniform3fv(uGlow, rgb(isDark ? '#0d6e4f' : '#8fd8b8'));
			gl.uniform3fv(uDeep, rgb(isDark ? '#12233a' : '#ccdcea'));
			gl.uniform1f(uGrain, isDark ? 0.05 : 0.035);
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
			gl.uniform1f(uTime, still ? 6 : (performance.now() - start) / 1000);
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

<canvas class="aurora" class:hidden={failed} bind:this={canvas} aria-hidden="true"></canvas>

<style>
	.aurora {
		position: absolute;
		inset: 0;
		inline-size: 100%;
		block-size: 100%;
		display: block;
	}
</style>
