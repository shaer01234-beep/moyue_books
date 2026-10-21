<template>
  <div class="lib-hub">
    <!-- tab 条 + 右侧操作区（参考设置页分段式导航） -->
    <div class="hub-bar">
      <nav class="hub-nav">
        <div
          class="nav-item"
          :class="{ active: activeTab === 'library' }"
          @click="switchTab('library')"
        >
          <el-icon :size="18"><Collection /></el-icon>
          <span>媒体库</span>
        </div>
        <div
          class="nav-item"
          :class="{ active: activeTab === 'import' }"
          @click="switchTab('import')"
        >
          <el-icon :size="18"><UploadFilled /></el-icon>
          <span>导入图书</span>
        </div>
      </nav>

      <div class="hub-actions">
        <!-- 媒体库：新建按钮 -->
        <el-button
          v-if="activeTab === 'library'"
          type="primary"
          @click="libRef && libRef.openCreate()"
        >
          <el-icon><Plus /></el-icon> 新建媒体库
        </el-button>
        <!-- 导入图书：搜索最近导入 -->
        <el-input
          v-else
          v-model="importQuery"
          placeholder="搜索最近导入..."
          clearable
          class="hub-search"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
      </div>
    </div>

    <div class="hub-content">
      <LibraryList v-show="activeTab === 'library'" ref="libRef" />
      <Upload v-show="activeTab === 'import'" :recent-query="importQuery" />
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import LibraryList from '@/views/LibraryList.vue'
import Upload from '@/views/Upload.vue'

const route = useRoute()
// 默认显示「媒体库」；链接带 ?tab=import 时直接进入「导入图书」
const activeTab = ref(route.query.tab === 'import' ? 'import' : 'library')
const importQuery = ref('')

const libRef = ref(null)

const switchTab = (key) => {
  if (activeTab.value === key) return
  activeTab.value = key
}

watch(
  () => route.query.tab,
  (t) => {
    if (t === 'import') activeTab.value = 'import'
  }
)
</script>

<style scoped>
.lib-hub {
  position: relative;
  max-width: 1400px;
  margin: 0 auto;
  padding: 0 0 40px;
}

/* 分段式 tab 条（与设置页 .settings-nav 一致） */
.hub-bar {
  position: relative;
  z-index: 1;
  display: flex;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
  margin-bottom: 22px;
  align-items: stretch;
  padding: 8px;
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.75);
  box-shadow: 0 8px 28px rgba(31, 38, 135, 0.08);
}

.hub-nav {
  display: flex;
  align-items: stretch;
  gap: 8px;
  padding: 8px;
}
.nav-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 20px;
  border-radius: 12px;
  cursor: pointer;
  color: #475569;
  font-weight: 600;
  font-size: 15px;
  transition: all 0.2s;
  user-select: none;
  white-space: nowrap;
}
.nav-item:hover {
  background: color-mix(in srgb, var(--app-primary) 6%, transparent);
  color: var(--app-primary);
}
.nav-item.active {
  background: linear-gradient(135deg, var(--app-primary), color-mix(in srgb, var(--app-primary) 55%, #8b5cf6));
  color: #fff;
  box-shadow: 0 8px 20px color-mix(in srgb, var(--app-primary) 35%, transparent);
}

.hub-actions {
  display: flex;
  align-items: center;
  margin-right: 10px;
}
.hub-search {
  width: 240px;
}

.hub-content {
  position: relative;
  z-index: 1;
}

/* ========== 输入框 / 选择框 / 按钮 与主题色一致 ========== */
/* 让 Element 组件内部主色跟随应用主题（聚焦环、下拉高亮等） */
.lib-hub :deep(.el-input),
.lib-hub :deep(.el-select) {
  --el-color-primary: var(--app-primary);
}

/* 输入框 / 选择框：圆角玻璃质感 + 主题色聚焦描边 */
.lib-hub :deep(.el-input__wrapper),
.lib-hub :deep(.el-select__wrapper) {
  background: rgba(255, 255, 255, 0.9);
  border-radius: 12px;
  box-shadow: 0 0 0 1px rgba(148, 163, 184, 0.4) inset;
  transition: box-shadow 0.2s, background 0.2s;
}
.lib-hub :deep(.el-input__wrapper:hover),
.lib-hub :deep(.el-select__wrapper:hover) {
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--app-primary) 60%, transparent) inset;
}
.lib-hub :deep(.el-input__wrapper.is-focus),
.lib-hub :deep(.el-select__wrapper.is-focused) {
  box-shadow: 0 0 0 1px var(--app-primary) inset, 0 4px 14px rgba(31, 38, 135, 0.1);
}

/* 主按钮：使用主题渐变（与 tab 高亮一致），去除默认蓝色 */
.lib-hub :deep(.el-button--primary) {
  background: var(--app-gradient, linear-gradient(135deg, #6366f1, #8b5cf6));
  border: none;
  border-radius: 10px;
  box-shadow: 0 8px 20px color-mix(in srgb, var(--app-primary) 32%, transparent);
  transition: filter 0.2s, transform 0.1s;
}
.lib-hub :deep(.el-button--primary:hover) {
  background: var(--app-gradient, linear-gradient(135deg, #6366f1, #8b5cf6));
  filter: brightness(1.06);
}
.lib-hub :deep(.el-button--primary:active) {
  filter: brightness(0.96);
}
.lib-hub :deep(.el-button--primary.is-disabled) {
  background: var(--app-gradient, linear-gradient(135deg, #6366f1, #8b5cf6));
  opacity: 0.5;
  box-shadow: none;
}

/* 移动端 */
@media (max-width: 768px) {
  .hub-bar {
    flex-direction: column;
    align-items: stretch;
    gap: 10px;
    padding: 8px;
  }
  /* tab 等分铺满，居中显示，避免挤压滚动 */
  .hub-nav {
    overflow-x: auto;
    padding: 4px;
    gap: 6px;
  }
  .nav-item {
    flex: 1 1 0;
    flex-shrink: 0;
    justify-content: center;
    padding: 10px 8px;
    font-size: 14px;
  }
  /* 操作区（按钮 / 搜索框）满宽且高度一致，整齐排布 */
  .hub-actions {
    width: 100%;
    margin-right: 0;
  }
  .hub-actions :deep(.el-button) {
    width: 100%;
    height: 40px;
    border-radius: 10px;
  }
  .hub-search {
    width: 100%;
  }
  .hub-search :deep(.el-input__wrapper) {
    height: 40px;
  }
}
</style>
