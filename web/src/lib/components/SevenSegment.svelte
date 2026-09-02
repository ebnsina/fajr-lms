<script lang="ts">
	let { digit }: { digit: string } = $props();

	// A real seven segment display shows its unlit segments too, which is what
	// makes a bedside clock read as an object rather than as text.
	const segments = [
		{ name: 'a', x: 2, y: 0, horizontal: true },
		{ name: 'f', x: 0, y: 2, horizontal: false },
		{ name: 'b', x: 12, y: 2, horizontal: false },
		{ name: 'g', x: 2, y: 12, horizontal: true },
		{ name: 'e', x: 0, y: 14, horizontal: false },
		{ name: 'c', x: 12, y: 14, horizontal: false },
		{ name: 'd', x: 2, y: 24, horizontal: true }
	];

	const lit = [
		'abcdef', 'bc', 'abged', 'abgcd', 'fgbc',
		'afgcd', 'afgecd', 'abc', 'abcdefg', 'abcdfg'
	];

	const horizontal = '2,0 10,0 12,2 10,4 2,4 0,2';
	const vertical = '0,2 2,0 4,2 4,10 2,12 0,10';

	// The strip carries every digit and rolls to the one being shown, the way the
	// numbers turn over on a mechanical clock.
	let value = $derived(Number.isNaN(Number(digit)) ? 0 : Number(digit));
</script>

<span class="digit-window">
	<span class="digit-strip" style="--offset: {value}">
		{#each lit as on, index (index)}
			<svg class="block h-11 w-auto" viewBox="0 0 16 28" role="presentation" focusable="false">
				{#each segments as seg (seg.name)}
					<polygon
						points={seg.horizontal ? horizontal : vertical}
						transform="translate({seg.x} {seg.y})"
						fill="currentColor"
						opacity={on.includes(seg.name) ? 1 : 0.08}
					/>
				{/each}
			</svg>
		{/each}
	</span>
</span>

<style>
	.digit-window {
		display: block;
		block-size: 2.75rem;
		overflow: hidden;
	}

	.digit-strip {
		display: block;
		transform: translateY(calc(var(--offset) * -2.75rem));
		transition: transform 320ms cubic-bezier(0.22, 1.2, 0.36, 1);
		will-change: transform;
	}

	@media (prefers-reduced-motion: reduce) {
		.digit-strip {
			transition: none;
		}
	}
</style>
