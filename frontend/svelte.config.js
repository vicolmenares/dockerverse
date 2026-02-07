import adapter from '@sveltejs/adapter-node';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		adapter: adapter({
			out: 'build',
			precompress: true,  // Enable gzip/brotli precompression
			envPrefix: ''
		}),
		// Preload critical modules
		prerender: {
			handleMissingId: 'ignore'
		},
		// Inline critical CSS
		inlineStyleThreshold: 5000
	},
	// Compile options for smaller output
	compilerOptions: {
		cssHash: ({ hash, css }) => `s-${hash(css)}`
	}
};

export default config;
