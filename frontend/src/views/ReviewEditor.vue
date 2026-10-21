<template>
  <div class="review-editor-page">
    <!-- 顶部栏 -->
    <header class="editor-topbar">
      <div class="topbar-left">
        <el-button class="back-btn" text :icon="ArrowLeft" @click="goBack" />
        <div class="topbar-book" v-if="book">
          <img v-if="book.cover_path" :src="coverUrl(book.id)" :alt="book.title" class="topbar-cover" />
          <div v-else class="topbar-cover cover-fallback">
            <el-icon :size="18"><Reading /></el-icon>
          </div>
          <div class="topbar-book-info">
            <span class="topbar-book-title">{{ book.title }}</span>
            <span class="topbar-book-sub">{{ annotations.length }} 条划线</span>
          </div>
        </div>
        <div class="topbar-book" v-else>
          <div class="topbar-cover cover-fallback">
            <el-icon :size="18"><Reading /></el-icon>
          </div>
          <span class="topbar-book-title">未选择图书</span>
        </div>
      </div>
      <div class="topbar-title-block">
        <el-input
          v-model="form.title"
          class="editor-title-input topbar-title-input"
          placeholder="为本篇读后感起个标题…"
          maxlength="120"
        />
        <el-rate v-model="form.rating" :max="5" class="editor-rate" />
      </div>
      <div class="topbar-right">
        <el-button v-if="!book" type="primary" plain :icon="Reading" @click="pickerVisible = true">
          选择图书
        </el-button>
        <el-button
          type="primary"
          class="save-btn"
          :loading="saving"
          :disabled="!book"
          @click="saveReview"
        >
          <el-icon style="margin-right: 4px"><Check /></el-icon> 保存
        </el-button>
      </div>
    </header>

    <!-- 主体 -->
    <div
      class="editor-body"
      v-if="book"
      :style="bodyGridStyle"
    >
      <!-- 左栏：本书划线笔记列表 -->
      <aside v-if="!sidebarCollapsed" class="editor-sidebar">
        <div class="sidebar-head">
          <span>划线笔记</span>
        </div>
        <div class="sidebar-list">
          <div
            v-for="a in sortedAnnotations"
            :key="a.id"
            class="side-item"
            :class="{ expanded: expandedId === a.id }"
            @click="toggleItem(a)"
          >
            <div class="side-item-head">
              <div class="side-item-title-row">
                <el-icon class="side-arrow" :size="13">
                  <CaretBottom v-if="expandedId === a.id" />
                  <CaretRight v-else />
                </el-icon>
                <span class="ann-style" :style="annStylePreview(a)">{{ annStyleLabel(a) }}</span>
                <span class="side-item-title">{{ a.content || '（无文本）' }}</span>
                <el-icon
                  v-if="hasData(a)"
                  class="side-item-del"
                  :size="15"
                  @click.stop="removeAnnotation(a)"
                ><Delete /></el-icon>
              </div>
              <div v-if="expandedId !== a.id" class="side-item-excerpt">{{ a.content }}</div>
              <div v-else class="side-item-full">
                <div class="ann-full-text">{{ a.content }}</div>
                <div v-if="a.note" class="ann-note">
                  <el-icon :size="13" style="margin-right: 4px"><EditPen /></el-icon>{{ a.note }}
                </div>
              </div>
              <div class="side-item-meta">
                <el-tag size="small" type="info" effect="dark">{{ annPageLabel(a) }}</el-tag>
                <span class="ann-time">{{ annTime(a.created_at) }}</span>
              </div>
            </div>
          </div>
          <div v-if="!sortedAnnotations.length" class="side-empty">
            <el-icon :size="26" class="side-empty-icon"><EditPen /></el-icon>
            <p>暂无划线</p>
            <p class="side-empty-sub">在阅读页面选中文字即可添加划线</p>
          </div>
        </div>
      </aside>

      <!-- 中栏：书写区 -->
      <main class="editor-main">
        <div class="editor-form">
          <div class="md-editor">
            <div class="md-editor-tabs">
              <div class="md-tabs-left">
                <span :class="['md-tab', { active: activePane === 'write' }]" @click="activePane = 'write'">编辑</span>
                <span :class="['md-tab', { active: activePane === 'preview' }]" @click="activePane = 'preview'">预览</span>
              </div>
              <div class="md-tabs-right">
                <el-tooltip content="划线列表" placement="bottom">
                  <el-button
                    size="small"
                    text
                    :class="['md-tab-btn', { active: !sidebarCollapsed }]"
                    @click="sidebarCollapsed = !sidebarCollapsed"
                  >
                    <el-icon :size="16"><Notebook /></el-icon>
                  </el-button>
                </el-tooltip>
                <el-tooltip content="实时预览" placement="bottom">
                  <el-button
                    size="small"
                    text
                    :class="['md-tab-btn', { active: !previewCollapsed }]"
                    @click="previewCollapsed = !previewCollapsed"
                  >
                    <el-icon :size="16"><View /></el-icon>
                  </el-button>
                </el-tooltip>
              </div>
            </div>
            <textarea
              v-if="activePane === 'write'"
              v-model="form.content"
              class="md-textarea"
              placeholder="在这里书写你的读书笔记…
支持 Markdown：**加粗**、*斜体*、- 列表、> 引用、``` 代码块、[链接](url)"
            ></textarea>
            <div v-else class="md-preview markdown-body" v-html="previewHtml"></div>
          </div>
        </div>
      </main>

      <!-- 右栏：实时预览（桌面端） -->
      <aside v-if="!previewCollapsed" class="editor-preview-pane">
        <div class="preview-head">
          <span class="preview-title">实时预览</span>
        </div>
        <div class="preview-scroll markdown-body" v-html="previewHtml"></div>
      </aside>
    </div>

    <!-- 未选书时：引导选择 -->
    <div class="editor-empty" v-else>
      <div class="empty-icon">
        <el-icon :size="44"><Notebook /></el-icon>
      </div>
      <p class="empty-title">为读后感选择一本书</p>
      <p class="empty-sub">选择后即可开始书写，页面将同时展示本书的划线笔记</p>
      <el-button type="primary" :icon="Reading" @click="pickerVisible = true">选择图书</el-button>
    </div>

    <!-- 图书选择器 -->
    <BookPickerDialog
      v-model="bookId"
      :visible="pickerVisible"
      title="选择图书"
      @update:visible="(v) => (pickerVisible = v)"
      @select="onBookSelected"
    />
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowLeft, Reading, Notebook, Delete, Check, CaretRight, CaretBottom, View, EditPen } from '@element-plus/icons-vue'
import { reviewApi } from '@/api/review'
import { bookApi } from '@/api/book'
import { renderMarkdown } from '@/utils/markdown'
import BookPickerDialog from '@/components/BookPickerDialog.vue'

const router = useRouter()
const route = useRoute()

const book = ref(null)
const bookId = ref(null)
const pickerVisible = ref(false)
const annotations = ref([])
const currentId = ref(null)
const expandedId = ref(null)
const saving = ref(false)
const activePane = ref('write')
const sidebarCollapsed = ref(false)
const previewCollapsed = ref(true)

const form = reactive({ title: '', rating: 0, content: '' })

const previewHtml = computed(() => renderMarkdown(form.content))

const bodyGridStyle = computed(() => {
  const cols = []
  if (!sidebarCollapsed.value) cols.push('280px')
  cols.push('minmax(0, 1fr)')
  if (!previewCollapsed.value) cols.push('340px')
  return { gridTemplateColumns: cols.join(' ') }
})

const sortedAnnotations = computed(() => {
  const ft = book.value?.file_type?.toLowerCase()
  return [...annotations.value]
    .filter((a) => a.type === 'highlight' || a.type === 'note')
    .sort((a, b) => {
      const la = annLocOf(a)
      const lb = annLocOf(b)
      return la === lb ? new Date(a.created_at) - new Date(b.created_at) : la - lb
    })
})

function coverUrl(id) {
  return bookApi.coverUrl(id)
}

function hasData(a) {
  return !!(a.content || a.note)
}

function toggleItem(a) {
  expandedId.value = expandedId.value === a.id ? null : a.id
}

function goBack() {
  router.push('/reviews')
}

const annLocOf = (ann) => {
  try {
    const pos = JSON.parse(ann.position || '{}')
    const ft = book.value?.file_type?.toLowerCase()
    return ft === 'pdf' ? pos.page || 0 : pos.chapter || 0
  } catch (e) {
    return 0
  }
}

const annPageLabel = (ann) => {
  const ft = book.value?.file_type?.toLowerCase()
  const loc = annLocOf(ann)
  return ft === 'pdf' ? `第 ${loc} 页` : `第 ${loc + 1} 章`
}

const annStyleLabel = (ann) => {
  const map = { highlight: '划', underline: '线', wavy: '波', strikethrough: '删', note: '批' }
  return map[ann.style] || '划'
}

const annStylePreview = (ann) => {
  const map = {
    highlight: { background: 'rgba(255, 213, 79, 0.45)' },
    underline: { borderBottom: '2px solid rgba(74, 144, 226, 0.6)' },
    wavy: { textDecoration: 'underline wavy rgba(38, 166, 91, 0.7)' },
    strikethrough: { textDecoration: 'line-through rgba(220, 60, 60, 0.7)' },
    note: { background: 'rgba(154, 132, 255, 0.28)' }
  }
  return map[ann.style] || map.highlight
}

const annTime = (t) => {
  if (!t) return ''
  const d = new Date(t)
  if (isNaN(d)) return ''
  const pad = (n) => String(n).padStart(2, '0')
  return `${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

async function loadBook(id) {
  try {
    const { data } = await bookApi.getBook(id)
    book.value = data
  } catch (e) {
    book.value = null
  }
}

async function loadAnnotations(bookId) {
  try {
    const { data } = await bookApi.listAnnotations(bookId)
    annotations.value = data || []
  } catch (e) {
    annotations.value = []
  }
}

async function saveReview() {
  if (!book.value) {
    ElMessage.warning('请先选择图书')
    return
  }
  if (!form.title.trim()) {
    ElMessage.warning('请填写标题')
    return
  }
  saving.value = true
  try {
    const payload = {
      book_id: book.value.id,
      title: form.title.trim(),
      content: form.content,
      rating: form.rating || 0
    }
    if (currentId.value) {
      await reviewApi.updateReview(currentId.value, payload)
      ElMessage.success('读后感已更新')
    } else {
      const { data } = await reviewApi.createReview(payload)
      currentId.value = data.id
      ElMessage.success('读后感已保存')
    }
  } catch (e) {
    console.error(e)
    ElMessage.error(e.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}

async function removeAnnotation(a) {
  try {
    await ElMessageBox.confirm('确定删除这条划线吗？此操作不可恢复。', '删除确认', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消'
    })
  } catch (e) {
    return
  }
  try {
    await bookApi.deleteAnnotation(a.id)
    ElMessage.success('已删除')
    annotations.value = annotations.value.filter((x) => x.id !== a.id)
    if (expandedId.value === a.id) expandedId.value = null
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '删除失败')
  }
}

function onBookSelected(b) {
  book.value = b
  currentId.value = null
  expandedId.value = null
  form.title = ''
  form.rating = 0
  form.content = ''
  loadAnnotations(b.id)
}

async function init() {
  // 编辑模式：?id=xxx
  const editId = Number(route.query.id) || null
  // 预选图书：?book_id=xxx
  const preBookId = Number(route.query.book_id) || null

  if (editId) {
    try {
      const { data } = await reviewApi.getReview(editId)
      currentId.value = data.id
      form.title = data.title
      form.rating = data.rating || 0
      form.content = data.content || ''
      bookId.value = data.book_id
      await loadBook(data.book_id)
      await loadAnnotations(data.book_id)
      return
    } catch (e) {
      ElMessage.error('读后感不存在或已删除')
      router.replace('/reviews')
      return
    }
  }

  if (preBookId) {
    bookId.value = preBookId
    await loadBook(preBookId)
    if (book.value) await loadAnnotations(preBookId)
  } else {
    pickerVisible.value = true
  }
}

watch(
  () => route.query,
  () => init()
)

init()
</script>

<style scoped>
.review-editor-page {
  position: relative;
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: linear-gradient(160deg, #f7f9fe, #eef4fb);
}

/* ========== 顶部栏 ========== */
.editor-topbar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 18px;
  background: rgba(255, 255, 255, 0.85);
  backdrop-filter: blur(14px);
  border-bottom: 1px solid rgba(226, 232, 240, 0.8);
  z-index: 2;
}
.topbar-left {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.back-btn {
  border-radius: 10px;
  font-size: 18px;
}
.topbar-book {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.topbar-cover {
  width: 38px;
  height: 52px;
  border-radius: 6px;
  object-fit: cover;
  box-shadow: 0 4px 12px rgba(31, 38, 135, 0.25);
  flex-shrink: 0;
}
.topbar-cover.cover-fallback {
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  background: linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 50%, transparent), rgba(34, 211, 238, 0.5));
}
.topbar-book-info {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.topbar-book-title {
  font-size: 15px;
  font-weight: 700;
  color: #1e293b;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 260px;
}
.topbar-book-sub {
  font-size: 12px;
  color: #94a3b8;
}
.topbar-title-block {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  margin: 0 12px;
}
.topbar-title-input {
  max-width: 520px;
}
.topbar-right {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}
.save-btn {
  border-radius: 10px;
  font-weight: 600;
  min-width: 96px;
}

/* ========== 主体三栏 ========== */
.editor-body {
  flex: 1;
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr) 340px;
  gap: 16px;
  padding: 16px 18px;
  min-height: 0;
  overflow: hidden;
}

/* 左栏 */
.editor-sidebar {
  background: rgba(255, 255, 255, 0.8);
  backdrop-filter: blur(12px);
  border: 1px solid rgba(255, 255, 255, 0.7);
  border-radius: 16px;
  box-shadow: 0 8px 32px rgba(31, 38, 135, 0.08);
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

.sidebar-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px;
  border-bottom: 1px solid #f0f2f5;
  font-size: 14px;
  font-weight: 700;
  color: #334155;
  flex-shrink: 0;
}
.sidebar-list {
  flex: 1;
  overflow: auto;
  padding: 8px;
}
.side-item {
  padding: 10px 12px;
  border-radius: 12px;
  cursor: pointer;
  margin-bottom: 6px;
  transition: background 0.2s, box-shadow 0.2s;
  border: 1px solid transparent;
}
.side-item:hover {
  background: #f8fafc;
}
.side-item.active {
  background: linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 10%, #fff), #f0f9ff);
  border-color: color-mix(in srgb, var(--app-primary) 25%, transparent);
  box-shadow: 0 6px 20px color-mix(in srgb, var(--app-primary) 12%, transparent);
}
.side-item-head {
  min-width: 0;
}
.side-item-title-row {
  display: flex;
  align-items: center;
  gap: 6px;
}
.side-arrow {
  color: #94a3b8;
  flex-shrink: 0;
  transition: transform 0.2s;
}
.side-item-title {
  flex: 1;
  min-width: 0;
  font-size: 14px;
  font-weight: 600;
  color: #1e293b;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.side-item-excerpt {
  margin-top: 4px;
  font-size: 12px;
  color: #64748b;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.ann-style {
  flex-shrink: 0;
  width: 18px;
  height: 18px;
  line-height: 18px;
  text-align: center;
  font-size: 11px;
  font-weight: 700;
  color: #475569;
  border-radius: 4px;
  user-select: none;
}
.side-item-full {
  margin-top: 4px;
  font-size: 12px;
  color: #64748b;
  line-height: 1.65;
  max-height: 260px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
  padding: 8px 10px;
  border-radius: 8px;
  background: color-mix(in srgb, var(--app-primary) 5%, #f8fafc);
}
.ann-full-text {
  white-space: pre-wrap;
  word-break: break-word;
  color: #334155;
}
.ann-note {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px dashed #dde3ec;
  display: flex;
  align-items: flex-start;
  color: #64748b;
  white-space: pre-wrap;
  word-break: break-word;
}
.ann-note .el-icon {
  margin-top: 1px;
  color: var(--app-primary);
}
.ann-time {
  font-size: 11px;
  color: #a5b3c6;
}
.side-item-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 6px;
}
.side-item-del {
  color: #cbd5e1;
  cursor: pointer;
  transition: color 0.2s;
  flex-shrink: 0;
  border-radius: 6px;
  padding: 2px;
}
.side-item-del:hover {
  color: #ef4444;
  background: color-mix(in srgb, #ef4444 10%, transparent);
}
.side-empty {
  height: 100%;
  min-height: 220px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 32px 16px;
  text-align: center;
  color: #94a3b8;
  font-size: 13px;
  line-height: 1.6;
}
.side-empty-icon {
  color: color-mix(in srgb, var(--app-primary) 45%, #94a3b8);
  margin-bottom: 4px;
}
.side-empty p {
  margin: 0;
}
.side-empty-sub {
  font-size: 12px;
  color: #b6c2d4;
  max-width: 180px;
}

/* 中栏 */
.editor-main {
  min-height: 0;
  overflow: auto;
}
.editor-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
  height: 100%;
}
.editor-title-input :deep(.el-input__wrapper) {
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.9);
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--app-primary) 18%, transparent) inset;
  padding: 4px 14px;
  min-height: 33px;
}
.editor-title-input :deep(.el-input__inner) {
  font-size: 15px;
  font-weight: 600;
  color: #1e293b;
}
.editor-title-input :deep(.el-input__inner::placeholder) {
  font-size: 13px;
  font-weight: 400;
  color: #94a3b8;
}
.editor-title-input :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1.5px var(--app-primary), 0 6px 20px color-mix(in srgb, var(--app-primary) 20%, transparent);
}
.editor-rate {
  flex-shrink: 0;
}
.md-editor {
  flex: 1;
  display: flex;
  flex-direction: column;
  border: 1px solid color-mix(in srgb, var(--app-primary) 22%, transparent);
  border-radius: 14px;
  overflow: hidden;
  background: #fff;
  box-shadow: 0 8px 32px rgba(31, 38, 135, 0.08);
  min-height: 0;
}
.md-editor-tabs {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-bottom: 1px solid #eef2f7;
  background: #f8fafc;
  flex-shrink: 0;
}
.md-tabs-left {
  display: flex;
  gap: 4px;
}
.md-tabs-right {
  display: flex;
  gap: 4px;
  align-items: center;
}
.md-tab {
  padding: 5px 16px;
  font-size: 13px;
  border-radius: 8px;
  color: #64748b;
  cursor: pointer;
  transition: all 0.2s ease;
  user-select: none;
}
.md-tab:hover {
  color: var(--app-primary);
}
.md-tab.active {
  color: #fff;
  background: var(--app-gradient);
  font-weight: 600;
}
.md-tab-btn {
  border-radius: 8px;
  padding: 5px 8px;
  color: #94a3b8;
  transition: all 0.2s;
}
.md-tab-btn:hover {
  color: var(--app-primary);
  background: color-mix(in srgb, var(--app-primary) 8%, transparent);
}
.md-tab-btn.active {
  color: var(--app-primary);
  background: color-mix(in srgb, var(--app-primary) 10%, transparent);
}
.md-textarea {
  flex: 1;
  width: 100%;
  border: none;
  outline: none;
  resize: none;
  padding: 18px 20px;
  font-size: 14px;
  line-height: 1.8;
  color: #1e293b;
  background: transparent;
  font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
  box-sizing: border-box;
  min-height: 0;
}
.md-preview {
  flex: 1;
  padding: 18px 20px;
  overflow: auto;
  min-height: 0;
}

/* 右栏预览 */
.editor-preview-pane {
  background: rgba(255, 255, 255, 0.8);
  backdrop-filter: blur(12px);
  border: 1px solid rgba(255, 255, 255, 0.7);
  border-radius: 16px;
  box-shadow: 0 8px 32px rgba(31, 38, 135, 0.08);
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}
.preview-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px;
  border-bottom: 1px solid #f0f2f5;
  font-size: 14px;
  font-weight: 700;
  color: #334155;
  flex-shrink: 0;
}
.preview-toggle-btn {
  color: #94a3b8;
  font-size: 13px;
}
.preview-toggle-btn:hover {
  color: var(--app-primary);
  background: color-mix(in srgb, var(--app-primary) 8%, transparent);
}

.preview-scroll {
  flex: 1;
  overflow: auto;
  padding: 16px;
  min-height: 0;
}

/* 空状态 */
.editor-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #94a3b8;
  padding-bottom: 80px;
}
.empty-icon {
  width: 96px;
  height: 96px;
  border-radius: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--app-primary);
  background: color-mix(in srgb, var(--app-primary) 10%, transparent);
  margin-bottom: 20px;
}
.empty-title {
  margin: 0 0 8px;
  font-size: 18px;
  font-weight: 700;
  color: #475569;
}
.empty-sub {
  margin: 0 0 22px;
  font-size: 13px;
}

/* ========== 响应式 ========== */
@media (max-width: 1100px) {
  .editor-body {
    grid-template-columns: 240px minmax(0, 1fr) !important;
  }
  .editor-preview-pane {
    display: none;
  }
}
@media (max-width: 760px) {
  .editor-body {
    grid-template-columns: 1fr !important;
    padding: 12px;
  }
  .editor-sidebar {
    display: none;
  }
  .editor-topbar {
    flex-wrap: wrap;
    row-gap: 8px;
  }
  .topbar-title-block {
    flex: 1 1 100%;
    order: 3;
    max-width: none;
    margin: 0;
    justify-content: flex-start;
  }
  .topbar-title-input {
    max-width: none;
    flex: 1;
  }
  .topbar-book-sub {
    display: none;
  }
  .topbar-book-title {
    max-width: 110px;
  }
  .editor-rate :deep(.el-rate__icon) {
    font-size: 18px;
  }
  .md-tabs-right {
    display: none;
  }
}
</style>

<style>
/* Markdown 渲染样式（全局，供 v-html 使用） */
.markdown-body {
  font-size: 14px;
  color: #334155;
  word-break: break-word;
  line-height: 1.8;
}
.markdown-body h1, .markdown-body h2, .markdown-body h3,
.markdown-body h4, .markdown-body h5, .markdown-body h6 {
  margin: 1.2em 0 0.6em;
  line-height: 1.4;
  color: #1e293b;
  font-weight: 700;
}
.markdown-body h1 { font-size: 22px; }
.markdown-body h2 { font-size: 19px; }
.markdown-body h3 { font-size: 17px; }
.markdown-body h4, .markdown-body h5, .markdown-body h6 { font-size: 15px; }
.markdown-body p { margin: 0.7em 0; }
.markdown-body strong { font-weight: 700; color: #1e293b; }
.markdown-body em { font-style: italic; }
.markdown-body del { color: #94a3b8; }
.markdown-body a {
  color: var(--app-primary);
  text-decoration: none;
  border-bottom: 1px dashed color-mix(in srgb, var(--app-primary) 40%, transparent);
}
.markdown-body code {
  background: color-mix(in srgb, var(--app-primary) 8%, #f1f5f9);
  color: #c026d3;
  padding: 2px 6px;
  border-radius: 6px;
  font-size: 13px;
  font-family: 'SFMono-Regular', Consolas, Menlo, monospace;
}
.markdown-body pre {
  background: #1e293b;
  color: #e2e8f0;
  padding: 14px 16px;
  border-radius: 12px;
  overflow: auto;
}
.markdown-body pre code {
  background: transparent;
  color: inherit;
  padding: 0;
}
.markdown-body blockquote {
  margin: 0.8em 0;
  padding: 8px 14px;
  border-left: 4px solid var(--app-primary);
  background: color-mix(in srgb, var(--app-primary) 6%, transparent);
  border-radius: 0 10px 10px 0;
  color: #475569;
}
.markdown-body ul, .markdown-body ol {
  margin: 0.7em 0;
  padding-left: 1.6em;
}
.markdown-body li { margin: 0.3em 0; }
.markdown-body hr {
  border: none;
  border-top: 1px solid #e2e8f0;
  margin: 1.4em 0;
}
.markdown-body img { max-width: 100%; border-radius: 10px; }
.markdown-body table {
  width: 100%;
  border-collapse: collapse;
  margin: 0.8em 0;
}
.markdown-body th, .markdown-body td {
  border: 1px solid #e2e8f0;
  padding: 8px 12px;
  text-align: left;
}
.markdown-body th {
  background: color-mix(in srgb, var(--app-primary) 8%, #f8fafc);
  font-weight: 600;
}
</style>
