<template>
  <div class="lib-page">
    <div class="bg-glow bg-glow-1"></div>
    <div class="bg-glow bg-glow-2"></div>

    <!-- 媒体库卡片网格 -->
    <div v-if="loading" class="lib-loading">
      <el-icon class="is-loading" :size="28"><Loading /></el-icon>
    </div>

    <div v-else-if="libraries.length === 0" class="empty-state cardIn">
      <el-icon :size="56"><FolderOpened /></el-icon>
      <h3>还没有媒体库</h3>
      <p>添加一个目录，系统会自动扫描其中的电子书并入库（不会移动或复制你的文件）。</p>
      <el-button type="primary" @click="openCreate">
        <el-icon><Plus /></el-icon> 新建媒体库
      </el-button>
    </div>

    <div v-else class="lib-grid">
      <div
        v-for="(lib, index) in libraries"
        :key="lib.id"
        class="lib-card cardIn"
        :style="{ '--i': index }"
      >
        <div class="lib-card-head">
          <div class="lib-name" :title="lib.name">{{ lib.name }}</div>
          <el-tag
            v-if="lib.status === 'scanning'"
            type="primary"
            size="small"
            effect="dark"
            class="status-tag scanning"
          >扫描中</el-tag>
          <el-tag
            v-else-if="lib.status === 'error'"
            type="danger"
            size="small"
            effect="plain"
            class="status-tag"
          >异常</el-tag>
          <el-tag
            v-else
            type="success"
            size="small"
            effect="plain"
            class="status-tag"
          >正常</el-tag>
        </div>

        <div class="lib-path" :title="lib.path">
          <el-icon><Folder /></el-icon>
          <span>{{ lib.path }}</span>
        </div>

        <div class="lib-tags">
          <span class="meta-tag" :class="{ on: lib.recursive }">
            {{ lib.recursive ? '递归子目录' : '仅当前目录' }}
          </span>
          <span class="meta-tag" :class="{ on: lib.auto_scan }">
            文件夹监听{{ lib.auto_scan ? ' · 开启' : '已关闭' }}
          </span>
          <span v-if="lib.category" class="meta-tag cat-tag">{{ lib.category }}</span>
        </div>

        <div class="lib-stats">
          <div class="stat">
            <b>{{ lib.book_count }}</b>
            <span>本图书</span>
          </div>
          <div class="stat">
            <b>{{ formatTime(lib.last_scan_at) }}</b>
            <span>上次扫描</span>
          </div>
        </div>

        <div class="lib-actions">
          <el-button size="small" @click="enterLib(lib)">
            <el-icon><Right /></el-icon> 进入
          </el-button>
          <el-button size="small" type="primary" :loading="scanningId === lib.id" :disabled="scanningId === lib.id || lib.status === 'scanning'" @click="scanLib(lib)">
            <el-icon><Search /></el-icon> 扫描
          </el-button>
          <el-button size="small" text @click="openEdit(lib)">
            <el-icon><Edit /></el-icon>
          </el-button>
          <el-button size="small" text type="danger" @click="removeLib(lib)">
            <el-icon><Delete /></el-icon>
          </el-button>
        </div>
      </div>
    </div>

    <!-- 新建 / 编辑对话框 -->
    <BaseDialog
      v-model="dialogVisible"
      :title="editing ? '编辑媒体库' : '新建媒体库'"
      :icon="editing ? 'Edit' : 'FolderAdd'"
      width="520px"
      destroy-on-close
    >
      <el-form :model="form" label-width="92px" class="lib-form">
        <el-form-item label="媒体库名称" required>
          <el-input v-model="form.name" placeholder="如：我的小说 / 技术资料" maxlength="64" />
        </el-form-item>
        <el-form-item label="目录路径" required>
          <el-input
            v-model="form.path"
            placeholder="服务器上的绝对目录，如 /Users/me/Books"
            :disabled="scanningId !== null"
          >
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
        <el-form-item label="归类分类">
          <el-select
            v-model="form.category"
            placeholder="不指定（入库后分类为空）"
            clearable
            filterable
            allow-create
            default-first-option
            style="width: 100%"
          >
            <el-option v-for="c in categoryOptions" :key="c" :label="c" :value="c" />
          </el-select>
          <span class="form-hint">扫描入库的图书将归入该分类</span>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveLib">保存</el-button>
      </template>
    </BaseDialog>

    <!-- 扫描进度对话框 -->
    <BaseDialog
      v-model="scanDialogVisible"
      title="扫描媒体库"
      icon="Search"
      width="520px"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
      :show-close="!scanActive"
    >
      <div v-if="scanActive || scanDone" class="scan-progress">
        <el-progress
          :percentage="scanPercent"
          :stroke-width="10"
          :status="scanDone && scanFailed === 0 ? 'success' : (scanDone && scanFailed > 0 ? 'warning' : undefined)"
        />
        <div class="scan-stats">
          <span class="stat-item">共 <b>{{ scanTotal }}</b></span>
          <span class="stat-item stat-ok"><el-icon><CircleCheck /></el-icon> 新增 <b>{{ scanAdded }}</b></span>
          <span class="stat-item stat-upd"><el-icon><Refresh /></el-icon> 更新 <b>{{ scanUpdated }}</b></span>
          <span class="stat-item stat-err"><el-icon><Remove /></el-icon> 缺失 <b>{{ scanDeleted }}</b></span>
        </div>
        <div v-if="scanCurrent" class="scan-current" :title="scanCurrent">正在处理：{{ scanCurrent }}</div>
        <div v-if="scanMessage" class="scan-done-msg">{{ scanMessage }}</div>
      </div>
      <template #footer>
        <el-button :disabled="scanActive" @click="scanDialogVisible = false">关闭</el-button>
      </template>
    </BaseDialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { libraryApi } from '@/api/library'
import { bookApi } from '@/api/book'
import { runLibraryScan } from '@/composables/useLibraryScan'
import BaseDialog from '@/components/BaseDialog.vue'

const router = useRouter()
const loading = ref(false)
const libraries = ref([])

const dialogVisible = ref(false)
const editing = ref(false)
const saving = ref(false)
const editingId = ref(null)
const form = ref({
  name: '',
  path: '',
  recursive: true,
  auto_scan: true,
  category: ''
})

// 分类选项：用于「添加书籍时归类到指定分类」
const categoryOptions = ref([])
const loadCategories = async () => {
  try {
    const { data } = await bookApi.getCategories()
    categoryOptions.value = (data.data || []).map(c => c.category).filter(Boolean)
  } catch (e) { /* 分类加载失败不影响主流程 */ }
}

const scanDialogVisible = ref(false)
const scanActive = ref(false)
const scanDone = ref(false)
const scanningId = ref(null)
const scanTotal = ref(0)
const scanAdded = ref(0)
const scanUpdated = ref(0)
const scanDeleted = ref(0)
const scanFailed = ref(0)
const scanCurrent = ref('')
const scanMessage = ref('')

const scanPercentCount = ref(0)
const scanPercent = computed(() => {
  if (scanTotal.value === 0) return 0
  return Math.min(100, Math.round((scanPercentCount.value / scanTotal.value) * 100))
})

const formatTime = (t) => {
  if (!t) return '从未'
  const d = new Date(t)
  const now = new Date()
  const sameDay = d.toDateString() === now.toDateString()
  const hh = String(d.getHours()).padStart(2, '0')
  const mm = String(d.getMinutes()).padStart(2, '0')
  if (sameDay) return `${hh}:${mm}`
  return `${d.getMonth() + 1}/${d.getDate()} ${hh}:${mm}`
}

const loadLibraries = async () => {
  loading.value = true
  try {
    const { data } = await libraryApi.list()
    libraries.value = data.data || []
  } catch (e) {
    ElMessage.error('加载媒体库失败')
  } finally {
    loading.value = false
  }
}

const openCreate = () => {
  editing.value = false
  editingId.value = null
  form.value = { name: '', path: '', recursive: true, auto_scan: true, category: '' }
  dialogVisible.value = true
}

const openEdit = (lib) => {
  editing.value = true
  editingId.value = lib.id
  form.value = {
    name: lib.name,
    path: lib.path,
    recursive: lib.recursive,
    auto_scan: lib.auto_scan,
    category: lib.category || ''
  }
  dialogVisible.value = true
}

const saveLib = async () => {
  if (!form.value.name.trim()) {
    ElMessage.warning('请填写媒体库名称')
    return
  }
  if (!form.value.path.trim()) {
    ElMessage.warning('请填写目录路径')
    return
  }
  saving.value = true
  try {
    if (editing.value) {
      await libraryApi.update(editingId.value, { ...form.value })
      ElMessage.success('已保存')
    } else {
      await libraryApi.create({ ...form.value })
      ElMessage.success('媒体库已创建，可立即扫描')
    }
    dialogVisible.value = false
    loadLibraries()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}

const enterLib = (lib) => router.push(`/library/${lib.id}`)

const scanLib = (lib) => {
  scanningId.value = lib.id
  scanDialogVisible.value = true
  scanActive.value = true
  scanDone.value = false
  scanTotal.value = 0
  scanAdded.value = 0
  scanUpdated.value = 0
  scanDeleted.value = 0
  scanFailed.value = 0
  scanPercentCount.value = 0
  scanCurrent.value = ''
  scanMessage.value = ''

  runLibraryScan(lib.id, {
    onProgress: (d) => {
      scanTotal.value = d.total || 0
      scanAdded.value = d.added || 0
      scanUpdated.value = d.updated || 0
      scanDeleted.value = d.deleted || 0
      scanFailed.value = d.failed || 0
      scanCurrent.value = d.current || ''
      scanPercentCount.value = (d.done || 0)
    },
    onDone: (d) => {
      scanTotal.value = d.total || 0
      scanAdded.value = d.added || 0
      scanUpdated.value = d.updated || 0
      scanDeleted.value = d.deleted || 0
      scanFailed.value = d.failed || 0
      scanPercentCount.value = scanTotal.value
      scanActive.value = false
      scanDone.value = true
      const msg = `扫描完成：新增 ${scanAdded.value}，更新 ${scanUpdated.value}，缺失 ${scanDeleted.value}`
      scanMessage.value = msg
      ElMessage.success(msg)
      loadLibraries()
    },
    onError: (msg) => {
      scanActive.value = false
      scanDone.value = true
      scanMessage.value = msg
      ElMessage.error(msg)
      loadLibraries()
    }
  }).catch(() => {
    scanActive.value = false
    scanDone.value = true
  }).finally(() => {
    // 兜底：无论 SSE 是否正常收到 done 事件，扫描流结束时都重置状态，
    // 避免关闭按钮因 scanActive/scanDone 卡在「扫描中」而一直禁用。
    scanningId.value = null
    scanActive.value = false
    scanDone.value = true
  })
}

const removeLib = async (lib) => {
  try {
    await ElMessageBox.confirm(
      `删除媒体库「${lib.name}」将同时移除其图书记录（不会删除你的原始文件），确定继续？`,
      '删除媒体库',
      { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' }
    )
  } catch (e) {
    return
  }
  try {
    await libraryApi.remove(lib.id)
    ElMessage.success('已删除')
    loadLibraries()
  } catch (e) {
    ElMessage.error('删除失败')
  }
}

onMounted(() => {
  loadLibraries()
  loadCategories()
})

// 供媒体库 tab 条右侧「新建媒体库」按钮调用
defineExpose({ openCreate })
</script>

<style scoped>
.lib-page {
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

.page-head {
  position: relative;
  z-index: 1;
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
.head-title-text h2 {
  margin: 0;
  font-size: 22px;
  font-weight: 700;
  background: linear-gradient(120deg, var(--app-primary-dark), #0891b2);
  -webkit-background-clip: text;
  background-clip: text;
  -webkit-text-fill-color: transparent;
}
.head-title-text p {
  margin: 4px 0 0;
  font-size: 13px;
  color: #64748b;
}
.create-btn {
  flex-shrink: 0;
  border-radius: 12px;
  background: var(--app-gradient, linear-gradient(135deg, #6366f1, #8b5cf6));
  border: none;
  box-shadow: 0 8px 20px color-mix(in srgb, var(--app-primary) 35%, transparent);
}

.lib-loading {
  position: relative;
  z-index: 1;
  text-align: center;
  padding: 80px 0;
  color: var(--app-primary);
}

.empty-state {
  position: relative;
  z-index: 1;
  text-align: center;
  padding: 56px 24px;
  border-radius: 20px;
  background: rgba(255, 255, 255, 0.82);
  backdrop-filter: blur(14px);
  border: 1px solid rgba(255, 255, 255, 0.7);
  box-shadow: 0 10px 36px rgba(31, 38, 135, 0.1);
  color: #94a3b8;
}
.empty-state h3 {
  margin: 16px 0 8px;
  color: #334155;
  font-size: 18px;
}
.empty-state p {
  margin: 0 auto 20px;
  max-width: 460px;
  font-size: 13px;
  line-height: 1.7;
}

.lib-grid {
  position: relative;
  z-index: 1;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 18px;
}
.lib-card {
  background: rgba(255, 255, 255, 0.86);
  backdrop-filter: blur(14px);
  -webkit-backdrop-filter: blur(14px);
  border: 1px solid rgba(255, 255, 255, 0.7);
  box-shadow: 0 10px 32px rgba(31, 38, 135, 0.1);
  border-radius: 18px;
  padding: 18px 20px;
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1), box-shadow 0.3s ease;
}
.lib-card:hover {
  transform: translateY(-6px);
  box-shadow: 0 20px 48px color-mix(in srgb, var(--app-primary) 22%, transparent);
}
.lib-card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 12px;
}
.lib-name {
  font-size: 17px;
  font-weight: 700;
  color: #0f172a;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.status-tag {
  flex-shrink: 0;
}
.lib-path {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #64748b;
  background: linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 6%, transparent), rgba(34, 211, 238, 0.05));
  border: 1px solid color-mix(in srgb, var(--app-primary) 12%, transparent);
  border-radius: 10px;
  padding: 8px 10px;
  margin-bottom: 12px;
}
.lib-path span {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.lib-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 14px;
}
.meta-tag {
  font-size: 11px;
  padding: 3px 10px;
  border-radius: 999px;
  color: #94a3b8;
  background: #f1f5f9;
  border: 1px solid #e2e8f0;
}
.meta-tag.on {
  color: var(--app-primary);
  background: color-mix(in srgb, var(--app-primary) 10%, transparent);
  border-color: color-mix(in srgb, var(--app-primary) 22%, transparent);
}
.lib-stats {
  display: flex;
  gap: 24px;
  margin-bottom: 16px;
  padding-top: 14px;
  border-top: 1px solid color-mix(in srgb, var(--app-primary) 12%, transparent);
}
.stat {
  display: flex;
  flex-direction: column;
}
.stat b {
  font-size: 18px;
  color: #0f172a;
  font-variant-numeric: tabular-nums;
}
.stat span {
  font-size: 11px;
  color: #94a3b8;
  margin-top: 2px;
}
.lib-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

/* 卡片操作按钮统一为同族主题样式，与「新建媒体库」主按钮风格一致 */
.lib-actions :deep(.el-button) {
  border-radius: 10px;
  margin-left: 0;
  transition: all 0.2s;
}
/* 次按钮（进入）：主题浅色描边，与主按钮同色系 */
.lib-actions :deep(.el-button:not(.el-button--primary):not(.el-button--danger):not(.is-text)) {
  background: color-mix(in srgb, var(--app-primary) 10%, #fff);
  border: 1px solid color-mix(in srgb, var(--app-primary) 32%, transparent);
  color: var(--app-primary);
}
.lib-actions :deep(.el-button:not(.el-button--primary):not(.el-button--danger):not(.is-text):hover) {
  background: color-mix(in srgb, var(--app-primary) 16%, #fff);
  border-color: var(--app-primary);
}
/* 主按钮（扫描）：与「新建媒体库」一致的渐变 */
.lib-actions :deep(.el-button--primary) {
  background: var(--app-gradient, linear-gradient(135deg, #6366f1, #8b5cf6));
  border: none;
  box-shadow: 0 6px 16px color-mix(in srgb, var(--app-primary) 28%, transparent);
}
.lib-actions :deep(.el-button--primary:hover) {
  background: var(--app-gradient, linear-gradient(135deg, #6366f1, #8b5cf6));
  filter: brightness(1.06);
}
/* 文本图标按钮（编辑 / 删除）：圆角 + 主题悬浮底色 */
.lib-actions :deep(.el-button.is-text) {
  border-radius: 10px;
}
.lib-actions :deep(.el-button.is-text:not(.el-button--danger):hover) {
  background: color-mix(in srgb, var(--app-primary) 12%, transparent);
  color: var(--app-primary);
}
.lib-actions :deep(.el-button.is-text.el-button--danger:hover) {
  background: color-mix(in srgb, var(--el-color-danger, #f56c6c) 14%, transparent);
}

/* 表单 */
.lib-form .form-hint {
  margin-left: 10px;
  font-size: 12px;
  color: #94a3b8;
}

/* 扫描进度 */
.scan-progress {
  padding: 6px 4px;
}
.scan-stats {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 16px;
  margin-top: 12px;
  font-size: 13px;
  color: #475569;
}
.stat-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.stat-item b { color: #0f172a; }
.stat-ok { color: #10b981; }
.stat-upd { color: #6366f1; }
.stat-err { color: #f43f5e; }
.scan-current {
  margin-top: 10px;
  font-size: 12px;
  color: #64748b;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.scan-done-msg {
  margin-top: 10px;
  font-size: 13px;
  font-weight: 600;
  color: var(--app-primary);
}

/* 入场动画 */
.cardIn {
  animation: cardIn 0.55s cubic-bezier(0.22, 1, 0.36, 1) both;
  animation-delay: calc(var(--i, 0) * 80ms);
}
@keyframes cardIn {
  from { opacity: 0; transform: translateY(18px) scale(0.98); }
  to { opacity: 1; transform: translateY(0) scale(1); }
}

@media (max-width: 768px) {
  .page-head {
    padding: 16px 18px;
  }
  .head-title-text h2 { font-size: 18px; }
  .lib-grid {
    grid-template-columns: 1fr;
  }
}
</style>
