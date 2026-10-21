<template>
  <div class="reviews-page">
    <!-- 背景光晕 -->
    <div class="bg-glow bg-glow-1"></div>
    <div class="bg-glow bg-glow-2"></div>

    <!-- 页面头部 -->
    <div class="page-head">
      <div class="head-title">
        <div class="head-title-icon">
          <el-icon :size="22"><Notebook /></el-icon>
        </div>
        <div class="head-title-text">
          <h2>读后感</h2>
          <p>记录阅读时的所思所想，支持 Markdown 书写</p>
        </div>
        </div>
        <div class="head-actions">
          <ViewModeSwitch v-model="viewMode" />
          <el-button type="primary" class="new-btn" :icon="Plus" @click="startWrite">
            写读后感
          </el-button>
        </div>
        </div>

    <!-- 搜索区 -->
    <div class="search-zone">
      <div class="search-box" :class="{ focus: searchFocus }">
        <el-icon class="search-ic"><Search /></el-icon>
        <input
          v-model="keyword"
          class="search-input"
          placeholder="搜索书名或读后感标题…"
          @focus="searchFocus = true"
          @blur="searchFocus = false"
          @keyup.enter="loadReviews()"
        />
        <el-button
          v-if="keyword"
          class="search-clear"
          text
          :icon="Close"
          @click="keyword = ''; loadReviews()"
        />
        <el-button type="primary" class="search-btn" @click="loadReviews()">
          搜索
        </el-button>
      </div>
    </div>

    <!-- 统计卡片（点击可筛选列表） -->
    <div class="stats-row">
      <div
        v-for="s in statCards"
        :key="s.label"
        class="stat-card"
        :class="[s.filter !== null ? 'clickable' : '', { active: activeFilter === s.filter }]"
        :title="s.filter !== null ? (s.filter === '' ? '显示全部读后感' : '点击筛选' + s.label) : ''"
        @click="setFilter(s.filter)"
      >
        <div class="stat-icon" :style="{ background: s.gradient, boxShadow: `0 10px 24px ${s.glow}` }">
          <el-icon :size="22"><component :is="s.icon" /></el-icon>
        </div>
        <div class="stat-body">
          <div class="stat-value">{{ s.value }}</div>
          <div class="stat-label">{{ s.label }}</div>
        </div>
        <span class="stat-shine" :style="{ background: s.gradient }"></span>
      </div>
    </div>

    <!-- 筛选提示条 -->
    <div v-if="activeFilter" class="filter-tip">
      <span class="filter-tip-ic"><el-icon :size="14"><Filter /></el-icon></span>
      <span class="filter-tip-text">
        正在筛选：<b>{{ activeFilter === 'month' ? '本月新增' : '高星好评（≥4 星）' }}</b> 的读后感
      </span>
      <span class="filter-tip-count">共 {{ filteredReviews.length }} 篇</span>
      <el-button size="small" text type="primary" class="filter-tip-clear" @click="setFilter('')">
        清除筛选
      </el-button>
    </div>

    <!-- 加载中 -->
    <div v-if="loading" class="loading">
      <el-icon class="is-loading" :size="32"><Loading /></el-icon>
      <p>加载中...</p>
    </div>

    <!-- 空状态 -->
    <div v-else-if="!filteredReviews.length" class="empty">
      <div class="empty-icon">
        <el-icon :size="40"><Notebook /></el-icon>
      </div>
      <p class="empty-title">{{ keyword ? '未找到匹配的读后感' : (activeFilter ? '没有符合条件的读后感' : '还没有读后感') }}</p>
      <p class="empty-sub">{{ keyword ? '换个关键词试试吧' : (activeFilter ? '试试清除筛选或换个关键词' : '记录你的第一段阅读感悟') }}</p>
      <el-button type="primary" class="new-btn" :icon="Plus" @click="startWrite">
        写读后感
      </el-button>
    </div>

    <!-- 读后感列表 -->
    <div v-else :class="viewMode === 'grid' ? 'review-grid' : 'review-list'">
      <div
        v-for="r in pagedReviews"
        :key="r.id"
        class="review-card"
        :class="{ 'in-grid': viewMode === 'grid' }"
        @click="viewDetail(r)"
      >
        <div class="review-cover" v-if="r.book_id">
          <img
            v-if="r.book_cover"
            :src="coverUrl(r.book_id)"
            :alt="r.book_title"
            loading="lazy"
          />
          <div v-else class="cover-fallback">
            <el-icon :size="18"><Notebook /></el-icon>
          </div>
        </div>
        <div class="review-body">
          <div class="review-head">
            <span class="review-book" v-if="r.book_title">{{ r.book_title }}</span>
            <span class="review-book" v-else>图书 #{{ r.book_id }}</span>
            <el-rate
              v-if="r.rating > 0"
              class="review-rate"
              :model-value="r.rating"
              disabled
              :max="5"
            />
          </div>
          <h3 class="review-title" :title="r.title">{{ r.title }}</h3>
          <div class="review-foot">
            <span class="review-time">
              <el-icon :size="12"><Clock /></el-icon>
              {{ formatTime(r.updated_at) }}
            </span>
            <span class="review-actions" @click.stop>
              <el-button size="small" text type="primary" :icon="EditPen" @click="editReview(r)">
                编辑
              </el-button>
              <el-button size="small" text type="danger" :icon="Delete" @click="removeReview(r)">
                删除
              </el-button>
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- 分页 -->
    <div v-if="total > pageSize" class="pagination">
      <PaginationBar
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        :page-sizes="pageSizesOptions"
        @size-change="onSizeChange"
      />
    </div>

    <!-- 详情弹窗（阅读视图） -->
    <el-dialog
      v-model="detailVisible"
      class="review-dialog"
      width="780px"
      align-center
      append-to-body
    >
      <template #header>
        <div class="rd-head">
          <div class="rd-cover" v-if="detail.book_id">
            <img
              v-if="detail.book_cover"
              :src="coverUrl(detail.book_id)"
              :alt="detail.book_title"
            />
            <div v-else class="rd-cover-fallback">
              <el-icon :size="20"><Notebook /></el-icon>
            </div>
          </div>
          <div class="rd-meta">
            <h3 class="rd-title">{{ detail.title }}</h3>
            <div class="rd-sub">
              <span class="rd-book" v-if="detail.book_title">{{ detail.book_title }}</span>
              <span class="rd-book" v-else-if="detail.book_id">图书 #{{ detail.book_id }}</span>
              <el-rate
                v-if="detail.rating > 0"
                class="rd-rate"
                :model-value="detail.rating"
                disabled
                :max="5"
              />
              <span class="rd-time" v-if="detail.updated_at">
                <el-icon :size="12"><Clock /></el-icon>
                {{ formatTime(detail.updated_at) }} 更新
              </span>
            </div>
          </div>
        </div>
      </template>

      <div class="rd-body markdown-body" v-html="detailHtml"></div>

      <template #footer>
        <div class="rd-foot">
          <span class="rd-stat" v-if="detailWords">
            <el-icon :size="13"><Document /></el-icon>
            {{ detailWords }} 字 · 约 {{ readMinutes }} 分钟读完
          </span>
          <div class="rd-actions">
            <el-button type="danger" plain :icon="Delete" @click="removeFromDetail">删除</el-button>
            <el-button type="primary" :icon="EditPen" @click="editFromDetail">编辑</el-button>
          </div>
        </div>
      </template>
    </el-dialog>

    <!-- 写读后感：选择图书 -->
    <BookPickerDialog
      v-model="writeBookId"
      :visible="writePickerVisible"
      title="为读后感选择图书"
      @update:visible="(v) => (writePickerVisible = v)"
      @select="onWriteBookSelected"
    />
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Notebook, Search, Plus, Calendar, Star, Loading,
  Clock, EditPen, Delete, Close, Filter, Document
} from '@element-plus/icons-vue'
import { reviewApi } from '@/api/review'
import { bookApi } from '@/api/book'
import { useViewPagination } from '@/composables/useViewPagination'
import { renderMarkdown } from '@/utils/markdown'
import BookPickerDialog from '@/components/BookPickerDialog.vue'
import ViewModeSwitch from '@/components/ViewModeSwitch.vue'
import PaginationBar from '@/components/PaginationBar.vue'

const router = useRouter()

const loading = ref(false)
const reviews = ref([])        // 当前展示全集（无关键词时为全量）
const statsAll = ref([])       // 全量读后感（用于统计，始终不受关键词影响）

// 书架风格分页（两行自动 + 倍数选项 + 按视图持久化 + 缩放重算）；前端切片分页无需重新请求
const { viewMode, page, pageSize, pageSizesOptions, onSizeChange } =
  useViewPagination('reviews', () => {})
const keyword = ref('')
const searchFocus = ref(false)

const writePickerVisible = ref(false)
const writeBookId = ref(null)

const detailVisible = ref(false)
const detail = ref({})

// 列表筛选：'' 全部 / 'month' 本月新增 / 'high' 高星好评（≥4 星）
const activeFilter = ref('')

// 统计展示值（数字滚动动画）
const displayedStats = reactive({ total: 0, thisMonth: 0, high: 0, avg: 0 })
let statsAnimTimer = null

const detailHtml = computed(() => renderMarkdown(detail.value.content || ''))

// 弹窗内字数与预计阅读时长
const detailWords = computed(() => {
  const text = (detail.value.content || '').replace(/\s/g, '')
  return text.length || 0
})
const readMinutes = computed(() => Math.max(1, Math.round(detailWords.value / 300)))

function coverUrl(bookId) {
  return bookApi.coverUrl(bookId)
}

function formatTime(t) {
  if (!t) return ''
  const d = new Date(t)
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// ---------- 统计卡片（样式与功能对齐书架页） ----------

const isThisMonth = (t) => {
  const d = new Date(t)
  if (Number.isNaN(d.getTime())) return false
  const now = new Date()
  return d.getFullYear() === now.getFullYear() && d.getMonth() === now.getMonth()
}

const avgText = computed(() => (displayedStats.avg > 0 ? displayedStats.avg.toFixed(1) : '—'))

const statCards = computed(() => [
  {
    label: '读后感总数', value: displayedStats.total,
    icon: 'Collection', gradient: 'linear-gradient(135deg, var(--app-primary), #8b5cf6)',
    glow: 'rgba(99, 102, 241, 0.45)', filter: ''
  },
  {
    label: '本月新增', value: displayedStats.thisMonth,
    icon: 'Calendar', gradient: 'linear-gradient(135deg, #06b6d4, #22d3ee)',
    glow: 'rgba(6, 182, 212, 0.45)', filter: 'month'
  },
  {
    label: '高星好评', value: displayedStats.high,
    icon: 'Star', gradient: 'linear-gradient(135deg, #10b981, #34d399)',
    glow: 'rgba(16, 185, 129, 0.45)', filter: 'high'
  },
  {
    label: '平均评分', value: avgText.value,
    icon: 'DataLine', gradient: 'linear-gradient(135deg, #f59e0b, #fbbf24)',
    glow: 'rgba(245, 158, 11, 0.45)', filter: null
  }
])

// 统计数值滚动动画（与书架页一致）
const animateStats = (target) => {
  if (statsAnimTimer) cancelAnimationFrame(statsAnimTimer)
  const from = { ...displayedStats }
  const duration = 700
  const start = performance.now()
  const ease = (t) => 1 - Math.pow(1 - t, 3)
  const tick = (now) => {
    const p = Math.min((now - start) / duration, 1)
    const e = ease(p)
    displayedStats.total = Math.round(from.total + (target.total - from.total) * e)
    displayedStats.thisMonth = Math.round(from.thisMonth + (target.thisMonth - from.thisMonth) * e)
    displayedStats.high = Math.round(from.high + (target.high - from.high) * e)
    displayedStats.avg = Math.round((from.avg + (target.avg - from.avg) * e) * 10) / 10
    if (p < 1) statsAnimTimer = requestAnimationFrame(tick)
  }
  statsAnimTimer = requestAnimationFrame(tick)
}

function applyStats(list) {
  let totalCount = 0, thisMonth = 0, high = 0, ratedSum = 0, ratedCount = 0
  for (const r of list) {
    totalCount += 1
    if (isThisMonth(r.created_at)) thisMonth += 1
    const rt = r.rating || 0
    if (rt >= 4) high += 1
    if (rt > 0) { ratedSum += rt; ratedCount += 1 }
  }
  animateStats({ total: totalCount, thisMonth, high, avg: ratedCount ? ratedSum / ratedCount : 0 })
}

// ---------- 列表（接口返回全量，本地切片分页 + 筛选） ----------

const filteredReviews = computed(() => {
  const list = reviews.value
  if (activeFilter.value === 'month') return list.filter((r) => isThisMonth(r.created_at))
  if (activeFilter.value === 'high') return list.filter((r) => (r.rating || 0) >= 4)
  return list
})

const total = computed(() => filteredReviews.value.length)

const pagedReviews = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return filteredReviews.value.slice(start, start + pageSize.value)
})

function setFilter(filter) {
  if (filter === null || filter === activeFilter.value) return
  activeFilter.value = filter
  page.value = 1
}

async function fetchList(q) {
  const params = q ? { q } : {}
  const res = await reviewApi.listReviews(params)
  return res.data?.data || []
}

async function refreshStats() {
  try {
    const list = await fetchList('')
    statsAll.value = list
    applyStats(list)
  } catch (e) {
    /* 忽略统计失败 */
  }
}

// 无关键词：一次请求同时供列表与统计使用；
// 有关键词：列表取搜索结果，统计仍基于全量 statsAll
async function loadReviews() {
  loading.value = true
  try {
    const q = keyword.value.trim()
    if (q) {
      reviews.value = await fetchList(q)
      if (!statsAll.value.length) await refreshStats()
    } else {
      reviews.value = await fetchList('')
      statsAll.value = reviews.value
      applyStats(statsAll.value)
    }
    page.value = 1
  } catch (e) {
    reviews.value = []
    if (!keyword.value.trim()) statsAll.value = []
  } finally {
    loading.value = false
  }
}

function startWrite() {
  writeBookId.value = null
  writePickerVisible.value = true
}

function onWriteBookSelected(b) {
  writePickerVisible.value = false
  router.push({ path: '/reviews/editor', query: { book_id: b.id } })
}

function editReview(r) {
  router.push({ path: '/reviews/editor', query: { id: r.id } })
}

function viewDetail(r) {
  detail.value = r
  detailVisible.value = true
}

function editFromDetail() {
  detailVisible.value = false
  editReview(detail.value)
}

async function removeReview(r) {
  try {
    await ElMessageBox.confirm(
      `确定删除读后感「${r.title}」吗？此操作不可恢复。`,
      '删除确认',
      { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' }
    )
  } catch (e) {
    return
  }
  try {
    await reviewApi.deleteReview(r.id)
    ElMessage.success('已删除')
    loadReviews()
    if (keyword.value.trim()) refreshStats()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '删除失败')
  }
}

async function removeFromDetail() {
  const r = detail.value
  detailVisible.value = false
  await removeReview(r)
}

onMounted(() => {
  loadReviews()
})
</script>

<style scoped>
.reviews-page {
  position: relative;
  max-width: 1400px;
  margin: 0 auto;
  min-height: calc(100vh - 120px);
}

/* ========== 背景光晕 ========== */
.bg-glow {
  position: fixed;
  border-radius: 50%;
  filter: blur(90px);
  pointer-events: none;
  z-index: 0;
}
.bg-glow-1 {
  width: 480px;
  height: 480px;
  background: radial-gradient(circle, color-mix(in srgb, var(--app-primary) 22%, transparent), transparent 70%);
  top: -120px;
  left: -120px;
  animation: float 12s ease-in-out infinite;
}
.bg-glow-2 {
  width: 420px;
  height: 420px;
  background: radial-gradient(circle, rgba(34, 211, 238, 0.16), transparent 70%);
  bottom: -100px;
  right: -100px;
  animation: float 14s ease-in-out infinite reverse;
}
@keyframes float {
  0%, 100% { transform: translate(0, 0); }
  50% { transform: translate(30px, -20px); }
}

.page-head,
.search-zone,
.stats-row,
.review-list,
.pagination,
.loading,
.empty {
  position: relative;
  z-index: 1;
}

/* ========== 页面头部 ========== */
.page-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 20px;
  margin-bottom: 20px;
  padding: 20px 24px;
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.78);
  backdrop-filter: blur(14px);
  border: 1px solid rgba(255, 255, 255, 0.7);
  box-shadow: 0 10px 36px rgba(31, 38, 135, 0.1);
}

.head-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
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
.new-btn {
  border-radius: 12px;
  font-weight: 600;
  padding: 10px 22px;
  box-shadow: 0 10px 24px color-mix(in srgb, var(--app-primary) 30%, transparent);
}

/* ========== 搜索区 ========== */
.search-zone {
  margin-bottom: 20px;
  display: flex;
  justify-content: center;
}
.search-box {
  width: min(680px, 100%);
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px 6px 16px;
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.88);
  backdrop-filter: blur(12px);
  border: 1px solid color-mix(in srgb, var(--app-primary) 16%, transparent);
  box-shadow: 0 8px 32px rgba(31, 38, 135, 0.08);
  transition: box-shadow 0.3s ease, border-color 0.3s ease, transform 0.3s ease;
}
.search-box.focus {
  border-color: var(--app-primary);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--app-primary) 16%, transparent),
    0 12px 36px color-mix(in srgb, var(--app-primary) 22%, transparent);
  transform: translateY(-1px);
}
.search-ic {
  color: var(--app-primary);
  flex-shrink: 0;
}
.search-input {
  flex: 1;
  border: none;
  outline: none;
  background: transparent;
  font-size: 15px;
  color: #1e293b;
  min-width: 0;
}
.search-input::placeholder {
  color: #94a3b8;
}
.search-clear {
  flex-shrink: 0;
  color: #94a3b8;
}
.search-btn {
  border-radius: 11px;
  font-weight: 600;
  padding: 10px 22px;
}

/* ========== 统计卡片（点击筛选，对齐书架页） ========== */
.stats-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
  margin-bottom: 20px;
}
.stat-card {
  position: relative;
  overflow: hidden;
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px 22px;
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.85);
  backdrop-filter: blur(12px);
  border: 1px solid rgba(255, 255, 255, 0.6);
  box-shadow: 0 8px 32px rgba(31, 38, 135, 0.08);
  transition: transform 0.3s ease, box-shadow 0.3s ease;
  cursor: default;
}
.stat-card.clickable {
  cursor: pointer;
}
.stat-card:hover {
  transform: translateY(-4px) scale(1.02);
  box-shadow: 0 16px 40px rgba(31, 38, 135, 0.16);
}
.stat-card.active {
  border: 1.5px solid transparent;
  background:
    linear-gradient(rgba(255, 255, 255, 0.92), rgba(255, 255, 255, 0.92)) padding-box,
    linear-gradient(135deg, var(--app-primary), #8b5cf6, #22d3ee) border-box;
  box-shadow: 0 12px 32px rgba(99, 102, 241, 0.2);
  transform: translateY(-3px);
}
.stat-card.active::after {
  content: '';
  position: absolute;
  top: 12px;
  right: 12px;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--app-primary);
  box-shadow: 0 0 0 4px color-mix(in srgb, var(--app-primary) 18%, transparent);
}
.stat-icon {
  flex-shrink: 0;
  width: 50px;
  height: 50px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
}
.stat-body {
  min-width: 0;
}
.stat-value {
  font-size: 26px;
  font-weight: 700;
  color: #0f172a;
  line-height: 1.1;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
.stat-label {
  margin-top: 4px;
  font-size: 13px;
  color: #64748b;
}
.stat-shine {
  position: absolute;
  top: -60px;
  right: -60px;
  width: 120px;
  height: 120px;
  border-radius: 50%;
  opacity: 0.08;
  transform: scale(1);
  transition: transform 0.5s ease;
}
.stat-card:hover .stat-shine {
  transform: scale(1.8);
}
.stat-card.active .stat-shine {
  opacity: 0.16;
  transform: scale(1.35);
}

/* ========== 筛选提示条 ========== */
.filter-tip {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 16px;
  margin-bottom: 16px;
  border-radius: 12px;
  background: linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 8%, #fff), #fff);
  border: 1px solid color-mix(in srgb, var(--app-primary) 18%, transparent);
  font-size: 13px;
  color: #475569;
}
.filter-tip-ic {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: 8px;
  background: var(--app-gradient);
  color: #fff;
  flex-shrink: 0;
}
.filter-tip-text b {
  color: var(--app-primary);
}
.filter-tip-count {
  margin-left: auto;
  font-size: 12px;
  color: #94a3b8;
  white-space: nowrap;
}
.filter-tip-clear {
  flex-shrink: 0;
}

/* ========== 读后感列表（紧凑小卡片，多列） ========== */
.review-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 14px;
}
.review-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.review-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  border-radius: 14px;
  min-width: 0;
  background: rgba(255, 255, 255, 0.85);
  backdrop-filter: blur(12px);
  border: 1px solid rgba(255, 255, 255, 0.6);
  box-shadow: 0 6px 24px rgba(31, 38, 135, 0.07);
  cursor: pointer;
  transition: transform 0.25s ease, box-shadow 0.25s ease, border-color 0.25s ease;
}
.review-card:hover {
  transform: translateY(-2px);
  border-color: color-mix(in srgb, var(--app-primary) 35%, transparent);
  box-shadow: 0 12px 28px color-mix(in srgb, var(--app-primary) 18%, rgba(31, 38, 135, 0.14));
}
.review-cover {
  flex-shrink: 0;
  width: 50px;
  height: 70px;
  border-radius: 8px;
  overflow: hidden;
  background: linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 55%, transparent), rgba(34, 211, 238, 0.55));
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 4px 12px rgba(31, 38, 135, 0.16);
}
.review-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.cover-fallback {
  color: #fff;
  opacity: 0.92;
}
.review-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.review-head {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.review-book {
  font-size: 11px;
  color: var(--app-primary);
  background: color-mix(in srgb, var(--app-primary) 10%, transparent);
  border: 1px solid color-mix(in srgb, var(--app-primary) 22%, transparent);
  padding: 1px 8px;
  border-radius: 999px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 120px;
  flex-shrink: 0;
}
.review-rate {
  margin-left: auto;
  flex-shrink: 0;
  display: flex;
}
.review-card :deep(.el-rate__icon) {
  font-size: 13px;
  margin-right: 1px;
}
.review-title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: #1e293b;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.review-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.review-time {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11.5px;
  color: #94a3b8;
  white-space: nowrap;
}
.review-actions {
  display: flex;
  align-items: center;
  gap: 2px;
  opacity: 0.75;
  transition: opacity 0.2s ease;
}
.review-card:hover .review-actions {
  opacity: 1;
}
.review-actions :deep(.el-button) {
  padding: 4px 6px;
  font-size: 12px;
}

/* ========== 加载 / 空状态 / 分页 ========== */
.loading,
.empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 0;
  color: #94a3b8;
}
.empty-icon {
  width: 80px;
  height: 80px;
  border-radius: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--app-primary);
  background: color-mix(in srgb, var(--app-primary) 10%, transparent);
  margin-bottom: 16px;
}
.empty-title {
  margin: 0 0 6px;
  font-size: 16px;
  font-weight: 600;
  color: #475569;
}
.empty-sub {
  margin: 0 0 18px;
  font-size: 13px;
}
.pagination {
  display: flex;
  justify-content: flex-end;
  padding: 28px 0;
}

/* ========== 阅读详情弹窗 ========== */
.review-dialog.el-overlay {
  background: rgba(15, 23, 42, 0.45);
  backdrop-filter: blur(6px);
  -webkit-backdrop-filter: blur(6px);
}
.review-dialog :deep(.el-dialog) {
  border-radius: 20px;
  overflow: hidden;
  background: #fff;
  box-shadow: 0 24px 80px rgba(15, 23, 42, 0.3);
}
.review-dialog :deep(.el-dialog__header) {
  margin-right: 0;
  padding: 20px 24px 16px;
  background: linear-gradient(135deg,
    color-mix(in srgb, var(--app-primary) 7%, #fff),
    color-mix(in srgb, #22d3ee 6%, #fff));
  border-bottom: 1px solid color-mix(in srgb, var(--app-primary) 10%, transparent);
}
.review-dialog :deep(.el-dialog__headerbtn) {
  top: 20px;
  right: 20px;
  width: 30px;
  height: 30px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.8);
  border: 1px solid color-mix(in srgb, var(--app-primary) 14%, transparent);
  box-shadow: 0 4px 12px rgba(31, 38, 135, 0.08);
}
.review-dialog :deep(.el-dialog__headerbtn:hover) {
  background: color-mix(in srgb, var(--app-primary) 12%, #fff);
}
.review-dialog :deep(.el-dialog__body) {
  padding: 20px 24px;
}
.review-dialog :deep(.el-dialog__footer) {
  padding: 0 24px 18px;
}
.rd-head {
  display: flex;
  align-items: center;
  gap: 16px;
  padding-right: 40px;
}
.rd-cover {
  flex-shrink: 0;
  width: 58px;
  height: 82px;
  border-radius: 10px;
  overflow: hidden;
  background: var(--app-gradient);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 8px 20px color-mix(in srgb, var(--app-primary) 28%, transparent);
}
.rd-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.rd-cover-fallback {
  color: #fff;
}
.rd-meta {
  min-width: 0;
}
.rd-title {
  margin: 0;
  font-size: 18px;
  font-weight: 700;
  color: #0f172a;
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}
.rd-sub {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  margin-top: 8px;
}
.rd-book {
  font-size: 12px;
  color: var(--app-primary);
  background: color-mix(in srgb, var(--app-primary) 10%, transparent);
  border: 1px solid color-mix(in srgb, var(--app-primary) 22%, transparent);
  padding: 2px 10px;
  border-radius: 999px;
  white-space: nowrap;
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
}
.rd-rate {
  display: flex;
  align-items: center;
}
.rd-rate :deep(.el-rate__icon) {
  font-size: 16px;
  margin-right: 2px;
}
.rd-time {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: #94a3b8;
}
.rd-body {
  line-height: 1.9;
  color: #334155;
  font-size: 14.5px;
  max-height: 56vh;
  overflow-y: auto;
  padding: 2px 6px 2px 2px;
}
.rd-body::-webkit-scrollbar {
  width: 8px;
}
.rd-body::-webkit-scrollbar-thumb {
  background: color-mix(in srgb, var(--app-primary) 25%, #cbd5e1);
  border-radius: 4px;
}
.rd-body::-webkit-scrollbar-thumb:hover {
  background: color-mix(in srgb, var(--app-primary) 40%, #cbd5e1);
}
.rd-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  width: 100%;
  padding-top: 14px;
  border-top: 1px solid #f1f5f9;
}
.rd-stat {
  display: flex;
  align-items: center;
  gap: 6px;
  color: #94a3b8;
  font-size: 12px;
  white-space: nowrap;
}
.rd-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

/* ========== 响应式 ========== */
@media (max-width: 900px) {
  .stats-row {
    grid-template-columns: repeat(2, 1fr);
    gap: 12px;
  }
  .review-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  .page-head {
    padding: 16px;
  }
}
@media (max-width: 640px) {
  .stats-row {
    grid-template-columns: repeat(2, 1fr);
    gap: 10px;
  }
  .stat-card {
    padding: 14px 12px;
  }
  .stat-icon {
    width: 40px;
    height: 40px;
  }
  .stat-value {
    font-size: 20px;
  }
  .head-title-icon {
    width: 40px;
    height: 40px;
  }
  .head-title-text h2 {
    font-size: 18px;
  }
  .head-title-text p {
    display: none;
  }
  .search-box {
    padding: 5px 6px 5px 12px;
  }
  .search-btn {
    padding: 8px 14px;
  }
  .review-grid {
    grid-template-columns: 1fr;
  }
  .review-card {
    padding: 10px 12px;
  }
  .review-cover {
    width: 46px;
    height: 64px;
  }
  .review-dialog :deep(.el-dialog__header) {
    padding: 16px 16px 12px;
  }
  .review-dialog :deep(.el-dialog__body) {
    padding: 16px;
  }
  .review-dialog :deep(.el-dialog__footer) {
    padding: 0 16px 14px;
  }
  .rd-head {
    gap: 12px;
  }
  .rd-cover {
    width: 48px;
    height: 68px;
  }
  .rd-foot {
    flex-direction: column;
    align-items: stretch;
  }
  .rd-actions {
    justify-content: flex-end;
  }
}
</style>

<style>
/* Markdown 渲染样式（非 scoped，供 v-html 内容使用） */
.markdown-body {
  font-size: 14px;
  color: #334155;
  word-break: break-word;
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
.markdown-body p {
  margin: 0.7em 0;
  line-height: 1.8;
}
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
  line-height: 1.6;
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
.markdown-body li {
  margin: 0.3em 0;
  line-height: 1.7;
}
.markdown-body hr {
  border: none;
  border-top: 1px solid #e2e8f0;
  margin: 1.4em 0;
}
.markdown-body img {
  max-width: 100%;
  border-radius: 10px;
}
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
