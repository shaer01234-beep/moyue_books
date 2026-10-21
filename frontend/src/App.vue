<template>
  <div class="app-container">
    <!-- 首次使用协议门禁：同意后放行（分享预览页对外浏览豁免） -->
    <AgreementGate v-if="showAgreementGate" @agreed="onAgreementAgreed" />

    <!-- 登录 / 分享预览等独立页面：全屏渲染，不套应用框架 -->
    <router-view v-if="isPublicPage" />

    <!-- 手机端：H5 版外壳 -->
    <MobileLayout v-else-if="isMobile" />

    <!-- 桌面端：PC 版侧边栏布局 -->
    <DesktopLayout v-else />
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import AgreementGate from '@/components/AgreementGate.vue'
import MobileLayout from '@/components/MobileLayout.vue'
import DesktopLayout from '@/components/DesktopLayout.vue'
import { hasAgreedAgreement } from '@/utils/agreement'
import { useIsMobile } from '@/utils/useDevice'

const route = useRoute()
const isMobile = useIsMobile()

// 首次使用协议门禁
const agreementAgreed = ref(hasAgreedAgreement())
const isSharePreview = computed(() => route.path.startsWith('/share/'))
const showAgreementGate = computed(() => !agreementAgreed.value && !isSharePreview.value)
const onAgreementAgreed = () => {
  agreementAgreed.value = true
}

// 登录、分享预览等独立页面：不套应用框架，全屏渲染
const isPublicPage = computed(() => route.path === '/login' || route.path.startsWith('/share/'))
</script>

<style scoped>
.app-container {
  height: 100vh;
  overflow: hidden;
}
</style>

<!-- 全局样式：移动端 Element Plus 组件适配 + 页面动效 -->
<style>
html, body {
  margin: 0;
  padding: 0;
  height: 100%;
  font-family: 'Helvetica Neue', Helvetica, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', Arial, sans-serif;
}

/* ===== 页面切换动效（设置中可关闭） ===== */
.page-fade-enter-active,
.page-fade-leave-active {
  transition: opacity 0.25s ease, transform 0.25s ease;
}

.page-fade-enter-from {
  opacity: 0;
  transform: translateY(10px);
}

.page-fade-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}

html[data-page-transition="off"] .page-fade-enter-active,
html[data-page-transition="off"] .page-fade-leave-active {
  transition: none;
}

html[data-page-transition="off"] .page-fade-enter-from,
html[data-page-transition="off"] .page-fade-leave-to {
  opacity: 1;
  transform: none;
}

/* 阅读器页面不做页面切换动画，避免全屏阅读时闪烁 */
.is-reader .page-fade-enter-active,
.is-reader .page-fade-leave-active {
  transition: none;
}

.is-reader .page-fade-enter-from,
.is-reader .page-fade-leave-to {
  opacity: 1;
  transform: none;
}

@media (max-width: 768px) {
  .el-dialog {
    width: 92% !important;
    max-width: 92% !important;
  }

  .el-message-box {
    width: 88% !important;
    max-width: 88% !important;
  }

  .el-drawer {
    width: 100% !important;
    max-width: 100% !important;
  }

  .el-drawer__body {
    padding: 0;
  }

  .el-dialog .el-form-item {
    display: block;
    margin-bottom: 14px;
  }

  .el-dialog .el-form-item__label {
    display: block;
    width: auto !important;
    text-align: left;
    padding: 0 0 6px;
    line-height: 1.4;
  }

  .el-dialog .el-form-item__content {
    margin-left: 0 !important;
  }

  .el-dialog .el-date-editor {
    width: 100% !important;
  }
}

/* ===== 用户下拉弹框（挂载在 body，需全局样式） ===== */
.user-popper {
  background: rgba(255, 255, 255, 0.92);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border: 1px solid rgba(255, 255, 255, 0.9);
  border-radius: 14px;
  box-shadow: 0 18px 48px rgba(31, 38, 135, 0.18);
  padding: 8px;
  margin-top: 8px;
}

.user-popper .el-dropdown-menu {
  background: transparent;
  padding: 0;
  box-shadow: none;
}

.user-popper-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px 12px;
  border-bottom: 1px solid rgba(99, 102, 241, 0.12);
}

.user-popper-avatar {
  flex-shrink: 0;
  width: 40px;
  height: 40px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  font-weight: 600;
  color: #fff;
  background: var(--app-gradient);
  box-shadow: 0 6px 16px color-mix(in srgb, var(--app-primary) 35%, transparent);
}

.user-popper-info {
  min-width: 0;
}

.user-popper-name {
  font-size: 14px;
  font-weight: 600;
  color: #1e293b;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 160px;
}

.user-popper-sub {
  margin-top: 2px;
  font-size: 12px;
  color: #94a3b8;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 160px;
}

.user-popper .el-dropdown-menu__item {
  border-radius: 9px;
  height: 42px;
  line-height: 42px;
  padding: 0 12px;
  margin-top: 6px;
  color: #f43f5e;
  font-size: 13px;
}

.user-popper .el-dropdown-menu__item:hover,
.user-popper .el-dropdown-menu__item:focus {
  background: rgba(244, 63, 94, 0.08);
  color: #f43f5e;
}
</style>
