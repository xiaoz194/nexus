import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    // 端口被占时直接报错，而不是默默顺延到 5174/5175（避免同时跑起多个 dev server）。
    strictPort: true,
    proxy: {
      // 仅开发用：把 API 请求转发到 Go 后端，避免跨域。
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
