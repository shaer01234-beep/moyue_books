<template>
  <div class="book-detail">
    <div class="bg-glow bg-glow-1"></div>
    <div class="bg-glow bg-glow-2"></div>

    <!-- 页面头 -->
    <div class="detail-head">
      <el-button text class="back-btn" @click="goBack">
        <el-icon><ArrowLeft /></el-icon> 返回
      </el-button>
      <div class="detail-title-wrap">
        <div class="detail-icon">
          <el-icon :size="20"><Reading /></el-icon>
        </div>
        <div class="detail-text">
          <h2>{{ book?.title || '图书详情' }}</h2>
          <p v-if="book">{{ formatAuthors(book.authors) }}</p>
        </div>
      </div>
    </div>

    <div v-if="!book" class="loading-state">
      <el-skeleton :rows="8" animated />
    </div>
    <el-row :gutter="24" v-else>
      <!-- 封面 -->
      <el-col :xs="24" :sm="8" :md="6">
        <div class="detail-cover-wrapper">
          <div class="detail-cover-inner">
            <img
              v-if="book.cover_path"
              :src="coverUrl(book.id)"
              class="detail-cover"
              alt="cover"
            />
            <div v-else class="detail-cover-placeholder">
              <el-icon :size="64"><Reading /></el-icon>
            </div>
            <div class="cover-change-hint" @click="triggerCoverUpload">
              <el-icon :size="15"><Camera /></el-icon>
              <span>更换封面</span>
            </div>
          </div>
          <input
            ref="coverInput"
            type="file"
            accept=".jpg,.jpeg,.png"
            style="display: none"
            @change="handleCoverChange"
          />
        </div>
      </el-col>

      <!-- 信息 -->
      <el-col :xs="24" :sm="16" :md="18">
        <div class="detail-info">
          <div class="detail-authors">{{ formatAuthors(book.authors) }}</div>

          <div class="detail-meta">
            <span v-if="book.file_type" class="type-tag">{{ book.file_type.toUpperCase() }}</span>
            <el-tag v-if="book.publisher" size="small" type="info" effect="plain">{{ book.publisher }}</el-tag>
            <el-tag v-if="book.publish_date" size="small" type="info" effect="plain">{{ book.publish_date }}</el-tag>
            <el-tag v-if="book.language" size="small" type="info" effect="plain">{{ book.language }}</el-tag>
            <el-tag v-if="book.isbn" size="small" type="info" effect="plain">ISBN: {{ book.isbn }}</el-tag>
          </div>

          <div class="detail-tags" v-if="tags.length > 0">
            <el-tag
              v-for="tag in tags"
              :key="tag"
              size="small"
              effect="plain"
              style="margin-right: 6px"
            >
              {{ tag }}
            </el-tag>
          </div>

          <p v-if="book.description" class="detail-description">{{ book.description }}</p>

          <div class="detail-progress" v-if="book.progress > 0">
            <span class="progress-label">阅读进度</span>
            <el-progress
              :percentage="Math.round(book.progress)"
              class="detail-progress-bar"
              :stroke-width="10"
              color="linear-gradient(90deg,var(--app-primary),#22d3ee)"
            />
            <span v-if="book.current_page > 0" class="progress-page">第 {{ book.current_page }} 页</span>
          </div>

          <div class="detail-collections" v-if="bookCollections.length > 0">
            <span class="collections-label">
              <el-icon :size="14"><StarFilled /></el-icon>
              已收藏到
            </span>
            <el-tag
              v-for="c in bookCollections"
              :key="c.id"
              size="small"
              class="collection-tag"
            >
              {{ c.name }}
            </el-tag>
          </div>

          <div class="detail-actions">
            <el-button
              v-if="book.file_type === 'epub' || book.file_type === 'kepub' || book.file_type === 'pdf' || book.file_type === 'azw3' || book.file_type === 'mobi'"
              type="primary"
              size="large"
              @click="goRead"
            >
              <el-icon><Reading /></el-icon> 阅读
            </el-button>
            <el-button
              size="large"
              :class="{ 'collect-active': bookCollections.length > 0 }"
              @click="openCollectDialog"
            >
              <el-icon>
                <StarFilled v-if="bookCollections.length > 0" />
                <Star v-else />
              </el-icon>
              {{ bookCollections.length > 0 ? '已收藏' : '收藏' }}
            </el-button>
            <el-button size="large" @click="downloadBook">
              <el-icon><Download /></el-icon> 下载
            </el-button>
            <el-button size="large" @click="openShareDialog">
              <el-icon><Share /></el-icon> 分享
            </el-button>
            <el-button size="large" @click="editDialogVisible = true">
              <el-icon><Edit /></el-icon> 编辑
            </el-button>
            <el-button size="large" type="danger" plain @click="deleteBook">
              <el-icon><Delete /></el-icon> 删除
            </el-button>
          </div>
        </div>
      </el-col>
    </el-row>

    <!-- 编辑对话框 -->
    <el-dialog
      v-model="editDialogVisible"
      title="编辑图书信息"
      width="600px"
      align-center
      append-to-body
      class="detail-dialog"
    >
      <el-form :model="editForm" label-width="100px">
        <el-form-item label="书名">
          <el-input v-model="editForm.title" />
        </el-form-item>
        <el-form-item label="作者">
          <el-input v-model="editForm.authors" placeholder="多个作者用逗号分隔" />
        </el-form-item>
        <el-form-item label="出版社">
          <el-input v-model="editForm.publisher" />
        </el-form-item>
        <el-form-item label="出版日期">
          <el-input v-model="editForm.publish_date" />
        </el-form-item>
        <el-form-item label="语言">
          <el-input v-model="editForm.language" />
        </el-form-item>
        <el-form-item label="ISBN">
          <el-input v-model="editForm.isbn" />
        </el-form-item>
        <el-form-item label="标签">
          <el-input v-model="editForm.tags" placeholder="多个标签用逗号分隔" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="editForm.description" type="textarea" :rows="4" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveEdit">保存</el-button>
      </template>
    </el-dialog>

    <!-- 收藏到收藏夹对话框 -->
    <el-dialog
      v-model="collectVisible"
      title="收藏到收藏夹"
      width="480px"
      align-center
      append-to-body
      class="detail-dialog"
    >
      <div class="collect-create">
        <el-input
          v-model="newCollectName"
          placeholder="新建收藏夹名称"
          maxlength="60"
          clearable
          @keyup.enter="createNewCollection"
        />
        <el-button
          type="primary"
          :disabled="!newCollectName.trim()"
          :loading="creatingCollect"
          @click="createNewCollection"
        >
          <el-icon><Plus /></el-icon> 新建
        </el-button>
      </div>
      <div class="collect-tip">选择要收藏的收藏夹（可多选）：</div>
      <div v-loading="loadingCollects" class="collect-list">
        <el-checkbox-group v-model="selectedCollections" class="collect-group">
          <el-checkbox
            v-for="c in allCollections"
            :key="c.id"
            :value="c.id"
            class="collect-item"
          >
            <span class="collect-name">{{ c.name }}</span>
            <span class="collect-count">{{ c.book_count }} 本</span>
          </el-checkbox>
        </el-checkbox-group>
        <div v-if="!loadingCollects && allCollections.length === 0" class="collect-empty">
          还没有收藏夹，先在上面新建一个
        </div>
      </div>
      <template #footer>
        <el-button @click="collectVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingCollect" @click="saveCollect">保存</el-button>
      </template>
    </el-dialog>

    <!-- 分享对话框 -->
    <el-dialog
      v-model="shareVisible"
      title="分享图书"
      width="560px"
      align-center
      append-to-body
      class="detail-dialog"
    >
      <div class="share-desc">
        <el-icon :size="16"><Share /></el-icon>
        <span>生成分享链接，好友打开链接即可查看并接收这本书到自己的书架。</span>
      </div>

      <!-- 创建分享 -->
      <div class="share-create">
        <el-radio-group v-model="shareExpiresDays" size="default">
          <el-radio-button :value="0">永久有效</el-radio-button>
          <el-radio-button :value="7">7 天</el-radio-button>
          <el-radio-button :value="30">30 天</el-radio-button>
        </el-radio-group>
        <el-button
          type="primary"
          :loading="creatingShare"
          :disabled="shares.length >= 5"
          @click="createShare"
        >
          <el-icon><Plus /></el-icon> 生成分享链接
        </el-button>
      </div>

      <!-- 分享列表 -->
      <div v-loading="loadingShares" class="share-list">
        <div v-if="!loadingShares && shares.length === 0" class="share-empty">
          还没有分享链接
        </div>
        <div v-for="s in shares" :key="s.id" class="share-item">
          <div class="share-item-info">
            <div class="share-item-link">{{ shareUrl(s.token) }}</div>
            <div class="share-item-meta">
              <el-tag v-if="s.expires_at" size="small" type="warning" effect="plain">
                至 {{ formatDate(s.expires_at) }}
              </el-tag>
              <el-tag v-else size="small" type="success" effect="plain">永久有效</el-tag>
              <span class="share-item-time">{{ formatDate(s.created_at) }} 创建</span>
            </div>
          </div>
          <div class="share-item-actions">
            <el-tooltip content="复制链接" placement="top">
              <el-button size="small" circle @click="copyShare(s.token)">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
            </el-tooltip>
            <el-tooltip content="撤销分享" placement="top">
              <el-button size="small" circle type="danger" plain @click="revokeShare(s)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </el-tooltip>
          </div>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onActivated, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getProgressUpdate, wasRecentlyUpdated, consumeProgressUpdate } from '@/utils/progressSync'
import { bookApi } from '@/api/book'
import { getShareConfig, shareApi } from '@/api/share'
import { copyText } from '@/utils/clipboard'

const props = defineProps({ id: String })
const route = useRoute()
const router = useRouter()

const book = ref(null)
const loading = ref(false)
const editDialogVisible = ref(false)
const coverInput = ref(null)

// ===== 收藏到收藏夹 =====
const collectVisible = ref(false)
const allCollections = ref([])
const selectedCollections = ref([])
const newCollectName = ref('')
const creatingCollect = ref(false)
const savingCollect = ref(false)
const loadingCollects = ref(false)
const bookCollections = ref([])

// 回显图书所属收藏夹
const loadBookCollections = async () => {
  try {
    const { data } = await bookApi.getBookCollections(props.id)
    bookCollections.value = data.data || []
  } catch (e) {
    bookCollections.value = []
  }
}

const openCollectDialog = async () => {
  selectedCollections.value = []
  allCollections.value = []
  newCollectName.value = ''
  collectVisible.value = true
  loadingCollects.value = true
  try {
    const { data } = await bookApi.listCollections()
    allCollections.value = data.data || []
  } catch (e) {
    allCollections.value = []
  } finally {
    loadingCollects.value = false
  }
  try {
    const { data } = await bookApi.getBookCollections(props.id)
    selectedCollections.value = (data.data || []).map((c) => c.id)
  } catch (e) { /* 忽略 */ }
}

const createNewCollection = async () => {
  const name = newCollectName.value.trim()
  if (!name) return
  creatingCollect.value = true
  try {
    const { data } = await bookApi.createCollection({ name })
    allCollections.value.push({ ...data, book_count: 0 })
    selectedCollections.value.push(data.id)
    newCollectName.value = ''
    ElMessage.success('收藏夹已创建')
  } catch (e) {
    ElMessage.error('创建收藏夹失败')
  } finally {
    creatingCollect.value = false
  }
}

const saveCollect = async () => {
  savingCollect.value = true
  try {
    await bookApi.setBookCollections(props.id, selectedCollections.value)
    ElMessage.success('已保存收藏')
    collectVisible.value = false
    loadBookCollections()
  } catch (e) {
    ElMessage.error('保存失败')
  } finally {
    savingCollect.value = false
  }
}

// ===== 分享 =====
const shareVisible = ref(false)
const shares = ref([])
const shareExpiresDays = ref(7)
const creatingShare = ref(false)
const loadingShares = ref(false)

const loadShares = async () => {
  loadingShares.value = true
  try {
    const { data } = await shareApi.listShares(props.id)
    shares.value = data.data || []
  } catch (e) {
    shares.value = []
  } finally {
    loadingShares.value = false
  }
}

const openShareDialog = () => {
  shareVisible.value = true
  shareExpiresDays.value = 7
  loadServicePort()
  loadShares()
}

// 服务端口（飞牛 service_port）：分享链接用端口直连，不走统一网关
const servicePort = ref('')

const loadServicePort = async () => {
  try {
    const { data } = await getShareConfig()
    servicePort.value = String(data.port || '')
  } catch (e) {
    servicePort.value = ''
  }
}

// 分享链接基础：优先端口直连（http://NAS_IP:service_port/#/share/xxx）
const shareBase = () => {
  const p = servicePort.value
  const { protocol, hostname, port, origin } = window.location
  if (p) {
    // 当前已在服务端口直连访问（开发环境 / 直连部署），直接用同源
    if (port === p || (!port && (p === '80' || p === '443'))) return origin
    // 飞牛统一网关场景：用 hostname + 服务端口直连
    return `${protocol}//${hostname}:${p}`
  }
  // 端口信息未获取到时回退：同源 + 网关前缀
  return `${origin}${import.meta.env.BASE_URL}`
}

// 如 http://NAS_IP:service_port/#/share/xxx（端口直连，不走统一网关）
const shareUrl = (token) => `${shareBase()}#/share/${token}`

// 从 axios 错误中提取后端返回的具体错误信息
const extractError = (e, fallback) => {
  if (e?.response?.status === 404 && (typeof e.response.data === 'string' || !e.response.data?.error)) {
    return '分享接口不可用，请确认后端服务已更新并重启'
  }
  const detail = e?.response?.data
  const msg = (typeof detail === 'string' && detail) ? detail : detail?.error
  return msg || fallback
}

const createShare = async () => {
  if (!props.id) {
    ElMessage.error('书籍 ID 无效，请从书架重新进入本书')
    return
  }
  creatingShare.value = true
  try {
    const { data } = await shareApi.createShare(props.id, shareExpiresDays.value)
    shares.value.unshift(data)
    copyShare(data.token)
  } catch (e) {
    ElMessage.error(extractError(e, '生成分享链接失败'))
  } finally {
    creatingShare.value = false
  }
}

const copyShare = async (token) => {
  const url = shareUrl(token)
  const ok = await copyText(url)
  if (ok) {
    ElMessage.success('链接已复制到剪贴板')
  } else {
    window.prompt('浏览器限制自动复制，请手动复制链接：', url)
  }
}

const revokeShare = async (s) => {
  try {
    await ElMessageBox.confirm('撤销后该链接将立即失效，确定撤销这条分享吗？', '撤销分享', { type: 'warning', confirmButtonText: '确定', cancelButtonText: '取消' })
    await shareApi.revokeShare(s.token)
    shares.value = shares.value.filter((x) => x.token !== s.token)
    ElMessage.success('已撤销分享')
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(extractError(e, '撤销失败'))
  }
}

const formatDate = (d) => {
  const date = new Date(d)
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

const editForm = ref({})

const tags = computed(() => {
  if (!book.value || !book.value.tags) return []
  try {
    const parsed = JSON.parse(book.value.tags)
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
})

const coverUrl = (id) => bookApi.coverUrl(id)

const formatAuthors = (authors) => {
  if (!authors) return '未知作者'
  try {
    const parsed = JSON.parse(authors)
    return Array.isArray(parsed) && parsed.length > 0 ? parsed.join(', ') : '未知作者'
  } catch {
    return '未知作者'
  }
}

const loadBook = async () => {
  loading.value = true
  try {
    const { data } = await bookApi.getBook(props.id)
    book.value = data
    editForm.value = {
      title: data.title,
      authors: formatAuthors(data.authors),
      publisher: data.publisher || '',
      publish_date: data.publish_date || '',
      language: data.language || '',
      isbn: data.isbn || '',
      tags: (() => { try { return JSON.parse(data.tags || '[]').join(', ') } catch { return '' } })(),
      description: data.description || ''
    }
    // 应用本地进度更新（如果阅读器刚保存过且本次加载返回了旧数据）
    const local = getProgressUpdate(props.id)
    if (local && local.updated_at && (!data.progress || data.progress < local.progress)) {
      book.value = { ...book.value, progress: local.progress, current_page: local.current_page }
    }
  } catch (e) {
    ElMessage.error('加载图书详情失败')
  } finally {
    loading.value = false
  }
}

const goRead = () => router.push(`/reader/${props.id}`)

// 返回上一页：用 router.beforeEach 记录的"来时路径"直接 replace，
// 避开刷新导致的重复历史项（否则可能落进重复 reader entry 而非真正的上一页）
const goBack = () => {
  let prev = ''
  try { prev = sessionStorage.getItem('moyue_prev_bookdetail_path') || '' } catch {}
  // 安全过滤：必须是站内、非详情页、非阅读页、非登录页（避免被误写时进入死循环）
  if (prev && prev.startsWith('/') && !prev.startsWith('/book/') && !prev.startsWith('/reader/') && prev !== '/login') {
    router.replace(prev)
  } else {
    router.replace('/bookshelf')
  }
}

const saveEdit = async () => {
  try {
    await bookApi.updateBook(props.id, {
      title: editForm.value.title,
      authors: editForm.value.authors.split(/[,，]/).map(s => s.trim()).filter(Boolean),
      publisher: editForm.value.publisher,
      publish_date: editForm.value.publish_date,
      language: editForm.value.language,
      isbn: editForm.value.isbn,
      tags: editForm.value.tags.split(/[,，]/).map(s => s.trim()).filter(Boolean),
      description: editForm.value.description
    })
    ElMessage.success('保存成功')
    editDialogVisible.value = false
    loadBook()
  } catch (e) {
    ElMessage.error('保存失败')
  }
}

const deleteBook = async () => {
  try {
    await ElMessageBox.confirm('确定要删除这本图书吗？此操作不可恢复。', '提示', { type: 'warning', confirmButtonText: '确定', cancelButtonText: '取消' })
    await bookApi.deleteBook(props.id)
    ElMessage.success('删除成功')
    router.push('/bookshelf')
  } catch (e) {
    if (e !== 'cancel') ElMessage.error('删除失败')
  }
}

const downloadBook = async () => {
  try {
    const { data, headers } = await bookApi.downloadBook(props.id)
    const blob = new Blob([data])
    const link = document.createElement('a')
    link.href = URL.createObjectURL(blob)
    const filename = headers['content-disposition']?.match(/filename="(.+)"/)?.[1] || 'book'
    link.download = filename
    link.click()
  } catch (e) {
    ElMessage.error('下载失败')
  }
}

const triggerCoverUpload = () => coverInput.value?.click()

const handleCoverChange = async (e) => {
  const file = e.target.files[0]
  if (!file) return
  try {
    await bookApi.uploadCover(props.id, file)
    ElMessage.success('封面已更新')
    loadBook()
  } catch (err) {
    ElMessage.error('封面上传失败')
  }
  e.target.value = ''
}

onMounted(() => {
  loadBook()
  loadBookCollections()
})

// 路由参数变化（如 /book/A → /book/B）时重新加载
watch(() => props.id, () => {
  loadBook()
  loadBookCollections()
})

// keep-alive 激活时：从阅读器返回后主动拉取最新进度
onActivated(() => {
  if (wasRecentlyUpdated(props.id)) {
    loadBook()
    consumeProgressUpdate(props.id)
  }
})
</script>

<style scoped>
.book-detail {
  position: relative;
  max-width: 1200px;
  margin: 0 auto;
  background: rgba(255, 255, 255, 0.88);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  border: 1px solid rgba(255, 255, 255, 0.7);
  box-shadow: 0 12px 40px rgba(31, 38, 135, 0.1);
  padding: 36px;
  border-radius: 20px;
  overflow: hidden;
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

.book-detail > *:not(.bg-glow) {
  position: relative;
  z-index: 1;
}

/* ===== 页面头 ===== */
.detail-head {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 24px;
  padding-bottom: 18px;
  border-bottom: 1px solid color-mix(in srgb, var(--app-primary) 10%, rgba(31, 38, 135, 0.06));
}
.back-btn {
  flex-shrink: 0;
  color: #64748b;
  font-weight: 600;
  padding-left: 8px;
  padding-right: 8px;
}
.back-btn:hover {
  color: var(--app-primary);
}
.detail-title-wrap {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 14px;
}
.detail-icon {
  flex-shrink: 0;
  width: 44px;
  height: 44px;
  border-radius: 13px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  background: var(--app-gradient);
  box-shadow: 0 8px 20px color-mix(in srgb, var(--app-primary) 35%, transparent);
}
.detail-text {
  min-width: 0;
}
.detail-text h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 700;
  color: #0f172a;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.detail-text p {
  margin: 2px 0 0;
  font-size: 13px;
  color: #64748b;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.detail-cover-wrapper {
  text-align: center;
}

.detail-cover-inner {
  position: relative;
  display: inline-block;
  width: 100%;
  max-width: 280px;
  border-radius: 14px;
}

.cover-change-hint {
  position: absolute;
  left: 50%;
  bottom: 12px;
  z-index: 2;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 7px 16px;
  border-radius: 999px;
  font-size: 13px;
  font-weight: 600;
  letter-spacing: 0.3px;
  color: #fff;
  cursor: pointer;
  background: linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 92%, transparent), rgba(34, 211, 238, 0.86));
  border: 1px solid rgba(255, 255, 255, 0.4);
  box-shadow: 0 8px 20px rgba(31, 38, 135, 0.35);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
  opacity: 0;
  transform: translate(-50%, 10px);
  transition: opacity 0.28s ease, transform 0.28s ease;
  white-space: nowrap;
}

.detail-cover-inner:hover .cover-change-hint,
.detail-cover-inner:focus-within .cover-change-hint {
  opacity: 1;
  transform: translate(-50%, 0);
}

.cover-change-hint:hover {
  background: var(--app-gradient);
  box-shadow: 0 12px 26px color-mix(in srgb, var(--app-primary) 45%, transparent);
}

.detail-cover {
  width: 100%;
  max-width: 280px;
  aspect-ratio: 2/3;
  object-fit: cover;
  border-radius: 14px;
  box-shadow: 0 16px 40px rgba(31, 38, 135, 0.25);
  transition: transform 0.3s ease, box-shadow 0.3s ease;
}

.detail-cover:hover {
  transform: translateY(-4px) scale(1.02);
  box-shadow: 0 24px 56px color-mix(in srgb, var(--app-primary) 35%, transparent);
}

.detail-cover-placeholder {
  width: 100%;
  max-width: 280px;
  aspect-ratio: 2/3;
  background: linear-gradient(160deg, #1e293b, #0f172a);
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #94a3b8;
  margin: 0 auto;
  box-shadow: 0 16px 40px rgba(31, 38, 135, 0.25);
}

.detail-info {
  padding-top: 8px;
}

/* ===== 收藏回显 ===== */
.detail-collections {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 20px;
  padding: 12px 16px;
  background: linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 7%, transparent), rgba(34, 211, 238, 0.05));
  border: 1px solid color-mix(in srgb, var(--app-primary) 15%, transparent);
  border-radius: 14px;
}

.collections-label {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 13px;
  font-weight: 600;
  color: var(--app-primary);
}

.collection-tag {
  margin-right: 0;
  border-radius: 8px;
  color: var(--app-primary) !important;
  background: color-mix(in srgb, var(--app-primary) 10%, transparent) !important;
  border-color: color-mix(in srgb, var(--app-primary) 25%, transparent) !important;
}

/* 收藏按钮已收藏状态 */
.detail-actions :deep(.el-button.collect-active) {
  background: var(--app-gradient);
  border: none;
  color: #fff;
  box-shadow: 0 8px 20px color-mix(in srgb, var(--app-primary) 35%, transparent);
}

.detail-actions :deep(.el-button.collect-active:hover) {
  color: #fff;
  transform: translateY(-2px);
  box-shadow: 0 12px 28px color-mix(in srgb, var(--app-primary) 45%, transparent);
}

.detail-title {
  margin: 0 0 12px 0;
  font-size: 30px;
  color: #0f172a;
  font-weight: 800;
  line-height: 1.3;
}

.detail-authors {
  font-size: 16px;
  color: #64748b;
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

.detail-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 16px;
}

.detail-tags {
  margin-bottom: 16px;
}

.detail-description {
  font-size: 14px;
  line-height: 1.9;
  color: #475569;
  margin-bottom: 24px;
}

.detail-progress {
  display: flex;
  align-items: center;
  margin-bottom: 24px;
  padding: 18px 20px;
  background: linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 7%, transparent), rgba(34, 211, 238, 0.05));
  border: 1px solid color-mix(in srgb, var(--app-primary) 15%, transparent);
  border-radius: 14px;
}

.progress-label {
  font-weight: 600;
  color: #334155;
}

.detail-progress-bar {
  width: 300px;
  margin-left: 12px;
}

.progress-page {
  margin-left: 12px;
  font-size: 13px;
  color: var(--app-primary);
  font-weight: 600;
}

.detail-actions {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.detail-actions :deep(.el-button--primary) {
  background: linear-gradient(135deg, var(--app-primary), color-mix(in srgb, var(--app-primary) 55%, #8b5cf6));
  border: none;
  box-shadow: 0 8px 20px color-mix(in srgb, var(--app-primary) 35%, transparent);
  transition: all 0.25s ease;
}

.detail-actions :deep(.el-button--primary:hover) {
  transform: translateY(-2px);
  box-shadow: 0 12px 28px color-mix(in srgb, var(--app-primary) 45%, transparent);
}

/* ===== 收藏对话框 ===== */
.collect-create {
  display: flex;
  gap: 10px;
  margin-bottom: 14px;
}
.collect-create .el-input {
  flex: 1;
}
.collect-tip {
  font-size: 13px;
  color: #64748b;
  margin-bottom: 10px;
}
.collect-list {
  min-height: 60px;
  max-height: 260px;
  overflow-y: auto;
}
.collect-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.collect-item {
  width: 100%;
  height: 42px;
  margin-right: 0;
  padding: 0 12px;
  border-radius: 10px;
  transition: background 0.2s ease;
}
.collect-item:hover {
  background: color-mix(in srgb, var(--app-primary) 6%, transparent);
}
.collect-name {
  font-weight: 500;
  color: #1e293b;
}
.collect-count {
  float: right;
  font-size: 12px;
  color: #94a3b8;
  margin-left: 12px;
}
.collect-empty {
  padding: 20px;
  text-align: center;
  font-size: 13px;
  color: #94a3b8;
}

/* ===== 分享弹窗 ===== */
.share-desc {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 12px 14px;
  margin-bottom: 16px;
  border-radius: 12px;
  font-size: 13px;
  line-height: 1.7;
  color: #475569;
  background: linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 7%, transparent), rgba(34, 211, 238, 0.05));
  border: 1px solid color-mix(in srgb, var(--app-primary) 15%, transparent);
}

.share-desc .el-icon {
  margin-top: 3px;
  flex-shrink: 0;
  color: var(--app-primary);
}

.share-create {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 16px;
}

.share-list {
  min-height: 60px;
  max-height: 300px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.share-empty {
  padding: 24px;
  text-align: center;
  font-size: 13px;
  color: #94a3b8;
  border: 1px dashed color-mix(in srgb, var(--app-primary) 25%, transparent);
  border-radius: 12px;
}

.share-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  border-radius: 12px;
  background: color-mix(in srgb, var(--app-primary) 4%, transparent);
  border: 1px solid color-mix(in srgb, var(--app-primary) 12%, transparent);
  transition: background 0.2s ease;
}

.share-item:hover {
  background: color-mix(in srgb, var(--app-primary) 8%, transparent);
}

.share-item-info {
  min-width: 0;
  flex: 1;
}

.share-item-link {
  font-size: 13px;
  font-weight: 500;
  color: var(--app-primary);
  word-break: break-all;
  margin-bottom: 6px;
}

.share-item-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.share-item-time {
  font-size: 12px;
  color: #94a3b8;
}

.share-item-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.share-item-actions .el-button + .el-button {
  margin-left: 0;
}


/* 无 hover 能力设备（触屏）封面按钮常显 */
@media (hover: none) {
  .cover-change-hint {
    opacity: 1;
    transform: translate(-50%, 0);
  }
}

/* ===== 手机端适配 ===== */
@media (max-width: 768px) {
  .book-detail {
    padding: 20px 16px;
    border-radius: 16px;
  }

  .detail-cover-inner {
    max-width: 200px;
  }

  .detail-cover {
    max-width: 200px;
  }

  .detail-cover-placeholder {
    max-width: 200px;
  }

  .detail-head {
    gap: 10px;
    margin-bottom: 18px;
    padding-bottom: 14px;
  }

  .detail-icon {
    width: 38px;
    height: 38px;
    border-radius: 11px;
  }

  .detail-text h2 {
    font-size: 17px;
  }

  .detail-text p {
    font-size: 12px;
  }

  .detail-authors {
    font-size: 14px;
  }

  .detail-description {
    font-size: 13px;
  }

  .detail-progress {
    flex-wrap: wrap;
    padding: 14px;
  }

  .detail-progress-bar {
    width: 100%;
    margin-left: 0;
    margin-top: 10px;
  }

  .progress-page {
    width: 100%;
    margin-left: 0;
    margin-top: 8px;
  }

  .detail-actions {
    position: fixed;
    left: 0;
    right: 0;
    /* bottom: calc(62px + env(safe-area-inset-bottom)); */
    z-index: 990;
    display: flex;
    flex-direction: row;
    flex-wrap: wrap;
    align-items: stretch;
    justify-content: flex-start;
    gap: 8px;
    padding: 10px 12px calc(10px + env(safe-area-inset-bottom));
    background: rgba(255, 255, 255, 0.92);
    backdrop-filter: blur(20px);
    -webkit-backdrop-filter: blur(20px);
    border-top: 1px solid rgba(255, 255, 255, 0.8);
    box-shadow: 0 -8px 30px rgba(31, 38, 135, 0.12);
    animation: detailBarIn 0.35s cubic-bezier(0.22, 1, 0.36, 1) both;
  }

  @keyframes detailBarIn {
    from {
      opacity: 0;
      transform: translateY(100%);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .detail-actions .el-button {
    margin-left: 0 !important;
    flex: 1 1 0;
    min-width: 0;
    padding: 14px 4px;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 4px;
    height: auto;
    font-size: 11px;
    border-radius: 12px;
  }

  /* 阅读（首个按钮）独占整行，保证小屏始终可见且突出 */
  .detail-actions .el-button:first-child {
    flex: 1 0 100%;
    flex-direction: row;
    gap: 6px;
    font-size: 15px;
    font-weight: 700;
    padding: 14px;
  }

  .detail-actions .el-button .el-icon {
    margin: 0;
    font-size: 18px;
  }

  /* 详情页底部预留空间：专属操作栏（两行）+ 全局底部导航两层 */
  .book-detail {
    padding-bottom: calc(156px + 2 * env(safe-area-inset-bottom));
  }

  /* 编辑表单：标签置顶 */
  .book-detail :deep(.el-form-item) {
    display: block;
    margin-bottom: 16px;
  }

  .book-detail :deep(.el-form-item__label) {
    display: block;
    width: auto !important;
    text-align: left;
    padding: 0 0 6px;
    line-height: 1.4;
  }

  .book-detail :deep(.el-form-item__content) {
    margin-left: 0 !important;
  }
}
</style>
