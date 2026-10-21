<template>
  <div class="bookshelf">
    <!-- 背景光晕 -->
    <div class="bg-glow bg-glow-1"></div>
    <div class="bg-glow bg-glow-2"></div>

    <!-- 页面头部：标题 + 搜索 -->
    <div class="page-head">
      <div class="head-title">
        <span class="head-title-icon">
          <el-icon :size="22"><Reading /></el-icon>
        </span>
        <div class="head-title-text">
          <h2>我的书架</h2>
          <p>管理你的藏书，继续阅读之旅</p>
        </div>
      </div>
      <div class="head-search">
        <el-input
          v-model="searchQuery"
          placeholder="搜索书名 / 作者..."
          clearable
          size="large"
          class="search-input"
          @keyup.enter="applySearch"
          @clear="applySearch"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
      </div>
    </div>

    <!-- 统计栏（点击可筛选阅读状态） -->
    <div class="stats-row">
      <div
        v-for="s in statCards"
        :key="s.label"
        class="stat-card"
        :class="[s.status !== null ? 'clickable' : '', { active: s.status !== null && activeStatus === s.status }]"
        :title="s.status !== null ? (s.status === '' ? '显示全部图书' : '点击筛选' + s.label) : ''"
        @click="setStatus(s.status)"
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

    <!-- 分类筛选 + 视图方式 -->
    <div class="view-toolbar">
      <div class="vt-cats">
        <el-select
          v-model="activeCategory"
          placeholder="全部分类"
          clearable
          size="large"
          class="cat-select"
          popper-class="cat-popper"
          @change="setCategory"
          @clear="setCategory('')"
        >
          <template #prefix>
            <el-icon><Collection /></el-icon>
          </template>
          <el-option
            v-for="c in categories"
            :key="c.category"
            :label="c.category"
            :value="c.category"
          >
            <div class="cat-opt">
              <span class="cat-opt-name">
                <el-icon><Folder /></el-icon>
                <span class="cat-opt-text">{{ c.category }}</span>
              </span>
              <span class="cat-opt-count">{{ c.count }} 本</span>
            </div>
          </el-option>
        </el-select>
      </div>

      <div class="vt-divider"></div>

      <div class="vt-controls">
        <div class="vt-group" v-if="!isMobile">
          <span class="vt-label">
            <el-icon><Grid /></el-icon> 视图
          </span>
          <el-radio-group v-model="viewMode" size="small">
            <el-radio-button label="grid">网格</el-radio-button>
            <el-radio-button label="list">列表</el-radio-button>
          </el-radio-group>
        </div>
        <div class="vt-group">
          <span class="vt-label">
            <el-icon><Sort /></el-icon> 排序
          </span>
          <el-radio-group v-model="sortBy" size="small">
            <el-radio-button label="created_at">时间</el-radio-button>
            <el-radio-button label="title">书名</el-radio-button>
            <el-radio-button label="last_read">最近阅读</el-radio-button>
          </el-radio-group>
        </div>
      </div>
    </div>

    <!-- 阅读状态筛选提示 -->
    <div v-if="activeStatus" class="filter-tip">
      <span class="filter-tip-ic"><el-icon :size="14"><Filter /></el-icon></span>
      <span class="filter-tip-text">正在筛选：<b>{{ activeStatus === 'reading' ? '正在阅读' : '已读完' }}</b> 的图书</span>
      <span class="filter-tip-count">共 {{ total }} 本</span>
      <el-button size="small" text type="primary" class="filter-tip-clear" @click="setStatus('')">
        清除筛选
      </el-button>
    </div>

    <!-- 加载中 -->
    <div v-if="loading" class="loading">
      <div class="skeleton-grid">
        <div v-for="i in 8" :key="i" class="skeleton-card">
          <div class="skeleton-cover"></div>
          <div class="skeleton-line"></div>
          <div class="skeleton-line short"></div>
        </div>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-else-if="books.length === 0" class="empty">
      <div class="empty-icon">
        <el-icon :size="56"><Notebook /></el-icon>
      </div>
      <template v-if="activeStatus">
        <h3>没有符合条件的图书</h3>
        <p>{{ activeStatus === 'reading' ? '当前没有「正在阅读」的图书，读完一本书后会自动归入该筛选' : '当前还没有「已读完」的图书，读完后会自动归入该筛选' }}</p>
        <el-button type="primary" size="large" round @click="setStatus('')">查看全部图书</el-button>
      </template>
      <template v-else>
        <h3>书架空空如也</h3>
        <p>导入你的第一本电子书，开启阅读之旅</p>
        <el-button type="primary" size="large" round @click="$router.push('/upload')">去导入图书</el-button>
      </template>
    </div>

    <!-- 网格视图 -->
    <div v-else-if="viewMode === 'grid'" class="grid-view">
      <el-row :gutter="20">
        <el-col
          v-for="(book, index) in books"
          :key="book.id"
          :xs="12" :sm="8" :md="6" :lg="4" :xl="3"
          class="grid-col"
          :style="{ '--i': index }"
        >
          <BookCard :book="book" :index="index" show-meta @click="goToDetail(book.id)">
            <template #actions>
              <el-button
                v-if="book.file_type === 'epub' || book.file_type === 'kepub' || book.file_type === 'pdf' || book.file_type === 'azw3' || book.file_type === 'mobi'"
                type="primary"
                size="small"
                circle
                class="action-btn"
                @click="goRead(book.id)"
              >
                <el-icon><Reading /></el-icon>
              </el-button>
              <el-button
                v-if="isAudiobookSupported(book.file_type)"
                size="small"
                circle
                class="action-btn audio-btn"
                title="生成有声书"
                @click="generateAudiobook(book)"
              >
                <el-icon><Headset /></el-icon>
              </el-button>
              <el-button
                size="small"
                circle
                class="action-btn share-btn"
                title="分享"
                @click="openShareDialog(book)"
              >
                <el-icon><Share /></el-icon>
              </el-button>
              <el-dropdown trigger="click">
                <el-button size="small" circle class="action-btn">
                  <el-icon><MoreFilled /></el-icon>
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item @click="goToDetail(book.id)">详情</el-dropdown-item>
                    <el-dropdown-item @click="openShareDialog(book)">分享</el-dropdown-item>
                    <el-dropdown-item @click="downloadBook(book.id)">下载</el-dropdown-item>
                    <el-dropdown-item v-if="isAudiobookSupported(book.file_type)" @click="generateAudiobook(book)">
                      生成有声书
                    </el-dropdown-item>
                    <el-dropdown-item @click="editBook(book)">编辑</el-dropdown-item>
                    <el-dropdown-item divided @click="deleteBook(book.id)" style="color: #f56c6c">
                      删除
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </template>
          </BookCard>
        </el-col>
      </el-row>
    </div>

    <!-- 列表视图 -->
    <div v-else class="list-view">
      <el-table :data="books" class="book-table">
        <el-table-column prop="title" label="书名" min-width="240">
          <template #default="{ row }">
            <div class="table-book">
              <div class="table-cover-box">
                <img v-if="row.cover_path" :src="coverUrl(row.id)" class="table-cover" />
                <div v-else class="table-cover-placeholder">
                  <el-icon><Reading /></el-icon>
                </div>
              </div>
              <div class="table-book-text">
                <div class="table-title">{{ row.title }}</div>
                <div class="table-author">{{ formatAuthors(row.authors) }}</div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="file_type" label="格式" width="140">
          <template #default="{ row }">
            <div class="table-tags">
              <span v-if="row.category" class="cat-tag" :title="row.category">{{ row.category }}</span>
              <span class="type-tag">{{ row.file_type }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="progress" label="进度" width="160">
          <template #default="{ row }">
            <div class="table-progress">
              <el-progress
                :percentage="Math.round(row.progress)"
                :stroke-width="8"
                :show-text="false"
                color="linear-gradient(90deg,var(--app-primary),#22d3ee)"
              />
              <span class="progress-text">{{ Math.round(row.progress) }}%</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="last_read" label="最近阅读" width="140">
          <template #default="{ row }">
            <span class="read-time">{{ row.last_read ? formatDate(row.last_read) : '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="190" fixed="right">
          <template #default="{ row }">
            <div class="op-actions">
              <el-button
                v-if="row.file_type === 'epub' || row.file_type === 'kepub' || row.file_type === 'pdf' || row.file_type === 'azw3' || row.file_type === 'mobi'"
                type="primary"
                size="small"
                class="action-btn"
                @click="goRead(row.id)"
              >
                阅读
              </el-button>
              <el-button size="small" class="action-btn" @click="goToDetail(row.id)">详情</el-button>
              <el-dropdown trigger="click">
                <span class="op-more">更多</span>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item @click="openShareDialog(row)">分享</el-dropdown-item>
                    <el-dropdown-item v-if="isAudiobookSupported(row.file_type)" @click="generateAudiobook(row)">
                      生成有声书
                    </el-dropdown-item>
                    <el-dropdown-item @click="downloadBook(row.id)">下载</el-dropdown-item>
                    <el-dropdown-item @click="editBook(row)">编辑</el-dropdown-item>
                    <el-dropdown-item divided @click="deleteBook(row.id)" style="color: #f56c6c">
                      删除
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 分页（每页条数可自定义，网格为两行倍数、列表为 10 倍数） -->
    <div class="pagination" v-if="total > pageSize">
      <PaginationBar
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        :page-sizes="pageSizesOptions"
        @current-change="loadBooks"
        @size-change="onSizeChange"
      />
    </div>

    <!-- 分享对话框 -->
    <el-dialog
      v-model="shareVisible"
      :title="`分享图书${shareBook ? '：' + shareBook.title : ''}`"
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

    <!-- 编辑对话框 -->
    <el-dialog v-model="editDialogVisible" title="编辑图书" width="500px">
      <el-form :model="editForm" label-width="80px">
        <el-form-item label="书名">
          <el-input v-model="editForm.title" />
        </el-form-item>
        <el-form-item label="作者">
          <el-input v-model="editForm.authors" placeholder="多个作者用逗号分隔" />
        </el-form-item>
        <el-form-item label="出版社">
          <el-input v-model="editForm.publisher" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="editForm.description" type="textarea" :rows="3" />
        </el-form-item>
        <el-form-item label="标签">
          <el-input v-model="editForm.tags" placeholder="多个标签用逗号分隔" />
        </el-form-item>
        <el-form-item label="分类">
          <el-select
            v-model="editForm.category"
            placeholder="选择或输入分类"
            clearable
            filterable
            allow-create
            default-first-option
            style="width: 100%"
          >
            <el-option v-for="c in categoryOptions" :key="c" :label="c" :value="c" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveEdit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted, reactive } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { bookApi } from '@/api/book'
import { useMobileForceGrid } from '@/composables/useMobileView'
import { getShareConfig, shareApi } from '@/api/share'
import { copyText } from '@/utils/clipboard'
import BookCard from '@/components/BookCard.vue'
import PaginationBar from '@/components/PaginationBar.vue'

const route = useRoute()
const router = useRouter()

const books = ref([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const VIEW_MODE_KEY = 'bookshelf_view_mode'
const viewMode = ref(
  window.innerWidth <= 768 ? 'grid' : localStorage.getItem(VIEW_MODE_KEY) || 'grid'
)
// 手机端强制网格视图（取消列表视图），并暴露 isMobile 供模板隐藏切换器
const { isMobile } = useMobileForceGrid(viewMode)

// 每页条数基准：网格视图 = 当前断点每行列数 × 2（即两行）；列表视图 = 10
const GRID_BREAKPOINTS = [
  { min: 1920, cols: 8 },
  { min: 1200, cols: 6 },
  { min: 992, cols: 4 },
  { min: 768, cols: 3 }
]
const gridColumns = () => {
  const w = window.innerWidth
  for (const b of GRID_BREAKPOINTS) {
    if (w >= b.min) return b.cols
  }
  return 2 // xs：每行列数
}
const pageBase = computed(() => (viewMode.value === 'grid' ? gridColumns() * 2 : 10))
// 分页条数选项：网格为两行的倍数，列表为 10 的倍数
const pageSizesOptions = computed(() => [
  pageBase.value,
  pageBase.value * 2,
  pageBase.value * 3,
  pageBase.value * 4
])
const sizeKey = (mode) => (mode === 'grid' ? 'bookshelf_page_size_grid' : 'bookshelf_page_size_list')
const persistedSize = (mode) => {
  const n = Number(localStorage.getItem(sizeKey(mode)))
  return n > 0 ? n : null
}
// 默认取该视图已保存的分页条数，未保存则取基准（两行 / 10）
const pageSize = ref(persistedSize(viewMode.value) || pageBase.value)
const sortBy = ref('created_at')
const searchQuery = ref(route.query.q || '')
const stats = ref({ total_books: 0, reading_count: 0, finished_count: 0, total_pages: 0 })
const categories = ref([])
const activeCategory = ref('')

const editDialogVisible = ref(false)
const editForm = ref({})
const editingId = ref(null)

// 当前阅读状态筛选：'' 全部 / reading 在读 / finished 已读完（null 表示不可筛选）
const activeStatus = ref('')
const setStatus = (status) => {
  if (status === null) return
  if (activeStatus.value === status) return
  activeStatus.value = status
  page.value = 1
  loadBooks()
}

const statCards = computed(() => [
  {
    label: '藏书总数',
    value: displayedStats.total_books,
    icon: 'Collection',
    gradient: 'linear-gradient(135deg,var(--app-primary),#8b5cf6)',
    glow: 'rgba(99,102,241,0.45)',
    status: ''
  },
  {
    label: '正在阅读',
    value: displayedStats.reading_count,
    icon: 'Reading',
    gradient: 'linear-gradient(135deg,#06b6d4,#22d3ee)',
    glow: 'rgba(6,182,212,0.45)',
    status: 'reading'
  },
  {
    label: '已读完',
    value: displayedStats.finished_count,
    icon: 'CircleCheck',
    gradient: 'linear-gradient(135deg,#10b981,#34d399)',
    glow: 'rgba(16,185,129,0.45)',
    status: 'finished'
  },
  {
    label: '总页数',
    value: displayedStats.total_pages,
    icon: 'Document',
    gradient: 'linear-gradient(135deg,#f59e0b,#fbbf24)',
    glow: 'rgba(245,158,11,0.45)',
    status: null
  }
])

const bookStatus = (book) => {
  if (book.progress >= 100) return { text: '已读完', cls: 'done' }
  if (book.progress > 0) return { text: '在读', cls: 'reading' }
  if (book.last_read) return { text: '读过', cls: 'read' }
  return { text: '未读', cls: 'new' }
}

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

const formatDate = (dateStr) => {
  if (!dateStr) return '-'
  const d = new Date(dateStr)
  return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')}`
}

const loadBooks = async () => {
  loading.value = true
  try {
    const params = {
      page: page.value,
      page_size: pageSize.value,
      sort: sortBy.value,
      order: 'desc',
      q: route.query.q || '',
      ...(activeCategory.value ? { category: activeCategory.value } : {}),
      ...(activeStatus.value ? { status: activeStatus.value } : {})
    }
    const { data } = await bookApi.listBooks(params)
    books.value = data.data || []
    total.value = data.total || 0
  } catch (e) {
    // 列表加载失败不弹错误提醒（空数据/网络抖动属正常情况），静默降级为空列表
    console.warn('加载图书列表失败', e)
  } finally {
    loading.value = false
  }
}

// 视口（断点）变化时按“两行”重设每页条数，变化时回到第一页并重新加载
let lastBase = pageBase.value
let resizeTimer = null
const applyAutoPageSize = () => {
  const next = pageBase.value
  if (next === lastBase) return
  lastBase = next
  pageSize.value = next
  page.value = 1
  loadBooks()
}
const onResize = () => {
  if (resizeTimer) clearTimeout(resizeTimer)
  resizeTimer = setTimeout(applyAutoPageSize, 200)
}

// 用户自定义分页条数：记住该视图的选择并重新加载
const onSizeChange = () => {
  page.value = 1
  localStorage.setItem(sizeKey(viewMode.value), String(pageSize.value))
  loadBooks()
}

const loadStats = async () => {
  try {
    const { data } = await bookApi.getStats()
    stats.value = data
    animateStats(data)
  } catch (e) {
    // ignore
  }
}

// ===== 统计数字滚动动画 =====
const displayedStats = reactive({ total_books: 0, reading_count: 0, finished_count: 0, total_pages: 0 })
let statsAnimTimer = null

const animateStats = (target) => {
  if (statsAnimTimer) cancelAnimationFrame(statsAnimTimer)
  const from = { ...displayedStats }
  const to = {
    total_books: target.total_books || 0,
    reading_count: target.reading_count || 0,
    finished_count: target.finished_count || 0,
    total_pages: target.total_pages || 0
  }
  const duration = 700
  const start = performance.now()
  const ease = (t) => 1 - Math.pow(1 - t, 3)
  const tick = (now) => {
    const p = Math.min((now - start) / duration, 1)
    const e = ease(p)
    displayedStats.total_books = Math.round(from.total_books + (to.total_books - from.total_books) * e)
    displayedStats.reading_count = Math.round(from.reading_count + (to.reading_count - from.reading_count) * e)
    displayedStats.finished_count = Math.round(from.finished_count + (to.finished_count - from.finished_count) * e)
    displayedStats.total_pages = Math.round(from.total_pages + (to.total_pages - from.total_pages) * e)
    if (p < 1) statsAnimTimer = requestAnimationFrame(tick)
  }
  statsAnimTimer = requestAnimationFrame(tick)
}

const loadCategories = async () => {
  try {
    const { data } = await bookApi.getCategories()
    categories.value = data.data || []
  } catch (e) {
    // ignore
  }
}

const setCategory = (cat) => {
  activeCategory.value = cat
  page.value = 1
  loadBooks()
}

const categoryOptions = computed(() => categories.value.map(c => c.category))

const goToDetail = (id) => router.push(`/book/${id}`)
const goRead = (id) => router.push(`/reader/${id}`)

// 支持生成有声书的格式（与后端保持一致）
const AUDIOBOOK_TYPES = ['epub', 'kepub', 'azw3', 'mobi', 'pdf', 'md', 'txt']
const isAudiobookSupported = (ft) => AUDIOBOOK_TYPES.includes((ft || '').toLowerCase())

// 跳转到有声书页并携带书籍参数，由有声书页自动弹出生成弹窗
const generateAudiobook = (book) => {
  router.push({
    path: '/audiobooks',
    query: {
      book_id: book.id,
      title: book.title || '',
      cover_path: book.cover_path || '',
      file_type: book.file_type || ''
    }
  })
}

// 搜索：回车后同步到路由 query，触发 watch 重新加载
const applySearch = () => {
  page.value = 1
  const q = searchQuery.value.trim()
  if (route.query.q === q || (!route.query.q && !q)) {
    loadBooks()
  } else {
    router.push({ path: '/bookshelf', query: q ? { q } : {} })
  }
}

const editBook = (book) => {
  editingId.value = book.id
  editForm.value = {
    title: book.title,
    authors: formatAuthors(book.authors),
    publisher: book.publisher || '',
    description: book.description || '',
    tags: (() => {
      try { return JSON.parse(book.tags || '[]').join(', ') } catch { return '' }
    })(),
    category: book.category || ''
  }
  editDialogVisible.value = true
}

const saveEdit = async () => {
  try {
    await bookApi.updateBook(editingId.value, {
      title: editForm.value.title,
      authors: editForm.value.authors.split(/[,，]/).map(s => s.trim()).filter(Boolean),
      publisher: editForm.value.publisher,
      description: editForm.value.description,
      tags: editForm.value.tags.split(/[,，]/).map(s => s.trim()).filter(Boolean),
      category: editForm.value.category || ''
    })
    ElMessage.success('保存成功')
    editDialogVisible.value = false
    loadBooks()
  } catch (e) {
    ElMessage.error('保存失败')
  }
}

const deleteBook = async (id) => {
  try {
    await ElMessageBox.confirm('确定要删除这本图书吗？', '提示', { type: 'warning', confirmButtonText: '确定', cancelButtonText: '取消' })
    await bookApi.deleteBook(id)
    ElMessage.success('删除成功')
    loadBooks()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error('删除失败')
  }
}

const downloadBook = async (id) => {
  try {
    const { data, headers } = await bookApi.downloadBook(id)
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

// ===== 分享 =====
const shareVisible = ref(false)
const shareBook = ref(null)
const shares = ref([])
const shareExpiresDays = ref(7)
const creatingShare = ref(false)
const loadingShares = ref(false)
// 服务端口（飞牛 service_port）：分享链接用端口直连，不走统一网关
const servicePort = ref('')

const extractError = (e) => {
  const msg = e?.response?.data?.error || e?.message || '操作失败'
  return typeof msg === 'string' ? msg : JSON.stringify(msg)
}

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

const openShareDialog = (book) => {
  shareBook.value = book
  shareVisible.value = true
  shareExpiresDays.value = 7
  loadServicePort()
  loadShares()
}

const loadShares = async () => {
  if (!shareBook.value) return
  loadingShares.value = true
  try {
    const { data } = await shareApi.listShares(shareBook.value.id)
    shares.value = data.data || []
  } catch (e) {
    shares.value = []
  } finally {
    loadingShares.value = false
  }
}

const createShare = async () => {
  creatingShare.value = true
  try {
    const { data } = await shareApi.createShare(shareBook.value.id, shareExpiresDays.value)
    shares.value.unshift(data)
    copyShare(data.token)
  } catch (e) {
    ElMessage.error(extractError(e))
  } finally {
    creatingShare.value = false
  }
}

const copyShare = async (token) => {
  const url = shareUrl(token)
  const ok = await copyText(url)
  if (ok) {
    ElMessage.success('链接已复制')
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
    if (e !== 'cancel') ElMessage.error(extractError(e))
  }
}

watch(viewMode, (v) => {
  // 手机端不持久化视图偏好，避免覆盖桌面端设置
  if (window.innerWidth > 768) localStorage.setItem(VIEW_MODE_KEY, v)
  // 切换到该视图已保存的分页条数，未保存则取基准（两行 / 10）
  pageSize.value = persistedSize(v) || pageBase.value
  lastBase = pageBase.value
  page.value = 1
  loadBooks()
})
watch(sortBy, () => { page.value = 1; loadBooks() })
watch(() => route.query.q, (val) => {
  searchQuery.value = val || ''
  page.value = 1
  loadBooks()
})

onMounted(() => {
  loadBooks()
  loadStats()
  loadCategories()
  window.addEventListener('resize', onResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', onResize)
  if (resizeTimer) clearTimeout(resizeTimer)
})
</script>

<style scoped>
.bookshelf {
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
.stats-row,
.toolbar,
.filter-tip,
.grid-view,
.list-view,
.pagination,
.loading,
.empty {
  position: relative;
  z-index: 1;
}

/* ========== 筛选提示条 ========== */
.filter-tip {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 16px;
  margin-bottom: 14px;
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
}
.filter-tip-clear {
  flex-shrink: 0;
}

/* ========== 页面头部（标题 + 搜索） ========== */
.page-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 24px;
  padding: 20px 24px;
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.78);
  backdrop-filter: blur(14px);
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

/* ========== 统计卡片 ========== */
.stats-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
  margin-bottom: 24px;
}

.stat-card {
  position: relative;
  overflow: hidden;
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 22px 24px;
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.85);
  backdrop-filter: blur(12px);
  border: 1px solid rgba(255, 255, 255, 0.6);
  box-shadow: 0 8px 32px rgba(31, 38, 135, 0.08);
  transition: transform 0.3s ease, box-shadow 0.3s ease;
  cursor: default;
}

.stat-card:hover {
  transform: translateY(-4px) scale(1.02);
  box-shadow: 0 16px 40px rgba(31, 38, 135, 0.16);
}

.stat-icon {
  flex-shrink: 0;
  width: 52px;
  height: 52px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: #0f172a;
  line-height: 1.1;
  font-variant-numeric: tabular-nums;
}

.stat-label {
  font-size: 13px;
  color: #64748b;
  margin-top: 4px;
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

/* 可点击的统计卡（筛选入口） */
.stat-card.clickable {
  cursor: pointer;
}
.stat-card.clickable:hover {
  box-shadow: 0 16px 40px rgba(31, 38, 135, 0.16);
}
/* 选中态：渐变描边 + 高亮光斑 */
.stat-card.active {
  border: 1.5px solid transparent;
  background:
    linear-gradient(rgba(255, 255, 255, 0.92), rgba(255, 255, 255, 0.92)) padding-box,
    linear-gradient(135deg, var(--app-primary), #8b5cf6, #22d3ee) border-box;
  box-shadow: 0 12px 32px rgba(99, 102, 241, 0.2);
  transform: translateY(-3px);
}
.stat-card.active .stat-shine {
  opacity: 0.16;
  transform: scale(1.35);
}
/* 选中标识：右上角小圆点 */
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

/* ========== 分类筛选 + 视图工具栏 ========== */
.view-toolbar {
  position: relative;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 14px;
  margin-bottom: 20px;
  padding: 12px 16px;
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.62);
  backdrop-filter: blur(16px) saturate(1.4);
  -webkit-backdrop-filter: blur(16px) saturate(1.4);
  border: 1px solid rgba(255, 255, 255, 0.7);
  box-shadow: 0 8px 28px rgba(31, 38, 135, 0.08), inset 0 1px 0 rgba(255, 255, 255, 0.8);
}
.vt-cats {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
  padding: 2px;
}
.cat-select {
  width: 262px;
  flex-shrink: 0;
}
/* 触发区：毛玻璃 + 渐变描边 + 聚焦发光 */
.cat-select :deep(.el-select__wrapper) {
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.78);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--app-primary) 22%, transparent), 0 4px 16px color-mix(in srgb, var(--app-primary) 10%, transparent);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
  padding: 4px 14px;
  min-height: 42px;
  transition: all 0.25s ease;
}
.cat-select :deep(.el-select__wrapper:hover) {
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--app-primary) 45%, transparent), 0 6px 20px color-mix(in srgb, var(--app-primary) 16%, transparent);
}
.cat-select :deep(.el-select__wrapper.is-focused) {
  box-shadow: inset 0 0 0 1.5px var(--app-primary), 0 0 0 4px color-mix(in srgb, var(--app-primary) 14%, transparent), 0 8px 24px color-mix(in srgb, var(--app-primary) 22%, transparent);
}
.cat-select :deep(.el-select__prefix) {
  color: var(--app-primary);
  margin-right: 2px;
}
.cat-select :deep(.el-select__selected-item) {
  font-size: 14px;
  font-weight: 600;
  color: #334155;
}
.cat-select :deep(.el-select__placeholder) {
  color: #94a3b8;
  font-weight: 500;
}
.cat-select :deep(.el-select__caret) {
  color: #94a3b8;
}
.cat-opt {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  width: 100%;
}
.cat-opt-name {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  max-width: 100%;
}
.cat-opt-text {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.cat-opt-count {
  flex-shrink: 0;
  font-size: 12px;
  font-weight: 600;
  color: var(--app-primary);
  background: linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 12%, transparent), rgba(34, 211, 238, 0.12));
  border: 1px solid color-mix(in srgb, var(--app-primary) 16%, transparent);
  border-radius: 999px;
  padding: 0 9px;
  line-height: 20px;
}

.vt-divider {
  width: 1px;
  height: 34px;
  flex-shrink: 0;
  background: linear-gradient(180deg, transparent, color-mix(in srgb, var(--app-primary) 35%, transparent), transparent);
}

.vt-controls {
  display: flex;
  align-items: center;
  gap: 14px;
  flex-shrink: 0;
}

.vt-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.vt-label {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: #64748b;
  font-weight: 500;
  white-space: nowrap;
}

.vt-label .el-icon {
  color: var(--app-primary);
}

.vt-controls .el-radio-button__inner {
  border-color: color-mix(in srgb, var(--app-primary) 25%, transparent);
  background: rgba(255, 255, 255, 0.65);
  color: #64748b;
}
.vt-controls .el-radio-button:first-child .el-radio-button__inner {
  border-left-color: color-mix(in srgb, var(--app-primary) 25%, transparent);
}
.vt-controls .el-radio-button__original-radio:checked + .el-radio-button__inner {
  background: var(--app-gradient);
  border-color: transparent;
  color: #fff;
  box-shadow: 0 4px 14px color-mix(in srgb, var(--app-primary) 35%, transparent);
}

/* ========== 图书卡片（网格） ========== */
.grid-col {
  animation: cardIn 0.45s cubic-bezier(0.22, 1, 0.36, 1) both;
  animation-delay: calc(var(--i, 0) * 40ms);
}

@keyframes cardIn {
  from {
    opacity: 0;
    transform: translateY(18px) scale(0.96);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

.grid-view .book-card {
  position: relative;
  border-radius: 10px;
  overflow: hidden;
  cursor: pointer;
  margin-bottom: 20px;
  background: #fff;
  box-shadow: 0 4px 16px rgba(31, 38, 135, 0.08);
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1), box-shadow 0.3s ease;
}

.grid-view .book-card::before {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: 10px;
  padding: 1.5px;
  background: linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 50%, transparent), rgba(34, 211, 238, 0.5), rgba(139, 92, 246, 0.5));
  -webkit-mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
  -webkit-mask-composite: xor;
  mask-composite: exclude;
  opacity: 0;
  transition: opacity 0.3s ease;
  pointer-events: none;
}

.grid-view .book-card:hover {
  transform: translateY(-8px) scale(1.02);
  box-shadow: 0 20px 48px color-mix(in srgb, var(--app-primary) 28%, transparent);
}

.grid-view .book-card:hover::before {
  opacity: 1;
}

.book-cover-wrapper {
  position: relative;
  aspect-ratio: 3/4;
  overflow: hidden;
  background: linear-gradient(160deg, #1e293b, #0f172a);
  display: flex;
  align-items: center;
  justify-content: center;
}

.book-cover {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 0.4s ease;
}

.book-card:hover .book-cover {
  transform: scale(1.08);
}

.cover-shine {
  position: absolute;
  inset: 0;
  background: linear-gradient(105deg, transparent 40%, rgba(255, 255, 255, 0.25) 50%, transparent 60%);
  transform: translateX(-120%);
  transition: transform 0.7s ease;
  pointer-events: none;
}

.book-card:hover .cover-shine {
  transform: translateX(120%);
}

.book-cover-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  color: #94a3b8;
  padding: 20px;
  text-align: center;
}

.book-cover-placeholder span {
  margin-top: 12px;
  font-size: 13px;
  line-height: 1.4;
  color: #cbd5e1;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

/* 状态徽章 */
.status-badge {
  position: absolute;
  top: 10px;
  left: 10px;
  padding: 3px 10px;
  border-radius: 20px;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.5px;
  backdrop-filter: blur(6px);
}
.status-badge.done {
  background: rgba(16, 185, 129, 0.85);
  color: #fff;
}
.status-badge.reading {
  background: rgba(6, 182, 212, 0.85);
  color: #fff;
}
.status-badge.read {
  background: rgba(100, 116, 139, 0.75);
  color: #fff;
}
.status-badge.new {
  background: rgba(255, 255, 255, 0.75);
  color: var(--app-primary);
}

/* 进度条 */
.book-progress {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: 0 6px 6px;
}

/* 图书信息 */
.book-info {
  padding: 5px 8px 10px;
}

.book-title {
  margin: 0 0 5px 0;
  font-size: 14px;
  font-weight: 600;
  color: #0f172a;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.book-author {
  margin: 0 0 5px 0;
  font-size: 12px;
  color: #64748b;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.book-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
  min-height: 20px;
}

.type-tag {
  display: inline-block;
  flex-shrink: 0;
  padding: 2px 8px;
  border-radius: 6px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--app-primary);
  background: linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 12%, transparent), rgba(34, 211, 238, 0.12));
  border: 1px solid color-mix(in srgb, var(--app-primary) 25%, transparent);
}

.cat-tag {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  padding: 2px 8px;
  border-radius: 6px;
  font-size: 11px;
  font-weight: 500;
  color: #0d9488;
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.12), rgba(20, 184, 166, 0.14));
  border: 1px solid rgba(16, 185, 129, 0.25);
}

/* 书籍块中分类标签与阅读时间同行时，允许收缩并显示省略号 */
.book-meta .cat-tag {
  flex: 0 1 auto;
  min-width: 0;
}

.table-tags {
  display: flex;
  align-items: center;
  gap: 5px;
  flex-wrap: wrap;
}

.read-time {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
  font-size: 11px;
  color: #94a3b8;
  margin-left: auto;
}

/* 操作按钮 */
.book-actions {
  position: absolute;
  top: 10px;
  right: 10px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  opacity: 0;
  transform: translateX(10px);
  transition: opacity 0.25s ease, transform 0.25s ease;
}

.book-card:hover .book-actions {
  opacity: 1;
  transform: translateX(0);
}

.action-btn {
  box-shadow: 0 4px 12px rgba(31, 38, 135, 0.2);
}

/* 列表操作列：始终单行，不换行 */
.op-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  white-space: nowrap;
}
.op-actions .el-button + .el-button {
  margin-left: 0;
}
/* “更多”纯文字，无按钮样式 */
.op-more {
  font-size: 13px;
  font-weight: 500;
  color: var(--app-primary);
  cursor: pointer;
  user-select: none;
  white-space: nowrap;
}
.op-more:hover {
  text-decoration: underline;
}
/* 覆盖 Element Plus 相邻按钮默认 margin-left，保证竖排按钮垂直对齐 */
.book-actions .el-button + .el-button,
.book-actions .el-button + .el-dropdown {
  margin-left: 0;
}
.audio-btn {
  background: linear-gradient(135deg, #a855f7, #6366f1) !important;
  border: none !important;
  color: #fff !important;
}
.audio-btn:hover {
  box-shadow: 0 4px 14px rgba(168, 85, 247, 0.45);
}
.share-btn {
  background: linear-gradient(135deg, #0ea5e9, #6366f1) !important;
  border: none !important;
  color: #fff !important;
}
.share-btn:hover {
  box-shadow: 0 4px 14px rgba(14, 165, 233, 0.45);
}

/* ========== 列表视图 ========== */
.list-view {
  border-radius: 14px;
  overflow: hidden;
  box-shadow: 0 8px 32px rgba(31, 38, 135, 0.1);
  background: #fff;
  padding: 8px;
}

.table-book {
  display: flex;
  align-items: center;
  gap: 12px;
}

.table-cover-box {
  flex-shrink: 0;
  width: 44px;
  height: 60px;
  border-radius: 6px;
  overflow: hidden;
  box-shadow: 0 3px 10px rgba(31, 38, 135, 0.18);
}

.table-cover {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.table-cover-placeholder {
  width: 100%;
  height: 100%;
  background: linear-gradient(160deg, #334155, #1e293b);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #94a3b8;
}

.table-book-text {
  min-width: 0;
}

.table-title {
  font-weight: 600;
  color: #0f172a;
}

.table-author {
  font-size: 12px;
  color: #64748b;
  margin-top: 2px;
}

.table-progress {
  display: flex;
  align-items: center;
  gap: 8px;
}

.table-progress .el-progress {
  flex: 1;
}

.progress-text {
  font-size: 12px;
  color: var(--app-primary);
  font-weight: 600;
  min-width: 36px;
  text-align: right;
}

/* ========== 骨架屏 ========== */
.skeleton-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 20px;
}

.skeleton-card {
  border-radius: 14px;
  overflow: hidden;
  background: #fff;
  padding: 0 0 14px;
  box-shadow: 0 4px 16px rgba(31, 38, 135, 0.08);
}

.skeleton-cover {
  aspect-ratio: 3/4;
  background: linear-gradient(90deg, #eef2f7 25%, #e2e8f0 37%, #eef2f7 63%);
  background-size: 400% 100%;
  animation: shimmer 1.4s ease infinite;
}

.skeleton-line {
  height: 14px;
  margin: 12px 14px 0;
  border-radius: 6px;
  background: linear-gradient(90deg, #eef2f7 25%, #e2e8f0 37%, #eef2f7 63%);
  background-size: 400% 100%;
  animation: shimmer 1.4s ease infinite;
}

.skeleton-line.short {
  width: 60%;
}

@keyframes shimmer {
  0% { background-position: 100% 0; }
  100% { background-position: 0 0; }
}

/* ========== 空状态 ========== */
.empty {
  padding: 80px 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}

.empty-icon {
  width: 110px;
  height: 110px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--app-primary);
  background: radial-gradient(circle at 30% 30%, color-mix(in srgb, var(--app-primary) 20%, transparent), rgba(34, 211, 238, 0.1));
  box-shadow: 0 20px 60px color-mix(in srgb, var(--app-primary) 30%, transparent);
  margin-bottom: 24px;
  animation: float 4s ease-in-out infinite;
}

.empty h3 {
  margin: 0 0 8px;
  font-size: 20px;
  color: #0f172a;
}

.empty p {
  margin: 0 0 24px;
  color: #64748b;
}

/* ========== 分页 ========== */
.pagination {
  margin-top: 24px;
  display: flex;
  justify-content: flex-end;
}

.pagination :deep(.el-pagination) {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  color: #475569;
  font-weight: 500;
}

/* 总数 / 每页条数 / 跳转 文字 */
.pagination :deep(.el-pagination__total),
.pagination :deep(.el-pagination__sizes),
.pagination :deep(.el-pagination__jump) {
  color: #64748b;
  font-size: 13px;
}

/* 每页条数下拉 + 跳转输入框：圆角 + 主题聚焦 */
.pagination :deep(.el-select .el-input__wrapper),
.pagination :deep(.el-pagination__jump .el-input__wrapper) {
  border-radius: 10px;
  background: #f8fafc;
  box-shadow: 0 0 0 1px #e2e8f0 inset;
  transition: box-shadow 0.2s ease;
}
.pagination :deep(.el-select .el-input__wrapper.is-focus),
.pagination :deep(.el-pagination__jump .el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1.5px var(--app-primary),
    0 4px 14px color-mix(in srgb, var(--app-primary) 18%, transparent);
}

/* 翻页按钮 / 页码：浅底圆角 */
.pagination :deep(.el-pagination.is-background .btn-prev),
.pagination :deep(.el-pagination.is-background .btn-next),
.pagination :deep(.el-pagination.is-background .el-pager li) {
  background: #f1f5f9;
  color: #475569;
  border-radius: 9px;
  font-weight: 600;
  min-width: 32px;
  height: 32px;
  transition: background 0.2s ease, color 0.2s ease;
}
.pagination :deep(.el-pagination.is-background .btn-prev:hover),
.pagination :deep(.el-pagination.is-background .btn-next:hover),
.pagination :deep(.el-pagination.is-background .el-pager li:hover) {
  color: var(--app-primary);
  background: color-mix(in srgb, var(--app-primary) 12%, #f1f5f9);
}

/* 当前页：主题渐变 */
.pagination :deep(.el-pagination.is-background .el-pager li.is-active) {
  background: var(--app-gradient, linear-gradient(135deg, #6366f1, #8b5cf6));
  color: #fff;
  border: none;
  box-shadow: 0 6px 16px color-mix(in srgb, var(--app-primary) 30%, transparent);
}
.pagination :deep(.el-pagination.is-background .el-pager li.is-active:hover) {
  color: #fff;
}

/* 适配列表/表格的深色标题 */
:deep(.el-table) {
  --el-table-header-bg-color: #f8fafc;
  --el-table-header-text-color: #475569;
  border-radius: 12px;
}

/* ========== 响应式 ========== */
@media (max-width: 768px) {
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
  .stats-row {
    grid-template-columns: repeat(2, 1fr);
    gap: 12px;
  }
  .stat-card {
    padding: 14px 12px;
    animation: cardIn 0.4s cubic-bezier(0.22, 1, 0.36, 1) both;
  }
  .stat-card:nth-child(2) {
    animation-delay: 60ms;
  }
  .stat-card:nth-child(3) {
    animation-delay: 120ms;
  }
  .stat-card:nth-child(4) {
    animation-delay: 180ms;
  }
  .stat-value {
    font-size: 22px;
  }
  .view-toolbar {
    flex-direction: column;
    align-items: stretch;
    gap: 10px;
    padding: 10px 12px;
  }
  .vt-cats {
    flex: none;
    margin: 0 -2px;
  }
  .cat-select {
    width: 100%;
  }
  .vt-divider {
    width: 100%;
    height: 1px;
  }
  .vt-controls {
    flex-wrap: wrap;
    justify-content: space-between;
    gap: 10px;
  }
  .vt-group {
    flex-wrap: wrap;
  }

  /* 分页压缩 */
  .pagination :deep(.el-pagination) {
    flex-wrap: wrap;
    justify-content: center;
    row-gap: 8px;
  }
  .pagination :deep(.el-pagination .el-pagination__total) {
    margin-right: 8px;
  }
  .pagination :deep(.el-pagination .el-pager li) {
    min-width: 28px;
    font-size: 13px;
  }
  .pagination :deep(.el-pagination button) {
    min-width: 28px;
  }
}

/* 超小屏：统计卡单列，网格封面 2 列 */
@media (max-width: 400px) {
  .stats-row {
    grid-template-columns: repeat(2, 1fr);
    gap: 10px;
  }
  .stat-card {
    padding: 12px 10px;
  }
}
</style>

<style>
/* 分类下拉面板（teleport 到 body，需全局样式） */
.cat-popper.el-select__popper {
  border-radius: 14px;
  border: 1px solid color-mix(in srgb, var(--app-primary) 14%, transparent);
  background: rgba(255, 255, 255, 0.94);
  backdrop-filter: blur(16px) saturate(1.3);
  -webkit-backdrop-filter: blur(16px) saturate(1.3);
  box-shadow: 0 16px 40px rgba(30, 41, 59, 0.16);
  overflow: hidden;
  max-width: min(92vw, 460px);
}
.cat-popper .el-select-dropdown__wrap {
  padding: 6px;
  max-height: 340px;
}
.cat-popper .el-select-dropdown__item {
  border-radius: 10px;
  margin: 4px 0;
  padding: 9px 12px;
  height: auto;
  line-height: 20px;
  font-weight: 500;
  color: #334155;
  transition: background 0.2s ease, color 0.2s ease;
}
.cat-popper .el-select-dropdown__item:hover {
  background: linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 10%, transparent), rgba(34, 211, 238, 0.1));
  color: var(--app-primary-dark);
}
.cat-popper .el-select-dropdown__item.is-selected,
.cat-popper .el-select-dropdown__item.is-selected:hover {
  background: var(--app-gradient);
  color: #fff;
  font-weight: 600;
}
.cat-popper .el-select-dropdown__item.is-selected .cat-opt-count {
  color: #fff;
  background: rgba(255, 255, 255, 0.2);
  border-color: transparent;
}
.cat-popper .el-select-dropdown__empty {
  color: #94a3b8;
  padding: 12px;
}

/* ===== 分享弹窗（teleport 到 body，需全局样式） ===== */
.share-desc {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 12px 14px;
  margin-bottom: 16px;
  border-radius: 12px;
  font-size: 13px;
  line-height: 20px;
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
</style>
