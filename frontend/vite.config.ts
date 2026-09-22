import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const frontendDir = fileURLToPath(new URL('.', import.meta.url));

export default defineConfig({
  root: frontendDir,
  plugins: [tailwindcss(), svelte()],
  resolve: {
    alias: {
      $lib: path.resolve(frontendDir, 'src/lib'),
    },
  },
  build: {
    outDir: path.resolve(frontendDir, '../dist'),
    emptyOutDir: false,
    rollupOptions: {
      input: {
        index: path.resolve(frontendDir, 'index.html'),
        menu: path.resolve(frontendDir, 'menu.html'),
        'cash-register': path.resolve(frontendDir, 'cash-register.html'),
        day: path.resolve(frontendDir, 'day.html'),
      },
    },
  },
  server: {
    proxy: {
      '/api': 'http://localhost:3000',
    },
  },
});
