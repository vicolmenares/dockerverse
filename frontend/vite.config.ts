import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
	define: {
		'import.meta.env.PUBLIC_API_URL': JSON.stringify(process.env.PUBLIC_API_URL || '')
	},
	build: {
		// Optimize chunk splitting
		rollupOptions: {
			output: {
				manualChunks: (id) => {
					// Don't chunk external modules
					if (id.includes('node_modules')) {
						if (id.includes('echarts') || id.includes('zrender')) {
							return 'echarts';
						}
						if (id.includes('@xterm')) {
							return 'xterm';
						}
						if (id.includes('lucide-svelte')) {
							return 'icons';
						}
					}
					return undefined;
				}
			}
		},
		// Minification
		minify: 'esbuild',
		target: 'es2020',
		// Source maps in prod for debugging
		sourcemap: false,
		// Chunk size warnings
		chunkSizeWarningLimit: 500
	},
	// Optimize deps
	optimizeDeps: {
		include: ['lucide-svelte', 'clsx'],
		exclude: ['@xterm/xterm']  // Lazy loaded
	},
	// Server config for dev
	server: {
		warmup: {
			clientFiles: ['./src/routes/+page.svelte', './src/lib/components/*.svelte']
		}
	}
});
