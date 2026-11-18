import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

const proxy = {
  '/api': {
    target: 'http://localhost:8080',
    changeOrigin: true,
    // LLM-backed /ai/chat calls can take longer than the proxy default.
    timeout: 120000,
    proxyTimeout: 120000
  }
}

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 3000,
    proxy
  },
  preview: {
    port: 3000,
    proxy
  }
})
