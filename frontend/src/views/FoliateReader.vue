<template>
  <div ref="rootEl" class="foliate-reader" :class="{ fullscreen }" :style="readerRootStyle">
    <!-- 顶部工具栏 -->
    <header class="fr-header" :class="{ hidden: fullscreen }">
      <button class="fr-icon-btn" title="返回" @click="goBack">
        <el-icon><ArrowLeft /></el-icon>
      </button>
      <div class="fr-title" :title="bookInfo?.title">
        <span class="fr-title-book">{{ bookInfo?.title || '阅读' }}</span>
        <span v-if="currentChapterTitle" class="fr-title-sep">·</span>
        <span v-if="currentChapterTitle" class="fr-title-chapter" :title="currentChapterTitle">{{ currentChapterTitle }}</span>
      </div>
      <div class="fr-spacer" />
      <template v-if="isPdf">
        <button class="fr-icon-btn" title="缩小" @click="pdfZoomOut"><el-icon><ZoomOut /></el-icon></button>
        <span class="fr-zoom">{{ pdfZoomPct }}%</span>
        <button class="fr-icon-btn" title="放大" @click="pdfZoomIn"><el-icon><ZoomIn /></el-icon></button>
      </template>
      <button class="fr-icon-btn" title="目录" @click="showToc = !showToc">
        <el-icon><List /></el-icon>
      </button>
      <button v-if="!isPdf" class="fr-icon-btn" title="搜索" @click="toggleSearch">
        <el-icon><Search /></el-icon>
      </button>
      
      <button class="fr-icon-btn" title="批注与笔记" @click="showAnnDrawer = !showAnnDrawer">
        <el-badge :value="annCount" :hidden="annCount === 0" :max="99">
          <el-icon><Notebook /></el-icon>
        </el-badge>
      </button>
      <!-- 语音朗读（划线朗读）：朗读中再点一次停止；EPUB 与 PDF 通用；手机端隐藏 -->
      <button v-if="true" class="fr-icon-btn mobile-hide" :class="{ active: tts.active }" :title="tts.active ? '停止朗读' : '语音朗读'" @click="onTtsHeaderClick">
        <el-icon><Microphone /></el-icon>
      </button>
      <button class="fr-icon-btn" title="设置" @click="showSettings = !showSettings">
        <el-icon><Setting /></el-icon>
      </button>
      <button class="fr-icon-btn mobile-hide" title="全屏" @click="toggleFullscreen">
        <el-icon><FullScreen /></el-icon>
      </button>
    </header>

    <!-- 移动端选择模式提示条：划线/笔记工具开启后自动进入，长按文字选中即可划线 -->
    <div v-if="selectMode" class="fr-select-hint">
      <span>{{ drawToolActive ? '已开启划线：长按文字选中后可划线 / 复制' : '长按文字即可选中，可划线 / 复制' }}</span>
      <button type="button" class="fr-select-hint-btn" @click="exitSelectMode">取消</button>
    </div>

    <!-- 搜索面板 -->
    <div v-if="searchOpen" class="fr-search-panel">
      <el-input
        v-model="searchQuery"
        placeholder="输入关键字搜索"
        clearable
        size="default"
        @keyup.enter="runSearch"
      >
        <template #prepend>
          <el-button :loading="searching" @click="runSearch">搜索</el-button>
        </template>
      </el-input>
      <div v-if="searchResults.length" class="fr-search-list">
        <div
          v-for="(r, i) in searchResults"
          :key="i"
          class="fr-search-item"
          @click="jumpToCfi(r.cfi)"
        >
          <span class="fr-search-excerpt">{{ r.excerpt }}</span>
        </div>
      </div>
      <div v-else-if="searchDone" class="fr-search-empty">未找到结果</div>
    </div>

    <!-- 目录抽屉 -->
    <transition name="fr-slide">
      <aside v-if="showToc" class="fr-drawer fr-drawer-right">
        <div class="fr-drawer-head">
          <span>目录</span>
          <el-icon class="fr-close" @click="showToc = false"><Close /></el-icon>
        </div>
        <div class="fr-toc">
          <template v-if="!isPdf">
            <div
              v-for="(item, i) in flatToc"
              :key="i"
              class="fr-toc-item"
              :style="{ paddingLeft: 12 + item.depth * 16 + 'px' }"
              @click="goToHref(item.href)"
            >
              {{ item.label }}
            </div>
            <div v-if="!flatToc.length" class="fr-empty">暂无目录</div>
          </template>
          <template v-else>
            <div
              v-for="(c, i) in pdfChapters"
              :key="i"
              class="fr-toc-item"
              :style="{ paddingLeft: 12 + ((c.level || 1) - 1) * 16 + 'px' }"
              @click="pdfGoToChapter(i)"
            >
              {{ c.title || ('章节 ' + (i + 1)) }}
            </div>
            <div v-if="!pdfChapters.length" class="fr-empty">暂无目录</div>
          </template>
        </div>
      </aside>
    </transition>

    <!-- 设置抽屉 -->
    <transition name="fr-slide">
      <aside v-if="showSettings" class="fr-drawer fr-drawer-right">
        <div class="fr-drawer-head">
          <span>阅读设置</span>
          <el-icon class="fr-close" @click="showSettings = false"><Close /></el-icon>
        </div>
        <div class="fr-settings">
          <template v-if="!isPdf">
            <div class="fr-setting-row">
              <span>字号</span>
              <div class="fr-stepper">
                <el-button circle size="small" @click="changeFont(-1)">−</el-button>
                <span class="fr-value">{{ fontSize }}px</span>
                <el-button circle size="small" @click="changeFont(1)">＋</el-button>
              </div>
            </div>
            <div class="fr-setting-row">
              <span>行距</span>
              <el-slider v-model="lineHeight" :min="1.2" :max="2.4" :step="0.1" style="width: 160px" />
            </div>
            <div class="fr-setting-row">
              <span>字体</span>
              <el-select v-model="fontFamily" style="width: 160px">
                <el-option
                  v-for="f in fontOptions"
                  :key="f.value"
                  :label="f.label"
                  :value="f.value"
                />
              </el-select>
            </div>
            <div class="fr-setting-row">
              <span>主题</span>
              <div class="fr-themes">
                <button
                  v-for="t in themeKeys"
                  :key="t"
                  class="fr-theme-dot"
                  :class="{ active: theme === t }"
                  :style="{ background: THEMES[t].bg, color: THEMES[t].fg }"
                  @click="theme = t"
                >A</button>
              </div>
            </div>
            <!-- 翻页方式已固定为连续滚动（多 iframe 堆叠），不再提供分页选项 -->
            <div class="fr-setting-row">
              <span>两端对齐</span>
              <el-switch v-model="justify" />
            </div>
          </template>
          <template v-else>
            <div class="fr-setting-row">
              <span>缩放</span>
              <el-slider :min="50" :max="300" :step="5" :model-value="pdfZoomPct" @input="pdfSetScale" @change="pdfSetScale" style="width: 160px" />
            </div>
            <div class="fr-setting-row">
              <span>版式</span>
              <el-radio-group :model-value="pdfLayout" @change="pdfSetLayout">
                <el-radio-button value="single">单页</el-radio-button>
                <el-radio-button value="two">双页</el-radio-button>
              </el-radio-group>
            </div>
          </template>
        </div>
      </aside>
    </transition>

    <!-- 阅读区 -->
    <div class="fr-body" ref="frBodyEl">
      <PdfReader v-if="isPdf" :embedded="true" ref="pdfRef" :id="props.id" :book="bookInfo"
        @pdf-select="onPdfSelect" @pdf-clear="onPdfClear" />
      <!-- 隐藏的 foliate-view 仅用于解析书籍（拿到 sections 数据） -->
      <foliate-view
        v-show="false"
        ref="hiddenFoliate"
      />
      <!-- 连续滚动阅读器（自建多 iframe 堆叠，章节连续滚动，无缝翻页） -->
      <ScrollContinuousReader
        v-if="continuousReady && !isPdf"
        ref="continuousReaderRef"
        class="continuous-host"
        :sections="continuousSections"
        :font-css="continuousFontCss"
        :select-mode="selectMode"
        :annotations="annotations"
        :initial-index="continuousInitial.index"
        :initial-fraction="continuousInitial.fraction"
        @progress="onContinuousProgress"
        @ready="onContinuousReady"
        @selection="onContinuousSelection"
        @selection-clear="onContinuousSelectionClear"
        @annotation-click="onContinuousAnnClick"
      />
      <div v-if="loading && !isPdf" class="fr-loading">
        <el-icon class="is-loading"><Loading /></el-icon>
        <span>正在加载书籍…</span>
      </div>
      <div v-if="errorMsg && !isPdf" class="fr-error">
        <p>{{ errorMsg }}</p>
        <el-button @click="retry">重试</el-button>
      </div>

      <!-- 翻页热区：滚动模式下作为"上一节/下一节"快捷翻页，分页模式下作为翻页按钮 -->
      <div v-show="!isPdf" class="fr-page-zone left" :class="{ disabled: selToolbar.visible }" @click="prevPage" />
      <div v-show="!isPdf" class="fr-page-zone right" :class="{ disabled: selToolbar.visible }" @click="nextPage" />
    </div>

    <!-- 底部进度条 -->
    <footer class="fr-footer" :class="{ hidden: fullscreen || isPdf }">
      <span class="fr-percent">{{ progress }}%</span>
      <div class="fr-progress" @click="onProgressClick">
        <div class="fr-progress-fill" :style="{ width: progress + '%' }" />
        <!-- 划线/笔记在进度条上的位置标记点 -->
        <span
          v-for="m in annotationMarkers"
          :key="m.id"
          class="fr-ann-marker"
          :class="{ note: m.type === 'note' }"
          :style="{ left: m.pct + '%' }"
        ></span>
      </div>
      <span class="fr-page-info">{{ pageLabel }}</span>
    </footer>

    <!-- 选区浮动工具条（连续滚动模式下由 iframe 内 mouseup 触发） -->
    <div
      v-if="selToolbar.visible"
      class="fr-sel-toolbar"
      :style="{ left: selToolbar.x + 'px', top: selToolbar.y + 'px' }"
      @mousedown.prevent.stop
    >
      <div class="fr-sel-colors">
        <button
          v-for="c in highlightColors" :key="c"
          class="fr-sel-color" :style="{ background: c }"
          :class="{ active: currentColor === c }"
          @click="currentColor = c"
        />
      </div>
      <div class="fr-sel-divider" />
      <div class="fr-sel-styles">
        <button class="fr-sel-style" :class="{ active: currentStyle === 'highlight' }" @click="addHighlight('highlight')">
          <span class="fr-sel-ico fr-sel-ico-hl" :style="{ background: currentColor }"></span>高亮
        </button>
        <button class="fr-sel-style" :class="{ active: currentStyle === 'underline' }" @click="addHighlight('underline')">
          <span class="fr-sel-ico"><i class="fr-sel-ico-ul" :style="{ background: currentColor }"></i></span>下划线
        </button>
        <button class="fr-sel-style" :class="{ active: currentStyle === 'wavy' }" @click="addHighlight('wavy')">
          <span class="fr-sel-ico"><svg width="22" height="8" viewBox="0 0 22 8"><path d="M0,4 Q5.5,0 11,4 T22,4" fill="none" :stroke="currentColor" stroke-width="1.8"/></svg></span>波浪
        </button>
        <button class="fr-sel-style" :class="{ active: currentStyle === 'strikethrough' }" @click="addHighlight('strikethrough')">
          <span class="fr-sel-ico"><i class="fr-sel-ico-st" :style="{ background: currentColor }"></i></span>删除线
        </button>
      </div>
      <div class="fr-sel-divider" />
      <div class="fr-sel-actions">
        <button class="fr-sel-action" @click="openNote">笔记</button>
        <button class="fr-sel-action" @click="copySelection">复制</button>
        <button class="fr-sel-action" @click="speakSelection">朗读</button>
      </div>
    </div>

    <!-- 笔记输入 -->
    <div
      v-if="noteMode"
      class="fr-note-pop"
      :style="{ left: selToolbar.x + 'px', top: selToolbar.y + 'px' }"
      @mousedown.prevent.stop
    >
      <div class="fr-note-head">添加笔记</div>
      <div class="fr-note-quote">{{ selState?.text }}</div>
      <el-input v-model="noteText" type="textarea" :rows="3" resize="none" maxlength="1000" show-word-limit placeholder="输入笔记…" @keydown.enter.exact.prevent="saveNote" />
      <div class="fr-note-actions">
        <el-button size="small" @click="noteMode = false; selToolbar.visible = false">取消</el-button>
        <el-button size="small" type="primary" @click="saveNote">保存</el-button>
      </div>
    </div>

    <!-- 批注与笔记抽屉 -->
    <transition name="fr-slide">
      <aside v-if="showAnnDrawer" class="fr-drawer fr-drawer-right">
        <div class="fr-drawer-head">
          <span>批注与笔记 ({{ annCount }})</span>
          <el-icon class="fr-close" @click="showAnnDrawer = false"><Close /></el-icon>
        </div>
        <div class="fr-ann-list">
          <div v-if="!annCount" class="fr-empty">还没有批注，选中正文即可划线或记笔记</div>
          <div
            v-for="ann in (isPdf ? pdfAnnotations : annotations)" :key="ann.id"
            class="fr-ann-item" :class="{ active: activeAnnId === ann.id }"
            @click="onAnnClick(ann)"
          >
            <span class="fr-ann-dot" :style="{ background: ann.color || '#ffeb3b' }"></span>
            <div class="fr-ann-body">
              <div class="fr-ann-text">{{ ann.content || ann.text }}</div>
              <div v-if="ann.note" class="fr-ann-note">{{ ann.note }}</div>
            </div>
            <el-button text type="danger" size="small" title="删除" @click.stop="onAnnDelete(ann)">
              <el-icon><Delete /></el-icon>
            </el-button>
          </div>
        </div>
      </aside>
    </transition>

    <!-- 语音朗读面板（划线朗读：从选中位置开始，Edge TTS 在线合成） -->
    <transition name="fr-slide">
      <aside v-if="showTtsDrawer" class="fr-drawer fr-drawer-right fr-tts-drawer">
        <div class="fr-drawer-head">
          <span>语音朗读</span>
          <el-icon class="fr-close" @click="showTtsDrawer = false"><Close /></el-icon>
        </div>
        <div class="fr-tts-body">
          <!-- 断点续播：有保存的朗读断点时，从上次位置继续（从头栏迁入此面板） -->
          <button v-if="savedTtsLocation && !tts.active" class="fr-tts-read-btn resume" title="从上次朗读断点继续" @click="resumeTtsFromSaved">
            <el-icon :size="15"><CaretRight /></el-icon>
            <span>继续阅读</span>
          </button>
          <!-- 从划线/选中位置开始朗读 -->
          <button class="fr-tts-read-btn" :class="{ 'is-pending': ttsPendingSelection }" @click="startTtsFromPanel">
            <el-icon :size="15"><Aim v-if="ttsPendingSelection" /><CaretRight v-else /></el-icon>
            <span>{{ ttsPendingSelection ? '请在正文中选中要朗读的文字…' : '从选中位置开始阅读' }}</span>
          </button>
          <button v-if="!tts.active" class="fr-tts-read-btn secondary" @click="startTtsFromCurrent">
            <el-icon :size="15"><Microphone /></el-icon>
            <span>从当前章节开始</span>
          </button>

          <!-- 朗读中：当前句 + 播放控制 -->
          <div v-if="tts.active" class="fr-tts-playing">
            <div class="fr-tts-current" :title="tts.currentText">
              <el-icon v-if="tts.loading" class="is-loading" :size="14"><Loading /></el-icon>
              <span>{{ tts.currentText || '正在准备朗读…' }}</span>
            </div>
            <div class="fr-tts-ctrl">
              <button class="fr-tts-ctrl-btn" :disabled="tts.index <= 0" title="上一句" @click="tts.prev()">
                <el-icon><Back /></el-icon>
              </button>
              <button class="fr-tts-ctrl-btn primary" :disabled="tts.loading" :title="tts.paused ? '继续' : '暂停'" @click="tts.togglePlay()">
                <el-icon :size="16"><VideoPause v-if="!tts.paused && tts.playing" /><VideoPlay v-else /></el-icon>
              </button>
              <button class="fr-tts-ctrl-btn" :disabled="tts.index >= tts.sentences.length - 1" title="下一句" @click="tts.next()">
                <el-icon><Right /></el-icon>
              </button>
              <button class="fr-tts-ctrl-btn danger" title="停止" @click="stopTtsReading">
                <el-icon><Close /></el-icon>
              </button>
            </div>
            <div class="fr-tts-progress">{{ ttsProgressLabel }}</div>
          </div>

          <!-- 朗读引擎：云端（需联网）或浏览器离线（不依赖微软端点，受限网络可用） -->
          <div class="fr-tts-opt">
            <span class="fr-tts-label">引擎</span>
            <el-radio-group v-model="tts.engine" size="small" style="flex: 1" @change="tts.setEngine">
              <el-radio-button value="edge-tts">云端</el-radio-button>
              <el-radio-button value="browser" :disabled="!tts.browserSupported">浏览器离线</el-radio-button>
            </el-radio-group>
          </div>

          <!-- 音色：浏览器引擎用本机音色，云端用 Edge 在线音色 -->
          <div class="fr-tts-opt" v-if="tts.engine === 'browser'">
            <span class="fr-tts-label">音色</span>
            <el-select v-model="tts.browserVoiceURI" filterable size="small" style="flex: 1" @change="tts.onBrowserVoiceChange" :placeholder="tts.browserVoices.length ? '选择本机语音' : '无可用语音'">
              <el-option-group v-for="grp in ttsBrowserVoiceGroups" :key="grp.lang" :label="grp.lang">
                <el-option v-for="v in grp.voices" :key="v.voiceURI" :label="v.name || v.voiceURI" :value="v.voiceURI" />
              </el-option-group>
            </el-select>
          </div>
          <div class="fr-tts-opt" v-else>
            <span class="fr-tts-label">音色</span>
            <el-select v-model="tts.voice" filterable size="small" placeholder="请选择音色" style="flex: 1" @change="tts.onVoiceChange">
              <el-option-group v-if="ttsFemaleVoices.length" label="女声">
                <el-option v-for="v in ttsFemaleVoices" :key="v.short_name" :label="v.friendly_name" :value="v.short_name" />
              </el-option-group>
              <el-option-group v-if="ttsMaleVoices.length" label="男声">
                <el-option v-for="v in ttsMaleVoices" :key="v.short_name" :label="v.friendly_name" :value="v.short_name" />
              </el-option-group>
              <el-option-group v-if="ttsOtherVoices.length" label="其他">
                <el-option v-for="v in ttsOtherVoices" :key="v.short_name" :label="v.friendly_name" :value="v.short_name" />
              </el-option-group>
            </el-select>
          </div>

          <div class="fr-tts-opt" v-if="tts.engine !== 'browser'">
            <span class="fr-tts-label">语气</span>
            <el-select v-model="tts.emotion" size="small" style="flex: 1" @change="tts.restartIfPlaying()">
              <el-option v-for="em in tts.emotions" :key="em" :label="em" :value="em" />
            </el-select>
          </div>
          <div class="fr-tts-opt">
            <span class="fr-tts-label">倍速</span>
            <el-slider v-model="tts.speed" :min="0.5" :max="2" :step="0.1" style="flex: 1" @change="tts.restartIfPlaying()" />
            <span class="fr-tts-speed-val">{{ tts.speed.toFixed(1) }}x</span>
          </div>
          <p class="fr-tts-tip">
            <template v-if="tts.engine === 'browser'">使用浏览器内置语音（离线、无需联网，不依赖微软端点）；本机系统需安装中文语音包（如 Windows 语音、macOS 朗读语音）。</template>
            <template v-else>朗读由 Edge TTS 在线合成，需联网；断点自动保存，下次打开本面板点「继续阅读」即可续播。若提示 403 / 连接失败（受限网络），可在启动后端的环境中设置 EDGE_TTS_PROXY 或 HTTPS_PROXY 经代理出网。</template>
          </p>
        </div>
      </aside>
    </transition>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowLeft, ArrowRight, List, Search, Setting, FullScreen, Close, Loading, Notebook, Delete, ZoomIn, ZoomOut, Microphone, CaretRight, Aim, Back, Right, VideoPlay, VideoPause } from '@element-plus/icons-vue'
import 'foliate-js/view.js' // 注册 <foliate-view> 自定义元素（用于隐藏实例解析书籍元数据）

import PdfReader from './PdfReader.vue'
import ScrollContinuousReader from './ScrollContinuousReader.vue'

import { bookApi } from '@/api/book'
import { notifyProgressUpdate } from '@/utils/progressSync'
import { getToken } from '@/api/auth'
import { staticBase, API_BASE } from '@/config'
import { loadReaderPrefs, saveReaderPrefs } from '@/utils/readerPrefs'
import { useIsMobile } from '@/utils/useDevice'
import { useTtsPlayer, splitSentences, normTtsText } from '@/utils/ttsPlayer'

const props = defineProps({
  id: { type: [String, Number], required: true },
  book: { type: Object, default: null }
})

const router = useRouter()
const rootEl = ref(null)
const hiddenFoliate = ref(null) // 隐藏的 foliate-view，仅用于解析书籍（拿到 sections/toc）
const continuousReaderRef = ref(null) // ScrollContinuousReader 实例引用

const loading = ref(true)
const errorMsg = ref('')
const bookInfo = ref(props.book)
const flatToc = ref([])
const showToc = ref(false)
const showSettings = ref(false)
const fullscreen = ref(false)
const progress = ref(0)
const pageLabel = ref('')
const currentChapterTitle = ref('')     // 当前章节标题（从 TOC 解析）
const currentChapterIndex = ref(0)      // 当前章节 idx
const searchOpen = ref(false)
const searchQuery = ref('')
const searchResults = ref([])
const searching = ref(false)

// 连续滚动阅读器状态
const continuousReady = ref(false)
const continuousSections = ref([])
const continuousInitial = ref({ index: 0, fraction: 0 })
// 注入到每个 ScrollContinuousReader iframe 的 CSS（字体/主题/行距）
const continuousFontCss = computed(() => buildCSS())
const searchDone = ref(false)

// ============ 划线 / 笔记（基于 foliate-js 标注系统） ============
const annotations = ref([])          // 已保存批注（含后端返回记录）
const showAnnDrawer = ref(false)     // 批注抽屉
const activeAnnId = ref(null)        // 当前高亮的批注
const highlightColors = ['#ffeb3b', '#a8e6cf', '#74add1', '#d8b4fe', '#fda4af']
const currentColor = ref('#ffeb3b')  // 当前划线颜色
const currentStyle = ref('highlight')// 当前划线样式：highlight/underline/wavy/strikethrough
const selToolbar = ref({ visible: false, x: 0, y: 0 }) // 选区浮动工具条
const selState = ref(null)           // 当前选区：{ text, cfi, index }
const noteMode = ref(false)          // 笔记输入态
const noteText = ref('')
const frBodyEl = ref(null)

// ============ 划线工具（持续划线模式，参考 1.0.0 阅读界面）============
// currentTool: null | 'highlight' | 'underline' | 'wavy' | 'strikethrough' | 'eraser'
// 选中划线工具后进入持续模式：桌面端选中文字即自动划线（可连续划多处）；
// 移动端弹划选操作条由用户确认后应用；橡皮擦模式下点击已有划线直接删除。
const currentTool = ref(null)
const showColorPicker = ref(false)
const drawToolActive = computed(() => !!currentTool.value && currentTool.value !== 'eraser')
// 移动端选择模式：划线/笔记工具开启时进入（屏蔽系统长按菜单，提示条可一键取消）
const selectMode = computed(() => isMobile.value && (drawToolActive.value || noteMode.value))

// 划线模式的笔形光标（注入章节 iframe）
const DRAW_CURSOR = 'url("data:image/svg+xml,%3Csvg xmlns=%27http://www.w3.org/2000/svg%27 width=%2724%27 height=%2724%27 viewBox=%270 0 24 24%27%3E%3Cpath d=%27M3 17.25V21h3.75L17.8 9.94l-3.75-3.75L3 17.25zM20.71 7.04a1 1 0 0 0 0-1.41l-2.34-2.34a1 1 0 0 0-1.41 0l-1.83 1.83 3.75 3.75 1.83-1.83z%27 fill=%27white%27 stroke=%27black%27 stroke-width=%271.2%27/%3E%3C/svg%3E") 3 21, crosshair'

function applyDrawCursor() {
  continuousReaderRef.value?.setCursor?.(drawToolActive.value ? DRAW_CURSOR : '')
}

function setTool(tool) {
  // 已有选区时点击划线工具 → 立即对当前选区应用（一次性），随后退出持续模式
  if (tool !== 'eraser' && selState.value?.text && currentTool.value !== tool) {
    currentTool.value = tool
    noteMode.value = false
    showColorPicker.value = false
    applyDrawCursor()
    const applied = tool
    addHighlight(tool).finally(() => {
      // 仅当仍是本次应用的工具时才退出，避免覆盖用户后续操作
      if (currentTool.value === applied) {
        currentTool.value = null
        applyDrawCursor()
      }
    })
    return
  }
  // 再次点击同一工具 → 退出持续模式
  currentTool.value = currentTool.value === tool ? null : tool
  noteMode.value = false
  showColorPicker.value = false
  if (tool !== 'eraser') finishSelection()
  applyDrawCursor()
}

// 退出选择模式并恢复滑动翻页（移动端提示条「取消」入口）
function exitSelectMode() {
  currentTool.value = null
  noteMode.value = false
  finishSelection()
  applyDrawCursor()
}

// 进度条上的划线/笔记位置标记点（按章节位置换算百分比）
const annotationMarkers = computed(() => {
  if (isPdf.value) return []
  const n = continuousSections.value.length
  if (!n || !annotations.value.length) return []
  const out = []
  for (const a of annotations.value) {
    if (a.type !== 'highlight' && a.type !== 'note') continue
    let idx = null
    try {
      const pos = JSON.parse(a.position || '{}')
      if (pos.kind === 'cr' && typeof pos.index === 'number') idx = pos.index
    } catch { /* 忽略损坏的 position */ }
    if (idx == null) continue
    out.push({ id: a.id, type: a.type, pct: Math.max(0, Math.min(100, ((idx + 0.5) / n) * 100)) })
  }
  return out
})

// 选区工具条水平位置钳制：工具条约半宽 260px，限制在视口内（两侧各留 8px）
function clampToolbarX(x) {
  const vw = window.innerWidth || 768
  const half = Math.min(260, vw / 2 - 8)
  return Math.min(Math.max(x, half), Math.max(half, vw - half))
}

// ============ PDF 内嵌（复用统一外壳） ============
const isPdf = computed(() => (bookInfo.value?.file_type || '').toLowerCase() === 'pdf')
const pdfRef = ref(null)
const pdfChapters = computed(() => pdfRef.value?.chapters || [])
const pdfAnnotations = computed(() => pdfRef.value?.annotations || [])
const pdfZoomPct = computed(() => Math.round((pdfRef.value?.pdfScale || 1) * 100))
const pdfLayout = computed(() => pdfRef.value?.pdfLayout || 'single')
const pdfCurrentPage = computed(() => pdfRef.value?.currentPdfPage || 0)
const pdfTotalPages = computed(() => pdfRef.value?.totalPdfPages || 0)
const annCount = computed(() => isPdf.value ? pdfAnnotations.value.length : annotations.value.length)

function pdfZoomIn() { pdfRef.value?.zoomIn() }
function pdfZoomOut() { pdfRef.value?.zoomOut() }
function pdfGoToChapter(i) { pdfRef.value?.goToChapter(i); showToc.value = false }
function pdfSetScale(pct) { pdfRef.value?.onPdfScaleChange(pct) }
function pdfSetLayout(v) { pdfRef.value?.onPdfLayoutChange(v) }
function onAnnClick(ann) {
  if (isPdf.value) pdfRef.value?.jumpToAnnotation(ann)
  else jumpToAnnotation(ann)
}

// PDF 文本选中后由 PdfReader 抛出的事件：在外壳弹出"选中即弹出"工具条
function onPdfSelect(p) {
  selState.value = { text: p.text, isPdf: true, page: p.page }
  selToolbar.value = { visible: true, x: p.x, y: p.y }
}
function onPdfClear() {
  if (noteMode.value) return
  selToolbar.value = { visible: false, x: 0, y: 0 }
}
function onAnnDelete(ann) {
  if (isPdf.value) pdfRef.value?.deleteAnnotation(ann.id)
  else deleteAnnotation(ann)
}

// ============ 主题 / 字体定义（与「设置 → 阅读偏好」保持完全一致）============
// paperBg：章节纸张色（比 bg 略暖/略亮，与页面 bg 形成"纸张放在桌面上"的层次感）
const THEMES = {
  light: { bg: '#f4f4f5', fg: '#1f2937', link: '#2563eb', paperBg: '#fdfaf3' },
  sepia: { bg: '#e8dcc0', fg: '#5b4636', link: '#a0522d', paperBg: '#f5ecd9' },
  dark: { bg: '#1a1a1a', fg: '#d4d4d4', link: '#7fb3ff', paperBg: '#232323' },
  // 纯黑：键名与设置页统一为 pitch（旧版为 black，下方初始化时迁移）
  pitch: { bg: '#000000', fg: '#a8a8a8', link: '#6fa8ff', paperBg: '#0e0e0e' }
}
const themeKeys = Object.keys(THEMES)

// 字体选项：与设置页 settings/ReadingPrefs.vue 的 fontOptions 完全一致（值为 CSS font-family）
const fontOptions = [
  { label: '系统默认', value: 'system-ui, -apple-system, "PingFang SC", "Microsoft YaHei", sans-serif' },
  { label: '宋体', value: '"Songti SC", "SimSun", serif' },
  { label: '黑体', value: '"Heiti SC", "SimHei", sans-serif' },
  { label: '楷体', value: '"Kaiti SC", "KaiTi", serif' },
  { label: '仿宋', value: '"FangSong", "FangSong SC", serif' },
  { label: '等线', value: '"DengXian", "Microsoft YaHei", sans-serif' }
]
const DEFAULT_FONT = fontOptions[0].value
// 旧版 foliFont 键名 → 新版 CSS font-family（一次性迁移用）
const LEGACY_FONT_MAP = {
  system: DEFAULT_FONT,
  song: '"Songti SC", "SimSun", serif',
  hei: '"Heiti SC", "SimHei", sans-serif',
  kai: '"Kaiti SC", "KaiTi", serif',
  serif: 'Georgia, "Times New Roman", "Noto Serif SC", serif'
}

// ============ 偏好（与设置页共用同一份 readerPrefs）============
// 主题/字体/字号/行距统一使用共享键 theme / fontFamily / fontSize / lineHeight，
// 保证在设置页改动后阅读器立即生效，在阅读器改动后设置页也能同步显示。
const stored = reactive(loadReaderPrefs())
const fontSize = ref(stored.fontSize || 18)
const lineHeight = ref(stored.lineHeight || 1.7)
// 主题：旧数据 foliTheme / 'black' 迁移到共享键 theme / 'pitch'
let initTheme = stored.theme || stored.foliTheme || 'light'
if (initTheme === 'black') initTheme = 'pitch'
if (!THEMES[initTheme]) initTheme = 'light'
const theme = ref(initTheme)
// 字体：优先共享键 fontFamily，回退旧键 foliFont
const fontFamily = ref(stored.fontFamily || LEGACY_FONT_MAP[stored.foliFont] || DEFAULT_FONT)
const isMobile = useIsMobile()
// 一次性迁移：旧版默认/缓存的 foliFlow='paginated' 会导致滚动无法翻章（只能左右翻页），
// 统一迁移为 scrolled；用户后续手动切换的偏好会被尊重。
if (!stored._foliFlowMigrated) {
  if (stored.foliFlow === 'paginated') stored.foliFlow = 'scrolled'
  stored._foliFlowMigrated = true
  saveReaderPrefs(stored)
}
// 全设备统一滚动模式（滚动即翻页）；手机端强制滚动（忽略存储偏好）
const flow = ref(isMobile.value ? 'scrolled' : (stored.foliFlow || 'scrolled'))
const justify = ref(stored.foliJustify ?? true)

// 把当前主题色作为 CSS 变量挂到根节点，让 ScrollContinuousReader 的纸张/页面背景跟随主题
const readerRootStyle = computed(() => {
  const t = THEMES[theme.value] || THEMES.light
  return {
    '--cr-bg': t.bg,
    '--cr-paper': t.paperBg
  }
})

function persistPrefs() {
  Object.assign(stored, {
    fontSize: fontSize.value,
    lineHeight: lineHeight.value,
    // 写入共享键，让「设置 → 阅读偏好」能读到阅读器里的改动
    theme: theme.value,
    fontFamily: fontFamily.value,
    foliFlow: flow.value,
    foliJustify: justify.value
  })
  saveReaderPrefs(stored)
}

// ============ 鉴权文件 URL ============
function withToken(url) {
  const token = getToken()
  if (!token) return url
  const sep = url.includes('?') ? '&' : '?'
  return `${url}${sep}token=${encodeURIComponent(token)}`
}

// ============ 样式注入 ============
function buildCSS() {
  const t = THEMES[theme.value] || THEMES.light
  const ff = fontFamily.value || DEFAULT_FONT
  // 文字到纸张边缘的留白：移动端 24px，桌面 48~96px（随视口宽度增长）
  const sidePad = isMobile.value ? '24px' : 'max(48px, 5vw)'
  return `
    html { font-size: ${fontSize.value}px !important; }
    body {
      font-size: 1em !important;
      color: ${t.fg} !important;
      background: ${t.paperBg} !important;
      font-family: ${ff} !important;
      padding-left: ${sidePad} !important;
      padding-right: ${sidePad} !important;
      padding-top: 16px !important;
      padding-bottom: 32px !important;
      box-sizing: border-box !important;
    }
    p, li, blockquote, dd, div, span, a { font-family: ${ff} !important; }
    p, li, blockquote, dd {
      line-height: ${lineHeight.value} !important;
      text-align: ${justify.value ? 'justify' : 'start'};
      /* 段落间距收敛：很多 EPUB 用大 margin 做间距，看起来是大段空白 */
      margin-top: 0 !important;
      margin-bottom: 0.6em !important;
    }
    /* MOBI/KF8（azw3）通常用 <div> 而非 <p> 承载正文段落。
       只对"叶子 div"（不含块级子元素的纯文本段落）套用段落样式，
       避免给 <div class="shuming"> 这类布局容器也加上多余间距。 */
    div:not(:has(div, p, ul, ol, table, blockquote, h1, h2, h3, h4, h5, h6)) {
      line-height: ${lineHeight.value} !important;
      text-align: ${justify.value ? 'justify' : 'start'};
      margin-top: 0 !important;
      margin-bottom: 0.6em !important;
    }
    /* 仅含单个 <br> 的 div：MOBI 用它做段落间空行，保留一行高度 */
    div:has(> br:only-child) {
      line-height: 1 !important;
      min-height: 1em !important;
    }
    /* 空段落/div/span：很多 EPUB 用空标签做行距，收敛为一行高 */
    p:empty, div:empty, span:empty {
      margin: 0 !important;
      padding: 0 !important;
      line-height: 1 !important;
      height: 0 !important;
    }
    /* 仅含 <br> 的段落：常被用作空行，压缩到最小高度 */
    p > br:only-child { line-height: 1 !important; }
    /* 连续 <br><br> 仅保留一个（消除双倍空行） */
    br + br { display: none !important; }
    /* 标题间距收敛：避免大字号自带的大 margin 撑出空白 */
    h1, h2, h3, h4, h5, h6 {
      margin-top: 0.8em !important;
      margin-bottom: 0.4em !important;
      line-height: 1.3 !important;
    }
    a { color: ${t.link} !important; }
    /* 图片自适应容器：避免原 EPUB 大图撑破版面（移动端尤其常见）
       max-width:100% + height:auto 保证等比缩放；svg 同处理避免溢出 */
    img, svg, video, canvas {
      max-width: 100% !important;
      height: auto !important;
      display: block !important;
      margin: 8px auto !important;
      box-sizing: border-box !important;
    }
    /* svg 没有"原始尺寸"概念时（如 width="100%"），上面规则仍生效 */
    picture { display: block; max-width: 100%; }
  `
}

// ============ 加载与打开 ============
async function loadBookInfo() {
  if (bookInfo.value) return
  const { data } = await bookApi.getBook(props.id)
  bookInfo.value = data
}

// ============ 书籍文件预取 ============
// 打开链路：ReaderEntry 取元数据 → FoliateReader 取批注 → 下载整本书文件 → 前端解析渲染。
// 其中「下载整本书文件」是绝对大头（EPUB 常达数十 MB），却串行排在批注等小请求之后。
// 挂载时立即发起文件下载（URL 只需路由 id，无需等元数据/批注），与其余请求并行；
// openBook 复用同一 Promise，不重复请求。失败即重置，保证「重试」可重新发起。
// PDF 不预取：pdf.js 自行按 Range 流式加载，提前整文件拉取反而浪费带宽。
let fileFetchPromise = null
function prestartFileFetch() {
  if (fileFetchPromise) return fileFetchPromise
  if ((props.book?.file_type || '').toLowerCase() === 'pdf') return null
  fileFetchPromise = fetch(bookApi.fileUrl(props.id))
  // 提前失败时不产生未处理的 rejection（openBook 仍会 await 到同一拒绝并走错误分支）
  fileFetchPromise.catch(() => {})
  return fileFetchPromise
}
async function fetchBookFile() {
  if (!fileFetchPromise) {
    fileFetchPromise = fetch(bookApi.fileUrl(props.id))
    fileFetchPromise.catch(() => {})
  }
  try {
    return await fileFetchPromise
  } catch (e) {
    fileFetchPromise = null
    throw e
  }
}

async function openBook() {
  loading.value = true
  errorMsg.value = ''
  // 重置连续滚动状态
  continuousReady.value = false
  continuousSections.value = []
  continuousInitial.value = { index: 0, fraction: 0 }
  try {
    const res = await fetchBookFile()
    if (!res.ok) {
      fileFetchPromise = null // 失败后重置，允许「重试」重新发起
      throw new Error(`无法获取书籍文件 (${res.status})`)
    }
    const blob = await res.blob()
    // foliate 的格式探测会读取 file.name（如 name.endsWith('.cbz')），
    // 而 fetch 得到的 Blob 没有 name，会抛 "Cannot read properties of undefined (reading 'endsWith')"。
    // 包装成带文件名的 File 即可正常执行格式探测。
    const filename = (bookInfo.value.file_path || 'book').split(/[\\/]/).pop()
    const file = new File([blob], filename, { type: blob.type || 'application/octet-stream' })
    // 用隐藏的 foliate-view 仅做"书籍解析"：拿到 sections/toc 元数据
    const v = hiddenFoliate.value
    await v.open(file)
    const sections = v.book?.sections || []
    if (!sections.length) throw new Error('未找到可用的章节数据')
    // 目录
    flatToc.value = flattenToc(v.book?.toc || [])
    // 恢复进度：兼容旧 CFI（解析到 section index）与新 JSON {"i","f"}
    let initialIndex = 0
    let initialFraction = 0
    const saved = localStorage.getItem(`foli_loc_${props.id}`)
    if (saved) {
      try {
        if (saved.startsWith('{')) {
          const p = JSON.parse(saved)
          initialIndex = Math.max(0, Math.min(p.i || 0, sections.length - 1))
          initialFraction = Math.max(0, Math.min(1, p.f || 0))
        } else {
          // 旧 CFI：尝试解析为章节 index（找不到则用全书进度近似）
          try {
            const resolved = await v.resolveNavigation(saved)
            initialIndex = Math.max(0, Math.min(resolved.index || 0, sections.length - 1))
          } catch (e) {
            const frac = (bookInfo.value?.progress || 0) / 100
            initialIndex = Math.floor(frac * sections.length)
            initialFraction = frac * sections.length - initialIndex
          }
        }
      } catch (e) {
        console.warn('恢复进度失败:', e)
      }
    }
    // 启动连续滚动阅读器
    continuousSections.value = sections
    continuousInitial.value = { index: initialIndex, fraction: initialFraction }
    continuousReady.value = true
    loading.value = false
  } catch (e) {
    console.error(e)
    errorMsg.value = e?.message || '打开书籍失败'
    loading.value = false
  }
}

function flattenToc(items, depth = 0, out = []) {
  for (const it of items || []) {
    out.push({ label: it.label || '(无标题)', href: it.href, depth })
    if (it.subitems?.length) flattenToc(it.subitems, depth + 1, out)
  }
  return out
}

// 构建 sectionIdx → TOC label 的映射（同一章节可能对应多个 TOC 条目，取最近的）
const sectionToTocLabel = computed(() => {
  const map = new Map()
  const v = hiddenFoliate.value
  if (!v?.book?.sections || !flatToc.value.length) return map
  for (const item of flatToc.value) {
    if (!item.href) continue
    let idx = -1
    // azw3/MOBI：resolveNavigation 是异步的（返回 Promise），computed 内无法 await，
    // 因此优先用同步的 splitTOCHref(href) → [index, pos] 直接取章节序号。
    if (typeof v.book?.splitTOCHref === 'function') {
      try {
        const r = v.book.splitTOCHref(item.href)
        if (Array.isArray(r) && typeof r[0] === 'number') idx = r[0]
      } catch {}
    }
    // EPUB 等格式：同步 resolveNavigation
    if (idx < 0) {
      try {
        const resolved = v.resolveNavigation(item.href)
        if (resolved && typeof resolved.index === 'number') idx = resolved.index
      } catch {}
    }
    if (idx >= 0) {
      // 后出现的覆盖前面（取最深嵌套 TOC 的 label）
      map.set(idx, item.label)
    }
  }
  return map
})

// 当前章节标题
function updateChapterTitle(idx) {
  currentChapterIndex.value = idx
  const label = sectionToTocLabel.value.get(idx) || sectionToTocLabel.value.get(idx - 1)
  currentChapterTitle.value = label || `第 ${idx + 1} 节`
}



// ============ 连续滚动阅读器：状态与回调 ============
// 连续滚动阅读器的核心机制（80% 预加载、进度保存、章节导航）全部由
// ScrollContinuousReader 内部处理，这里只暴露必要回调。

// 进度保存：连续滚动没有 CFI，按章节 index + 节内 fraction 持久化。
// debounce 缩到 800ms，确保用户中途退出（goBack / 跳详情）也能很快同步到后端
let saveTimer = null
function scheduleSave(idx, frac) {
  clearTimeout(saveTimer)
  saveTimer = setTimeout(() => doSave(idx, frac), 800)
}
async function doSave(idx, frac) {
  try {
    localStorage.setItem(`foli_loc_${props.id}`, JSON.stringify({ i: idx, f: frac }))
  } catch {}
  const secCount = continuousSections.value.length
  const total = secCount > 0
    ? Math.round((idx + frac) / secCount * 100)
    : 0
  // 朗读断点：朗读中按「朗读章节 + 句号」保存，否则传空串（后端不更新已有断点）
  const ttsLoc = tts.active
    ? JSON.stringify({ type: 'epub', chapter: ttsChapter.value, sent: Math.max(0, tts.index) })
    : ''
  // 本地已同步落盘；后端上报异步进行（goBack 不再 await，返回不卡顿）。
  // 保存成功后通知全局 store，详情页/书架实时联动。
  try {
    await bookApi.updateProgress(props.id, total, idx, ttsLoc)
    // 保存成功后通知全局 store，详情页/书架实时联动
    try { notifyProgressUpdate(props.id, { progress: total, current_page: idx }) } catch {}
  } catch {}
}
async function flushSave() {
  clearTimeout(saveTimer)
  const p = continuousReaderRef.value?.getProgress?.()
  if (p) await doSave(p.index, p.fraction)
}

function onContinuousProgress(p) {
  if (!p) return
  progress.value = Math.round(p.totalFraction * 100)
  // 连续滚动无单章内的"页"概念，清空旧分页的标签
  pageLabel.value = ''
  // 章节切换时更新标题
  if (p.index !== currentChapterIndex.value) updateChapterTitle(p.index)
  scheduleSave(p.index, p.fraction)
}


function onContinuousReady() {
  // 阅读器准备就绪：可在此触发恢复进度滚动（initialFraction 已在组件内部应用）
}

// ============ 导航（连续滚动版） ============
// 目录跳转：隐藏 foliate-view 解析 href → 解析到章节 index → 滚动到该章
// 关键：azw3/MOBI 的 book.resolveHref 是 async（返回 Promise），而 EPUB 的是同步返回。
// 必须 await 才能兼容两种格式——否则 azw3 下拿到 Promise，resolved.index 为 undefined，
// 判断不成立，目录点击看起来"没有任何反应"。
async function goToHref(href) {
  if (!href) return
  const v = hiddenFoliate.value
  if (!v) return
  try {
    const resolved = await v.resolveNavigation(href)
    if (resolved && typeof resolved.index === 'number') {
      continuousReaderRef.value?.scrollToSection?.(resolved.index)
    }
  } catch (e) { console.warn('goToHref failed', e) }
  showToc.value = false
}
function prevPage() {
  if (isPdf.value) { pdfRef.value?.prevPage(); return }
  // 滚动模式：翻一屏（更贴合"左右翻页"的语义，跳整章太突兀）
  if (continuousReaderRef.value?.pageFlip) {
    continuousReaderRef.value.pageFlip(-1)
  }
}
function nextPage() {
  if (isPdf.value) { pdfRef.value?.nextPage(); return }
  // 滚动模式：翻一屏
  if (continuousReaderRef.value?.pageFlip) {
    continuousReaderRef.value.pageFlip(1)
  }
}
// 搜索结果跳转：与 goToHref 同理，azw3/MOBI 的解析是异步的，需要 await
async function jumpToCfi(cfi) {
  if (!cfi) return
  const v = hiddenFoliate.value
  if (!v) return
  try {
    const resolved = await v.resolveNavigation(cfi)
    if (resolved && typeof resolved.index === 'number') {
      continuousReaderRef.value?.scrollToSection?.(resolved.index)
    }
  } catch (e) { console.warn('jumpToCfi failed', e) }
  searchOpen.value = false
}
function onProgressClick(ev) {
  const rect = ev.currentTarget.getBoundingClientRect()
  const frac = (ev.clientX - rect.left) / rect.width
  continuousReaderRef.value?.scrollToFraction?.(Math.min(1, Math.max(0, frac)))
}

// ============ 搜索（Phase 1 暂禁用：依赖 foliate CFI 搜索，连续滚动架构下未适配） ============
async function runSearch() {
  ElMessage.warning('搜索功能将在后续版本恢复')
}
function toggleSearch() {
  searchOpen.value = !searchOpen.value
  if (!searchOpen.value) {
    searchResults.value = []
    searchDone.value = false
  }
}

// ============ 设置变更 ============
function changeFont(delta) {
  fontSize.value = Math.min(42, Math.max(10, fontSize.value + delta))
  persistPrefs()
}
watch(lineHeight, () => { persistPrefs() })
watch(fontFamily, () => { persistPrefs() })
watch(theme, () => { persistPrefs() })
watch(justify, () => { persistPrefs() })

// ============ 全屏 / 返回 ============
function toggleFullscreen() {
  const el = rootEl.value
  if (!document.fullscreenElement) el?.requestFullscreen?.().catch(() => {})
  else document.exitFullscreen?.()
}
function goBack() {
  // 退出阅读页：进度保存放后台（localStorage 同步落盘 + 后端异步上报），不阻塞返回。
  // 阅读过程中已按 800ms 节流自动保存，这里无需 await 网络请求，避免返回时卡顿。
  flushSave()
  // 用 router.beforeEach 记录的"来时路径"，并用 replace（不新增 history 项）
  // 避免详情页 → 阅读 → 返回 → 详情 → 返回 时落进重复 reader entry
  let prev = ''
  try { prev = sessionStorage.getItem('moyue_prev_reader_path') || '' } catch {}
  // 只接受站内路径，过滤掉登录页等不期望的目标
  if (prev && prev.startsWith('/') && !prev.startsWith('/reader/') && prev !== '/login') {
    router.replace(prev)
  } else {
    router.replace('/bookshelf')
  }
}
function retry() { openBook() }

// ============ 划线 / 笔记 ============
function annCfi(a) {
  if (a.cfi) return a.cfi
  try { return JSON.parse(a.position || '{}').cfi || '' } catch { return '' }
}
function annIndex(a) {
  try { return JSON.parse(a.position || '{}').index } catch { return undefined }
}
// 滚动模式专用：识别 position.kind === 'cr' 的标注
function isCrAnn(a) {
  try { return JSON.parse(a.position || '{}').kind === 'cr' } catch { return false }
}
// 连续滚动模式下，ScrollContinuousReader 把 iframe 内选区传上来
function onContinuousSelection(p) {
  if (!p || !p.text) return
  // 选区发生的此刻，该区块 iframe 必定已就绪——立即抓取全文快照，
  // 朗读时优先用它切句，规避"取句时区块被重载/未就绪→回退服务端错误章节"导致的高亮失效
  const sectionText = continuousReaderRef.value?.getSectionText?.(p.blockIdx) || ''
  selState.value = { text: p.text, blockIdx: p.blockIdx, range: p.range, sectionText }
  // 划线语音朗读待选模式：选中文字即从该处开始朗读
  if (ttsPendingSelection.value) {
    ttsPendingSelection.value = false
    startTtsFromSelection()
    return
  }
  // 橡皮擦模式：不弹工具条（点击已有划线即删除，见 onContinuousAnnClick）
  if (currentTool.value === 'eraser') return
  // 持续划线模式（桌面端）：选中文字即自动应用当前工具，可连续划多处
  if (drawToolActive.value && !isMobile.value) {
    selToolbar.value = { visible: false, x: 0, y: 0 }
    addHighlight(currentTool.value)
    return
  }
  // 常规 / 移动端确认模式：弹出选区工具条（位置做视口钳制，避免溢出屏幕）
  selToolbar.value = {
    visible: true,
    x: clampToolbarX(p.x),
    y: Math.max(64, p.y)
  }
}
function onContinuousSelectionClear() {
  if (noteMode.value) return
  selToolbar.value = { visible: false, x: 0, y: 0 }
}
function onContinuousAnnClick(p) {
  const ann = annotations.value.find(a => a.id === p.id)
  if (!ann) return
  // 橡皮擦模式：点击划线直接删除
  if (currentTool.value === 'eraser') {
    deleteAnnotation(ann)
    return
  }
  // 常规：打开批注抽屉并高亮当前项
  activeAnnId.value = ann.id
  showAnnDrawer.value = true
}

async function loadAnnotations() {
  try {
    const { data } = await bookApi.listAnnotations(props.id)
    annotations.value = data || []
  } catch (e) { console.warn('加载批注失败', e) }
}

// ============ 阅读会话统计 ============
// 连续滚动阅读器此前不上报阅读时长，导致 EPUB/AZW3/MOBI 的阅读不计入阅读统计。
// 规则：每 5s 累计一次（仅页面可见时，后台时间不计），离开时上报，不足 10 秒忽略。
// EPUB 与 PDF 统一由本外壳计时上报（PDF 的阅读/朗读都从这里记录）；
// 其中“朗读”时间单独累计，离开时拆分为“阅读 + 朗读”两条记录（时长合计不变）。
let sessionStartAt = Date.now()
let activeSeconds = 0
let ttsSeconds = 0 // 本次会话中处于朗读（tts 播放中、未暂停）的累计秒数
let sessionTimer = null
let sessionReported = false

// 朗读记录的定位：EPUB 记章节下标，PDF 记页码（存入 start_page 供前端展示）
function ttsLocationPage() {
  if (isPdf.value) return pdfRef.value?.currentPdfPage || 0
  return ttsChapter.value || 0
}

function startReadingSession() {
  sessionStartAt = Date.now()
  activeSeconds = 0
  ttsSeconds = 0
  sessionReported = false
  sessionTimer = setInterval(() => {
    if (!document.hidden) {
      activeSeconds += 5
      if (tts.active && !tts.paused) ttsSeconds += 5
    }
  }, 5000)
}

// 常规离开（SPA 路由切换 / 组件卸载）：直接发请求。
// 有实质朗读（≥10s）时拆分上报，否则按原样记录为一次阅读（保持既有行为）。
async function reportReadingSession() {
  if (sessionTimer) {
    clearInterval(sessionTimer)
    sessionTimer = null
  }
  if (sessionReported) return
  sessionReported = true
  const readPart = Math.max(0, activeSeconds - ttsSeconds)
  const ttsPart = ttsSeconds
  if (readPart < 10 && ttsPart < 10) return // 不足 10 秒忽略
  const start = new Date(sessionStartAt).toISOString()
  const end = new Date().toISOString()
  try {
    if (ttsPart < 10) {
      if (readPart >= 10) {
        await bookApi.reportReading(Number(props.id), readPart, start, end, { type: 'read' })
      }
    } else {
      if (readPart >= 10) {
        await bookApi.reportReading(Number(props.id), readPart, start, end, { type: 'read' })
      }
      await bookApi.reportReading(Number(props.id), ttsPart, start, end, {
        type: 'tts',
        engine: tts.engine || '',
        start_page: ttsLocationPage()
      })
    }
  } catch (e) {
    // 静默失败，不影响阅读
  }
}

// 页面关闭/刷新/移动端切后台：异步请求会被浏览器取消，
// 用 sendBeacon（后端鉴权兼容 ?token=）保活投递；随后快照重置——
// 页面若从 bfcache 恢复继续阅读，已上报部分不重复计入，剩余时长作为新会话在最终离开时上报。
function beaconReadingSession() {
  if (sessionReported) return
  const readPart = Math.max(0, activeSeconds - ttsSeconds)
  const ttsPart = ttsSeconds
  if (readPart < 10 && ttsPart < 10) return
  sessionReported = true
  try {
    const token = getToken()
    if (!token) return
    const url = `${API_BASE}/reading-history?token=${encodeURIComponent(token)}`
    const start = new Date(sessionStartAt).toISOString()
    const end = new Date().toISOString()
    if (ttsPart < 10) {
      if (readPart >= 10) {
        navigator.sendBeacon(url, JSON.stringify({ book_id: Number(props.id), type: 'read', duration: readPart, start_time: start, end_time: end }))
      }
    } else {
      if (readPart >= 10) {
        navigator.sendBeacon(url, JSON.stringify({ book_id: Number(props.id), type: 'read', duration: readPart, start_time: start, end_time: end }))
      }
      navigator.sendBeacon(url, JSON.stringify({
        book_id: Number(props.id), type: 'tts', engine: tts.engine || '',
        duration: ttsPart, start_time: start, end_time: end, start_page: ttsLocationPage()
      }))
    }
  } catch (e) {
    // 静默失败
  }
  activeSeconds = 0
  ttsSeconds = 0
  sessionStartAt = Date.now()
  sessionReported = false
}

// ============ 语音朗读（划线语音朗读，Edge TTS）============
// 交互对齐 1.0.0 阅读界面：选中正文 → 「从选中位置开始阅读」→ 从该句起逐句朗读、
// 段落级高亮跟随、一章读完自动切下一章；断点写入 book.tts_location，支持续播。
// 合成/播放引擎收敛在 utils/ttsPlayer.js，这里只负责文本来源、高亮与章节编排。
const showTtsDrawer = ref(false)
const ttsChapter = ref(0)              // 当前朗读章节 idx（foliate sections 下标）
const ttsChapterCache = new Map()      // 章节 idx -> 句子列表
const savedTtsLocation = ref(null)     // 保存的朗读断点 {type:'epub',chapter,sent}
const ttsPendingSelection = ref(false) // 待选模式：等待用户选中文字后自动开始朗读

// reactive 包裹使模板中 tts.active / tts.voice 等直接可用（ref 自动解包）
const tts = reactive(useTtsPlayer({
  // 每句播放前：高亮当前句并滚动到视口中部
  onSentence: (i, sentence) => {
    if (isPdf.value) {
      // PDF：高亮当前朗读页中被朗读句子覆盖的文本层 span（i 用于记录朗读断点）
      pdfRef.value?.ttsOnSentence?.(i, sentence)
    } else {
      continuousReaderRef.value?.highlightSentence(ttsChapter.value, sentence)
      continuousReaderRef.value?.scrollHighlightedIntoView?.()
      // 逐句落朗读断点（进度按朗读章节计，见 doSave）
      scheduleSave(ttsChapter.value, 0)
    }
  },
  // 一章/一页读完：加载下一段继续
  onEnd: () => {
    if (isPdf.value) advanceTtsPdfPage()
    else advanceTtsChapter()
  }
}))

// PDF 朗读完一页后翻到下一页继续（已到末页则结束）
async function advanceTtsPdfPage() {
  const sents = await pdfRef.value?.ttsNextPage?.()
  if (sents && sents.length) {
    tts.setSentences(sents)
  } else {
    ElMessage.success('全书朗读完毕')
    stopTtsReading()
  }
}

const ttsFemaleVoices = computed(() => tts.voices.filter(v => v.gender === 'Female'))
const ttsMaleVoices = computed(() => tts.voices.filter(v => v.gender === 'Male'))
const ttsOtherVoices = computed(() => tts.voices.filter(v => v.gender !== 'Female' && v.gender !== 'Male'))
// 浏览器本机音色按语言分组（Web Speech API 的 voice 不含 gender，无法按男女分组）
const ttsBrowserVoiceGroups = computed(() => {
  const groups = {}
  for (const v of tts.browserVoices) {
    const lang = v.lang || '其他'
    ;(groups[lang] = groups[lang] || []).push(v)
  }
  return Object.keys(groups).map(lang => ({ lang, voices: groups[lang] }))
})
// 朗读面板进度文案：PDF 显示页码，EPUB 显示章号
const ttsProgressLabel = computed(() => {
  const idx = Math.max(0, tts.index + 1)
  const total = tts.sentences.length
  const where = isPdf.value ? `第 ${pdfRef.value?.currentPdfPage || ''} 页` : `第 ${ttsChapter.value + 1} 章`
  return `第 ${idx} / ${total} 句 · ${where}`
})

// 打开面板时收起其他抽屉，并预热音色/语气列表
watch(showTtsDrawer, (v) => {
  if (!v) return
  showToc.value = false
  showSettings.value = false
  showAnnDrawer.value = false
  tts.loadSettings()
})

// 解析书籍保存的朗读断点（打开书时）
function parseSavedTtsLocation() {
  try {
    const raw = bookInfo.value?.tts_location
    const loc = raw ? JSON.parse(raw) : null
    const ok = loc && (
      loc.type === 'epub' ? typeof loc.chapter === 'number'
      : loc.type === 'pdf' ? typeof loc.page === 'number'
      : false
    )
    savedTtsLocation.value = ok ? loc : null
  } catch (e) {
    savedTtsLocation.value = null
  }
}

// 获取章节句子列表：句子必须取自 iframe 已渲染文本（与朗读高亮的 DOM 文本同源），
// 否则服务端按"书籍章节索引"取文，与连续滚动阅读器的 section 下标不对齐，会导致
// 句子与高亮 DOM 文本不一致、高亮永远失效。这里轮询等待区块就绪（最多约 3s），
// 确保句子来自渲染文本；服务端文本仅作极端兜底（且不缓存，避免污染后续朗读）。
async function getChapterSentences(idx) {
  if (ttsChapterCache.has(idx)) return ttsChapterCache.get(idx)
  let sents = []
  let iframeText = ''
  // 优先用选区时刻捕获的区块快照（此刻 iframe 必定就绪，且与高亮 DOM 同源）
  if (idx === selState.value?.blockIdx && selState.value?.sectionText) {
    iframeText = selState.value.sectionText
  }
  if (!iframeText || !iframeText.trim()) {
    // 直接加载目标区块（绕过滚动窗口懒加载，isSectionLoaded 只看 block.loaded 会误判），
    // 确保该区块 iframe 进入 iframeRefs 且内容已解析；随后轮询其渲染文本，最长约 5s，
    // 句子必须取自渲染文本才能与高亮 DOM 匹配（此前回退服务端会因章节索引不对齐而错位）。
    try { await continuousReaderRef.value?.loadSection?.(idx) } catch (e) { /* noop */ }
    continuousReaderRef.value?.scrollToSection?.(idx, { behavior: 'auto' })
    for (let attempt = 0; attempt < 50; attempt++) {
      iframeText = continuousReaderRef.value?.getSectionText?.(idx) || ''
      if (iframeText && iframeText.trim()) break
      await new Promise(r => setTimeout(r, 100))
    }
  }
  if (iframeText && iframeText.trim()) {
    sents = splitSentences(iframeText)
    ttsChapterCache.set(idx, sents)
  } else {
    // 极端兜底：区块始终取不到渲染文本时回退服务端（用真实书籍章节索引，错误结果不缓存）
    try {
      const { data } = await bookApi.getBookChapterText(props.id, currentChapterIndex.value || idx)
      sents = splitSentences(data?.data?.text || '')
    } catch (e) {
      console.warn('获取章节文本失败：', e)
    }
  }
  return sents
}

// 等待章节 iframe 加载完成（切章后高亮需要文档就绪）
async function waitTtsChapterLoaded(idx, timeout = 4000) {
  if (continuousReaderRef.value?.isSectionLoaded?.(idx)) return true
  const started = Date.now()
  while (Date.now() - started < timeout) {
    await new Promise(r => setTimeout(r, 150))
    if (continuousReaderRef.value?.isSectionLoaded?.(idx)) return true
  }
  return false
}

// 头部朗读按钮：朗读中且面板已开 → 停止；朗读中但面板关 → 打开面板调声色/倍速（不停止）；否则开关面板
function onTtsHeaderClick() {
  if (tts.active) {
    if (showTtsDrawer.value) stopTtsReading()
    else showTtsDrawer.value = true
    return
  }
  showTtsDrawer.value = !showTtsDrawer.value
}

// 停止朗读并清理高亮
function stopTtsReading() {
  tts.stop()
  continuousReaderRef.value?.clearSentenceHighlight?.()
  pdfRef.value?.ttsStop?.()
  ttsPendingSelection.value = false
}

// 从当前章节开始朗读
async function startTtsFromCurrent() {
  // PDF：从当前视口页开始朗读（由外壳统一 TTS 驱动，复用 PdfReader 取句/高亮/翻页）
  if (isPdf.value) {
    const page = pdfRef.value?.currentPdfPage || 1
    pdfRef.value?.ttsSetPage?.(page)
    const sents = await pdfRef.value?.ttsGetPageSentences?.(page) || []
    if (!sents.length) { ElMessage.warning('当前页没有可朗读的文本'); return }
    const ok = await tts.start(sents, 0)
    if (ok) showTtsDrawer.value = true
    return
  }
  const idx = Math.max(0, currentChapterIndex.value || 0)
  ttsChapter.value = idx
  continuousReaderRef.value?.resetSentenceSearch?.()
  await waitTtsChapterLoaded(idx)
  const sents = await getChapterSentences(idx)
  if (!sents.length) {
    ElMessage.warning('当前章节没有可朗读的文本')
    return
  }
  const ok = await tts.start(sents, 0)
  if (ok) showTtsDrawer.value = true
}

// 划线语音朗读：从选中文字所在句子开始（选中内容的前 10 字定位句子）
async function startTtsFromSelection() {
  // PDF：从选中文字所在页开始朗读
  if (isPdf.value) {
    const sel = selState.value
    if (!sel || !sel.text) { ElMessage.warning('请先选中文字，再从该处开始朗读'); return }
    if (tts.active) tts.stop()
    const page = sel.page || pdfRef.value?.currentPdfPage || 1
    pdfRef.value?.ttsSetPage?.(page)
    const sents = await pdfRef.value?.ttsGetPageSentences?.(page) || []
    if (!sents.length) { ElMessage.warning('当前页没有可朗读的文本'); return }
    const key = normTtsText(sel.text).slice(0, 12)
    let startIdx = 0
    if (key) { const f = sents.findIndex(s => normTtsText(s).includes(key)); if (f > 0) startIdx = f }
    const ok = await tts.start(sents, startIdx)
    if (ok) { showTtsDrawer.value = true; finishSelection(); selState.value = null }
    return
  }
  const text = (selState.value?.text || '').trim()
  if (!text) {
    ElMessage.warning('请先选中文字，再从该处开始朗读')
    return
  }
  // 正在朗读则先停止，重新从选中位置开始
  if (tts.active) tts.stop()
  const selChapter = selState.value?.blockIdx
  const idx = typeof selChapter === 'number' ? selChapter : (currentChapterIndex.value || 0)
  ttsChapter.value = idx
  continuousReaderRef.value?.resetSentenceSearch?.()
  await waitTtsChapterLoaded(idx)
  const sents = await getChapterSentences(idx)
  if (!sents.length) {
    ElMessage.warning('当前章节没有可朗读的文本')
    return
  }
  // 找到包含选中文字前 10 字的句子并从该句开始（找不到则从本章第一句）
  const key = normTtsText(text).slice(0, 10)
  let startIdx = 0
  if (key) {
    const found = sents.findIndex(s => normTtsText(s).includes(key))
    if (found > 0) startIdx = found
  }
  const ok = await tts.start(sents, startIdx)
  if (ok) {
    showTtsDrawer.value = true // 朗读中保持面板，可随时调节声色/倍速（同 1.0.0）
    finishSelection()
    selState.value = null // 清除选区，避免面板再次进入时复用过期选区
  }
}

// 面板入口：有待选区直接开始；否则进入待选模式，选中后自动开始（onContinuousSelection 消费）
function startTtsFromPanel() {
  if (ttsPendingSelection.value) {
    ttsPendingSelection.value = false
    ElMessage.info('已取消，重新点击可再次选择朗读位置')
    return
  }
  if ((selState.value?.text || '').trim()) {
    startTtsFromSelection()
    return
  }
  ttsPendingSelection.value = true
  ElMessage.info('请在正文中选中要朗读的文字，选中后自动开始朗读')
}

// 选区工具条「朗读」按钮
function speakSelection() {
  startTtsFromSelection()
}

// 断点续播：定位到断点章节与句子继续朗读
async function resumeTtsFromSaved() {
  const loc = savedTtsLocation.value
  if (!loc) return
  // PDF：从断点页继续
  if (isPdf.value) {
    if (loc.type !== 'pdf') return
    const page = loc.page
    pdfRef.value?.ttsSetPage?.(page)
    const sents = await pdfRef.value?.ttsGetPageSentences?.(page) || []
    if (!sents.length) { ElMessage.warning('断点页没有可朗读的文本'); return }
    const sent = (typeof loc.sent === 'number' && loc.sent >= 0 && loc.sent < sents.length) ? loc.sent : 0
    const ok = await tts.start(sents, sent)
    if (ok) showTtsDrawer.value = true
    return
  }
  const total = continuousSections.value.length
  const idx = Math.max(0, Math.min(loc.chapter || 0, Math.max(0, total - 1)))
  ttsChapter.value = idx
  continuousReaderRef.value?.resetSentenceSearch?.()
  continuousReaderRef.value?.scrollToSection?.(idx, { behavior: 'auto' })
  await waitTtsChapterLoaded(idx)
  const sents = await getChapterSentences(idx)
  if (!sents.length) {
    ElMessage.warning('断点章节没有可朗读的文本')
    return
  }
  const sent = loc.sent >= 0 && loc.sent < sents.length ? loc.sent : 0
  const ok = await tts.start(sents, sent)
  if (ok) showTtsDrawer.value = true
}

// 一章朗读完毕：向后查找下一个有文本的章节继续；到末尾则结束
async function advanceTtsChapter() {
  const total = continuousSections.value.length
  let next = ttsChapter.value + 1
  while (next < total) {
    continuousReaderRef.value?.scrollToSection?.(next, { behavior: 'auto' })
    await waitTtsChapterLoaded(next)
    const sents = await getChapterSentences(next)
    if (sents.length) {
      ttsChapter.value = next
      continuousReaderRef.value?.resetSentenceSearch?.()
      tts.setSentences(sents)
      return
    }
    next++
  }
  ElMessage.success('全书朗读完毕')
  stopTtsReading()
}

function hideSelToolbar() {
  if (noteMode.value) return
  selToolbar.value = { visible: false, x: 0, y: 0 }
}

function onDocMouseDown(e) {
  // 点击颜色选择器之外 → 收起
  if (showColorPicker.value && !e.target.closest?.('.fr-color-wrap')) {
    showColorPicker.value = false
  }
  if (selToolbar.value.visible &&
      !e.target.closest?.('.fr-sel-toolbar') &&
      !e.target.closest?.('.fr-note-pop')) {
    hideSelToolbar()
  }
}

async function addHighlight(style) {
  const s = selState.value
  if (!s) return
  currentStyle.value = style
  const color = currentColor.value
  // PDF：委托 PdfReader 在当前选区上创建划线
  if (s.isPdf) {
    await pdfRef.value?.pdfApplyHighlight({ style, color, note: '' })
    finishSelection()
    return
  }
  // 连续滚动模式：用 ScrollContinuousReader 的 DOM 包裹
  if (typeof s.blockIdx === 'number' && !isPdf.value) {
    await saveCrAnnotation({ ...s, style, color, note: '', type: 'highlight' })
    finishSelection()
    return
  }
  // PDF 之外但没有 blockIdx 是异常状态，直接忽略
  ElMessage.warning('请在正文中选择文本后再划线')
  finishSelection()
}

function openNote() {
  if (!selState.value) return
  noteMode.value = true
  noteText.value = ''
  selToolbar.value = { ...selToolbar.value, visible: false }
}

async function saveNote() {
  const s = selState.value
  if (!s) return
  const color = currentColor.value
  const style = currentStyle.value || 'highlight'
  // PDF：委托 PdfReader 在当前选区上创建笔记
  if (s.isPdf) {
    await pdfRef.value?.pdfApplyHighlight({ style, color, note: noteText.value, isNote: true })
    finishSelection()
    return
  }
  // 连续滚动模式
  if (typeof s.blockIdx === 'number' && !isPdf.value) {
    await saveCrAnnotation({ ...s, style, color, note: noteText.value, type: 'note' })
    finishSelection()
    return
  }
  ElMessage.warning('请在正文中选择文本后再保存笔记')
  finishSelection()
}

// 连续滚动模式下保存一条标注：先调后端接口，拿到 id 后让 ScrollContinuousReader 包裹 DOM
async function saveCrAnnotation({ blockIdx, text, style, color, note, type }) {
  // 计算 matchIndex：在已存在标注中，相同章节+相同文本的出现次数（用于区分重复文本）
  let matchIndex = 0
  for (const a of annotations.value) {
    if (!isCrAnn(a)) continue
    try {
      const p = JSON.parse(a.position)
      if (p && p.index === blockIdx && p.text === text) matchIndex++
    } catch {}
  }
  try {
    const { data } = await bookApi.createAnnotation(props.id, {
      type,
      content: text,
      note,
      color,
      style,
      // 滚动模式位置：章节 idx + 文本 + 第 N 次出现（用于在 iframe 中精确匹配）
      position: JSON.stringify({ kind: 'cr', index: blockIdx, text, matchIndex }),
      page: 0,
      chapter: String(blockIdx)
    })
    annotations.value.unshift(data)
  } catch (e) {
    ElMessage.error(type === 'note' ? '保存笔记失败' : '添加划线失败')
  }
}

function copySelection() {
  const s = selState.value
  if (s?.text) navigator.clipboard?.writeText(s.text).catch(() => {})
  finishSelection()
}

function finishSelection() {
  selToolbar.value = { visible: false, x: 0, y: 0 }
  noteMode.value = false
  noteText.value = ''
  activeAnnId.value = null
  // 清除连续滚动阅读器内的选区高亮（不影响已保存的标注 DOM）
  continuousReaderRef.value?.clearSelection?.()
  if (isPdf.value) pdfRef.value?.pdfClearSelection?.()
}

async function deleteAnnotation(ann) {
  try { await bookApi.deleteAnnotation(ann.id) } catch {}
  annotations.value = annotations.value.filter(a => a.id !== ann.id)
  // 连续滚动模式：让 ScrollContinuousReader 清理对应 DOM 包裹
  if (isCrAnn(ann)) {
    continuousReaderRef.value?.removeAnnotation?.(ann.id)
  }
  // 非滚动模式（如 PDF）由各自处理；此处不再需要 foliate-view 旧路径
}

async function jumpToAnnotation(ann) {
  activeAnnId.value = ann.id
  showAnnDrawer.value = false
  // 连续滚动模式
  if (isCrAnn(ann)) {
    continuousReaderRef.value?.scrollToAnnotation?.(ann)
    return
  }
  // PDF 等其他格式的批注跳转交给对应阅读器（PdfReader 自身处理）
  if (pdfRef.value?.jumpToAnnotation) {
    try { await pdfRef.value.jumpToAnnotation(ann) } catch {}
  }
}

function onKey(e) {
  if (e.key === 'ArrowLeft') prevPage()
  else if (e.key === 'ArrowRight') nextPage()
  else if (e.key === 'Escape') {
    showToc.value = showSettings.value = searchOpen.value = false
    if (document.fullscreenElement) document.exitFullscreen?.()
  }
}

onMounted(async () => {
  document.addEventListener('keydown', onKey)
  document.addEventListener('mousedown', onDocMouseDown)
  window.addEventListener('pagehide', flushSave)
  window.addEventListener('pagehide', beaconReadingSession)
  window.addEventListener('beforeunload', flushSave)
  prestartFileFetch() // 立即并行下载书籍文件，不等元数据/批注请求
  // 阅读/朗读会话统计：EPUB 与 PDF 统一由本外壳计时上报
  startReadingSession()
  try {
    await loadBookInfo()
    parseSavedTtsLocation()
    tts.loadSettings() // 预热音色/语气/倍速配置（不阻塞首屏）
    if (isPdf.value) {
      // PDF 由内嵌的 PdfReader（pdf.js）自行加载与渲染，外壳只提供工具栏/抽屉
      loading.value = false
      return
    }
    // 批注异步加载、不阻塞首屏（ScrollContinuousReader 监听 annotations 变化后会补应用到已渲染章节）
    loadAnnotations()
    await nextTick()
    await openBook()
  } catch (e) {
    errorMsg.value = e?.message || '加载失败'
    loading.value = false
  }
})

onBeforeUnmount(() => {
  document.removeEventListener('keydown', onKey)
  document.removeEventListener('mousedown', onDocMouseDown)
  window.removeEventListener('pagehide', flushSave)
  window.removeEventListener('pagehide', beaconReadingSession)
  window.removeEventListener('beforeunload', flushSave)
  flushSave() // 先落库进度与朗读断点（请求同步发出）
  tts.stop() // 再停止音频与 WebSocket，避免进行中的合成写入过期断点
  continuousReaderRef.value?.clearSentenceHighlight?.()
  // 上报本次阅读会话（不 await：进度已本地落盘，统计上报异步进行，返回不卡顿）
  reportReadingSession()
})
</script>

<style scoped>
.foliate-reader {
  position: fixed;
  inset: 0;
  display: flex;
  flex-direction: column;
  background: #f1f5f9;
  color: #1f2937;
  z-index: 100;
}
.foliate-reader.fullscreen { background: #000; }

.fr-header {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  row-gap: 4px;
  gap: 8px;
  padding: 10px 16px;
  background: rgba(255, 255, 255, 0.9);
  backdrop-filter: blur(12px);
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  position: relative;
  z-index: 5;
}
.fr-header.hidden { display: none; }
.fr-title {
  font-weight: 600;
  font-size: 15px;
  max-width: 50vw;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: flex;
  align-items: center;
  gap: 6px;
}
.fr-title-book { overflow: hidden; text-overflow: ellipsis; }
.fr-title-sep { color: #94a3b8; font-weight: 400; }
.fr-title-chapter {
  font-weight: 400;
  font-size: 13px;
  color: #64748b;
  overflow: hidden;
  text-overflow: ellipsis;
  flex-shrink: 1;
}
.fr-spacer { flex: 1; }
.fr-zoom { color: #64748b; font-size: 13px; min-width: 42px; text-align: center; }
.fr-icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  border: none;
  border-radius: 10px;
  background: transparent;
  color: #475569;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.fr-icon-btn:hover { background: #eef2ff; color: var(--app-primary, #6366f1); }
/* 批注按钮用 el-badge 包裹图标。el-badge 默认是 inline-block 且带自身 line-height，
   会把图标整体推离按钮中心，看起来与其他图标不在同一水平线；
   这里让 badge 自身也参与 flex 居中即可对齐。 */
.fr-icon-btn .el-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  line-height: 1;
}

/* 移动端头栏：按钮全部收进第一行——不换行、间隙收紧；隐藏占位/缩放%/朗读/全屏；
   书名占用剩余空间，显示不下用省略号；极窄屏允许横向滚动（仍在第一行） */
@media (max-width: 768px) {
  .fr-header {
    flex-wrap: nowrap;
    gap: 2px;
    padding: 8px;
    overflow-x: auto;
    scrollbar-width: none;
    -webkit-overflow-scrolling: touch;
  }
  .fr-header::-webkit-scrollbar { display: none; }
  .fr-header .fr-spacer,
  .fr-header .fr-zoom { display: none; }
  .fr-header .mobile-hide { display: none !important; }
  .fr-header .fr-icon-btn { flex: 0 0 auto; width: 34px; height: 34px; }
  /* 书名：弹性占满剩余空间，窄时省略号；手机端隐藏章节名，把空间让给书名 */
  .fr-header .fr-title { flex: 1 1 auto; min-width: 0; max-width: none; }
  .fr-header .fr-title .fr-title-sep,
  .fr-header .fr-title .fr-title-chapter { display: none; }
  .fr-header .fr-title .fr-title-book { min-width: 0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
}

.fr-body {
  position: relative;
  flex: 1;
  overflow: hidden;
  background: #e9eef5;
}
.foliate-host {
  display: block;
  width: 100%;
  height: 100%;
}
.fr-loading, .fr-error {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
  align-items: center;
  justify-content: center;
  color: #64748b;
  background: inherit;
}
.fr-error { color: #ef4444; }

.fr-page-zone {
  position: absolute;
  top: 0;
  bottom: 0;
  width: 88px;
  cursor: pointer;
  z-index: 10;
  transition: background 0.2s ease;
}
.fr-page-zone.left { left: 0; }
.fr-page-zone.right { right: 0; }
.fr-page-zone.disabled { pointer-events: none; }

/* 悬停底色（与 PDF 翻页热区一致，使用主题色），加宽范围并增强可见度 */
.fr-page-zone.left:hover {
  background: linear-gradient(to right, color-mix(in srgb, var(--app-primary) 20%, transparent) 0%, transparent 78%);
}
.fr-page-zone.right:hover {
  background: linear-gradient(to left, color-mix(in srgb, var(--app-primary) 20%, transparent) 0%, transparent 78%);
}

/* 左右翻页箭头（主题色） */
.fr-page-zone.left::before,
.fr-page-zone.right::before {
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
.fr-page-zone.left::before {
  left: 28px;
  transform: translateY(-50%) rotate(45deg);
}
.fr-page-zone.right::before {
  right: 28px;
  transform: translateY(-50%) rotate(-135deg);
}
.fr-page-zone:hover::before {
  opacity: 1;
}

/* 选区浮动工具条 */
.fr-sel-toolbar {
  position: fixed;
  transform: translate(-50%, -100%);
  z-index: 60;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.18);
  max-width: 92vw;
  flex-wrap: wrap;
}
.fr-sel-colors { display: flex; gap: 6px; }
.fr-sel-color {
  width: 22px; height: 22px; border-radius: 50%;
  border: 2px solid transparent; cursor: pointer; padding: 0;
  transition: transform 0.12s;
}
.fr-sel-color.active { border-color: #1f2937; transform: scale(1.12); }
.fr-sel-divider { width: 1px; height: 22px; background: #e2e8f0; }
.fr-sel-styles, .fr-sel-actions { display: flex; gap: 4px; }
.fr-sel-style, .fr-sel-action {
  border: none; background: #f1f5f9; color: #334155;
  font-size: 12px; padding: 5px 9px; border-radius: 8px; cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.fr-sel-style:hover, .fr-sel-action:hover { background: #e2e8f0; }
.fr-sel-style.active { background: var(--app-primary, #6366f1); color: #fff; }

/* ============ 顶部划线工具组（持续划线模式）============ */
.fr-draw-tools {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
}
.fr-tool-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border: none;
  border-radius: 9px;
  background: transparent;
  color: #475569;
  cursor: pointer;
  padding: 0;
  transition: background 0.15s, color 0.15s, box-shadow 0.15s;
}
.fr-tool-btn:hover { background: #eef2ff; color: var(--app-primary, #6366f1); }
.fr-tool-btn.active {
  background: #eef2ff;
  color: var(--app-primary, #6366f1);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--app-primary, #6366f1) 45%, transparent);
}
.fr-tool-ico { display: inline-flex; align-items: center; justify-content: center; }
.fr-tool-ico svg { display: block; }
/* 工具图标经 v-html 注入，scoped 属性不会落到内部元素上，用 :deep() 后代选择器穿透 */
.fr-tool-ico :deep(.fr-di) { display: block; }
/* 工具图标用当前划线颜色着色，跟随颜色选择联动 */
.fr-tool-ico :deep(.fr-di-hl) { width: 18px; height: 12px; border-radius: 3px; background: currentColor; opacity: 0.55; }
.fr-tool-ico :deep(.fr-di-ul) { width: 18px; height: 12px; position: relative; }
.fr-tool-ico :deep(.fr-di-ul)::after {
  content: ''; position: absolute; left: 0; right: 0; bottom: 1px;
  height: 2px; border-radius: 1px; background: currentColor;
}
.fr-tool-ico :deep(.fr-di-st) { width: 18px; height: 12px; position: relative; }
.fr-tool-ico :deep(.fr-di-st)::after {
  content: ''; position: absolute; left: 0; right: 0; top: 50%;
  height: 2px; border-radius: 1px; background: currentColor;
  transform: translateY(-50%);
}
.fr-color-wrap { position: relative; display: inline-flex; }
.fr-tool-color {
  display: block;
  width: 16px; height: 16px;
  border-radius: 50%;
  border: 2px solid #fff;
  box-shadow: 0 0 0 1px #cbd5e1;
}
.fr-color-pop {
  position: absolute;
  top: calc(100% + 6px);
  right: 0;
  display: flex;
  gap: 6px;
  padding: 8px;
  background: #fff;
  border-radius: 10px;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.18);
  z-index: 70;
}
.fr-color-opt {
  width: 22px; height: 22px;
  border-radius: 50%;
  border: 2px solid transparent;
  cursor: pointer;
  transition: transform 0.12s;
}
.fr-color-opt.active { border-color: #1f2937; transform: scale(1.12); }

/* 移动端选择模式提示条 */
.fr-select-hint {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 7px 12px;
  font-size: 12.5px;
  color: #475569;
  background: #eef2ff;
  border-bottom: 1px solid #e0e7ff;
}
.fr-select-hint-btn {
  border: none;
  background: var(--app-primary, #6366f1);
  color: #fff;
  font-size: 12px;
  padding: 3px 12px;
  border-radius: 999px;
  cursor: pointer;
  flex-shrink: 0;
}

/* 头部图标按钮激活态（朗读中） */
.fr-icon-btn.active {
  background: #eef2ff;
  color: var(--app-primary, #6366f1);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--app-primary, #6366f1) 45%, transparent);
}

/* ============ 语音朗读面板 ============ */
.fr-tts-drawer { width: 320px; }
.fr-tts-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px;
  flex: 1;
  overflow-y: auto;
}
/* 从选中位置开始阅读 */
.fr-tts-read-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  width: 100%;
  padding: 10px 12px;
  border: none;
  border-radius: 10px;
  background: var(--app-primary, #6366f1);
  color: #fff;
  font-size: 13.5px;
  cursor: pointer;
  transition: filter 0.15s;
}
.fr-tts-read-btn:hover { filter: brightness(1.08); }
.fr-tts-read-btn.secondary {
  background: #eef2ff;
  color: var(--app-primary, #6366f1);
}
/* 继续阅读（断点续播）：绿色以区别于“开始朗读” */
.fr-tts-read-btn.resume {
  background: linear-gradient(135deg, #10b981, #34d399);
  box-shadow: 0 6px 16px rgba(16, 185, 129, 0.32);
}
.fr-tts-read-btn.is-pending {
  background: #fef3c7;
  color: #b45309;
  animation: fr-tts-pulse 1.6s ease-in-out infinite;
}
@keyframes fr-tts-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.72; }
}
/* 朗读中：当前句 + 控制 */
.fr-tts-playing {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
}
.fr-tts-current {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  min-height: 38px;
  max-height: 76px;
  overflow-y: auto;
  font-size: 13px;
  line-height: 1.6;
  color: #334155;
}
.fr-tts-current .el-icon { flex-shrink: 0; margin-top: 3px; }
.fr-tts-ctrl {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
}
.fr-tts-ctrl-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border: none;
  border-radius: 50%;
  background: #fff;
  color: #475569;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.12);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.fr-tts-ctrl-btn:hover:not(:disabled) { background: #eef2ff; color: var(--app-primary, #6366f1); }
.fr-tts-ctrl-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.fr-tts-ctrl-btn.primary {
  width: 42px;
  height: 42px;
  background: var(--app-primary, #6366f1);
  color: #fff;
}
.fr-tts-ctrl-btn.primary:hover:not(:disabled) { filter: brightness(1.1); background: var(--app-primary, #6366f1); color: #fff; }
.fr-tts-ctrl-btn.danger:hover:not(:disabled) { background: #fee2e2; color: #ef4444; }
.fr-tts-progress {
  text-align: center;
  font-size: 12px;
  color: #94a3b8;
}
/* 设置行 */
.fr-tts-opt {
  display: flex;
  align-items: center;
  gap: 10px;
}
.fr-tts-label {
  flex-shrink: 0;
  width: 32px;
  font-size: 12.5px;
  color: #64748b;
}
.fr-tts-speed-val {
  flex-shrink: 0;
  width: 36px;
  text-align: right;
  font-size: 12.5px;
  color: #64748b;
}
.fr-tts-tip {
  margin: 4px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: #94a3b8;
}

/* 复用 PDF 工具栏的高亮/下划线/波浪/删除线 图标样式 */
.fr-sel-ico {
  display: inline-block;
  width: 20px; height: 12px;
  margin-right: 5px;
  position: relative;
  vertical-align: -1px;
}
.fr-sel-ico-hl { background: currentColor; opacity: 0.6; border-radius: 3px; }
.fr-sel-ico-ul, .fr-sel-ico-st {
  position: absolute; left: 2px; right: 2px; height: 2px; background: currentColor;
}
.fr-sel-ico-ul { bottom: 2px; }
.fr-sel-ico-st { top: 50%; transform: translateY(-50%); }

/* 笔记输入弹窗 */
.fr-note-pop {
  position: fixed;
  transform: translate(-50%, calc(-100% - 12px));
  z-index: 60;
  width: min(360px, 90vw);
  background: #fff;
  border-radius: 14px;
  box-shadow: 0 12px 36px rgba(0, 0, 0, 0.2);
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.fr-note-head { font-weight: 600; font-size: 14px; color: #1f2937; }
.fr-note-quote {
  font-size: 12px; color: #64748b; line-height: 1.5;
  max-height: 48px; overflow: hidden;
  display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical;
}
.fr-note-actions { display: flex; justify-content: flex-end; gap: 8px; }

/* 批注与笔记列表 */
.fr-ann-list { overflow-y: auto; padding: 8px 0; }
.fr-ann-item {
  display: flex; align-items: flex-start; gap: 10px;
  padding: 10px 16px; cursor: pointer; transition: background 0.15s;
  border-left: 3px solid transparent;
}
.fr-ann-item:hover { background: #f5f7ff; }
.fr-ann-item.active { border-left-color: var(--app-primary, #6366f1); background: #f5f7ff; }
.fr-ann-dot { width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0; margin-top: 5px; }
.fr-ann-body { flex: 1; min-width: 0; }
.fr-ann-text {
  font-size: 13px; color: #334155; line-height: 1.5;
  display: -webkit-box; -webkit-line-clamp: 3; -webkit-box-orient: vertical; overflow: hidden;
}
.fr-ann-note {
  margin-top: 4px; font-size: 12px; color: #64748b; line-height: 1.5;
  background: #f8fafc; border-radius: 6px; padding: 6px 8px;
}

.fr-footer {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 8px 18px;
  background: rgba(255, 255, 255, 0.9);
  border-top: 1px solid rgba(0, 0, 0, 0.06);
  font-size: 13px;
  color: #64748b;
  z-index: 5;
}
.fr-footer.hidden { display: none; }
.fr-percent { width: 38px; text-align: right; }
.fr-progress {
  flex: 1;
  height: 6px;
  border-radius: 6px;
  background: #e2e8f0;
  /* 标记点需溢出轨道显示，去掉 hidden，圆角改由 fill 自身承担 */
  overflow: visible;
  position: relative;
  cursor: pointer;
}
.fr-progress-fill {
  height: 100%;
  border-radius: 6px;
  background: var(--app-gradient, linear-gradient(135deg, #6366f1, #8b5cf6));
}
/* 进度条上的划线/笔记位置标记点 */
.fr-ann-marker {
  position: absolute;
  top: 50%;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #f59e0b;
  transform: translate(-50%, -50%);
  pointer-events: none;
  box-shadow: 0 0 0 1.5px #fff;
}
.fr-ann-marker.note { background: #ef4444; border-radius: 2px; }

/* 抽屉 */
.fr-drawer {
  position: absolute;
  top: 0;
  bottom: 0;
  width: 300px;
  max-width: 80vw;
  background: #fff;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.18);
  z-index: 20;
  display: flex;
  flex-direction: column;
}
.fr-drawer-left { left: 0; }
.fr-drawer-right { right: 0; }
.fr-drawer-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px;
  font-weight: 600;
  border-bottom: 1px solid #eef2f7;
}
.fr-close { cursor: pointer; color: #94a3b8; }
.fr-close:hover { color: #ef4444; }
.fr-toc { overflow-y: auto; padding: 8px 0; }
.fr-toc-item {
  padding: 9px 12px;
  font-size: 14px;
  color: #334155;
  cursor: pointer;
  border-left: 3px solid transparent;
}
.fr-toc-item:hover { background: #f5f7ff; border-left-color: var(--app-primary, #6366f1); }
.fr-empty { padding: 24px; text-align: center; color: #94a3b8; font-size: small; }

.fr-settings { padding: 16px; overflow-y: auto; }
.fr-setting-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 0;
  border-bottom: 1px solid #f1f5f9;
  font-size: 14px;
}
.fr-stepper { display: flex; align-items: center; gap: 10px; }
.fr-value { min-width: 42px; text-align: center; }
.fr-themes { display: flex; gap: 8px; }
.fr-theme-dot {
  width: 30px;
  height: 30px;
  border-radius: 50%;
  border: 2px solid transparent;
  font-weight: 700;
  cursor: pointer;
}
.fr-theme-dot.active { border-color: var(--app-primary, #6366f1); }

.fr-search-panel {
  position: absolute;
  top: 60px;
  left: 50%;
  transform: translateX(-50%);
  width: min(560px, 92vw);
  background: #fff;
  border-radius: 14px;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.18);
  padding: 12px;
  z-index: 30;
}
.fr-search-list { max-height: 40vh; overflow-y: auto; margin-top: 8px; }
.fr-search-item {
  padding: 8px 10px;
  border-radius: 8px;
  cursor: pointer;
  font-size: 13px;
  color: #334155;
}
.fr-search-item:hover { background: #f5f7ff; }
.fr-search-excerpt { display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
.fr-search-empty { padding: 14px; text-align: center; color: #94a3b8; }

.fr-slide-enter-active, .fr-slide-leave-active { transition: transform 0.22s ease; }
.fr-slide-enter-from.fr-drawer-left, .fr-slide-leave-to.fr-drawer-left { transform: translateX(-100%); }
.fr-slide-enter-from.fr-drawer-right, .fr-slide-leave-to.fr-drawer-right { transform: translateX(100%); }
</style>
