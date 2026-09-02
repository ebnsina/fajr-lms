<script lang="ts">
	import '../app.css';
	import type { LayoutProps } from './$types';

	let { data, children }: LayoutProps = $props();

	// The server stamps direction and theme onto the document, but a client side
	// navigation does not reload it, so they are kept in step here too.
	$effect(() => {
		const root = document.documentElement;
		root.dir = data.session?.tenant?.default_dir === 'rtl' ? 'rtl' : 'ltr';
		root.lang = data.session?.tenant?.locale ?? 'en';
	});
</script>

<svelte:head>
	<title>Fajr</title>
</svelte:head>

{@render children()}
