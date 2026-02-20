import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/replay': 'http://localhost:8080',
      '/scenarios': 'http://localhost:8080',
      '/healthz': 'http://localhost:8080',
      '/metrics': 'http://localhost:8080'
    }
  }
});
