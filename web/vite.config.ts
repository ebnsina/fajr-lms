import adapter from '@sveltejs/adapter-node';
import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit({
			compilerOptions: {
				// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},

			// Self-hosted next to the API, per the deployment plan.
			adapter: adapter()
		})
	],

	ssr: {
		// The package ships .svelte source, which Node cannot load directly, so it
		// has to be bundled rather than treated as an external dependency.
		noExternal: ['@lucide/svelte']
	}
});
