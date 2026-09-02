<script lang="ts">
	let { percent, label }: { percent: number; label?: string } = $props();
</script>

<div class="flex items-center gap-3">
	<div
		class="h-2 flex-1 overflow-hidden rounded-full bg-line"
		role="progressbar"
		aria-valuenow={percent}
		aria-valuemin="0"
		aria-valuemax="100"
		aria-label={label ?? 'Progress'}
	>
		<div class="h-full rounded-full bg-brand transition-[width]" style="width: {percent}%"></div>
	</div>
	<span class="text-sm tabular-nums text-ink-soft">{percent}%</span>
</div>
