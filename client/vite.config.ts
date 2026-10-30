import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

// Dev-Server leitet /ws an den Go-Server (kaiserd, Port 8080) weiter,
// damit Client und Server lokal unter einer Adresse laufen.
export default defineConfig({
  plugins: [svelte()],
  server: {
    port: 5173,
    proxy: {
      '/ws': { target: 'ws://127.0.0.1:8080', ws: true },
      '/upload': 'http://127.0.0.1:8080',
      '/media': 'http://127.0.0.1:8080',
      '/api': 'http://127.0.0.1:8080',
    },
  },
});
