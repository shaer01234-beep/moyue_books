import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  // 默认相对路径加载资源：App（WebView file://）打包可用
  // 网关/子路径部署时由构建脚本注入 VITE_BASE（如 /app/moyue/），
  // 避免相对路径在 /app/moyue（无尾斜杠）下被解析到 /app/ 导致资源 404
  base: process.env.VITE_BASE || './',
  plugins: [vue({
    template: {
      compilerOptions: {
        // foliate-js 注册的原生自定义元素，避免 Vue 警告/误编译
        isCustomElement: tag => tag === 'foliate-view'
      }
    }
  })],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src')
    }
  },
  optimizeDeps: {
    include: [
      'pdfjs-dist'
    ],
    exclude: [
      // foliate-js 含大量动态 import（vendor/zip.js 等），交由浏览器原生 ESM 加载更稳
      'foliate-js'
    ]
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:18080',
        changeOrigin: true,
        ws: true // 转发 WebSocket 升级（/api/tts/ws 语音合成）
      },
      '/static': {
        target: 'http://localhost:18080',
        changeOrigin: true
      }
    }
  }
})
