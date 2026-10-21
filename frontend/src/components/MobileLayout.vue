<template>
  <div class="m-app">
    <!-- 顶部站头（移动网页风格：站名 + 汉堡菜单） -->
    <div v-if="!isReader" class="m-header">
      <div class="m-header-left">
        <div class="m-menu-btn" @click="drawer = true" role="button" aria-label="菜单">
          <el-icon :size="20"><Menu /></el-icon>
        </div>
        <div class="m-brand">
          <div class="m-brand-icon">
            <img v-if="isCustomLogo" :src="logoIcon" class="m-brand-img" alt="" />
            <el-icon v-else :size="18"><component :is="logoIcon" /></el-icon>
          </div>
          <span class="m-brand-name">{{ siteName }}</span>
        </div>
      </div>
      <el-dropdown trigger="click" popper-class="user-popper" @command="handleUserCommand">
        <div class="m-header-avatar">{{ avatarText }}</div>
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

    <!-- 主内容 -->
    <div ref="scrollRef" class="m-main" :class="{ 'is-reader': isReader }" @scroll="onScroll">
      <router-view :key="$route.path" />
    </div>

    <!-- 左侧汉堡抽屉（移动网页导航） -->
    <transition name="m-mask-fade">
      <div v-if="drawer" class="m-mask" @click="drawer = false" />
    </transition>
    <transition name="m-drawer-slide">
      <aside v-if="drawer" class="m-drawer">
        <div class="m-drawer-head">
          <div class="m-drawer-brand">
            <div class="m-drawer-brand-icon">
              <img v-if="isCustomLogo" :src="logoIcon" class="m-brand-img" alt="" />
              <el-icon v-else :size="20"><component :is="logoIcon" /></el-icon>
            </div>
            <div class="m-drawer-brand-text">
              <span class="m-drawer-brand-name">{{ siteName }}</span>
              <span class="m-drawer-brand-version">v{{ APP_VERSION }}</span>
            </div>
          </div>
        </div>

        <div class="m-drawer-user">
          <div class="m-drawer-avatar">{{ avatarText }}</div>
          <div class="m-drawer-user-info">
            <div class="m-drawer-name">{{ displayName }}</div>
            <div class="m-drawer-sub">{{ username || '未登录' }}</div>
          </div>
        </div>

        <nav class="m-menu">
          <div
            v-for="item in menuItems"
            :key="item.path"
            class="m-menu-item"
            :class="{ active: isNavActive(item) }"
            @click="navTo(item.path)"
          >
            <el-icon :size="19"><component :is="item.icon" /></el-icon>
            <span>{{ item.label }}</span>
          </div>
        </nav>

        <div class="m-drawer-foot">
          <div class="m-menu-item m-logout" @click="logout">
            <el-icon :size="19"><SwitchButton /></el-icon>
            <span>退出登录</span>
          </div>
        </div>
      </aside>
    </transition>

    <!-- 回到顶部 -->
    <transition name="m-backtop-fade">
      <div v-if="!isReader && showBackTop" class="m-backtop" @click="scrollToTop">
        <el-icon :size="18"><Top /></el-icon>
      </div>
    </transition>

    <!-- 全局有声书播放器（后台播放）；阅读器页不展示 -->
    <GlobalPlayer v-if="!isReader" />
  </div>
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

const appPrefs = ref(loadAppPrefs())
const siteName = computed(() => appPrefs.value.siteName || '墨阅')
const logoIcon = computed(() => (isCustomIcon(appPrefs.value.siteIcon) ? appPrefs.value.siteIcon : appPrefs.value.siteIcon || 'Collection'))
const isCustomLogo = computed(() => isCustomIcon(appPrefs.value.siteIcon))

const user = ref(getUser())
const displayName = computed(() => user.value?.nickname || user.value?.username || '未登录')
const username = computed(() => user.value?.username || '')
const avatarText = computed(() => (displayName.value || 'U').charAt(0).toUpperCase())

const handleUserCommand = (cmd) => {
  if (cmd === 'logout') logout()
}
const logout = () => {
  clearToken()
  setUser(null)
  user.value = null
  ElMessage.success('已退出登录')
  router.replace('/login')
}

// 抽屉开关
const drawer = ref(false)
const navTo = (path) => {
  drawer.value = false
  router.push(path)
}

// 阅读器/读后感书写页：沉浸全屏，隐藏站头
const isReader = computed(() => route.path.startsWith('/reader') || route.path.startsWith('/reviews/editor'))

watch(
  () => route.path,
  (path) => {
    if (path.startsWith('/reader') || path.startsWith('/reviews/editor')) pausePlay()
    if (!path.startsWith('/login')) user.value = getUser()
    showBackTop.value = false
  }
)

// 全部分类（移动网页：汉堡抽屉列出，不占用底部空间）
const menuItems = [
  { path: '/bookshelf', label: '书架', icon: 'Reading' },
  { path: '/audiobooks', label: '有声书', icon: 'Headset' },
  { path: '/collections', label: '收藏夹', icon: 'Star' },
  { path: '/categories', label: '分类管理', icon: 'FolderOpened' },
  { path: '/libraries', label: '媒体库', icon: 'Collection' },
  { path: '/stats', label: '阅读统计', icon: 'TrendCharts' },
  { path: '/reviews', label: '读后感', icon: 'Notebook' },
  { path: '/settings', label: '设置', icon: 'Setting' }
]

const isNavActive = (item) => {
  if (route.path.startsWith('/book/')) return item.path === '/bookshelf'
  if (route.path.startsWith('/library/')) return item.path === '/libraries'
  return route.path === item.path || route.path.startsWith(item.path + '/')
}

// 回到顶部
const scrollRef = ref(null)
const showBackTop = ref(false)
const onScroll = () => {
  const el = scrollRef.value
  showBackTop.value = !!el && el.scrollTop > 400
}
const scrollToTop = () => {
  const el = scrollRef.value
  if (el) el.scrollTo({ top: 0, behavior: 'smooth' })
}
</script>

<style scoped>
.m-app {
  height: 100vh;
  height: 100dvh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: #f5f7fa;
}

/* ===== 顶部站头 ===== */
.m-header {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: calc(54px + env(safe-area-inset-top));
  padding: env(safe-area-inset-top) 14px 0;
  background: #fff;
  border-bottom: 1px solid #eceff3;
  box-shadow: 0 1px 6px rgba(15, 23, 42, 0.04);
  user-select: none;
  -webkit-tap-highlight-color: transparent;
  position: relative;
  z-index: 20;
}

.m-header-left {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.m-menu-btn {
  flex-shrink: 0;
  width: 38px;
  height: 38px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #475569;
  cursor: pointer;
  transition: background 0.2s ease;
}

.m-menu-btn:active {
  background: #f1f5f9;
}

.m-brand {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.m-brand-icon {
  flex-shrink: 0;
  width: 30px;
  height: 30px;
  border-radius: 9px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  background: var(--app-gradient);
  overflow: hidden;
}

.m-brand-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.m-brand-name {
  font-size: 17px;
  font-weight: 700;
  color: #0f172a;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.m-header-avatar {
  flex-shrink: 0;
  width: 34px;
  height: 34px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  font-weight: 600;
  color: #fff;
  background: var(--app-gradient);
  box-shadow: 0 4px 12px color-mix(in srgb, var(--app-primary) 35%, transparent);
  cursor: pointer;
  transition: transform 0.2s ease;
}

.m-header-avatar:active {
  transform: scale(0.92);
}

/* ===== 主内容 ===== */
.m-main {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  -webkit-overflow-scrolling: touch;
  padding: 14px 14px 20px;
}

.m-main.is-reader {
  padding: 0;
  overflow: hidden;
}

/* ===== 抽屉遮罩 ===== */
.m-mask {
  position: fixed;
  inset: 0;
  z-index: 30;
  background: rgba(15, 23, 42, 0.45);
}

/* ===== 左侧抽屉 ===== */
.m-drawer {
  position: fixed;
  top: 0;
  left: 0;
  bottom: 0;
  z-index: 31;
  width: 80%;
  max-width: 320px;
  background: #fff;
  display: flex;
  flex-direction: column;
  box-shadow: 8px 0 30px rgba(15, 23, 42, 0.18);
  padding-top: env(safe-area-inset-top);
}

.m-drawer-head {
  padding: 18px 18px 14px;
  border-bottom: 1px solid #f1f5f9;
}

.m-drawer-brand {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 18px;
  font-weight: 700;
  color: #0f172a;
}

/* 站点名 + 版本号（版本号为名称下方小字） */
.m-drawer-brand-text {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
  overflow: hidden;
}
.m-drawer-brand-name {
  font-size: 18px;
  font-weight: 700;
  line-height: 1.2;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.m-drawer-brand-version {
  font-size: 11px;
  font-weight: 400;
  line-height: 1.2;
  color: #94a3b8;
  letter-spacing: 0.2px;
}

.m-drawer-brand-icon {
  width: 34px;
  height: 34px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  background: var(--app-gradient);
  overflow: hidden;
}

.m-drawer-user {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 18px;
}

.m-drawer-avatar {
  flex-shrink: 0;
  width: 44px;
  height: 44px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 17px;
  font-weight: 600;
  color: #fff;
  background: var(--app-gradient);
}

.m-drawer-user-info {
  min-width: 0;
}

.m-drawer-name {
  font-size: 15px;
  font-weight: 600;
  color: #1e293b;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.m-drawer-sub {
  margin-top: 2px;
  font-size: 12px;
  color: #94a3b8;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.m-menu {
  flex: 1;
  overflow-y: auto;
  padding: 6px 12px;
}

.m-menu-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 13px 14px;
  border-radius: 12px;
  color: #475569;
  font-size: 15px;
  cursor: pointer;
  transition: background 0.2s ease, color 0.2s ease;
  -webkit-tap-highlight-color: transparent;
}

.m-menu-item:active {
  background: #f1f5f9;
}

.m-menu-item.active {
  background: color-mix(in srgb, var(--app-primary) 12%, transparent);
  color: var(--app-primary);
  font-weight: 600;
}

.m-drawer-foot {
  padding: 10px 12px calc(16px + env(safe-area-inset-bottom));
  border-top: 1px solid #f1f5f9;
}

.m-logout {
  color: #f43f5e;
}

.m-logout:active {
  background: rgba(244, 63, 94, 0.08);
}

/* ===== 回到顶部 ===== */
.m-backtop {
  position: fixed;
  right: 16px;
  bottom: calc(20px + env(safe-area-inset-bottom));
  z-index: 25;
  width: 42px;
  height: 42px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  background: var(--app-gradient);
  box-shadow: 0 8px 24px color-mix(in srgb, var(--app-primary) 40%, transparent);
  cursor: pointer;
  transition: transform 0.25s cubic-bezier(0.34, 1.56, 0.64, 1);
  -webkit-tap-highlight-color: transparent;
}

.m-backtop:active {
  transform: scale(0.9);
}

/* ===== 动效 ===== */
.m-mask-fade-enter-active,
.m-mask-fade-leave-active {
  transition: opacity 0.28s ease;
}

.m-mask-fade-enter-from,
.m-mask-fade-leave-to {
  opacity: 0;
}

.m-drawer-slide-enter-active,
.m-drawer-slide-leave-active {
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.m-drawer-slide-enter-from,
.m-drawer-slide-leave-to {
  transform: translateX(-100%);
}

.m-backtop-fade-enter-active,
.m-backtop-fade-leave-active {
  transition: opacity 0.3s ease, transform 0.3s ease;
}

.m-backtop-fade-enter-from,
.m-backtop-fade-leave-to {
  opacity: 0;
  transform: translateY(12px);
}
</style>
