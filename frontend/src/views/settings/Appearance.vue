<template>
  <div class="settings-section appearance-settings">
    <div class="section-title">
      <el-icon><Brush /></el-icon>
      <div>
        <h3>外观设置</h3>
        <p>自定义全局主题色、网站名称与图标，切换页面动效与菜单透光效果</p>
      </div>
    </div>

    <!-- 主题色 -->
    <div class="pref-group">
      <div class="pref-label">
        <span>主题色</span>
        <span class="pref-value">{{ prefs.themeColor }}</span>
      </div>
      <div class="color-row">
        <div
          v-for="c in APP_COLOR_PRESETS"
          :key="c"
          class="color-swatch"
          :class="{ active: prefs.themeColor.toLowerCase() === c }"
          :style="{ background: c }"
          @click="setPref('themeColor', c)"
        >
          <el-icon v-if="prefs.themeColor.toLowerCase() === c" :size="14" class="swatch-check"><Check /></el-icon>
        </div>
        <el-color-picker
          :model-value="prefs.themeColor"
          :predefine="APP_COLOR_PRESETS"
          @update:model-value="setPref('themeColor', $event)"
        />
        <span class="color-tip">点击色块或取色器自定义主题色，全站即时生效</span>
      </div>
    </div>

    <!-- 网站名称 -->
    <div class="pref-group">
      <div class="pref-label"><span>网站名称</span></div>
      <el-input
        v-model="siteNameDraft"
        style="width: 100%"
        maxlength="12"
        show-word-limit
        placeholder="请输入网站名称"
        @blur="saveSiteName"
        @keyup.enter="$event.target.blur()"
      >
        <template #prepend>名称</template>
      </el-input>
      <p class="setting-tip">将显示在侧边栏 Logo、浏览器标签页标题与收藏夹图标；输入完成后失焦或回车即自动保存</p>
    </div>

    <!-- 网站图标 -->
    <div class="pref-group">
      <div class="pref-label"><span>网站图标</span></div>
      <div class="icon-options">
        <div
          v-for="ic in APP_ICON_OPTIONS"
          :key="ic"
          class="icon-option"
          :class="{ active: prefs.siteIcon === ic }"
          @click="setPref('siteIcon', ic)"
        >
          <el-icon :size="20"><component :is="ic" /></el-icon>
        </div>
        <div class="icon-option icon-custom" :class="{ active: isCustom }" @click="toggleCustomIcon">
          <el-icon :size="20"><Link /></el-icon>
        </div>
      </div>
      <div v-if="isCustom" class="custom-icon-row">
        <el-input
          :model-value="customIconUrl"
          style="width: 100%"
          placeholder="粘贴图片 URL（http/https）作为网站图标"
          @update:model-value="onCustomIconInput"
        />
      </div>
      <p class="setting-tip">浏览器标签页与收藏夹图标将同步更新</p>
    </div>

    <!-- 页面切换动效 -->
    <div class="pref-group switch-row">
      <div class="pref-label">
        <span>页面切换动效</span>
        <p class="pref-desc">路由切换时的淡入上移动画</p>
      </div>
      <el-switch :model-value="prefs.pageTransition" @update:model-value="setPref('pageTransition', $event)" />
    </div>

    <!-- 菜单透光效果 -->
    <div class="pref-group switch-row">
      <div class="pref-label">
        <span>菜单透光效果</span>
        <p class="pref-desc">侧边栏菜单悬停/选中时的柔光渐变</p>
      </div>
      <el-switch :model-value="prefs.menuGlow" @update:model-value="setPref('menuGlow', $event)" />
    </div>

    <div class="pref-actions">
      <el-button :icon="RefreshLeft" @click="resetAll">恢复默认外观</el-button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { RefreshLeft } from '@element-plus/icons-vue'
import {
  loadAppPrefs, saveAppPrefs, resetAppPrefs, applyAppPrefs,
  APP_ICON_OPTIONS, APP_COLOR_PRESETS, isCustomIcon
} from '@/utils/appPrefs'

const prefs = ref(loadAppPrefs())

const isCustom = computed(() => isCustomIcon(prefs.value.siteIcon))

// 网站名称使用独立草稿，输入过程不保存、可自由清空重输，失焦或回车时才保存
const siteNameDraft = ref(prefs.value.siteName)

const saveSiteName = () => {
  const val = (siteNameDraft.value || '').trim()
  if (!val) {
    siteNameDraft.value = prefs.value.siteName
    ElMessage.warning('网站名称不能为空')
    return
  }
  if (val === prefs.value.siteName) return
  prefs.value = { ...prefs.value, siteName: val }
  saveAppPrefs(prefs.value)
  applyAppPrefs(prefs.value)
  ElMessage.success('网站名称已保存')
}
const customIconUrl = ref(isCustom.value ? prefs.value.siteIcon : '')

const setPref = (key, value) => {
  if (value === undefined || value === null || value === '') return
  prefs.value = { ...prefs.value, [key]: value }
  saveAppPrefs(prefs.value)
  applyAppPrefs(prefs.value)
  ElMessage.success('外观已更新')
}

const toggleCustomIcon = () => {
  if (isCustom.value) {
    prefs.value = { ...prefs.value, siteIcon: APP_ICON_OPTIONS[0] }
    saveAppPrefs(prefs.value)
    applyAppPrefs(prefs.value)
  } else {
    customIconUrl.value = ''
  }
}

const onCustomIconInput = (url) => {
  customIconUrl.value = url
  if (url && /^https?:\/\//.test(url)) {
    prefs.value = { ...prefs.value, siteIcon: url }
    saveAppPrefs(prefs.value)
    applyAppPrefs(prefs.value)
  }
}

const resetAll = () => {
  prefs.value = resetAppPrefs()
  saveAppPrefs(prefs.value)
  applyAppPrefs(prefs.value)
  siteNameDraft.value = prefs.value.siteName
  customIconUrl.value = ''
  ElMessage.success('已恢复默认外观')
}
</script>

<style scoped>
.section-title {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 24px;
}
.section-title > .el-icon {
  font-size: 26px;
  color: var(--el-color-primary);
  background: color-mix(in srgb, var(--app-primary) 12%, transparent);
  padding: 10px;
  border-radius: 12px;
}
.section-title h3 {
  margin: 0;
  font-size: 18px;
}
.section-title p {
  margin: 4px 0 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.pref-group {
  margin-bottom: 26px;
}
.pref-label {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
  font-size: 14px;
  font-weight: 500;
}
.pref-value {
  font-size: 12px;
  color: var(--el-color-primary);
  font-variant-numeric: tabular-nums;
}
.pref-desc {
  margin: 2px 0 0;
  font-size: 12px;
  font-weight: 400;
  color: var(--el-text-color-secondary);
}
.setting-tip {
  margin: 8px 0 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

/* 主题色 */
.color-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}
.color-swatch {
  width: 30px;
  height: 30px;
  border-radius: 8px;
  cursor: pointer;
  border: 2px solid transparent;
  box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.08);
  transition: transform 0.15s, box-shadow 0.15s;
  display: flex;
  align-items: center;
  justify-content: center;
}
.color-swatch:hover {
  transform: scale(1.12);
}
.color-swatch.active {
  border-color: #fff;
  box-shadow: 0 0 0 2px var(--app-primary), inset 0 0 0 1px rgba(0, 0, 0, 0.08);
}
.swatch-check {
  color: #fff;
  filter: drop-shadow(0 1px 1px rgba(0, 0, 0, 0.35));
}
.color-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

/* 网站图标 */
.icon-options {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.icon-option {
  width: 44px;
  height: 44px;
  border-radius: 10px;
  border: 2px solid var(--el-border-color-lighter);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #64748b;
  cursor: pointer;
  transition: all 0.18s;
}
.icon-option:hover {
  color: var(--el-color-primary);
  border-color: var(--el-color-primary-light-5);
}
.icon-option.active {
  color: var(--el-color-primary);
  border-color: var(--el-color-primary);
  background: color-mix(in srgb, var(--app-primary) 8%, transparent);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--app-primary) 15%, transparent);
}
.custom-icon-row {
  margin-top: 10px;
}

/* 开关行 */
.switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.switch-row .pref-label {
  margin-bottom: 0;
  display: block;
}

.pref-actions {
  margin-top: 8px;
  padding-top: 16px;
  border-top: 1px dashed var(--el-border-color-lighter);
}
</style>
