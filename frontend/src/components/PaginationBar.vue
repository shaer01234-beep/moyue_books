<template>
  <!-- 桌面端：保持完整分页 -->
  <el-pagination
    v-if="!isMobile"
    v-model:current-page="current"
    v-model:page-size="size"
    :total="total"
    :page-sizes="pageSizes"
    layout="total, sizes, prev, pager, next, jumper"
    background
    @current-change="(p) => emit('current-change', p)"
    @size-change="(s) => emit('size-change', s)"
  />

  <!-- 手机端：简洁分页（上一页 / 第 X / Y 页 / 下一页） -->
  <div v-else class="mp-bar">
    <el-button class="mp-btn" :disabled="current <= 1" @click="go(current - 1)">
      <el-icon><ArrowLeft /></el-icon>
      <span>上一页</span>
    </el-button>
    <span class="mp-info">第 {{ current }} / {{ maxPage }} 页</span>
    <el-button
      class="mp-btn"
      type="primary"
      :disabled="current >= maxPage"
      @click="go(current + 1)"
    >
      <span>下一页</span>
      <el-icon><ArrowRight /></el-icon>
    </el-button>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { ArrowLeft, ArrowRight } from '@element-plus/icons-vue'

const props = defineProps({
  currentPage: { type: Number, required: true },
  pageSize: { type: Number, required: true },
  total: { type: Number, required: true },
  pageSizes: { type: Array, default: () => [10, 20, 50, 100] }
})
const emit = defineEmits([
  'update:currentPage',
  'update:pageSize',
  'current-change',
  'size-change'
])

const current = computed({
  get: () => props.currentPage,
  set: (v) => emit('update:currentPage', v)
})
const size = computed({
  get: () => props.pageSize,
  set: (v) => emit('update:pageSize', v)
})
const maxPage = computed(() =>
  Math.max(1, Math.ceil(props.total / Math.max(1, props.pageSize)))
)

const isMobile = ref(false)
const checkMobile = () => { isMobile.value = window.innerWidth <= 768 }
let resizeTimer = null
const onResize = () => {
  if (resizeTimer) clearTimeout(resizeTimer)
  resizeTimer = setTimeout(checkMobile, 150)
}
onMounted(() => {
  checkMobile()
  window.addEventListener('resize', onResize)
})
onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize)
  if (resizeTimer) clearTimeout(resizeTimer)
})

const go = (p) => {
  const next = Math.min(Math.max(1, p), maxPage.value)
  if (next === current.value) return
  current.value = next
  emit('current-change', next)
}
</script>

<style scoped>
.mp-bar {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 14px;
  width: 100%;
}
.mp-info {
  font-size: 14px;
  font-weight: 600;
  color: #475569;
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}
.mp-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
</style>
