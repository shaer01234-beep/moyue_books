<template>
  <div
    class="scr-root"
    ref="rootEl"
    :style="{ opacity: positioned ? 1 : 0 }"
    @scroll.passive="onScroll"
    @touchstart.passive="onTouchStart"
    @touchmove.passive="onTouchMove"
  >
    <!-- 所有章节包进同一张连续纸张，章节间不再以独立卡片/间隙"断开" -->
    <div class="scr-column">
      <div
        v-for="block in blocks"
        :key="block.idx"
        class="scr-block"
        :class="{ 'scr-empty': block.isEmpty }"
        :data-idx="block.idx"
        :style="{ height: (block.height || estimatedHeight) + 'px' }"
      >
        <iframe
          v-if="block.loaded && block.src"
          :ref="el => setIframeRef(block.idx, el)"
          :src="block.src"
          sandbox="allow-same-origin allow-scripts"
          class="scr-iframe"
          @load="onIframeLoad(block)"
          scrolling="no"
        />
        <div v-else class="scr-placeholder">
          <span v-if="!block.error" class="scr-placeholder-dot" />
          <button v-else class="scr-reload" @click="loadBlock(block.idx)">加载失败，点击重试</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import { normTtsText } from '@/utils/ttsPlayer'

const props = defineProps({
  sections: { type: Array, required: true },
  fontCss: { type: String, default: '' },
  initialIndex: { type: Number, default: 0 },
  initialFraction: { type: Number, default: 0 },
  annotations: { type: Array, default: () => [] },
  // 移动端选择模式：划线/笔记工具开启时父组件置 true，
  // iframe 内允许文本选中并屏蔽系统长按呼出的菜单
  selectMode: { type: Boolean, default: false },
})

const emit = defineEmits(['progress', 'ready', 'section-loaded', 'selection', 'selection-clear', 'annotation-click'])

const blocks = ref([])
const rootEl = ref(null)
const positioned = ref(false)   // 揭幕标志：定位完成前容器透明，避免进入时"从开头跳到进度"的闪烁
const iframeRefs = new Map()
let resizeObserver = null
let lastProgressEmit = 0
const defaultBlockHeight = 600

// ===== 按需加载参数 =====
// 进入时只加载当前进度附近（前 1 + 后 2 页），不再一次性铺开整本书
const INITIAL_BEFORE = 1    // 打开时：当前章节之前预载的章节数（仅小缓冲，避免回看空白）
const INITIAL_AFTER = 2     // 打开时：当前章节之后预载的章节数（进入只加载当前进度附近）
// 滚动时：下滑到接近已加载内容底部 / 上滑到接近顶部时，按"下一页"方式预载前后若干页
const PRELOAD_AHEAD = 3     // 触发后单方向预载的后续章节数
const PRELOAD_TRIGGER = 1.5 // 距顶部/底部不足该倍数屏幕高度时触发对应方向预载（提前量，覆盖"内容短"场景）

// 单节估算高度：未加载章节的占位高度。
// 默认 1800px（中文 EPUB 章节典型高度），用中位数（而非均值）抗极值干扰，
// 最低 600px 避免占位过矮（极短章）
const estimatedHeight = computed(() => {
  const heights = []
  for (const b of blocks.value) {
    if (b.loaded && b.height > 0 && b.height < 8000) heights.push(b.height)
  }
  if (!heights.length) return 1800
  heights.sort((a, b) => a - b)
  const median = heights[Math.floor(heights.length / 2)]
  return Math.max(600, median)
})

function setIframeRef(idx, el) {
  if (el) iframeRefs.set(idx, el)
  else iframeRefs.delete(idx)
}

onMounted(async () => {
  // 初始化所有 block（高度 0 占位，加载后测量）
  blocks.value = props.sections.map((sec, idx) => ({
    idx,
    sec,
    loaded: false,
    loading: false,
    error: false,
    src: null,
    height: 0,
    isEmpty: false,
  }))

  const total = props.sections.length
  const cur = Math.max(0, Math.min(props.initialIndex || 0, total - 1))
  // 按需加载：初始仅加载当前章节附近的小窗口（前 INITIAL_BEFORE + 后 INITIAL_AFTER），
  // 首屏不阻塞；其余章节完全随滚动按需加载，避免一次性把整本书的 iframe 都创建出来导致打开极慢。
  const initialStart = Math.max(0, cur - INITIAL_BEFORE)
  const initialEnd = Math.min(total - 1, cur + INITIAL_AFTER)
  const initialLoadPromise = loadRange(initialStart, initialEnd, 6)

  // 监听首屏章节加载完成 → 定位到初始位置并揭幕（揭幕前容器透明，避免"从开头跳到进度"的闪烁）
  const onFirstLoaded = () => {
    nextTick(() => {
      if (rootEl.value) scrollToInitial()
      positioned.value = true
      setupResizeObserver()
      emit('ready')
    })
  }
  // 当前章节加载完成时即触发；首屏内任意章节加载完也作为兜底
  let stopped = false
  const stopWatch = () => { stopped = true }
  watch(() => blocks.value[cur]?.loaded, (loaded) => {
    if (loaded && !stopped) { onFirstLoaded(); stopWatch() }
  }, { immediate: true })
  // 兜底：等初始窗口内任意章节加载完（防止当前章节加载失败导致 ready 永远不发）
  watch(() => blocks.value.slice(initialStart, initialEnd + 1).some(b => b.loaded), (anyLoaded) => {
    if (anyLoaded && !stopped) { onFirstLoaded(); stopWatch() }
  }, { immediate: true })

  // 渐进式校正（用户尚未滚动时，用更多真实高度修正估算偏差）：500ms / 1.5s / 3s 三个时间点
  ;[500, 1500, 3000].forEach(delay => {
    setTimeout(() => {
      if (!userScrolled && rootEl.value) scrollToInitial()
    }, delay)
  })
  // 兜底：若首屏章节始终未加载（异常/超时），4s 后强制揭幕并定位，避免页面长期空白
  setTimeout(() => {
    if (!positioned.value && rootEl.value) { scrollToInitial(); positioned.value = true }
  }, 4000)

  // 初始窗口加载完成后，按当前视口做一次预加载（覆盖首屏上下方）；
  // 之后完全由 onScroll 触发的 triggerPreload 按需加载，不再一次性预载整本书。
  initialLoadPromise.then(() => triggerPreload()).catch(() => {})
})

// 按范围并发加载：浏览器对同域 iframe 并行加载上限约 6，提升至 6 充分利用
async function loadRange(from, to, concurrency = 6) {
  if (from > to) return
  const ids = []
  for (let i = from; i <= to; i++) ids.push(i)
  let cursor = 0
  const workers = Array.from({ length: Math.min(concurrency, ids.length) }, async () => {
    while (true) {
      const i = cursor++
      if (i >= ids.length) return
      // eslint-disable-next-line no-await-in-loop
      await loadBlock(ids[i])
    }
  })
  await Promise.all(workers)
}

onBeforeUnmount(() => {
  if (resizeObserver) {
    resizeObserver.disconnect()
    resizeObserver = null
  }
  // 释放已加载的 section（foliate Loader 引用计数）
  for (const block of blocks.value) {
    if (block.loaded && block.sec && typeof block.sec.unload === 'function') {
      try { block.sec.unload() } catch (e) { /* noop */ }
    }
  }
  iframeRefs.clear()
})

async function loadBlock(idx) {
  const block = blocks.value[idx]
  if (!block || block.loaded || block.loading || !block.sec) return
  block.loading = true
  block.error = false
  // 最多重试 2 次（间隔 200ms），避免偶发失败导致该章节永远空白
  for (let attempt = 0; attempt < 3; attempt++) {
    try {
      const src = await block.sec.load()
      block.src = src
      block.loaded = true
      block.loading = false
      blocks.value = [...blocks.value]
      emit('section-loaded', idx)
      return
    } catch (e) {
      if (attempt === 2) {
        console.warn(`[ScrollContinuousReader] load section ${idx} failed after 3 attempts`, e)
      } else {
        await new Promise(r => setTimeout(r, 200))
      }
    }
  }
  // 三次都失败：标记错误，停止转圈，改为可点击重试
  block.loading = false
  block.error = true
  blocks.value = [...blocks.value]
}

async function waitForHeights(start, end) {
  for (let i = start; i <= end; i++) {
    // eslint-disable-next-line no-await-in-loop
    await waitForBlockHeight(i, 50)
  }
}

async function waitForBlockHeight(idx, maxTries = 30) {
  let tries = 0
  while (tries < maxTries) {
    const block = blocks.value[idx]
    if (!block || !block.loaded) return
    if (block.height > 50) return
    await new Promise(r => setTimeout(r, 80))
    tries++
  }
}

// 用已加载章节估算单节平均高度（响应式 computed 在顶部已定义 estimatedHeight）
// 这里保留 alias 给 scrollToInitial 使用
function estimatedBlockHeight() { return estimatedHeight.value }

function scrollToInitial() {
  if (!rootEl.value) return
  const idx = Math.max(0, Math.min(props.initialIndex, blocks.value.length - 1))
  const est = estimatedBlockHeight()
  let y = 0
  let loadedCount = 0
  let loadedHeight = 0
  // 分段累加：已加载章节用真实高度，未加载章节用估算；
  // 累计已加载章节数和高度，用于后续"已加载段平均高度"校正估算偏差
  for (let i = 0; i < idx; i++) {
    const b = blocks.value[i]
    if (b.loaded) {
      y += b.height
      if (!b.isEmpty) {
        loadedHeight += b.height
        loadedCount++
      }
    } else {
      y += est
    }
  }
  // 当 idx 前面有较多已加载章节时，用它们的平均高度替换估算值
  // （解决"前面的章节没加载完时估算偏差大"的问题）
  if (loadedCount >= 5) {
    const realAvg = loadedHeight / loadedCount
    // 把前面那段"已经累加的估算值"重新计算：用真实平均
    // （取真实平均与全局估算的中位数，兼顾已加载/未加载段）
    const effectiveEst = Math.round((realAvg + est) / 2)
    let y2 = 0
    for (let i = 0; i < idx; i++) {
      const b = blocks.value[i]
      y2 += b.loaded ? b.height : effectiveEst
    }
    y = y2
  }
  const curBlock = blocks.value[idx]
  const curH = curBlock?.loaded ? curBlock.height : est
  y += curH * props.initialFraction
  // 用 scrollTo + behavior:'auto' 避免全局 scroll-behavior:smooth 带来的
  // 长距离平滑动画（看起来像"快速滚动穿过多章"）
  rootEl.value.scrollTo({ top: y, behavior: 'auto' })
}

function onIframeLoad(block) {
  nextTick(() => {
    injectStyles(block)
    injectAnnStyles(block)
    injectSelectMode(block)
    if (drawCursor) setIframeCursor(block, drawCursor)
    measureHeight(block)
    bindSelectionEvents(block)
    applyAnnotationsForBlock(block)
  })
}

// 选择模式：注入 iframe 样式（允许文本选中、屏蔽系统长按呼出的查询/分享菜单）
const SELECT_MODE_CSS = `
.cr-select-mode, .cr-select-mode * {
  -webkit-user-select: text !important;
  user-select: text !important;
  -webkit-touch-callout: none !important;
}
`.trim()

function injectSelectMode(block) {
  const iframe = iframeRefs.get(block.idx)
  if (!iframe) return
  try {
    const doc = iframe.contentDocument
    if (!doc) return
    let styleEl = doc.getElementById('__cr_selectmode_style')
    if (!styleEl) {
      styleEl = doc.createElement('style')
      styleEl.id = '__cr_selectmode_style'
      if (doc.head) doc.head.appendChild(styleEl)
      else if (doc.documentElement) doc.documentElement.appendChild(styleEl)
    }
    styleEl.textContent = props.selectMode ? SELECT_MODE_CSS : ''
    doc.body?.classList.toggle('cr-select-mode', !!props.selectMode)
    doc.documentElement?.classList.toggle('cr-select-mode', !!props.selectMode)
  } catch (e) { /* noop */ }
}

// 选择模式切换时同步到所有已加载章节（后加载的章节在 onIframeLoad 中应用）
watch(() => props.selectMode, () => {
  for (const block of blocks.value) {
    if (block.loaded) injectSelectMode(block)
  }
})

// 划线模式的笔形光标：记录当前值，新加载章节同样应用
let drawCursor = ''
function setIframeCursor(block, cursor) {
  const iframe = iframeRefs.get(block.idx)
  try {
    const doc = iframe?.contentDocument
    if (doc?.body) doc.body.style.cursor = cursor || ''
  } catch (e) { /* noop */ }
}

// 在 iframe 中注入划线/笔记样式表（按颜色与样式渲染高亮/下划线/波浪/删除线）
function injectAnnStyles(block) {
  const iframe = iframeRefs.get(block.idx)
  if (!iframe) return
  try {
    const doc = iframe.contentDocument
    if (!doc) return
    let styleEl = doc.getElementById('__cr_ann_style')
    if (!styleEl) {
      styleEl = doc.createElement('style')
      styleEl.id = '__cr_ann_style'
      if (doc.head) doc.head.appendChild(styleEl)
      else if (doc.documentElement) doc.documentElement.appendChild(styleEl)
    }
    styleEl.textContent = ANN_STYLE_CSS
  } catch (e) { /* noop */ }
}

// 划线/笔记的样式（注入到每个 iframe 文档），由 wrapWithAnn 应用类名 + data 属性
const ANN_STYLE_CSS = `
.fr-ann { cursor: pointer; border-radius: 2px; transition: background-color .15s; }
.fr-ann[data-style="highlight"] { background-color: var(--fr-ann-bg, rgba(255,235,59,0.45)); }
.fr-ann[data-style="underline"] { text-decoration: underline; text-decoration-thickness: 2px; text-underline-offset: 2px; }
.fr-ann[data-style="wavy"] { text-decoration: underline wavy; text-decoration-thickness: 2px; text-underline-offset: 3px; }
.fr-ann[data-style="strikethrough"] { text-decoration: line-through; text-decoration-thickness: 2px; }
.fr-ann:hover { filter: brightness(0.95); }
.fr-ann.active { outline: 2px solid var(--app-primary, #6366f1); outline-offset: 1px; }
.fr-ann[data-has-note="1"]::after {
  content: '📝'; font-size: 0.8em; margin-left: 2px; vertical-align: super; opacity: 0.7;
}
/* 语音朗读：当前句所在段落的整段高亮 */
.tts-speaking-block {
  background: linear-gradient(180deg, rgba(250, 204, 21, 0.35), rgba(251, 191, 36, 0.42)) !important;
  border-radius: 4px !important;
  transition: background 0.2s;
}
`.trim()

// 解析 annotation 的 position 字段，提取连续滚动阅读器的格式
function parseAnnPosition(ann) {
  if (!ann || !ann.position) return null
  try {
    const p = JSON.parse(ann.position)
    if (p && p.kind === 'cr') return p
  } catch {}
  return null
}

// 绑定 iframe 内选区事件（mouseup/touchend → emit selection；mousedown → 清理）
function bindSelectionEvents(block) {
  const iframe = iframeRefs.get(block.idx)
  if (!iframe) return
  try {
    const doc = iframe.contentDocument
    if (!doc || doc.__crSelBound) return
    doc.__crSelBound = true
    const win = iframe.contentWindow
    doc.addEventListener('mouseup', () => onDocMouseUp(block))
    doc.addEventListener('touchend', () => onDocMouseUp(block))
    doc.addEventListener('mousedown', () => {
      // 在 iframe 内按下鼠标，若已有选区则通知父组件清理（让旧工具条消失）
      setTimeout(() => {
        try {
          const sel = doc.getSelection()
          if (!sel || sel.isCollapsed) emit('selection-clear')
        } catch {}
      }, 0)
    })
    // 标注点击：转发到父组件（用 closest 兼容跨元素包裹场景，点击内部子元素也能命中）
    doc.addEventListener('click', (e) => {
      const t = e.target
      const annEl = (t && typeof t.closest === 'function') ? t.closest('.fr-ann') : null
      if (annEl) {
        const id = annEl.dataset.id
        if (id) emit('annotation-click', { id: Number(id), blockIdx: block.idx })
      }
    })
  } catch (e) { /* noop */ }
}

// iframe 内 mouseup：把选区范围连同 iframe 在父坐标系的位置发出去
function onDocMouseUp(block) {
  const iframe = iframeRefs.get(block.idx)
  if (!iframe) return
  try {
    const doc = iframe.contentDocument
    const win = iframe.contentWindow
    if (!doc || !win) return
    const sel = doc.getSelection?.() || win.getSelection?.()
    if (!sel || sel.rangeCount === 0 || sel.isCollapsed) {
      emit('selection-clear')
      return
    }
    const text = sel.toString().trim()
    if (!text || text.length < 2) { emit('selection-clear'); return }
    const range = sel.getRangeAt(0)
    const rect = range.getBoundingClientRect()
    if (!rect || (rect.width === 0 && rect.height === 0)) { emit('selection-clear'); return }
    // 把 iframe 视口坐标转换为父文档坐标
    const iframeRect = iframe.getBoundingClientRect()
    emit('selection', {
      blockIdx: block.idx,
      text,
      range, // Range 对象，跨 iframe 通过 postMessage 也无法拷贝，故父组件仅用 text+matchIndex
      x: iframeRect.left + rect.left + rect.width / 2,
      y: iframeRect.top + rect.top - 6,
      // 框选矩形（用于父组件定位工具条）
      rect: {
        top: iframeRect.top + rect.top,
        left: iframeRect.left + rect.left,
        width: rect.width,
        height: rect.height
      }
    })
  } catch (e) { /* noop */ }
}

// 给指定章节块应用某条 annotation（在 DOM 中包裹文字）
function applyAnnotationToBlock(ann) {
  const pos = parseAnnPosition(ann)
  if (!pos || typeof pos.index !== 'number') return false
  const iframe = iframeRefs.get(pos.index)
  if (!iframe) return false
  const doc = iframe.contentDocument
  if (!doc || !doc.body) return false
  // 已存在同 id 标注则不重复包裹
  if (doc.querySelector(`.fr-ann[data-id="${ann.id}"]`)) return true
  const range = findTextRange(doc.body, pos.text || ann.content || '', pos.matchIndex || 0)
  if (!range) {
    console.warn('[ScrollContinuousReader] 找不到标注文字:', ann.content?.slice(0, 30))
    return false
  }
  // range.surroundContents 不能跨元素；拆分为多段分别包裹
  wrapRangeWithSpans(doc, range, ann)
  return true
}

// 给指定章节块应用所有匹配 annotation
function applyAnnotationsForBlock(block) {
  if (!props.annotations?.length) return
  for (const ann of props.annotations) {
    const pos = parseAnnPosition(ann)
    if (!pos || pos.index !== block.idx) continue
    try {
      applyAnnotationToBlock(ann)
      appliedAnnIds.add(ann.id)
    } catch (e) { console.warn('applyAnnotation failed', e) }
  }
}

// ===== 语音朗读：逐句高亮 =====
// 上一次句子命中结束的文本位置；同章顺序朗读从该位置继续搜索，
// 避免短句/重复文本命中更早位置（如上一句内）导致"高亮的是上一句"
let ttsSearchPos = 0
// 上一句高亮的块元素：切句时只清除它，避免整篇累积高亮
let lastTtsBlock = null
const TTS_BLOCK_RE = /^(P|DIV|H1|H2|H3|H4|H5|H6|LI|BLOCKQUOTE|SECTION|ARTICLE|TD|TH|TR)$/i

// 去掉字母数字以外的一切（标点/引号/破折号/空白），用于服务端文本与
// EPUB 渲染文本标点字形不一致时的容错匹配
const TTS_STRIP_RE = /[^\p{L}\p{N}]+/gu

// 按需确保 iframe 文档内有朗读高亮样式（兜底：injectAnnStyles 未执行或文档被重载清空时）
function ensureTtsStyle(doc) {
  try {
    if (!doc || !doc.head) return
    if (doc.getElementById('__cr_tts_style')) return
    const s = doc.createElement('style')
    s.id = '__cr_tts_style'
    s.textContent = '.tts-speaking-block{background:linear-gradient(180deg,rgba(250,204,21,0.35),rgba(251,191,36,0.42))!important;border-radius:4px!important;transition:background .2s;}'
    doc.head.appendChild(s)
  } catch (e) { /* noop */ }
}

// 在单个 iframe 文档中定位句子，返回 { block, nextPos }；未命中返回 null。
// 依次尝试：精确匹配（顺序位置/从头）→ 句首 8 字片段 → 去标点整句 → 去标点句首片段
function findSentenceInDoc(doc, sent, fromPos) {
  const target = normTtsText(sent)
  if (!target) return null
  // 收集正文文本节点（跳过 script/style 与空白节点），保留两种形态：
  // raws = 仅去空白（精确匹配空间），strips = 再去标点（容错匹配空间）
  const nodes = []
  const raws = []
  const strips = []
  const walker = doc.createTreeWalker(doc.body, NodeFilter.SHOW_TEXT)
  let n
  while ((n = walker.nextNode())) {
    const p = n.parentElement
    if (p && p.closest && p.closest('script, style')) continue
    const t = normTtsText(n.nodeValue)
    if (!t) continue
    nodes.push(n)
    raws.push(t)
    strips.push(t.replace(TTS_STRIP_RE, ''))
  }
  if (!nodes.length) return null
  let full = ''
  let sfull = ''
  const lens = []
  const slens = []
  for (let k = 0; k < nodes.length; k++) {
    full += raws[k]
    sfull += strips[k]
    lens.push(raws[k].length)
    slens.push(strips[k].length)
  }
  // 在指定匹配空间里把字符位移映射回文本节点，再提升到块级元素
  const hitBlock = (idx, lensArr) => {
    let acc = 0
    let hitNode = null
    for (let k = 0; k < nodes.length; k++) {
      if (lensArr[k] > 0 && idx < acc + lensArr[k]) { hitNode = nodes[k]; break }
      acc += lensArr[k]
    }
    if (!hitNode) return null
    let block = hitNode.parentElement
    while (block && !TTS_BLOCK_RE.test(block.tagName)) {
      block = block.parentElement
    }
    return block
  }
  // 1) 精确匹配：先从上次命中结束位置之后（顺序朗读），找不到再从头
  let idx = full.indexOf(target, fromPos)
  if (idx < 0) idx = full.indexOf(target)
  if (idx >= 0) {
    const block = hitBlock(idx, lens)
    if (block) return { block, nextPos: idx + target.length }
  }
  // 2) 句首 8 字片段容错（标签打断/句中字符差异）
  const frag = target.slice(0, 8)
  if (frag) {
    idx = full.indexOf(frag, fromPos)
    if (idx < 0) idx = full.indexOf(frag)
    if (idx >= 0) {
      const block = hitBlock(idx, lens)
      if (block) return { block, nextPos: idx + frag.length }
    }
  }
  // 3) 去标点整句匹配（服务端与渲染文本引号/破折号等标点不一致）
  const starget = target.replace(TTS_STRIP_RE, '')
  if (starget) {
    idx = sfull.indexOf(starget)
    if (idx >= 0) {
      const block = hitBlock(idx, slens)
      if (block) return { block, nextPos: 0 }
    }
    // 4) 去标点句首 6 字片段
    const sfrag = starget.slice(0, 6)
    if (sfrag) {
      idx = sfull.indexOf(sfrag)
      if (idx >= 0) {
        const block = hitBlock(idx, slens)
        if (block) return { block, nextPos: 0 }
        }
        }
        }
  return null
}

// 在指定章节 iframe 中定位句子并高亮其所在段落，返回块元素（未命中返回 null）
function highlightSentenceInBlock(blockIdx, sent) {
  const iframe = iframeRefs.get(blockIdx)
  const doc = iframe?.contentDocument
  if (!doc?.body) return null
  ensureTtsStyle(doc)
  const r = findSentenceInDoc(doc, sent, ttsSearchPos)
  if (!r || !r.block) return null
  ttsSearchPos = r.nextPos
  if (lastTtsBlock && lastTtsBlock !== r.block) {
    try { lastTtsBlock.classList.remove('tts-speaking-block') } catch (e) { /* noop */ }
  }
  r.block.classList.add('tts-speaking-block')
  lastTtsBlock = r.block
  return r.block
}

// 在指定章节 iframe 中高亮所在句；未命中则搜索全部已加载章节（容错章节索引与
// 服务端/渲染文本不一致的情况），保证朗读时当前句始终被高亮
function highlightSentenceAll(blockIdx, sent) {
  const hit = highlightSentenceInBlock(blockIdx, sent)
  if (hit) return hit
  for (const [idx, iframe] of iframeRefs) {
    if (idx === blockIdx) continue
    const doc = iframe?.contentDocument
    if (!doc?.body) continue
    const r = findSentenceInDoc(doc, sent, 0)
    if (r && r.block) {
      if (lastTtsBlock && lastTtsBlock !== r.block) {
        try { lastTtsBlock.classList.remove('tts-speaking-block') } catch (e) { /* noop */ }
      }
      r.block.classList.add('tts-speaking-block')
      lastTtsBlock = r.block
      return r.block
    }
  }
  console.warn('[TTS] 未能定位句子进行高亮：', (sent || '').slice(0, 30))
  return null
}

// 清除所有章节 iframe 的朗读高亮，并重置搜索位置（换章/停止时调用）
function clearSentenceHighlight() {
  for (const [, iframe] of iframeRefs) {
    try {
      const doc = iframe.contentDocument
      doc?.querySelectorAll('.tts-speaking-block').forEach(el => el.classList.remove('tts-speaking-block'))
    } catch (e) { /* noop */ }
  }
  ttsSearchPos = 0
  lastTtsBlock = null
}

// 把当前高亮的朗读句滚动到父阅读容器视口中部（连续滚动模式滚动发生在父容器，
// iframe 内的 scrollIntoView 无法滚动父视图，故在此手动计算偏移）
function scrollHighlightedIntoView() {
  if (!rootEl.value) return
  const rootRect = rootEl.value.getBoundingClientRect()
  for (const [, iframe] of iframeRefs) {
    const doc = iframe.contentDocument
    if (!doc) continue
    const block = doc.querySelector('.tts-speaking-block')
    if (!block) continue
    const innerRect = block.getBoundingClientRect()
    const frameRect = iframe.getBoundingClientRect()
    const y = (frameRect.top - rootRect.top) + rootEl.value.scrollTop +
      innerRect.top - rootEl.value.clientHeight / 2 + innerRect.height / 2
    rootEl.value.scrollTo({ top: Math.max(0, y), behavior: 'smooth' })
    return
  }
}

// 仅重置搜索位置（跳句/换章后从头匹配，不清理 DOM 高亮）
function resetSentenceSearch() {
  ttsSearchPos = 0
}

// 在 root 中查找第 N 次出现的 text 文本，返回对应的 Range
function findTextRange(root, text, matchIndex = 0) {
  if (!text) return null
  const walker = root.ownerDocument.createTreeWalker(root, NodeFilter.SHOW_TEXT)
  let node, fullText = ''
  const offsets = [] // 每个文本节点起始位置
  while ((node = walker.nextNode())) {
    offsets.push({ node, start: fullText.length })
    fullText += node.textContent
  }
  let from = 0, found = -1, count = 0
  while (count <= matchIndex) {
    found = fullText.indexOf(text, from)
    if (found === -1) return null
    from = found + text.length
    count++
  }
  const start = found, end = found + text.length
  // 找到 start/end 所在的文本节点 + 节点内偏移
  let sNode = null, sOff = 0, eNode = null, eOff = 0
  for (let i = 0; i < offsets.length; i++) {
    const { node, start: ns } = offsets[i]
    const ne = ns + node.textContent.length
    if (!sNode && ne > start) { sNode = node; sOff = start - ns }
    if (ne >= end) { eNode = node; eOff = end - ns; break }
  }
  if (!sNode || !eNode) return null
  const doc = root.ownerDocument
  const range = doc.createRange()
  try {
    range.setStart(sNode, sOff)
    range.setEnd(eNode, eOff)
    return range
  } catch { return null }
}

// 把 Range 包裹为 <span class="fr-ann">
// 简单情况（同文本节点）：用文本切片 + DOM 片段包裹
// 跨元素：extractContents 提取后整体包裹（保留 inline 元素如 <a><em>）
function wrapRangeWithSpans(doc, range, ann) {
  const startNode = range.startContainer
  const endNode = range.endContainer
  const startOff = range.startOffset
  const endOff = range.endOffset
  // 简单情况：起止都在同一文本节点
  if (startNode === endNode && startNode.nodeType === Node.TEXT_NODE) {
    const text = startNode.textContent
    const before = text.slice(0, startOff)
    const mid = text.slice(startOff, endOff)
    const after = text.slice(endOff)
    const parent = startNode.parentNode
    const frag = doc.createDocumentFragment()
    if (before) frag.appendChild(doc.createTextNode(before))
    if (mid) frag.appendChild(makeAnnSpan(doc, ann, mid))
    if (after) frag.appendChild(doc.createTextNode(after))
    parent.replaceChild(frag, startNode)
    return
  }
  // 跨元素：用 extractContents 提取跨元素内容并整体包裹
  try {
    const frag = range.extractContents()
    const span = makeAnnSpan(doc, ann)
    span.appendChild(frag)
    range.insertNode(span)
  } catch (e) {
    console.warn('[ScrollContinuousReader] 跨元素划线包裹失败:', e)
  }
}

// 创建标注包裹 span
// 文本内容可选：单文本节点场景下传入；跨元素场景下后续 appendChild
function makeAnnSpan(doc, ann, text) {
  const span = doc.createElement('span')
  span.className = 'fr-ann'
  span.dataset.id = String(ann.id)
  span.dataset.style = ann.style || 'highlight'
  // highlight 颜色通过 CSS 变量注入；其他样式用 text-decoration-color
  if (ann.color) span.style.setProperty('--fr-ann-bg', hexToRgba(ann.color, 0.45))
  if (ann.color && ann.style !== 'highlight') span.style.textDecorationColor = ann.color
  if (ann.note) span.dataset.hasNote = '1'
  if (text != null) span.textContent = text
  return span
}

// #ffeb3b → rgba(255,235,59,0.45)
function hexToRgba(hex, a) {
  const m = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex || '')
  if (!m) return `rgba(255,235,59,${a})`
  return `rgba(${parseInt(m[1],16)},${parseInt(m[2],16)},${parseInt(m[3],16)},${a})`
}

function injectStyles(block) {
  const iframe = iframeRefs.get(block.idx)
  if (!iframe || !props.fontCss) return
  try {
    const doc = iframe.contentDocument
    if (!doc) return
    let styleEl = doc.getElementById('__cr_style')
    if (!styleEl) {
      styleEl = doc.createElement('style')
      styleEl.id = '__cr_style'
      if (doc.head) doc.head.appendChild(styleEl)
      else if (doc.documentElement) doc.documentElement.appendChild(styleEl)
    }
    styleEl.textContent = props.fontCss
  } catch (e) {
    console.warn('[ScrollContinuousReader] injectStyles failed', e)
  }
}

function measureHeight(block) {
  const iframe = iframeRefs.get(block.idx)
  if (!iframe) return
  try {
    const doc = iframe.contentDocument
    if (!doc || !doc.documentElement || !doc.body) return
    // 检测空内容：无文本且无图片等媒体，这类章节通常是 EPUB 里的空白 front matter、
    // 分隔页或转换产生的空节，应当收为 0 高度，避免在阅读器里显示成大片空白纸张。
    const text = (doc.body.innerText || '').replace(/\s/g, '')
    const hasMedia = !!doc.body.querySelector('img, svg, picture, canvas, video, audio')
    const isEmpty = text.length === 0 && !hasMedia
    if (isEmpty) {
      if (block.height !== 0 || !block.isEmpty) {
        block.height = 0
        block.isEmpty = true
        blocks.value = [...blocks.value]
        scheduleProgressEmit()
      }
      return
    }
    block.isEmpty = false
    // 用 Range 测量 body 真实内容高度，规避 html/body 被设为 height:100%（部分
    // EPUB 自带样式或 foliate 默认会让其撑满 iframe 视口）时，scrollHeight 取
    // iframe 视口高度（远大于实际内容）导致章节底部出现大片空白的问题。
    const range = doc.createRange()
    range.selectNodeContents(doc.body)
    let h = range.getBoundingClientRect().height
    // range 不含 body 自身上下 padding，补回以保证内容不被 overflow:hidden 裁切
    const cs = doc.defaultView?.getComputedStyle?.(doc.body)
    if (cs) {
      h += (parseFloat(cs.paddingTop) || 0) + (parseFloat(cs.paddingBottom) || 0)
    }
    h = Math.round(h)
    if (h > 0 && Math.abs(block.height - h) > 4) {
      block.height = h
      blocks.value = [...blocks.value]
      scheduleProgressEmit()
    }
  } catch (e) { /* noop */ }
}

function setupResizeObserver() {
  if (typeof window === 'undefined' || !window.ResizeObserver) return
  resizeObserver = new ResizeObserver(() => {
    // 内容尺寸变化时重新测量对应 block（用 document.documentElement 高度变化触发）
    for (const [idx, el] of iframeRefs) {
      if (el && el.contentDocument) {
        const block = blocks.value[idx]
        if (block) measureHeight(block)
      }
    }
  })
}

function onScroll() {
  userScrolled = true
  scheduleProgressEmit()
  triggerPreload()
}

function onTouchStart() { /* 允许原生滚动 */ }
function onTouchMove() { /* 允许原生滚动 */ }

function scheduleProgressEmit() {
  const now = Date.now()
  if (now - lastProgressEmit < 100) return
  lastProgressEmit = now
  const p = computeProgress()
  if (p) emit('progress', p)
}

function computeProgress() {
  if (!rootEl.value) return null
  const scrollTop = rootEl.value.scrollTop
  const clientHeight = rootEl.value.clientHeight
  let totalHeight = 0
  for (const b of blocks.value) totalHeight += (b.height || 0)
  if (totalHeight === 0) {
    return { index: props.initialIndex || 0, fraction: 0, totalFraction: 0 }
  }
  const scrollable = totalHeight - clientHeight
  let cumY = 0
  let currentIdx = 0
  let blockY = 0
  for (let i = 0; i < blocks.value.length; i++) {
    const h = blocks.value[i].height || 0
    if (cumY + h > scrollTop || i === blocks.value.length - 1) {
      currentIdx = i
      blockY = cumY
      break
    }
    cumY += h
  }
  const blockH = blocks.value[currentIdx]?.height || 0
  const fraction = blockH > 0
    ? Math.max(0, Math.min(1, (scrollTop - blockY) / blockH))
    : 0
  const totalFraction = scrollable > 0
    ? Math.max(0, Math.min(1, scrollTop / scrollable))
    : 0
  return { index: currentIdx, fraction, totalFraction, scrollTop, totalHeight }
}

// 预加载节流：避免 onScroll 高频触发时反复扫描 blocks
let preloadThrottle = 0
let userScrolled = false    // 用户是否发生过滚动；用于禁止"渐进式校正"在用户阅读中把视口拽回进度处
// 按需预加载：进入只加载当前进度附近，下滑到接近已加载内容底部 / 上滑到接近顶部时，
// 按"下一页"方式预载前后各 PRELOAD_AHEAD 页，而不是一次性把整本书铺开。
function triggerPreload() {
  if (!rootEl.value) return
  const now = Date.now()
  if (now - preloadThrottle < 150) return
  preloadThrottle = now

  const root = rootEl.value
  const scrollTop = root.scrollTop
  const clientHeight = root.clientHeight
  const est = estimatedBlockHeight()
  // 当前所在章节（用估算高度累加，与 scrollToInitial 口径一致）
  const cur = currentIndexFromScroll(scrollTop, clientHeight, est)

  const pending = []
  // 确保当前视口所在章节已加载：快速下滑可能越过已加载区，使 cur 本身落在未加载章节上，
  // 此时必须从 cur 开始预载，否则会"只预载 cur 之后、永远跳过正看着的这页"导致一直转圈。
  if (blocks.value[cur] && !blocks.value[cur].loaded) pending.push(cur)

  // 向后：从 cur 之后第一个未加载章节起算"已加载内容底部"，
  // 当距底部不足 PRELOAD_TRIGGER 屏时，预载后 PRELOAD_AHEAD 页
  let fwdFrom = cur + 1
  while (fwdFrom < blocks.value.length && blocks.value[fwdFrom].loaded) fwdFrom++
  const loadedBottomY = blockTop(fwdFrom)   // 第一个未加载章节顶部 = 已加载内容底部
  if (loadedBottomY - (scrollTop + clientHeight) < clientHeight * PRELOAD_TRIGGER) {
    let cnt = 0
    for (let i = fwdFrom; i < blocks.value.length && cnt < PRELOAD_AHEAD; i++) {
      if (!blocks.value[i].loaded) { pending.push(i); cnt++ }
    }
  }
  // 向上：距顶部不足 PRELOAD_TRIGGER 屏时，预载前 PRELOAD_AHEAD 页（回看）
  if (scrollTop < clientHeight * PRELOAD_TRIGGER) {
    let cnt = 0
    for (let i = cur - 1; i >= 0 && cnt < PRELOAD_AHEAD; i--) {
      if (!blocks.value[i].loaded) { pending.push(i); cnt++ }
    }
  }

  if (!pending.length) return
  pending.forEach(i => loadBlock(i))
  // 加载前记录滚动锚点（视口顶对应章节 + 章节内偏移），加载后用真实高度重新锚定，
  // 保持当前阅读位置不跳动：仅纠正"上方章节加载完导致整体下移"，绝不跳回开头。
  const anchorIdx = cur
  const anchorOffset = Math.max(0, scrollTop - blockTop(cur))
  const topAtStart = scrollTop
  waitForBlocksLoaded(pending).then(() => {
    if (!rootEl.value) return
    // 加载期间用户若发生了滚动，则不纠正，避免与用户操作冲突
    if (rootEl.value.scrollTop !== topAtStart) return
    const newTop = blockTop(anchorIdx) + anchorOffset
    rootEl.value.scrollTo({ top: newTop, behavior: 'auto' })
  }).catch(() => {})
}

// 根据滚动位置推断当前所在章节 index（与 scrollToInitial 的高度累加口径一致）
function currentIndexFromScroll(scrollTop, clientHeight, est) {
  let cumY = 0
  for (let i = 0; i < blocks.value.length; i++) {
    const h = blocks.value[i].height || est
    if (cumY + h > scrollTop || i === blocks.value.length - 1) return i
    cumY += h
  }
  return 0
}

// 计算第 idx 个章节块顶部的累计偏移（未加载的用估算高度），用于加载后保持滚动锚点
function blockTop(idx) {
  const est = estimatedBlockHeight()
  let y = 0
  for (let i = 0; i < idx && i < blocks.value.length; i++) {
    y += blocks.value[i].height || est
  }
  return y
}

// 等待一组章节全部加载完成（每个最多 4s 兜底超时）
async function waitForBlocksLoaded(ids) {
  const start = Date.now()
  const TIMEOUT = 4000
  const POLL = 120
  while (Date.now() - start < TIMEOUT) {
    if (ids.every(i => blocks.value[i]?.loaded)) return
    await new Promise(r => setTimeout(r, POLL))
  }
}

defineExpose({
  scrollToSection,
  scrollToFraction,
  getProgress: computeProgress,
  // 获取已加载章节的纯文本长度（用于阅读时长估算）。未加载返回 0。
  getSectionTextLength(idx) {
    const iframe = iframeRefs.get(idx)
    if (!iframe) return 0
    try {
      const doc = iframe.contentDocument
      if (!doc || !doc.body) return 0
      // 取 body 文本，去掉空白
      return (doc.body.innerText || '').replace(/\s+/g, '').length
    } catch (e) { return 0 }
  },
  // 与标注 findTextRange 同源：遍历正文文本节点拼接 textContent（不用 innerText，
  // 避免 text-transform / 不可见元素 / 换行差异导致和高亮搜索的文本不一致）
  getSectionText(idx) {
    const iframe = iframeRefs.get(idx)
    if (!iframe) return ''
    try {
      const doc = iframe.contentDocument
      if (!doc?.body) return ''
      const walker = doc.createTreeWalker(doc.body, NodeFilter.SHOW_TEXT)
      let node, out = ''
      while ((node = walker.nextNode())) out += node.textContent
      return out
    } catch (e) { return '' }
  },
  // 章节是否已加载完成（语音朗读切章时等待用）
  isSectionLoaded(idx) {
    return !!blocks.value[idx]?.loaded
  },
  // 直接加载指定章节（绕过滚动窗口的懒加载），供语音朗读取句前确保该区块 iframe 已就绪
  async loadSection(idx) {
    await loadBlock(idx)
  },
  // 跨章节全文搜索：在每个已加载 iframe 文档里查找 query，返回匹配项位置 + 章节 idx + 上下文
  // 返回数组 [{ idx, offset, snippet }]，offset 是文本偏移（用于后续 highlight）
  async searchInAllSections(query, maxResults = 100) {
    if (!query || !query.trim()) return []
    const q = query.trim().toLowerCase()
    const results = []
    for (const block of blocks.value) {
      if (!block.loaded) continue
      if (results.length >= maxResults) break
      const iframe = iframeRefs.get(block.idx)
      if (!iframe) continue
      try {
        const doc = iframe.contentDocument
        if (!doc || !doc.body) continue
        const text = (doc.body.innerText || '').toLowerCase()
        let from = 0
        while (true) {
          const idx = text.indexOf(q, from)
          if (idx === -1) break
          // 提取上下文（前后 30 字符），用原始文本（非 lowercased）保持可读
          const origText = doc.body.innerText || ''
          const start = Math.max(0, idx - 30)
          const end = Math.min(origText.length, idx + q.length + 30)
          const snippet = (start > 0 ? '...' : '') + origText.slice(start, end) + (end < origText.length ? '...' : '')
          results.push({ idx: block.idx, offset: idx, snippet })
          if (results.length >= maxResults) break
          from = idx + q.length
        }
      } catch (e) { /* skip */ }
    }
    return results
  },
  // 滚动模式下"翻一屏"：向上/向下滚动一个视口高度（带小幅重叠，更像翻页）。
  // 自动跳过未加载章节（避免落到"加载中"占位上）。
  pageFlip(direction) {
    if (!rootEl.value) return
    const step = rootEl.value.clientHeight * 0.92
    const target = direction > 0
      ? rootEl.value.scrollTop + step
      : rootEl.value.scrollTop - step
    rootEl.value.scrollTo({ top: Math.max(0, target), behavior: 'smooth' })
  },
  // 应用一条 annotation（已存于后端）：包裹对应章节 iframe 中的文字
  applyAnnotation(ann) {
    return applyAnnotationToBlock(ann)
  },
  // 移除一条 annotation 的 DOM 包裹
  removeAnnotation(annId) {
    for (const [idx, iframe] of iframeRefs) {
      try {
        const doc = iframe.contentDocument
        if (!doc) continue
        const span = doc.querySelector(`.fr-ann[data-id="${annId}"]`)
        if (!span) continue
        const parent = span.parentNode
        while (span.firstChild) parent.insertBefore(span.firstChild, span)
        parent.removeChild(span)
        parent.normalize && parent.normalize()
      } catch (e) { /* noop */ }
    }
  },
  // 滚动到指定 annotation 所在的章节，并短暂高亮
  scrollToAnnotation(ann) {
    const pos = parseAnnPosition(ann)
    if (!pos || typeof pos.index !== 'number') return
    const idx = pos.index
    // 若章节未加载，先滚动到目标位置触发预加载，然后等加载完再聚焦
    if (!blocks.value[idx]?.loaded) {
      scrollToSection(idx)
      // 简单等待（无更多信号就用轮询）
      let tries = 0
      const timer = setInterval(() => {
        tries++
        if (blocks.value[idx]?.loaded || tries > 30) {
          clearInterval(timer)
          focusAnnInIframe(idx, ann.id)
        }
      }, 200)
    } else {
      scrollToSection(idx)
      nextTick(() => focusAnnInIframe(idx, ann.id))
    }
  },
  // 清除所有 iframe 的当前选区
  clearSelection() {
    for (const [, iframe] of iframeRefs) {
      try {
        const doc = iframe.contentDocument
        doc?.getSelection?.()?.removeAllRanges?.()
      } catch (e) { /* noop */ }
    }
  },
  // 语音朗读：在指定章节 iframe 中高亮当前句所在段落，未命中则跨块容错（调用方负责滚动）
  highlightSentence(blockIdx, sent) {
    return highlightSentenceAll(blockIdx, sent)
  },
  // 语音朗读：清除全部章节的朗读高亮并重置匹配位置
  clearSentenceHighlight,
  // 语音朗读：把当前高亮句滚动到父容器视口中部（跟随朗读）
  scrollHighlightedIntoView,
  // 语音朗读：仅重置匹配位置（换章/跳句后从头找）
  resetSentenceSearch,
  // 设置章节 iframe 的光标（划线模式传笔形光标，退出传空串还原）
  setCursor(cursor) {
    drawCursor = cursor || ''
    for (const block of blocks.value) {
      if (block.loaded) setIframeCursor(block, drawCursor)
    }
  },
  refreshHeights() {
    for (const [, el] of iframeRefs) {
      // 由 caller 触发：遍历所有 iframe 重新测量
    }
    for (const block of blocks.value) {
      if (block.loaded) measureHeight(block)
    }
  },
})

// 在指定章节块中临时高亮某条 annotation（用于从抽屉跳转时）
function focusAnnInIframe(idx, annId) {
  const iframe = iframeRefs.get(idx)
  if (!iframe) return
  try {
    const doc = iframe.contentDocument
    if (!doc) return
    // 先取消之前的高亮
    for (const el of doc.querySelectorAll('.fr-ann.active')) el.classList.remove('active')
    const span = doc.querySelector(`.fr-ann[data-id="${annId}"]`)
    if (!span) return
    span.classList.add('active')
    span.scrollIntoView({ block: 'center', behavior: 'smooth' })
    setTimeout(() => span.classList.remove('active'), 2500)
  } catch (e) { /* noop */ }
}

function scrollToSection(idx, opts = {}) {
  if (!rootEl.value) return
  const y = computeSectionY(idx)
  if (opts.behavior !== undefined) {
    rootEl.value.scrollTo({ top: y, behavior: opts.behavior })
  } else {
    rootEl.value.scrollTo({ top: y, behavior: 'smooth' })
  }
  // 滚动到搜索结果位置（基于章节内 text 偏移）
  if (typeof opts.textOffset === 'number') {
    nextTick(() => {
      const block = blocks.value[idx]
      if (!block || !block.loaded) return
      const iframe = iframeRefs.get(idx)
      if (!iframe) return
      try {
        const doc = iframe.contentDocument
        if (!doc || !doc.body) return
        const totalText = (doc.body.innerText || '').replace(/\s+/g, '').length
        const frac = totalText > 0 ? Math.min(1, opts.textOffset / totalText) : 0
        // 在当前章节 iframe 内部做次级滚动
        iframe.contentWindow?.scrollTo({ top: iframe.contentDocument.documentElement.scrollHeight * frac })
      } catch {}
    })
  }
}

function scrollToFraction(fraction) {
  if (!rootEl.value) return
  let totalHeight = 0
  for (const b of blocks.value) totalHeight += (b.height || 0)
  const scrollable = totalHeight - rootEl.value.clientHeight
  rootEl.value.scrollTo({ top: Math.max(0, scrollable * fraction), behavior: 'smooth' })
}

function computeSectionY(idx) {
  let y = 0
  for (let i = 0; i < idx && i < blocks.value.length; i++) {
    y += blocks.value[i].height || 0
  }
  return y
}

// 字体 CSS 变化时重新注入到已加载的 iframe
watch(() => props.fontCss, () => {
  for (const block of blocks.value) {
    if (block.loaded) injectStyles(block)
  }
})

// 外部 annotations 变化时：把当前所有标注应用到 DOM，并清理已从数组中移除的标注。
// 注意：Vue 3 的 watch(..., {deep:true}) 中 oldValue === newValue（同一引用），
// 不能用 prev 推导新增项——直接遍历 next 由 applyAnnotationToBlock 内部的
// "已包裹则跳过"检查保证幂等；用 appliedAnnIds 跟踪"已应用的标注集合"以做差集清理。
const appliedAnnIds = new Set()

function removeAnnFromDom(annId) {
  for (const [, iframe] of iframeRefs) {
    try {
      const doc = iframe.contentDocument
      const span = doc?.querySelector(`.fr-ann[data-id="${annId}"]`)
      if (!span) continue
      const parent = span.parentNode
      while (span.firstChild) parent.insertBefore(span.firstChild, span)
      parent.removeChild(span)
      parent.normalize && parent.normalize()
    } catch (e) { /* noop */ }
  }
}

watch(() => props.annotations, (next) => {
  if (!next) return
  const nextIds = new Set()
  for (const ann of next) {
    nextIds.add(ann.id)
    applyAnnotationToBlock(ann)
  }
  // 清理被删除的标注
  for (const id of appliedAnnIds) {
    if (!nextIds.has(id)) removeAnnFromDom(id)
  }
  appliedAnnIds.clear()
  for (const id of nextIds) appliedAnnIds.add(id)
}, { deep: true })
</script>

<style scoped>
.scr-root {
  width: 100%;
  height: 100%;
  overflow-y: auto;
  overflow-x: hidden;
  transition: opacity .25s ease;  /* 揭幕淡入，消除进入时定位跳跃的视觉突兀 */
  /* 不开 scroll-behavior:smooth：长距离跳转会触发长动画，
     "看起来快速滚动"。pageFlip 等小范围滚动由调用方显式指定 behavior:'smooth' */
  -webkit-overflow-scrolling: touch;
  /* 页面背景（"桌面"），由父组件用 --cr-bg 注入主题色 */
  background: var(--cr-bg, #f4f4f5);
  /* 上下留白交给 .scr-column 的 margin 控制，便于短书纸张铺满整屏 */
  padding-top: 0;
  padding-bottom: 0;
  /* 避免 foliate-view 残留影响 */
  position: relative;
}

/* 所有章节包进同一张连续纸张：整本书像一张长纸，章节间不再以独立卡片/间隙"断开" */
.scr-column {
  max-width: 820px;          /* 宽屏不撑满，可读性更好 */
  margin: 16px auto 32px;    /* 居中于桌面，首尾与屏幕边缘留白 */
  position: relative;
  display: block;
  /* 纸张效果：纸张色背景 + 柔和阴影 + 微圆角（主题色由 --cr-paper 注入） */
  background: var(--cr-paper, #fdfaf3);
  border-radius: 6px;
  box-shadow:
    0 1px 2px rgba(0, 0, 0, 0.06),
    0 4px 12px rgba(0, 0, 0, 0.08);
  padding: 28px 40px 60px;   /* 纸张内边距，各章节在纸上连续排布 */
  /* 短书也能铺满一屏高度（减去上下 margin 共 48px） */
  min-height: calc(100% - 48px);
}

.scr-block {
  width: 100%;
  position: relative;
  display: block;
  overflow: hidden;
}
/* 空内容章节（无文本、无图片等媒体）直接隐藏，避免显示成空白纸张 */
.scr-block.scr-empty {
  height: 0 !important;
  min-height: 0 !important;
  margin-top: 0 !important;
  margin-bottom: 0 !important;
  padding: 0 !important;
  box-shadow: none !important;
  overflow: hidden;
}

.scr-iframe {
  display: block;
  width: 100%;
  height: 100%;       /* 填满 .scr-block（block 高度已由估算/实测决定） */
  border: 0;
  margin: 0;
  padding: 0;
  background: transparent;
}

.scr-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;   /* 关键：填满纸张块高度，否则小圈只会贴在顶部 */
  /* 透明：露出连续纸张底色，加载中仅一个小圆点提示，不出现"断开的空白卡片" */
  background: transparent;
  opacity: 1;
}
.scr-placeholder-dot {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  border: 2.5px solid color-mix(in srgb, var(--cr-paper, #fdfaf3) 25%, #cbd5e1);
  border-top-color: var(--app-primary, #6366f1);
  animation: scr-spin 0.8s linear infinite;
}
.scr-reload {
  font-size: 13px;
  color: #ef4444;
  background: transparent;
  border: 1px solid color-mix(in srgb, #ef4444 45%, transparent);
  border-radius: 8px;
  padding: 6px 12px;
  cursor: pointer;
}
.scr-reload:hover { background: color-mix(in srgb, #ef4444 8%, transparent); }
@keyframes scr-spin {
  to { transform: rotate(360deg); }
}

/* 窄屏：纸张铺满、去掉圆角与阴影，章节在屏幕内连续流动，无"卡片断开"感 */
@media (max-width: 640px) {
  .scr-column {
    max-width: 100%;
    margin: 0;
    border-radius: 0;
    box-shadow: none;
    padding: 16px 18px 40px;
    min-height: 100%;
  }
}
</style>