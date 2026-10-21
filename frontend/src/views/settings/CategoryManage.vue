<template>
  <div class="category-manage">
    <!-- 背景光晕 -->
    <div class="bg-glow bg-glow-1"></div>
    <div class="bg-glow bg-glow-2"></div>

    <!-- 页面头部 -->
    <div class="page-head">
      <div class="head-left">
        <span class="head-title-icon">
          <el-icon :size="22"><FolderOpened /></el-icon>
        </span>
        <div class="head-title-text">
          <h2>分类管理</h2>
          <div v-if="!loading && breadcrumb.length" class="crumb-bar">
            <el-icon class="crumb-root"><FolderOpened /></el-icon>
            <template v-for="(node, idx) in breadcrumb" :key="node.id">
              <span
                class="crumb-item"
                :class="{ active: idx === breadcrumb.length - 1 }"
                @click="goToCrumb(idx)"
              >{{ node.name }}</span>
              <el-icon v-if="idx < breadcrumb.length - 1" class="crumb-sep"><ArrowRight /></el-icon>
            </template>
          </div>
        </div>
      </div>
      <div class="head-actions">
        <el-button type="primary" size="large" round class="add-btn" @click="openCreate(isRoot ? null : currentNode)">
          <el-icon><Plus /></el-icon> {{ isRoot ? '新增分类' : '新建子分类' }}
        </el-button>
      </div>
    </div>

    <!-- 工具栏：搜索 + 视图切换 -->
    <div class="view-toolbar">
      <div class="vt-search">
        <el-input
          v-model="searchQuery"
          placeholder="搜索分类名称..."
          clearable
          size="large"
          class="search-input"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
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
      </div>
    </div>

    <!-- 加载中骨架屏 -->
    <div v-if="loading" class="loading">
      <div class="skeleton-grid">
        <div v-for="i in 6" :key="i" class="skeleton-card">
          <div class="skeleton-cover"></div>
          <div class="skeleton-line"></div>
          <div class="skeleton-line short"></div>
        </div>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-if="!loading && currentNode && filteredChildren.length === 0 && displayBooks.length === 0" class="empty">
      <div class="empty-icon">
        <el-icon :size="56"><FolderOpened /></el-icon>
      </div>
      <template v-if="searchQuery">
        <h3>没有匹配的结果</h3>
        <p>没有分类或图书名称包含「{{ searchQuery }}」</p>
        <el-button type="primary" size="large" round @click="searchQuery = ''">清除搜索</el-button>
      </template>
      <template v-else-if="isRoot">
        <h3>还没有分类</h3>
        <p>创建第一个分类，把喜欢的图书归类管理</p>
        <el-button type="primary" size="large" round @click="openCreate(currentNode)">
          <el-icon><Plus /></el-icon> 新建子分类
        </el-button>
      </template>
      <template v-else>
        <h3>这个分类还没有内容</h3>
        <p>「{{ currentNode.name }}」下暂无子分类或图书</p>
        <el-button round @click="goToCrumb(breadcrumb.length - 2)">返回上一级</el-button>
      </template>
    </div>

    <!-- 子分类区块（面包屑下钻） -->
    <div v-if="!loading && currentNode && filteredChildren.length > 0" class="section">
      <div class="section-head">
        <span class="section-bar"></span>
        <h3 class="section-title">子分类 · {{ currentNode.children.length }}</h3>
      </div>

      <!-- 网格视图 -->
      <div v-if="viewMode === 'grid'" class="cat-grid">
        <div
          v-for="(cat, index) in filteredChildren"
          :key="cat.id"
          class="cat-card cardIn"
          :style="{ '--i': index }"
          @click="openNode(cat)"
        >
          <div class="cat-cover">
            <template v-if="cat.books && cat.books.length">
              <img
                v-for="b in cat.books.slice(0, 4)"
                :key="b.id"
                :src="bookApi.coverUrl(b.id)"
                class="cover-item"
                alt=""
                loading="lazy"
                @error="onCoverError"
              />
            </template>
            <div v-else class="cover-placeholder">
              <el-icon :size="34"><FolderOpened /></el-icon>
            </div>
            <div class="cover-shine"></div>
            <span class="cat-count"><el-icon><Document /></el-icon> {{ cat.count }}</span>
          </div>
          <div class="cat-info">
            <h4 class="cat-name" :title="cat.name">{{ cat.name }}</h4>
            <p class="cat-meta">{{ cat.count }} 本</p>
          </div>
          <div v-if="cat.realCategory" class="cat-actions" @click.stop>
            <el-tooltip content="新建子分类" placement="top">
              <el-button circle text size="small" class="act-btn" @click="openCreate(cat)">
                <el-icon><FolderAdd /></el-icon>
              </el-button>
            </el-tooltip>
            <el-tooltip content="添加图书" placement="top">
              <el-button circle text size="small" class="act-btn" @click="openAddBooks(cat)">
                <el-icon><Plus /></el-icon>
              </el-button>
            </el-tooltip>
            <el-tooltip content="移除图书" placement="top">
              <el-button circle text size="small" class="act-btn" @click="openRemoveBooks(cat)">
                <el-icon><Minus /></el-icon>
              </el-button>
            </el-tooltip>
            <el-tooltip content="重命名" placement="top">
              <el-button circle text size="small" class="act-btn" @click="openRename(cat)">
                <el-icon><EditPen /></el-icon>
              </el-button>
            </el-tooltip>
            <el-tooltip content="删除" placement="top">
              <el-button circle text size="small" type="danger" class="act-btn" @click="openDelete(cat)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </el-tooltip>
          </div>
        </div>
      </div>

      <!-- 列表视图 -->
      <div v-else class="list-view">
        <el-table :data="filteredChildren" class="cat-table" @row-click="openNode">
          <el-table-column label="分类" min-width="240">
            <template #default="{ row }">
              <div class="table-cat">
                <span class="table-cat-icon" :style="{ background: iconGradient(row.name) }">
                  <el-icon><Folder /></el-icon>
                </span>
                <span class="table-cat-name">{{ row.name }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="图书数" width="140">
            <template #default="{ row }">
              <span class="count-badge">{{ row.count }} 本</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="320" fixed="right">
            <template #default="{ row }">
              <template v-if="row.realCategory">
                <el-button size="small" class="act-btn" @click.stop="openCreate(row)">子分类</el-button>
                <el-button size="small" class="act-btn" @click.stop="openAddBooks(row)">添加</el-button>
                <el-button size="small" class="act-btn" @click.stop="openRemoveBooks(row)">移除</el-button>
                <el-button size="small" class="act-btn" @click.stop="openRename(row)">重命名</el-button>
                <el-button size="small" type="danger" text @click.stop="openDelete(row)">删除</el-button>
              </template>
              <span v-else class="table-cat-sub">子分类 · 进入查看</span>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <!-- 图书区块 -->
    <div v-if="!loading && currentNode && displayBooks.length > 0" class="section">
      <div class="section-head">
        <span class="section-bar"></span>
        <h3 class="section-title">图书 · {{ bookTotal }}</h3>
      </div>

      <!-- 网格视图 -->
      <el-row v-loading="catBooksLoading" v-if="viewMode === 'grid'" :gutter="20" class="book-grid">
        <el-col
          v-for="(book, index) in displayBooks"
          :key="book.id"
          :xs="12" :sm="8" :md="6" :lg="4" :xl="3"
        >
          <BookCard
            :book="book"
            :index="index"
            :progress-color="'linear-gradient(90deg,#f59e0b,#f97316)'"
            @click="goToDetail(book.id)"
          >
            <template #actions>
              <el-button size="small" circle class="action-btn remove-btn" @click="removeBookFromCategory(book)">
                <el-icon><Close /></el-icon>
              </el-button>
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
            </template>
          </BookCard>
        </el-col>
      </el-row>

      <!-- 列表视图 -->
      <div v-else v-loading="catBooksLoading" class="book-list">
        <div
          v-for="book in displayBooks"
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
            <el-button
              v-if="isReadable(book)"
              type="primary"
              size="small"
              class="action-btn"
              @click="goRead(book.id)"
            >阅读</el-button>
            <el-button size="small" class="action-btn" @click="removeBookFromCategory(book)">移出</el-button>
          </div>
        </div>
      </div>
    </div>

    <!-- 图书分页 -->
    <div v-if="!loading && currentNode && bookTotal > bookPageSize" class="pagination">
      <PaginationBar
        v-model:current-page="bookPage"
        v-model:page-size="bookPageSize"
        :total="bookTotal"
        :page-sizes="pageSizesOptions"
        @current-change="onBookPageChange"
        @size-change="onSizeChange"
      />
    </div>

    <!-- 新增分类 -->
    <BaseDialog v-model="createVisible" :title="createForm.parent ? '新建子分类' : '新增分类'" icon="FolderAdd" width="520px">
      <el-form label-position="top">
        <el-form-item v-if="createForm.parent" label="父级分类">
          <div class="parent-chip">
            <el-icon><FolderOpened /></el-icon>
            <span>{{ createForm.parent }}</span>
          </div>
        </el-form-item>
        <el-form-item label="分类名称">
          <el-input v-model="createForm.name" :placeholder="createForm.parent ? '如：长篇、短篇小说' : '如：科幻、文学、技术'" maxlength="40" show-word-limit />
        </el-form-item>
      </el-form>

      <!-- 选择图书（可选）：与「添加书籍到分类」弹窗列表完全一致 -->
      <div class="create-book-section">
        <p class="add-books-target">选择图书（可选，可留空仅创建空分类）</p>
        <div class="add-search">
          <el-input
            v-model="createBookQuery"
            placeholder="搜索书名或作者…"
            clearable
            size="large"
            class="search-input"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
          <span class="add-count">已选 {{ createForm.bookIds.length }}</span>
        </div>
        <div v-if="filteredCreateBooks.length === 0" class="add-empty">
          <el-icon :size="28"><Document /></el-icon>
          <span>{{ createBookQuery ? '没有匹配的图书' : '暂无可添加的未分类图书' }}</span>
        </div>
        <div v-else class="add-list">
          <div
            v-for="b in filteredCreateBooks"
            :key="b.id"
            class="add-item"
            :class="{ selected: createForm.bookIds.includes(b.id) }"
            @click="toggleCreateBook(b.id)"
          >
            <div class="add-cover-wrap">
              <img :src="bookApi.coverUrl(b.id)" class="add-cover" alt="" loading="lazy" @error="onCoverError" />
              <span v-if="createForm.bookIds.includes(b.id)" class="add-badge"><el-icon><Check /></el-icon></span>
            </div>
            <div class="add-info">
              <div class="add-title" :title="b.title">{{ b.title }}</div>
              <div class="add-sub">{{ b.author || '未知作者' }}</div>
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitCreate">保存</el-button>
      </template>
    </BaseDialog>

    <!-- 重命名 -->
    <BaseDialog v-model="renameVisible" title="重命名分类" icon="EditPen" width="420px">
      <el-form label-position="top">
        <el-form-item label="分类名称">
          <el-input v-model="renameForm.newName" placeholder="请输入新名称" maxlength="40" show-word-limit />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="renameVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitRename">确定</el-button>
      </template>
    </BaseDialog>

    <!-- 删除 -->
    <BaseDialog v-model="deleteVisible" title="删除分类" icon="Delete" width="520px">
      <p class="delete-tip">
        分类「<b>{{ deleteForm.name }}</b>」下共有 <b>{{ deleteForm.count }}</b> 本图书，删除后无法恢复。
      </p>

      <div v-if="deleteForm.count > 0" class="delete-options">
        <div
          class="delete-option"
          :class="{ active: deleteForm.moveTo === '' }"
          @click="deleteForm.moveTo = ''"
        >
          <div class="do-radio">
            <el-icon v-if="deleteForm.moveTo === ''"><Check /></el-icon>
          </div>
          <div class="do-body">
            <div class="do-title">移除分类</div>
            <div class="do-desc">图书保留但变为未分类</div>
          </div>
        </div>

        <div
          v-if="moveTargets.length"
          class="delete-option"
          :class="{ active: deleteForm.moveTo !== '' }"
          @click="deleteForm.moveTo = moveTargets.includes(deleteForm.moveTo) ? deleteForm.moveTo : moveTargets[0]"
        >
          <div class="do-radio">
            <el-icon v-if="deleteForm.moveTo !== ''"><Check /></el-icon>
          </div>
          <div class="do-body">
            <div class="do-title">移动到其他分类</div>
            <div class="do-desc">将图书批量迁移到指定分类</div>
          </div>
        </div>
        <div v-else class="delete-option disabled">
          <div class="do-radio"></div>
          <div class="do-body">
            <div class="do-title">移动到其他分类</div>
            <div class="do-desc">暂无可移动的目标分类</div>
          </div>
        </div>

        <div v-if="deleteForm.moveTo !== '' && moveTargets.length" class="delete-target-wrap">
          <span class="dt-label">目标分类</span>
          <el-select v-model="deleteForm.moveTo" placeholder="请选择目标分类" class="dt-select">
            <el-option v-for="t in moveTargets" :key="t" :label="t" :value="t" />
          </el-select>
        </div>
      </div>

      <template #footer>
        <el-button @click="deleteVisible = false">取消</el-button>
        <el-button type="danger" :loading="saving" @click="submitDelete">删除分类</el-button>
      </template>
    </BaseDialog>

    <!-- 添加书籍 -->
    <BaseDialog v-model="addBooksVisible" title="添加书籍到分类" icon="Plus" width="520px">
      <p class="add-books-target">目标分类：<b>{{ addBooksForm.name }}</b></p>
      <div class="add-search">
        <el-input
          v-model="addBookQuery"
          placeholder="搜索书名或作者…"
          clearable
          size="large"
          class="search-input"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <span class="add-count">已选 {{ addBooksForm.bookIds.length }}</span>
      </div>
      <div v-if="filteredAddBooks.length === 0" class="add-empty">
        <el-icon :size="28"><Document /></el-icon>
        <span>{{ addBookQuery ? '没有匹配的图书' : '暂无可添加的未分类图书' }}</span>
      </div>
      <div v-else class="add-list">
        <div
          v-for="b in filteredAddBooks"
          :key="b.id"
          class="add-item"
          :class="{ selected: addBooksForm.bookIds.includes(b.id) }"
          @click="toggleBook(b.id)"
        >
          <div class="add-cover-wrap">
            <img :src="bookApi.coverUrl(b.id)" class="add-cover" alt="" loading="lazy" @error="onCoverError" />
            <span v-if="addBooksForm.bookIds.includes(b.id)" class="add-badge"><el-icon><Check /></el-icon></span>
          </div>
          <div class="add-info">
            <div class="add-title" :title="b.title">{{ b.title }}</div>
            <div class="add-sub">{{ b.author || '未知作者' }}</div>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="addBooksVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" class="gradient-btn" @click="submitAddBooks">添加</el-button>
      </template>
    </BaseDialog>

    <!-- 移除图书 -->
    <BaseDialog v-model="removeBooksVisible" title="从分类移除图书" icon="Minus" width="520px">
      <p class="add-books-target">目标分类：<b>{{ removeBooksForm.name }}</b></p>
      <div class="add-search">
        <el-input
          v-model="removeBookQuery"
          placeholder="搜索书名或作者…"
          clearable
          size="large"
          class="search-input"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <span class="add-count">已选 {{ removeBooksForm.bookIds.length }}</span>
      </div>
      <div v-if="filteredRemoveBooks.length === 0" class="add-empty">
        <el-icon :size="28"><Document /></el-icon>
        <span>{{ removeBookQuery ? '没有匹配的图书' : '该分类下暂无可移除的图书' }}</span>
      </div>
      <div v-else class="add-list">
        <div
          v-for="b in filteredRemoveBooks"
          :key="b.id"
          class="add-item"
          :class="{ selected: removeBooksForm.bookIds.includes(b.id) }"
          @click="toggleRemoveBook(b.id)"
        >
          <div class="add-cover-wrap">
            <img :src="bookApi.coverUrl(b.id)" class="add-cover" alt="" loading="lazy" @error="onCoverError" />
            <span v-if="removeBooksForm.bookIds.includes(b.id)" class="add-badge"><el-icon><Check /></el-icon></span>
          </div>
          <div class="add-info">
            <div class="add-title" :title="b.title">{{ b.title }}</div>
            <div class="add-sub">{{ b.author || '未知作者' }}</div>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="removeBooksVisible = false">取消</el-button>
        <el-button type="danger" :loading="saving" @click="submitRemoveBooks">移除</el-button>
      </template>
    </BaseDialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { bookApi } from '@/api/book'
import { useViewPagination } from '@/composables/useViewPagination'
import { useIsMobile } from '@/composables/useMobileView'
import BaseDialog from '@/components/BaseDialog.vue'
import BookCard from '@/components/BookCard.vue'
import PaginationBar from '@/components/PaginationBar.vue'

const loading = ref(false)
const saving = ref(false)
const rawCategories = ref([])
const allBooks = ref([])
const searchQuery = ref('')

// 图书区采用书架风格分页：两行自动条数 + 倍数选项 + 按视图持久化 + 缩放重算
const reloadBooks = () => loadNodeBooks(currentNode.value)
const { viewMode, page: bookPage, pageSize: bookPageSize, pageSizesOptions, onSizeChange } =
  useViewPagination('category_books', reloadBooks)
// 手机端隐藏视图切换器（列表视图已由 composable 强制为网格）
const { isMobile } = useIsMobile()

// ===== 分类树（真实后端：支持父子层级） =====
const rootNode = ref(null)
const path = ref([])

function buildTree(list) {
  // 先按名称建索引，再据 parent 挂到父节点下
  const map = new Map()
  list.forEach((c) => {
    map.set(c.category, {
      id: 'real:' + c.category,
      name: c.category,
      category: c.category,
      parent: c.parent || '',
      directCount: c.count,
      count: c.count,
      books: (c.books || []).map((b) => ({ ...b })),
      realCategory: c.category,
      children: [],
      depth: 0
    })
  })
  const roots = []
  map.forEach((node) => {
    if (node.parent && map.has(node.parent)) {
      map.get(node.parent).children.push(node)
    } else {
      roots.push(node)
    }
  })
  // 递归累加子孙图书数，使父级显示聚合数量（含深度保护防环）
  const sum = (n, d = 0) => {
    if (d > 20) return 0
    let s = n.directCount
    for (const ch of n.children) s += sum(ch, d + 1)
    n.count = s
    return s
  }
  roots.forEach((r) => sum(r))
  const total = roots.reduce((s, r) => s + r.count, 0)
  return {
    id: 'root',
    name: '全部',
    category: null,
    count: total,
    books: [],
    realCategory: null,
    children: roots,
    depth: -1
  }
}

function buildAndResetTree() {
  rootNode.value = buildTree(rawCategories.value)
  path.value = [rootNode.value]
}

const currentNode = computed(() => path.value[path.value.length - 1] || null)
const breadcrumb = computed(() => path.value)
const isRoot = computed(() => path.value.length <= 1)

const filteredChildren = computed(() => {
  const kids = currentNode.value ? currentNode.value.children : []
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return kids
  return kids.filter((c) => c.name.toLowerCase().includes(q))
})

const openNode = (node) => { path.value = [...path.value, node] }
const goToCrumb = (idx) => { path.value = path.value.slice(0, idx + 1) }

// 在分类树中按名称定位节点，并将 path 设为「根 -> 目标」完整路径（用于创建后跳转到该分类）
const focusCategoryByName = (name) => {
  if (!rootNode.value) return
  const find = (node, trail) => {
    const next = [...trail, node]
    if (node.category === name) return next
    for (const ch of node.children || []) {
      const r = find(ch, next)
      if (r) return r
    }
    return null
  }
  const p = find(rootNode.value, [])
  if (p) path.value = p
}

// 当前节点图书（分页）
const nodeBooks = ref([])
const catBooksLoading = ref(false)
const displayBooks = computed(() => nodeBooks.value)
const bookTotal = ref(0)

// 收集某节点自身及其所有子孙分类名（用于递归拉取图书）
const collectCategoryNames = (node, acc = []) => {
  if (node.realCategory) acc.push(node.realCategory)
  for (const ch of node.children || []) collectCategoryNames(ch, acc)
  return acc
}

// 请求序号：确保并发的多次 loadNodeBooks 中，最后一次调用始终生效（避免旧的根节点/父级请求覆盖当前分类数据）
let nodeBooksSeq = 0
const loadNodeBooks = async (node, resetPage = false) => {
  if (!node) { nodeBooks.value = []; bookTotal.value = 0; return }
  if (resetPage) bookPage.value = 1
  const seq = ++nodeBooksSeq
  catBooksLoading.value = true
  try {
    // 根节点拉取全部；分类节点（含子分类）按分类名集合过滤（后端 category 支持逗号分隔）
    const params = node.realCategory
      ? { category: collectCategoryNames(node).join(','), page: bookPage.value, page_size: bookPageSize.value }
      : { page: bookPage.value, page_size: bookPageSize.value }
    const res = await bookApi.listBooks(params)
    if (seq !== nodeBooksSeq) return // 已是过期请求，丢弃
    nodeBooks.value = res.data.data || []
    bookTotal.value = res.data.total || 0
  } catch (e) {
    if (seq !== nodeBooksSeq) return
    nodeBooks.value = []
    bookTotal.value = 0
  } finally {
    if (seq === nodeBooksSeq) catBooksLoading.value = false
  }
}
const onBookPageChange = () => loadNodeBooks(currentNode.value)
watch(currentNode, (n) => loadNodeBooks(n, true), { immediate: true })

const router = useRouter()

// 书籍卡片工具函数（与收藏夹一致）
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
const goToDetail = (id) => router.push(`/book/${id}`)
const goRead = (id) => router.push(`/reader/${id}`)

// 从当前节点移除某本书
const removeBookFromCategory = async (book) => {
  const node = currentNode.value
  if (!node) return
  try {
    await ElMessageBox.confirm(
      `确定将《${book.title}》移出分类「${node.name}」吗？`,
      '移出分类',
      { confirmButtonText: '移出', cancelButtonText: '取消', type: 'warning' }
    )
  } catch {
    return
  }
  try {
    // 用图书自身的分类（而非当前节点）移除，父级聚合视图下也能正确移出
    if (book.category) {
      await bookApi.removeBooksFromCategory(book.category, [book.id])
      ElMessage.success('已移出分类')
    }
    nodeBooks.value = nodeBooks.value.filter((b) => b.id !== book.id)
    node.count = Math.max(0, node.count - 1)
    bookTotal.value = Math.max(0, bookTotal.value - 1)
  } catch (e) {
    ElMessage.error('移出失败')
  }
}

const loadCategories = async () => {
  loading.value = true
  try {
    const res = await bookApi.getCategories()
    const list = res.data.data || []
    const withBooks = await Promise.all(
      list.map(async (c) => {
        try {
          const b = await bookApi.listBooks({ category: c.category, page_size: 4 })
          return { ...c, books: b.data.data || [] }
        } catch (e) {
          return { ...c, books: [] }
        }
      })
    )
    rawCategories.value = withBooks
    buildAndResetTree()
  } catch (e) {
    // 列表加载失败不弹错误提醒（无分类属正常情况），静默降级为空列表
    console.warn('加载分类失败', e)
  } finally {
    loading.value = false
  }
}

const totalBooks = computed(() => rawCategories.value.reduce((s, c) => s + c.count, 0))

const iconGradient = (name) => {
  const gradients = [
    'linear-gradient(135deg, var(--app-primary), color-mix(in srgb, var(--app-primary) 55%, #8b5cf6))',
    'linear-gradient(135deg, #06b6d4, #22d3ee)',
    'linear-gradient(135deg, #f59e0b, #f97316)',
    'linear-gradient(135deg, #10b981, #059669)',
    'linear-gradient(135deg, #ec4899, #f43f5e)',
    'linear-gradient(135deg, #3b82f6, #06b6d4)',
    'linear-gradient(135deg, #a855f7, var(--app-primary))',
    'linear-gradient(135deg, #14b8a6, #0ea5e9)'
  ]
  let h = 0
  for (const ch of name) h = (h * 31 + ch.charCodeAt(0)) >>> 0
  return gradients[h % gradients.length]
}

const onCoverError = (e) => {
  e.target.style.visibility = 'hidden'
}

// 新增分类（支持指定父级，创建子分类）
const createVisible = ref(false)
const createForm = ref({ name: '', bookIds: [], parent: '' })

const openCreate = async (parentCat = null) => {
  createForm.value = { name: '', bookIds: [], parent: parentCat && parentCat.realCategory ? parentCat.realCategory : '' }
  createBookQuery.value = ''
  // 每次打开都重新拉取，确保候选列表与「添加书籍到分类」一致（仅显示未分类图书）
  try {
    const res = await bookApi.listBooks({ page_size: 100 })
    allBooks.value = res.data.data || []
  } catch (e) { /* 忽略 */ }
  createVisible.value = true
}

// 新增分类弹窗的图书选择（与「添加书籍到分类」列表一致）
const createBookQuery = ref('')
const filteredCreateBooks = computed(() => {
  const q = createBookQuery.value.trim().toLowerCase()
  const list = availableBooks.value
  if (!q) return list
  return list.filter(
    (b) => (b.title || '').toLowerCase().includes(q) || (b.author || '').toLowerCase().includes(q)
  )
})
const toggleCreateBook = (id) => {
  const arr = createForm.value.bookIds
  const idx = arr.indexOf(id)
  if (idx >= 0) arr.splice(idx, 1)
  else arr.push(id)
}

const submitCreate = async () => {
  const name = createForm.value.name.trim()
  if (!name) return ElMessage.warning('请输入分类名称')
  saving.value = true
  try {
    await bookApi.createCategory({ name, book_ids: createForm.value.bookIds, parent: createForm.value.parent })
    ElMessage.success('分类创建成功')
    createVisible.value = false
    await loadCategories()
    focusCategoryByName(name)
  } catch (e) {
    ElMessage.error('创建分类失败')
  } finally {
    saving.value = false
  }
}

// 重命名分类
const renameVisible = ref(false)
const renameForm = ref({ oldName: '', newName: '' })

const openRename = (cat) => {
  renameForm.value = { oldName: cat.category, newName: cat.category }
  renameVisible.value = true
}

const submitRename = async () => {
  const newName = renameForm.value.newName.trim()
  if (!newName) return ElMessage.warning('请输入新名称')
  if (newName === renameForm.value.oldName) {
    renameVisible.value = false
    return
  }
  if (categories.value.some((c) => c.category === newName && c.category !== renameForm.value.oldName)) {
    return ElMessage.warning('已存在同名分类')
  }
  saving.value = true
  try {
    await bookApi.renameCategory(renameForm.value.oldName, newName)
    ElMessage.success('重命名成功')
    renameVisible.value = false
    loadCategories()
  } catch (e) {
    ElMessage.error('重命名失败')
  } finally {
    saving.value = false
  }
}

// 删除分类
const deleteVisible = ref(false)
const deleteForm = ref({ name: '', count: 0, moveTo: '' })

const moveTargets = computed(() =>
  rawCategories.value.map((c) => c.category).filter((c) => c !== deleteForm.value.name)
)

const openDelete = (cat) => {
  deleteForm.value = { name: cat.category, count: cat.count, moveTo: '' }
  deleteVisible.value = true
}

const submitDelete = async () => {
  saving.value = true
  try {
    await bookApi.deleteCategory(deleteForm.value.name, deleteForm.value.moveTo)
    ElMessage.success('分类已删除')
    deleteVisible.value = false
    loadCategories()
  } catch (e) {
    ElMessage.error('删除分类失败')
  } finally {
    saving.value = false
  }
}

// 向已有分类追加书籍
const addBooksVisible = ref(false)
const addBooksForm = ref({ name: '', bookIds: [] })

// 仅展示尚未归入任何分类的图书（每本书只能属于一个分类）
const availableBooks = computed(() => allBooks.value.filter((b) => !b.category))

const openAddBooks = async (cat) => {
  addBooksForm.value = { name: cat.category, bookIds: [] }
  addBookQuery.value = ''
  // 每次打开都重新拉取，确保刚添加完的图书不再出现
  try {
    const res = await bookApi.listBooks({ page_size: 100 })
    allBooks.value = res.data.data || []
  } catch (e) { /* 忽略 */ }
  addBooksVisible.value = true
}

const toggleBook = (id) => {
  const arr = addBooksForm.value.bookIds
  const idx = arr.indexOf(id)
  if (idx >= 0) arr.splice(idx, 1)
  else arr.push(id)
}

const refreshCurrentBooks = () => loadNodeBooks(currentNode.value)

const addBookQuery = ref('')
const filteredAddBooks = computed(() => {
  const q = addBookQuery.value.trim().toLowerCase()
  if (!q) return availableBooks.value
  return availableBooks.value.filter(
    (b) => (b.title || '').toLowerCase().includes(q) || (b.author || '').toLowerCase().includes(q)
  )
})

const removeBookQuery = ref('')
const filteredRemoveBooks = computed(() => {
  const list = removeBooksForm.value.candidates || []
  const q = removeBookQuery.value.trim().toLowerCase()
  if (!q) return list
  return list.filter(
    (b) => (b.title || '').toLowerCase().includes(q) || (b.author || '').toLowerCase().includes(q)
  )
})

const toggleRemoveBook = (id) => {
  const arr = removeBooksForm.value.bookIds
  const idx = arr.indexOf(id)
  if (idx >= 0) arr.splice(idx, 1)
  else arr.push(id)
}

const submitAddBooks = async () => {
  if (addBooksForm.value.bookIds.length === 0) return ElMessage.warning('请至少选择一本图书')
  saving.value = true
  try {
    await bookApi.addBooksToCategory(addBooksForm.value.name, addBooksForm.value.bookIds)
    ElMessage.success('已添加到分类')
    addBooksVisible.value = false
    const targetName = addBooksForm.value.name
    await loadCategories()
    focusCategoryByName(targetName) // 重新加载后停留在当前分类，不跳回「全部」
    refreshCurrentBooks()
  } catch (e) {
    ElMessage.error('添加失败')
  } finally {
    saving.value = false
  }
}

// 从分类移除图书
const removeBooksVisible = ref(false)
const removeBooksForm = ref({ name: '', bookIds: [], candidates: [] })

const openRemoveBooks = async (cat) => {
  removeBooksForm.value = { name: cat.category, bookIds: [], candidates: [] }
  removeBookQuery.value = ''
  try {
    const res = await bookApi.listBooks({ category: cat.category, page_size: 1000 })
    removeBooksForm.value.candidates = res.data.data || []
  } catch (e) { /* 忽略 */ }
  removeBooksVisible.value = true
}

const submitRemoveBooks = async () => {
  if (removeBooksForm.value.bookIds.length === 0) return ElMessage.warning('请至少选择一本要移除的图书')
  saving.value = true
  try {
    await bookApi.removeBooksFromCategory(removeBooksForm.value.name, removeBooksForm.value.bookIds)
    ElMessage.success('已从分类移除')
    removeBooksVisible.value = false
    loadCategories()
    refreshCurrentBooks()
  } catch (e) {
    ElMessage.error('移除失败')
  } finally {
    saving.value = false
  }
}

onMounted(loadCategories)
</script>

<style scoped>
.category-manage {
  position: relative;
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
.view-toolbar,
.loading,
.empty,
.cat-grid,
.list-view {
  position: relative;
  z-index: 1;
}

/* ========== 页面头部 ========== */
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

.head-left {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.head-back {
  color: #64748b;
  padding-left: 6px;
  margin-right: 2px;
}
.head-back:hover {
  color: var(--app-primary);
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

.head-actions {
  flex-shrink: 0;
}

.add-btn {
  background: linear-gradient(135deg, var(--app-primary), color-mix(in srgb, var(--app-primary) 55%, #8b5cf6));
  border: none;
  box-shadow: 0 8px 20px color-mix(in srgb, var(--app-primary) 35%, transparent);
  transition: all 0.25s ease;
}
.add-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 12px 28px color-mix(in srgb, var(--app-primary) 45%, transparent);
}

/* ========== 统计卡片 ========== */
.stats-row {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
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
  transition: transform 0.5s ease;
}

.stat-card:hover .stat-shine {
  transform: scale(1.8);
}

/* ========== 工具栏 ========== */
.view-toolbar {
  position: relative;
  display: flex;
  align-items: center;
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

.vt-search {
  flex: 1;
  min-width: 0;
  max-width: 360px;
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
.search-input :deep(.el-input__prefix) {
  color: #94a3b8;
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

/* ========== 分类网格卡片（与收藏夹一致） ========== */
.cat-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(210px, 1fr));
  gap: 20px;
  min-height: 120px;
}
.cat-card {
  position: relative;
  border-radius: 16px;
  overflow: hidden;
  cursor: pointer;
  background: #fff;
  box-shadow: 0 4px 16px rgba(31, 38, 135, 0.08);
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1), box-shadow 0.3s ease;
}
.cat-card::before {
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
.cat-card:hover {
  transform: translateY(-6px) scale(1.02);
  box-shadow: 0 20px 48px rgba(245, 158, 11, 0.28);
}
.cat-card:hover::before {
  opacity: 1;
}
.cat-cover {
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
.cat-card:hover .cover-item {
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
.cat-card:hover .cover-shine {
  transform: translateX(120%);
}
.cat-count {
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
.cat-info {
  padding: 14px 16px 16px;
}
.cat-name {
  margin: 0 0 4px;
  font-size: 16px;
  font-weight: 700;
  color: #0f172a;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.cat-meta {
  margin: 0;
  font-size: 13px;
  color: #64748b;
}
/* 操作按钮：仿收藏夹，默认隐藏，悬浮左上角显现 */
.cat-actions {
  position: absolute;
  top: 10px;
  left: 10px;
  display: flex;
  gap: 4px;
  opacity: 0;
  transform: translateX(-10px);
  transition: opacity 0.25s ease, transform 0.25s ease;
  z-index: 2;
}
.cat-card:hover .cat-actions {
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

/* ========== 面包屑导航（置于标题下方，非独立行） ========== */
.crumb-bar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: 6px;
}
.crumb-root {
  color: var(--app-primary);
  margin-right: 2px;
}
.crumb-item {
  font-size: 14px;
  color: #64748b;
  cursor: pointer;
  padding: 2px 6px;
  border-radius: 8px;
  transition: color 0.2s ease, background 0.2s ease;
}
.crumb-item:hover {
  color: var(--app-primary);
  background: color-mix(in srgb, var(--app-primary) 10%, transparent);
}
.crumb-item.active {
  color: var(--app-primary);
  font-weight: 700;
  cursor: default;
}
.crumb-sep {
  color: #cbd5e1;
  font-size: 12px;
}

/* ========== 分区 ========== */
.section { margin-top: 28px; }
.book-grid { row-gap: 22px; }

/* 图书列表视图 */
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
  box-shadow: 0 14px 34px rgba(245, 158, 11, 0.18);
  border-color: color-mix(in srgb, #f59e0b 40%, transparent);
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

/* ========== 图书分页 ========== */
.pagination {
  margin-top: 24px;
  display: flex;
  justify-content: flex-end;
}
.section-head {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 22px 0 16px;
}
.section-bar {
  width: 4px;
  height: 18px;
  border-radius: 4px;
  background: linear-gradient(180deg, #f59e0b, #f97316);
  box-shadow: 0 2px 8px rgba(245, 158, 11, 0.4);
}
.section-title {
  margin: 0;
  font-size: 17px;
  font-weight: 700;
  color: #0f172a;
}
.table-cat-sub {
  font-size: 13px;
  color: #94a3b8;
}

/* ========== 列表视图 ========== */
.list-view {
  border-radius: 14px;
  overflow: hidden;
  box-shadow: 0 8px 32px rgba(31, 38, 135, 0.1);
  background: #fff;
  padding: 8px;
}
.table-cat {
  display: flex;
  align-items: center;
  gap: 10px;
}
.table-cat-icon {
  flex-shrink: 0;
  width: 34px;
  height: 34px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
}
.table-cat-name {
  font-weight: 600;
  color: #0f172a;
}
.count-badge {
  display: inline-block;
  padding: 3px 10px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
  color: var(--app-primary);
  background: linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 12%, transparent), rgba(34, 211, 238, 0.12));
  border: 1px solid color-mix(in srgb, var(--app-primary) 25%, transparent);
}
.list-view .act-btn {
  margin-left: 0;
}
.list-view .el-button + .el-button {
  margin-left: 8px;
}

/* ========== 骨架屏 ========== */
.loading {
  margin-top: 4px;
}
.skeleton-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 20px;
}
.skeleton-card {
  border-radius: 16px;
  overflow: hidden;
  background: #fff;
  padding: 0 0 16px;
  box-shadow: 0 4px 16px rgba(31, 38, 135, 0.08);
}
.skeleton-cover {
  height: 110px;
  background: linear-gradient(90deg, #eef2f7 25%, #e2e8f0 37%, #eef2f7 63%);
  background-size: 400% 100%;
  animation: shimmer 1.4s ease infinite;
}
.skeleton-line {
  height: 14px;
  margin: 14px 16px 0;
  border-radius: 6px;
  background: linear-gradient(90deg, #eef2f7 25%, #e2e8f0 37%, #eef2f7 63%);
  background-size: 400% 100%;
  animation: shimmer 1.4s ease infinite;
}
.skeleton-line.short {
  width: 50%;
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

/* ========== 弹窗内文案 ========== */
.delete-tip {
  margin: 0 0 14px;
  color: #475569;
  font-size: 14px;
}
.add-books-target b,
.delete-tip b {
  color: var(--app-primary);
}
/* 删除分类弹窗：二选一卡片 + 目标分类下拉 */
.delete-options {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.delete-option {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  border-radius: 14px;
  background: #fff;
  border: 1px solid #e2e8f0;
  box-shadow: 0 2px 8px rgba(31, 38, 135, 0.04);
  cursor: pointer;
  transition: all 0.2s ease;
}
.delete-option:hover:not(.disabled) {
  border-color: color-mix(in srgb, var(--app-primary) 35%, transparent);
  box-shadow: 0 6px 18px rgba(31, 38, 135, 0.08);
}
.delete-option.active {
  background: color-mix(in srgb, var(--app-primary) 6%, transparent);
  border-color: color-mix(in srgb, var(--app-primary) 50%, transparent);
  box-shadow: 0 6px 18px color-mix(in srgb, var(--app-primary) 12%, transparent);
}
.delete-option.disabled {
  opacity: 0.55;
  cursor: not-allowed;
  background: #f8fafc;
}
.do-radio {
  flex-shrink: 0;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 2px solid #cbd5e1;
  color: #fff;
  transition: all 0.2s ease;
}
.delete-option.active .do-radio {
  background: var(--app-primary);
  border-color: var(--app-primary);
}
.do-body {
  flex: 1;
  min-width: 0;
}
.do-title {
  font-size: 14px;
  font-weight: 600;
  color: #1e293b;
  margin-bottom: 4px;
}
.do-desc {
  font-size: 12px;
  color: #64748b;
}
.delete-target-wrap {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 4px;
  padding: 12px 16px;
  border-radius: 12px;
  background: color-mix(in srgb, var(--app-primary) 5%, transparent);
  border: 1px dashed color-mix(in srgb, var(--app-primary) 25%, transparent);
}
.dt-label {
  flex-shrink: 0;
  font-size: 13px;
  font-weight: 500;
  color: #475569;
}
.dt-select {
  flex: 1;
  min-width: 0;
}

/* ========== 添加弹框：书籍列表（对齐有声书弹框列表样式） ========== */
.add-books-target {
  margin: 0 0 14px;
  padding: 10px 14px;
  border-radius: 12px;
  font-size: 13px;
  color: #475569;
  background: linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 7%, transparent), rgba(34, 211, 238, 0.05));
  border: 1px solid color-mix(in srgb, var(--app-primary) 15%, transparent);
}
.add-books-target b {
  color: var(--app-primary);
}
/* 新增分类弹窗：图书选择与「添加书籍到分类」弹窗完全一致的间距 */
.create-book-section {
  margin-top: 18px;
}
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
.add-count {
  flex-shrink: 0;
  font-size: 12px;
  font-weight: 600;
  color: var(--app-primary);
  padding: 7px 12px;
  border-radius: 999px;
  background: linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 12%, transparent), rgba(34, 211, 238, 0.12));
  border: 1px solid color-mix(in srgb, var(--app-primary) 25%, transparent);
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
  border-color: color-mix(in srgb, var(--app-primary) 30%, transparent);
}
.add-item.selected {
  background: color-mix(in srgb, var(--app-primary) 6%, transparent);
  border-color: color-mix(in srgb, var(--app-primary) 45%, transparent);
  box-shadow: 0 6px 18px color-mix(in srgb, var(--app-primary) 14%, transparent);
}
.add-cover-wrap {
  position: relative;
  flex-shrink: 0;
  width: 44px;
  height: 58px;
  border-radius: 6px;
  background: color-mix(in srgb, var(--app-primary) 8%, transparent);
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
  background: var(--app-gradient);
  border: 2px solid #fff;
  box-shadow: 0 2px 8px color-mix(in srgb, var(--app-primary) 45%, transparent);
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
  border: 1px dashed color-mix(in srgb, var(--app-primary) 25%, transparent);
  border-radius: 12px;
}
.gradient-btn {
  background: var(--app-gradient) !important;
  border: none !important;
  box-shadow: 0 6px 18px color-mix(in srgb, var(--app-primary) 35%, transparent);
}

/* ========== 响应式 ========== */
@media (max-width: 768px) {
  .page-head {
    flex-direction: column;
    align-items: flex-start;
    gap: 14px;
    padding: 16px 18px;
  }
  .head-title-icon { width: 40px; height: 40px; }
  .head-title-text h2 { font-size: 18px; }
  .head-title-text p { font-size: 12px; }
  .head-actions { width: 100%; }
  .add-btn { width: 100%; }
  .stats-row { grid-template-columns: repeat(2, 1fr); gap: 12px; }
  .stat-card { padding: 14px 12px; }
  .stat-value { font-size: 22px; }
  .view-toolbar {
    flex-direction: column;
    align-items: stretch;
    gap: 10px;
    padding: 10px 12px;
  }
  .vt-search { max-width: 100%; }
  .vt-divider { width: 100%; height: 1px; }
  .cat-grid { grid-template-columns: repeat(auto-fill, minmax(160px, 1fr)); gap: 14px; }
  .cat-cover { height: 100px; }
  /* 移动端无 hover，操作按钮常显 */
  .cat-actions { opacity: 1; transform: none; }
  .cat-actions .act-btn { width: 30px; height: 30px; }
}
@media (max-width: 400px) {
  .stats-row { grid-template-columns: repeat(2, 1fr); gap: 10px; }
  .stat-card { padding: 12px 10px; }
}

/* ===== 分类详情：书籍卡片（与收藏夹一致） ===== */
.book-grid { min-height: 120px; }
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
.book-card:hover::before { opacity: 1; }
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
.book-card:hover .book-cover-wrapper::after { transform: translateX(120%); }
.book-cover {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 0.4s ease;
}
.book-card:hover .book-cover { transform: scale(1.08); }
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
.book-card:hover .book-actions { opacity: 1; transform: translateX(0); }
.book-actions .el-button + .el-button { margin-left: 0; }
.action-btn { box-shadow: 0 4px 12px rgba(31, 38, 135, 0.2); }
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
.book-info { padding: 12px 14px 14px; }
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
.cardIn {
  animation: cardIn 0.55s cubic-bezier(0.22, 1, 0.36, 1) both;
  animation-delay: calc(var(--i, 0) * 80ms);
}
@keyframes cardIn {
  from { opacity: 0; transform: translateY(16px); }
  to { opacity: 1; transform: translateY(0); }
}

/* ===== 分类详情头部 ===== */
.cat-detail { margin-top: 8px; }
.detail-head {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 16px;
  margin-bottom: 18px;
  background: #fff;
  border: 1px solid #eef2f7;
  border-radius: 16px;
  box-shadow: 0 4px 16px rgba(31, 38, 135, 0.06);
}
.back-btn {
  color: #64748b;
  padding-left: 6px;
}
.back-btn:hover { color: var(--app-primary); }
.detail-title-wrap {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
  min-width: 0;
}
.detail-icon {
  width: 42px;
  height: 42px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  background: var(--app-gradient, linear-gradient(135deg, #6366f1, #8b5cf6));
  box-shadow: 0 8px 18px color-mix(in srgb, var(--app-primary, #6366f1) 30%, transparent);
  flex-shrink: 0;
}
.detail-text { min-width: 0; }
.detail-text h2 {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: #1e293b;
}
.detail-meta {
  font-size: 12px;
  color: #64748b;
}
.detail-actions {
  display: flex;
  gap: 10px;
  flex-shrink: 0;
}
@media (max-width: 768px) {
  .detail-head { flex-wrap: wrap; }
  .detail-actions { width: 100%; }
  .detail-actions .el-button { flex: 1; }
}

/* ========== 父级分类标识 ========== */
.parent-chip {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 6px 14px;
  border-radius: 10px;
  background: color-mix(in srgb, var(--app-primary) 10%, transparent);
  color: var(--app-primary);
  font-size: 14px;
  font-weight: 600;
}

</style>
