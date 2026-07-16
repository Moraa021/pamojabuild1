import { defineConfig } from 'vite';
 
export default defineConfig({
  // Serve from project root so index.html is found
  root: '.',
 
  server: {
    port: 3000,
    // Proxy all /api/* requests to the Go backend during development
    proxy: {
      '/api': {
        target:      'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
 
  build: {
    outDir:    'dist',
    emptyOutDir: true,
    rollupOptions: {
      input: 'index.html',
    },
  },
 
  test: {
    // Vitest config (when used via `vite.config.js`)
    environment: 'jsdom',
    include:     ['tests/unit/**/*.test.js', 'tests/integration/**/*.test.js'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html'],
    },
  },
});
 
