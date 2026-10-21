<template>
  <div class="reader-entry">
    <div v-if="!ready && !error" class="re-loading">
      <el-icon class="is-loading"><Loading /></el-icon>
      <span>加载中…</span>
    </div>
    <div v-else-if="error" class="re-error">{{ error }}</div>
    <FoliateReader v-else :id="id" :book="book" />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { Loading } from '@element-plus/icons-vue'
import { bookApi } from '@/api/book'
import FoliateReader from './FoliateReader.vue'

const props = defineProps({
  id: { type: [String, Number], required: true }
})

const ready = ref(false)
const error = ref('')
const book = ref(null)

onMounted(async () => {
  try {
    const { data } = await bookApi.getBook(props.id)
    book.value = data
  } catch (e) {
    error.value = '书籍信息加载失败'
  } finally {
    ready.value = true
  }
})
</script>

<style scoped>
.reader-entry, .re-loading, .re-error {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  gap: 10px;
  color: #64748b;
  background: #f1f5f9;
}
.re-error { color: #ef4444; }
</style>
