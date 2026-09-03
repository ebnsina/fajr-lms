<script lang="ts">
	import WifiOff from '@lucide/svelte/icons/wifi-off';
</script>

<svelte:head><title>No connection · Fajr LMS</title></svelte:head>

<div class="mx-auto flex min-h-dvh max-w-md flex-col items-center justify-center gap-4 p-6 text-center">
	<WifiOff class="text-ink-soft" size={32} aria-hidden="true" />
	<h1 class="text-xl font-semibold">You are offline</h1>
	<p class="mb-0 text-ink-soft">
		This page needs a connection. What you had already opened is still there, and anything you
		typed is kept until you are back.
	</p>
	<button class="btn" type="button" onclick={() => location.reload()}>Try again</button>
</div>
