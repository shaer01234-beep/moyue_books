<template>
  <el-container class="app-layout">
    <!-- 侧边栏（桌面端）；阅读器/读后感书写页为沉浸全屏，隐藏菜单 -->
    <el-aside v-if="!isReader" width="208px" class="sidebar">
      <div class="logo">
        <div class="logo-icon">
          <img v-if="isCustomLogo" :src="logoIcon" class="logo-img" alt="" />
          <el-icon v-else :size="22"><component :is="logoIcon" /></el-icon>
        </div>
        <div class="logo-text">
          <span class="logo-name">{{ siteName }}</span>
          <span class="logo-version">v{{ APP_VERSION }}</span>
        </div>
      </div>
      <el-menu :default-active="activeMenu" router class="el-menu-vertical">
        <el-menu-item index="/bookshelf">
          <el-icon><Reading /></el-icon>
          <span>书架</span>
        </el-menu-item>
        <el-menu-item index="/audiobooks">
          <el-icon><Headset /></el-icon>
          <span>有声书</span>
        </el-menu-item>
        <el-menu-item index="/collections">
          <el-icon><Star /></el-icon>
          <span>收藏夹</span>
        </el-menu-item>
        <el-menu-item index="/categories">
          <el-icon><FolderOpened /></el-icon>
          <span>分类管理</span>
        </el-menu-item>
        <el-menu-item index="/libraries">
          <el-icon><Collection /></el-icon>
          <span>媒体库</span>
        </el-menu-item>
        <el-menu-item index="/stats">
          <el-icon><TrendCharts /></el-icon>
          <span>阅读统计</span>
        </el-menu-item>
        <el-menu-item index="/reviews">
          <el-icon><Notebook /></el-icon>
          <span>读后感</span>
        </el-menu-item>
        <el-menu-item index="/settings">
          <el-icon><Setting /></el-icon>
          <span>设置</span>
        </el-menu-item>
      </el-menu>

      <!-- 用户信息（桌面端） -->
      <div class="user-section">
        <el-dropdown trigger="click" popper-class="user-popper" @command="handleUserCommand">
          <div class="user-card">
            <div class="user-avatar">{{ avatarText }}</div>
            <div class="user-info">
              <div class="user-name">{{ displayName }}</div>
              <div class="user-sub">{{ username }}</div>
            </div>
            <el-icon class="user-arrow"><ArrowDown /></el-icon>
          </div>
          <template #dropdown>
            <div class="user-popper-header">
              <div class="user-popper-avatar">{{ avatarText }}</div>
              <div class="user-popper-info">
                <div class="user-popper-name">{{ displayName }}</div>
                <div class="user-popper-sub">{{ username || '未登录' }}</div>
              </div>
            </div>
            <el-dropdown-menu>
              <el-dropdown-item command="logout">
                <el-icon><SwitchButton /></el-icon>
                <span>退出登录</span>
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </el-aside>

    <!-- 主内容 -->
    <el-main ref="mainContentRef" class="main-content" :class="{ 'is-reader': isReader }">
      <router-view :key="$route.path" />
    </el-main>

    <!-- 全局有声书播放器（后台播放）；阅读器页不展示 -->
    <GlobalPlayer v-if="!isReader" />
  </el-container>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getUser, clearToken, setUser } from '@/api/auth'
import { loadAppPrefs, isCustomIcon } from '@/utils/appPrefs'
import { APP_VERSION } from '@/utils/version'
import GlobalPlayer from '@/components/GlobalPlayer.vue'
import { pausePlay } from '@/player'

const route = useRoute()
const router = useRouter()

// 全局外观配置
const appPrefs = ref(loadAppPrefs())
const siteName = computed(() => appPrefs.value.siteName || '墨阅')
const logoIcon = computed(() => (isCustomIcon(appPrefs.value.siteIcon) ? appPrefs.value.siteIcon : appPrefs.value.siteIcon || 'Collection'))
const isCustomLogo = computed(() => isCustomIcon(appPrefs.value.siteIcon))

// 当前登录用户
const user = ref(getUser())
const displayName = computed(() => user.value?.nickname || user.value?.username || '未登录')
const username = computed(() => user.value?.username || '')
const avatarText = computed(() => (displayName.value || 'U').charAt(0).toUpperCase())

const handleUserCommand = (cmd) => {
  if (cmd === 'logout') {
    clearToken()
    setUser(null)
    user.value = null
    ElMessage.success('已退出登录')
    router.replace('/login')
  }
}

// 阅读器/读后感书写页：独立沉浸布局（全屏）
const isReader = computed(() => route.path.startsWith('/reader') || route.path.startsWith('/reviews/editor'))

// 当前激活菜单：图书详情页（/book/:id）归属「书架」；媒体库详情归属「媒体库」
const activeMenu = computed(() => {
  if (route.path.startsWith('/book/')) return '/bookshelf'
  if (route.path.startsWith('/library/')) return '/libraries'
  return route.path
})

watch(
  () => route.path,
  (path) => {
    if (path.startsWith('/reader') || path.startsWith('/reviews/editor')) pausePlay()
    if (!path.startsWith('/login')) user.value = getUser()
  }
)
</script>

<style scoped>
.app-layout {
  height: 100%;
}

/* ===== 侧边栏（浅色玻璃拟态） ===== */
.sidebar {
  background: rgba(255, 255, 255, 0.72);
  backdrop-filter: blur(18px);
  -webkit-backdrop-filter: blur(18px);
  border-right: 1px solid rgba(255, 255, 255, 0.7);
  box-shadow: 4px 0 28px rgba(31, 38, 135, 0.08);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.logo {
  padding: 22px 18px;
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 17px;
  font-weight: 700;
  color: #0f172a;
  border-bottom: 1px solid color-mix(in srgb, var(--app-primary) 12%, transparent);
  white-space: nowrap;
}

/* 站点名 + 版本号（纵向排列，版本号为名称下方的小字） */
.logo-text {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
  overflow: hidden;
}
.logo-name {
  font-size: 17px;
  font-weight: 700;
  line-height: 1.2;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.logo-version {
  font-size: 11px;
  font-weight: 400;
  line-height: 1.2;
  color: #94a3b8;
  letter-spacing: 0.2px;
}

.logo-icon {
  flex-shrink: 0;
  width: 40px;
  height: 40px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  background: var(--app-gradient);
  box-shadow: 0 8px 20px color-mix(in srgb, var(--app-primary) 35%, transparent);
  overflow: hidden;
}

.logo-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.el-menu-vertical {
  flex: 1;
  border-right: none;
  padding: 12px 10px;
  background: transparent !important;
  overflow-y: auto;
}

:deep(.el-menu-item) {
  font-size: 14px;
  height: 46px;
  line-height: 46px;
  border-radius: 12px;
  margin-bottom: 4px;
  color: #64748b;
  background: transparent;
  transition: all 0.2s ease;
}

:deep(.el-menu-item:hover) {
  background: color-mix(in srgb, var(--app-primary) 8%, transparent);
  color: var(--app-primary);
}

html[data-menu-glow="on"] :deep(.el-menu-item:hover) {
  box-shadow: 0 4px 18px color-mix(in srgb, var(--app-primary) 18%, transparent);
}

:deep(.el-menu-item.is-active) {
  background: linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 14%, transparent), color-mix(in srgb, var(--app-primary) 5%, transparent));
  color: var(--app-primary);
  font-weight: 600;
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--app-primary) 25%, transparent);
}

html[data-menu-glow="on"] :deep(.el-menu-item.is-active) {
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.6),
    inset 0 0 0 1px color-mix(in srgb, var(--app-primary) 25%, transparent),
    0 6px 20px color-mix(in srgb, var(--app-primary) 22%, transparent);
}

:deep(.el-menu-item .el-icon) {
  font-size: 18px;
}

/* ===== 用户卡片（侧边栏底部） ===== */
.user-section {
  padding: 12px;
  border-top: 1px solid color-mix(in srgb, var(--app-primary) 12%, transparent);
}

.user-card {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 12px;
  cursor: pointer;
  transition: background 0.2s ease;
  user-select: none;
}

.user-card:hover {
  background: color-mix(in srgb, var(--app-primary) 8%, transparent);
}

.user-avatar {
  flex-shrink: 0;
  width: 36px;
  height: 36px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 15px;
  font-weight: 600;
  color: #fff;
  background: var(--app-gradient);
  box-shadow: 0 4px 12px color-mix(in srgb, var(--app-primary) 35%, transparent);
}

.user-info {
  flex: 1;
  min-width: 0;
}

.user-name {
  font-size: 13px;
  font-weight: 600;
  color: #1e293b;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.user-sub {
  font-size: 11px;
  color: #94a3b8;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.user-arrow {
  font-size: 12px;
  color: #94a3b8;
}

/* ===== 主内容 ===== */
.main-content {
  background: linear-gradient(135deg, #f5f7fa 0%, #eef2ff 45%, #f0f9ff 100%);
  padding: 24px;
  overflow-y: auto;
  height: 100%;
}

.main-content.is-reader {
  padding: 0;
  overflow: hidden;
}

@media (min-width: 769px) and (max-width: 1024px) {
  .main-content {
    padding: 20px;
  }
}
</style>
