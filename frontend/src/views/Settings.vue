<template>
  <div class="settings-page">
    <div class="bg-glow bg-glow-1"></div>
    <div class="bg-glow bg-glow-2"></div>

    <div class="settings-container">

      <div class="settings-body">
        <!-- 分类导航（桌面端侧栏 / 移动端横排） -->
        <nav class="settings-nav cardIn" style="--i: 0">
          <div
            v-for="s in sections"
            :key="s.key"
            class="nav-item"
            :class="{ active: activeTab === s.key }"
            @click="switchTab(s.key)"
          >
            <el-icon :size="18"><component :is="s.icon" /></el-icon>
            <div class="nav-text">
              <span class="nav-label">{{ s.label }}</span>
              <span class="nav-desc">{{ s.desc }}</span>
            </div>
          </div>
          <div class="nav-footer">更多设置即将上线</div>
        </nav>

        <!-- 内容区 -->
        <div class="settings-content cardIn" style="--i: 1">
          <transition name="fade" mode="out-in">
            <component :is="activeComp" :key="activeTab" />
          </transition>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, markRaw, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import ReadingPrefs from './settings/ReadingPrefs.vue'
import AccountSettings from './settings/AccountSettings.vue'
import Appearance from './settings/Appearance.vue'
import Feedback from './settings/Feedback.vue'

// ===== 设置分类注册表：后续新增设置功能时，只需在此追加一项 =====
const sections = [
  {
    key: 'appearance',
    label: '外观设置',
    desc: '主题色 · 名称 · 图标',
    icon: 'Brush',
    component: markRaw(Appearance)
  },
  {
    key: 'reading',
    label: '阅读偏好',
    desc: '主题 · 字体 · 排版',
    icon: 'Reading',
    component: markRaw(ReadingPrefs)
  },
  {
    key: 'account',
    label: '账号安全',
    desc: '昵称 · 密码 · 登录',
    icon: 'User',
    component: markRaw(AccountSettings)
  },
  {
    key: 'feedback',
    label: '问题反馈',
    desc: 'QQ · 邮件反馈',
    icon: 'ChatDotRound',
    component: markRaw(Feedback)
  }
]

const route = useRoute()
const router = useRouter()

const activeTab = ref('appearance')

const activeComp = computed(() => {
  const s = sections.find((x) => x.key === activeTab.value)
  return s ? s.component : sections[0].component
})

const switchTab = (key) => {
  if (activeTab.value === key) return
  router.replace({ path: '/settings', query: { tab: key } })
}

// 支持 ?tab=xxx 深链接定位到对应设置分类
const syncFromRoute = () => {
  const t = route.query.tab
  if (t && sections.some((s) => s.key === t)) {
    activeTab.value = t
  }
}

watch(() => route.query.tab, syncFromRoute)
onMounted(syncFromRoute)
</script>

<style scoped>
.settings-page {
  position: relative;
  min-height: calc(100vh - 120px);
  /* 与首页 .bookshelf 完全一致：1400px 居中，无左右内边距，内容铺满 */
  max-width: 1400px;
  margin: 0 auto;
  padding: 0px 0 60px;
}

.bg-glow {
  position: fixed;
  border-radius: 50%;
  filter: blur(90px);
  pointer-events: none;
  z-index: 0;
}
.bg-glow-1 {
  width: 460px;
  height: 460px;
  background: radial-gradient(circle, color-mix(in srgb, var(--app-primary) 16%, transparent), transparent 70%);
  top: -100px;
  left: -100px;
  animation: float 12s ease-in-out infinite;
}
.bg-glow-2 {
  width: 400px;
  height: 400px;
  background: radial-gradient(circle, rgba(34, 211, 238, 0.12), transparent 70%);
  bottom: -80px;
  right: -80px;
  animation: float 14s ease-in-out infinite reverse;
}
@keyframes float {
  0%, 100% { transform: translate(0, 0); }
  50% { transform: translate(30px, -20px); }
}

.settings-container {
  position: relative;
  z-index: 1;
  width: 100%;
}

.page-header {
  margin-bottom: 24px;
}
.header-left h1 {
  margin: 0 0 6px;
  font-size: 26px;
  font-weight: 800;
  color: #0f172a;
  display: flex;
  align-items: center;
  gap: 10px;
}
.header-left h1 .el-icon { color: var(--app-primary); }
.header-left p {
  margin: 0;
  font-size: 14px;
  color: #64748b;
}

/* 布局：与首页一致的全宽单栏结构 */
.settings-body {
  display: flex;
  flex-direction: column;
  gap: 20px;
  align-items: stretch;
}

/* 分类导航：横向 tab 条（不再占用侧边宽度） */
.settings-nav {
  display: flex;
  align-items: stretch;
  gap: 8px;
  padding: 10px;
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.75);
  box-shadow: 0 8px 28px rgba(31, 38, 135, 0.08);
  overflow-x: auto;
  scrollbar-width: none;
}
.settings-nav::-webkit-scrollbar {
  display: none;
}
.nav-item {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 20px;
  border-radius: 12px;
  cursor: pointer;
  color: #475569;
  transition: all 0.2s;
  user-select: none;
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
.nav-text {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.nav-label {
  font-size: 14px;
  font-weight: 600;
}
.nav-desc {
  font-size: 11px;
  opacity: 0.75;
  margin-top: 2px;
  white-space: nowrap;
}
.nav-footer {
  display: none;
}

/* 内容区：不套大卡片，子组件自带卡片背景，与首页一样直接铺满全宽 */
.settings-content {
  min-width: 0;
}

/* 各设置子组件统一卡片样式（原 settings-content 大白卡片样式迁移至此） */
.settings-content :deep(.settings-section) {
  padding: 28px 32px;
  border-radius: 20px;
  background: rgba(255, 255, 255, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.75);
  box-shadow: 0 8px 28px rgba(31, 38, 135, 0.08);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.fade-enter-from {
  opacity: 0;
  transform: translateY(8px);
}
.fade-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

/* ===== 入场动画 ===== */
.cardIn {
  animation: cardIn 0.55s cubic-bezier(0.22, 1, 0.36, 1) both;
  animation-delay: calc(var(--i, 0) * 90ms);
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

/* 移动端 */
@media (max-width: 768px) {
  .settings-page {
    padding: 0px 0 40px;
  }
  .settings-body {
    flex-direction: column;
  }
  .settings-nav {
    overflow-x: auto;
    padding: 8px;
    gap: 4px;
  }
  .nav-item {
    flex-shrink: 0;
    flex-direction: column;
    gap: 4px;
    padding: 10px 14px;
    text-align: center;
  }
  .nav-text {
    align-items: center;
  }
  .nav-desc {
    display: none;
  }
  .settings-content :deep(.settings-section) {
    padding: 20px 16px;
  }
}
</style>
