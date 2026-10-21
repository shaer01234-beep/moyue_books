<template>
  <div class="categories-page">
    <CategoryManage />
  </div>
</template>

<script setup>
import CategoryManage from './settings/CategoryManage.vue'
</script>

<style scoped>
.categories-page {
  position: relative;
  /* 与书架页 .bookshelf 一致：1400px 居中，内容铺满 */
  max-width: 1400px;
  margin: 0 auto;
  padding: 0 0 60px;
}

@media (max-width: 768px) {
  .categories-page {
    padding: 0 0 40px;
  }
}
</style>
