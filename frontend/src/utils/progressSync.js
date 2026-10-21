// 全局阅读进度 store：阅读器更新进度后通知详情页/书架刷新
// 用单例 reactive 对象，避免引入 Pinia 增加打包体积
import { reactive } from 'vue'

const state = reactive({
  // bookId -> { progress, current_page, tts_location, updated_at }
  updates: {},
  // 标记某本书最近被更新过，用于详情页/书架 onMounted 时主动拉取
  recentlyUpdated: {}
})

// 记录一次进度更新（阅读器调用）
export function notifyProgressUpdate(bookId, payload) {
  if (!bookId) return
  const now = Date.now()
  state.updates[bookId] = { ...(state.updates[bookId] || {}), ...payload, updated_at: now }
  state.recentlyUpdated[bookId] = now
  // 30 秒后清除 recentlyUpdated 标记，避免详情页一直认为需要刷新
  setTimeout(() => {
    if (state.recentlyUpdated[bookId] === now) delete state.recentlyUpdated[bookId]
  }, 30_000)
}

// 读取某本书的最近进度（详情页/书架调用）
export function getProgressUpdate(bookId) {
  return state.updates[bookId] || null
}

// 检查某本书最近 30 秒内是否更新过（详情页 onMounted 时据此决定是否主动重新拉取）
export function wasRecentlyUpdated(bookId) {
  return !!state.recentlyUpdated[bookId]
}

// 消费更新：标记后清除（详情页拉取完成后调用，避免重复触发）
export function consumeProgressUpdate(bookId) {
  delete state.recentlyUpdated[bookId]
}

// 暴露 state 用于组件响应式订阅
export function getProgressState() {
  return state
}