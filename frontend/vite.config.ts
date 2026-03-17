import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      '/api': { target: process.env.API_GATEWAY_URL || 'http://localhost:8080', changeOrigin: true },
      '/auth': { target: process.env.API_GATEWAY_URL || 'http://localhost:8080', changeOrigin: true },
    },
  },
})
