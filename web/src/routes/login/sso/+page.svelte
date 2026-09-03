<script lang="ts">
	import type { PageProps } from './$types';
	let { data }: PageProps = $props();
</script>

<svelte:head><title>Sign in · Fajr LMS</title></svelte:head>

<div class="mx-auto flex min-h-dvh max-w-md flex-col justify-center gap-4 p-6">
	<h1 class="text-xl font-semibold">That sign-in did not go through</h1>
	<p class="banner-bad text-sm" role="alert">{data.problem}</p>
	<a class="btn" href="/login">Try again</a>
</div>
