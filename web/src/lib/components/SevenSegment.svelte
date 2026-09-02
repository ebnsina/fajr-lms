<script lang="ts">
	let { digit }: { digit: string } = $props();

	// A real seven segment display shows its unlit segments too, which is what
	// makes a bedside clock read as a physical object rather than as text.
	const segments = {
		a: { x: 2, y: 0, horizontal: true },
		f: { x: 0, y: 2, horizontal: false },
		b: { x: 12, y: 2, horizontal: false },
		g: { x: 2, y: 12, horizontal: true },
		e: { x: 0, y: 14, horizontal: false },
		c: { x: 12, y: 14, horizontal: false },
		d: { x: 2, y: 24, horizontal: true }
	} as const;

	const lit: Record<string, string> = {
		'0': 'abcdef',
		'1': 'bc',
		'2': 'abged',
		'3': 'abgcd',
		'4': 'fgbc',
		'5': 'afgcd',
		'6': 'afgecd',
		'7': 'abc',
		'8': 'abcdefg',
		'9': 'abcdfg'
	};

	const horizontal = '2,0 10,0 12,2 10,4 2,4 0,2';
	const vertical = '0,2 2,0 4,2 4,10 2,12 0,10';

	let on = $derived(lit[digit] ?? '');
</script>

<svg class="block h-11 w-auto" viewBox="0 0 16 28" role="presentation" focusable="false">
	{#each Object.entries(segments) as [name, seg] (name)}
		<polygon
			points={seg.horizontal ? horizontal : vertical}
			transform="translate({seg.x} {seg.y})"
			fill="currentColor"
			opacity={on.includes(name) ? 1 : 0.08}
		/>
	{/each}
</svg>
