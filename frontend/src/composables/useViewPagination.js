import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'

// 响应式断点 -> 网格每行列数（与书架页一致）
const GRID_BREAKPOINTS = [
  { min: 1920, cols: 8 },
  { min: 1200, cols: 6 },
  { min: 992, cols: 4 },
  { min: 768, cols: 3 }
]

function gridColumns() {
  const w = window.innerWidth
  for (const b of GRID_BREAKPOINTS) {
    if (w >= b.min) return b.cols
  }
  return 2 // xs
}

const MOBILE_MAX_WIDTH = 768
const isMobileNow = () => window.innerWidth <= MOBILE_MAX_WIDTH

/**
 * 书架风格的分页组合函数：
 * - 每页条数基准 = 网格两行数量（每行列数 × 2）/ 列表固定 10
 * - 选项为基准的倍数（网格：两行倍数；列表：10 的倍数）
 * - 视图模式与每页条数均按视图分别记忆到 localStorage
 * - 窗口缩放（断点变化）自动重算网格基准并重新加载
 *
 * @param {string} storagePrefix 持久化键前缀（区分不同页面）
 * @param {() => void} reload 重新加载/切片当前页数据的回调
 */
export function useViewPagination(storagePrefix, reload) {
  const VIEW_KEY = `${storagePrefix}_view_mode`
  const storedMode = () => localStorage.getItem(VIEW_KEY) || 'grid'
  // 手机端强制网格视图；桌面端读取已保存偏好
  const viewMode = ref(isMobileNow() ? 'grid' : storedMode())

  // 基准：网格=两行数量，列表=10
  const pageBase = computed(() => (viewMode.value === 'grid' ? gridColumns() * 2 : 10))
  // 选项：网格为两行倍数，列表为 10 的倍数
  const pageSizesOptions = computed(() => [
    pageBase.value,
    pageBase.value * 2,
    pageBase.value * 3,
    pageBase.value * 4
  ])

  const sizeKey = (mode) => `${storagePrefix}_page_size_${mode}`
  const persistedSize = (mode) => {
    const n = Number(localStorage.getItem(sizeKey(mode)))
    return n > 0 ? n : null
  }

  const page = ref(1)
  const pageSize = ref(persistedSize(viewMode.value) || pageBase.value)

  let lastBase = pageBase.value
  let resizeTimer = null

  const onSizeChange = () => {
    page.value = 1
    localStorage.setItem(sizeKey(viewMode.value), String(pageSize.value))
    reload()
  }

  const applyAutoBase = () => {
    const next = pageBase.value
    if (next === lastBase) return
    lastBase = next
    pageSize.value = next
    page.value = 1
    reload()
  }

  const onResize = () => {
    if (resizeTimer) clearTimeout(resizeTimer)
    resizeTimer = setTimeout(() => {
      applyAutoBase()
      if (isMobileNow()) {
        // 切入手机端：强制网格视图（不写回偏好，保留桌面设置）
        if (viewMode.value !== 'grid') viewMode.value = 'grid'
      } else {
        // 回到桌面端：恢复已保存的视图偏好
        const pref = storedMode()
        if (viewMode.value !== pref) viewMode.value = pref
      }
    }, 200)
  }

  watch(viewMode, (v) => {
    // 手机端不持久化视图偏好，避免覆盖桌面端设置
    if (!isMobileNow()) localStorage.setItem(VIEW_KEY, v)
    pageSize.value = persistedSize(v) || pageBase.value
    lastBase = pageBase.value
    page.value = 1
    reload()
  })

  onMounted(() => window.addEventListener('resize', onResize))
  onBeforeUnmount(() => {
    window.removeEventListener('resize', onResize)
    if (resizeTimer) clearTimeout(resizeTimer)
  })

  return { viewMode, page, pageSize, pageSizesOptions, onSizeChange }
}
