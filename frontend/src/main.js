import './polyfills.js' // 旧版 WebView 兼容：Object.groupBy / Map.groupBy（foliate-js 依赖）
import { createApp } from 'vue'
import axios from 'axios'
import ElementPlus from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import 'element-plus/dist/index.css'
import './styles/dialog.css'
import './styles/pagination.css'
import App from './App.vue'
import router from './router'
import { getToken, clearToken, setUser } from './api/auth'
import { applyAppPrefs } from './utils/appPrefs'

// axios 全局拦截器：自动携带 token，401 时跳转登录页
axios.interceptors.request.use((config) => {
  const token = getToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

axios.interceptors.response.use(
  (response) => response,
  (error) => {
    // hash 路由：登录页路径为 #/login，跳转需带 BASE_URL 前缀（兼容飞牛网关 /app/moyue 部署）
    if (error.response?.status === 401 && !window.location.hash.startsWith('#/login')) {
      clearToken()
      setUser(null)
      const redirect = encodeURIComponent(window.location.hash.slice(1) || '/bookshelf')
      window.location.replace(`${import.meta.env.BASE_URL}#/login?redirect=${redirect}`)
    }
    return Promise.reject(error)
  }
)

const app = createApp(App)

// 启动时应用全局外观配置（主题色 / 网站名称 / 网站图标 / 动效开关）
applyAppPrefs()

for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

app.use(ElementPlus, { locale: zhCn })
app.use(router)
app.mount('#app')
