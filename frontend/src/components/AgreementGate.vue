<template>
  <div class="agreement-gate">
    <div class="agreement-card">
      <div class="agreement-head">
        <div class="agreement-logo">
          <img v-if="isCustomLogo" :src="logoIcon" class="logo-img" alt="" />
          <el-icon v-else :size="24"><component :is="logoIcon" /></el-icon>
        </div>
        <h2>{{ appName }} · 用户使用协议</h2>
        <p class="sub">更新日期：{{ AGREEMENT_META.updatedAt }}</p>
      </div>

      <div ref="bodyRef" class="agreement-body">
        <div class="intro">
          欢迎使用{{ appName }}。在使用本应用前，请你仔细阅读并理解以下协议条款。请你确认已阅读、理解并同意全部内容后再继续使用。
        </div>
        <section v-for="sec in AGREEMENT_SECTIONS" :key="sec.title" class="sec">
          <h3>{{ sec.title }}</h3>
          <p v-if="sec.content">{{ sec.content }}</p>
          <ul v-if="sec.items">
            <li v-for="(item, i) in sec.items" :key="i">{{ item }}</li>
          </ul>
        </section>
      </div>

      <div class="agreement-foot">
        <el-checkbox v-model="accepted">我已阅读并同意以上《{{ appName }}用户使用协议》全部条款</el-checkbox>
        <div class="agree-actions">
          <el-button :disabled="!accepted" type="primary" :icon="Check" @click="agree">同意并开始使用</el-button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { Check } from '@element-plus/icons-vue'
import { loadAppPrefs, isCustomIcon } from '@/utils/appPrefs'
import {
  AGREEMENT_META,
  AGREEMENT_SECTIONS,
  hasAgreedAgreement,
  markAgreementAgreed
} from '@/utils/agreement'

const emit = defineEmits(['agreed'])

const appPrefs = loadAppPrefs()
const appName = computed(() => appPrefs.siteName || AGREEMENT_META.appName)
const logoIcon = computed(() =>
  isCustomIcon(appPrefs.siteIcon) ? appPrefs.siteIcon : appPrefs.siteIcon || 'Collection'
)
const isCustomLogo = computed(() => isCustomIcon(appPrefs.siteIcon))

const accepted = ref(false)
const bodyRef = ref(null)

// 启动时兜底：若已同意过则直接放行（正常由父组件控制 v-if，此处防竞态）
if (hasAgreedAgreement()) emit('agreed')

const agree = () => {
  if (!accepted.value) return
  markAgreementAgreed()
  emit('agreed')
}
</script>

<style scoped>
.agreement-gate {
  position: fixed;
  inset: 0;
  z-index: 4000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background:
    radial-gradient(1200px 600px at 80% -10%, color-mix(in srgb, var(--app-primary) 22%, transparent), transparent 60%),
    radial-gradient(900px 500px at -10% 110%, color-mix(in srgb, var(--app-primary) 18%, transparent), transparent 55%),
    linear-gradient(135deg, #f6f8fb 0%, #eef4f0 45%, #eafaf1 100%);
}

.agreement-card {
  display: flex;
  flex-direction: column;
  width: min(760px, 100%);
  max-height: min(86vh, 860px);
  background: rgba(255, 255, 255, 0.92);
  backdrop-filter: blur(24px);
  -webkit-backdrop-filter: blur(24px);
  border: 1px solid rgba(255, 255, 255, 0.95);
  border-radius: 22px;
  box-shadow: 0 32px 80px rgba(16, 60, 40, 0.18);
  overflow: hidden;
}

.agreement-head {
  padding: 26px 30px 18px;
  text-align: center;
  border-bottom: 1px solid color-mix(in srgb, var(--app-primary) 12%, transparent);
  background: linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 8%, transparent), transparent 60%);
}
.agreement-logo {
  width: 56px;
  height: 56px;
  margin: 0 auto 12px;
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  background: var(--app-gradient);
  box-shadow: 0 10px 26px color-mix(in srgb, var(--app-primary) 40%, transparent);
  overflow: hidden;
}
.logo-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.agreement-head h2 {
  margin: 0;
  font-size: 20px;
  color: #0f172a;
}
.agreement-head .sub {
  margin: 6px 0 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.agreement-body {
  flex: 1;
  overflow-y: auto;
  padding: 20px 30px;
  line-height: 1.8;
}
.agreement-body .intro {
  font-size: 13.5px;
  color: var(--el-text-color-regular);
  background: color-mix(in srgb, var(--app-primary) 7%, transparent);
  border-left: 3px solid var(--app-primary);
  border-radius: 8px;
  padding: 10px 14px;
  margin-bottom: 6px;
}
.sec {
  margin-top: 16px;
}
.sec h3 {
  margin: 0 0 6px;
  font-size: 15px;
  color: #1e293b;
}
.sec p {
  margin: 0;
  font-size: 13.5px;
  color: var(--el-text-color-regular);
  text-align: justify;
}
.sec ul {
  margin: 6px 0 0;
  padding-left: 20px;
}
.sec li {
  font-size: 13.5px;
  color: var(--el-text-color-regular);
  text-align: justify;
  margin-bottom: 4px;
}

.agreement-foot {
  padding: 16px 30px 22px;
  border-top: 1px solid color-mix(in srgb, var(--app-primary) 12%, transparent);
  background: rgba(255, 255, 255, 0.6);
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.agree-actions {
  display: flex;
  justify-content: center;
}
.agree-actions .el-button {
  min-width: 200px;
  height: 42px;
  font-size: 15px;
  border-radius: 12px;
}

@media (max-width: 768px) {
  .agreement-gate {
    padding: 0;
    align-items: stretch;
  }
  .agreement-card {
    width: 100%;
    max-height: 100vh;
    height: 100vh;
    border-radius: 0;
    border: none;
  }
  .agreement-head {
    padding: 18px 18px 12px;
  }
  .agreement-logo {
    width: 46px;
    height: 46px;
  }
  .agreement-body {
    padding: 16px 18px;
  }
  .agreement-foot {
    padding: 12px 16px 20px;
    padding-bottom: calc(16px + env(safe-area-inset-bottom));
  }
}
</style>
