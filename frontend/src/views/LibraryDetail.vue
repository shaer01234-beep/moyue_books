<template>
  <div class="ld-page">
    <div class="bg-glow bg-glow-1"></div>
    <div class="bg-glow bg-glow-2"></div>

    <!-- 头部 -->
    <div class="page-head">
      <div class="head-left">
        <el-button class="back-btn" circle @click="goBack">
          <el-icon><ArrowLeft /></el-icon>
        </el-button>
        <div class="head-title-text">
          <h2>{{ lib.name || '媒体库' }}</h2>
          <p>
            <el-icon><Folder /></el-icon> {{ lib.path }}
          </p>
        </div>
      </div>
      <div class="head-right">
        <el-button
          type="primary"
          :loading="scanActive"
          :disabled="scanActive || (lib && lib.status === 'scanning')"
          @click="scanLib"
        >
          <el-icon><Search /></el-icon> {{ scanActive ? '扫描中…' : '立即扫描' }}
        </el-button>
        <el-button text @click="openEdit">
          <el-icon><Edit /></el-icon> 编辑
        </el-button>
        <el-button text type="danger" @click="removeLib">
          <el-icon><Delete /></el-icon> 删除
        </el-button>
      </div>
    </div>

    <!-- 概览 -->
    <div class="overview cardIn" v-if="!loadingLib">
      <div class="ov-item">
        <b>{{ lib.book_count }}</b><span>本图书</span>
      </div>
      <div class="ov-item">
        <b>{{ lib.recursive ? '递归' : '仅当前' }}</b><span>扫描范围</span>
      </div>
      <div class="ov-item">
        <b>{{ lib.auto_scan ? '监听中' : '已关闭' }}</b><span>文件夹监听</span>
      </div>
      <div class="ov-item">
        <b>{{ lib.category || '未指定' }}</b><span>归类分类</span>
      </div>
      <div class="ov-item">
        <b>{{ formatTime(lib.last_scan_at) }}</b><span>上次扫描</span>
      </div>
      <div class="ov-item" v-if="lastResult">
        <b class="hl">{{ lastResult.added }}/{{ lastResult.deleted }}</b>
        <span>上次新增/缺失</span>
      </div>
    </div>

    <!-- 扫描进度条 -->
    <div v-if="scanActive || scanDone" class="scan-bar cardIn" style="--i:1">
      <el-progress :percentage="scanPercent" :stroke-width="9" :status="scanDone ? (scanFailed===0?'success':'warning') : undefined" />
      <div class="scan-stats">
        <span>共 <b>{{ scanTotal }}</b></span>
        <span class="stat-ok">新增 {{ scanAdded }}</span>
        <span class="stat-upd">更新 {{ scanUpdated }}</span>
        <span class="stat-err">缺失 {{ scanDeleted }}</span>
        <span v-if="scanCurrent" class="scan-current" :title="scanCurrent">{{ scanCurrent }}</span>
      </div>
    </div>

    <!-- 标签页 -->
    <el-tabs v-model="activeTab" class="ld-tabs">
      <el-tab-pane label="图书" name="books">
        <div v-if="booksLoading" class="tab-loading">
          <el-icon class="is-loading" :size="24"><Loading /></el-icon>
        </div>
        <div v-else-if="books.length === 0" class="tab-empty">
          该媒体库暂无图书，点击「立即扫描」从目录导入。
        </div>
        <template v-else>
          <el-row :gutter="16">
            <el-col
              v-for="(book, index) in books"
              :key="book.id"
              :xs="12" :sm="8" :md="6" :lg="4"
            >
              <BookCard :book="book" :index="index % 12" @click="openBook(book)" />
            </el-col>
          </el-row>
          <div class="pagination" v-if="bookTotal > bookPageSize">
            <PaginationBar
              v-model:current-page="bookPage"
              v-model:page-size="bookPageSize"
              :total="bookTotal"
              :page-sizes="bookPageSizes"
              @current-change="loadBooks"
              @size-change="onBookSizeChange"
            />
          </div>
        </template>
      </el-tab-pane>

      <el-tab-pane label="扫描日志" name="logs">
        <div v-if="logsLoading" class="tab-loading">
          <el-icon class="is-loading" :size="24"><Loading /></el-icon>
        </div>
        <div v-else-if="logs.length === 0" class="tab-empty">
          暂无扫描记录。
        </div>
        <el-timeline v-else class="log-timeline">
          <el-timeline-item
            v-for="log in logs"
            :key="log.id"
            :timestamp="formatFull(log.scan_at)"
            placement="top"
            :type="log.deleted > 0 ? 'warning' : (log.added > 0 ? 'primary' : '')"
          >
            <div class="log-card">
              <div class="log-summary">
                <span class="log-count">发现 {{ log.total_found }} 个文件</span>
                <span v-if="log.added" class="stat-ok">新增 {{ log.added }}</span>
                <span v-if="log.updated" class="stat-upd">更新 {{ log.updated }}</span>
                <span v-if="log.deleted" class="stat-err">缺失 {{ log.deleted }}</span>
                <span v-if="!log.added && !log.updated && !log.deleted" class="log-none">无变化</span>
              </div>
              <div v-if="logDetail(log).added.length" class="log-detail">
                <div class="log-detail-title">新增</div>
                <div class="log-list">
                  <span v-for="(t, i) in logDetail(log).added" :key="'a'+i" class="log-chip ok">{{ t }}</span>
                </div>
              </div>
              <div v-if="logDetail(log).deleted.length" class="log-detail">
                <div class="log-detail-title">缺失</div>
                <div class="log-list">
                  <span v-for="(t, i) in logDetail(log).deleted" :key="'d'+i" class="log-chip err">{{ t }}</span>
                </div>
              </div>
            </div>
          </el-timeline-item>
        </el-timeline>
      </el-tab-pane>
    </el-tabs>

    <!-- 编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      title="编辑媒体库"
      width="520px"
      destroy-on-close
    >
      <el-form :model="form" label-width="92px" class="lib-form">
        <el-form-item label="媒体库名称" required>
          <el-input v-model="form.name" maxlength="64" />
        </el-form-item>
        <el-form-item label="目录路径" required>
          <el-input v-model="form.path">
            <template #prefix><el-icon><Folder /></el-icon></template>
          </el-input>
        </el-form-item>
        <el-form-item label="扫描范围">
          <el-switch v-model="form.recursive" active-text="递归子目录" inactive-text="仅当前目录" />
        </el-form-item>
        <el-form-item label="文件夹监听">
          <el-switch v-model="form.auto_scan" active-text="开启" inactive-text="关闭" />
          <span class="form-hint">开启后，目录下文件变动会自动重新扫描（防抖 10 秒）</span>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveLib">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { libraryApi } from '@/api/library'
import { bookApi } from '@/api/book'
import { runLibraryScan } from '@/composables/useLibraryScan'
import BookCard from '@/components/BookCard.vue'
import PaginationBar from '@/components/PaginationBar.vue'

const route = useRoute()
const router = useRouter()
const libId = computed(() => route.params.id)

const lib = ref({})
const loadingLib = ref(false)
const lastResult = ref(null)

const activeTab = ref('books')
const books = ref([])
const booksLoading = ref(false)
const bookPage = ref(1)
const bookPageSize = ref(24)
const bookTotal = ref(0)
// 与书架一致的分页条数选项
const bookPageSizes = [12, 24, 48, 96]
const onBookSizeChange = () => {
  bookPage.value = 1
  loadBooks()
}

const logs = ref([])
const logsLoading = ref(false)

const dialogVisible = ref(false)
const saving = ref(false)
const form = ref({ name: '', path: '', recursive: true, auto_scan: true })

// 扫描状态
const scanActive = ref(false)
const scanDone = ref(false)
const scanTotal = ref(0)
const scanAdded = ref(0)
const scanUpdated = ref(0)
const scanDeleted = ref(0)
const scanFailed = ref(0)
const scanCount = ref(0)
const scanCurrent = ref('')
const scanPercent = computed(() => {
  if (scanTotal.value === 0) return 0
  return Math.min(100, Math.round((scanCount.value / scanTotal.value) * 100))
})

const formatTime = (t) => {
  if (!t) return '从未'
  const d = new Date(t)
  const now = new Date()
  if (d.toDateString() === now.toDateString()) {
    return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
  }
  return `${d.getMonth() + 1}/${d.getDate()} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}
const formatFull = (t) => {
  if (!t) return ''
  const d = new Date(t)
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
}

const logDetail = (log) => {
  try {
    const d = JSON.parse(log.detail || '{}')
    return { added: d.added || [], updated: d.updated || [], deleted: d.deleted || [] }
  } catch (e) {
    return { added: [], updated: [], deleted: [] }
  }
}

const loadLib = async () => {
  loadingLib.value = true
  try {
    const { data } = await libraryApi.get(libId.value)
    lib.value = data
    if (data.last_result) {
      try { lastResult.value = JSON.parse(data.last_result) } catch (e) { lastResult.value = null }
    }
  } catch (e) {
    ElMessage.error('加载媒体库失败')
  } finally {
    loadingLib.value = false
  }
}

const loadBooks = async () => {
  booksLoading.value = true
  try {
    const { data } = await bookApi.listBooks({
      library_id: libId.value,
      page: bookPage.value,
      page_size: bookPageSize.value,
      sort: 'created_at',
      order: 'desc'
    })
    books.value = data.data || []
    bookTotal.value = data.total || 0
  } catch (e) {
    ElMessage.error('加载图书失败')
  } finally {
    booksLoading.value = false
  }
}

const loadLogs = async () => {
  logsLoading.value = true
  try {
    const { data } = await libraryApi.logs(libId.value, { page: 1, page_size: 50 })
    logs.value = data.data || []
  } catch (e) {
    // ignore
  } finally {
    logsLoading.value = false
  }
}

const scanLib = () => {
  scanActive.value = true
  scanDone.value = false
  scanTotal.value = 0
  scanAdded.value = 0
  scanUpdated.value = 0
  scanDeleted.value = 0
  scanFailed.value = 0
  scanCount.value = 0
  scanCurrent.value = ''
  runLibraryScan(libId.value, {
    onProgress: (d) => {
      scanTotal.value = d.total || 0
      scanAdded.value = d.added || 0
      scanUpdated.value = d.updated || 0
      scanDeleted.value = d.deleted || 0
      scanFailed.value = d.failed || 0
      scanCurrent.value = d.current || ''
      scanCount.value = d.done || 0
    },
    onDone: (d) => {
      scanTotal.value = d.total || 0
      scanAdded.value = d.added || 0
      scanUpdated.value = d.updated || 0
      scanDeleted.value = d.deleted || 0
      scanFailed.value = d.failed || 0
      scanCount.value = scanTotal.value
      scanActive.value = false
      scanDone.value = true
      ElMessage.success(`扫描完成：新增 ${scanAdded.value}，更新 ${scanUpdated.value}，缺失 ${scanDeleted.value}`)
      loadLib()
      loadBooks()
      loadLogs()
    },
    onError: (msg) => {
      scanActive.value = false
      scanDone.value = true
      ElMessage.error(msg)
      loadLib()
    }
  }).catch(() => {
    scanActive.value = false
    scanDone.value = true
  })
}

const openEdit = () => {
  form.value = {
    name: lib.value.name,
    path: lib.value.path,
    recursive: lib.value.recursive,
    auto_scan: lib.value.auto_scan
  }
  dialogVisible.value = true
}

const saveLib = async () => {
  saving.value = true
  try {
    await libraryApi.update(libId.value, { ...form.value })
    ElMessage.success('已保存')
    dialogVisible.value = false
    loadLib()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}

const removeLib = async () => {
  try {
    await ElMessageBox.confirm(
      `删除媒体库「${lib.value.name}」将同时移除其图书记录（不会删除你的原始文件），确定继续？`,
      '删除媒体库', { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' }
    )
  } catch (e) {
    return
  }
  try {
    await libraryApi.remove(libId.value)
    ElMessage.success('已删除')
    router.replace('/libraries')
  } catch (e) {
    ElMessage.error('删除失败')
  }
}

const openBook = (book) => router.push(`/book/${book.id}`)
const goBack = () => router.back()

onMounted(() => {
  loadLib()
  loadBooks()
  loadLogs()
})
</script>

<style scoped>
.ld-page {
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
  width: 440px; height: 440px;
  background: radial-gradient(circle, color-mix(in srgb, var(--app-primary) 16%, transparent), transparent 70%);
  top: -100px; left: -100px;
  animation: float 12s ease-in-out infinite;
}
.bg-glow-2 {
  width: 380px; height: 380px;
  background: radial-gradient(circle, rgba(34, 211, 238, 0.12), transparent 70%);
  bottom: -80px; right: -80px;
  animation: float 14s ease-in-out infinite reverse;
}
@keyframes float {
  0%, 100% { transform: translate(0, 0); }
  50% { transform: translate(30px, -20px); }
}

.page-head {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
  margin-bottom: 18px;
  padding: 16px 20px;
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.78);
  backdrop-filter: blur(14px);
  -webkit-backdrop-filter: blur(14px);
  border: 1px solid rgba(255, 255, 255, 0.7);
  box-shadow: 0 10px 36px rgba(31, 38, 135, 0.1);
}
.head-left {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.back-btn {
  flex-shrink: 0;
  border: none;
  background: color-mix(in srgb, var(--app-primary) 10%, transparent);
  color: var(--app-primary);
}
.head-title-text {
  min-width: 0;
}
.head-title-text h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 700;
  color: #0f172a;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.head-title-text p {
  margin: 4px 0 0;
  font-size: 12px;
  color: #64748b;
  display: flex;
  align-items: center;
  gap: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.head-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}
.head-right :deep(.el-button--primary) {
  background: linear-gradient(135deg, var(--app-primary), color-mix(in srgb, var(--app-primary) 55%, #8b5cf6));
  border: none;
  box-shadow: 0 8px 20px color-mix(in srgb, var(--app-primary) 35%, transparent);
}

.overview {
  position: relative;
  z-index: 1;
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 16px;
  padding: 16px 20px;
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.82);
  backdrop-filter: blur(14px);
  border: 1px solid rgba(255, 255, 255, 0.7);
  box-shadow: 0 8px 28px rgba(31, 38, 135, 0.08);
}
.ov-item {
  display: flex;
  flex-direction: column;
  padding: 0 18px;
  border-right: 1px solid color-mix(in srgb, var(--app-primary) 12%, transparent);
}
.ov-item:last-child { border-right: none; }
.ov-item b {
  font-size: 18px;
  color: #0f172a;
  font-variant-numeric: tabular-nums;
}
.ov-item b.hl { color: var(--app-primary); }
.ov-item span {
  font-size: 11px;
  color: #94a3b8;
  margin-top: 2px;
}

.scan-bar {
  position: relative;
  z-index: 1;
  margin-bottom: 16px;
  padding: 14px 18px;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.82);
  border: 1px solid color-mix(in srgb, var(--app-primary) 15%, transparent);
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
.scan-current {
  color: #64748b;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
}
.stat-ok { color: #10b981; }
.stat-upd { color: #6366f1; }
.stat-err { color: #f43f5e; }

.ld-tabs {
  position: relative;
  z-index: 1;
}
.ld-tabs :deep(.el-tabs__nav-wrap::after) {
  background: color-mix(in srgb, var(--app-primary) 12%, transparent);
}
.ld-tabs :deep(.el-tabs__item.is-active) {
  color: var(--app-primary);
  font-weight: 600;
}
.ld-tabs :deep(.el-tabs__active-bar) {
  background: var(--app-primary);
}

.tab-loading, .tab-empty {
  text-align: center;
  padding: 60px 0;
  color: #94a3b8;
}
.tab-loading { color: var(--app-primary); }

.log-timeline {
  padding: 20px 8px 8px 4px;
  max-width: 760px;
}
.log-card {
  background: rgba(255, 255, 255, 0.86);
  border: 1px solid rgba(255, 255, 255, 0.7);
  border-radius: 14px;
  padding: 12px 16px;
  box-shadow: 0 6px 20px rgba(31, 38, 135, 0.06);
}
.log-summary {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 14px;
  font-size: 13px;
  font-weight: 600;
  color: #334155;
}
.log-none { color: #94a3b8; font-weight: 500; }
.log-detail {
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px dashed color-mix(in srgb, var(--app-primary) 15%, transparent);
}
.log-detail-title {
  font-size: 12px;
  color: #64748b;
  margin-bottom: 6px;
}
.log-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.log-chip {
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 8px;
  max-width: 200px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.log-chip.ok { color: #047857; background: rgba(16, 185, 129, 0.1); }
.log-chip.err { color: #be123c; background: rgba(244, 63, 94, 0.1); }

.lib-form .form-hint {
  margin-left: 10px;
  font-size: 12px;
  color: #94a3b8;
}

.cardIn {
  animation: cardIn 0.5s cubic-bezier(0.22, 1, 0.36, 1) both;
  animation-delay: calc(var(--i, 0) * 80ms);
}
@keyframes cardIn {
  from { opacity: 0; transform: translateY(16px); }
  to { opacity: 1; transform: translateY(0); }
}

@media (max-width: 768px) {
  .page-head { padding: 14px 16px; }
  .head-title-text h2 { font-size: 17px; }
  .ov-item { padding: 0 12px; }
  .ov-item b { font-size: 16px; }
}

/* ========== 分页（与书架一致的主题样式） ========== */
.pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 22px;
}
.pagination :deep(.el-pagination.is-background .btn-prev),
.pagination :deep(.el-pagination.is-background .btn-next),
.pagination :deep(.el-pagination.is-background .el-pager li) {
  background: #f1f5f9;
  color: #475569;
  border-radius: 9px;
}
.pagination :deep(.el-pagination.is-background .btn-prev:hover),
.pagination :deep(.el-pagination.is-background .btn-next:hover),
.pagination :deep(.el-pagination.is-background .el-pager li:hover) {
  color: var(--app-primary);
  background: color-mix(in srgb, var(--app-primary) 12%, #f1f5f9);
}
.pagination :deep(.el-pagination.is-background .el-pager li.is-active) {
  background: var(--app-gradient, linear-gradient(135deg, #6366f1, #8b5cf6));
  color: #fff;
  border: none;
  box-shadow: 0 6px 16px color-mix(in srgb, var(--app-primary) 30%, transparent);
}
.pagination :deep(.el-pagination.is-background .el-pager li.is-active:hover) {
  color: #fff;
}
.pagination :deep(.el-pagination .el-pagination__total) {
  margin-right: 8px;
}
</style>
