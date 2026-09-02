<script lang="ts">
	let {
		name = 'code',
		length = 6,
		id = 'code'
	}: { name?: string; length?: number; id?: string } = $props();

	let value = $state('');
	let focused = $state(false);
	let input: HTMLInputElement | null = $state(null);

	let digits = $derived(Array.from({ length }, (_, i) => value[i] ?? ''));
	// The caret sits on the next empty slot, or on the last one when full.
	let active = $derived(focused ? Math.min(value.length, length - 1) : -1);

	function onInput(event: Event) {
		const el = event.currentTarget as HTMLInputElement;
		value = el.value.replace(/\D/g, '').slice(0, length);
		el.value = value;
	}
</script>

<!-- One real input underneath the slots, so pasting a code, the SMS autofill
     prompt and password managers all keep working. The slots are only paint. -->
<div class="relative">
	<input
		bind:this={input}
		{id}
		{name}
		class="absolute inset-0 h-full w-full cursor-pointer rounded-xl bg-transparent text-transparent caret-transparent outline-none selection:bg-transparent"
		type="text"
		inputmode="numeric"
		autocomplete="one-time-code"
		pattern="[0-9]*"
		maxlength={length}
		required
		dir="ltr"
		oninput={onInput}
		onfocus={() => (focused = true)}
		onblur={() => (focused = false)}
	/>

	<div class="pointer-events-none flex gap-2" aria-hidden="true">
		{#each digits as digit, index (index)}
			<div
				class="flex flex-1 items-center justify-center rounded-xl border bg-raised font-mono text-xl transition-colors"
				class:border-line-strong={index !== active}
				class:border-brand={index === active}
				style="height: var(--size-control)"
			>
				{#if digit}
					{digit}
				{:else if index === active}
					<span class="h-5 w-px animate-pulse bg-ink"></span>
				{/if}
			</div>
		{/each}
	</div>
</div>
