<template>
  <div class="settings-section reading-prefs">
    <div class="section-title">
      <el-icon><Reading /></el-icon>
      <div>
        <h3>阅读偏好</h3>
        <p>调整阅读器的主题、字体与排版，修改后立即生效并自动保存</p>
      </div>
    </div>

    <!-- 主题 -->
    <div class="pref-group">
      <div class="pref-label">
        <span>阅读主题</span>
      </div>
      <div class="theme-cards">
        <div
          v-for="t in themeList"
          :key="t.key"
          class="theme-card"
          :class="{ active: prefs.theme === t.key }"
          @click="setPref('theme', t.key)"
        >
          <div class="theme-swatch" :style="t.swatchStyle">
            <div class="theme-line" :style="{ background: t.lineColor }"></div>
            <div class="theme-line short" :style="{ background: t.lineColor }"></div>
            <div class="theme-line" :style="{ background: t.lineColor }"></div>
          </div>
          <div class="theme-meta">
            <span class="theme-name">{{ t.label }}</span>
            <el-icon v-if="prefs.theme === t.key" class="theme-check"><Check /></el-icon>
          </div>
        </div>
      </div>
    </div>

    <!-- 字体大小 -->
    <div class="pref-group">
      <div class="pref-label">
        <span>字体大小</span>
        <span class="pref-value">{{ prefs.fontSize }}px</span>
      </div>
      <el-slider
        :model-value="prefs.fontSize"
        :min="12"
        :max="28"
        :step="1"
        show-stops
        @update:model-value="setPref('fontSize', $event)"
      />
    </div>

    <!-- 行间距 -->
    <div class="pref-group">
      <div class="pref-label">
        <span>行间距</span>
        <span class="pref-value">{{ prefs.lineHeight.toFixed(1) }}</span>
      </div>
      <el-slider
        :model-value="prefs.lineHeight"
        :min="1.4"
        :max="2.6"
        :step="0.1"
        @update:model-value="setPref('lineHeight', $event)"
      />
    </div>

    <!-- 页面边距 -->
    <div class="pref-group">
      <div class="pref-label">
        <span>页面边距</span>
        <span class="pref-value">{{ prefs.pageMargin }}px</span>
      </div>
      <el-slider
        :model-value="prefs.pageMargin"
        :min="0"
        :max="80"
        :step="2"
        show-stops
        @update:model-value="setPref('pageMargin', $event)"
      />
    </div>

    <!-- 字体 -->
    <div class="pref-group">
      <div class="pref-label">
        <span>正文字体</span>
      </div>
      <el-select
        :model-value="prefs.fontFamily"
        style="width: 100%"
        @update:model-value="setPref('fontFamily', $event)"
      >
        <el-option
          v-for="f in fontOptions"
          :key="f.value"
          :label="f.label"
          :value="f.value"
        >
          <span :style="{ fontFamily: f.value }">{{ f.label }}</span>
        </el-option>
      </el-select>
    </div>

    <div class="pref-actions">
      <el-button :icon="RefreshLeft" @click="resetAll">恢复默认设置</el-button>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { RefreshLeft } from '@element-plus/icons-vue'
import { loadReaderPrefs, saveReaderPrefs, resetReaderPrefs } from '@/utils/readerPrefs'

const prefs = ref(loadReaderPrefs())

const themeList = [
  {
    key: 'light',
    label: '日间',
    swatchStyle: 'background: #ffffff;',
    lineColor: '#3a3f4a'
  },
  {
    key: 'sepia',
    label: '护眼',
    swatchStyle: 'background: #f7f1e3;',
    lineColor: '#6b5d43'
  },
  {
    key: 'dark',
    label: '夜间',
    swatchStyle: 'background: #26282e;',
    lineColor: '#9aa0ab'
  },
  {
    key: 'pitch',
    label: '纯黑',
    swatchStyle: 'background: #000000;',
    lineColor: '#7c8494'
  }
]

const fontOptions = [
  { label: '系统默认', value: 'system-ui, -apple-system, "PingFang SC", "Microsoft YaHei", sans-serif' },
  { label: '宋体', value: '"Songti SC", "SimSun", serif' },
  { label: '黑体', value: '"Heiti SC", "SimHei", sans-serif' },
  { label: '楷体', value: '"Kaiti SC", "KaiTi", serif' },
  { label: '仿宋', value: '"FangSong", "FangSong SC", serif' },
  { label: '等线', value: '"DengXian", "Microsoft YaHei", sans-serif' }
]

const setPref = (key, value) => {
  prefs.value = { ...prefs.value, [key]: value }
  saveReaderPrefs(prefs.value)
}

const resetAll = () => {
  prefs.value = resetReaderPrefs()
  saveReaderPrefs(prefs.value)
  ElMessage.success('已恢复默认阅读设置')
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
  font-size: 13px;
  color: var(--el-color-primary);
  font-variant-numeric: tabular-nums;
}

/* 主题卡片 */
.theme-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
}
.theme-card {
  cursor: pointer;
  border: 2px solid var(--el-border-color-lighter);
  border-radius: 12px;
  overflow: hidden;
  transition: all 0.2s;
}
.theme-card:hover {
  border-color: var(--el-color-primary-light-5);
}
.theme-card.active {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--app-primary) 15%, transparent);
}
.theme-swatch {
  height: 64px;
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 5px;
}
.theme-line {
  height: 3px;
  border-radius: 2px;
  width: 100%;
  opacity: 0.55;
}
.theme-line.short {
  width: 65%;
}
.theme-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 10px;
  font-size: 13px;
  background: var(--el-fill-color-blank);
}
.theme-check {
  color: var(--el-color-primary);
  font-size: 15px;
}

.pref-actions {
  margin-top: 8px;
  padding-top: 16px;
  border-top: 1px dashed var(--el-border-color-lighter);
}

@media (max-width: 768px) {
  .theme-cards {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
