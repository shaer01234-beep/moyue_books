import { ref, watch, onMounted, onBeforeUnmount } from 'vue'

const MOBILE_MAX_WIDTH = 768
const isMobileNow = () => typeof window !== 'undefined' && window.innerWidth <= MOBILE_MAX_WIDTH

/**
 * 返回响应式的 isMobile，随窗口缩放实时更新。
 */
export function useIsMobile() {
  const isMobile = ref(isMobileNow())
  const check = () => { isMobile.value = isMobileNow() }
  let timer = null
  const onResize = () => {
    if (timer) clearTimeout(timer)
    timer = setTimeout(check, 150)
  }
  onMounted(() => {
    check()
    window.addEventListener('resize', onResize)
  })
  onBeforeUnmount(() => {
    window.removeEventListener('resize', onResize)
    if (timer) clearTimeout(timer)
  })
  return { isMobile }
}

/**
 * 手机端强制使用网格视图（取消列表视图）。
 * - 初始即在手机端时，viewMode 直接置为 'grid'
 * - 窗口缩放切入手机端时，自动切回 'grid'
 * @param {import('vue').Ref<string>} viewModeRef 视图模式 ref（'grid' | 'list'）
 * @returns {{ isMobile: import('vue').Ref<boolean> }}
 */
export function useMobileForceGrid(viewModeRef) {
  const { isMobile } = useIsMobile()
  const enforce = () => {
    if (isMobile.value && viewModeRef.value !== 'grid') {
      viewModeRef.value = 'grid'
    }
  }
  if (isMobile.value) enforce()
  watch(isMobile, enforce)
  return { isMobile }
}
