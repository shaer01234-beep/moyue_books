<template>
  <div class="reader-page" :class="{ embedded }">
    <!-- 阅读器头部 -->
    <div class="reader-header" v-show="!fullscreen">
      <div class="header-left">
        <el-button text @click="goBack">
          <el-icon><ArrowLeft /></el-icon> <span class="btn-text">返回</span>
        </el-button>
        <span class="book-title">{{ book?.title || '阅读器' }}</span>
      </div>
      <div class="header-right">
        <template v-if="book?.file_type?.toLowerCase() === 'pdf'">
          <el-button text @click="zoomOut">
            <el-icon><ZoomOut /></el-icon>
          </el-button>
          <span class="zoom-level">{{ Math.round(pdfScale * 100) }}%</span>
          <el-button text @click="zoomIn">
            <el-icon><ZoomIn /></el-icon>
          </el-button>
        </template>
        <el-button text @click="showAnnotations = true">
          <el-icon><EditPen /></el-icon>
          <span v-if="annotations.length > 0" class="ann-count">{{ annotations.length }}</span>
        </el-button>
        <el-button text @click="openReviews">
          <el-icon><Notebook /></el-icon>
          <span class="btn-text">读后感</span>
        </el-button>
        <el-button text @click="showToc = true">
          <el-icon><Menu /></el-icon> <span class="btn-text">目录</span>
        </el-button>
        <el-button text class="mobile-hide" @click="showSettings = true">
          <el-icon><Setting /></el-icon> <span class="btn-text">设置</span>
        </el-button>
        <el-button text class="mobile-hide" @click="toggleFullscreen">
          <el-icon><FullScreen /></el-icon>
          <span class="btn-text">{{ fullscreen ? '退出' : '全屏' }}</span>
        </el-button>
      </div>
      <!-- 阅读进度条：标题栏下边框，点击跳转进度，圆点为标记位置 -->
      <div class="reader-progress" @click="onProgressClick">
        <div class="progress-fill" :style="{ width: currentProgress + '%' }"></div>
        <div v-for="m in annotationMarkers" :key="m.id"
             class="progress-mark" :class="'mark-' + m.type"
             :style="{ left: m.pct + '%' }"
             @click.stop="onProgressMarkClick(m)"></div>
      </div>
    </div>

    <!-- 阅读区域（全屏模式下浏览器只渲染此子树，所有浮层必须放在内部） -->
    <div class="reader-body" :class="{ fullscreen: fullscreen, 'select-mode': selectMode }" :data-theme="currentTheme" :style="{ '--pdf-desktop-bg': pdfDesktopBg, '--pdf-paper-bg': pdfPaperBg }" ref="readerBody" @click="handleReaderClick" @wheel="handleWheel">
      <!-- 全屏悬浮信息栏 -->
      <div class="fullscreen-toolbar" v-if="fullscreen" @click.stop>
        <div class="fs-left">
          <el-button text size="small" @click="goBack">
            <el-icon><ArrowLeft /></el-icon>
          </el-button>
          <span class="fs-title">{{ book?.title || '' }}</span>
        </div>
        <div class="fs-center">
          <span>{{ currentPdfPage }} / {{ totalPdfPages }}</span>
        </div>
        <div class="fs-right">
          <el-badge :value="annotations.length" :hidden="annotations.length === 0" class="fs-badge">
            <el-button text size="small" @click="showAnnotations = !showAnnotations">
              <el-icon><EditPen /></el-icon> <span class="btn-text">标记</span>
            </el-button>
          </el-badge>
          <el-button text size="small" @click="openReviews">
            <el-icon><Notebook /></el-icon> <span class="btn-text">读后感</span>
          </el-button>
          <el-button text size="small" @click="toggleFullscreen">
            <el-icon><FullScreen /></el-icon> <span class="btn-text">退出</span>
          </el-button>
        </div>
      </div>

      <!-- PDF 工具栏（选中弹出划线/笔记）由外壳 FoliateReader 提供，此处不再保留独立工具栏 -->

      <!-- 翻页按钮区域（双页模式下禁用左右翻页，避免遮挡双页正文） -->
      <div class="page-zone page-prev" v-show="pdfLayout !== 'two'" @click="prevPage"></div>
      <div class="page-zone page-next" v-show="pdfLayout !== 'two'" @click="nextPage"></div>

      <!-- PDF 渲染区域 -->
      <div v-if="book?.file_type?.toLowerCase() === 'pdf'" class="pdf-container" ref="pdfContainer">
        <div v-if="pdfLoading && pageInfos.length === 0" class="pdf-loading">
          <el-icon class="is-loading" :size="48"><Loading /></el-icon>
          <p>正在加载 PDF...</p>
        </div>
        <div v-else class="pdf-scroll-viewport" :class="{ 'two-page': pdfLayout === 'two' }">
          <div v-for="(group, gi) in pdfPageGroups" :key="'g' + gi" class="pdf-page-group">
            <div
              v-for="page in group"
              :key="page.num"
              :id="'pdf-page-' + page.num"
              class="pdf-page-wrapper"
              :style="{ minHeight: page.height + 'px', width: page.width + 'px' }"
            >
              <template v-if="page.rendered">
                <canvas :ref="el => setPageCanvasRef(el, page.num)" class="pdf-canvas"></canvas>
                <div :ref="el => setPageTextLayerRef(el, page.num)" class="pdf-text-layer"></div>
                <div :ref="el => setPageOverlayRef(el, page.num)" class="pdf-highlights-overlay"></div>
              </template>
            </div>
          </div>
        </div>
      </div>


      <!-- PDF 页码指示器 -->
      <div v-if="book?.file_type?.toLowerCase() === 'pdf' && totalPdfPages > 0" class="pdf-page-indicator">
        {{ currentPdfPage }} / {{ totalPdfPages }}
      </div>

      <!-- 渲染中提示 -->
      <div v-if="book?.file_type?.toLowerCase() === 'pdf' && renderingCount > 0" class="rendering-hint">
        <el-icon class="is-loading"><Loading /></el-icon>
        <span>渲染中...</span>
      </div>

      <!-- 语音朗读控制条已迁移至工具栏"阅读"按钮的朗读设置面板 -->

    <!-- 目录面板（与标记列表面板同构） -->
    <div v-if="showToc" class="fullscreen-ann-panel toc-panel" :class="fullscreen ? '' : 'ann-panel'" @click.stop>
      <div class="fs-ann-header toc-header">
        <span>文档目录 ({{ chapters.length }})</span>
        <el-button text size="small" @click="showToc = false">
          <el-icon><Close /></el-icon>
        </el-button>
      </div>
      <div v-if="chapters.length === 0" class="fs-ann-empty">
        暂无目录<br/>PDF 需带书签或目录页，EPUB 需带目录文件
      </div>
      <div v-else class="fs-ann-list toc-list">
        <div v-for="(chapter, index) in chapters" :key="index"
          :ref="el => setTocItemRef(index, el)"
          class="fs-ann-item toc-item"
          :class="{ current: isChapterActive(index), l1: (chapter.level || 1) === 1, sub: (chapter.level || 1) > 1 }"
          :style="{ paddingLeft: (chapter.level ? (chapter.level - 1) * 18 : 0) + 14 + 'px' }"
          @click="goToChapter(index)">
          <span class="toc-branch" v-if="(chapter.level || 1) > 1"></span>
          <span class="toc-title-text">{{ chapter.title || `章节 ${index + 1}` }}</span>
          <span class="toc-page" v-if="chapter.page">p.{{ chapter.page }}</span>
        </div>
      </div>
    </div>

    <!-- 标记列表面板（全屏/非全屏统一） -->
    <div v-if="showAnnotations" class="fullscreen-ann-panel" :class="fullscreen ? '' : 'ann-panel'" @click.stop>
      <div class="fs-ann-header">
        <span>全部标记 ({{ annotations.length }})</span>
        <el-button text size="small" @click="showAnnotations = false">
          <el-icon><Close /></el-icon>
        </el-button>
      </div>
      <div v-if="annotations.length === 0" class="fs-ann-empty">
        暂无标记<br/>选中文本即可添加划线或笔记
      </div>
      <div v-else class="fs-ann-list">
        <div v-for="ann in sortedAnnotations" :key="ann.id" class="fs-ann-item"
          :class="{ current: isCurrentPageAnnotation(ann) }"
          @click="jumpToAnnotation(ann)">
          <div class="fs-ann-style" :style="annStylePreview(ann)">{{ annStyleLabel(ann) }}</div>
          <div class="fs-ann-body">
            <div class="fs-ann-text">{{ ann.content }}</div>
            <div class="fs-ann-meta">
              <el-tag size="small" type="info" effect="dark">{{ annPageLabel(ann) }}</el-tag>
              <span v-if="ann.note" class="fs-ann-note-icon"><el-icon><EditPen /></el-icon></span>
            </div>
            <div v-if="ann.note" class="fs-ann-note">{{ ann.note }}</div>
          </div>
          <div class="fs-ann-delete" @click.stop>
            <el-button text type="danger" size="small" title="删除" @click.stop="deleteAnnotation(ann.id)">
              <el-icon :size="15"><Delete /></el-icon>
            </el-button>
          </div>
        </div>
      </div>
    </div>

    <!-- 设置面板（标记列表同款样式） -->
    <div v-if="showSettings" class="fullscreen-ann-panel" :class="fullscreen ? '' : 'ann-panel'" @click.stop>
      <div class="fs-ann-header">
        <span>阅读设置</span>
        <el-button text size="small" @click="showSettings = false">
          <el-icon><Close /></el-icon>
        </el-button>
      </div>
      <div class="fs-ann-list">
        <div class="settings-panel">
          <p class="settings-save-tip">所有配置即时生效并自动保存</p>
          <h4>主题</h4>
          <div class="theme-options">
            <div v-for="theme in themes" :key="theme.name"
              class="theme-option"
              :class="{ active: currentTheme === theme.name }"
              :style="{ background: theme.bg, color: theme.text }"
              @click="applyTheme(theme)">
              {{ theme.label }}
            </div>
          </div>
          <template v-if="book?.file_type?.toLowerCase() === 'pdf'">
            <h4>页面布局</h4>
            <div class="layout-options">
              <div class="layout-option" :class="{ active: pdfLayout === 'single' }" @click="onPdfLayoutChange('single')">
                <el-icon><Document /></el-icon>
                <span>单页滚动</span>
              </div>
              <div class="layout-option" :class="{ active: pdfLayout === 'two' }" @click="onPdfLayoutChange('two')">
                <el-icon><Files /></el-icon>
                <span>双页平铺</span>
              </div>
            </div>
            <p class="setting-tip">双页平铺适合宽屏，自动缩放让两页并排完整显示；窄屏自动回退单页</p>
            <h4>页面缩放</h4>
            <el-slider v-model="pdfScaleSlider" :min="50" :max="300" :step="5" :show-tooltip="false" @input="onPdfScaleChange" @change="onPdfScaleChange" />
            <div class="setting-value">{{ Math.round(pdfScale * 100) }}%</div>
            <p class="setting-tip">拖动滑块调整页面显示比例，也可使用阅读页顶部的 + / - 按钮</p>
          </template>
          <template v-else>
            <template v-if="book?.file_type?.toLowerCase() === 'epub' || book?.file_type?.toLowerCase() === 'kepub' || book?.file_type?.toLowerCase() === 'azw3'">
              <p class="setting-tip">章节纸张连续排列、自动衔接，滚动到底部自动加载下一章；文字在纸张内居中显示</p>
            </template>
            <h4>字体大小</h4>
            <el-slider v-model="fontSize" :min="12" :max="28" :step="1" show-stops />
            <div class="setting-value">{{ fontSize }}px</div>
            <h4>行间距</h4>
            <el-slider v-model="lineHeight" :min="1.2" :max="2.5" :step="0.1" />
            <div class="setting-value">{{ lineHeight }}</div>
            <h4>页面边距</h4>
            <el-slider v-model="pageMargin" :min="8" :max="80" :step="4" />
            <div class="setting-value">{{ pageMargin }}px</div>
            <h4>字体</h4>
            <el-select v-model="fontFamily" style="width: 100%">
              <el-option label="系统默认" value="system-ui, -apple-system, sans-serif" />
              <el-option label="宋体" value="SimSun, serif" />
              <el-option label="黑体" value="SimHei, sans-serif" />
              <el-option label="楷体" value="KaiTi, serif" />
            </el-select>
          </template>
        </div>
      </div>
    </div>

    <!-- 读后感面板 -->
    <div v-if="showReviews" class="fullscreen-ann-panel" :class="fullscreen ? '' : 'ann-panel'" @click.stop>
      <div class="fs-ann-header">
        <span>读后感 <b class="fs-ann-count">{{ bookReviews.length }}</b></span>
        <el-button text size="small" @click="showReviews = false">
          <el-icon><Close /></el-icon>
        </el-button>
      </div>
      <div class="fs-ann-list">
        <div v-if="bookReviews.length === 0" class="fs-ann-empty">
          <el-icon :size="34"><Notebook /></el-icon>
          <p>这本书还没有读后感</p>
          <span>记录阅读时的所思所想</span>
        </div>
        <template v-else>
          <div v-for="r in bookReviews" :key="r.id" class="review-item">
            <div class="review-item-head">
              <span class="review-item-title">{{ r.title }}</span>
              <el-rate v-if="r.rating > 0" :model-value="r.rating" disabled :max="5" size="small" />
            </div>
            <div class="review-item-excerpt">{{ reviewExcerpt(r.content) }}</div>
            <div class="review-item-foot">
              <span class="review-item-time">
                <el-icon :size="12"><Clock /></el-icon>
                {{ reviewTime(r.updated_at) }}
              </span>
              <span class="review-item-actions">
                <el-button text type="primary" size="small" @click.stop="editReviewInReader(r)">编辑</el-button>
                <el-button text type="danger" size="small" @click.stop="removeReview(r)">删除</el-button>
              </span>
            </div>
          </div>
        </template>
        <div class="review-write-btn">
          <el-button type="primary" style="width: 100%; border-radius: 10px" @click="writeReviewInReader()">
            <el-icon style="margin-right: 4px"><Plus /></el-icon> 写读后感
          </el-button>
        </div>
      </div>
    </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch, nextTick, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { bookApi } from '@/api/book'
import { reviewApi } from '@/api/review'
import { getToken } from '@/api/auth'
import { API_BASE, wsApiBase } from '@/config'
import { loadReaderPrefs, saveReaderPrefs } from '@/utils/readerPrefs'
import { markdownToText } from '@/utils/markdown'
import { copyText } from '@/utils/clipboard'
import { notifyProgressUpdate } from '@/utils/progressSync'
import * as pdfjsLib from 'pdfjs-dist'

// 用 BASE_URL 拼接：兼容网关前缀（/app/moyue/pdf.worker.min.js）与 App file:// 相对加载
pdfjsLib.GlobalWorkerOptions.workerSrc = `${import.meta.env.BASE_URL}pdf.worker.min.js`

const props = defineProps({ id: String, book: Object, embedded: Boolean })
const emit = defineEmits(['pdf-select', 'pdf-clear'])
const router = useRouter()

// === 阅读时长统计 ===
const sessionStart = ref(Date.now())
let activeSeconds = 0
let sessionTimer = null

// 定时累计有效阅读时长（页面可见时）
const startReadingSession = () => {
  sessionStart.value = Date.now()
  activeSeconds = 0
  sessionTimer = setInterval(() => {
    if (!document.hidden) {
      activeSeconds += 5
    }
  }, 5000)
}

// 上报阅读会话（离开页面或组件卸载时）
const reportReadingSession = async () => {
  if (activeSeconds < 10) return // 不足 10 秒忽略
  const now = new Date()
  try {
    await bookApi.reportReading(
      Number(props.id),
      activeSeconds,
      new Date(sessionStart.value).toISOString(),
      now.toISOString()
    )
  } catch (e) {
    // 静默失败，不影响阅读
  }
}

// === 基础状态 ===
const book = ref(null)
const chapters = ref([])
const currentChapter = ref(0)
const currentProgress = ref(0)
const showToc = ref(false)
const showSettings = ref(false)
const showAnnotations = ref(false)
const showReviews = ref(false)
const fullscreen = ref(false)

// 弹框互斥：目录/标记/设置/读后感/选中文字工具栏，打开其一自动关闭其它，避免浮层重叠遮挡
const closeAllPanels = () => {
  showToc.value = false
  showSettings.value = false
  showAnnotations.value = false
  showReviews.value = false
  showHighlightToolbar.value = false
}
watch(showToc, (v) => {
  if (v) { showSettings.value = false; showAnnotations.value = false; showReviews.value = false; showHighlightToolbar.value = false }
})
watch(showSettings, (v) => {
  if (v) { showToc.value = false; showAnnotations.value = false; showReviews.value = false; showHighlightToolbar.value = false }
})
watch(showAnnotations, (v) => {
  if (v) { showToc.value = false; showSettings.value = false; showReviews.value = false; showHighlightToolbar.value = false }
})
watch(showReviews, (v) => {
  if (v) { showToc.value = false; showSettings.value = false; showAnnotations.value = false; showHighlightToolbar.value = false }
})
const readerBody = ref(null)

// === 阅读设置（与全局设置页共享同一份持久化配置） ===
const savedPrefs = loadReaderPrefs()
const fontSize = ref(savedPrefs.fontSize)
const lineHeight = ref(savedPrefs.lineHeight)
const pageMargin = ref(
  typeof window !== 'undefined' && window.innerWidth <= 768 ? Math.min(savedPrefs.pageMargin, 16) : savedPrefs.pageMargin
)
const fontFamily = ref(savedPrefs.fontFamily)
// 旧版纯黑主题键名为 'black'，与「设置 → 阅读偏好」及 EPUB 阅读器统一为 'pitch'
// （否则设置页选纯黑后这里拿到 'pitch'，CSS 只匹配 'black' 会导致 PDF 丢失主题样式）
let initTheme = savedPrefs.theme || 'light'
if (initTheme === 'black') initTheme = 'pitch'
const currentTheme = ref(initTheme)
// EPUB 连续滚动模式（默认分页，持久化）
// EPUB 仅连续滚动模式（分页阅读已移除，此值恒为 true 以保持既有分支逻辑简化）
const epubScroll = ref(true)

const themes = [
  { name: 'light', label: '日间', bg: '#ffffff', text: '#1a1a2e', bodyBg: '#ffffff' },
  { name: 'sepia', label: '护眼', bg: '#f4ecd8', text: '#433422', bodyBg: '#f4ecd8' },
  { name: 'dark', label: '夜间', bg: '#1a1a2e', text: '#e0e0e0', bodyBg: '#1a1a2e' },
  { name: 'pitch', label: '纯黑', bg: '#000000', text: '#888888', bodyBg: '#000000' }
]

// EPUB 阅读模式专用配色（严格按参考图设计）：
// - 桌面背景：纸张/侧栏以外的区域（浅色主题浅灰、暗色主题深灰）
// - 侧栏背景：纸张左右两侧的"装饰条"（参考图为淡紫灰，形成纸张的视觉边界）
const getEpubDesktopBg = (themeName) => {
  if (themeName === 'dark') return '#15151a'
  if (themeName === 'pitch') return '#050505'
  if (themeName === 'sepia') return '#d8c8a8'
  return '#eef0f2' // light
}
const getEpubSidebarBg = (themeName) => {
  if (themeName === 'dark') return '#1a1a20'
  if (themeName === 'pitch') return '#0a0a0a'
  if (themeName === 'sepia') return '#ddd0b6'
  return '#e7eaf2' // light（参考图淡紫灰）
}
const getEpubPaperBg = (themeName) => {
  const t = themes.find(x => x.name === themeName)
  return (t && t.bodyBg) || '#ffffff'
}
const epubDesktopBg = ref(getEpubDesktopBg(currentTheme.value))
const epubSidebarBg = ref(getEpubSidebarBg(currentTheme.value))
watch(currentTheme, (v) => {
  epubDesktopBg.value = getEpubDesktopBg(v)
  epubSidebarBg.value = getEpubSidebarBg(v)
})

// PDF 主题配色：直接复用 EPUB 的同一套取值（桌面背景 + 纸张色），
// 保证 PDF 与 EPUB 在不同主题下的观感一致（"纸张放在桌面上"的层次）。
const pdfDesktopBg = computed(() => getEpubDesktopBg(currentTheme.value))
const pdfPaperBg = computed(() => getEpubPaperBg(currentTheme.value))

// === 语音朗读（GPT-SoVITS：声音克隆 + 情感控制）===
// 朗读功能已恢复显示（曾临时隐藏入口，代码一直保留）
const ttsHidden = false
const ttsActive = ref(false)
const ttsPlaying = ref(false)
const ttsPaused = ref(false)
// 朗读专用的当前 PDF 页码（与视口 IntersectionObserver 跟踪的 currentPdfPage 解耦，
// 防止朗读中滚动时页码被自动切走导致后续句子高亮失败/翻页错乱）
const ttsPdfPage = ref(0)
// 已保存的朗读断点（打开书时从 book.tts_location 解析）：{ type: 'pdf'|'epub', page|chapter, sent }
const savedTtsLocation = ref(null)
// 阅读进度保存节流（定时器 + 防重入）
let progressSaveTimer = null
let progressSaving = false
const ttsLoading = ref(false)
const ttsSentences = ref([])
const ttsIndex = ref(-1)
// 外壳统一 TTS 驱动 PDF 朗读时的“当前句下标”（内联 ttsIndex 已停用，断点改取此值）
const pdfTtsSentence = ref(-1)
const ttsSpeed = ref(1.0)
const ttsEmotion = ref('平静')
const ttsCurrentText = ref('')
const ttsEmotions = ref(['平静'])
const ttsVoices = ref([])
const ttsVoice = ref('zh-CN-XiaoxiaoNeural') // Edge TTS 音色 short name
const ttsEngines = ref([
  { key: 'edge-tts', label: 'Edge TTS（在线，免费）', desc: '微软在线语音，无需本地模型，需联网。情感/语速为近似模拟。', available: true, default: true },
  { key: 'gpt-sovits', label: 'GPT-SoVITS（声音克隆）', desc: '官方 api_v2 服务，参考音频克隆音色，需本地部署模型。', available: true, default: false },
  { key: 'fish-speech', label: 'Fish-Speech（声音克隆）', desc: '官方 API server，支持预置音色 reference_id 或参考音频克隆。', available: true, default: false },
  { key: 'chattts', label: 'ChatTTS（本地模型）', desc: '本地 ChatTTS 模型合成，支持参考音色、固定音色种子与语速控制。需安装 ChatTTS 并下载模型。', available: true, default: false }
])
const ttsConfig = ref({
  base_url: '',
  ref_audio_path: '',
  prompt_text: '',
  prompt_lang: 'zh',
  text_lang: 'zh',
  speed_factor: 1.0,
  engine: 'edge-tts',
  reference_id: '',
  emotion_enabled: true
})
const ttsChapterCache = new Map()
// 复用的播放元素：移动端自动播放策略只允许"已被用户手势解锁"的媒体元素在
// 无手势时继续 play()，因此整个朗读过程复用同一个元素（每次仅换 src），
// 而不是为每个句子 new Audio()，否则第 2 句起的 play() 会被浏览器拒绝。
let ttsAudio = null
let ttsObjectUrl = null
// 播放被自动播放策略拒绝后待恢复的句子索引（等待用户轻触屏幕解锁后重试）
let ttsPendingPlay = null
let ttsUnlockBound = false
// 预合成缓存：句子索引 -> Promise<Blob>，播放当前句时后台预取下一句，实现无缝续播
const ttsCache = new Map()
// 连续合成失败计数，超过阈值则停止朗读，避免无限跳过
let ttsFailCount = 0

// 加载 TTS 配置与情感列表
const loadTtsSettings = async () => {
  try {
    const { data } = await bookApi.getTtsSettings()
    if (data?.data?.config) {
      ttsConfig.value = { ...ttsConfig.value, ...data.data.config }
      if (data.data.config.speed_factor > 0) ttsSpeed.value = data.data.config.speed_factor
    }
    if (data?.data?.emotions?.length) ttsEmotions.value = data.data.emotions
    if (data?.data?.engines?.length) ttsEngines.value = data.data.engines
    if (data?.data?.voices?.length) {
      ttsVoices.value = data.data.voices
      if (!ttsVoice.value && ttsVoices.value.length) ttsVoice.value = ttsVoices.value[0].short_name
    } else {
      // 音色列表加载失败（离线/网络），仍保留默认晓晓
      ttsVoices.value = [{ short_name: ttsVoice.value, friendly_name: '晓晓（默认）', gender: 'Female' }]
    }
    // 音色偏好优先取本地保存值（Go 后端不保存朗读配置），其次用默认配置
    const savedVoice = loadReaderPrefs().ttsVoice
    if (savedVoice) ttsVoice.value = savedVoice
    else if (data?.data?.config?.voice) ttsVoice.value = data.data.config.voice
  } catch (e) {
    // 静默失败，使用默认配置
  }
}

// 文本切分为朗读句子（中英标点分段，过长再按逗号截断）
const splitSentences = (text) => {
  const cleaned = (text || '').replace(/\s+/g, ' ').trim()
  if (!cleaned) return []
  const parts = cleaned.match(/[^。！？!?；;\n]+[。！？!?；;]?/g) || []
  const sents = []
  let buf = ''
  const pushBuf = () => { if (buf.trim()) sents.push(buf.trim()); buf = '' }
  for (const p of parts) {
    const t = p.trim()
    if (!t) continue
    buf += t
    while (buf.length > 90) {
      let cut = buf.slice(0, 90)
      const idx = Math.max(cut.lastIndexOf('，'), cut.lastIndexOf(','), cut.lastIndexOf(' '))
      if (idx > 20) cut = cut.slice(0, idx + 1)
      sents.push(cut.trim())
      buf = buf.slice(cut.length)
    }
    if (/[。！？!?；;]$/.test(buf)) pushBuf()
  }
  pushBuf()
  return sents
}

// EPUB：获取章节纯文本（缓存）
const getEpubChapterText = async (chapter) => {
  if (ttsChapterCache.has(chapter)) return ttsChapterCache.get(chapter)
  const { data } = await bookApi.getBookChapterText(props.id, chapter)
  const text = data?.data?.text || ''
  ttsChapterCache.set(chapter, text)
  return text
}

// PDF：提取指定页文本（默认朗读页；未指定时取 ttsPdfPage）
const getPdfPageText = (pageNum) => {
  const layer = pageTextLayerEls[pageNum ?? ttsPdfPage.value]
  if (!layer) return ''
  return Array.from(layer.querySelectorAll('span'))
    .map(s => s.textContent || '')
    .join('')
}

const buildTtsSentencesFromChapter = async (chapter) => splitSentences(await getEpubChapterText(chapterFileId(chapter)))

const buildTtsSentencesFromPdf = (pageNum) => splitSentences(getPdfPageText(pageNum ?? ttsPdfPage.value))

// 等待某页文本层渲染完成并返回其全文（防止自动翻页时读到空白页导致跳页/停止）
const waitForPdfPageText = async (pageNum, timeout = 8000) => {
  const page = pageInfos.value.find(p => p.num === pageNum)
  if (page && !page.rendered) {
    page.rendered = true
    nextTick(() => renderPdfPage(pageNum))
  }
  const start = Date.now()
  while (Date.now() - start < timeout) {
    const layer = pageTextLayerEls[pageNum]
    if (layer) {
      const text = Array.from(layer.querySelectorAll('span')).map(s => s.textContent || '').join('')
      if (text.trim()) return text
    }
    await new Promise(r => setTimeout(r, 120))
  }
  const layer = pageTextLayerEls[pageNum]
  return layer ? Array.from(layer.querySelectorAll('span')).map(s => s.textContent || '').join('') : ''
}

// 停止当前音频（保留复用元素，供后续句子继续播放）
const stopTtsAudio = () => {
  if (ttsAudio) {
    ttsAudio.pause()
    ttsAudio.onended = null
    ttsAudio.onerror = null
  }
  if (ttsObjectUrl) {
    URL.revokeObjectURL(ttsObjectUrl)
    ttsObjectUrl = null
  }
}

// 获取复用的播放元素（懒创建；iOS 需要 playsinline 才能内联播放）
const getTtsPlayElement = () => {
  if (!ttsAudio) {
    ttsAudio = new Audio()
    ttsAudio.preload = 'auto'
    ttsAudio.setAttribute('playsinline', '')
    ttsAudio.setAttribute('webkit-playsinline', '')
  }
  return ttsAudio
}

// 播放被自动播放策略拒绝后，等待用户轻触屏幕（产生用户手势）自动恢复播放
const resumeTtsAfterUnlock = () => {
  cleanupTtsUnlock()
  const idx = ttsPendingPlay
  ttsPendingPlay = null
  if (idx == null || !ttsActive.value) return
  // 元素 src 已就绪：直接在有手势的上下文中恢复播放，无需重新合成
  if (ttsAudio && ttsAudio.src) {
    ttsAudio.play().then(() => {
      ttsPlaying.value = true
      ttsPaused.value = false
    }).catch(() => {
      playSentence(idx) // 极少数 WebView 仍拒绝：走完整重播流程
    })
    return
  }
  playSentence(idx)
}
const cleanupTtsUnlock = () => {
  if (!ttsUnlockBound) return
  ttsUnlockBound = false
  window.removeEventListener('pointerdown', resumeTtsAfterUnlock, { capture: true })
}
const registerTtsUnlock = () => {
  if (ttsUnlockBound) return
  ttsUnlockBound = true
  window.addEventListener('pointerdown', resumeTtsAfterUnlock, { capture: true, once: true })
}

// 判断 play() 是否因移动端自动播放策略被拒绝（而非真正的合成/解码失败）
const isAutoplayBlocked = (e) => {
  if (!e) return false
  if (e.name === 'NotAllowedError') return true
  return /notallowed|autoplay|interact with the document/i.test(e.message || '')
}

// 归一化文本（忽略全部空白），用于跨 span/节点的朗读文本匹配
const normTtsText = (s) => (s || '').replace(/\s+/g, '')

// 清除朗读高亮（PDF overlay 矩形 + 旧 span 类 + EPUB 段落块）
const clearTtsHighlights = () => {
  clearTtsHighlightRects()
  document.querySelectorAll('.tts-speaking').forEach(el => el.classList.remove('tts-speaking'))
  // 连续滚动模式：清空所有已加载章节 iframe 内的高亮
  if (epubScroll.value) {
    Object.values(epubIframes).forEach(f => {
      if (f?.contentDocument) f.contentDocument.querySelectorAll('.tts-speaking-block').forEach(el => el.classList.remove('tts-speaking-block'))
    })
    return
  }
  const iframe = currentIframe.value
  if (iframe?.contentDocument) {
    iframe.contentDocument.querySelectorAll('.tts-speaking-block').forEach(el => el.classList.remove('tts-speaking-block'))
  }
}

// 记录上一次句子匹配结束的位置（归一化空间下标 + 页码）：同页顺序朗读从此继续，
// 避免重复文本被更早位置命中而“高亮到上一句”；换页/未命中时重置。
let ttsPdfNormFrom = 0
let ttsPdfSearchPage = 0

// 每页文本层索引缓存：原始拼接文本 + 去空白归一化文本 + 归一化下标→原始下标 映射 + 各 span 文本片段。
// 文本层渲染后静态不变，缓存以免每句重复遍历整页。
const pdfTextIndex = new Map()
const getPdfTextIndex = (pageNum) => {
  const cached = pdfTextIndex.get(pageNum)
  if (cached) return cached
  const layer = pageTextLayerEls[pageNum]
  if (!layer) return null
  const spans = Array.from(layer.querySelectorAll('span'))
  let raw = ''
  let norm = ''
  const normToRaw = []
  const segs = []
  for (const sp of spans) {
    const node = sp.firstChild
    const t = sp.textContent || ''
    const segStart = raw.length
    raw += t
    for (let i = 0; i < t.length; i++) {
      if (!/\s/.test(t[i])) { norm += t[i]; normToRaw.push(segStart + i) }
    }
    if (node && node.nodeType === Node.TEXT_NODE && t.length) {
      segs.push({ node, start: segStart, end: raw.length })
    }
  }
  const idx = { spans, raw, norm, normToRaw, segs }
  pdfTextIndex.set(pageNum, idx)
  return idx
}

// 当前朗读高亮画在 overlay 上的矩形元素（跨页收集，便于下次/停止时一次性清除）
const ttsHighlightRects = []
function clearTtsHighlightRects() {
  if (ttsHighlightRects.length) {
    ttsHighlightRects.forEach(el => el.remove())
    ttsHighlightRects.length = 0
  }
}

// PDF：按句子字符区间【精确】高亮——用 Range 取该句字符的客户端矩形，画到页面 overlay。
// 只覆盖当前句字符，不会因 span 跨句而波及下一句。返回 { firstRect } 供滚动，未命中返回 null。
const highlightPdfSentence = (sent, pageNum) => {
  clearTtsHighlightRects() // 先清除上一句高亮
  const idx = getPdfTextIndex(pageNum)
  if (!idx || !idx.norm) return null
  const nsent = normTtsText(sent)
  if (!nsent) return null
  // 顺序光标定位（归一化空间）：优先从上次结束处继续，找不到再全局找
  let nStart = -1
  if (ttsPdfSearchPage === pageNum && ttsPdfNormFrom > 0) {
    nStart = idx.norm.indexOf(nsent, ttsPdfNormFrom)
  }
  if (nStart < 0) nStart = idx.norm.indexOf(nsent)
  if (nStart < 0) { ttsPdfNormFrom = 0; ttsPdfSearchPage = pageNum; return null }
  const nEnd = nStart + nsent.length
  ttsPdfNormFrom = nEnd
  ttsPdfSearchPage = pageNum
  // 归一化下标 → 原始下标（rawEnd 取最后一个命中字符的后一位，不含后续空白/下一句）
  const rawStart = idx.normToRaw[nStart]
  const rawEnd = idx.normToRaw[nEnd - 1] + 1
  // 找到覆盖 [rawStart, rawEnd) 的首尾 span 文本片段
  let firstSeg = null, lastSeg = null
  for (const seg of idx.segs) {
    if (seg.end <= rawStart) continue
    if (seg.start >= rawEnd) break
    if (!firstSeg) firstSeg = seg
    lastSeg = seg
  }
  if (!firstSeg) return null
  // 建 Range 精确覆盖该句字符（可能跨多个 span），取每行片段矩形
  let rects = []
  try {
    const range = document.createRange()
    const sn = firstSeg.node, en = lastSeg.node
    range.setStart(sn, Math.max(0, Math.min(rawStart - firstSeg.start, (sn.textContent || '').length)))
    range.setEnd(en, Math.max(0, Math.min(rawEnd - lastSeg.start, (en.textContent || '').length)))
    rects = Array.from(range.getClientRects())
  } catch (e) { rects = [] }
  // 画到页面 overlay（视口坐标 → overlay 相对坐标，天然适配当前缩放）
  const overlay = pageOverlayEls[pageNum]
  if (overlay && rects.length) {
    const oRect = overlay.getBoundingClientRect()
    for (const r of rects) {
      if (r.width < 0.5 || r.height < 0.5) continue
      const div = document.createElement('div')
      div.className = 'tts-speaking-rect'
      div.style.left = (r.left - oRect.left) + 'px'
      div.style.top = (r.top - oRect.top) + 'px'
      div.style.width = r.width + 'px'
      div.style.height = r.height + 'px'
      overlay.appendChild(div)
      ttsHighlightRects.push(div)
    }
  }
  return { firstRect: rects[0] || null, rects }
}

// PDF：将当前句平滑滚动到容器约 35% 高度处（保持上下文可见）；已在可视区则不打扰
const scrollPdfToSentence = (res) => {
  const container = pdfContainer.value
  if (!container || !res?.firstRect) return
  const cRect = container.getBoundingClientRect()
  const r = res.firstRect
  if (r.top > cRect.top + cRect.height * 0.15 && r.bottom < cRect.bottom - cRect.height * 0.15) return
  const targetTop = r.top - cRect.top + container.scrollTop - cRect.height * 0.35
  container.scrollTo({ top: Math.max(0, targetTop), behavior: 'smooth' })
}

// EPUB：记录上一次句子命中结束的文本位置；同章顺序朗读从该位置继续搜索，
// 避免短句/重复文本命中更早位置（如上一句内）导致"选中的是上一句"
let ttsEpubSearchPos = 0

// EPUB：在 iframe 文档中定位句子并高亮其所在段落，返回块元素
const highlightEpubSentence = (sent) => {
  // 滚动模式：朗读始终高亮在朗读章节的 iframe 上（即使用户滚动到其他章节）
  const iframe = epubScroll.value && ttsEpubChapter != null
    ? (epubIframes[ttsEpubChapter] || currentIframe.value)
    : currentIframe.value
  const doc = iframe?.contentDocument
  if (!doc?.body) return null
  const target = normTtsText(sent)
  if (!target) return null
  // 收集正文文本节点（跳过 script/style 与空白节点）
  const nodes = []
  const walker = doc.createTreeWalker(doc.body, NodeFilter.SHOW_TEXT, {
    acceptNode(node) {
      if (!normTtsText(node.nodeValue)) return NodeFilter.FILTER_REJECT
      const p = node.parentElement
      if (!p || (p.closest && p.closest('script, style'))) return NodeFilter.FILTER_REJECT
      return NodeFilter.FILTER_ACCEPT
    }
  })
  let n = walker.nextNode()
  while (n) { nodes.push(n); n = walker.nextNode() }
  // 拼接归一化全文，用 indexOf 定位句子起点（标签打断时用句首片段容错）
  let full = ''
  const lens = []
  for (const node of nodes) {
    const t = normTtsText(node.nodeValue)
    full += t
    lens.push(t.length)
  }
  // 优先从上次命中结束位置之后搜索（顺序朗读）；找不到则回退从头（覆盖选句跳读/重渲染）
  let idx = full.indexOf(target, ttsEpubSearchPos)
  let matchedLen = target.length
  if (idx < 0) {
    const frag = target.slice(0, 8)
    idx = full.indexOf(frag, ttsEpubSearchPos)
    matchedLen = frag.length
  }
  if (idx < 0) {
    idx = full.indexOf(target)
    matchedLen = target.length
    if (idx < 0) {
      const frag = target.slice(0, 8)
      idx = full.indexOf(frag)
      matchedLen = frag.length
    }
  }
  if (idx < 0) return null
  ttsEpubSearchPos = idx + matchedLen
  // 找到包含 idx 的文本节点
  let acc = 0
  let hitNode = null
  for (let k = 0; k < nodes.length; k++) {
    if (lens[k] > 0 && idx < acc + lens[k]) { hitNode = nodes[k]; break }
    acc += lens[k]
  }
  if (!hitNode) return null
  // 提升到最近的块级元素做段落级高亮
  let block = hitNode.parentElement
  while (block && !/^(P|DIV|H1|H2|H3|H4|H5|LI|BLOCKQUOTE|SECTION|ARTICLE)$/.test(block.tagName)) {
    block = block.parentElement
  }
  if (block) block.classList.add('tts-speaking-block')
  return block
}

const highlightTtsSentence = (i) => {
  clearTtsHighlights()
  const sent = ttsSentences.value[i]
  if (!sent) return
  const ft = book.value?.file_type?.toLowerCase()
  if (ft === 'pdf') {
    const hit = highlightPdfSentence(sent, ttsPdfPage.value)
    if (hit?.firstRect) scrollPdfToSentence(hit)
  } else if (ft === 'epub' || ft === 'kepub' || ft === 'azw3') {
    const block = highlightEpubSentence(sent)
    if (block) block.scrollIntoView({ block: 'center', behavior: 'smooth' })
  }
}

// ===== Edge TTS WebSocket 直连 =====
// 一条 WS 连接复用握手，支持并发合成多句，提升朗读预取效率。
let ttsWs = null
let ttsWsSeq = 0
let ttsWsDisabled = false // 连接级失败后本会话不再尝试 WS（自动回退 HTTP）
let ttsWsConnecting = null // 正在建立中的连接 Promise，供并发调用复用（避免重复建连）
const ttsWsPending = new Map() // id -> {resolve, reject}

const rejectAllTtsPending = (err) => {
  ttsWsPending.forEach(p => p.reject(err))
  ttsWsPending.clear()
}

const ensureTtsWs = () => {
  if (ttsWsDisabled) return Promise.reject(new Error('TTS WebSocket 已禁用'))
  if (ttsWs && ttsWs.readyState === WebSocket.OPEN) return Promise.resolve(ttsWs)
  if (ttsWsConnecting) return ttsWsConnecting
  const proto = location.protocol === 'https:' ? 'wss://' : 'ws://'
  ttsWsConnecting = new Promise((resolve, reject) => {
    const ws = new WebSocket(`${wsApiBase()}/tts/ws?token=${getToken()}`)
    ws.binaryType = 'arraybuffer'
    let settled = false
    // 关键：连接建立失败/断开时必须 reject 当前 Promise，
    // 否则调用方（synthesizeOneWs）将永久挂起，HTTP 回退也不会触发，朗读无法开始。
    const fail = (msg) => {
      if (settled) return
      settled = true
      ttsWsConnecting = null
      reject(new Error(msg))
    }
    ws.onopen = () => {
      settled = true
      ttsWsConnecting = null
      ttsWs = ws
      resolve(ws)
    }
    ws.onerror = () => {
      ttsWsDisabled = true
      ttsWs = null
      fail('TTS WebSocket 连接失败')
      rejectAllTtsPending(new Error('TTS WebSocket 连接失败'))
    }
    ws.onclose = () => {
      ttsWsConnecting = null
      if (ttsWs === ws) ttsWs = null
      fail('TTS WebSocket 连接已断开')
      rejectAllTtsPending(new Error('TTS WebSocket 连接已断开'))
    }
    ws.onmessage = (ev) => {
      if (typeof ev.data === 'string') {
        let msg = null
        try { msg = JSON.parse(ev.data) } catch (e) { return }
        const p = ttsWsPending.get(msg.id)
        if (p) { ttsWsPending.delete(msg.id); p.reject(new Error(msg.error || '合成失败')) }
        return
      }
      // 二进制：[int32 BE id][MP3]
      const buf = new DataView(ev.data)
      const id = buf.getInt32(0)
      const p = ttsWsPending.get(id)
      if (!p) return
      ttsWsPending.delete(id)
      p.resolve(new Blob([ev.data.slice(4)], { type: 'audio/mpeg' }))
    }
  })
  return ttsWsConnecting
}

// 关闭 WS 连接（停止朗读时调用），并拒绝所有未决请求
const closeTtsWs = () => {
  rejectAllTtsPending(new Error('朗读已停止'))
  if (ttsWs) {
    ttsWs.onclose = null
    ttsWs.onerror = null
    ttsWs.close()
    ttsWs = null
  }
}

// 通过 WebSocket 合成一句（Edge TTS Go 直连）
const synthesizeOneWs = (i) => {
  return ensureTtsWs().then(ws => new Promise((resolve, reject) => {
    const id = ++ttsWsSeq
    ttsWsPending.set(id, { resolve, reject })
    ws.send(JSON.stringify({
      id,
      text: ttsSentences.value[i],
      emotion: ttsEmotion.value,
      speed_factor: ttsSpeed.value,
      voice: ttsVoice.value
    }))
  }))
}

// 合成单句（返回 Promise<Blob>，播放与预取共用）
// 优先走 WebSocket（连接复用 + 并发合成），失败自动回退 HTTP
const synthesizeOne = (i) => {
  const wsPromise = synthesizeOneWs(i)
    .catch(err => {
      console.warn('TTS WebSocket 合成失败，回退 HTTP：', err.message)
      return bookApi.synthesizeTts({
        text: ttsSentences.value[i],
        emotion: ttsEmotion.value,
        speed_factor: ttsSpeed.value,
        media_type: 'wav',
        voice: ttsVoice.value
      }).then(resp => new Blob([resp.data], { type: resp.headers['content-type'] || 'audio/wav' }))
    })
  return wsPromise
}

// 切换人声：更新当前音色并本地记住偏好（与阅读设置一致），朗读中则用新音色重播当前句
const onTtsVoiceChange = (v) => {
  ttsConfig.value.voice = v
  if (ttsActive.value && ttsIndex.value >= 0) restartTtsIfPlaying()
  saveReaderPrefs({ ...loadReaderPrefs(), ttsVoice: v })
}

// 音色显示名（朗读条下拉）
const ttsVoiceLabel = (v) => {
  let name = (v.friendly_name || v.short_name || '').replace(/^Microsoft\s*/, '')
  name = name.replace(/\s*-\s*Chinese\s*\(Mainland\)\s*$/, '')
  const g = v.gender === 'Female' ? '女声' : v.gender === 'Male' ? '男声' : ''
  return g ? `${name}（${g}）` : name
}

// 取一句的音频 Blob：优先用预取缓存，未命中则实时合成
// （播放统一走复用的 getTtsPlayElement，避免移动端自动播放策略拒绝新建元素）
const getSentenceBlob = (i) => {
  const item = ttsCache.get(i)
  ttsCache.delete(i)
  if (item) return item
  return synthesizeOne(i)
}

// 清空预取缓存（blob 尚未创建 objectURL，无需逐个 revoke）
const clearTtsCache = () => {
  ttsCache.clear()
}

// 滚动预取后续若干句（后台合成，失败 resolve null，播放时实时重试）
// 窗口覆盖长句合成耗时，配合 WS 连接复用 + 并发合成，最大限度减少句间卡顿
const TTS_PREFETCH_WINDOW = 6 // 当前句 + 后 5 句
const prefetchTTS = (i) => {
  const n = ttsSentences.value.length
  if (!ttsActive.value || ttsPaused.value || i < 0 || i >= n) return
  for (let k = i; k < i + TTS_PREFETCH_WINDOW && k < n; k++) {
    if (ttsCache.has(k)) continue
    ttsCache.set(k, synthesizeOne(k).catch(() => null))
  }
}

// 播放指定句子
const playSentence = async (i) => {
  const n = ttsSentences.value.length
  if (i < 0 || i >= n) {
    handleTtsEnd()
    return
  }
  ttsIndex.value = i
  ttsCurrentText.value = ttsSentences.value[i]
  highlightTtsSentence(i)
  scheduleProgressSave() // 每句开始播放时节流记录朗读断点
  stopTtsAudio()
  ttsLoading.value = true
  prefetchTTS(i) // 并发预取当前句与下一句，利用 WS 复用连接
  try {
    let blob = await getSentenceBlob(i)
    // 预取失败（resolve null）时实时重试一次
    if (!blob && ttsIndex.value === i) blob = await synthesizeOne(i)
    if (ttsIndex.value !== i) return // 已被跳过/停止
    if (!blob) throw new Error('语音合成失败')
    // 复用已解锁的播放元素（移动端自动播放策略：新建元素的 play() 会被拒绝）
    const audio = getTtsPlayElement()
    const url = URL.createObjectURL(blob)
    ttsObjectUrl = url
    audio.src = url
    audio.load()
    audio.onended = () => {
      if (ttsObjectUrl === url) ttsObjectUrl = null
      URL.revokeObjectURL(url)
      ttsFailCount = 0
      if (!ttsPaused.value && ttsActive.value) playSentence(i + 1)
    }
    audio.onerror = () => {
      if (ttsObjectUrl === url) ttsObjectUrl = null
      URL.revokeObjectURL(url)
      ElMessage.error('语音播放失败，已停止朗读')
      stopTTS()
    }
    await audio.play()
    ttsPlaying.value = true
    ttsPaused.value = false
  } catch (e) {
    ttsCache.delete(i)
    // 移动端自动播放策略拒绝无手势 play()：不是合成失败，等待用户轻触屏幕恢复
    if (isAutoplayBlocked(e)) {
      ttsPendingPlay = i
      ttsPlaying.value = false
      ttsPaused.value = true
      ElMessage.warning('请轻触屏幕继续朗读')
      registerTtsUnlock()
      return
    }
    // 真正的合成/解码/播放失败：释放本次未播放成功的 blob URL 再跳过
    if (ttsObjectUrl) {
      URL.revokeObjectURL(ttsObjectUrl)
      ttsObjectUrl = null
    }
    ttsFailCount += 1
    ElMessage.warning(`第 ${i + 1} 句合成失败，已跳过：${e.response?.data?.error || e.message || '请检查 TTS 配置'}`)
    if (ttsFailCount >= 3) {
      ElMessage.error('连续多句合成失败，已停止朗读')
      stopTTS()
    } else if (ttsActive.value) {
      playSentence(i + 1)
    }
  } finally {
    ttsLoading.value = false
  }
}

// 当前内容播放完毕：PDF 翻页 / EPUB 切章 / 结束
const handleTtsEnd = async () => {
  if (!ttsActive.value) return
  const ft = book.value?.file_type?.toLowerCase()
  if (ft === 'pdf') {
    if (ttsPdfPage.value < totalPdfPages.value) {
      const next = ttsPdfPage.value + 1
      ttsPdfPage.value = next
      currentPdfPage.value = next
      // 平滑滚动到下一页，模拟自动翻页
      scrollToPdfPage(next, true)
      clearTtsCache()
      ttsFailCount = 0
      scheduleProgressSave(true) // 翻页即记录断点
      // 等待下一页文本层真正渲染完成，避免读到空文本导致跳页
      const text = await waitForPdfPageText(next)
      ttsSentences.value = splitSentences(text)
      if (ttsSentences.value.length > 0) {
        playSentence(0)
      } else if (next < totalPdfPages.value) {
        handleTtsEnd()
      } else {
        ElMessage.success('全书朗读完毕')
        stopTTS()
      }
    } else {
      ElMessage.success('全书朗读完毕')
      stopTTS()
    }
  } else if (ft === 'epub' || ft === 'kepub' || ft === 'azw3') {
    if (currentChapter.value < chapters.value.length - 1) {
      await renderChapter(currentChapter.value + 1)
      clearTtsCache()
      ttsFailCount = 0
      scheduleProgressSave(true) // 切章即记录断点
      ttsEpubChapter = currentChapter.value
      ttsEpubSearchPos = 0 // 章节文档已更换，重置高亮搜索位置
      ttsSentences.value = await buildTtsSentencesFromChapter(currentChapter.value)
      if (ttsSentences.value.length > 0) {
        playSentence(0)
      } else {
        handleTtsEnd()
      }
    } else {
      ElMessage.success('全书朗读完毕')
      stopTTS()
    }
  } else {
    stopTTS()
  }
}

// 开始朗读（默认从当前页/章开始；resumeLoc 为保存的断点 {type,page|chapter,sent} 时断点续播）
const startTTS = async (resumeLoc = null) => {
  if (ttsActive.value) {
    stopTTS()
    return
  }
  const ft = book.value?.file_type?.toLowerCase()
  if (ft !== 'pdf' && ft !== 'epub' && ft !== 'kepub' && ft !== 'azw3') {
    ElMessage.warning('该格式暂不支持语音朗读')
    return
  }
  // 加载引擎/情感/音色列表（Go 后端直连 Edge TTS，无需配置服务地址；失败静默使用默认）
  await loadTtsSettings()
  ttsActive.value = true
  ttsPlaying.value = true
  ttsPaused.value = false
  ttsIndex.value = -1
  ttsChapterCache.clear()
  try {
    if (ft === 'pdf') {
      // 断点续播：定位到断点页；否则以当前视口页作为朗读起点
      if (resumeLoc?.page && resumeLoc.page >= 1 && resumeLoc.page <= totalPdfPages.value) {
        ttsPdfPage.value = resumeLoc.page
        currentPdfPage.value = resumeLoc.page
        scrollToPdfPage(resumeLoc.page)
      } else {
        ttsPdfPage.value = currentPdfPage.value || 1
      }
      const text = await waitForPdfPageText(ttsPdfPage.value)
      ttsSentences.value = splitSentences(text)
    } else {
      if (resumeLoc?.chapter != null && resumeLoc.chapter >= 0 && resumeLoc.chapter < chapters.value.length) {
        await renderChapter(resumeLoc.chapter)
      }
      ttsEpubChapter = currentChapter.value
      ttsEpubSearchPos = 0 // 开始新朗读（可能更换了章节/文档），重置高亮搜索位置
      ttsSentences.value = await buildTtsSentencesFromChapter(currentChapter.value)
    }
  } catch (e) {
    ElMessage.error('获取朗读文本失败')
    stopTTS()
    return
  }
  if (ttsSentences.value.length === 0) {
    ElMessage.warning('当前内容为空，无法朗读')
    stopTTS()
    return
  }
  // 从断点句开始（越界则回退到本页/本章第一句）
  const startIdx = resumeLoc?.sent != null && resumeLoc.sent >= 0 && resumeLoc.sent < ttsSentences.value.length
    ? resumeLoc.sent
    : 0
  playSentence(startIdx)
}

// 从选中文字的位置开始朗读（面板"从选中位置开始阅读"）
const startTTSFromSelection = async () => {
  lockCollapseHide()
  const text = (selectedText.value || '').trim()
  if (!text) {
    ElMessage.warning('请先选中文字，再从该处开始朗读')
    return
  }
  const ft = book.value?.file_type?.toLowerCase()
  if (ft !== 'pdf' && ft !== 'epub' && ft !== 'kepub' && ft !== 'azw3') {
    ElMessage.warning('该格式暂不支持语音朗读')
    return
  }
  // 正在朗读则先停止，重新从选中位置开始
  if (ttsActive.value) stopTTS()
  // 定位到选中文字所在页/章节（取章节索引后再取消弹框，避免引用丢失）
  const selChapter = lastSelectionFrame?.dataset?.chapterIndex
  const loc = ft === 'pdf'
    ? { type: 'pdf', page: currentPdfPage.value || 1, sent: 0 }
    : { type: 'epub', chapter: selChapter != null ? Number(selChapter) : currentChapter.value, sent: 0 }
  cancelHighlight()
  await startTTS(loc)
  if (!ttsActive.value || !ttsSentences.value.length) return
  // 找到包含选中文字前 10 字的句子并从该句开始（找不到则保持本页/本章第一句）
  const key = normTtsText(text).slice(0, 10)
  if (!key) return
  const idx = ttsSentences.value.findIndex(s => normTtsText(s).includes(key))
  if (idx > 0) playSentence(idx)
}

// 从保存的朗读断点继续播放
const resumeTtsFromSaved = async () => {
  const loc = savedTtsLocation.value
  if (!loc) return
  const ft = book.value?.file_type?.toLowerCase()
  const isEpub = ft === 'epub' || ft === 'kepub' || ft === 'azw3'
  if ((ft === 'pdf' && loc.type !== 'pdf') || (isEpub && loc.type !== 'epub')) {
    ElMessage.warning('朗读断点与当前书籍不匹配，已从头开始')
    savedTtsLocation.value = null
  }
  await startTTS(savedTtsLocation.value)
  // 续播成功后保留断点（朗读中会持续更新），失败时清除避免反复点
  if (!ttsActive.value) savedTtsLocation.value = null
}

// 暂停 / 继续
const toggleTTSPlay = () => {
  if (!ttsAudio || !ttsActive.value) return
  // 处于"等待解锁"状态：由全局 pointerdown 手势监听统一恢复，这里不重复处理
  if (ttsPendingPlay != null) return
  if (ttsPaused.value) {
    ttsAudio.play().catch(() => {})
    ttsPaused.value = false
  } else {
    ttsAudio.pause()
    ttsPaused.value = true
  }
}

// 上 / 下一句
const prevTtsSentence = () => {
  if (ttsIndex.value > 0) playSentence(ttsIndex.value - 1)
}

const nextTtsSentence = () => {
  if (ttsIndex.value < ttsSentences.value.length - 1) playSentence(ttsIndex.value + 1)
}

// 切换情感后，若正在播放则用新情感重播当前句
const restartTtsIfPlaying = () => {
  if (ttsActive.value && ttsIndex.value >= 0) {
    clearTtsCache() // 清掉旧情感的预取缓存，确保后续句也按新情感重新合成
    playSentence(ttsIndex.value)
  }
}

// 生成朗读断点：PDF 记录页+句，EPUB 记录章节+句
const buildTtsLocation = () => {
  if (!book.value) return null
  const ft = book.value.file_type?.toLowerCase()
  const sent = Math.max(0, ttsIndex.value)
  if (ft === 'pdf') {
    // 朗读中记录朗读页，否则记录视口页（避免残留上次朗读的页码）
    const reading = ttsActive.value || pdfTtsActive.value
    const p = reading && ttsPdfPage.value ? ttsPdfPage.value : (currentPdfPage.value || 1)
    // 朗读中的“第几句”取外壳统一播放器回调下发的下标（pdfTtsSentence），否则回退内联 ttsIndex
    const s = reading ? Math.max(0, pdfTtsSentence.value) : Math.max(0, ttsIndex.value)
    return { type: 'pdf', page: p, sent: s }
  }
  if (ft === 'epub' || ft === 'kepub' || ft === 'azw3') {
    return { type: 'epub', chapter: currentChapter.value, sent }
  }
  return null
}

// 立即保存阅读进度 + 朗读断点（keepalive 用于页面关闭时尽可能送达）
// 返回 Promise 以便调用方 await 完成（保证退出阅读器时进度已落库）
const doSaveProgress = (keepalive = false) => {
  if (!book.value || progressSaving) return Promise.resolve()
  const ft = book.value.file_type?.toLowerCase()
  let progress, page
  const loc = buildTtsLocation()
  if (ft === 'pdf') {
    // 朗读时以朗读页为准，否则用视口页
    const reading = ttsActive.value || pdfTtsActive.value
    const p = reading && ttsPdfPage.value ? ttsPdfPage.value : currentPdfPage.value
    progress = totalPdfPages.value > 1 ? ((p - 1) / (totalPdfPages.value - 1)) * 100 : 0
    page = p
  } else {
    progress = currentProgress.value
    page = currentChapter.value
  }
  progress = Math.max(0, Math.min(100, progress))
  const ttsLoc = loc ? JSON.stringify(loc) : ''
  if (keepalive) {
    try {
      const headers = { 'Content-Type': 'application/json' }
      const token = getToken()
      if (token) headers.Authorization = `Bearer ${token}`
      fetch(`${API_BASE}/books/${props.id}/progress`, {
        method: 'PUT',
        headers,
        body: JSON.stringify({ progress, current_page: page, tts_location: ttsLoc }),
        keepalive: true
      }).catch(() => {})
      return
    } catch (e) { /* fallthrough 到 axios */ }
  }
  progressSaving = true
  return bookApi.updateProgress(props.id, progress, page, ttsLoc)
    .then(() => {
      // 保存成功后通知全局 store，详情页/书架可实时刷新进度
      notifyProgressUpdate(props.id, { progress, current_page: page, tts_location: ttsLoc })
    })
    .catch(() => {})
    .finally(() => { progressSaving = false })
}

// 同步落库进度（用于退出阅读器/卸载组件时 await，确保详情页 reload 拿到最新进度）
const flushProgressSave = async () => {
  if (progressSaveTimer) {
    clearTimeout(progressSaveTimer)
    progressSaveTimer = null
  }
  await doSaveProgress()
}

// 节流保存：朗读每句/滚动时 2s 合并一次；force 用于翻页/停止/卸载等关键节点立即保存
const scheduleProgressSave = (force = false) => {
  if (force) {
    if (progressSaveTimer) {
      clearTimeout(progressSaveTimer)
      progressSaveTimer = null
    }
    doSaveProgress()
    return
  }
  if (!progressSaveTimer) {
    progressSaveTimer = setTimeout(() => {
      progressSaveTimer = null
      doSaveProgress()
    }, 2000)
  }
}

// 停止朗读
const stopTTS = () => {
  if (ttsActive.value) scheduleProgressSave(true) // 记录停止时的朗读位置
  ttsEpubChapter = null
  ttsEpubSearchPos = 0 // 停止后重新朗读从文档开头搜索
  ttsPdfNormFrom = 0
  ttsPdfSearchPage = 0
  // 注意：不清空 pdfTtsSentence，否则卸载时先 stop 再保存会把断点句子写成 0（续播从头开始）
  stopTtsAudio()
  closeTtsWs() // 关闭 TTS WebSocket，取消未决合成
  clearTtsHighlights()
  clearTtsCache()
  cleanupTtsUnlock()
  ttsPendingPlay = null
  ttsFailCount = 0
  ttsActive.value = false
  ttsPlaying.value = false
  ttsPaused.value = false
  ttsLoading.value = false
  ttsSentences.value = []
  ttsIndex.value = -1
  ttsCurrentText.value = ''
}

// === PDF 状态 ===
// pdfDoc 使用普通变量，避免 Vue Proxy 导致 pdfjs-dist 私有字段访问错误
let pdfDoc = null
const currentPdfPage = ref(1)
const totalPdfPages = ref(0)
const pdfScale = ref(savedPrefs.pdfScale ?? 1.5)
// 手动调整缩放时持久化（自动适配的缩放不覆盖用户设置）
const savePdfScalePref = (v) => {
  const prefs = { ...loadReaderPrefs(), pdfScale: v }
  saveReaderPrefs(prefs)
}
// 双页平铺布局（仅 PDF）：single 单页滚动 / two 双页并排；持久化到本地偏好
const pdfLayout = ref(savedPrefs.pdfTwoPage ? 'two' : 'single')
watch(pdfLayout, (v) => {
  const prefs = { ...loadReaderPrefs(), pdfTwoPage: v === 'two' }
  saveReaderPrefs(prefs)
})
// 设置抽屉中的缩放滑块（50%-300%），跟随实际缩放值同步
const pdfScaleSlider = ref(Math.round(pdfScale.value * 100))
watch(pdfScale, (v) => { pdfScaleSlider.value = Math.round(v * 100) })
const onPdfScaleChange = (pct) => {
  mobileFitActive = false // 用户手动缩放后不再自动适配，尊重其选择
  applyZoom(pct / 100)
  savePdfScalePref(pdfScale.value)
}
const pdfCanvas = ref(null)
const pdfTextLayer = ref(null)
const pdfPageWrapper = ref(null)
const pdfOverlay = ref(null)
const pdfLoading = ref(false)

// === 连续滚动 PDF ===
const pdfContainer = ref(null)
const epubViewer = ref(null) // EPUB 渲染容器（滚动模式下监听滚动）
const pageInfos = ref([])
const visiblePages = ref(new Set())

// PDF 页面分组：单页每页一组；双页首页单独、其后两两成组并排
const pdfPageGroups = computed(() => {
  const two = pdfLayout.value === 'two'
  const pages = pageInfos.value
  const groups = []
  if (!two) {
    pages.forEach((p) => groups.push([p]))
    return groups
  }
  let i = 0
  if (pages.length > 0) {
    groups.push([pages[0]])
    i = 1
  }
  for (; i < pages.length; i += 2) {
    const g = [pages[i]]
    if (i + 1 < pages.length) g.push(pages[i + 1])
    groups.push(g)
  }
  return groups
})

// 切换布局：窄屏自动回退单页；切到双页自动调整缩放让两页并排完整显示，切回单页恢复原缩放
let singlePageScaleBeforeTwo = null
const onPdfLayoutChange = async (val) => {
  if (val === 'two' && window.innerWidth <= 768) {
    pdfLayout.value = 'single'
    ElMessage.warning('双页平铺需要较宽的屏幕，已自动回退单页')
    return
  }
  if (val === 'two') {
    singlePageScaleBeforeTwo = pdfScale.value
    pdfLayout.value = 'two'
    // 自动计算缩放：两页并排（含 14px 间距）正好铺满容器宽度，留 24px 边距
    await nextTick()
    const container = pdfContainer.value
    const page = pageInfos.value[0]
    if (container && page && pdfScale.value > 0) {
      const baseWidth = page.width / pdfScale.value // scale=1 时的页面宽度
      const fit = Math.max(0.5, Math.min(3.0, (container.clientWidth - 14 - 24) / (2 * baseWidth)))
      const zoomed = await applyZoom(fit)
      // 缩放未变时分组 DOM 已重建，需强制重绘并重绑观察器
      if (!zoomed) await rerenderPdfNearby()
    }
  } else {
    pdfLayout.value = 'single'
    if (singlePageScaleBeforeTwo !== null) {
      const restore = singlePageScaleBeforeTwo
      singlePageScaleBeforeTwo = null
      await nextTick()
      const zoomed = await applyZoom(restore)
      if (!zoomed) await rerenderPdfNearby()
    } else {
      // 进入页面即双页且未经过手动切换：切换回单页时 DOM 分组同样重建，需强制重绘
      await nextTick()
      await rerenderPdfNearby()
    }
  }
  nextTick(() => scrollToPdfPage(currentPdfPage.value))
}

// 页面元素引用（非响应式对象，避免 Vue Proxy 开销）
const pageCanvasEls = {}
const pageTextLayerEls = {}
const pageOverlayEls = {}

const setPageCanvasRef = (el, num) => { if (el) pageCanvasEls[num] = el; else delete pageCanvasEls[num] }
const setPageTextLayerRef = (el, num) => { if (el) pageTextLayerEls[num] = el; else delete pageTextLayerEls[num] }
const setPageOverlayRef = (el, num) => { if (el) pageOverlayEls[num] = el; else delete pageOverlayEls[num] }

let pageObserver = null
let scrollRaf = null

// === 批注/划线状态 ===
const annotations = ref([])
const selectedText = ref('')
const selectionRect = ref(null)
const showHighlightToolbar = ref(false)
const noteText = ref('')
const currentIframe = ref(null)
const highlightColors = ['#ffeb3b', '#a8e6cf', '#74add1', '#d8b4fe', '#fda4af']
const currentStyle = ref('highlight')
const currentColor = ref('#ffeb3b')
const currentTool = ref(null) // null | 'highlight' | 'underline' | 'wavy' | 'strikethrough' | 'eraser'
const noteMode = ref(false) // 选中记录笔记模式（不生成划线）
const showColorPicker = ref(false) // 工具栏颜色选择器展开状态
const toolbarPos = ref({ left: 0, top: 0 })

// 最近一次文字选中的时间戳：用于拦截紧随其后的 click（移动端 click 可能延迟 300ms+）
let lastSelectionAt = 0
// 最近一次选择发生的 iframe（连续滚动模式下多个 iframe 并存，划线需精确归属章节）
let lastSelectionFrame = null
// 点击移动端操作条按钮会让 iframe 失焦、选区折叠，期间抑制"折叠即收起"以免误关操作条
let suppressCollapseHide = false
const lockCollapseHide = () => {
  suppressCollapseHide = true
  setTimeout(() => { suppressCollapseHide = false }, 350)
}
// TTS 朗读归属的章节（滚动模式下高亮/切章以此为准，避免滚动导致串章）
let ttsEpubChapter = null

// === 当前页/章节的批注 ===
const currentPageAnnotations = computed(() => {
  const ft = book.value?.file_type?.toLowerCase()
  return annotations.value.filter(a => {
    try {
      const pos = JSON.parse(a.position || '{}')
      if (ft === 'pdf') return pos.page === currentPdfPage.value && a.type === 'highlight'
      return pos.chapter === currentChapter.value && a.type === 'highlight'
    } catch { return false }
  })
})

// === 全部标记按页码排序 ===
const annLocOf = (ann) => {
  try {
    const pos = JSON.parse(ann.position || '{}')
    const ft = book.value?.file_type?.toLowerCase()
    return ft === 'pdf' ? (pos.page || 0) : (pos.chapter || 0)
  } catch { return 0 }
}

const sortedAnnotations = computed(() => {
  return [...annotations.value].filter(a => a.type === 'highlight' || a.type === 'note').sort((a, b) => annLocOf(a) - annLocOf(b))
})

// 进度条上的标记位置点（PDF 按页、EPUB 按章换算成百分比）
const annotationMarkers = computed(() => {
  const ft = book.value?.file_type?.toLowerCase()
  const out = []
  if (ft === 'pdf') {
    const total = totalPdfPages.value
    if (!total) return out
    for (const a of annotations.value) {
      if (a.type !== 'highlight' && a.type !== 'note') continue
      try {
        const pos = JSON.parse(a.position || '{}')
        if (pos.page) out.push({ id: a.id, type: a.type, pct: Math.max(0, Math.min(100, (pos.page / total) * 100)) })
      } catch { /* 忽略损坏的 position */ }
    }
  } else if (ft === 'epub' || ft === 'kepub' || ft === 'azw3') {
    const n = chapters.value.length
    if (!n) return out
    for (const a of annotations.value) {
      if (a.type !== 'highlight' && a.type !== 'note') continue
      try {
        const pos = JSON.parse(a.position || '{}')
        if (pos.chapter != null) out.push({ id: a.id, type: a.type, pct: Math.max(0, Math.min(100, ((pos.chapter + 0.5) / n) * 100)) })
      } catch { /* 忽略损坏的 position */ }
    }
  }
  return out
})

const annPageLabel = (ann) => {
  const ft = book.value?.file_type?.toLowerCase()
  return ft === 'pdf' ? `第 ${annLocOf(ann)} 页` : `第 ${annLocOf(ann) + 1} 章`
}

const isCurrentPageAnnotation = (ann) => {
  const ft = book.value?.file_type?.toLowerCase()
  return ft === 'pdf' ? annLocOf(ann) === currentPdfPage.value : annLocOf(ann) === currentChapter.value
}



// === 工具切换 ===
// 划线/删除线模式下光标改为笔（内联 SVG data URI），擦除模式保持普通光标
const DRAW_CURSOR = 'url("data:image/svg+xml,%3Csvg xmlns=%27http://www.w3.org/2000/svg%27 width=%2724%27 height=%2724%27 viewBox=%270 0 24 24%27%3E%3Cpath d=%27M3 17.25V21h3.75L17.8 9.94l-3.75-3.75L3 17.25zM20.71 7.04a1 1 0 0 0 0-1.41l-2.34-2.34a1 1 0 0 0-1.41 0l-1.83 1.83 3.75 3.75 1.83-1.83z%27 fill=%27white%27 stroke=%27black%27 stroke-width=%271.2%27/%3E%3C/svg%3E") 3 21, crosshair'

const applyDrawCursor = () => {
  const draw = !!currentTool.value && currentTool.value !== 'eraser'
  const cursor = draw ? DRAW_CURSOR : ''
  // 阅读区容器兜底（空白处）
  if (readerBody.value) readerBody.value.style.cursor = cursor
  // PDF 文本层为 JS 动态创建（scoped 样式不命中），直接设置 style
  document.querySelectorAll('.pdf-text-layer').forEach(l => { l.style.cursor = cursor })
  // EPUB 内容在 iframe 独立 document 中，需注入其 body
  const frames = epubScroll.value ? Object.values(epubIframes) : [currentIframe.value]
  frames.forEach(f => {
    if (f?.contentDocument?.body) f.contentDocument.body.style.cursor = cursor
  })
}



// === 朗读设置面板（工具栏"阅读"按钮） ===
const showTtsPanel = ref(false)
// 待选状态：点击"从选中位置开始阅读"后等待用户在页面上选中文字
const ttsPendingSelection = ref(false)



// === 图书加载 ===
const loadBookInfo = async () => {
  if (props.book) {
    book.value = props.book
    return
  }
  try {
    const { data } = await bookApi.getBook(props.id)
    book.value = data
  } catch (e) {
    ElMessage.error('加载图书失败')
  }
}

const loadChapters = async () => {
  try {
    // 统一目录接口：EPUB/KEPUB/PDF/MD 等，异步解析中则轮询等待
    for (let i = 0; i < 60; i++) {
      const { data } = await bookApi.getBookTOC(props.id)
      if (data.status === 'parsing') {
        await new Promise(r => setTimeout(r, 500))
        continue
      }
      chapters.value = data.chapters || []
      if (book.value) book.value.title = data.title || book.value.title
      return
    }
    // 轮询超时：回退旧内容接口（EPUB）
    const { data } = await bookApi.getBookContent(props.id, '')
    chapters.value = data.chapters || []
    if (book.value) book.value.title = data.title || book.value.title
  } catch (e) {
    console.warn('无法加载目录')
  }
}

const loadAnnotations = async () => {
  try {
    const { data } = await bookApi.listAnnotations(props.id)
    annotations.value = data || []
    // 批注为异步加载，可能晚于章节 iframe 渲染；加载完成后重新应用到已渲染页面，
    // 否则会出现"有记录但不回显"（含跨端创建的标注）
    nextTick(() => reapplyAllAnnotations())
  } catch (e) {
    console.warn('加载批注失败', e)
  }
}

// 将已加载的批注重新应用到当前已渲染页面（EPUB 各章节 iframe / PDF 各页），
// 用于批注异步加载晚于页面渲染的场景，保证回显与跨端一致
const reapplyAllAnnotations = () => {
  const ft = book.value?.file_type?.toLowerCase()
  if (ft === 'pdf') {
    Object.keys(pageOverlayEls).forEach(k => {
      const overlay = pageOverlayEls[k]
      if (overlay) overlay.querySelectorAll('.ann-overlay').forEach(el => el.remove())
    })
    pageInfos.value.forEach(p => { if (p.rendered) applyPdfHighlights(p.num) })
  } else {
    const frames = epubScroll.value ? Object.values(epubIframes) : [currentIframe.value]
    frames.forEach(f => {
      if (!f || !f.contentDocument) return
      const idx = epubScroll.value ? Number(f.dataset.chapterIndex) : (currentChapter.value || 0)
      applyEpubHighlights(idx, f)
      applyEpubRegions(idx, f)
    })
  }
}

// === 读后感 ===
const bookReviews = ref([])
const reviewExcerpt = (md) => markdownToText(md, 80)
const reviewTime = (t) => {
  if (!t) return ''
  const d = new Date(t)
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

const loadBookReviews = async () => {
  try {
    const { data } = await reviewApi.listBookReviews(props.id)
    bookReviews.value = (data && data.data) || []
  } catch (e) {
    bookReviews.value = []
  }
}

const openReviews = () => {
  showReviews.value = true
  loadBookReviews()
}

// 跳转全屏书写页（不做弹框）
const writeReviewInReader = () => {
  router.push({ path: '/reviews/editor', query: { book_id: props.id } })
}

const editReviewInReader = (r) => {
  router.push({ path: '/reviews/editor', query: { id: r.id, book_id: props.id } })
}

const removeReview = async (r) => {
  try {
    await ElMessageBox.confirm(
      `确定删除读后感「${r.title}」吗？此操作不可恢复。`,
      '删除确认',
      { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' }
    )
  } catch (e) {
    return
  }
  try {
    await reviewApi.deleteReview(r.id)
    ElMessage.success('已删除')
    loadBookReviews()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '删除失败')
  }
}

// === EPUB 渲染 ===
// 章节 TOC 下标 → 后端内容文件序号（EPUB 为 Contents/spine 序号）。
// TOC 条目数与内容文件数可能不一致（如含 Cover/结构性文件），直接拿数组下标请求会整体错位，
// 必须用 TOCItem.index（后端已映射好的 spine/Contents 序号）。
const chapterFileId = (ch) => chapters.value[ch]?.index ?? ch

// 重写 EPUB 内嵌图片 src：相对路径 → 带鉴权的资源接口
// （iframe srcdoc 内 <img> 无法携带 Authorization 头，也无法解析相对路径）
const rewriteEpubImages = (html, chapterIndex) => {
  const chapterInfo = chapters.value[chapterIndex]
  if (!chapterInfo?.path) return html
  return html.replace(/<img\s+([^>]*?)src=["']([^"']+)["']/gi, (m, attrs, src) => {
    if (/^(https?:|data:|blob:|#|javascript:)/i.test(src) || src.startsWith('/')) return m
    // 以章节文件为基准解析相对资源路径（正确处理 ./ ../ 及带锚点的引用）
    const resolved = new URL(src, 'http://local/' + chapterInfo.path)
    const resPath = decodeURIComponent(resolved.pathname.replace(/^\//, ''))
    const resUrl = bookApi.resourceUrl(props.id, resPath)
    return `<img ${attrs} src="${resUrl}"`
  })
}

// 注入主题/排版样式
// 拆分样式生成函数：buildEpubBaseStyle 返回 body/排版相关样式（依赖字号/主题/字体/边距/行间距），
// buildEpubScrollStyle 返回滚动模式专用样式（依赖主题色：纸张、侧栏）。
// style 标签带固定 id，updateEpubIframeStyles 可直接替换已加载 iframe 中的标签实现主题热更新，
// 不重建 DOM，避免文字位置/高度跳动。
const buildEpubBaseStyle = () => {
  const theme = themes.find(t => t.name === currentTheme.value) || themes[0]
  return `
      body {
        font-size: ${fontSize.value}px !important;
        line-height: ${lineHeight.value} !important;
        font-family: ${fontFamily.value} !important;
        padding: ${pageMargin.value}px !important;
        margin: 0 !important;
        color: ${theme.text} !important;
        background: ${theme.bodyBg} !important;
        -webkit-user-select: text !important;
        user-select: text !important;
      }
      img { max-width: 100%; height: auto; }
      * { max-width: 100%; }
      /* 分页防截断：块级元素整体换页，避免行/半行被列边界硬切；orphans/widows 防段首尾孤立行 */
      p, h1, h2, h3, h4, h5, h6, li, blockquote, figure, table, pre {
        break-inside: avoid;
        page-break-inside: avoid;
      }
      p { orphans: 2; widows: 2; }
      /* 参考图排版：段首无缩进、段落间距大、标题上下留白 */
      p { text-indent: 0 !important; margin: 0 0 0.95em 0 !important; }
      h1, h2, h3, h4, h5, h6 { margin: 1.3em 0 0.65em 0 !important; line-height: 1.35 !important; }
      /* 注意：不要在这里用 vh/vw 限制图片尺寸——滚动模式下 iframe 高度自适应（从 0 撑开），
         vh 会随 iframe 高度塌缩导致图片高度归零、scrollHeight 测量错误、内容重叠遮挡。
         大图限制（max-height）只对分页模式有效，滚动模式不使用 vh。 */
      .tts-speaking-block {
        background: linear-gradient(180deg, rgba(250, 204, 21, 0.35), rgba(251, 191, 36, 0.42)) !important;
        border-radius: 4px;
        transition: background 0.2s;
      }
  `
}

const buildEpubStyleHtml = (html) => {
  const style = `<style id="__reader_base_style">${buildEpubBaseStyle()}</style>`
  html = html.replace('</head>', style + '</head>')
  if (!html.includes('<head>')) {
    html = '<!DOCTYPE html><html><head>' + style + '</head><body>' + html + '</body></html>'
  }
  return html
}

// 实时更新所有已加载 EPUB iframe 的样式（用于主题色切换）：
// 主题色仅影响颜色/纸张背景/侧栏色，不改变排版尺寸，因此可原地替换 style 标签，
// DOM 结构与布局保持不变，文字位置不会跳动。
const updateEpubIframeStyles = () => {
  const baseStyleHtml = `<style id="__reader_base_style">${buildEpubBaseStyle()}</style>`
  const scrollStyleHtml = `<style id="__reader_scroll_style">${buildEpubScrollStyle()}</style>`
  Object.values(epubIframes).forEach(iframe => {
    try {
      const doc = iframe.contentDocument
      if (!doc || !doc.head) return
      // 基础样式：替换或新增
      let baseEl = doc.getElementById('__reader_base_style')
      if (baseEl) baseEl.textContent = buildEpubBaseStyle()
      else doc.head.insertAdjacentHTML('beforeend', baseStyleHtml)
      // 滚动样式：滚动模式才需要
      let scrollEl = doc.getElementById('__reader_scroll_style')
      if (epubScroll.value) {
        if (scrollEl) scrollEl.textContent = buildEpubScrollStyle()
        else doc.head.insertAdjacentHTML('beforeend', scrollStyleHtml)
      } else if (scrollEl) {
        scrollEl.remove()
      }
    } catch (e) {
      console.warn('更新 iframe 样式失败', e)
    }
  })
}

// 创建章节 iframe；mode: ''=整章一屏 / 'scroll'=连续滚动 / 'paged'=多列分页
// 注意：iframe 必须挂载到 DOM 后 srcdoc 才会开始加载，因此本函数仅创建并注册
// load 回调，由调用方负责 appendChild，再通过 waitIframeLoaded 等待加载完成。
const buildEpubIframe = (html, chapterIndex, mode = '') => {
  const scrollMode = mode === 'scroll'
  const iframe = document.createElement('iframe')
  iframe.style.width = '100%'
  iframe.style.border = 'none'
  iframe.style.height = scrollMode ? '0px' : '100%'
  iframe.style.display = 'block'
  if (scrollMode) iframe.classList.add('epub-scroll-iframe')
  iframe.srcdoc = html
  // 标记章节归属：连续滚动模式下划线/朗读需精确区分 iframe 对应的章节
  iframe.dataset.chapterIndex = String(chapterIndex)

  iframe.addEventListener('load', () => {
    try {
      // 桌面端用 mouseup；移动端长按选择依赖 selectionchange（handleIframeSelectionChange）
      iframe.contentDocument.addEventListener('mouseup', handleIframeSelection)
      iframe.contentDocument.addEventListener('selectionchange', handleIframeSelectionChange)
      // 移动端长按程序化选词，绕过系统识屏抢占
      initLongPressSelection(iframe.contentDocument, { iframe })
      // 画笔模式：iframe 内拖动画区域标记
      initEpubRegionDraw(iframe.contentDocument, iframe, chapterIndex)
      applyEpubHighlights(chapterIndex, iframe)
      applyEpubRegions(chapterIndex, iframe)
      // 新加载的章节 body 重新应用划线光标
      applyDrawCursor()
      if (scrollMode) resizeEpubScrollIframe(iframe)
    } catch (e) {
      console.warn('无法监听 iframe 选择事件', e)
    }
  })
  return iframe
}

// 等待 iframe 加载完成（须先挂载到 DOM）
const waitIframeLoaded = (iframe) => new Promise(resolve => {
  if (iframe.contentDocument?.readyState === 'complete') resolve()
  else iframe.addEventListener('load', resolve, { once: true })
})

// 滚动模式下按内容实际高度撑开 iframe（图片加载完成后重新测量）
const resizeEpubScrollIframe = (iframe) => {
  try {
    const doc = iframe.contentDocument
    if (!doc || !doc.body) return
    const measure = () => {
      iframe.style.height = doc.body.scrollHeight + 'px'
    }
    measure()
    if (doc.images && doc.images.length) {
      const images = Array.from(doc.images)
      let pending = images.length
      const done = () => { pending--; if (pending <= 0) measure() }
      images.forEach(img => {
        if (img.complete) done()
        else {
          img.addEventListener('load', done)
          img.addEventListener('error', done)
        }
      })
    }
  } catch (e) {
    console.warn('resize iframe 失败', e)
  }
}

// === EPUB 连续滚动（分页阅读已移除，仅滚动模式） ===

// 连续滚动模式：正文包装进一张"纸张"（居中、白底/随主题、极轻阴影、近直角），
// 纸张左右各有一条固定"侧栏"（参考图淡紫灰装饰条，不随滚动移动、随主题变色），
// 文字始终位于纸张内（宽度明确避让侧栏），不会溢出展示区。
// 章节纸张上下无间隙（body padding 0），章节连续排列；iframe 高度由 body.scrollHeight 自适应。
// 样式与颜色相关部分抽出到 buildEpubScrollStyle，便于主题热更新（updateEpubIframeStyles）。
const buildEpubScrollStyle = () => {
  const paperBg = getEpubPaperBg(currentTheme.value)
  const sidebarBg = getEpubSidebarBg(currentTheme.value)
  return `
    body { background: transparent !important; margin: 0 !important; padding: 0 !important; position: relative; -webkit-touch-callout: none; user-select: text; -webkit-user-select: text; }
    /* 区域标注（画笔模式拖动画出的矩形标记）与画笔预览 */
    .epub-region, .region-preview { position: absolute; pointer-events: auto; cursor: pointer; border-radius: 2px; }
    .epub-region.ann-highlight, .region-preview.ann-highlight { background-color: var(--ann-color); opacity: 0.4; }
    .epub-region.ann-underline, .region-preview.ann-underline { border-bottom: 2px solid var(--ann-color); background: transparent; }
    .epub-region.ann-wavy, .region-preview.ann-wavy { border-bottom: 2px dashed var(--ann-color); background: transparent; }
    .epub-region.ann-strikethrough, .region-preview.ann-strikethrough { background: linear-gradient(transparent 45%, var(--ann-color) 45%, var(--ann-color) 55%, transparent 55%); opacity: 0.7; }
    .region-preview { pointer-events: none !important; z-index: 60; outline: 1px dashed var(--ann-color); }
    /* 侧栏：fixed 于视口两侧，宽度 min(12%, 220px) 确保窄屏安全（纸张区始终 ≥ sheet 宽度） */
    body::before, body::after { content: ""; position: fixed; top: 0; bottom: 0; width: min(12%, 220px); background: ${sidebarBg}; z-index: 998; }
    body::before { left: 0; box-shadow: inset -16px 0 20px -18px rgba(0,0,0,.35); }
    body::after { right: 0; box-shadow: inset 16px 0 20px -18px rgba(0,0,0,.35); }
    /* 纸张：居中，宽度避让两侧栏（calc 100% - 24%），文字绝不超出纸张/侧栏 */
    #__sheet { box-sizing: border-box; width: calc(100% - 24%); margin: 0 auto; background: ${paperBg}; box-shadow: 0 1px 3px rgba(0,0,0,.06); border-radius: 0; padding: 48px 6% 64px; overflow-wrap: break-word; word-break: break-word; }
    /* 手机端：去掉侧栏，纸张铺满屏幕宽度，正文保留舒适边距（不贴边、有呼吸感） */
    @media (max-width: 768px) {
      body::before, body::after { display: none !important; }
      #__sheet { width: 100% !important; padding: 16px max(16px, 5%) 32px !important; box-shadow: none !important; }
    }
  `
}

const buildEpubScrollHtml = (html, paperBg = '#ffffff', sidebarBg = '#e7eaf2') => {
  const style = `<style id="__reader_scroll_style">${buildEpubScrollStyle()}</style>`
  const script = `<script>
(function () {
  var body = document.body
  if (document.getElementById('__sheet')) return
  var sheet = document.createElement('div'); sheet.id = '__sheet'
  while (body.firstChild) sheet.appendChild(body.firstChild)
  body.appendChild(sheet)
})();
<\/script>`
  if (html.includes('</body>')) return html.replace('</body>', style + script + '</body>')
  return html + style + script
}

// === EPUB 连续滚动 ===
const epubIframes = {}               // chapter index -> iframe
const fileIdToIframe = {}            // 内容文件序号 -> iframe（多目录条目指向同一文件时共享，避免重复渲染）
const epubLoadedChapters = []        // 已加载章节索引（有序）
let epubScrollBusy = false
let epubScrollRaf = 0

const appendEpubChapter = async (index) => {
  if (epubIframes[index] || epubScrollBusy) return
  const container = document.getElementById('epub-viewer')
  if (!container) return
  const fileId = chapterFileId(index)
  // 去重：多个目录条目指向同一内容文件（如 EPUB 目录"第1章 + 1.1~1.9"全部指向同一 spine 文件）时，
  // 不重复创建 iframe，仅建立 章节索引 -> 既有 iframe 的映射，并补应用该章节的批注/区域
  const shared = fileIdToIframe[fileId]
  if (shared) {
    epubIframes[index] = shared
    epubLoadedChapters.push(index)
    epubLoadedChapters.sort((a, b) => a - b)
    const applyAnn = () => {
      applyEpubHighlights(index, shared)
      applyEpubRegions(index, shared)
    }
    if (shared.contentDocument?.readyState === 'complete') applyAnn()
    else shared.addEventListener('load', applyAnn, { once: true })
    return
  }
  try {
    epubScrollBusy = true
    const { data } = await bookApi.getBookContent(props.id, String(fileId))
    let html = rewriteEpubImages(data, index)
    html = buildEpubStyleHtml(html)
    html = buildEpubScrollHtml(html, getEpubPaperBg(currentTheme.value), getEpubSidebarBg(currentTheme.value))
    const iframe = buildEpubIframe(html, index, 'scroll')
    // 按章节索引顺序插入 DOM：插到第一个索引更大的已加载 iframe 之前（向上补章时保证文档顺序正确）
    let insertBeforeNode = null
    for (const idx of epubLoadedChapters) {
      if (idx > index) { insertBeforeNode = epubIframes[idx]; break }
    }
    const beforeScroll = container.scrollTop
    if (insertBeforeNode) container.insertBefore(iframe, insertBeforeNode)
    else container.appendChild(iframe)
    epubIframes[index] = iframe
    fileIdToIframe[fileId] = iframe
    epubLoadedChapters.push(index)
    epubLoadedChapters.sort((a, b) => a - b)
    await waitIframeLoaded(iframe)
    // 新章节插入在视口之前（向上补章）时，补偿滚动偏移保持视口内容不跳动；
    // 等一帧让 iframe 高度被 resizeEpubScrollIframe 更新后再补偿，减少误差
    if (insertBeforeNode) {
      await new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)))
      container.scrollTop = beforeScroll + (iframe.offsetHeight || 0)
    }
  } catch (e) {
    console.error('加载章节失败', index, e)
  } finally {
    epubScrollBusy = false
  }
}

// 确保 target 章节附近（前后各 radius 章）均已加载，用于目录/标记跳转（任意方向都能补齐）
const ensureEpubChaptersAround = async (target, radius = 2) => {
  const n = chapters.value.length
  const from = Math.max(0, target - radius)
  const to = Math.min(n - 1, target + radius)
  for (let i = from; i <= to; i++) {
    await appendEpubChapter(i)
  }
}

// 等待 iframe 内所有 <img> 加载完成（兜底超时 5s,避免图片卡死导致跳转一直等待）
const waitForIframeImages = (iframe) => {
  return new Promise(resolve => {
    try {
      const doc = iframe.contentDocument
      if (!doc) { resolve(); return }
      const images = Array.from(doc.images || [])
      if (images.length === 0) { resolve(); return }
      let pending = images.length
      let settled = false
      const done = () => {
        if (settled) return
        pending--
        if (pending <= 0) { settled = true; resolve() }
      }
      images.forEach(img => {
        if (img.complete) done()
        else {
          img.addEventListener('load', done, { once: true })
          img.addEventListener('error', done, { once: true })
        }
      })
      setTimeout(() => { if (!settled) { settled = true; resolve() } }, 5000)
    } catch (e) {
      resolve()
    }
  })
}

// 等待从 startIdx 到 endIdx（含）所有已加载章节的图片加载完成。
// 目标章节 iframe.offsetTop 依赖于前面所有已加载章节的累计高度，
// 只有前面章节高度稳定后，目标章节 offsetTop 才稳定；因此需要一并等待。
const waitForEpubImagesUpTo = async (endIdx) => {
  for (const idx of epubLoadedChapters) {
    if (idx > endIdx) break
    const f = epubIframes[idx]
    if (f) {
      await waitForIframeImages(f)
      resizeEpubScrollIframe(f)
    }
  }
}

// 取得目录项对应的锚点（EPUB 内容文件内的 id，由后端从 NCX/NAV 的 href 中解析）。
// 多个目录项共享同一 spine 文件时，anchor 是区分"章/节"位置的唯一线索。
const chapterAnchor = (index) => {
  const ch = chapters.value[index]
  return ch?.anchor || ''
}

// 在 iframe 文档内寻找带指定 id 的元素，并返回其到 iframe 顶部的偏移。
// 容错：CSS 选择器转义、找不到则返回 null。
const findAnchorOffsetInIframe = (iframe, anchorId) => {
  if (!anchorId) return null
  try {
    const doc = iframe.contentDocument
    if (!doc) return null
    // 优先按 id 精确匹配；EPUB 中 id 通常全局唯一，直接 getElementById 即可
    let el = doc.getElementById(anchorId)
    // 容错：少数 EPUB 使用 name 属性，浏览器定位也可能命中带 name 的 a 标签
    if (!el) el = doc.querySelector(`a[name="${anchorId}"]`)
    if (!el) return null
    // element.getBoundingClientRect().top 是相对视口的，需要加上 iframe.contentWindow.scrollY
    // 才能得到"到 iframe 文档顶部的偏移"。但为兼容滚动模式 iframe 自身高度自适应（不是真滚动条）
    // 这里用 element.offsetTop 配合祖先 offsetParent 累加，更稳妥
    let top = 0
    let cur = el
    while (cur && cur !== doc.body) {
      top += cur.offsetTop || 0
      cur = cur.offsetParent
    }
    return top
  } catch (e) {
    return null
  }
}

const scrollEpubToChapter = async (index, frac = 0) => {
  const iframe = epubIframes[index]
  const container = document.getElementById('epub-viewer')
  if (!iframe || !container) return
  // 等待目标章节及之前所有已加载章节的图片加载完成（iframe.offsetTop 取决于前面累计高度）
  await waitForEpubImagesUpTo(index)
  // 再等 2 帧让浏览器完成布局（确保所有 height 已应用）
  await new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)))
  // 处理 anchor：多个目录项共享同一 spine 时，靠 anchor 区分章/节具体位置。
  // 有 anchor：在 iframe 内滚到该元素；外部容器只需对齐到 iframe 顶部。
  const anchor = chapterAnchor(index)
  let top = iframe.offsetTop
  if (anchor) {
    const innerOffset = findAnchorOffsetInIframe(iframe, anchor)
    if (innerOffset != null) {
      // iframe 自身不会滚动（高度自适应），把锚点偏移作用到外部容器的 scrollTop
      top = iframe.offsetTop + innerOffset
      // 锚点跳转视为"章节起点"，不再叠加 frac
      frac = 0
    }
    // 找不到 anchor 元素时回退到原行为（滚到 iframe 顶部）
  }
  // 章节内偏移（0~0.999）：恢复阅读时定位到之前的滚动位置
  container.scrollTop = top + Math.min(Math.max(frac, 0), 0.999) * iframe.offsetHeight
  currentChapter.value = index
  if (epubIframes[index]) currentIframe.value = epubIframes[index]
  updateProgressFromChapter()
  if (frac > 0 && chapters.value.length > 1) {
    currentProgress.value = ((index + Math.min(frac, 0.999)) / chapters.value.length) * 100
  }
  scheduleProgressSave()
}

// 滚动模式：基于容器的真实 scrollTop/scrollHeight 比例计算进度，
// 准确反映已读内容占比（不受章节长度不均影响）。
const saveScrollOffset = () => {
  const container = document.getElementById('epub-viewer')
  if (!container) return
  const total = container.scrollHeight - container.clientHeight
  if (total <= 0) return
  const ratio = container.scrollTop / total
  currentProgress.value = Math.max(0, Math.min(100, ratio * 100))
}

// 滚动状态联动：确定当前阅读章节、保存进度、滚动到底自动加载下一章
const updateEpubScrollState = () => {
  const container = document.getElementById('epub-viewer')
  if (!container) return
  if (!epubLoadedChapters.length) return
  const viewTop = container.scrollTop + 80
  let cur = epubLoadedChapters[0]
  // 同一内容文件可能被多个目录条目共享（同 iframe），只按首个条目判定当前章节
  const seenIframes = new Set()
  for (const idx of epubLoadedChapters) {
    const iframe = epubIframes[idx]
    if (!iframe || seenIframes.has(iframe)) continue
    seenIframes.add(iframe)
    if (iframe.offsetTop <= viewTop) cur = idx
    else break
  }
  const chChanged = cur !== currentChapter.value
  if (chChanged) {
    currentChapter.value = cur
    currentIframe.value = epubIframes[cur] || null
    updateProgressFromChapter()
  }
  // 章内滚动也要保存偏移（节流 2s）；跨章立即保存
  saveScrollOffset()
  scheduleProgressSave(chChanged)
}

const maybeLoadNextEpubChapter = async () => {
  const container = document.getElementById('epub-viewer')
  if (!container || epubScrollBusy) return
  const last = epubLoadedChapters.length ? epubLoadedChapters[epubLoadedChapters.length - 1] : -1
  if (last >= chapters.value.length - 1) return
  if (container.scrollTop + container.clientHeight >= container.scrollHeight - 500) {
    await appendEpubChapter(last + 1)
    // 追加后视口保持连续（新内容自然出现在底部，无需补偿）
  }
}

// 滚动接近顶部时向上补章：解决从中间章节恢复时前面章节"丢失"、无法向前连续阅读的问题
const maybeLoadPrevEpubChapter = async () => {
  const container = document.getElementById('epub-viewer')
  if (!container || epubScrollBusy) return
  if (!epubLoadedChapters.length) return
  const first = epubLoadedChapters[0]
  if (first <= 0) return
  if (container.scrollTop <= 300) {
    for (let i = first - 1; i >= Math.max(0, first - 2); i--) {
      await appendEpubChapter(i)
    }
  }
}

const onEpubScroll = () => {
  if (epubScrollRaf) return
  epubScrollRaf = requestAnimationFrame(() => {
    epubScrollRaf = 0
    if (!epubScroll.value) return
    maybeLoadNextEpubChapter()
    maybeLoadPrevEpubChapter()
    updateEpubScrollState()
  })
}

// 初始化/重建滚动容器：清空后从起始章附近开始顺序加载
// startPos 支持浮点章节位置（整数部分=章节，小数部分=章内滚动比例）
const initEpubScroll = async (startPos) => {
  const container = document.getElementById('epub-viewer')
  if (!container) return
  container.innerHTML = ''
  Object.keys(epubIframes).forEach(k => delete epubIframes[k])
  Object.keys(fileIdToIframe).forEach(k => delete fileIdToIframe[k])
  epubLoadedChapters.length = 0
  epubScrollBusy = false

  const n = chapters.value.length
  // 支持两种入参：
  //  - 浮点章节位置（旧接口，向后兼容）
  //  - 0~100 的进度百分比（与 saveScrollOffset 公式配套）
  let targetPct = 0
  if (n > 1) {
    if (startPos > n - 1) {
      // 进度百分比模式
      targetPct = Math.max(0, Math.min(startPos, 100)) / 100
    } else {
      // 浮点章节位置模式：换算成近似百分比
      targetPct = startPos / (n - 1)
    }
  }
  // 起始章节：基于百分比推算（精度足够，无需加载全部章节）
  const start = Math.min(Math.floor(targetPct * n), n - 1)
  // 预加载起始章前后章节，保证初始即可向前/向后连续滚动
  const preStart = Math.max(0, start - 2)
  const preEnd = Math.min(n - 1, start + 2)
  for (let i = preStart; i <= preEnd; i++) {
    await appendEpubChapter(i)
  }
  // 滚到目标百分比对应的真实位置（与 saveScrollOffset 同源）
  const total = container.scrollHeight - container.clientHeight
  if (total > 0) {
    container.scrollTop = total * targetPct
  }
  // 同步当前章节
  if (epubLoadedChapters.length) {
    const viewTop = container.scrollTop + 80
    let cur = epubLoadedChapters[0]
    for (const idx of epubLoadedChapters) {
      const iframe = epubIframes[idx]
      if (!iframe) continue
      if (iframe.offsetTop <= viewTop) cur = idx
      else break
    }
    currentChapter.value = cur
    currentIframe.value = epubIframes[cur] || null
  }
}



const renderChapter = async (index) => {
  if (!book.value) return
  const ft = book.value.file_type?.toLowerCase()
  if (ft !== 'epub' && ft !== 'kepub' && ft !== 'azw3') return

  try {
    // 滚动模式：确保目标章节前后均已加载并滚动定位，不重建容器
    await ensureEpubChaptersAround(index)
    await scrollEpubToChapter(index)
    updateEpubScrollState()
  } catch (e) {
    console.error('渲染章节失败', e)
  }
}

const updateProgressFromChapter = () => {
  if (chapters.value.length > 0) {
    currentProgress.value = (currentChapter.value / chapters.value.length) * 100
  }
}

// === PDF 渲染 ===
// 窄屏（手机）判定与 PDF 自适应宽度相关状态
const isNarrow = () => window.innerWidth <= 768
let mobileFitActive = false // 当前是否处于手机端“适配宽度”状态（用户手动缩放后置否）
let pdfBasePageWidth = 0    // scale=1 时的首页宽度，用于 resize 时重算适配缩放

// 手机端 resize / 旋屏时，若仍处于适配宽度状态，按新容器宽度重算缩放并重渲染
const onPdfResize = () => {
  if (!pdfDoc || !isNarrow() || !mobileFitActive) return
  const container = pdfContainer.value
  if (!container || !pdfBasePageWidth) return
  const fit = (container.clientWidth - 24) / pdfBasePageWidth
  applyZoom(Math.max(0.5, Math.min(3.0, fit)))
}

const loadPDF = async () => {
  try {
    const url = bookApi.fileUrl(book.value.id)
    const loadingTask = pdfjsLib.getDocument({ url })
    pdfDoc = await loadingTask.promise
    totalPdfPages.value = pdfDoc.numPages

    // 用第一页尺寸估算全部页面（渲染时按实际尺寸修正），避免逐页取 viewport 导致加载缓慢
    pdfLoading.value = true
    const firstPage = await pdfDoc.getPage(1)
    // 窄屏强制回退单页（之前可能保存过双页偏好）
    if (window.innerWidth <= 768 && pdfLayout.value === 'two') {
      pdfLayout.value = 'single'
    }
    // 自动适配缩放：
    // - 双页平铺（宽屏）：按两页并排宽度适配
    // - 单页：默认按容器宽度适配，避免页面过宽需横向滚动（用户未手动设置过缩放时才自动适配）
    let scale = pdfScale.value
    const container = pdfContainer.value
    if (container) {
      const baseVp = firstPage.getViewport({ scale: 1 })
      pdfBasePageWidth = baseVp.width
      if (pdfLayout.value === 'two' && !isNarrow()) {
        // 双页平铺：两页并排（含 14px 间距）正好铺满容器宽度，留 24px 边距
        scale = Math.max(0.5, Math.min(3.0, (container.clientWidth - 14 - 24) / (2 * baseVp.width)))
        mobileFitActive = false
      } else {
        // 单页：按容器宽度适配，确保展示宽度正确、不超出视口
        const fit = (container.clientWidth - 24) / baseVp.width
        const clamped = Math.max(0.5, Math.min(3.0, fit))
        // 手机端（窄屏）始终按容器宽度适配，忽略已保存的缩放偏好，避免页面超出视口；
        // 桌面端则尊重用户曾手动设置过的缩放，否则使用适配宽度
        if (isNarrow()) {
          scale = clamped
          mobileFitActive = true
        } else {
          scale = savedPrefs.pdfScale != null ? pdfScale.value : clamped
          mobileFitActive = false
        }
      }
      pdfScale.value = scale
    }
    const vp = firstPage.getViewport({ scale })
    const infos = []
    for (let i = 1; i <= totalPdfPages.value; i++) {
      infos.push({ num: i, width: vp.width, height: vp.height, rendered: false })
    }
    pageInfos.value = infos
    pdfLoading.value = false

    // 确定起始页
    let startPage = 1
    if (book.value?.current_page && book.value.current_page > 0) {
      startPage = Math.min(book.value.current_page, totalPdfPages.value)
    } else if (book.value?.progress && book.value.progress > 0) {
      startPage = Math.max(1, Math.min(totalPdfPages.value, Math.round((book.value.progress / 100) * totalPdfPages.value)))
    }

    // 初始渲染：起始页及前后各1页
    const toRender = new Set()
    for (let i = Math.max(1, startPage - 1); i <= Math.min(totalPdfPages.value, startPage + 1); i++) {
      toRender.add(i)
    }
    pageInfos.value.forEach(p => { if (toRender.has(p.num)) p.rendered = true })

    nextTick(() => {
      toRender.forEach(num => renderPdfPage(num))
      setupPageObserver()
      scrollToPdfPage(startPage)
      pdfContainer.value?.addEventListener('scroll', throttledScroll)
    })
  } catch (e) {
    console.error('PDF 加载失败', e)
    ElMessage.error('PDF 加载失败')
    pdfLoading.value = false
  }
}

// === 按需渲染单页（带任务取消，防止并发渲染同一 canvas 报错） ===
const pageRenderTasks = {}
const pageRenderGen = {}
const renderingCount = ref(0)

const renderPdfPage = async (pageNum) => {
  const canvas = pageCanvasEls[pageNum]
  const textLayer = pageTextLayerEls[pageNum]
  const overlay = pageOverlayEls[pageNum]
  if (!pdfDoc || !canvas || !textLayer || !overlay) return

  const gen = (pageRenderGen[pageNum] || 0) + 1
  pageRenderGen[pageNum] = gen
  if (pageRenderTasks[pageNum]) {
    try { pageRenderTasks[pageNum].cancel() } catch (e) {}
  }

  renderingCount.value++
  try {
    const page = await pdfDoc.getPage(pageNum)
    if (pageRenderGen[pageNum] !== gen) return
    const scale = pdfScale.value
    const viewport = page.getViewport({ scale })

    // 更新页面尺寸（修正初始估算）
    const info = pageInfos.value.find(p => p.num === pageNum)
    if (info) {
      info.width = viewport.width
      info.height = viewport.height
    }

    const ctx = canvas.getContext('2d')
    const dpr = window.devicePixelRatio || 1
    canvas.width = Math.floor(viewport.width * dpr)
    canvas.height = Math.floor(viewport.height * dpr)
    canvas.style.width = Math.floor(viewport.width) + 'px'
    canvas.style.height = Math.floor(viewport.height) + 'px'

    const renderViewport = page.getViewport({ scale: scale * dpr })
    const renderTask = page.render({ canvasContext: ctx, viewport: renderViewport })
    pageRenderTasks[pageNum] = renderTask
    await renderTask.promise
    if (pageRenderGen[pageNum] !== gen) return

    // 创建文本层
    textLayer.innerHTML = ''
    pdfTextIndex.delete(pageNum) // 文本层已重建，作废该页朗读高亮索引缓存
    textLayer.style.width = viewport.width + 'px'
    textLayer.style.height = viewport.height + 'px'

    const textContent = await page.getTextContent()
    if (pageRenderGen[pageNum] !== gen) return
    createTextLayer(textContent, textLayer, viewport)

    // 桌面端用 mouseup 捕获选择；移动端长按选择依赖 document selectionchange（handleDocSelectionChange）
    if (!isTouchDevice()) {
      textLayer.removeEventListener('mouseup', handlePdfSelection)
      textLayer.addEventListener('mouseup', handlePdfSelection)
    }
    // 新渲染的文本层重新应用划线光标
    applyDrawCursor()

    // 应用已保存的划线高亮（不闪屏）
    overlay.innerHTML = ''
    applyPdfHighlights(pageNum)
  } catch (e) {
    if (e?.name === 'RenderingCancelledException') return
    console.error('PDF 渲染失败', pageNum, e)
  } finally {
    renderingCount.value--
  }
}

const createTextLayer = (textContent, container, viewport) => {
  const items = textContent.items
  items.forEach(item => {
    if (!item.str || !item.str.trim()) return

    const tx = pdfjsLib.Util.transform(viewport.transform, item.transform)
    const fontHeight = Math.hypot(tx[2], tx[3])
    const fontWidth = Math.hypot(tx[0], tx[1])

    const span = document.createElement('span')
    span.textContent = item.str
    span.style.position = 'absolute'
    span.style.left = `${tx[4]}px`
    span.style.top = `${tx[5] - fontHeight}px`
    span.style.fontSize = `${fontHeight}px`
    span.style.fontFamily = 'sans-serif'
    span.style.whiteSpace = 'pre'
    span.style.userSelect = 'text'
    span.style.color = 'transparent'
    span.style.cursor = 'text'

    if (Math.abs(fontWidth - fontHeight) > 0.5) {
      span.style.transform = `scaleX(${fontWidth / fontHeight})`
      span.style.transformOrigin = 'left bottom'
    }

    container.appendChild(span)
  })
}

// === PDF 划线应用（不闪屏）===
const applyPdfHighlights = (pageNum) => {
  const overlay = pageOverlayEls[pageNum]
  if (!overlay) return
  const pageAnnotations = annotations.value.filter(a => {
    try {
      const pos = JSON.parse(a.position || '{}')
      return pos.page === pageNum && a.type === 'highlight'
    } catch { return false }
  })

  pageAnnotations.forEach(ann => {
    renderSinglePdfAnnotation(ann)
  })
}

const renderSinglePdfAnnotation = (ann) => {
  try {
    const pos = JSON.parse(ann.position || '{}')
    if (!pos.rect || !pos.page) return

    const overlay = pageOverlayEls[pos.page]
    if (!overlay) return

    const div = document.createElement('div')
    const style = ann.style || 'highlight'
    const color = ann.color || '#ffeb3b'
    const scaleRatio = pdfScale.value / (pos.scale || pdfScale.value)

    div.className = `ann-overlay ann-${style}`
    div.dataset.annId = ann.id
    div.style.left = `${pos.rect.left * scaleRatio}px`
    div.style.top = `${pos.rect.top * scaleRatio}px`
    div.style.width = `${pos.rect.width * scaleRatio}px`
    div.style.height = `${pos.rect.height * scaleRatio}px`
    div.style.setProperty('--ann-color', color)
    div.addEventListener('click', (e) => {
      e.stopPropagation()
      if (currentTool.value === 'eraser') {
        deleteAnnotation(ann.id)
      }
    })

    if (style === 'wavy') {
      const svg = `data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='20' height='4' viewBox='0 0 20 4'%3E%3Cpath d='M0,2 Q5,0 10,2 T20,2' fill='none' stroke='${encodeURIComponent(color)}' stroke-width='1.5'/%3E%3C/svg%3E`
      div.style.backgroundImage = `url("${svg}")`
      div.style.backgroundRepeat = 'repeat-x'
      div.style.backgroundPosition = 'bottom'
      div.style.backgroundSize = '20px 4px'
    }

    overlay.appendChild(div)
  } catch (e) {}
}

// === EPUB 划线应用 ===
const applyEpubHighlights = (chapterIndex, frame) => {
  // 连续滚动模式：只应用到对应章节 iframe（多个 iframe 并存）
  const iframe = frame || (epubScroll.value ? (epubIframes[chapterIndex] || null) : currentIframe.value)
  if (!iframe || !iframe.contentDocument) return

  const chapterAnnotations = annotations.value.filter(a => {
    try {
      const pos = JSON.parse(a.position || '{}')
      // 区域标记按坐标渲染（applyEpubRegions），不走文本匹配
      return !pos.region && pos.chapter === chapterIndex && a.type === 'highlight'
    } catch { return false }
  })

  chapterAnnotations.forEach(ann => {
    highlightTextInNode(iframe.contentDocument.body, ann.content, ann.color || '#ffeb3b', ann.style || 'highlight', ann.id)
  })
}

const highlightTextInNode = (node, text, color, style, annId) => {
  if (node.nodeType === Node.TEXT_NODE) {
    const index = node.textContent.indexOf(text)
    if (index >= 0 && node.parentElement && !node.parentElement.dataset.highlight) {
      const span = document.createElement('span')
      span.dataset.highlight = 'true'
      span.dataset.annId = annId
      span.dataset.style = style
      span.addEventListener('click', (e) => {
        e.stopPropagation()
        if (currentTool.value === 'eraser') {
          deleteAnnotation(annId)
        }
      })

      switch (style) {
        case 'underline':
          span.style.textDecoration = 'underline solid ' + color + ' 2px'
          break
        case 'wavy':
          span.style.textDecoration = 'underline wavy ' + color + ' 1.5px'
          break
        case 'strikethrough':
          span.style.textDecoration = 'line-through solid ' + color + ' 2px'
          break
        case 'highlight':
        default:
          span.style.backgroundColor = color
          span.style.opacity = '0.5'
          break
      }

      const after = node.splitText(index + text.length)
      const mid = node.splitText(index)
      span.appendChild(mid)
      node.parentNode.insertBefore(span, after)
      return true
    }
  } else if (node.nodeType === Node.ELEMENT_NODE && !['SCRIPT', 'STYLE'].includes(node.tagName) && !node.dataset.highlight) {
    for (let child of Array.from(node.childNodes)) {
      if (highlightTextInNode(child, text, color, style, annId)) return true
    }
  }
  return false
}

// === 文本选择捕获 ===
// 移动端长按选择后 mouseup 合成事件时序不可靠，需用 selectionchange 检测
const isTouchDevice = () => window.matchMedia('(pointer: coarse)').matches || window.innerWidth <= 768
let selectionTimer = null
let lastSelectionText = ''

// === 移动端文本选择：交由浏览器原生选择手柄接管 ===
// 早期实现会在长按时机程序化选中整个文本节点并 preventDefault，导致只能整段/整词选中、
// 无法自由拖选连续文字。现改为完全放行原生选择：长按文字由系统弹出选择手柄，用户可自由
// 拖拽扩展选区；选区变化由 selectionchange（handleIframeSelectionChange /
// handleDocSelectionChange）统一捕获，再弹出划选操作条（复制 / 划线 / 朗读）。
const selectMode = ref(false)

// 绑定移动端文本选择：不再拦截默认行为、不再程序化整节点选中，原生选择即可自由拖选。
// 选择结果由已在 iframe load / 全局 document 注册的 selectionchange 监听统一处理。
const initLongPressSelection = (doc, opts = {}) => {
  if (!doc || !isTouchDevice()) return
  // 放行原生长按选择：不 preventDefault、不整节点选中，原生手柄接管自由拖选连续文字。
}

// 移动端选择模式：由侧栏划线/笔记工具自动进入（原顶部"选择"按钮已移除）。
// 进入后屏蔽系统识屏/长按菜单，长按文字选中 → 弹出划选操作条（复制/划线/朗读）。
const syncTouchSelectMode = () => {
  // 移动端不支持"先选划线工具再选中文字"，仅允许选中文字后弹操作条处理
  if (isTouchDevice()) return
  const needSelect = isDrawToolActive() || noteMode.value
  if (needSelect) {
    if (!selectMode.value) selectMode.value = true
  } else if (selectMode.value) {
    selectMode.value = false
    cancelHighlight()
  }
}

// 退出选择模式并恢复滑动翻页（提示条"取消"入口）
const exitSelectMode = () => {
  selectMode.value = false
  if (currentTool.value) currentTool.value = null
  if (noteMode.value) noteMode.value = false
  cancelHighlight()
  applyDrawCursor()
}

// 划选操作条：复制选中文字（复制能力从顶部按钮迁移到划线/选中工具上）
const copySelectedText = async () => {
  lockCollapseHide()
  const text = selectedText.value
  if (!text) return
  const ok = await copyText(text)
  if (ok) ElMessage.success('已复制')
  else ElMessage.error('复制失败')
}

// 划选操作条：关闭（保留工具激活，可继续选择其他文字）
const dismissMarkToolbar = () => {
  showHighlightToolbar.value = false
  selectedText.value = ''
  window.getSelection()?.removeAllRanges()
  const iw = lastSelectionFrame?.contentWindow
  iw?.getSelection()?.removeAllRanges()
}

// 划线工具对应操作条按钮文案
const toolLabelMap = { highlight: '底色', underline: '横线', wavy: '波浪', strikethrough: '删除' }
const markToolLabel = computed(() => {
  const t = currentTool.value
  return t && toolLabelMap[t] ? `加${toolLabelMap[t]}` : '划线'
})

// 移动端"先选文字后处理"：选中文字后点"笔记"→ 直接针对当前选区弹出笔记框
const openNoteForSelection = () => {
  lockCollapseHide()
  noteMode.value = true
  showHighlightToolbar.value = true
}

const fallbackCopy = (text, done) => {
  const ta = document.createElement('textarea')
  ta.value = text
  ta.style.cssText = 'position:fixed;opacity:0;left:-9999px;top:0'
  document.body.appendChild(ta)
  ta.focus()
  ta.select()
  try {
    document.execCommand('copy')
    done()
  } catch (e) {
    ElMessage.error('复制失败')
  }
  document.body.removeChild(ta)
}

// === 画笔模式：直接在页面上拖动画矩形区域（PDF 页面 / EPUB iframe） ===
// 画笔工具（底色/横线/波浪/删除）激活时，按住拖动即可圈出指定区域生成标记，
// 无需先选中文字（适合圈图表、图片、整段区域），移动端同样支持
let regionDrawing = null // { kind, doc, host, pageNum?, chapter?, iframe?, startX, startY, el, style, color, prevTouchAction }
const REGION_MIN_SIZE = 10

const isDrawToolActive = () => {
  const t = currentTool.value
  return !!t && t !== 'eraser'
}

const createRegionPreview = (doc, host, style, color) => {
  const el = doc.createElement('div')
  el.className = 'region-preview ann-' + style
  el.style.setProperty('--ann-color', color)
  host.appendChild(el)
  return el
}

const startRegionDraw = (kind, doc, host, e, extra = {}) => {
  if (regionDrawing || !isDrawToolActive()) return
  const rect = host.getBoundingClientRect()
  const style = currentTool.value || 'highlight'
  const color = currentColor.value || '#ffeb3b'
  const el = createRegionPreview(doc, host, style, color)
  regionDrawing = {
    kind, doc, host, startX: e.clientX - rect.left, startY: e.clientY - rect.top,
    el, style, color, prevTouchAction: host.style.touchAction, ...extra
  }
  // 绘制期间锁定触摸滚动，避免手指拖动时页面跟着滚
  host.style.touchAction = 'none'
  document.addEventListener('pointermove', handleRegionMove)
  document.addEventListener('pointerup', handleRegionUp)
  document.addEventListener('pointercancel', handleRegionCancel)
}

const handleRegionMove = (e) => {
  const d = regionDrawing
  if (!d) return
  const rect = d.host.getBoundingClientRect()
  const x = e.clientX - rect.left
  const y = e.clientY - rect.top
  const left = Math.min(d.startX, x)
  const top = Math.min(d.startY, y)
  const w = Math.abs(x - d.startX)
  const h = Math.abs(y - d.startY)
  d.el.style.left = left + 'px'
  d.el.style.top = top + 'px'
  d.el.style.width = w + 'px'
  d.el.style.height = h + 'px'
}

const stopRegionDraw = () => {
  document.removeEventListener('pointermove', handleRegionMove)
  document.removeEventListener('pointerup', handleRegionUp)
  document.removeEventListener('pointercancel', handleRegionCancel)
  if (regionDrawing) {
    regionDrawing.host.style.touchAction = regionDrawing.prevTouchAction || ''
    regionDrawing.el.remove()
    regionDrawing = null
  }
}

const handleRegionUp = (e) => {
  const d = regionDrawing
  if (!d) return
  const rect = d.host.getBoundingClientRect()
  const x = e.clientX - rect.left
  const y = e.clientY - rect.top
  const left = Math.min(d.startX, x)
  const top = Math.min(d.startY, y)
  const w = Math.abs(x - d.startX)
  const h = Math.abs(y - d.startY)
  const rectData = { left, top, width: w, height: h }
  const style = d.style
  const color = d.color
  const kind = d.kind
  const pageNum = d.pageNum
  const chapter = d.chapter
  const iframe = d.iframe
  stopRegionDraw()
  // 拖动距离过小视为误触，丢弃
  if (w < REGION_MIN_SIZE || h < REGION_MIN_SIZE) return
  if (kind === 'pdf') savePdfRegion(pageNum, rectData, style, color)
  else saveEpubRegion(iframe, chapter, rectData, style, color)
}

const handleRegionCancel = () => stopRegionDraw()

// 保存 PDF 区域标记：复用文字划线的渲染管线（renderSinglePdfAnnotation）
const savePdfRegion = async (pageNum, rect, style, color) => {
  if (!pdfDoc) return
  const position = { page: pageNum, rect, scale: pdfScale.value, region: true, text: '区域标注' }
  try {
    const { data } = await bookApi.createAnnotation(props.id, {
      type: 'highlight',
      content: '区域标注',
      note: '',
      color,
      style,
      position: JSON.stringify(position),
      page: pageNum,
      chapter: ''
    })
    annotations.value.unshift(data)
    renderSinglePdfAnnotation(data)
    lastSelectionAt = Date.now() // 拦截绘制结束后派发的 click
  } catch (e) {
    ElMessage.error('添加区域标注失败')
  }
}

// 保存 EPUB 区域标记：坐标基于 iframe 内容文档
const saveEpubRegion = async (iframe, chapter, rect, style, color) => {
  const doc = iframe?.contentDocument
  if (!doc) return
  const position = {
    chapter,
    region: rect,
    docW: doc.documentElement.scrollWidth || doc.body?.scrollWidth || 0,
    docH: doc.documentElement.scrollHeight || doc.body?.scrollHeight || 0,
    region: true,
    text: '区域标注'
  }
  try {
    const { data } = await bookApi.createAnnotation(props.id, {
      type: 'highlight',
      content: '区域标注',
      note: '',
      color,
      style,
      position: JSON.stringify(position),
      page: 0,
      chapter: String(chapter)
    })
    annotations.value.unshift(data)
    appendEpubRegion(iframe, data)
    lastSelectionAt = Date.now()
  } catch (e) {
    ElMessage.error('添加区域标注失败')
  }
}

// 渲染单个 EPUB 区域标记
const appendEpubRegion = (iframe, ann) => {
  const doc = iframe?.contentDocument
  const body = doc?.body
  if (!body) return
  let pos
  try { pos = JSON.parse(ann.position || '{}') } catch { return }
  if (!pos.region) return
  const div = doc.createElement('div')
  div.className = 'epub-region ann-' + (ann.style || 'highlight')
  div.dataset.annId = ann.id
  div.style.setProperty('--ann-color', ann.color || '#ffeb3b')
  div.style.left = pos.region.left + 'px'
  div.style.top = pos.region.top + 'px'
  div.style.width = pos.region.width + 'px'
  div.style.height = pos.region.height + 'px'
  div.addEventListener('click', (e) => {
    e.stopPropagation()
    if (currentTool.value === 'eraser') deleteAnnotation(ann.id)
  })
  body.appendChild(div)
}

// 应用某章节的所有 EPUB 区域标记
const applyEpubRegions = (chapterIndex, frame) => {
  const iframe = frame || (epubScroll.value ? (epubIframes[chapterIndex] || null) : currentIframe.value)
  if (!iframe || !iframe.contentDocument) return
  const list = annotations.value.filter(a => {
    try {
      const pos = JSON.parse(a.position || '{}')
      return pos.region && pos.chapter === chapterIndex && a.type === 'highlight'
    } catch { return false }
  })
  list.forEach(ann => appendEpubRegion(iframe, ann))
}

// 判断按下点是否命中可见文字（EPUB：文本节点矩形精确命中）
// 命中文字 → 放行原生文字选择（划线流程）；未命中（空白/图片）→ 区域绘制
const hasTextAtPoint = (doc, x, y) => {
  const el = doc.elementFromPoint(x, y)
  if (!el) return false
  // 命中 body / 纸张外层等大容器 → 视为空白，避免整页文本误判
  if (el === doc.body || el.id === '__sheet') return false
  const walker = doc.createTreeWalker(el, NodeFilter.SHOW_TEXT)
  while (walker.nextNode()) {
    const n = walker.currentNode
    if (!n.nodeValue || !n.nodeValue.trim()) continue
    const range = doc.createRange()
    range.selectNodeContents(n)
    const rect = range.getBoundingClientRect()
    if (rect.left <= x && x <= rect.right && rect.top <= y && y <= rect.bottom) return true
  }
  return false
}

// PDF：画笔模式下按住拖动圈区域（事件委托到滚动容器，覆盖整页含空白处）
// 桌面端按下位置命中文字 span 时放行，保证「选中文字 → 划线」原有流程不受影响
const handlePdfRegionPointerDown = (e) => {
  if (!isDrawToolActive() || !pdfDoc) return
  const t = e.target
  if (t.closest?.('.ann-overlay')) return // 让位：点击已有标记（橡皮删除）
  const wrapper = t.closest?.('.pdf-page-wrapper')
  if (!wrapper) return
  // 按下点在文字 span 上 → 放行文字选择（桌面：文字划线；移动端选择模式：长按选词划线/复制）
  if (t.tagName === 'SPAN' && t.closest?.('.pdf-text-layer')) return
  // 其余位置（空白/图片/表格区）：按住拖动圈区域（移动端同样保留区域高亮能力）
  const pageNum = parseInt(wrapper.id.replace('pdf-page-', ''))
  const overlay = pageOverlayEls[pageNum]
  if (!overlay) return
  e.preventDefault() // 阻止文字选择开始
  startRegionDraw('pdf', document, overlay, e, { pageNum })
}

// EPUB：画笔模式下在 iframe 内按住拖动圈区域
const initEpubRegionDraw = (doc, iframe, chapterIndex) => {
  if (!doc) return
  doc.addEventListener('pointerdown', (e) => {
    if (!isDrawToolActive()) return
    const t = e.target
    if (t.closest?.('.epub-region')) return // 让位：点击已有标记（橡皮删除）
    if (t.closest?.('.epub-highlight')) return // 让位：文字划线标记
    if (t.closest?.('a')) return // 链接交还浏览器
    // 按下点命中文字 → 放行文字选择（桌面：文字划线；移动端选择模式：长按选词划线/复制）
    if (hasTextAtPoint(doc, e.clientX, e.clientY)) return
    e.preventDefault()
    startRegionDraw('epub', doc, doc.body, e, { chapter: chapterIndex, iframe })
  })
}

const processPdfSelection = (textLayer) => {
  const selection = window.getSelection()
  const text = selection.toString().trim()
  if (text.length > 0) {
    selectedText.value = text
    let selViewport = { x: 0, y: 0 }
    if (selection.rangeCount > 0) {
      const r = selection.getRangeAt(0).getBoundingClientRect()
      // 找到选择所在的页面 wrapper
      const wrapper = textLayer?.closest('.pdf-page-wrapper')
      const wrapperRect = wrapper?.getBoundingClientRect()
      if (wrapperRect) {
        selectionRect.value = {
          left: r.left - wrapperRect.left,
          top: r.top - wrapperRect.top,
          width: r.width,
          height: r.height
        }
        // 记录当前选中的页码
        const pageNum = parseInt(wrapper.id.replace('pdf-page-', ''))
        if (pageNum) currentPdfPage.value = pageNum
      }
      selViewport = {
        x: r.left + r.width / 2,
        y: Math.max(r.top - 60, 60)
      }
    }

    // 待选朗读状态：选中文字后自动从该处开始朗读
    if (ttsPendingSelection.value) {
      ttsPendingSelection.value = false
      showTtsPanel.value = false
      lastSelectionAt = Date.now()
      startTTSFromSelection()
      return
    }

    // 记录选中时间，拦截移动端合成 click（防止刚选中就被"点击空白处"逻辑取消）
    lastSelectionAt = Date.now()

    // 嵌入式 PDF：选区抛给外壳，由外壳的"选中即弹出"工具条处理（划线/笔记/复制）
    if (props.embedded) {
      emit('pdf-select', {
        text: selectedText.value,
        x: selViewport.x,
        y: selViewport.y,
        page: currentPdfPage.value
      })
    } else if (isTouchDevice()) {
      // 移动端：纯选区 → 弹出划选操作条
      showHighlightToolbar.value = true
    }
  }
}

const handlePdfSelection = (e) => {
  processPdfSelection(e?.currentTarget)
}

const processIframeSelection = (doc) => {
  const win = doc?.defaultView
  const selection = win?.getSelection()
  const text = selection?.toString().trim() || ''
  if (text.length > 0) {
    selectedText.value = text
    // 连续滚动模式：从事件源 document 反查所在 iframe，精确归属章节
    const iframe = doc?.defaultView?.frameElement || currentIframe.value
    lastSelectionFrame = iframe
    if (iframe) {
      const iframeRect = iframe.getBoundingClientRect()
      const r = selection.getRangeAt(0).getBoundingClientRect()
      toolbarPos.value = {
        left: iframeRect.left + r.left + r.width / 2,
        top: Math.max(iframeRect.top + r.top - 60, 60)
      }
    }

    // 待选朗读状态：选中文字后自动从该处开始朗读
    if (ttsPendingSelection.value) {
      ttsPendingSelection.value = false
      showTtsPanel.value = false
      lastSelectionAt = Date.now()
      startTTSFromSelection()
      return
    }

    // 记录选中时间，拦截移动端合成 click（防止刚选中就被"点击空白处"逻辑取消）
    lastSelectionAt = Date.now()

    if (currentTool.value && currentTool.value !== 'eraser') {
      // 桌面画笔模式：自动创建标记；移动端走"先选文字后处理"，由操作条确认
      if (isTouchDevice()) showHighlightToolbar.value = true
      else saveHighlight('')
    } else if (noteMode.value) {
      showHighlightToolbar.value = true
    } else if (isTouchDevice()) {
      // 移动端：纯选区 → 直接弹出"先选文字后处理"操作条
      showHighlightToolbar.value = true
    }
  }
}

const handleIframeSelection = (e) => {
  processIframeSelection(e.target?.ownerDocument)
}

// 移动端：PDF 文本层位于父 document，监听全局 selectionchange（防抖 250ms）
// 注意：仅 PDF 生效。EPUB 内容在 iframe 独立 document 中，其 selection 变化会连带父
// document 的 selection 变为折叠状态，若在这里处理会误关 EPUB 的选中浮层/笔记框。
const handleDocSelectionChange = () => {
  if (!isTouchDevice()) return
  const ft = book.value?.file_type?.toLowerCase()
  if (ft !== 'pdf') return
  clearTimeout(selectionTimer)
  selectionTimer = setTimeout(() => {
    const sel = window.getSelection()
    if (!sel || sel.isCollapsed) {
      // 选择被取消则收起工具栏（按钮动作期间的折叠需抑制，避免误关）
      if (showHighlightToolbar.value && !currentTool.value && !suppressCollapseHide) showHighlightToolbar.value = false
      lastSelectionText = ''
      return
    }
    const text = sel.toString().trim()
    if (!text) return
    // anchor 可能落在选区边缘的非文本节点，优先取公共祖先定位文本层
    let ca = sel.anchorNode
    if (sel.rangeCount > 0 && !(ca?.closest?.('.pdf-text-layer'))) {
      ca = sel.getRangeAt(0).commonAncestorContainer
    }
    const el = ca?.nodeType === Node.TEXT_NODE ? ca.parentElement : ca
    const textLayer = el?.closest?.('.pdf-text-layer')
    if (!textLayer) return
    if (lastSelectionText === text) return
    lastSelectionText = text
    processPdfSelection(textLayer)
  }, 250)
}

// 移动端：EPUB 内容在 iframe 独立 document 中，单独监听其 selectionchange。
// selectionchange 事件的 target 是触发它的 document，由此反查所在 iframe，
// 避免连续滚动模式下多个 iframe 并存时误用 currentIframe（可能不是选中所在的章节）。
const handleIframeSelectionChange = (e) => {
  if (!isTouchDevice()) return
  const doc = e?.target
  const iframe = doc?.defaultView?.frameElement || currentIframe.value
  if (!iframe?.contentDocument) return
  clearTimeout(selectionTimer)
  selectionTimer = setTimeout(() => {
    const win = iframe.contentDocument.defaultView
    const sel = win?.getSelection()
    if (!sel || sel.isCollapsed) {
      // 选择被取消则收起工具栏（按钮动作期间的折叠需抑制，避免误关）
      if (showHighlightToolbar.value && !currentTool.value && !suppressCollapseHide) showHighlightToolbar.value = false
      lastSelectionText = ''
      return
    }
    const text = sel.toString().trim()
    if (!text) return
    if (lastSelectionText === text) return
    lastSelectionText = text
    lastSelectionFrame = iframe
    processIframeSelection(iframe.contentDocument)
  }, 250)
}

// 当前是否有活动选区（PDF 在父 document，EPUB 在各 iframe 内）
const hasActiveSelection = () => {
  const ft = book.value?.file_type?.toLowerCase()
  if (ft === 'pdf') {
    const sel = window.getSelection()
    return !!sel && !sel.isCollapsed && !!sel.toString().trim()
  }
  if (ft === 'epub' || ft === 'kepub' || ft === 'azw3') {
    const frames = epubScroll.value ? Object.values(epubIframes) : [currentIframe.value]
    return frames.some(f => {
      try {
        const sel = f?.contentWindow?.getSelection()
        return !!sel && !sel.isCollapsed && !!sel.toString().trim()
      } catch (e) { return false }
    })
  }
  return false
}

const handleReaderClick = (e) => {
  // 选中文字后紧随的 click（含移动端延迟派发的合成 click）不当作"点击空白处"处理
  if (Date.now() - lastSelectionAt < 500) return
  if (hasActiveSelection()) return
  if (!e.target.closest('.note-toolbar') && !e.target.closest('.tts-panel') && !e.target.closest('.tools-sidebar') && !e.target.closest('.el-button')) {
    cancelHighlight()
    noteMode.value = false
    showColorPicker.value = false
    showTtsPanel.value = false
    showAnnotations.value = false
  }
}

// === 划线/批注操作 ===
const cancelHighlight = () => {
  showHighlightToolbar.value = false
  selectedText.value = ''
  selectionRect.value = null
  noteText.value = ''
  lastSelectionFrame = null
  window.getSelection().removeAllRanges()
  // 清掉所有已加载 EPUB iframe 的选区（实际选中可能不在 currentIframe 中）
  const frames = epubScroll.value ? Object.values(epubIframes) : [currentIframe.value]
  frames.forEach(f => {
    try { f?.contentWindow?.getSelection()?.removeAllRanges() } catch (e) {}
  })
}

const saveHighlight = async (note = '', explicitStyle = null) => {
  lockCollapseHide()
  if (!selectedText.value) return

  const position = { text: selectedText.value }
  let pageNum = 0
  let chapterStr = ''

  const ft = book.value?.file_type?.toLowerCase()
  if (ft === 'pdf') {
    position.page = currentPdfPage.value
    pageNum = currentPdfPage.value
    if (selectionRect.value) {
      position.rect = selectionRect.value
      position.scale = pdfScale.value
    }
  } else {
    // 连续滚动模式：以实际选择所在 iframe 的章节为准
    const selChapter = lastSelectionFrame?.dataset?.chapterIndex
    const chapterIdx = selChapter != null ? Number(selChapter) : currentChapter.value
    position.chapter = chapterIdx
    chapterStr = String(chapterIdx)
  }

  // 笔记模式：仅保存笔记，不生成划线
  const isNote = noteMode.value
  // 画笔模式使用 currentTool，否则使用 currentStyle
  const style = isNote ? '' : (explicitStyle || (currentTool.value && currentTool.value !== 'eraser' ? currentTool.value : currentStyle.value))

  try {
    const { data } = await bookApi.createAnnotation(props.id, {
      type: isNote ? 'note' : 'highlight',
      content: selectedText.value,
      note: note,
      color: currentColor.value,
      style: style,
      position: JSON.stringify(position),
      page: pageNum,
      chapter: chapterStr
    })
    annotations.value.unshift(data)

    // 不重新渲染页面，直接插入新标记到 DOM（笔记模式不生成划线）
    if (!isNote) {
      if (ft === 'pdf') {
        renderSinglePdfAnnotation(data)
      } else {
        // 渲染必须在 cancelHighlight 之前：cancelHighlight 会清空 lastSelectionFrame，
        // 而渲染依赖它定位到选中文字所在 iframe（连续滚动模式下 currentIframe 可能不是选中章节）
        const targetDoc = (lastSelectionFrame || currentIframe.value)?.contentDocument
        highlightTextInNode(targetDoc?.body, data.content, data.color, data.style, data.id)
      }
    }
    cancelHighlight()
  } catch (e) {
    ElMessage.error(isNote ? '保存笔记失败' : '添加划线失败')
  }
}

// 供外壳（FoliateReader）调用：以当前 PDF 选区创建划线 / 笔记
const pdfApplyHighlight = ({ style, color, note = '', isNote = false } = {}) => {
  if (!selectedText.value) return
  currentColor.value = color || currentColor.value
  currentStyle.value = style || currentStyle.value
  noteMode.value = !!isNote
  return saveHighlight(note, style)
}

// 供外壳调用：清除当前 PDF 选区与高亮态
const pdfClearSelection = () => {
  showHighlightToolbar.value = false
  selectedText.value = ''
  try { window.getSelection()?.removeAllRanges() } catch {}
}

const deleteAnnotation = async (annId) => {
  try {
    await bookApi.deleteAnnotation(annId)
    annotations.value = annotations.value.filter(a => a.id !== annId)

    // 不重新渲染页面，直接从 DOM 移除
    Object.values(pageOverlayEls).forEach(overlay => {
      if (!overlay) return
      const el = overlay.querySelector(`[data-ann-id="${annId}"]`)
      if (el) el.remove()
    })
    // 连续滚动模式：划线可能位于任意已加载章节的 iframe 中，全部检查
    const targetFrames = epubScroll.value ? Object.values(epubIframes) : [currentIframe.value]
    targetFrames.forEach(f => {
      if (!f?.contentDocument) return
      const span = f.contentDocument.querySelector(`[data-ann-id="${annId}"]`)
      if (span && span.parentNode) {
        const parent = span.parentNode
        while (span.firstChild) {
          parent.insertBefore(span.firstChild, span)
        }
        parent.removeChild(span)
        parent.normalize()
      }
    })
  } catch (e) {
    ElMessage.error('删除失败')
  }
}

const jumpToAnnotation = async (ann) => {
  showAnnotations.value = false
  try {
    const pos = JSON.parse(ann.position || '{}')
    const ft = book.value?.file_type?.toLowerCase()
    if (ft === 'pdf' && pos.page) {
      currentPdfPage.value = pos.page
      scrollToPdfPage(pos.page)
    } else if ((ft === 'epub' || ft === 'kepub' || ft === 'azw3') && pos.chapter !== undefined) {
      const idx = Math.max(0, Number(pos.chapter) || 0)
      if (epubScroll.value) {
        // 连续滚动模式：先确保章节加载，再精确滚动到标记文字所在位置
        await renderChapter(idx)
        const iframe = epubIframes[idx]
        if (iframe) {
          await waitIframeLoaded(iframe)
          if (!scrollEpubToAnnotation(iframe, ann)) {
            scrollEpubToChapter(idx)
          }
          updateEpubScrollState()
        }
      } else {
        renderChapter(idx)
      }
    }
  } catch (e) {
    console.warn('跳转失败', e)
  }
}

// 连续滚动模式：在章节 iframe 中查找标记文字并滚动容器到该位置（找不到返回 false）
const scrollEpubToAnnotation = (iframe, ann) => {
  const container = document.getElementById('epub-viewer')
  if (!iframe || !container) return false
  try {
    const doc = iframe.contentDocument
    if (!doc || !doc.body) return false
    let posText = ''
    try { posText = JSON.parse(ann.position || '{}').text || '' } catch (e) {}
    const text = posText || ann.content || ''
    if (!text) return false
    // 查找包含标记文字的文本节点（优先完整匹配）
    const walker = doc.createTreeWalker(doc.body, NodeFilter.SHOW_TEXT)
    let target = null
    let matchIdx = -1
    while (walker.nextNode()) {
      const t = walker.currentNode.textContent || ''
      const idx = t.indexOf(text)
      if (idx >= 0) { target = walker.currentNode; matchIdx = idx; break }
    }
    if (!target) return false
    const range = doc.createRange()
    range.setStart(target, matchIdx)
    range.setEnd(target, Math.min(matchIdx + text.length, (target.textContent || '').length))
    const rect = range.getBoundingClientRect()
    const iframeRect = iframe.getBoundingClientRect()
    const containerRect = container.getBoundingClientRect()
    // 容器滚动目标 = 当前滚动 + iframe 相对容器偏移 + 文字相对 iframe 偏移 - 顶部留白
    const targetTop = container.scrollTop + (iframeRect.top - containerRect.top) + (rect.top - iframeRect.top) - 120
    container.scrollTo({ top: Math.max(0, targetTop), behavior: 'smooth' })
    return true
  } catch (e) {
    console.warn('定位标记文字失败', e)
    return false
  }
}

const annStyleLabel = (ann) => {
  if (ann.type === 'note') return '笔'
  const map = { highlight: '底', underline: '横', wavy: '波', strikethrough: '删' }
  return map[ann.style] || '底'
}

const annStylePreview = (ann) => {
  if (ann.type === 'note') {
    return { width: '16px', height: '16px', display: 'inline-flex', alignItems: 'center', justifyContent: 'center', fontSize: '10px', color: '#a5b4fc', border: '1px solid color-mix(in srgb, var(--app-primary) 45%, transparent)', borderRadius: '4px' }
  }
  const color = ann.color || '#ccc'
  switch (ann.style) {
    case 'underline': return { borderBottom: `2px solid ${color}`, width: '16px', height: '16px', display: 'inline-flex', alignItems: 'center', justifyContent: 'center', fontSize: '10px', color: '#999' }
    case 'wavy': return { borderBottom: `2px dashed ${color}`, width: '16px', height: '16px', display: 'inline-flex', alignItems: 'center', justifyContent: 'center', fontSize: '10px', color: '#999' }
    case 'strikethrough': return { textDecoration: `line-through solid ${color} 2px`, width: '16px', height: '16px', display: 'inline-flex', alignItems: 'center', justifyContent: 'center', fontSize: '10px', color: '#999' }
    case 'highlight':
    default: return { background: color, width: '16px', height: '16px', borderRadius: '3px', display: 'inline-flex', alignItems: 'center', justifyContent: 'center', fontSize: '10px', color: '#fff', opacity: 0.7 }
  }
}

// === 翻页 / 导航 ===
const scrollToPdfPage = (pageNum, smooth = false) => {
  const el = document.getElementById('pdf-page-' + pageNum)
  if (el && pdfContainer.value) {
    // 朗读自动翻页时平滑滚动；手动翻页保持即时到位
    pdfContainer.value.scrollTo({ top: el.offsetTop, behavior: smooth ? 'smooth' : 'auto' })
  }
}

// === 阅读进度条：点击跳转 ===
const onProgressClick = (e) => {
  const el = e.currentTarget
  if (!el) return
  const rect = el.getBoundingClientRect()
  if (rect.width <= 0) return
  const pct = Math.max(0, Math.min(100, ((e.clientX - rect.left) / rect.width) * 100))
  applyProgress(pct)
}

const onProgressMarkClick = (m) => {
  const ann = annotations.value.find(a => a.id === m.id)
  if (ann) jumpToAnnotation(ann)
}

const applyProgress = (pct) => {
  const ft = book.value?.file_type?.toLowerCase()
  if (ft === 'pdf') {
    if (!pdfContainer.value) return
    const c = pdfContainer.value
    const maxScroll = c.scrollHeight - c.clientHeight
    if (maxScroll <= 0) return
    // 与显示公式（scrollTop/maxScroll）同源：先换算目标滚动位置，再找最近的页面
    const targetScroll = Math.max(0, Math.min(maxScroll, (pct / 100) * maxScroll))
    let target = 1
    let best = Infinity
    for (const p of pageInfos.value) {
      const pEl = document.getElementById('pdf-page-' + p.num)
      if (!pEl) continue
      const d = Math.abs(pEl.offsetTop - targetScroll)
      if (d < best) { best = d; target = p.num }
    }
    currentPdfPage.value = target
    scrollToPdfPage(target)
    scheduleProgressSave(true)
  } else if (ft === 'epub' || ft === 'kepub' || ft === 'azw3') {
    if (!epubScroll.value || !chapters.value.length) return
    jumpEpubToPercent(pct)
  }
}

// EPUB 按百分比跳转：先按章节数线性估算目标章并补齐加载，再按已加载内容高度比例定位
// （与 saveScrollOffset 显示公式同源，保证"点哪跳哪"，不受各章长短不均影响）
const jumpEpubToPercent = async (pct) => {
  const container = document.getElementById('epub-viewer')
  if (!container) return
  const n = chapters.value.length
  const p = Math.max(0, Math.min(pct, 100))
  // 1) 线性估算目标章节并确保其前后章节已加载，让点击位置附近的内容真实存在
  const est = Math.min(n - 1, Math.floor((p / 100) * n))
  await ensureEpubChaptersAround(est)
  // 2) 与显示公式同源定位：scrollTop = pct * (scrollHeight - clientHeight)
  const maxScroll = container.scrollHeight - container.clientHeight
  if (maxScroll <= 0) return
  container.scrollTop = Math.max(0, Math.min(maxScroll, (p / 100) * maxScroll))
  updateEpubScrollState()
  saveScrollOffset()
  scheduleProgressSave(true)
}

const prevPage = () => {
  if (showHighlightToolbar.value || currentTool.value) return
  const ft = book.value?.file_type?.toLowerCase()
  if (ft === 'pdf') {
    const target = Math.max(1, currentPdfPage.value - 1)
    scrollToPdfPage(target)
  } else if (ft === 'epub' || ft === 'kepub' || ft === 'azw3') {
    if (currentChapter.value > 0) renderChapter(currentChapter.value - 1)
  } else if (currentChapter.value > 0) {
    renderChapter(currentChapter.value - 1)
  }
}

const nextPage = () => {
  if (showHighlightToolbar.value || currentTool.value) return
  const ft = book.value?.file_type?.toLowerCase()
  if (ft === 'pdf') {
    const target = Math.min(totalPdfPages.value, currentPdfPage.value + 1)
    scrollToPdfPage(target)
  } else if (ft === 'epub' || ft === 'kepub' || ft === 'azw3') {
    if (currentChapter.value < chapters.value.length - 1) renderChapter(currentChapter.value + 1)
  } else if (currentChapter.value < chapters.value.length - 1) {
    renderChapter(currentChapter.value + 1)
  }
}

const goToChapter = (index) => {
  showToc.value = false
  const ft = book.value?.file_type?.toLowerCase()
  const chapter = chapters.value[index]
  // PDF：目录项带页码，直接跳转到对应页
  if (ft === 'pdf' && chapter?.page) {
    currentPdfPage.value = chapter.page
    scrollToPdfPage(chapter.page)
    return
  }
  renderChapter(index)
}

// === 目录辅助 ===
const tocItemRefs = {}
const setTocItemRef = (index, el) => {
  if (el) tocItemRefs[index] = el
}

// 当前章节是否处于激活态（PDF 按页码，其余按章节序号）
const isChapterActive = (index) => {
  const ft = book.value?.file_type?.toLowerCase()
  const chapter = chapters.value[index]
  if (ft === 'pdf') return !!chapter?.page && chapter.page === currentPdfPage.value
  return currentChapter.value === index
}

// 打开目录/切换章节时，将当前项滚动到可视区域
const scrollTocToActive = () => {
  if (!showToc.value) return
  nextTick(() => {
    const ft = book.value?.file_type?.toLowerCase()
    const idx = ft === 'pdf'
      ? chapters.value.findIndex(c => !!c.page && c.page === currentPdfPage.value)
      : currentChapter.value
    const el = tocItemRefs[idx]
    if (el) el.scrollIntoView({ block: 'center', behavior: 'smooth' })
  })
}

const zoomIn = () => {
  applyZoom(pdfScale.value + 0.25)
  savePdfScalePref(pdfScale.value)
}

const zoomOut = () => {
  applyZoom(pdfScale.value - 0.25)
  savePdfScalePref(pdfScale.value)
}

// 缩放：尺寸按比例换算（viewport 与 scale 成正比，无需重新取页），只重渲染视口附近页面
// 返回是否真正执行了缩放（false 表示缩放值未变化，调用方需自行处理 DOM 重建后的重绘）
const applyZoom = async (newScale) => {
  newScale = Math.max(0.5, Math.min(3.0, newScale))
  if (newScale === pdfScale.value || !pdfDoc || pageInfos.value.length === 0) return false
  const oldScale = pdfScale.value
  const ratio = newScale / oldScale
  pdfScale.value = newScale

  const container = pdfContainer.value
  // 锚定当前页，记录页内偏移
  const anchorEl = document.getElementById('pdf-page-' + currentPdfPage.value)
  const offsetInPage = container && anchorEl
    ? Math.max(0, container.scrollTop - anchorEl.offsetTop)
    : 0

  // 全部页面尺寸按比例缩放
  pageInfos.value.forEach(p => {
    p.width = Math.round(p.width * ratio)
    p.height = Math.round(p.height * ratio)
  })

  // 取消进行中的渲染，清空全部已渲染状态
  Object.values(pageRenderTasks).forEach(t => { try { t.cancel() } catch (e) {} })
  pageInfos.value.forEach(p => { p.rendered = false })

  await nextTick()

  // 恢复阅读位置（锚定当前页 + 页内偏移等比换算）
  if (container) {
    const newAnchor = document.getElementById('pdf-page-' + currentPdfPage.value)
    if (newAnchor) {
      container.scrollTop = newAnchor.offsetTop + offsetInPage * ratio
    }
  }

  // 只渲染当前页及前后各一页，其余由 IntersectionObserver 滚动时按需渲染
  const cur = currentPdfPage.value
  const from = Math.max(1, cur - 1)
  const to = Math.min(totalPdfPages.value, cur + 1)
  for (let i = from; i <= to; i++) {
    const p = pageInfos.value.find(x => x.num === i)
    if (p) p.rendered = true
  }
  await nextTick()
  for (let i = from; i <= to; i++) {
    renderPdfPage(i)
  }

  // 重新触发观察器，让其它仍处于视口内的页面重新按需渲染
  if (pageObserver) {
    pageInfos.value.forEach(page => {
      const el = document.getElementById('pdf-page-' + page.num)
      if (el) {
        pageObserver.unobserve(el)
        pageObserver.observe(el)
      }
    })
  }
  return true
}

// 布局切换后强制重绘当前页附近并重绑观察器（分组 DOM 已重建，原 canvas 内容丢失）
const rerenderPdfNearby = async () => {
  const cur = currentPdfPage.value
  const from = Math.max(1, cur - 1)
  const to = Math.min(totalPdfPages.value, cur + 1)
  // 取消进行中的渲染，清空全部已渲染状态
  Object.values(pageRenderTasks).forEach(t => { try { t.cancel() } catch (e) {} })
  pageInfos.value.forEach(p => { p.rendered = false })
  for (let i = from; i <= to; i++) {
    const p = pageInfos.value.find(x => x.num === i)
    if (p) p.rendered = true
  }
  await nextTick()
  for (let i = from; i <= to; i++) {
    renderPdfPage(i)
  }
  // 重新绑定观察器到新 DOM 元素
  if (pageObserver) {
    pageInfos.value.forEach(page => {
      const el = document.getElementById('pdf-page-' + page.num)
      if (el) {
        pageObserver.unobserve(el)
        pageObserver.observe(el)
      }
    })
  }
}

const goBack = () => router.back()

const downloadBook = async () => {
  try {
    const { data, headers } = await bookApi.downloadBook(props.id)
    const blob = new Blob([data])
    const link = document.createElement('a')
    link.href = URL.createObjectURL(blob)
    const filename = headers['content-disposition']?.match(/filename="(.+)"/)?.[1] || 'book'
    link.download = filename
    link.click()
  } catch (e) {
    ElMessage.error('下载失败')
  }
}

const applyTheme = (theme) => {
  // 主题色变化统一由 watch(currentTheme) 处理（原地更新样式，不跳动）
  // 这里直接赋值即可，避免重复触发重建
  currentTheme.value = theme.name
}

// === 全屏 ===
const toggleFullscreen = async () => {
  closeAllPanels()
  try {
    if (!document.fullscreenElement) {
      await readerBody.value?.requestFullscreen()
    } else {
      await document.exitFullscreen()
    }
  } catch (e) {
    fullscreen.value = !fullscreen.value
  }
}

const handleFullscreenChange = () => {
  fullscreen.value = !!document.fullscreenElement
}

// === 连续滚动：IntersectionObserver + Scroll ===
const setupPageObserver = () => {
  if (pageObserver) pageObserver.disconnect()

  pageObserver = new IntersectionObserver((entries) => {
    let bestPage = currentPdfPage.value
    let bestRatio = 0

    entries.forEach(entry => {
      const num = parseInt(entry.target.id.replace('pdf-page-', ''))
      if (entry.isIntersecting) {
        visiblePages.value.add(num)
        const page = pageInfos.value.find(p => p.num === num)
        if (page && !page.rendered) {
          page.rendered = true
          nextTick(() => renderPdfPage(num))
        }
        if (entry.intersectionRatio > bestRatio) {
          bestRatio = entry.intersectionRatio
          bestPage = num
        }
      } else {
        visiblePages.value.delete(num)
      }
    })

    if (bestPage !== currentPdfPage.value) {
      currentPdfPage.value = bestPage
    }
  }, {
    root: pdfContainer.value,
    threshold: [0, 0.1, 0.25, 0.5, 0.75, 1.0]
  })

  nextTick(() => {
    pageInfos.value.forEach(page => {
      const el = document.getElementById('pdf-page-' + page.num)
      if (el) pageObserver.observe(el)
    })
  })
}

const throttledScroll = () => {
  if (scrollRaf) return
  scrollRaf = requestAnimationFrame(() => {
    handlePdfScroll()
    scrollRaf = null
  })
}

const handlePdfScroll = () => {
  if (!pdfContainer.value) return
  const c = pdfContainer.value
  const maxScroll = c.scrollHeight - c.clientHeight
  if (maxScroll > 0) {
    currentProgress.value = (c.scrollTop / maxScroll) * 100
  }
  scheduleProgressSave() // 滚动阅读时节流记录位置
}

// === 滚轮：交给容器自然滚动，不再强制翻页 ===
const handleWheel = (e) => {
  if (showHighlightToolbar.value || currentTool.value) {
    e.preventDefault()
    return
  }
}

// === 键盘导航 ===
const handleKeydown = (e) => {
  if (e.key === 'ArrowLeft') prevPage()
  if (e.key === 'ArrowRight' || e.key === ' ') nextPage()
  if (e.key === 'Escape') {
    if (document.fullscreenElement) document.exitFullscreen()
    fullscreen.value = false
    showHighlightToolbar.value = false
    currentTool.value = null
    showAnnotations.value = false
  }
}

// 打开目录 / 章节或 PDF 页码变化时，自动滚动目录到当前项
watch([showToc, currentChapter, currentPdfPage], () => {
  scrollTocToActive()
})

// === 监听设置变化（EPUB 专用） ===
// 字号/行距/边距/字体：影响排版尺寸，必须重建容器才能让文字重新排版
watch([fontSize, lineHeight, pageMargin, fontFamily], () => {
  // 同步保存到全局阅读偏好（与设置页共享）
  saveReaderPrefs({
    fontSize: fontSize.value,
    lineHeight: lineHeight.value,
    pageMargin: pageMargin.value,
    fontFamily: fontFamily.value
  })
  if (book.value?.file_type?.toLowerCase() !== 'pdf') {
    // 滚动模式：重建整个滚动容器让所有章节应用新样式
    initEpubScroll(currentChapter.value)
  }
})

// 主题色：仅切换颜色/纸张/侧栏背景，不改变排版尺寸，因此原地替换 iframe 中的 style 标签即可，
// 不重建 DOM，文字位置与高度保持稳定，避免"主题色切换文字跳动"。
watch(currentTheme, () => {
  saveReaderPrefs({ theme: currentTheme.value })
  if (book.value?.file_type?.toLowerCase() !== 'pdf') {
    updateEpubIframeStyles()
  }
})

// === 生命周期 ===
// 关闭/刷新/切换标签页时尽可能保存当前阅读位置与朗读断点
const handlePageHide = () => { doSaveProgress(true) }

onMounted(() => {
  // 内嵌于 FoliateReader 外壳时，阅读/朗读会话由外壳统一计时上报，此处不再重复上报
  if (!props.embedded) startReadingSession()
  loadTtsSettings()
  loadBookInfo().then(() => {
    loadAnnotations()
    // 解析已保存的朗读断点（用于"续播"入口）
    try {
      if (book.value?.tts_location) savedTtsLocation.value = JSON.parse(book.value.tts_location)
    } catch (e) {
      savedTtsLocation.value = null
    }
    const ft = book.value?.file_type?.toLowerCase()
    if (ft === 'pdf') {
      nextTick(() => loadPDF())
      // 移动端长按选词：PDF 文本层（事件委托绑定一次即可，随实例存活）
      initLongPressSelection(document, { textLayerOnly: true })
      // 画笔模式区域绘制：事件委托到 PDF 滚动容器（覆盖整页含空白区域）
      nextTick(() => {
        pdfContainer.value?.removeEventListener('pointerdown', handlePdfRegionPointerDown)
        pdfContainer.value?.addEventListener('pointerdown', handlePdfRegionPointerDown)
      })
      // PDF 目录（书签/目录页/编号标题）通过统一目录接口加载，与 PDF 渲染并行
      loadChapters()
    } else if (ft === 'epub' || ft === 'kepub' || ft === 'azw3') {
      // 监听必须放在 EPUB 容器渲染之后（book 加载完成、模板已更新）
      epubViewer.value?.addEventListener('scroll', onEpubScroll, { passive: true })
      loadChapters().then(() => {
        // 滚动模式：直接传入进度百分比，与 saveScrollOffset 公式同源
        const pos = book.value?.progress > 0
          ? book.value.progress
          : 0
        initEpubScroll(pos)
      })
    }
  })
  window.addEventListener('keydown', handleKeydown)
  window.addEventListener('wheel', handleWheel, { passive: true })
  document.addEventListener('fullscreenchange', handleFullscreenChange)
  document.addEventListener('selectionchange', handleDocSelectionChange)
  window.addEventListener('beforeunload', handlePageHide)
  window.addEventListener('pagehide', handlePageHide)
  window.addEventListener('resize', onPdfResize)
})

onUnmounted(() => {
  // 先落库进度与朗读断点（此时朗读位置/句子仍有效），再停止朗读，避免断点被写成开头
  flushProgressSave().catch(() => {})
  stopTTS()
  window.removeEventListener('keydown', handleKeydown)
  window.removeEventListener('wheel', handleWheel, { passive: true })
  document.removeEventListener('fullscreenchange', handleFullscreenChange)
  document.removeEventListener('selectionchange', handleDocSelectionChange)
  window.removeEventListener('beforeunload', handlePageHide)
  window.removeEventListener('pagehide', handlePageHide)
  window.removeEventListener('resize', onPdfResize)
  if (selectionTimer) clearTimeout(selectionTimer)
  if (pageObserver) pageObserver.disconnect()
  if (scrollRaf) cancelAnimationFrame(scrollRaf)
  if (sessionTimer) clearInterval(sessionTimer)
  epubViewer.value?.removeEventListener('scroll', onEpubScroll)
  pdfContainer.value?.removeEventListener('scroll', throttledScroll)
  if (book.value) {
    reportReadingSession()
  }
})

// PDF 被外壳统一 TTS 驱动时的朗读态（用于进度保存记录朗读页而非视口页）
const pdfTtsActive = ref(false)

// 内嵌模式（在 FoliateReader 统一外壳中打开 PDF）下，向父级暴露导航与批注数据
defineExpose({
  isPdf: true,
  chapters,
  annotations,
  goToChapter,
  jumpToAnnotation,
  deleteAnnotation,
  prevPage,
  nextPage,
  zoomIn,
  zoomOut,
  onPdfScaleChange,
  onPdfLayoutChange,
  pdfScale,
  pdfLayout,
  currentPdfPage,
  totalPdfPages,
  pdfApplyHighlight,
  pdfClearSelection,
  // === 朗读桥接：由外壳统一 TTS 播放器（useTtsPlayer）驱动，复用本组件已有的取句/高亮/翻页 ===
  // 外壳朗读开始时设定"朗读页"并标记为朗读中（进度保存据此记录朗读页而非视口页）
  ttsSetPage: (page) => { ttsPdfPage.value = page || 1; pdfTtsActive.value = true; pdfTtsSentence.value = -1 },
  // 朗读停止：清除高亮与朗读态
  ttsStop: () => { pdfTtsActive.value = false; clearTtsHighlights() },
  // 取某页句子列表（等待文本层渲染完成）
  ttsGetPageSentences: async (page) => splitSentences(await waitForPdfPageText(page ?? ttsPdfPage.value)),
  // 朗读完一页后翻到下一页并返回其句子；已到末页返回空数组
  ttsNextPage: async () => {
    if (ttsPdfPage.value < totalPdfPages.value) {
      const next = ttsPdfPage.value + 1
      ttsPdfPage.value = next
      currentPdfPage.value = next
      scrollToPdfPage(next, true)
      const text = await waitForPdfPageText(next)
      return splitSentences(text)
    }
    return []
  },
  // 每句播放前：精确高亮当前朗读页中该句的字符并滚动到视口中部。
  // i 为外壳统一播放器下发的“当前句下标”，据此记录朗读断点（pdfTtsSentence）。
  ttsOnSentence: (i, sentence) => {
    if (typeof i === 'number') pdfTtsSentence.value = i
    const hit = highlightPdfSentence(sentence, ttsPdfPage.value)
    if (hit?.firstRect) scrollPdfToSentence(hit)
    scheduleProgressSave()
    return !!hit
  }
})
</script>

<style scoped>
.reader-page {
  height: 100vh;
  width: 100%;
  display: flex;
  flex-direction: column;
  background: #eef1f6;
}

.reader-page::before {
  content: '';
  position: fixed;
  width: 520px;
  height: 520px;
  border-radius: 50%;
  background: radial-gradient(circle, color-mix(in srgb, var(--app-primary) 14%, transparent), transparent 70%);
  top: -160px;
  left: -160px;
  filter: blur(90px);
  pointer-events: none;
  z-index: 0;
}

.reader-page::after {
  content: '';
  position: fixed;
  width: 440px;
  height: 440px;
  border-radius: 50%;
  background: radial-gradient(circle, rgba(34, 211, 238, 0.1), transparent 70%);
  bottom: -120px;
  right: -120px;
  filter: blur(90px);
  pointer-events: none;
  z-index: 0;
}

.reader-header {
  position: relative;
  z-index: 100;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 16px;
  height: 54px;
  background: rgba(255, 255, 255, 0.9);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  color: #1f2937;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  flex-shrink: 0;
}

/* 选择模式按钮：默认（桌面端）隐藏，移动端媒体查询中显示 */
.reader-header .mobile-only {
  display: none;
}

/* 阅读进度条：标题栏下边框（替代原 border-bottom），hover 加高便于点击 */
.reader-progress {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 3px;
  z-index: 102;
  cursor: pointer;
  background: rgba(0, 0, 0, 0.08);
  transition: height 0.2s ease;
}
.reader-progress:hover {
  height: 9px;
}
.reader-progress .progress-fill {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  border-radius: 0 2px 2px 0;
  background: var(--app-primary);
  transition: width 0.15s ease;
}
.reader-progress .progress-mark {
  position: absolute;
  top: 50%;
  width: 6px;
  height: 6px;
  margin-top: -3px;
  border-radius: 50%;
  background: #ffb020;
  box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.35);
}
.reader-progress .progress-mark.mark-note {
  background: #f472b6;
}
.reader-progress:hover .progress-mark {
  width: 9px;
  height: 9px;
  margin-top: -4.5px;
}

.header-left, .header-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

/* 用 flex gap 统一按钮间距，覆盖 Element Plus 默认的 margin-left */
.header-left :deep(.el-button + .el-button),
.header-right :deep(.el-button + .el-button) {
  margin-left: 0;
}

/* 顶部按钮过多时（窄屏）允许横向滚动，不挤压书名 */
.header-right {
  min-width: 0;
  overflow-x: auto;
  scrollbar-width: none;
  -webkit-overflow-scrolling: touch;
}
.header-right::-webkit-scrollbar {
  display: none;
}
.header-right :deep(.el-button) {
  flex-shrink: 0;
}

.header-left :deep(.el-button),
.header-right :deep(.el-button) {
  color: #475569;
  border-radius: 10px;
  transition: all 0.15s;
}

.header-left :deep(.el-button:hover),
.header-right :deep(.el-button:hover) {
  color: var(--app-primary, #6366f1);
  background: #eef2ff;
  box-shadow: 0 0 12px color-mix(in srgb, var(--app-primary) 25%, transparent);
}

.book-title {
  font-size: 15px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 400px;
  color: #1f2937;
}

.zoom-level {
  font-size: 12px;
  color: #475569;
  min-width: 40px;
  text-align: center;
}

.ann-count {
  margin-left: 4px;
  font-size: 11px;
  background: var(--app-primary);
  color: #fff;
  border-radius: 8px;
  padding: 0 5px;
  line-height: 14px;
  height: 14px;
  display: inline-flex;
  align-items: center;
}

.reader-body {
  flex: 1;
  position: relative;
  overflow: hidden;
  z-index: 1;
  /* PDF 左右翻页热区宽度：浮动工具条据此避让，避免遮挡/重叠 */
  --pdf-zone: 44px;
}

.reader-body.fullscreen {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 2000;
}

/* 右侧画笔工具栏（桌面端直接显示，手机端隐藏） */
.tools-sidebar {
  position: absolute;
  top: 50%;
  right: 12px;
  transform: translateY(-50%);
  z-index: 55;
  background: rgba(255, 255, 255, 0.9);
  backdrop-filter: blur(14px);
  -webkit-backdrop-filter: blur(14px);
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 14px;
  box-shadow: 0 10px 36px rgba(0,0,0,0.12), inset 0 1px 0 rgba(255,255,255,0.6);
  padding: 10px 6px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  transition: padding 0.2s ease, box-shadow 0.2s ease;
}

/* 收起状态：仅保留展开按钮，压缩成小胶囊 */
.tools-sidebar.collapsed {
  padding: 5px;
  background: rgba(255, 255, 255, 0.85);
  box-shadow: 0 6px 20px rgba(0,0,0,0.1), inset 0 1px 0 rgba(255,255,255,0.6);
}
.tools-expand {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 10px;
  cursor: pointer;
  background: rgba(0, 0, 0, 0.04);
  border: 1px solid rgba(0, 0, 0, 0.08);
  color: #475569;
  transition: all 0.2s;
}
.tools-expand:hover {
  background: color-mix(in srgb, var(--app-primary) 14%, transparent);
  border-color: color-mix(in srgb, var(--app-primary) 35%, transparent);
  color: var(--app-primary, #6366f1);
  box-shadow: 0 0 14px color-mix(in srgb, var(--app-primary) 25%, transparent);
}

/* 展开状态底部的收起按钮 */
.tools-toggle {
  width: 34px;
  height: 26px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  cursor: pointer;
  color: #94a3b8;
  transition: all 0.2s;
}
.tools-toggle:hover {
  background: rgba(0, 0, 0, 0.05);
  color: var(--app-primary, #6366f1);
}

.tools-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
  align-items: center;
}

.tools-group.colors {
  gap: 4px;
}

.tools-divider {
  width: 20px;
  height: 1px;
  background: linear-gradient(90deg, transparent, color-mix(in srgb, var(--app-primary) 45%, transparent), transparent);
  margin: 2px 0;
}

.paint-tool {
  width: 44px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  padding: 6px 0;
  border-radius: 10px;
  cursor: pointer;
  border: 2px solid transparent;
  transition: all 0.2s;
  background: rgba(255, 255, 255, 0.06);
}

.paint-tool:hover {
  background: color-mix(in srgb, var(--app-primary) 16%, transparent);
  box-shadow: 0 0 10px color-mix(in srgb, var(--app-primary) 20%, transparent);
}

.paint-tool.active {
  border-color: var(--app-primary);
  background: linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 28%, transparent), color-mix(in srgb, var(--app-primary) 14%, transparent));
  box-shadow: 0 0 16px color-mix(in srgb, var(--app-primary) 40%, transparent);
}

.paint-tool.eraser.active {
  border-color: #f56c6c;
  background: #fef0f0;
}

/* 划线统计按钮 */
.paint-tool.stats-tool:hover {
  background: color-mix(in srgb, var(--app-primary) 16%, transparent);
}

.stats-icon {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
}

.stats-count {
  position: absolute;
  top: -4px;
  right: -6px;
  min-width: 14px;
  height: 14px;
  padding: 0 3px;
  border-radius: 7px;
  background: #f56c6c;
  color: #fff;
  font-size: 9px;
  font-weight: 600;
  line-height: 14px;
  text-align: center;
}

.paint-preview {
  width: 28px;
  height: 20px;
  border-radius: 3px;
  box-sizing: border-box;
  flex-shrink: 0;
}

.paint-preview.eraser-icon {
  display: flex;
  align-items: center;
  justify-content: center;
}

/* 选中记录笔记按钮 */
.note-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  color: #666;
}
.paint-tool.note-tool:hover .note-icon {
  color: #a5b4fc;
}
.paint-tool.note-tool.active .note-icon {
  color: #a5b4fc;
}

/* 从这里开始阅读 */
.read-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  color: #67e8f9;
}
.paint-tool.read-tool:hover .read-icon {
  color: #a5f3fc;
}
.paint-tool.read-tool.active .read-icon {
  color: #a5f3fc;
}

.paint-label {
  font-size: 10px;
  color: #b8c5d6;
  line-height: 1;
  user-select: none;
}

.paint-tool.active .paint-label {
  color: #a5b4fc;
  font-weight: 500;
}

.paint-tool.eraser.active .paint-label {
  color: #f56c6c;
}

/* 颜色选择按钮 + 弹出层 */
.tools-group.colors {
  position: relative;
}
.color-picker {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 8px 6px;
  border-radius: 12px;
  cursor: pointer;
  user-select: none;
  transition: background 0.2s;
}
.color-picker:hover,
.color-picker.open {
  background: color-mix(in srgb, var(--app-primary) 16%, transparent);
}
.color-dot {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  border: 2px solid rgba(255, 255, 255, 0.35);
  box-shadow: 0 0 10px color-mix(in srgb, var(--app-primary) 45%, transparent);
  transition: transform 0.15s;
}
.color-picker:hover .color-dot {
  transform: scale(1.1);
}
.color-popover {
  position: absolute;
  right: calc(100% + 10px);
  top: 0;
  z-index: 1001;
  display: flex;
  gap: 8px;
  padding: 10px;
  background: rgba(24, 24, 48, 0.96);
  backdrop-filter: blur(20px) saturate(1.4);
  -webkit-backdrop-filter: blur(20px) saturate(1.4);
  border: 1px solid color-mix(in srgb, var(--app-primary) 35%, transparent);
  border-radius: 14px;
  box-shadow: 0 12px 36px rgba(0, 0, 0, 0.5), inset 0 1px 0 rgba(255, 255, 255, 0.1);
}
.color-opt {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  cursor: pointer;
  border: 2px solid transparent;
  transition: transform 0.15s, border-color 0.15s, box-shadow 0.15s;
}
.color-opt:hover {
  transform: scale(1.18);
}
.color-opt.active {
  border-color: #fff;
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--app-primary) 45%, transparent);
  transform: scale(1.12);
}

/* ===== 笔记框（笔记模式专用，放大版） ===== */
/* 选择模式顶部提示条：引导长按选词，说明识屏已屏蔽 */
.select-hint {
  position: fixed;
  top: calc(64px + env(safe-area-inset-top));
  left: 50%;
  transform: translateX(-50%);
  z-index: 1200;
  display: flex;
  align-items: center;
  gap: 8px;
  max-width: calc(100vw - 28px);
  padding: 8px 12px 8px 14px;
  border-radius: 999px;
  font-size: 12.5px;
  color: #1f2937;
  background: rgba(255, 255, 255, 0.96);
  backdrop-filter: blur(18px) saturate(1.3);
  -webkit-backdrop-filter: blur(18px) saturate(1.3);
  border: 1px solid color-mix(in srgb, var(--app-primary) 45%, transparent);
  box-shadow: 0 10px 28px rgba(0, 0, 0, 0.16);
  animation: hintIn 0.35s cubic-bezier(0.22, 1, 0.36, 1) both;
  white-space: nowrap;
}
@keyframes hintIn {
  from {
    opacity: 0;
    transform: translate(-50%, -12px) scale(0.96);
  }
  to {
    opacity: 1;
    transform: translate(-50%, 0) scale(1);
  }
}
.select-hint .el-icon {
  flex-shrink: 0;
  color: var(--app-primary, #6366f1);
}
.select-hint-exit {
  flex-shrink: 0;
  margin-left: 4px;
  padding: 4px 12px;
  border: none;
  border-radius: 999px;
  font-size: 12.5px;
  color: #fff;
  background: var(--app-primary, #6366f1);
  cursor: pointer;
  transition: filter 0.15s ease;
}
.select-hint-exit:active {
  filter: brightness(0.9);
}

/* ===== 移动端划选操作条：划线工具开启并选中文字后，复制 / 划线 / 朗读 ===== */
.mark-toolbar {
  position: fixed;
  left: 12px;
  right: 12px;
  bottom: calc(16px + env(safe-area-inset-bottom));
  z-index: 1300;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 12px 10px;
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.98);
  backdrop-filter: blur(20px) saturate(1.4);
  -webkit-backdrop-filter: blur(20px) saturate(1.4);
  border: 1px solid color-mix(in srgb, var(--app-primary) 45%, transparent);
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.2);
  animation: markToolbarIn 0.28s cubic-bezier(0.22, 1, 0.36, 1) both;
}
@keyframes markToolbarIn {
  from {
    opacity: 0;
    transform: translateY(16px) scale(0.98);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}
.mt-text {
  font-size: 12.5px;
  line-height: 1.55;
  color: rgba(0, 0, 0, 0.7);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  word-break: break-all;
  padding: 0 2px;
}
.mt-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
.mt-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  border: none;
  cursor: pointer;
  font-size: 13px;
  color: #1f2937;
  background: rgba(0, 0, 0, 0.05);
  border-radius: 999px;
  padding: 8px 14px;
  transition: background 0.15s ease, transform 0.1s ease, filter 0.15s ease;
  -webkit-tap-highlight-color: transparent;
}
.mt-btn:active {
  transform: scale(0.96);
  filter: brightness(1.1);
}
.mt-btn.primary {
  color: #fff;
  background: linear-gradient(135deg, var(--app-primary), #8b5cf6);
  box-shadow: 0 6px 18px color-mix(in srgb, var(--app-primary) 45%, transparent);
}
.mt-close {
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 50%;
  cursor: pointer;
  color: #64748b;
  background: rgba(0, 0, 0, 0.05);
  transition: background 0.15s ease;
  -webkit-tap-highlight-color: transparent;
}
.mt-close:active {
  background: rgba(0, 0, 0, 0.1);
}
@keyframes selToolbarIn {
  from { opacity: 0; transform: translate(-50%, 14px); }
  to { opacity: 1; transform: translate(-50%, 0); }
}
/* 移动端"先选文字后处理"操作条：与侧边栏同款玻璃质感，底部停靠（始终在屏内） */
.sel-toolbar {
  position: fixed;
  left: 50%;
  bottom: max(16px, env(safe-area-inset-bottom, 0px));
  z-index: 1000;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 10px;
  background: rgba(255, 255, 255, 0.96);
  backdrop-filter: blur(14px);
  -webkit-backdrop-filter: blur(14px);
  border: 1px solid color-mix(in srgb, var(--app-primary) 35%, transparent);
  border-radius: 16px;
  box-shadow: 0 10px 36px rgba(0, 0, 0, 0.18), inset 0 1px 0 rgba(255, 255, 255, 0.6);
  animation: selToolbarIn 0.26s cubic-bezier(0.22, 1, 0.36, 1) both;
  /* 宽度留白避让左右翻页热区，避免遮挡导致按钮点不到 */
  max-width: calc(100vw - 2 * var(--pdf-zone) - 24px);
  transform: translateX(-50%);
}
.sel-tool {
  width: 42px;
  height: 42px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 12px;
  background: rgba(0, 0, 0, 0.04);
  color: #475569;
  cursor: pointer;
  -webkit-tap-highlight-color: transparent;
  transition: background 0.18s ease, transform 0.1s ease, box-shadow 0.18s ease;
}
.sel-tool:active {
  transform: scale(0.92);
  background: color-mix(in srgb, var(--app-primary) 22%, transparent);
  box-shadow: 0 0 12px color-mix(in srgb, var(--app-primary) 30%, transparent);
}
.sel-tool.sel-close {
  background: rgba(0, 0, 0, 0.04);
  color: #94a3b8;
}
.sel-preview {
  width: 26px;
  height: 18px;
  border-radius: 3px;
  box-sizing: border-box;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  border: 1px solid rgba(0, 0, 0, 0.15);
}
.sel-underline {
  position: absolute;
  left: 2px;
  right: 2px;
  bottom: 2px;
  height: 2px;
  border-radius: 2px;
}
.sel-strike {
  position: absolute;
  left: 2px;
  right: 2px;
  top: 50%;
  height: 2px;
  transform: translateY(-50%);
  border-radius: 2px;
}

.note-toolbar {
  background: rgba(255, 255, 255, 0.98);
  backdrop-filter: blur(20px) saturate(1.4);
  -webkit-backdrop-filter: blur(20px) saturate(1.4);
  border: 1px solid color-mix(in srgb, var(--app-primary) 40%, transparent);
  border-radius: 18px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.22), inset 0 1px 0 rgba(255, 255, 255, 0.6);
  padding: 18px 20px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 360px;
  max-width: calc(100vw - 24px);
  max-height: calc(100vh - 24px);
  overflow-y: auto;
  color: #1f2937;
}
.note-toolbar::after {
  content: '';
  position: absolute;
  left: 18px;
  right: 18px;
  bottom: 0;
  height: 2px;
  border-radius: 0 0 18px 18px;
  background: linear-gradient(90deg, transparent, color-mix(in srgb, var(--app-primary) 70%, transparent), transparent);
  opacity: 0.9;
}
.nt-header {
  font-size: 14px;
  font-weight: 600;
  color: #e5e7eb;
  display: flex;
  align-items: center;
  gap: 8px;
}
.nt-header::before {
  content: '';
  width: 4px;
  height: 14px;
  border-radius: 2px;
  background: var(--app-gradient);
}
.nt-text {
  font-size: 13px;
  line-height: 1.6;
  color: rgba(229, 231, 235, 0.9);
  background: color-mix(in srgb, var(--app-primary) 12%, transparent);
  border: 1px solid color-mix(in srgb, var(--app-primary) 24%, transparent);
  border-radius: 10px;
  padding: 8px 10px;
  max-height: 96px;
  overflow-y: auto;
  word-break: break-word;
}
.nt-note {
  display: flex;
}
.nt-note :deep(.el-textarea__inner) {
  background: rgba(255, 255, 255, 0.07);
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--app-primary) 30%, transparent) inset;
  border-radius: 10px;
  color: #e5e7eb;
  font-size: 13px;
  line-height: 1.6;
}
.nt-note :deep(.el-textarea__inner:focus) {
  box-shadow: 0 0 0 1.5px var(--app-primary) inset;
  background: color-mix(in srgb, var(--app-primary) 12%, transparent);
}
.nt-note :deep(.el-textarea__inner::placeholder) {
  color: rgba(229, 231, 235, 0.45);
}
.nt-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  border-top: 1px solid color-mix(in srgb, var(--app-primary) 20%, transparent);
  padding-top: 12px;
}
.nt-actions-right {
  display: flex;
  align-items: center;
  gap: 8px;
}
.nt-read.el-button {
  margin-left: 0;
  display: inline-flex;
  align-items: center;
  background: rgba(34, 211, 238, 0.12);
  border: 1px solid rgba(34, 211, 238, 0.4);
  color: #a5f3fc;
  border-radius: 10px;
  padding: 5px 12px;
}
.nt-read.el-button:hover {
  background: rgba(34, 211, 238, 0.2);
  border-color: rgba(34, 211, 238, 0.6);
  color: #cffafe;
}
.nt-cancel.el-button {
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.14);
  color: rgba(229, 231, 235, 0.8);
  border-radius: 10px;
}
.nt-cancel.el-button:hover {
  background: rgba(255, 255, 255, 0.12);
  border-color: rgba(255, 255, 255, 0.25);
  color: #fff;
}
.nt-save.el-button {
  background: var(--app-gradient);
  border: none;
  color: #fff;
  border-radius: 10px;
  padding: 5px 18px;
}
.nt-save.el-button:hover {
  filter: brightness(1.12);
}
/* PDF 阅读器 — 连续滚动 */
.pdf-container {
  width: 100%;
  height: 100%;
  overflow-y: auto;
  overflow-x: auto;
  background:
    radial-gradient(1200px 600px at 15% -10%, color-mix(in srgb, var(--app-primary) 14%, transparent), transparent 60%),
    radial-gradient(1000px 500px at 105% 110%, rgba(34, 211, 238, 0.1), transparent 60%),
    linear-gradient(180deg, #1b1b33, #232338);
}

.pdf-scroll-viewport {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 20px 0;
  gap: 16px;
}

/* 页面分组容器：单页时一组一页居中；双页时一组两页并排 */
.pdf-page-group {
  display: flex;
  justify-content: center;
  align-items: flex-start;
  width: max-content;
  margin-inline: auto;
}

.pdf-scroll-viewport.two-page .pdf-page-group {
  gap: 14px;
}

.pdf-page-wrapper {
  position: relative;
  background: #fff;
  border-radius: 4px;
  overflow: hidden;
  box-shadow: 0 6px 30px rgba(0,0,0,0.45);
  line-height: 1;
  flex-shrink: 0;
}

.pdf-canvas {
  display: block;
}

/* === PDF 阅读主题适配（随设置页主题切换） ===
   桌面背景与纸张色复用 EPUB 的同一套取值（getEpubDesktopBg / getEpubPaperBg），
   由 .reader-body 上的 --pdf-desktop-bg / --pdf-paper-bg 注入，
   保证 PDF 与 EPUB 在各主题下观感一致（"纸张放在桌面上"的层次）。
   注：PDF 内容本身始终是白底黑字，暗色主题下靠 canvas 反色滤镜实现，故滤镜按主题保留。 */
.pdf-container,
.pdf-page-wrapper,
.pdf-canvas {
  transition: background 0.3s, filter 0.3s;
}
.pdf-container {
  background: var(--pdf-desktop-bg, #eef0f2);
}
.pdf-page-wrapper {
  background: var(--pdf-paper-bg, #ffffff);
}

/* 日间 */
.reader-body[data-theme='light'] .pdf-page-wrapper {
  box-shadow: 0 6px 24px rgba(31, 38, 135, 0.16);
}
.reader-body[data-theme='light'] .pdf-canvas {
  filter: none;
}

/* 护眼（米黄纸 + 深棕字） */
.reader-body[data-theme='sepia'] .pdf-page-wrapper {
  box-shadow: 0 6px 24px rgba(120, 96, 40, 0.18);
}
.reader-body[data-theme='sepia'] .pdf-canvas {
  filter: brightness(0.94) sepia(0.5) saturate(0.9) contrast(0.98);
}

/* 夜间（深蓝纸 + 浅色字，颜色保真反转） */
.reader-body[data-theme='dark'] .pdf-page-wrapper {
  box-shadow: 0 6px 30px rgba(0, 0, 0, 0.6);
}
.reader-body[data-theme='dark'] .pdf-canvas {
  filter: invert(0.9) hue-rotate(180deg) brightness(0.95) contrast(0.95);
}

/* 纯黑（黑白反转） */
.reader-body[data-theme='pitch'] .pdf-page-wrapper {
  box-shadow: 0 6px 30px rgba(0, 0, 0, 0.8);
}
.reader-body[data-theme='pitch'] .pdf-canvas {
  filter: invert(1) hue-rotate(180deg);
}

.pdf-text-layer {
  position: absolute;
  top: 0;
  left: 0;
  line-height: 1;
  -webkit-user-select: text;
  user-select: text;
  -webkit-touch-callout: none;
  touch-action: pan-y;
}

.pdf-text-layer span {
  cursor: text;
}

.pdf-highlights-overlay {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: 5;
  pointer-events: none;
}


.pdf-page-indicator {
  position: absolute;
  bottom: 50px;
  left: 50%;
  transform: translateX(-50%);
  background: rgba(18, 18, 40, 0.72);
  backdrop-filter: blur(10px);
  border: 1px solid color-mix(in srgb, var(--app-primary) 35%, transparent);
  color: #e2e8f0;
  padding: 5px 18px;
  border-radius: 16px;
  font-size: 13px;
  z-index: 20;
  pointer-events: none;
  box-shadow: 0 6px 20px rgba(0,0,0,0.35);
  font-variant-numeric: tabular-nums;
}

/* 渲染中提示 */
.rendering-hint {
  position: absolute;
  top: 16px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 6px;
  background: rgba(18, 18, 40, 0.72);
  backdrop-filter: blur(10px);
  border: 1px solid color-mix(in srgb, var(--app-primary) 30%, transparent);
  color: #e2e8f0;
  padding: 6px 16px;
  border-radius: 16px;
  font-size: 12px;
  z-index: 25;
  pointer-events: none;
  box-shadow: 0 6px 20px rgba(0,0,0,0.35);
}

.pdf-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #b8c5d6;
  gap: 12px;
}

.pdf-loading p {
  margin: 0;
  font-size: 14px;
  background: linear-gradient(90deg, #a5b4fc, #67e8f9);
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
}

/* EPUB 阅读器 */
.epub-container {
  width: 100%;
  height: 100%;
}

#epub-viewer {
  width: 100%;
  height: 100%;
}

/* 连续滚动模式：容器纵向滚动，章节 iframe 高度自适应内容；
   侧栏与纸张均在 iframe 内实现（body::before/::after 固定定位 + 纸张避让），
   容器仅负责纵向滚动，底部保留少量留白 */
#epub-viewer.epub-scroll-viewport {
  position: relative;
  overflow-y: auto;
  overflow-x: hidden;
  overscroll-behavior: contain;
  -webkit-overflow-scrolling: touch;
  padding-bottom: 40px;
}

#epub-viewer.epub-scroll-viewport .epub-scroll-iframe {
  box-sizing: border-box;
}

.other-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #e0e0e0;
}

/* 翻页区域 */
.page-zone {
  position: absolute;
  top: 0;
  bottom: 0;
  width: var(--pdf-zone);
  z-index: 10;
  cursor: pointer;
  transition: background 0.2s, opacity 0.2s ease;
}

/* 选择模式：禁用翻页热区，避免长按选词时误翻页 */
.reader-body.select-mode .page-zone {
  opacity: 0;
  pointer-events: none;
}

.page-zone:hover {
  background: linear-gradient(to right, color-mix(in srgb, var(--app-primary) 12%, transparent), transparent);
}

.page-zone.page-next:hover {
  background: linear-gradient(to left, color-mix(in srgb, var(--app-primary) 12%, transparent), transparent);
}

.page-prev::before,
.page-next::before {
  content: '';
  position: absolute;
  top: 50%;
  width: 14px;
  height: 14px;
  border-left: 2.5px solid var(--app-primary);
  border-bottom: 2.5px solid var(--app-primary);
  opacity: 0.35;
  transition: opacity 0.25s;
}

.page-prev::before {
  left: 15px;
  transform: translateY(-50%) rotate(45deg);
}

.page-next::before {
  right: 15px;
  transform: translateY(-50%) rotate(-135deg);
}

.page-zone:hover::before {
  opacity: 1;
}

.page-zone.disabled {
  pointer-events: none;
  cursor: default;
}

.page-prev { left: 0; }
.page-next { right: 0; }

/* ===== 朗读设置面板（工具栏"阅读"按钮弹出） ===== */
.tools-group {
  position: relative;
}
.tts-panel {
  position: fixed;
  z-index: 2500;
  width: 340px;
  max-height: calc(100vh - 16px);
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 14px;
  background: rgba(255, 255, 255, 0.98);
  backdrop-filter: blur(20px) saturate(1.4);
  -webkit-backdrop-filter: blur(20px) saturate(1.4);
  border: 1px solid color-mix(in srgb, var(--app-primary) 35%, transparent);
  border-radius: 14px;
  box-shadow: 0 12px 36px rgba(0, 0, 0, 0.2), inset 0 1px 0 rgba(255, 255, 255, 0.6);
  color: #1f2937;
}
.tts-panel-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 600;
  color: #1f2937;
  padding-bottom: 8px;
  border-bottom: 1px solid color-mix(in srgb, var(--app-primary) 20%, transparent);
}
.tts-panel-title .el-icon {
  color: #22d3ee;
}
.tts-panel-title .tts-panel-close.el-button {
  margin-left: auto;
  width: 24px;
  height: 24px;
  padding: 0;
  background: rgba(0, 0, 0, 0.04);
  border: 1px solid rgba(0, 0, 0, 0.08);
  color: #475569;
}
.tts-panel-title .tts-panel-close.el-button .el-icon {
  color: inherit;
}
.tts-panel-title .tts-panel-close.el-button:hover {
  background: rgba(0, 0, 0, 0.06);
  border-color: rgba(0, 0, 0, 0.12);
  color: var(--app-primary, #6366f1);
}
/* 设置区：音色 / 语气 / 倍速 一行 */
.tts-options-row {
  display: flex;
  gap: 10px;
}
.tts-opt {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.tts-opt-label {
  font-size: 11px;
  color: rgba(229, 231, 235, 0.55);
}
.tts-opt-speed {
  flex: 1.3;
}
.tts-opt-speed-row {
  display: flex;
  align-items: center;
  gap: 6px;
  min-height: 30px;
}
.tts-select {
  flex: 1;
  min-width: 0;
}
.tts-select :deep(.el-select__wrapper) {
  background: rgba(255, 255, 255, 0.06);
  box-shadow: 0 0 0 1px rgba(255, 255, 255, 0.14) inset;
  border-radius: 8px;
  padding: 2px 10px;
  min-height: 30px;
  transition: box-shadow 0.15s, background 0.15s;
}
.tts-select :deep(.el-select__wrapper:hover) {
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--app-primary) 45%, transparent) inset;
}
.tts-select :deep(.el-select__placeholder),
.tts-select :deep(.el-select__selected-item) {
  font-size: 12px;
  color: rgba(229, 231, 235, 0.9);
}
.tts-select :deep(.el-select__caret) {
  color: rgba(229, 231, 235, 0.5);
}
.tts-panel-speed {
  flex: 1;
  min-width: 0;
}
.tts-panel-speed :deep(.el-slider__runway) {
  height: 4px;
  background: rgba(255, 255, 255, 0.15);
}
.tts-panel-speed :deep(.el-slider__bar) {
  height: 4px;
  background: linear-gradient(90deg, var(--app-primary), color-mix(in srgb, var(--app-primary) 55%, #22d3ee));
}
.tts-panel-speed :deep(.el-slider__button) {
  width: 12px;
  height: 12px;
  border: 2px solid #fff;
  background: var(--app-primary);
  box-shadow: 0 0 10px color-mix(in srgb, var(--app-primary) 70%, transparent);
}
.tts-panel-speed-val {
  font-size: 12px;
  color: #cbd5e1;
  min-width: 30px;
  text-align: right;
  flex-shrink: 0;
}
.tts-panel-read.el-button {
  margin-left: 0;
  width: 100%;
  height: 40px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  background: var(--app-gradient);
  border: none;
  color: #fff;
  border-radius: 10px;
  padding: 0 14px;
  font-weight: 600;
  letter-spacing: 0.5px;
  box-shadow: 0 4px 14px color-mix(in srgb, var(--app-primary) 35%, transparent), inset 0 1px 0 rgba(255, 255, 255, 0.25);
  transition: transform 0.2s ease, filter 0.2s ease, box-shadow 0.2s ease;
}
.tts-panel-read.el-button .el-icon {
  color: rgba(255, 255, 255, 0.92);
}
.tts-panel-read.el-button:hover {
  filter: brightness(1.1);
  color: #fff;
  transform: translateY(-1px);
  box-shadow: 0 6px 18px color-mix(in srgb, var(--app-primary) 50%, transparent), inset 0 1px 0 rgba(255, 255, 255, 0.3);
}
.tts-panel-read.el-button:active {
  transform: translateY(0);
  filter: brightness(0.95);
}
/* 待选状态：等待用户在页面上选中文字 */
.tts-panel-read.el-button.is-pending {
  background: linear-gradient(135deg, #f59e0b, #f97316);
  box-shadow: 0 0 0 2px rgba(245, 158, 11, 0.35), 0 4px 16px rgba(245, 158, 11, 0.35), inset 0 1px 0 rgba(255, 255, 255, 0.3);
  animation: ttsPendingPulse 1.6s ease-in-out infinite;
}
.tts-panel-read.el-button.is-pending:hover {
  transform: none;
  filter: brightness(1.05);
}
@keyframes ttsPendingPulse {
  0%, 100% {
    box-shadow: 0 0 0 2px rgba(245, 158, 11, 0.35), 0 4px 14px rgba(245, 158, 11, 0.3);
  }
  50% {
    box-shadow: 0 0 0 4px rgba(245, 158, 11, 0.12), 0 4px 20px rgba(245, 158, 11, 0.55);
  }
}
/* 朗读控制区（开始阅读 + 状态合并置顶） */
.tts-actions {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.tts-actions-status {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.08);
}
.tts-actions-text {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #cbd5e1;
  line-height: 1.5;
  max-height: 40px;
  overflow-y: auto;
}
.tts-actions-text .el-icon.is-loading {
  color: #22d3ee;
  flex-shrink: 0;
}
.tts-actions-ctrl {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}
.tts-actions-ctrl .el-button {
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.14);
  color: rgba(229, 231, 235, 0.85);
}
.tts-actions-ctrl .el-button:hover {
  background: rgba(255, 255, 255, 0.12);
  color: #fff;
}
.tts-panel-play.el-button {
  background: var(--app-gradient);
  border: none;
  color: #fff;
}
.tts-panel-play.el-button:hover {
  filter: brightness(1.1);
  color: #fff;
}
.tts-panel-stop.el-button {
  color: #f87171;
}

/* 头部朗读按钮激活态 */
.tts-header-active {
  color: #22d3ee !important;
  background: rgba(34, 211, 238, 0.12) !important;
  border-radius: 8px;
}

.tts-header-active::after {
  content: '';
  display: inline-block;
  width: 6px;
  height: 6px;
  margin-left: 4px;
  border-radius: 50%;
  background: #22d3ee;
  box-shadow: 0 0 8px #22d3ee;
  animation: ttsPulse 1.4s ease-in-out infinite;
}

@keyframes ttsPulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.25; }
}

/* PDF 朗读高亮 */
:deep(.pdf-text-layer span.tts-speaking) {
  color: transparent !important;
  background: rgba(250, 204, 21, 0.5);
  border-radius: 3px;
  padding: 3px;
}

/* 设置抽屉提示文案 */
.settings-save-tip {
  margin: 0 0 4px;
  font-size: 12px;
  color: #94a3b8;
  line-height: 1.5;
}

.setting-tip {
  margin: 6px 0 0;
  font-size: 12px;
  color: #94a3b8;
  line-height: 1.5;
}

/* ===== 目录面板：与标记列表面板同构（复用 fs-ann-* 体系） ===== */

/* 目录项复用 fs-ann-item 基础布局，追加层级/树形/页码 */
.toc-item {
  padding-top: 10px;
  padding-bottom: 10px;
}

.toc-item .toc-title-text {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  color: #4b5563;
  line-height: 1.5;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 一级标题：更大更醒目，与子级间隔开 */
.toc-item.l1 {
  margin-top: 4px;
}

.toc-item.l1 .toc-title-text {
  font-size: 14px;
  font-weight: 600;
  color: #1a1a2e;
}

.toc-item.sub .toc-title-text {
  font-size: 12.5px;
  color: #6b7280;
}

.toc-item:hover .toc-title-text {
  color: #111827;
}

/* 激活态：沿用 fs-ann-item.current 的左侧渐变竖线 + 浅渐变背景 */
.toc-item.current .toc-title-text {
  color: var(--app-primary-dark);
  font-weight: 600;
}

/* 子级：左侧短横线枝干，形成树形引导 */
.toc-branch {
  position: relative;
  top: -1px;
  flex-shrink: 0;
  width: 10px;
  height: 1px;
  margin-left: 2px;
  background: #c7d2fe;
}

/* 页码：弱化、等宽 */
.toc-page {
  flex-shrink: 0;
  font-size: 11px;
  color: #9ca3af;
  font-variant-numeric: tabular-nums;
  letter-spacing: 0.4px;
}

.toc-item:hover .toc-page {
  color: #6b7280;
}

.toc-item.current .toc-page {
  color: var(--app-primary);
}

/* 批注侧边栏（复用全屏标记列表的全局样式 fs-ann-*） */
.annotation-drawer-body {
  padding-right: 4px;
}

/* 抽屉头部与全屏标记面板一致 */
.reader-page :deep(.el-drawer__header) {
  margin-bottom: 0;
  padding: 12px 16px;
  border-bottom: 1px solid #ebeef5;
  color: #1a1a2e;
  font-weight: 600;
}

.reader-page :deep(.el-drawer__body) {
  padding: 0;
}

/* 设置面板 */
.settings-panel {
  padding: 0 20px 20px;
}

.settings-panel h4 {
  margin: 20px 0 12px 0;
  font-size: 14px;
  color: #606266;
}

.theme-options {
  display: flex;
  gap: 8px;
}

.theme-option {
  width: 60px;
  height: 40px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  border: 2px solid transparent;
  font-size: 13px;
}

.theme-option.active {
  border-color: var(--app-primary);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--app-primary) 25%, transparent);
}

/* 页面布局选项（与主题选项统一卡片风格） */
.layout-options {
  display: flex;
  gap: 8px;
}

.layout-option {
  flex: 1;
  height: 44px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  cursor: pointer;
  border: 2px solid transparent;
  font-size: 13px;
  color: #606266;
  background: #f5f7fa;
  transition: all 0.2s;
}

.layout-option:hover {
  background: #eef0f8;
}

.layout-option.active {
  border-color: var(--app-primary);
  color: var(--app-primary-dark);
  background: color-mix(in srgb, var(--app-primary) 8%, transparent);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--app-primary) 25%, transparent);
}

.setting-value {
  text-align: center;
  color: #909399;
  font-size: 13px;
  margin-top: 4px;
}

/* 全屏模式调整 */
.reader-body.fullscreen .page-zone {
  bottom: 0;
}

.reader-body.fullscreen .pdf-page-indicator {
  bottom: 16px;
}

/* ===== 手机端适配 ===== */
@media (max-width: 768px) {
  .reader-page {
    height: 100dvh;
  }

  .reader-header {
    height: calc(54px + env(safe-area-inset-top));
    padding: env(safe-area-inset-top) 10px 0;
  }

  .reader-header .btn-text {
    display: none;
  }

  /* 移动端精简顶部按钮：隐藏低频的「续播」「设置」 */
  .reader-header .mobile-hide {
    display: none !important;
  }

  /* PDF 缩放百分比文本在移动端隐藏，只保留缩放图标按钮 */
  .reader-header .zoom-level {
    display: none;
  }

  /* 选择模式按钮仅移动端显示，激活态高亮 */
  .reader-header .mobile-only {
    display: inline-flex;
  }
  .reader-header .mobile-only.active {
    color: var(--app-primary, #6366f1);
    background: color-mix(in srgb, var(--app-primary, #6366f1) 12%, transparent);
  }

  .reader-header .title-area {
    font-size: 15px;
    min-width: 0;
    flex: 1;
  }

  .reader-header .book-title {
    max-width: 32vw;
  }

  .reader-header .zoom-level {
    display: none;
  }

  .reader-header .header-left {
    flex-shrink: 0;
    min-width: 0;
  }

  .reader-header .header-right {
    gap: 2px;
  }

  /* 手机端按钮压缩为紧凑图标按钮，避免一排溢出 */
  .reader-header .header-right :deep(.el-button) {
    padding: 6px;
    margin: 0 !important;
  }
  .reader-header .header-right :deep(.el-button + .el-button) {
    margin-left: 0 !important;
  }

  /* 移动端进度条稍加高，便于直接点按 */
  .reader-progress {
    height: 4px;
  }

  /* 手机端工具栏默认收起（仅显示展开按钮，可手动展开） */
  .tools-sidebar {
    right: 8px;
  }
  .tools-sidebar.collapsed {
    padding: 4px;
  }

  /* 非全屏标记面板：手机全宽 */
  .ann-panel {
    right: 12px;
    left: 12px;
    width: auto;
  }

  /* 全屏工具栏适配刘海安全区 */
  .fullscreen-toolbar {
    height: calc(50px + env(safe-area-inset-top));
    padding-top: env(safe-area-inset-top);
  }

  /* 笔记框移动端适配：居中底部抽屉，确保完整在屏内（!important 覆盖 toolbarFixedStyle 内联 left/top） */
  .note-toolbar {
    position: fixed !important;
    left: 50% !important;
    right: auto !important;
    top: auto !important;
    bottom: max(16px, env(safe-area-inset-bottom, 0px)) !important;
    transform: translateX(-50%) !important;
    min-width: 0;
    width: calc(100vw - 24px);
    max-width: calc(100vw - 24px);
    max-height: calc(100vh - 32px);
    padding: 16px;
  }

  /* 翻页热区缩窄并禁止选中，避免遮挡文本长按选择 */
  .page-zone {
    width: 6%;
    -webkit-user-select: none;
    user-select: none;
  }

  /* 全屏工具栏压缩 */
  .fullscreen-toolbar {
    padding: 8px 12px;
  }

  .fullscreen-toolbar .btn-text {
    display: none;
  }

  .fullscreen-toolbar .el-button + .el-button {
    margin-left: 4px;
  }

  /* PDF 页码指示器 */
  .pdf-page-indicator {
    font-size: 11px;
    padding: 4px 10px;
  }
}
</style>

<!-- 全局样式：动态创建的划线元素不受 Vue scoped CSS 约束 -->
<style>
.ann-overlay {
  position: absolute;
  cursor: pointer;
  border-radius: 1px;
}

/* PDF 朗读高亮：按句子字符矩形绘制在页面 overlay 上，只盖当前句、不波及下一句 */
.tts-speaking-rect {
  position: absolute;
  pointer-events: none;
  border-radius: 2px;
  background: rgba(250, 204, 21, 0.42);
  box-shadow: 0 0 0 1px rgba(245, 158, 11, 0.28);
}

/* 区域画笔预览（拖动画矩形时实时显示，松开即落定） */
.region-preview {
  position: absolute;
  pointer-events: none;
  z-index: 60;
  outline: 1px dashed color-mix(in srgb, var(--ann-color) 90%, #000);
  outline-offset: 0;
  transition: none;
}
.region-preview.ann-highlight {
  opacity: 0.45;
}
.region-preview.ann-underline,
.region-preview.ann-wavy {
  background: transparent !important;
}
.region-preview.ann-wavy {
  border-bottom: 2px dashed var(--ann-color);
}

.ann-overlay:hover {
  outline: 1px dashed var(--app-primary);
}

.ann-highlight {
  background-color: var(--ann-color);
  opacity: 0.4;
  border-radius: 2px;
  pointer-events: auto;
}

.ann-underline {
  border-bottom: 2px solid var(--ann-color);
  background: transparent;
  height: calc(100% + 2px);
  pointer-events: auto;
}

.ann-wavy {
  background: transparent;
  pointer-events: auto;
}

.ann-strikethrough {
  background: linear-gradient(transparent 45%, var(--ann-color) 45%, var(--ann-color) 55%, transparent 55%);
  pointer-events: auto;
}

/* ========== 全屏顶部工具栏 ========== */
.fullscreen-toolbar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 200;
  height: 50px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  background: rgba(255, 255, 255, 0.92);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border-bottom: 1px solid rgba(0, 0, 0, 0.08);
  color: #1f2937;
  transition: opacity 0.3s;
}

.fs-center {
  font-size: 13px;
  color: #94a3b8;
  font-variant-numeric: tabular-nums;
}

.fs-left, .fs-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

/* 「标记」按钮被 el-badge 包裹（角标显示批注数量）。
   el-badge 默认是 inline-block 并带自身 vertical-align / line-height，
   会把被包裹的按钮推离相邻按钮的基线，看起来与「读后感」「退出」不在同一水平线；
   这里让 badge 自身参与 flex 居中即可对齐。 */
.fs-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  line-height: 1;
}

.fs-page-info {
  font-size: 14px;
  color: #606266;
  margin-right: 8px;
}

.fs-ann-toggle {
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  padding: 4px 10px;
  border-radius: 16px;
  background: #f5f7fa;
  transition: background 0.2s;
}

.fs-ann-toggle:hover {
  background: #e4e7ed;
}

.fs-ann-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: 9px;
  background: #f56c6c;
  color: #fff;
  font-size: 11px;
  font-weight: 600;
}

/* ========== 全屏批注浮层 ========== */
/* 全屏时工具栏高度为 50px + 刘海安全区，面板从其下方开始，避免被"标题栏"遮挡 */
.fullscreen-ann-panel {
  position: fixed;
  top: calc(50px + env(safe-area-inset-top) + 12px);
  right: 12px;
  width: 330px;
  max-height: calc(100vh - 50px - env(safe-area-inset-top) - 24px);
  z-index: 201;
  background: rgba(255, 255, 255, 0.97);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  border: 1px solid color-mix(in srgb, var(--app-primary) 25%, transparent);
  border-radius: 16px;
  box-shadow: 0 14px 44px rgba(0,0,0,0.4);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* 非全屏标记面板：复用全屏样式，改为阅读区内绝对定位。
   z-index 高于顶部标题栏（100），即使发生重叠也始终显示在标题栏之上 */
.ann-panel {
  position: absolute;
  top: 12px;
  right: 12px;
  max-height: calc(100% - 24px);
  z-index: 300;
  animation: annPanelIn 0.2s ease;
}

@keyframes annPanelIn {
  from { opacity: 0; transform: translateY(-8px); }
  to { opacity: 1; transform: translateY(0); }
}

.fs-ann-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid #ebeef5;
}

.fs-ann-header span {
  font-size: 14px;
  font-weight: 600;
  color: #1a1a2e;
}

.fs-ann-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px 0;
}

.fs-ann-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 16px;
  cursor: pointer;
  transition: background 0.2s;
  border-left: 3px solid transparent;
}

.fs-ann-item:hover {
  background: #f5f7fa;
}

.fs-ann-item.current {
  border-left-color: var(--app-primary);
  background: linear-gradient(90deg, color-mix(in srgb, var(--app-primary) 10%, transparent), color-mix(in srgb, var(--app-primary) 4%, transparent));
}

.fs-ann-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 4px;
}

.fs-ann-note-icon {
  color: var(--app-primary);
  font-size: 12px;
  display: inline-flex;
  align-items: center;
}

/* 划线条目右侧删除按钮（半透明常显，悬停加深） */
.fs-ann-delete {
  flex-shrink: 0;
  align-self: flex-start;
  opacity: 0.45;
  transition: opacity 0.2s;
}

.fs-ann-delete .el-button {
  background: transparent;
  transition: background 0.2s;
}

.fs-ann-item:hover .fs-ann-delete {
  opacity: 1;
}

.fs-ann-item:hover .fs-ann-delete .el-button {
  color: #f43f5e;
}

.fs-ann-color {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
  margin-top: 6px;
}

.fs-ann-body {
  flex: 1;
  min-width: 0;
}

/* 样式预览块（全屏面板与抽屉共用） */
.fs-ann-style {
  flex-shrink: 0;
  margin-top: 2px;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.15);
}

/* 标记文本（全屏面板与抽屉共用） */
.fs-ann-text {
  font-size: 13px;
  color: #303133;
  line-height: 1.5;
  word-break: break-all;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.fs-ann-note {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
  word-break: break-all;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.fs-ann-empty {
  padding: 40px 16px;
  text-align: center;
  color: #94a3b8;
  font-size: 13px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
}
.fs-ann-empty p {
  margin: 4px 0 0;
  font-weight: 600;
  color: #64748b;
}
.fs-ann-empty span {
  font-size: 12px;
  color: #94a3b8;
}

/* ========== 读后感面板 ========== */
.fs-ann-count {
  color: var(--app-primary);
  font-size: 15px;
}
.review-item {
  margin: 8px 10px;
  padding: 12px 14px;
  border-radius: 12px;
  background: linear-gradient(135deg, #ffffff, #f8fafc);
  border: 1px solid #eef2f7;
  box-shadow: 0 4px 14px rgba(31, 38, 135, 0.06);
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}
.review-item:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 20px rgba(31, 38, 135, 0.12);
}
.review-item-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.review-item-title {
  font-size: 14px;
  font-weight: 600;
  color: #1e293b;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.review-item-excerpt {
  margin-top: 4px;
  font-size: 12px;
  color: #64748b;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.review-item-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px dashed #eef2f7;
}
.review-item-time {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: #94a3b8;
}
.review-item-actions {
  display: flex;
  align-items: center;
}
.review-write-btn {
  padding: 12px 16px;
  border-top: 1px solid #f0f2f5;
}

/* ========== 底部进度条批注角标 ========== */


/* ===== 移动端：标记面板全宽显示 ===== */
@media (max-width: 768px) {
  .fullscreen-ann-panel {
    right: 12px;
    left: 12px;
    width: auto;
    max-height: calc(100dvh - 50px - env(safe-area-inset-top) - 24px);
  }

  .fs-ann-header {
    padding: 12px 14px;
  }

  .fs-ann-title {
    font-size: 14px;
  }

  .fs-ann-list {
    padding: 8px;
  }

  /* 移动端无 hover，删除按钮常显 */
  .fs-ann-delete {
    opacity: 1;
  }
  .fs-ann-delete .el-button {
    padding: 6px;
  }
}

/* 朗读设置下拉面板（popper 挂载到 body，需全局样式） */
.tts-popper {
  z-index: 3000 !important;
  border-radius: 10px;
  background: #1b1b33;
  border: 1px solid rgba(255, 255, 255, 0.1);
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.5);
}
.tts-popper .el-select-dropdown__item {
  font-size: 12px;
  color: rgba(229, 231, 235, 0.85);
  height: 30px;
  line-height: 30px;
}
.tts-popper .el-select-dropdown__item.hover,
.tts-popper .el-select-dropdown__item:hover {
  background: rgba(255, 255, 255, 0.08);
}
.tts-popper .el-select-dropdown__item.is-selected {
  color: #fff;
  background: color-mix(in srgb, var(--app-primary) 25%, transparent);
  font-weight: 600;
}
.tts-popper .el-select-group__title {
  font-size: 11px;
  color: rgba(229, 231, 235, 0.5);
  padding-left: 12px;
}

/* 内嵌到 FoliateReader 外壳时：隐藏 PDF 自带的头部/浮层/设置/朗读面板，
   由外壳统一提供顶部工具栏与目录、批注、设置抽屉；保留左侧画笔工具栏与选区工具条以创建批注 */
.foliate-reader .reader-page.embedded {
  position: absolute;
  inset: 0;
  height: 100%;
}
.foliate-reader .reader-page.embedded .reader-header,
.foliate-reader .reader-page.embedded .fullscreen-toolbar,
.foliate-reader .reader-page.embedded .fullscreen-ann-panel,
.foliate-reader .reader-page.embedded .settings-panel,
.foliate-reader .reader-page.embedded .tts-panel {
  display: none !important;
}
</style>
