<template>
  <div class="collections-page">
    <!-- 背景光晕 -->
    <div class="bg-glow bg-glow-1"></div>
    <div class="bg-glow bg-glow-2"></div>

    <!-- ========== 收藏夹列表视图 ========== -->
    <template v-if="view === 'list'">
      <div class="page-head">
        <div class="head-title">
          <span class="head-title-icon">
            <el-icon :size="22"><Star /></el-icon>
          </span>
          <div class="head-title-text">
            <h2>我的收藏夹</h2>
            <p>把喜欢的图书整理成个人书单</p>
          </div>
        </div>
        <el-button type="primary" round class="add-btn" @click="openCreate">
          <el-icon><Plus /></el-icon> 新建收藏夹
        </el-button>
      </div>

      <div v-loading="loading" class="col-grid">
        <div
          v-for="(col, index) in collections"
          :key="col.id"
          class="col-card cardIn"
          :style="{ '--i': index }"
          @click="openDetail(col)"
        >
          <div class="col-cover">
            <template v-if="col.covers && col.covers.length">
              <img
                v-for="cid in col.covers.slice(0, 4)"
                :key="cid"
                :src="bookApi.coverUrl(cid)"
                class="cover-item"
                alt=""
                loading="lazy"
                @error="onCoverError"
                @click.stop="goToDetail(cid)"
              />
            </template>
            <div v-else class="cover-placeholder">
              <el-icon :size="34"><FolderOpened /></el-icon>
            </div>
            <div class="cover-shine"></div>
            <span class="col-count"><el-icon><Document /></el-icon> {{ col.book_count }}</span>
          </div>
          <div class="col-info">
            <h4 class="col-name" :title="col.name">{{ col.name }}</h4>
            <p class="col-desc" v-if="col.description">{{ col.description }}</p>
            <p class="col-date">创建于 {{ formatDate(col.created_at) }}</p>
          </div>
          <div class="col-actions" @click.stop>
            <el-tooltip content="打开" placement="top">
              <el-button circle text size="small" class="act-btn" @click="openDetail(col)">
                <el-icon><View /></el-icon>
              </el-button>
            </el-tooltip>
            <el-tooltip content="编辑" placement="top">
              <el-button circle text size="small" class="act-btn" @click="openEdit(col)">
                <el-icon><EditPen /></el-icon>
              </el-button>
            </el-tooltip>
            <el-tooltip content="删除" placement="top">
              <el-button circle text size="small" type="danger" class="act-btn" @click="removeCollection(col)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </el-tooltip>
          </div>
        </div>
      </div>

      <div v-if="!loading && collections.length === 0" class="empty">
        <div class="empty-icon">
          <el-icon :size="56"><Star /></el-icon>
        </div>
        <h3>还没有收藏夹</h3>
        <p>新建收藏夹，把喜欢的图书整理成书单</p>
        <el-button type="primary" size="large" round @click="openCreate">新建收藏夹</el-button>
      </div>
    </template>

    <!-- ========== 收藏夹详情视图 ========== -->
    <template v-else>
      <div class="detail-head">
        <el-button text class="back-btn" @click="backToList">
          <el-icon><ArrowLeft /></el-icon> 返回
        </el-button>
        <div class="detail-title-wrap">
          <div class="detail-icon">
            <el-icon :size="20"><Folder /></el-icon>
          </div>
          <div class="detail-text">
            <h2>{{ currentCollection.name }}</h2>
            <p v-if="currentCollection.description">{{ currentCollection.description }}</p>
            <span class="detail-meta">{{ total }} 本图书</span>
          </div>
        </div>
        <div class="detail-actions">
          <ViewModeSwitch v-model="viewMode" />
          <el-button type="primary" round @click="openAddBooks">
            <el-icon><Plus /></el-icon> 添加图书
          </el-button>
          <el-button round @click="openEdit(currentCollection)">
            <el-icon><EditPen /></el-icon> 编辑
          </el-button>
        </div>
      </div>

      <!-- 网格视图 -->
      <div v-loading="detailLoading" v-if="viewMode === 'grid'" class="book-grid">
        <div
          v-for="(book, index) in books"
          :key="book.id"
          class="book-grid-cell"
        >
        <BookCard :book="book" :index="index" @click="goToDetail(book.id)">
          <template #actions>
            <el-tooltip content="移出收藏夹" placement="top">
              <el-button
                size="small"
                circle
                class="action-btn remove-btn"
                @click="removeBook(book)"
              >
                <el-icon><Close /></el-icon>
              </el-button>
            </el-tooltip>
            <el-tooltip content="阅读" placement="top">
              <el-button
                v-if="isReadable(book)"
                type="primary"
                size="small"
                circle
                class="action-btn"
                @click="goRead(book.id)"
              >
                <el-icon><Reading /></el-icon>
              </el-button>
            </el-tooltip>
          </template>
        </BookCard>
        </div>
      </div>

      <!-- 列表视图 -->
      <div v-else v-loading="detailLoading" class="book-list">
        <div
          v-for="book in books"
          :key="book.id"
          class="book-list-row"
          @click="goToDetail(book.id)"
        >
          <img v-if="book.cover_path" :src="bookApi.coverUrl(book.id)" class="bl-cover" alt="" loading="lazy" @error="onCoverError" />
          <div v-else class="bl-cover bl-cover-ph"><el-icon><Reading /></el-icon></div>
          <div class="bl-info">
            <div class="bl-title" :title="book.title">{{ book.title }}</div>
            <div class="bl-author">{{ formatAuthors(book.authors) }}</div>
          </div>
          <div class="bl-actions" @click.stop>
            <el-tooltip content="移出收藏夹" placement="top">
              <el-button size="small" circle class="action-btn remove-btn" @click="removeBook(book)">
                <el-icon><Close /></el-icon>
              </el-button>
            </el-tooltip>
            <el-tooltip content="阅读" placement="top">
              <el-button
                v-if="isReadable(book)"
                type="primary"
                size="small"
                circle
                class="action-btn"
                @click="goRead(book.id)"
              >
                <el-icon><Reading /></el-icon>
              </el-button>
            </el-tooltip>
          </div>
        </div>
      </div>

      <div v-if="!detailLoading && books.length === 0" class="empty">
        <div class="empty-icon">
          <el-icon :size="56"><FolderOpened /></el-icon>
        </div>
        <h3>收藏夹还是空的</h3>
        <p>点击「添加图书」把喜欢的书放进来</p>
        <el-button type="primary" size="large" round @click="openAddBooks">添加图书</el-button>
      </div>

      <div v-if="total > pageSize" class="pagination">
        <PaginationBar
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="pageSizesOptions"
          @current-change="loadDetailBooks"
          @size-change="onSizeChange"
        />
      </div>
    </template>

    <!-- 新建收藏夹对话框 -->
    <BaseDialog v-model="createVisible" title="新建收藏夹" icon="FolderAdd" width="480px">
      <el-form label-position="top">
        <el-form-item label="收藏夹名称">
          <el-input v-model="createForm.name" placeholder="如：想读清单、科幻精选" maxlength="60" show-word-limit />
        </el-form-item>
        <el-form-item label="描述（可选）">
          <el-input v-model="createForm.description" type="textarea" :rows="2" maxlength="120" show-word-limit />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitCreate">创建</el-button>
      </template>
    </BaseDialog>

    <!-- 编辑收藏夹对话框 -->
    <BaseDialog v-model="editVisible" title="编辑收藏夹" icon="Edit" width="480px">
      <el-form label-position="top">
        <el-form-item label="收藏夹名称">
          <el-input v-model="editForm.name" maxlength="60" show-word-limit />
        </el-form-item>
        <el-form-item label="描述（可选）">
          <el-input v-model="editForm.description" type="textarea" :rows="2" maxlength="120" show-word-limit />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitEdit">保存</el-button>
      </template>
    </BaseDialog>

    <!-- 添加图书对话框 -->
    <BaseDialog v-model="addVisible" title="添加图书到收藏夹" icon="Plus" width="520px">
      <div class="add-search">
        <el-input
          v-model="addQuery"
          placeholder="搜索书名或作者…"
          clearable
          size="large"
          class="search-input"
          @input="onAddQueryInput"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <span class="add-count">已选 {{ addBookIds.length }}</span>
      </div>
      <div v-if="searching" class="add-empty">
        <el-icon class="is-loading" :size="26"><Loading /></el-icon>
        <span>搜索中…</span>
      </div>
      <div v-else-if="searchResults.length === 0" class="add-empty">
        <el-icon :size="28"><Document /></el-icon>
        <span>{{ addQuery ? '没有匹配的图书' : '输入关键字搜索图书' }}</span>
      </div>
      <div v-else class="add-list">
        <div
          v-for="b in searchResults"
          :key="b.id"
          class="add-item"
          :class="{ selected: addBookIds.includes(b.id) }"
          @click="toggleAddBook(b.id)"
        >
          <div class="add-cover-wrap">
            <img :src="bookApi.coverUrl(b.id)" class="add-cover" alt="" loading="lazy" @error="onCoverError" />
            <span v-if="addBookIds.includes(b.id)" class="add-badge"><el-icon><Check /></el-icon></span>
          </div>
          <div class="add-info">
            <div class="add-title" :title="b.title">{{ b.title }}</div>
            <div class="add-sub">{{ formatAuthors(b.authors) }}</div>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="addVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" class="gradient-btn" @click="submitAddBooks">
          添加（{{ addBookIds.length }}）
        </el-button>
      </template>
    </BaseDialog>
  </div>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { bookApi } from '@/api/book'
import { useViewPagination } from '@/composables/useViewPagination'
import BaseDialog from '@/components/BaseDialog.vue'
import BookCard from '@/components/BookCard.vue'
import ViewModeSwitch from '@/components/ViewModeSwitch.vue'
import PaginationBar from '@/components/PaginationBar.vue'

const router = useRouter()

// ===== 列表视图 =====
const view = ref('list')
const loading = ref(false)
const collections = ref([])

const loadCollections = async () => {
  loading.value = true
  try {
    const { data } = await bookApi.listCollections()
    collections.value = data.data || []
  } catch (e) {
    // 列表加载失败不弹错误提醒（空数据属正常情况），静默降级为空列表
    console.warn('加载收藏夹失败', e)
  } finally {
    loading.value = false
  }
}

// ===== 详情视图 =====
const currentCollection = ref(null)
const books = ref([])
const total = ref(0)
const detailLoading = ref(false)

// 图书区：书架风格分页（两行自动 + 倍数选项 + 按视图持久化 + 缩放重算）
const { viewMode, page, pageSize, pageSizesOptions, onSizeChange } =
  useViewPagination('collection_books', () => loadDetailBooks())

const openDetail = (col) => {
  currentCollection.value = col
  page.value = 1
  view.value = 'detail'
  loadDetailBooks()
}

const backToList = () => {
  view.value = 'list'
  loadCollections()
}

const loadDetailBooks = async () => {
  detailLoading.value = true
  try {
    const { data } = await bookApi.getCollection(currentCollection.value.id, {
      page: page.value,
      page_size: pageSize.value
    })
    books.value = data.data || []
    total.value = data.total || 0
  } catch (e) {
    // 列表加载失败不弹错误提醒，静默降级为空列表
    console.warn('加载收藏夹内图书失败', e)
  } finally {
    detailLoading.value = false
  }
}

// ===== 新建 =====
const createVisible = ref(false)
const createForm = ref({ name: '', description: '' })
const saving = ref(false)

const openCreate = () => {
  createForm.value = { name: '', description: '' }
  createVisible.value = true
}

const submitCreate = async () => {
  const name = createForm.value.name.trim()
  if (!name) return ElMessage.warning('请输入收藏夹名称')
  saving.value = true
  try {
    await bookApi.createCollection({
      name,
      description: createForm.value.description.trim()
    })
    ElMessage.success('收藏夹创建成功')
    createVisible.value = false
    loadCollections()
  } catch (e) {
    ElMessage.error('创建失败')
  } finally {
    saving.value = false
  }
}

// ===== 编辑 =====
const editVisible = ref(false)
const editForm = ref({ name: '', description: '' })
const editingId = ref(null)

const openEdit = (col) => {
  editingId.value = col.id
  editForm.value = { name: col.name, description: col.description || '' }
  editVisible.value = true
}

const submitEdit = async () => {
  const name = editForm.value.name.trim()
  if (!name) return ElMessage.warning('请输入收藏夹名称')
  saving.value = true
  try {
    await bookApi.updateCollection(editingId.value, { name, description: editForm.value.description.trim() })
    ElMessage.success('保存成功')
    editVisible.value = false
    if (view.value === 'detail' && currentCollection.value.id === editingId.value) {
      currentCollection.value = { ...currentCollection.value, name, description: editForm.value.description.trim() }
    }
    loadCollections()
  } catch (e) {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

// ===== 删除收藏夹 =====
const removeCollection = async (col) => {
  try {
    await ElMessageBox.confirm(
      `确定删除收藏夹「${col.name}」吗？收藏夹内的图书不会被删除。`,
      '删除收藏夹',
      { type: 'warning', confirmButtonText: '确定', cancelButtonText: '取消' }
    )
    await bookApi.deleteCollection(col.id)
    ElMessage.success('收藏夹已删除')
    if (view.value === 'detail' && currentCollection.value.id === col.id) {
      view.value = 'list'
    }
    loadCollections()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error('删除失败')
  }
}

// ===== 添加图书 =====
const addVisible = ref(false)
const addBookIds = ref([])
const addQuery = ref('')
const searchResults = ref([])
const searching = ref(false)
let searchTimer = null

const openAddBooks = async () => {
  addBookIds.value = []
  addQuery.value = ''
  searchResults.value = []
  addVisible.value = true
  await searchBooks('')
}

const searchBooks = async (keyword) => {
  searching.value = true
  try {
    const { data } = await bookApi.listBooks({
      q: keyword,
      page: 1,
      page_size: 50,
      sort: 'created_at'
    })
    const all = data.data || []
    const exist = new Set(books.value.map((b) => b.id))
    searchResults.value = all.filter((b) => !exist.has(b.id))
  } catch (e) {
    searchResults.value = []
  } finally {
    searching.value = false
  }
}

const onAddQueryInput = (val) => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => searchBooks(val), 300)
}

const toggleAddBook = (id) => {
  const arr = addBookIds.value
  const idx = arr.indexOf(id)
  if (idx >= 0) arr.splice(idx, 1)
  else arr.push(id)
}

const submitAddBooks = async () => {
  if (addBookIds.value.length === 0) return ElMessage.warning('请选择要添加的图书')
  saving.value = true
  try {
    await bookApi.addBooksToCollection(currentCollection.value.id, addBookIds.value)
    ElMessage.success(`已添加 ${addBookIds.value.length} 本图书`)
    addVisible.value = false
    page.value = 1
    loadDetailBooks()
  } catch (e) {
    ElMessage.error('添加失败')
  } finally {
    saving.value = false
  }
}

// ===== 移除图书 =====
const removeBook = async (book) => {
  try {
    await ElMessageBox.confirm(`将「${book.title}」移出收藏夹？`, '移出收藏', { type: 'warning', confirmButtonText: '确定', cancelButtonText: '取消' })
    await bookApi.removeBookFromCollection(currentCollection.value.id, book.id)
    ElMessage.success('已移出收藏夹')
    loadDetailBooks()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error('移除失败')
  }
}

// ===== 工具函数 =====
const bookStatus = (book) => {
  if (book.progress >= 100) return { text: '已读完', cls: 'done' }
  if (book.progress > 0) return { text: '在读', cls: 'reading' }
  if (book.last_read) return { text: '读过', cls: 'read' }
  return { text: '未读', cls: 'new' }
}

const isReadable = (book) => ['epub', 'kepub', 'pdf', 'azw3', 'mobi'].includes(book.file_type)

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
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

const onCoverError = (e) => {
  e.target.style.visibility = 'hidden'
}

const goToDetail = (id) => router.push(`/book/${id}`)
const goRead = (id) => router.push(`/reader/${id}`)

watch(view, (v) => {
  if (v === 'list') window.scrollTo({ top: 0 })
})

onMounted(loadCollections)
</script>

<style scoped>
.collections-page {
  position: relative;
  max-width: 1400px;
  margin: 0 auto;
  min-height: calc(100vh - 120px);
}

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
  background: radial-gradient(circle, rgba(245, 158, 11, 0.2), transparent 70%);
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
.col-grid,
.detail-head,
.book-grid,
.pagination,
.empty {
  position: relative;
  z-index: 1;
}

/* ===== 页头 ===== */
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
  background: linear-gradient(135deg, #f59e0b, #f97316);
  box-shadow: 0 10px 24px rgba(245, 158, 11, 0.35);
}
.head-title-text h2 {
  margin: 0;
  font-size: 22px;
  font-weight: 700;
  background: linear-gradient(120deg, #d97706, #f59e0b);
  -webkit-background-clip: text;
  background-clip: text;
  -webkit-text-fill-color: transparent;
}
.head-title-text p {
  margin: 4px 0 0;
  font-size: 13px;
  color: #64748b;
}
.add-btn {
  background: linear-gradient(135deg, #f59e0b, #f97316);
  border: none;
  box-shadow: 0 8px 20px rgba(245, 158, 11, 0.35);
  padding: 20px 22px;
  font-size: 14px;
  transition: all 0.25s ease;
}
.add-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 12px 28px rgba(245, 158, 11, 0.45);
}

/* ===== 收藏夹卡片 ===== */
.col-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(210px, 1fr));
  gap: 20px;
  min-height: 120px;
}
.col-card {
  position: relative;
  border-radius: 16px;
  overflow: hidden;
  cursor: pointer;
  background: #fff;
  box-shadow: 0 4px 16px rgba(31, 38, 135, 0.08);
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1), box-shadow 0.3s ease;
}
.col-card::before {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: 16px;
  padding: 1.5px;
  background: linear-gradient(135deg, rgba(245, 158, 11, 0.5), rgba(249, 115, 22, 0.5), rgba(251, 191, 36, 0.5));
  -webkit-mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
  -webkit-mask-composite: xor;
  mask-composite: exclude;
  opacity: 0;
  transition: opacity 0.3s ease;
  pointer-events: none;
}
.col-card:hover {
  transform: translateY(-6px) scale(1.02);
  box-shadow: 0 20px 48px rgba(245, 158, 11, 0.28);
}
.col-card:hover::before {
  opacity: 1;
}
.col-cover {
  position: relative;
  height: 128px;
  overflow: hidden;
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  background: linear-gradient(160deg, #334155, #1e293b);
}
.cover-item {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 0.4s ease;
  min-width: 0;
}
.col-card:hover .cover-item {
  transform: scale(1.06);
}
.cover-item:nth-child(1):only-child { grid-column: 1 / -1; }
.cover-item:nth-child(1):nth-last-child(1) { grid-column: 1 / -1; }
.cover-item:nth-child(1):nth-last-child(2),
.cover-item:nth-child(2):nth-last-child(1) { grid-column: span 2; }
.cover-item:nth-child(1):nth-last-child(3),
.cover-item:nth-child(2):nth-last-child(2) { grid-column: span 2; }
.cover-item:nth-child(3):nth-last-child(1) { grid-column: 1 / -1; }
.cover-item:nth-child(1):nth-last-child(4),
.cover-item:nth-child(2):nth-last-child(3),
.cover-item:nth-child(3):nth-last-child(2),
.cover-item:nth-child(4):nth-last-child(1) { grid-column: span 1; }
.cover-placeholder {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: rgba(255, 255, 255, 0.55);
}
.cover-shine {
  position: absolute;
  inset: 0;
  background: linear-gradient(105deg, transparent 40%, rgba(255, 255, 255, 0.18) 50%, transparent 60%);
  transform: translateX(-120%);
  transition: transform 0.7s ease;
  pointer-events: none;
}
.col-card:hover .cover-shine {
  transform: translateX(120%);
}
.col-count {
  position: absolute;
  top: 10px;
  right: 10px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 10px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 600;
  color: #fff;
  background: rgba(15, 23, 42, 0.55);
  backdrop-filter: blur(6px);
}
.col-info {
  padding: 14px 16px 16px;
}
.col-name {
  margin: 0 0 4px;
  font-size: 16px;
  font-weight: 700;
  color: #0f172a;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.col-desc {
  margin: 0 0 4px;
  font-size: 13px;
  color: #64748b;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.col-date {
  margin: 6px 0 0;
  font-size: 12px;
  color: #94a3b8;
}
.col-actions {
  position: absolute;
  top: 10px;
  left: 10px;
  display: flex;
  gap: 4px;
  opacity: 0;
  transform: translateX(-10px);
  transition: opacity 0.25s ease, transform 0.25s ease;
}
.col-card:hover .col-actions {
  opacity: 1;
  transform: translateX(0);
}
.act-btn {
  background: rgba(255, 255, 255, 0.9);
  box-shadow: 0 4px 12px rgba(31, 38, 135, 0.2);
}
.act-btn:hover {
  color: #f59e0b;
}

/* ===== 详情页头 ===== */
.detail-head {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 24px;
  padding: 18px 24px;
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.78);
  backdrop-filter: blur(14px);
  border: 1px solid rgba(255, 255, 255, 0.7);
  box-shadow: 0 10px 36px rgba(31, 38, 135, 0.1);
}
.back-btn {
  flex-shrink: 0;
  color: #64748b;
  font-weight: 600;
}
.back-btn:hover {
  color: #f59e0b;
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
  width: 46px;
  height: 46px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  background: linear-gradient(135deg, #f59e0b, #f97316);
  box-shadow: 0 10px 24px rgba(245, 158, 11, 0.35);
}
.detail-text {
  min-width: 0;
}
.detail-text h2 {
  margin: 0 0 2px;
  font-size: 20px;
  font-weight: 700;
  color: #0f172a;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.detail-text p {
  margin: 0;
  font-size: 13px;
  color: #64748b;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.detail-meta {
  display: inline-block;
  margin-top: 4px;
  font-size: 12px;
  color: #b45309;
  background: rgba(245, 158, 11, 0.12);
  border-radius: 20px;
  padding: 2px 10px;
}
.detail-actions {
  flex-shrink: 0;
  display: flex;
  gap: 10px;
}
.detail-actions .el-button--primary {
  background: linear-gradient(135deg, #f59e0b, #f97316);
  border: none;
  box-shadow: 0 8px 20px rgba(245, 158, 11, 0.35);
}

/* ===== 图书网格（CSS Grid 显式控制列数，移动端固定 2 列） ===== */
.book-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
  min-height: 120px;
}
@media (min-width: 576px) {
  .book-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 16px;
  }
}
@media (min-width: 992px) {
  .book-grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 18px;
  }
}
@media (min-width: 1200px) {
  .book-grid {
    grid-template-columns: repeat(6, minmax(0, 1fr));
    gap: 20px;
  }
}
@media (min-width: 1920px) {
  .book-grid {
    grid-template-columns: repeat(8, minmax(0, 1fr));
  }
}
.book-grid-cell {
  min-width: 0;
}

/* ===== 图书列表视图 ===== */
.book-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 120px;
}
.book-list-row {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 12px 16px;
  border-radius: 14px;
  background: #fff;
  border: 1px solid #eef2f7;
  box-shadow: 0 4px 16px rgba(31, 38, 135, 0.06);
  cursor: pointer;
  transition: transform 0.25s ease, box-shadow 0.25s ease, border-color 0.25s ease;
}
.book-list-row:hover {
  transform: translateY(-3px);
  box-shadow: 0 14px 34px rgba(99, 102, 241, 0.18);
  border-color: color-mix(in srgb, var(--app-primary) 40%, transparent);
}
.bl-cover {
  flex-shrink: 0;
  width: 44px;
  height: 60px;
  object-fit: cover;
  border-radius: 6px;
  box-shadow: 0 4px 12px rgba(31, 38, 135, 0.18);
}
.bl-cover-ph {
  display: flex;
  align-items: center;
  justify-content: center;
  color: #94a3b8;
  background: color-mix(in srgb, var(--app-primary) 8%, transparent);
}
.bl-info {
  flex: 1;
  min-width: 0;
}
.bl-title {
  font-size: 15px;
  font-weight: 700;
  color: #0f172a;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.bl-author {
  margin-top: 4px;
  font-size: 13px;
  color: #64748b;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.bl-actions {
  flex-shrink: 0;
  display: flex;
  gap: 8px;
}
.bl-actions .action-btn {
  box-shadow: 0 4px 12px rgba(31, 38, 135, 0.2);
}
.book-card {
  position: relative;
  border-radius: 14px;
  overflow: hidden;
  cursor: pointer;
  background: #fff;
  box-shadow: 0 4px 16px rgba(31, 38, 135, 0.08);
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1), box-shadow 0.3s ease;
}
.book-card::before {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: 14px;
  padding: 1.5px;
  background: linear-gradient(135deg, rgba(245, 158, 11, 0.5), rgba(249, 115, 22, 0.5), rgba(251, 191, 36, 0.5));
  -webkit-mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
  -webkit-mask-composite: xor;
  mask-composite: exclude;
  opacity: 0;
  transition: opacity 0.3s ease;
  pointer-events: none;
  z-index: 1;
}
.book-card:hover {
  transform: translateY(-8px) scale(1.02);
  box-shadow: 0 20px 48px rgba(245, 158, 11, 0.28);
}
.book-card:hover::before {
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
.book-cover-wrapper::after {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(105deg, transparent 40%, rgba(255, 255, 255, 0.14) 50%, transparent 60%);
  transform: translateX(-120%);
  transition: transform 0.7s ease;
  pointer-events: none;
  z-index: 1;
}
.book-card:hover .book-cover-wrapper::after {
  transform: translateX(120%);
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
.status-badge {
  position: absolute;
  top: 10px;
  left: 10px;
  padding: 3px 10px;
  border-radius: 20px;
  font-size: 11px;
  font-weight: 600;
  backdrop-filter: blur(6px);
}
.status-badge.done { background: rgba(16, 185, 129, 0.85); color: #fff; }
.status-badge.reading { background: rgba(6, 182, 212, 0.85); color: #fff; }
.status-badge.read { background: rgba(100, 116, 139, 0.75); color: #fff; }
.status-badge.new { background: rgba(255, 255, 255, 0.75); color: #f59e0b; }
.book-progress {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: 0 6px 6px;
}
.book-actions {
  position: absolute;
  top: 10px;
  right: 10px;
  z-index: 2;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  opacity: 0;
  transform: translateX(10px);
  transition: opacity 0.25s ease, transform 0.25s ease;
}
.book-card:hover .book-actions {
  opacity: 1;
  transform: translateX(0);
}
/* 覆盖 Element 默认 .el-button + .el-button 的 margin-left，保证纵向按钮居中对齐 */
.book-actions .el-button + .el-button {
  margin-left: 0;
}
/* 与首页书架图书列表的操作图标统一：默认 small circle 尺寸 + 阴影 */
.action-btn {
  box-shadow: 0 4px 12px rgba(31, 38, 135, 0.2);
}
/* 移出收藏夹：白底红叉，与首页"更多"按钮同尺寸同质感 */
.remove-btn {
  background: rgba(255, 255, 255, 0.92);
  color: #ef4444;
  border: 1px solid rgba(239, 68, 68, 0.25);
}
.remove-btn:hover {
  color: #fff;
  background: #ef4444;
  border-color: transparent;
  transform: translateY(-1px);
  box-shadow: 0 6px 16px rgba(239, 68, 68, 0.4);
}
/* 阅读：primary 按钮沿用收藏夹橙色主题 */
.book-actions .el-button--primary {
  background: linear-gradient(135deg, #f59e0b, #f97316);
  border-color: transparent;
  box-shadow: 0 6px 16px rgba(245, 158, 11, 0.4);
}
.book-actions .el-button--primary:hover {
  background: linear-gradient(135deg, #f97316, #ea580c);
  border-color: transparent;
  transform: translateY(-1px);
  box-shadow: 0 8px 20px rgba(245, 158, 11, 0.55);
}
.book-info {
  padding: 12px 14px 14px;
}
.book-title {
  margin: 0 0 6px;
  font-size: 14px;
  font-weight: 600;
  color: #0f172a;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.book-author {
  margin: 0;
  font-size: 12px;
  color: #64748b;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* ===== 分页 ===== */
.pagination {
  margin-top: 24px;
  display: flex;
  justify-content: flex-end;
}

/* ===== 空状态 ===== */
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
  color: #f59e0b;
  background: radial-gradient(circle at 30% 30%, rgba(245, 158, 11, 0.2), rgba(34, 211, 238, 0.1));
  box-shadow: 0 20px 60px rgba(245, 158, 11, 0.3);
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

/* 搜索选项作者 */
.opt-author {
  float: right;
  font-size: 12px;
  color: #94a3b8;
  margin-left: 12px;
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

/* ===== 响应式 ===== */
@media (max-width: 768px) {
  .page-head {
    flex-direction: column;
    align-items: flex-start;
    gap: 14px;
    padding: 16px 18px;
  }
  .add-btn {
    padding: 16px 18px;
    width: 100%;
  }
  .col-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 14px;
  }
  .col-cover {
    height: 100px;
  }
  .detail-head {
    flex-wrap: wrap;
    padding: 14px 16px;
  }
  .detail-title-wrap {
    order: 2;
    width: 100%;
  }
  .back-btn {
    order: 1;
  }
  .detail-actions {
    order: 3;
    width: 100%;
  }
  .detail-actions .el-button {
    flex: 1;
  }
  /* 移动端无 hover，操作按钮常显 */
  .col-actions,
  .book-actions {
    opacity: 1;
    transform: none;
  }
  .col-actions .act-btn {
    width: 30px;
    height: 30px;
  }
}

/* ========== 添加图书弹框：卡片列表 + 模糊搜索 ========== */
.add-search {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 14px;
}
.add-search .search-input {
  flex: 1;
  min-width: 0;
}
.add-search .search-input :deep(.el-input__wrapper) {
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.92);
  box-shadow: 0 0 0 1px color-mix(in srgb, #f59e0b 20%, transparent) inset;
  padding: 4px 16px;
  transition: box-shadow 0.25s ease;
}
.add-search .search-input :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1.5px #f59e0b, 0 6px 20px rgba(245, 158, 11, 0.22);
}
.add-search .search-input :deep(.el-input__prefix) {
  color: #94a3b8;
}
.add-count {
  flex-shrink: 0;
  font-size: 12px;
  font-weight: 600;
  color: #b45309;
  padding: 7px 12px;
  border-radius: 999px;
  background: linear-gradient(135deg, rgba(245, 158, 11, 0.12), rgba(249, 115, 22, 0.12));
  border: 1px solid color-mix(in srgb, #f59e0b 30%, transparent);
}
.add-list {
  max-height: 320px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.add-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border-radius: 12px;
  background: #fff;
  border: 1px solid #eef2f7;
  box-shadow: 0 2px 8px rgba(31, 38, 135, 0.05);
  cursor: pointer;
  transition: transform 0.2s ease, box-shadow 0.2s ease, border-color 0.2s ease, background 0.2s ease;
}
.add-item:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 22px rgba(31, 38, 135, 0.12);
  border-color: color-mix(in srgb, #f59e0b 35%, transparent);
}
.add-item.selected {
  background: color-mix(in srgb, #f59e0b 7%, transparent);
  border-color: color-mix(in srgb, #f59e0b 50%, transparent);
  box-shadow: 0 6px 18px rgba(245, 158, 11, 0.16);
}
.add-cover-wrap {
  position: relative;
  flex-shrink: 0;
  width: 44px;
  height: 58px;
  border-radius: 6px;
  background: color-mix(in srgb, #f59e0b 10%, transparent);
}
.add-cover {
  width: 44px;
  height: 58px;
  object-fit: cover;
  border-radius: 6px;
  box-shadow: 0 4px 12px rgba(31, 38, 135, 0.18);
}
.add-badge {
  position: absolute;
  top: -6px;
  right: -6px;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  background: linear-gradient(135deg, #f59e0b, #f97316);
  border: 2px solid #fff;
  box-shadow: 0 2px 8px rgba(245, 158, 11, 0.45);
}
.add-badge .el-icon {
  font-size: 12px;
}
.add-info {
  min-width: 0;
  flex: 1;
}
.add-title {
  font-size: 14px;
  font-weight: 600;
  color: #0f172a;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.add-sub {
  font-size: 12px;
  color: #64748b;
  margin-top: 2px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.add-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 28px 0;
  color: #94a3b8;
  font-size: 13px;
  border: 1px dashed color-mix(in srgb, #f59e0b 30%, transparent);
  border-radius: 12px;
}
.gradient-btn {
  background: linear-gradient(135deg, #f59e0b, #f97316) !important;
  border: none !important;
  box-shadow: 0 6px 18px rgba(245, 158, 11, 0.35);
}
</style>
