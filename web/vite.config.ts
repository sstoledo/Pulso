import tailwindcss from '@tailwindcss/vite'
import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vitest/config'

const apiProxyTarget = process.env.VITE_API_PROXY_TARGET ?? 'http://localhost:8080'

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  server: {
    host: '0.0.0.0',
    port: 5173,
    proxy: {
      '/api': apiProxyTarget,
      '/healthz': apiProxyTarget,
      '/readyz': apiProxyTarget,
    },
  },
  test: {
    environment: 'jsdom',
  },
})
