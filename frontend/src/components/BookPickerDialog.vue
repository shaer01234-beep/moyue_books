<template>
  <el-dialog
    v-model="localVisible"
    :title="title"
    width="760px"
    class="book-picker-dialog"
    append-to-body
    destroy-on-close
  >
    <div class="picker-search">
      <el-input
        v-model="keyword"
        placeholder="搜索图书标题"
        clearable
        :prefix-icon="Search"
        @input="filterBooks"
      >
        <template #append>
          <el-button :icon="Search" @click="filterBooks" />
        </template>
      </el-input>
    </div>

    <div class="picker-grid" v-if="filtered.length">
      <div
        v-for="b in filtered"
        :key="b.id"
        class="picker-item"
        :class="{ selected: modelValue === b.id }"
        @click="select(b)"
      >
        <div class="picker-cover">
          <img v-if="b.cover_path" :src="coverUrl(b.id)" :alt="b.title" loading="lazy" />
          <div v-else class="cover-fallback">
            <el-icon :size="24"><Reading /></el-icon>
          </div>
          <div class="pick-check" v-if="modelValue === b.id">
            <el-icon :size="16"><Check /></el-icon>
          </div>
        </div>
        <div class="picker-title">{{ b.title }}</div>
      </div>
    </div>

    <div v-else class="picker-empty">
      <el-icon :size="40"><Reading /></el-icon>
      <p>{{ books.length ? '未找到匹配的图书' : '书架暂无图书，请先导入' }}</p>
    </div>

    <div class="picker-tip" v-if="books.length">
      <el-icon :size="14"><Mouse /></el-icon>
      点击任意一本书即可选中
    </div>
  </el-dialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { Search, Reading, Check, Mouse } from '@element-plus/icons-vue'
import { bookApi } from '@/api/book'

const props = defineProps({
  modelValue: { type: Number, default: null },
  visible: { type: Boolean, default: false },
  title: { type: String, default: '选择图书' }
})
const emit = defineEmits(['update:visible', 'update:modelValue', 'select'])

const books = ref([])
const keyword = ref('')
const filtered = ref([])
const picked = ref(null)

const localVisible = computed({
  get: () => props.visible,
  set: (v) => emit('update:visible', v)
})

function coverUrl(id) {
  return bookApi.coverUrl(id)
}

async function loadBooks() {
  try {
    const res = await bookApi.listBooks({ page: 1, page_size: 1000 })
    books.value = res.data?.data || res.data || []
    filtered.value = [...books.value]
  } catch (e) {
    books.value = []
    filtered.value = []
  }
}

function filterBooks() {
  const k = keyword.value.trim().toLowerCase()
  if (!k) {
    filtered.value = [...books.value]
    return
  }
  filtered.value = books.value.filter((b) => (b.title || '').toLowerCase().includes(k))
}

function select(b) {
  picked.value = b.id
  emit('update:modelValue', b.id)
  emit('select', b)
}

watch(
  () => props.visible,
  (v) => {
    if (v) {
      keyword.value = ''
      picked.value = props.modelValue
      loadBooks()
    }
  }
)
</script>

<style scoped>
.book-picker-dialog :deep(.el-dialog) {
  border-radius: 18px;
  background: linear-gradient(160deg, #fdfdff, #f6f8fc);
}
.picker-search {
  margin-bottom: 16px;
}
.picker-search :deep(.el-input__wrapper) {
  border-radius: 12px 0 0 12px;
  background: rgba(255, 255, 255, 0.95);
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--app-primary) 18%, transparent) inset;
}
.picker-search :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1.5px var(--app-primary), 0 6px 20px color-mix(in srgb, var(--app-primary) 20%, transparent);
}
.picker-search :deep(.el-input-group__append) {
  border-radius: 0 12px 12px 0;
}

.picker-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(112px, 1fr));
  gap: 14px;
  max-height: 52vh;
  overflow: auto;
  padding: 4px;
}
.picker-item {
  cursor: pointer;
  transition: transform 0.2s ease;
  border-radius: 12px;
  padding: 6px;
}
.picker-item:hover {
  transform: translateY(-3px);
}
.picker-cover {
  position: relative;
  aspect-ratio: 3 / 4;
  border-radius: 10px;
  overflow: hidden;
  background: linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 45%, transparent), rgba(34, 211, 238, 0.45));
  box-shadow: 0 6px 18px rgba(31, 38, 135, 0.18);
  display: flex;
  align-items: center;
  justify-content: center;
}
.picker-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.cover-fallback {
  color: #fff;
}
.picker-item.selected .picker-cover {
  box-shadow: 0 0 0 3px var(--app-primary), 0 8px 24px color-mix(in srgb, var(--app-primary) 35%, transparent);
}
.pick-check {
  position: absolute;
  top: 6px;
  right: 6px;
  width: 26px;
  height: 26px;
  border-radius: 50%;
  background: var(--app-gradient);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}
.picker-title {
  margin-top: 8px;
  font-size: 12px;
  color: #475569;
  text-align: center;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.picker-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 48px 0;
  color: #94a3b8;
}
.picker-tip {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  margin-top: 14px;
  padding-top: 12px;
  border-top: 1px dashed #e2e8f0;
  font-size: 12px;
  color: #94a3b8;
}
.picker-tip .el-icon {
  color: var(--app-primary);
}
.picker-empty p {
  margin: 0;
  font-size: 13px;
}
</style>
