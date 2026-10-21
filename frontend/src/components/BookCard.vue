<template>
  <div class="book-card cardIn" :style="{ '--i': index }" @click="onClick">
    <div class="book-cover-wrapper">
      <img
        v-if="book.cover_path"
        :src="cover"
        class="book-cover"
        alt="cover"
        @error="onCoverError"
      />
      <div v-else class="book-cover-placeholder">
        <el-icon :size="44"><Reading /></el-icon>
        <span>{{ book.title }}</span>
      </div>
      <div class="cover-shine"></div>
      <span v-if="book.source_missing" class="status-badge missing">源文件缺失</span>
      <span v-else class="status-badge" :class="status.cls">{{ status.text }}</span>
      <div v-if="book.progress > 0" class="book-progress">
        <el-progress
          :percentage="Math.round(book.progress)"
          :stroke-width="5"
          :show-text="false"
          :color="progressColor"
        />
      </div>
      <div v-if="$slots.actions" class="book-actions" @click.stop>
        <slot name="actions" />
      </div>
    </div>
    <div class="book-info">
      <h4 class="book-title" :title="book.title">{{ book.title }}</h4>
      <p class="book-author">{{ formatAuthors(book.authors) }}</p>
      <div v-if="showMeta" class="book-meta">
        <span v-if="book.category" class="cat-tag" :title="book.category">{{ book.category }}</span>
        <span v-if="book.last_read" class="read-time">
          <el-icon><Clock /></el-icon> {{ formatDate(book.last_read) }}
        </span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { bookApi } from '@/api/book'

const props = defineProps({
  book: { type: Object, required: true },
  showMeta: { type: Boolean, default: false },
  index: { type: Number, default: 0 },
  progressColor: {
    type: String,
    default: 'var(--app-primary)'
  }
})
const emit = defineEmits(['click'])

const cover = computed(() => bookApi.coverUrl(props.book.id))

const status = computed(() => {
  const b = props.book
  if (b.progress >= 100) return { text: '已读完', cls: 'done' }
  if (b.progress > 0) return { text: '在读', cls: 'reading' }
  if (b.last_read) return { text: '读过', cls: 'read' }
  return { text: '未读', cls: 'new' }
})

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

const onCoverError = (e) => { e.target.style.visibility = 'hidden' }
const onClick = () => emit('click', props.book)
</script>

<style scoped>
.book-card {
  position: relative;
  border-radius: 10px;
  overflow: hidden;
  cursor: pointer;
  margin-bottom: 20px;
  background: #fff;
  box-shadow: 0 4px 16px rgba(31, 38, 135, 0.08);
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1), box-shadow 0.3s ease;
}
.book-card.cardIn {
  animation: cardIn 0.45s cubic-bezier(0.22, 1, 0.36, 1) both;
}
@keyframes cardIn {
  from { opacity: 0; transform: translateY(16px); }
  to { opacity: 1; transform: translateY(0); }
}
.book-card::before {
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
.book-card:hover {
  transform: translateY(-8px) scale(1.02);
  box-shadow: 0 20px 48px color-mix(in srgb, var(--app-primary) 28%, transparent);
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
.status-badge.done { background: rgba(16, 185, 129, 0.85); color: #fff; }
.status-badge.reading { background: rgba(6, 182, 212, 0.85); color: #fff; }
.status-badge.read { background: rgba(100, 116, 139, 0.75); color: #fff; }
.status-badge.new { background: rgba(255, 255, 255, 0.75); color: var(--app-primary); }
.status-badge.missing { background: rgba(244, 63, 94, 0.9); color: #fff; }
/* 进度条 */
.book-progress {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: 0 6px 6px;
}
/* 阅读进度统一使用主题色渐变（覆盖 el-progress 默认蓝） */
.book-progress :deep(.el-progress-bar__outer) {
  background-color: rgba(255, 255, 255, 0.35);
}
.book-progress :deep(.el-progress-bar__inner) {
  background: linear-gradient(90deg, var(--app-primary), #22d3ee) !important;
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
.book-meta .cat-tag {
  flex: 0 1 auto;
  min-width: 0;
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
/* 操作按钮（作用于插槽内容，需要穿透到父组件作用域） */
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
.book-actions :deep(.el-button + .el-button),
.book-actions :deep(.el-button + .el-dropdown) {
  margin-left: 0;
}
.book-actions :deep(.action-btn) {
  box-shadow: 0 4px 12px rgba(31, 38, 135, 0.2);
}
.book-actions :deep(.audio-btn) {
  background: linear-gradient(135deg, #a855f7, #6366f1) !important;
  border: none !important;
  color: #fff !important;
}
.book-actions :deep(.audio-btn:hover) {
  box-shadow: 0 4px 14px rgba(168, 85, 247, 0.45);
}
.book-actions :deep(.share-btn) {
  background: linear-gradient(135deg, #0ea5e9, #6366f1) !important;
  border: none !important;
  color: #fff !important;
}
.book-actions :deep(.share-btn:hover) {
  box-shadow: 0 4px 14px rgba(14, 165, 233, 0.45);
}
.book-actions :deep(.remove-btn) {
  background: linear-gradient(135deg, #f43f5e, #ef4444) !important;
  border: none !important;
  color: #fff !important;
}
.book-actions :deep(.remove-btn:hover) {
  box-shadow: 0 4px 14px rgba(244, 63, 94, 0.45);
}
</style>
