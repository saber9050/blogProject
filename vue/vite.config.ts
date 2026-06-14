import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:9527', // 后端地址
        changeOrigin: true,
        // 不重写路径，直接转发（关键！）
        rewrite: (path) => path
      }
    }
  }
})