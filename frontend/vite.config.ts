import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  base: '/static/',
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:65000',
        changeOrigin: true,
      },
    },
  },
  build: {
    // 构建输出到后端源码的static目录，Go编译时会一起打包
    // 也可被build.sh复制到dist/windows/static和dist/linux/static
    outDir: resolve(__dirname, '../backend/static'),
    emptyOutDir: true,
    assetsDir: 'assets',
  },
})
