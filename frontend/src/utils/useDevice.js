// 设备自适应：手机（<=768px）自动进入 H5 版，桌面保持 PC 布局
import { ref } from 'vue'

const MOBILE_QUERY = '(max-width: 768px)'
const isMobile = ref(false)

// 模块加载时初始化一次（单例，避免重复监听）
if (typeof window !== 'undefined' && window.matchMedia) {
  const mql = window.matchMedia(MOBILE_QUERY)
  isMobile.value = mql.matches
  const handler = (e) => {
    isMobile.value = e.matches
  }
  if (mql.addEventListener) {
    mql.addEventListener('change', handler)
  } else if (mql.addListener) {
    mql.addListener(handler)
  }
}

// 返回响应式引用，组件可直接使用 isMobile.value
export function useIsMobile() {
  return isMobile
}
