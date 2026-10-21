<template>
  <div class="upload-page">
    <div class="bg-glow bg-glow-1"></div>
    <div class="bg-glow bg-glow-2"></div>

    <!-- 导入时归类分类 -->
    <div class="import-cat-bar cardIn">
      <span class="import-cat-label">归类分类</span>
      <el-select
        v-model="category"
        placeholder="不指定（导入后分类为空）"
        clearable
        filterable
        allow-create
        default-first-option
        style="width: 280px"
      >
        <el-option v-for="c in categoryOptions" :key="c" :label="c" :value="c" />
      </el-select>
      <span class="form-hint">导入的图书将归入所选分类</span>
    </div>

    <div class="upload-card cardIn">
      <!-- 导入方式卡片（眉头下的地块） -->
      <div class="format-list">
        <div class="format-item"><span class="format-dot dot-1"></span><strong>电子书格式：</strong>EPUB、KEPUB、PDF、AZW3、MOBI、MD、TXT</div>
      </div>

      <!-- 拖拽上传区域 -->
      <el-upload
        ref="uploadRef"
        drag
        :action="API_BASE + '/books'"
        :auto-upload="false"
        :on-change="handleChange"
        :on-remove="handleRemove"
        :file-list="fileList"
        multiple
        accept=".epub,.kepub,.pdf,.azw3,.mobi,.md,.txt"
      >
        <el-icon class="el-icon--upload" :size="60"><UploadFilled /></el-icon>
        <div class="el-upload__text">
          将文件拖拽到此处，或 <em>点击上传</em>
        </div>
        <template #tip>
          <div class="el-upload__tip">
            支持 EPUB / KEPUB / PDF / AZW3 / MOBI，以及 Markdown(.md) 与纯文本(.txt)，单个文件不超过 500MB
          </div>
        </template>
      </el-upload>

      <!-- 文件列表和操作 -->
      <div v-if="fileList.length > 0" class="upload-actions cardIn" style="--i: 2">
        <div class="file-count">已选择 {{ fileList.length }} 个文件</div>
        <div class="upload-btn-row">
          <el-button
            type="primary"
            size="large"
            :loading="uploading"
            @click="startUpload"
            class="upload-btn"
          >
            {{ uploading ? `上传中 (${uploadedCount}/${fileList.length})` : '开始上传' }}
          </el-button>
          <el-button size="large" @click="clearAll" class="clear-btn">清空</el-button>
        </div>
      </div>

      <!-- 或从文件夹扫描导入 -->
      <div class="scan-section" ref="scanRef">
        <div class="scan-head">
          <div class="scan-title">
            <span class="scan-icon"><el-icon><FolderOpened /></el-icon></span>
            <span>扫描文件夹导入</span>
            <span class="scan-tag">批量</span>
          </div>
          <div class="scan-desc">扫描服务器指定目录下的电子书文件，自动批量导入书库</div>
        </div>

        <div class="scan-form">
          <el-input
            v-model="scanPath"
            class="scan-input"
            placeholder="输入文件夹路径，如 /Users/me/Books"
            clearable
            :disabled="scanning"
            @keyup.enter="startScan"
          >
            <template #prefix>
              <el-icon><Folder /></el-icon>
            </template>
          </el-input>
          <el-button
            type="primary"
            class="scan-btn"
            :loading="scanning"
            :disabled="!scanPath.trim()"
            @click="startScan"
          >
            {{ scanning ? '扫描中…' : '开始扫描' }}
          </el-button>
        </div>

        <!-- 扫描进度 -->
        <div v-if="scanning || scanDone" class="scan-progress">
          <el-progress
            :percentage="scanPercent"
            :stroke-width="10"
            :status="scanDone && scanFailed === 0 ? 'success' : (scanDone && scanFailed > 0 ? 'warning' : undefined)"
            class="scan-progress-bar"
          />
          <div class="scan-stats">
            <span class="stat-item">共 <b>{{ scanTotal }}</b> 个</span>
            <span class="stat-item stat-ok"><el-icon><CircleCheck /></el-icon> 成功 <b>{{ scanSuccess }}</b></span>
            <span class="stat-item stat-skip"><el-icon><RefreshRight /></el-icon> 跳过 <b>{{ scanExists }}</b></span>
            <span class="stat-item stat-err"><el-icon><CircleClose /></el-icon> 失败 <b>{{ scanFailed }}</b></span>
          </div>
          <div v-if="scanCurrent" class="scan-current" :title="scanCurrent">
            正在导入：{{ scanCurrent }}
          </div>
          <div v-if="scanMessage" class="scan-done-msg">{{ scanMessage }}</div>
        </div>
      </div>

      <!-- 上传进度 -->
      <div v-if="uploading" class="progress-list">
        <div
          v-for="item in uploadResults"
          :key="item.filename"
          class="progress-item"
          :class="{ success: item.success, error: !item.success }"
        >
          <el-icon v-if="item.success" class="status-icon"><CircleCheck /></el-icon>
          <el-icon v-else class="status-icon"><CircleClose /></el-icon>
          <span class="filename">{{ item.filename }}</span>
          <span v-if="!item.success" class="error-msg">{{ item.error }}</span>
        </div>
      </div>

      <!-- 最近上传 -->
      <div v-if="recentBooks.length > 0" class="recent-section">
        <h3><span class="recent-icon"><el-icon><Clock /></el-icon></span> 最近上传</h3>
        <el-row :gutter="16">
          <el-col
            v-for="(book, index) in displayRecent"
            :key="book.id"
            :xs="12" :sm="8" :md="6" :lg="4"
          >
            <BookCard :book="book" :index="index + 3" @click="goToBook(book.id)" />
          </el-col>
        </el-row>
        <div v-if="filteredRecent.length > recentLimit" class="recent-more">
          <el-button text type="primary" @click="recentExpanded = !recentExpanded">
            {{ recentExpanded ? '收起' : `查看全部 ${filteredRecent.length} 本` }}
          </el-button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { bookApi } from '@/api/book'

// 最近导入搜索框已上移至媒体库 tab 条右侧，由父组件（LibraryHub）通过 prop 传入
const props = defineProps({
  recentQuery: { type: String, default: '' }
})
import { getToken } from '@/api/auth'
import { API_BASE } from '@/config'
import BookCard from '@/components/BookCard.vue'

const router = useRouter()
const uploadRef = ref(null)
const scanRef = ref(null)
const fileList = ref([])

// 导入时归类分类
const category = ref('')
const categoryOptions = ref([])
const loadCategories = async () => {
  try {
    const { data } = await bookApi.getCategories()
    categoryOptions.value = (data.data || []).map(c => c.category).filter(Boolean)
  } catch (e) { /* 分类加载失败不影响主流程 */ }
}

const uploading = ref(false)
const uploadedCount = ref(0)
const uploadResults = ref([])
const recentBooks = ref([])

const coverUrl = (id) => bookApi.coverUrl(id)

// 最近导入搜索过滤（搜索框位于媒体库 tab 条右侧，经 prop 传入）
const filteredRecent = computed(() => {
  const q = (props.recentQuery || '').trim().toLowerCase()
  if (!q) return recentBooks.value
  return recentBooks.value.filter(b => (b.title || '').toLowerCase().includes(q))
})

// 最近上传：按页面宽度计算每行列数（与下方 el-col 断点一致），只显示两行
const winWidth = ref(window.innerWidth)
const recentColumns = computed(() => {
  const w = winWidth.value
  if (w >= 1200) return 6   // lg：span 4
  if (w >= 992) return 4    // md：span 6
  if (w >= 768) return 3    // sm：span 8
  return 2                  // xs：span 12
})
const recentLimit = computed(() => recentColumns.value * 2)
const recentExpanded = ref(false)
const displayRecent = computed(() =>
  recentExpanded.value ? filteredRecent.value : filteredRecent.value.slice(0, recentLimit.value)
)
let recentResizeTimer = null
const onRecentResize = () => {
  if (recentResizeTimer) clearTimeout(recentResizeTimer)
  recentResizeTimer = setTimeout(() => { winWidth.value = window.innerWidth }, 150)
}

const handleChange = (file, files) => {
  fileList.value = files
}

const handleRemove = (file, files) => {
  fileList.value = files
}

const clearAll = () => {
  fileList.value = []
  uploadResults.value = []
}

const startUpload = async () => {
  if (fileList.value.length === 0) return

  uploading.value = true
  uploadedCount.value = 0
  uploadResults.value = []

  const files = fileList.value.map(f => f.raw)

  try {
    const { data } = await bookApi.uploadBooks(files, category.value)
    uploadResults.value = data.results || []
    uploadedCount.value = uploadResults.value.filter(r => r.success).length

    const successCount = uploadResults.value.filter(r => r.success).length
    ElMessage.success(`上传完成：${successCount}/${fileList.value.length} 成功`)

    // 加载最近上传
    loadRecent()
  } catch (e) {
    ElMessage.error('上传失败')
  } finally {
    uploading.value = false
    fileList.value = []
  }
}

// ===== 扫描文件夹导入 =====
const scanPath = ref('')
const scanning = ref(false)
const scanDone = ref(false)
const scanTotal = ref(0)
const scanDoneCount = ref(0)
const scanSuccess = ref(0)
const scanFailed = ref(0)
const scanExists = ref(0)
const scanCurrent = ref('')
const scanMessage = ref('')

const scanPercent = computed(() => {
  if (scanTotal.value === 0) return 0
  return Math.min(100, Math.round((scanDoneCount.value / scanTotal.value) * 100))
})

const startScan = async () => {
  const path = scanPath.value.trim()
  if (!path || scanning.value) return

  scanning.value = true
  scanDone.value = false
  scanTotal.value = 0
  scanDoneCount.value = 0
  scanSuccess.value = 0
  scanFailed.value = 0
  scanExists.value = 0
  scanCurrent.value = ''
  scanMessage.value = ''

  try {
    const resp = await fetch(`${API_BASE}/books/scan`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${getToken()}`
      },
      body: JSON.stringify({ path, category: category.value })
    })

    if (!resp.ok) {
      let msg = `扫描失败 (${resp.status})`
      try {
        const err = await resp.json()
        if (err.error) msg = err.error
      } catch (e) { /* ignore */ }
      ElMessage.error(msg)
      scanMessage.value = msg
      return
    }

    // 流式读取 SSE 进度
    const reader = resp.body.getReader()
    const decoder = new TextDecoder()
    let buffer = ''

    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      buffer += decoder.decode(value, { stream: true })
      let idx
      while ((idx = buffer.indexOf('\n\n')) !== -1) {
        const raw = buffer.slice(0, idx)
        buffer = buffer.slice(idx + 2)
        handleScanEvent(raw)
      }
    }
  } catch (e) {
    ElMessage.error('扫描连接中断')
  } finally {
    scanning.value = false
    scanDone.value = true
    loadRecent()
  }
}

const handleScanEvent = (raw) => {
  let event = 'message'
  const dataLines = []
  for (const line of raw.split('\n')) {
    if (line.startsWith('event:')) event = line.slice(6).trim()
    else if (line.startsWith('data:')) dataLines.push(line.slice(5).trim())
  }
  if (dataLines.length === 0) return
  try {
    const data = JSON.parse(dataLines.join('\n'))
    if (event === 'progress') {
      scanTotal.value = data.total || 0
      scanDoneCount.value = data.done || 0
      scanSuccess.value = data.success || 0
      scanFailed.value = data.failed || 0
      scanExists.value = data.exists || 0
      scanCurrent.value = data.current || ''
    } else if (event === 'done') {
      scanTotal.value = data.total || 0
      scanDoneCount.value = data.done || 0
      scanSuccess.value = data.success || 0
      scanFailed.value = data.failed || 0
      scanExists.value = data.exists || 0
      scanCurrent.value = ''
      if (data.message) scanMessage.value = data.message
    } else if (event === 'error') {
      scanMessage.value = data.error || '扫描失败'
      ElMessage.error(scanMessage.value)
    }
  } catch (e) { /* ignore */ }
}

const goToBook = (id) => router.push(`/book/${id}`)

const loadRecent = async () => {
  try {
    const { data } = await bookApi.listBooks({ page_size: 8, sort: 'created_at' })
    recentBooks.value = data.data || []
  } catch (e) {
    // ignore
  }
}

onMounted(() => {
  loadCategories()
  loadRecent()
  window.addEventListener('resize', onRecentResize)
})
onBeforeUnmount(() => {
  window.removeEventListener('resize', onRecentResize)
  if (recentResizeTimer) clearTimeout(recentResizeTimer)
})
</script>

<style scoped>
.upload-page {
  position: relative;
  max-width: 1400px;
  margin: 0 auto;
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

.upload-card {
  position: relative;
  z-index: 1;
  background: rgba(255, 255, 255, 0.88);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  border: 1px solid rgba(255, 255, 255, 0.7);
  box-shadow: 0 12px 40px rgba(31, 38, 135, 0.1);
  border-radius: 20px;
  padding: 32px;
}

.import-cat-bar {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  background: rgba(255, 255, 255, 0.88);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  border: 1px solid rgba(255, 255, 255, 0.7);
  box-shadow: 0 12px 40px rgba(31, 38, 135, 0.1);
  border-radius: 20px;
  padding: 18px 24px;
  margin-bottom: 18px;
}
.import-cat-label {
  font-weight: 600;
  color: #334155;
}
.import-cat-bar .form-hint {
  color: #94a3b8;
  font-size: 13px;
}

/* ========== 页面头部（标题 + 搜索，沿用书架页眉头） ========== */
.page-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  flex-wrap: wrap;
  margin-bottom: 24px;
  padding: 20px 24px;
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.78);
  backdrop-filter: blur(14px);
  -webkit-backdrop-filter: blur(14px);
  border: 1px solid rgba(255, 255, 255, 0.7);
  box-shadow: 0 10px 36px rgba(31, 38, 135, 0.1);
}

.head-title {
  display: flex;
  align-items: center;
  gap: 14px;
  min-width: 0;
}

.head-title-icon {
  flex-shrink: 0;
  width: 46px;
  height: 46px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  background: var(--app-gradient);
  box-shadow: 0 10px 24px color-mix(in srgb, var(--app-primary) 35%, transparent);
}

.head-title-text {
  min-width: 0;
}

.head-title-text h2 {
  margin: 0;
  font-size: 22px;
  font-weight: 700;
  background: linear-gradient(120deg, var(--app-primary-dark), #0891b2);
  -webkit-background-clip: text;
  background-clip: text;
  -webkit-text-fill-color: transparent;
  white-space: nowrap;
}

.head-title-text p {
  margin: 4px 0 0;
  font-size: 13px;
  color: #64748b;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.head-search {
  width: min(360px, 100%);
  flex-shrink: 0;
}

.search-input :deep(.el-input__wrapper) {
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.92);
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--app-primary) 18%, transparent) inset;
  padding: 4px 16px;
  transition: box-shadow 0.25s ease;
}

.search-input :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1.5px var(--app-primary), 0 6px 20px color-mix(in srgb, var(--app-primary) 22%, transparent);
}

.search-input :deep(.el-input__inner) {
  font-size: 14px;
}

.search-input :deep(.el-input__prefix) {
  color: #94a3b8;
}

.format-list {
  font-size: 13px;
  color: #475569;
  margin-bottom: 20px;
  padding: 16px 18px;
  background: linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 6%, transparent), rgba(34, 211, 238, 0.05));
  border: 1px solid color-mix(in srgb, var(--app-primary) 15%, transparent);
  border-radius: 12px;
  line-height: 2;
}

.format-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.format-dot {
  flex-shrink: 0;
  width: 8px;
  height: 8px;
  border-radius: 50%;
}
.dot-1 { background: var(--app-primary); box-shadow: 0 0 8px color-mix(in srgb, var(--app-primary) 60%, transparent); }
.dot-2 { background: #06b6d4; box-shadow: 0 0 8px rgba(6, 182, 212, 0.6); }
.dot-3 { background: #f59e0b; box-shadow: 0 0 8px rgba(245, 158, 11, 0.6); }

.upload-actions {
  margin-top: 20px;
  text-align: center;
  padding: 20px;
  border-top: 1px solid color-mix(in srgb, var(--app-primary) 15%, transparent);
}

.file-count {
  margin-bottom: 12px;
  color: #64748b;
}

.upload-btn-row {
  display: flex;
  gap: 12px;
  justify-content: center;
  align-items: center;
}

.upload-btn {
  width: 200px;
}

.clear-btn {
  min-width: 96px;
}

.upload-actions :deep(.el-button--primary) {
  background: linear-gradient(135deg, var(--app-primary), color-mix(in srgb, var(--app-primary) 55%, #8b5cf6));
  border: none;
  box-shadow: 0 8px 20px color-mix(in srgb, var(--app-primary) 35%, transparent);
}

/* ===== 扫描文件夹导入 ===== */
.scan-section {
  margin-top: 20px;
  padding: 20px;
  background: linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 5%, transparent), rgba(139, 92, 246, 0.06));
  border: 1px solid color-mix(in srgb, var(--app-primary) 18%, transparent);
  border-radius: 14px;
}

.scan-head {
  margin-bottom: 14px;
}

.scan-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 700;
  color: #0f172a;
}

.scan-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: 9px;
  color: #fff;
  background: linear-gradient(135deg, var(--app-primary), color-mix(in srgb, var(--app-primary) 55%, #8b5cf6));
  box-shadow: 0 6px 14px color-mix(in srgb, var(--app-primary) 35%, transparent);
}

.scan-tag {
  font-size: 11px;
  font-weight: 600;
  color: var(--app-primary);
  padding: 2px 8px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--app-primary) 12%, transparent);
}

.scan-desc {
  margin-top: 6px;
  padding-left: 38px;
  font-size: 12px;
  color: #64748b;
}

.scan-form {
  display: flex;
  gap: 12px;
  align-items: center;
}

.scan-input {
  flex: 1;
}

.scan-input :deep(.el-input__wrapper) {
  border-radius: 10px;
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--app-primary) 25%, transparent) inset;
}

.scan-input :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1px var(--app-primary) inset, 0 0 0 3px color-mix(in srgb, var(--app-primary) 15%, transparent);
}

.scan-btn {
  flex-shrink: 0;
  min-width: 120px;
  border-radius: 10px;
  background: linear-gradient(135deg, var(--app-primary), color-mix(in srgb, var(--app-primary) 55%, #8b5cf6));
  border: none;
  box-shadow: 0 8px 20px color-mix(in srgb, var(--app-primary) 35%, transparent);
}

.scan-btn:hover {
  background: linear-gradient(135deg, var(--app-primary-dark), #7c3aed);
}

.scan-btn.is-disabled {
  background: linear-gradient(135deg, #a5b4fc, #c4b5fd);
  box-shadow: none;
}

.scan-progress {
  margin-top: 16px;
  padding: 14px 16px;
  background: rgba(255, 255, 255, 0.75);
  border: 1px solid color-mix(in srgb, var(--app-primary) 15%, transparent);
  border-radius: 12px;
}

.scan-progress-bar {
  width: 100%;
}

.scan-progress-bar :deep(.el-progress-bar__outer) {
  background: color-mix(in srgb, var(--app-primary) 12%, transparent);
}

.scan-progress-bar :deep(.el-progress-bar__inner) {
  background: linear-gradient(90deg, var(--app-primary), #8b5cf6);
}

.scan-stats {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 16px;
  margin-top: 10px;
  font-size: 13px;
  color: #475569;
}

.stat-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.stat-item b {
  color: #0f172a;
}

.stat-ok {
  color: #10b981;
}

.stat-skip {
  color: #f59e0b;
}

.stat-err {
  color: #f56c6c;
}

.scan-current {
  margin-top: 8px;
  font-size: 12px;
  color: #64748b;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.scan-done-msg {
  margin-top: 8px;
  font-size: 13px;
  font-weight: 600;
  color: var(--app-primary);
}

.progress-list {
  margin-top: 20px;
  padding: 18px;
  background: rgba(245, 247, 250, 0.7);
  backdrop-filter: blur(8px);
  border: 1px solid color-mix(in srgb, var(--app-primary) 12%, transparent);
  border-radius: 14px;
}

.progress-item {
  display: flex;
  align-items: center;
  padding: 8px 0;
  gap: 8px;
}

.progress-item.success {
  color: #10b981;
}

.progress-item.error {
  color: #f56c6c;
}

.status-icon {
  font-size: 18px;
}

.filename {
  flex: 1;
  font-size: 14px;
}

.error-msg {
  font-size: 12px;
  color: #f56c6c;
}

.recent-section {
  margin-top: 32px;
  padding-top: 24px;
  border-top: 1px solid color-mix(in srgb, var(--app-primary) 15%, transparent);
}

.recent-more {
  margin-top: 12px;
  text-align: center;
}

.recent-section h3 {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 0 16px 0;
  font-size: 16px;
  color: #0f172a;
}

.recent-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 8px;
  color: var(--app-primary);
  background: linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 12%, transparent), rgba(34, 211, 238, 0.12));
}

.recent-card {
  cursor: pointer;
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1), box-shadow 0.3s ease;
  margin-bottom: 12px;
  border-radius: 12px;
}

.recent-card:hover {
  transform: translateY(-6px) scale(1.02);
}

.recent-cover {
  width: 100%;
  aspect-ratio: 2/3;
  object-fit: cover;
  border-radius: 12px;
  background: #e2e8f0;
  box-shadow: 0 6px 20px rgba(31, 38, 135, 0.15);
  transition: box-shadow 0.3s ease;
}

.recent-card:hover .recent-cover {
  box-shadow: 0 16px 36px color-mix(in srgb, var(--app-primary) 30%, transparent);
}

.recent-cover-placeholder {
  width: 100%;
  aspect-ratio: 2/3;
  background: linear-gradient(160deg, #1e293b, #0f172a);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #94a3b8;
  box-shadow: 0 6px 20px rgba(31, 38, 135, 0.15);
}

.recent-title {
  margin-top: 8px;
  font-size: 13px;
  color: #1e293b;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-weight: 500;
}

/* 拖拽上传区 */
.upload-page :deep(.el-upload-dragger) {
  background: rgba(245, 247, 250, 0.6);
  border: 2px dashed color-mix(in srgb, var(--app-primary) 30%, transparent);
  border-radius: 16px;
  padding: 36px 20px;
  transition: all 0.3s ease;
}

.upload-page :deep(.el-upload-dragger:hover) {
  border-color: var(--app-primary);
  background: linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 6%, transparent), rgba(34, 211, 238, 0.05));
  box-shadow: 0 0 24px color-mix(in srgb, var(--app-primary) 15%, transparent);
}

.upload-page :deep(.el-upload-dragger .el-icon--upload) {
  color: var(--app-primary);
  margin-bottom: 12px;
}

.upload-page :deep(.el-upload-dragger .el-upload__text) {
  color: #475569;
}

.upload-page :deep(.el-upload-dragger .el-upload__text em) {
  color: var(--app-primary);
  font-style: normal;
  font-weight: 600;
}

.upload-page :deep(.el-upload__tip) {
  color: #94a3b8;
}

/* ===== 入场动画 ===== */
.cardIn {
  animation: cardIn 0.55s cubic-bezier(0.22, 1, 0.36, 1) both;
  animation-delay: calc(var(--i, 0) * 80ms);
}
@keyframes cardIn {
  from {
    opacity: 0;
    transform: translateY(18px) scale(0.98);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

/* ===== 手机端适配 ===== */
@media (max-width: 768px) {
  .upload-card {
    padding: 20px 16px;
    border-radius: 16px;
  }

  .page-head {
    flex-direction: column;
    align-items: flex-start;
    gap: 14px;
    padding: 16px 18px;
  }

  .head-title {
    gap: 10px;
  }

  .head-title-icon {
    width: 40px;
    height: 40px;
  }

  .head-title-text h2 {
    font-size: 18px;
  }

  .head-title-text p {
    font-size: 12px;
  }

  .head-search {
    width: 100%;
  }

  .format-list {
    font-size: 12px;
    padding: 14px;
    line-height: 2.2;
  }

  .upload-actions {
    padding: 16px 0;
  }

  .upload-btn-row {
    flex-direction: column;
    gap: 10px;
  }

  .upload-btn {
    width: 100%;
  }

  .clear-btn {
    width: 100%;
    margin-left: 0 !important;
  }

  .upload-page :deep(.el-upload-dragger) {
    padding: 24px 12px;
  }

  .scan-section {
    padding: 16px 12px;
  }

  .scan-desc {
    padding-left: 0;
  }

  .scan-form {
    flex-direction: column;
    align-items: stretch;
    gap: 10px;
  }

  .scan-btn {
    width: 100%;
  }

  .scan-stats {
    gap: 10px;
  }

  .recent-section {
    margin-top: 24px;
    padding-top: 20px;
  }
}
</style>
