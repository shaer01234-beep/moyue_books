<template>
  <div class="share-preview">
    <div class="bg-glow bg-glow-1"></div>
    <div class="bg-glow bg-glow-2"></div>

    <div v-if="loading" class="share-loading">
      <div class="loading-card">
        <el-icon class="is-loading" :size="34"><Loading /></el-icon>
        <p>正在加载分享…</p>
      </div>
    </div>

    <div v-else-if="error" class="share-error">
      <div class="error-card">
        <el-icon :size="44" class="error-icon"><CircleCloseFilled /></el-icon>
        <h2>{{ errorTitle }}</h2>
        <p>{{ errorMsg }}</p>
        <el-button type="primary" @click="goHome">返回书架</el-button>
      </div>
    </div>

    <div v-else-if="info" class="share-card">
      <div class="share-cover-col">
        <div class="share-cover-inner">
          <img
            v-if="info.book.has_cover"
            :src="shareCoverUrl"
            class="share-cover"
            alt="cover"
          />
          <div v-else class="share-cover-placeholder">
            <el-icon :size="64"><Reading /></el-icon>
          </div>
        </div>
      </div>

      <div class="share-info">
        <div class="share-badge">
          <el-icon :size="14"><Share /></el-icon>
          好友分享的图书
        </div>

        <h1 class="share-title">{{ info.book.title }}</h1>
        <div class="share-authors">{{ formatAuthors(info.book.authors) }}</div>

        <div class="share-meta">
          <span v-if="info.book.file_type" class="type-tag">{{ info.book.file_type.toUpperCase() }}</span>
          <el-tag v-if="info.book.publisher" size="small" type="info" effect="plain">{{ info.book.publisher }}</el-tag>
          <el-tag v-if="info.book.language" size="small" type="info" effect="plain">{{ info.book.language }}</el-tag>
          <el-tag v-if="info.book.pages" size="small" type="info" effect="plain">{{ info.book.pages }} 页</el-tag>
          <el-tag v-if="info.book.file_size" size="small" type="info" effect="plain">{{ formatSize(info.book.file_size) }}</el-tag>
        </div>

        <p v-if="info.book.description" class="share-description">{{ info.book.description }}</p>

        <div class="share-from">
          <el-icon><User /></el-icon>
          <span>由 <b>{{ info.sharer || '书友' }}</b> 分享</span>
          <span v-if="info.share.expires_at" class="expire-tip">
            · {{ isExpired ? '已过期' : `有效期至 ${formatDate(info.share.expires_at)}` }}
          </span>
        </div>

        <div class="share-actions">
          <el-button
            v-if="isOwnBook"
            type="info"
            size="large"
            disabled
          >
            <el-icon><CircleCheck /></el-icon>
            这是你分享的书
          </el-button>
          <el-button
            v-else-if="!isLogin"
            type="primary"
            size="large"
            @click="goLogin"
          >
            <el-icon><Key /></el-icon>
            登录后接收分享
          </el-button>
          <el-button
            v-else
            type="primary"
            size="large"
            :loading="accepting"
            @click="accept"
          >
            <el-icon><Download /></el-icon>
            接收分享 · 加入我的书架
          </el-button>
        </div>

        <div class="share-tip">
          <el-icon :size="14"><InfoFilled /></el-icon>
          <span>接收后会复制到你的书架，拥有独立副本，可随时阅读与管理。</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { shareApi } from '@/api/share'
import { getToken } from '@/api/auth'

const route = useRoute()
const router = useRouter()

const loading = ref(true)
const error = ref(false)
const errorTitle = ref('分享不存在')
const errorMsg = ref('')
const info = ref(null)
const accepting = ref(false)

const isLogin = computed(() => !!getToken())
const isOwnBook = computed(() => info.value && info.value.is_own)
const shareCoverUrl = computed(() => info.value ? shareApi.shareCoverUrl(info.value.share.token) : '')
const isExpired = computed(() => {
  if (!info.value?.share?.expires_at) return false
  return new Date(info.value.share.expires_at).getTime() < Date.now()
})

const formatAuthors = (authors) => {
  if (!authors) return '未知作者'
  try {
    const parsed = JSON.parse(authors)
    return Array.isArray(parsed) && parsed.length > 0 ? parsed.join(', ') : '未知作者'
  } catch {
    return '未知作者'
  }
}

const formatSize = (size) => {
  if (!size) return ''
  if (size < 1024 * 1024) return (size / 1024).toFixed(0) + ' KB'
  return (size / 1024 / 1024).toFixed(1) + ' MB'
}

const formatDate = (d) => {
  const date = new Date(d)
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

const load = async () => {
  loading.value = true
  error.value = false
  try {
    const { data } = await shareApi.getShareInfo(route.params.token)
    info.value = data
    // 判断是否是自己的分享：公开接口返回 is_own 需要登录态，此处通过额外请求判断
    if (isLogin.value) {
      try {
        const own = await shareApi.checkOwn(route.params.token)
        info.value = { ...data, is_own: own.data?.is_own }
      } catch (e) { /* 忽略 */ }
    }
  } catch (e) {
    error.value = true
    if (e.response?.status === 404) {
      errorTitle.value = '分享不存在'
      errorMsg.value = '链接可能已被分享者撤销，或书籍已被删除。'
    } else if (e.response?.status === 410) {
      errorTitle.value = '分享已过期'
      errorMsg.value = '该分享链接已超过有效期，请联系分享者重新生成。'
    } else {
      errorTitle.value = '加载失败'
      errorMsg.value = '网络异常，请稍后重试。'
    }
  } finally {
    loading.value = false
  }
}

const goLogin = () => {
  router.push({ path: '/login', query: { redirect: route.fullPath } })
}

const accept = async () => {
  accepting.value = true
  try {
    const { data } = await shareApi.acceptShare(route.params.token)
    ElMessage.success('已添加到你的书架')
    router.push(`/book/${data.id}`)
  } catch (e) {
    if (e.response?.status === 409) {
      const bookId = e.response.data?.book_id
      ElMessage.warning('这本书已在你的书架中')
      if (bookId) router.push(`/book/${bookId}`)
    } else if (e.response?.status === 410) {
      ElMessage.error('分享已过期')
      load()
    } else if (e.response?.status === 400) {
      ElMessage.info('这是你分享的书')
      router.push('/bookshelf')
    } else {
      ElMessage.error('接收失败，请稍后重试')
    }
  } finally {
    accepting.value = false
  }
}

const goHome = () => router.push('/bookshelf')

onMounted(load)
</script>

<style scoped>
.share-preview {
  position: relative;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  background:
    radial-gradient(1200px 600px at 15% -10%, color-mix(in srgb, var(--app-primary) 16%, transparent), transparent 60%),
    radial-gradient(1000px 500px at 110% 110%, rgba(34, 211, 238, 0.14), transparent 60%),
    linear-gradient(160deg, #eef2ff 0%, #f8fafc 50%, #f0f9ff 100%);
}

.bg-glow {
  position: fixed;
  border-radius: 50%;
  filter: blur(90px);
  pointer-events: none;
  z-index: 0;
}
.bg-glow-1 {
  width: 460px;
  height: 460px;
  background: radial-gradient(circle, color-mix(in srgb, var(--app-primary) 16%, transparent), transparent 70%);
  top: -100px;
  left: -100px;
  animation: float 12s ease-in-out infinite;
}
.bg-glow-2 {
  width: 400px;
  height: 400px;
  background: radial-gradient(circle, rgba(34, 211, 238, 0.12), transparent 70%);
  bottom: -80px;
  right: -80px;
  animation: float 14s ease-in-out infinite reverse;
}
@keyframes float {
  0%, 100% { transform: translate(0, 0); }
  50% { transform: translate(30px, -20px); }
}

.share-preview > *:not(.bg-glow) {
  position: relative;
  z-index: 1;
}

/* 加载 / 错误状态 */
.share-loading,
.share-error {
  display: flex;
  align-items: center;
  justify-content: center;
}

.loading-card,
.error-card {
  width: 420px;
  max-width: 92vw;
  padding: 48px 32px;
  text-align: center;
  background: rgba(255, 255, 255, 0.9);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border: 1px solid rgba(255, 255, 255, 0.9);
  border-radius: 20px;
  box-shadow: 0 24px 64px rgba(31, 38, 135, 0.18);
}

.loading-card p {
  margin: 16px 0 0;
  color: #64748b;
}

.error-icon {
  color: #f43f5e;
}

.error-card h2 {
  margin: 16px 0 8px;
  font-size: 20px;
  color: #0f172a;
}

.error-card p {
  margin: 0 0 24px;
  font-size: 14px;
  color: #64748b;
  line-height: 1.7;
}

/* 主卡片 */
.share-card {
  display: flex;
  gap: 40px;
  width: 860px;
  max-width: 94vw;
  padding: 44px;
  background: rgba(255, 255, 255, 0.9);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border: 1px solid rgba(255, 255, 255, 0.9);
  border-radius: 22px;
  box-shadow: 0 28px 80px rgba(31, 38, 135, 0.2);
}

.share-cover-col {
  flex-shrink: 0;
  width: 240px;
}

.share-cover-inner {
  width: 100%;
}

.share-cover {
  width: 100%;
  aspect-ratio: 2/3;
  object-fit: cover;
  border-radius: 14px;
  box-shadow: 0 16px 40px rgba(31, 38, 135, 0.25);
}

.share-cover-placeholder {
  width: 100%;
  aspect-ratio: 2/3;
  background: linear-gradient(160deg, #1e293b, #0f172a);
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #94a3b8;
  box-shadow: 0 16px 40px rgba(31, 38, 135, 0.25);
}

.share-info {
  flex: 1;
  min-width: 0;
}

.share-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 14px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
  color: var(--app-primary);
  background: linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 12%, transparent), rgba(34, 211, 238, 0.12));
  border: 1px solid color-mix(in srgb, var(--app-primary) 25%, transparent);
  margin-bottom: 14px;
}

.share-title {
  margin: 0 0 8px;
  font-size: 28px;
  font-weight: 800;
  color: #0f172a;
  line-height: 1.3;
}

.share-authors {
  font-size: 15px;
  color: #64748b;
  margin-bottom: 14px;
}

.share-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 16px;
}

.type-tag {
  display: inline-flex;
  align-items: center;
  padding: 3px 12px;
  border-radius: 8px;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.5px;
  color: var(--app-primary);
  background: linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 12%, transparent), rgba(34, 211, 238, 0.12));
  border: 1px solid color-mix(in srgb, var(--app-primary) 25%, transparent);
}

.share-description {
  font-size: 14px;
  line-height: 1.9;
  color: #475569;
  margin-bottom: 18px;
  max-height: 120px;
  overflow-y: auto;
}

.share-from {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #64748b;
  margin-bottom: 22px;
}

.share-from b {
  color: var(--app-primary);
}

.expire-tip {
  color: #f59e0b;
}

.share-actions {
  margin-bottom: 16px;
}

.share-actions :deep(.el-button--primary) {
  background: linear-gradient(135deg, var(--app-primary), color-mix(in srgb, var(--app-primary) 55%, #8b5cf6));
  border: none;
  box-shadow: 0 8px 20px color-mix(in srgb, var(--app-primary) 35%, transparent);
  transition: all 0.25s ease;
}

.share-actions :deep(.el-button--primary:hover) {
  transform: translateY(-2px);
  box-shadow: 0 12px 28px color-mix(in srgb, var(--app-primary) 45%, transparent);
}

.share-tip {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  padding: 10px 14px;
  border-radius: 10px;
  font-size: 12px;
  color: #64748b;
  background: color-mix(in srgb, var(--app-primary) 6%, transparent);
  border: 1px solid color-mix(in srgb, var(--app-primary) 12%, transparent);
  line-height: 1.6;
}

/* ===== 手机端适配 ===== */
@media (max-width: 768px) {
  .share-preview {
    padding: 24px 12px;
    align-items: flex-start;
  }

  .share-card {
    flex-direction: column;
    gap: 24px;
    padding: 28px 20px;
  }

  .share-cover-col {
    width: 100%;
    display: flex;
    justify-content: center;
  }

  .share-cover-inner {
    width: 180px;
  }

  .share-title {
    font-size: 22px;
  }

  .share-actions {
    display: flex;
  }

  .share-actions .el-button {
    flex: 1;
  }
}
</style>
