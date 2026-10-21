// 全局有声书播放器核心（单例）
// - 音频元素全局唯一，切换页面/路由不中断播放（音乐播放器模式）
// - 每本书独立播放记录（后端 + localStorage），再次打开自动续播
import { reactive, computed } from 'vue'
import { audiobookApi } from '@/api/audiobook'
import { bookApi } from '@/api/book'
import { ElMessage } from 'element-plus'

// ===== 播放状态（全局单例） =====
export const playerState = reactive({
  task: null,          // 当前播放任务
  segments: [],        // 分段列表
  index: -1,           // 当前段索引
  playing: false,      // 是否播放中
  waitingNext: false,  // 边生成边播放：等待续播
  loading: false,      // 分段加载中
  currentTime: 0,      // 当前播放位置（秒）
  duration: 0,         // 当前段时长（秒）
  panelVisible: false, // 播放详情弹层
})

// 本地播放记录（跨页面刷新兜底）
const PROGRESS_KEY = 'audiobook_progress'

let audio = null
let pollTimer = null
let lastSaveAt = 0

const isGenerating = () => playerState.task &&
  (playerState.task.status === 'processing' || playerState.task.status === 'pending')

export const doneSegCount = computed(() => playerState.segments.filter((s) => s.done).length)

export const currentSegTitle = computed(() => {
  const seg = playerState.segments[playerState.index]
  return seg ? seg.title || '第 ' + seg.index + ' 段' : ''
})

export const progressWidth = computed(() =>
  playerState.duration ? (playerState.currentTime / playerState.duration) * 100 : 0
)

export const currentTimeStr = computed(() => formatDuration(playerState.currentTime))
export const durationStr = computed(() => formatDuration(playerState.duration))

export const formatDuration = (sec) => {
  if (!sec || sec <= 0) return '--:--'
  const m = Math.floor(sec / 60)
  const s = Math.floor(sec % 60)
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
}

// ===== 音频元素（全局单例，首次 import 即创建，不播放不触发自动播放限制） =====
function ensureAudio() {
  if (audio) return
  audio = new Audio()
  audio.preload = 'auto'
  audio.addEventListener('timeupdate', onTimeUpdate)
  audio.addEventListener('ended', onEnded)
  audio.addEventListener('error', onError)
  audio.addEventListener('play', () => {
    playerState.playing = true
    updateMediaSession()
  })
  audio.addEventListener('pause', () => {
    playerState.playing = false
    updateMediaSession()
  })
  audio.addEventListener('loadedmetadata', () => {
    playerState.duration = audio.duration || 0
  })
}
ensureAudio()

// ===== 播放记录 =====
function loadProgressMap() {
  try { return JSON.parse(localStorage.getItem(PROGRESS_KEY) || '{}') } catch { return {} }
}

function saveLocalProgress() {
  if (!playerState.task || playerState.index < 0) return
  const map = loadProgressMap()
  map[playerState.task.id] = {
    seg: playerState.index,
    pos: playerState.currentTime,
    at: Date.now(),
  }
  try { localStorage.setItem(PROGRESS_KEY, JSON.stringify(map)) } catch {}
}

function saveServerProgress() {
  if (!playerState.task || playerState.index < 0) return
  audiobookApi.saveProgress(
    playerState.task.id,
    playerState.index,
    Math.round(playerState.currentTime)
  ).catch(() => {})
}

export function saveProgress() {
  saveLocalProgress()
  saveServerProgress()
}

// ===== 播放控制 =====
function setSrcAndPlay(idx, seekTime) {
  const seg = playerState.segments[idx]
  if (!seg || !seg.done) return false
  playerState.waitingNext = false
  playerState.index = idx
  const src = audiobookApi.audioUrl(playerState.task.id, seg.file)
  if (audio.getAttribute('src') !== src) {
    audio.removeAttribute('src')
    audio.src = src
    audio.currentTime = 0
    const seekTo = seekTime || 0
    if (seekTo > 0) {
      audio.addEventListener('loadedmetadata', function once() {
        audio.removeEventListener('loadedmetadata', once)
        try { audio.currentTime = seekTo } catch {}
        audio.play().catch(() => {})
      })
    } else {
      audio.play().catch(() => {})
    }
  } else {
    try { audio.currentTime = seekTime || 0 } catch {}
    audio.play().catch(() => {})
  }
  updateMediaSession()
  return true
}

// 打开播放器（从书籍/任务入口进入，自动恢复上次播放位置）
// showPanel 为 false 时只显示迷你播放条，不展开播放列表面板
export async function openPlay(task, showPanel = true) {
  playerState.task = task
  playerState.panelVisible = !!showPanel
  playerState.index = -1
  playerState.segments = []
  playerState.waitingNext = false
  playerState.currentTime = 0
  playerState.duration = 0
  updateMediaSession()
  await loadSegments()
  resumeFromRecord()
  startPolling()
}

// 关闭详情弹层（后台继续播放）
export function closePanel() {
  playerState.panelVisible = false
  saveProgress()
}

// 恢复上次播放位置：优先后端记录，其次本地记录；段未生成则跳到下一已生成段
function resumeFromRecord() {
  const segs = playerState.segments
  if (!segs.length) return
  let rec = null
  if (playerState.task.last_played_at && playerState.task.last_segment > 0) {
    rec = { seg: playerState.task.last_segment, pos: playerState.task.last_position || 0 }
  } else {
    const local = loadProgressMap()[playerState.task.id]
    if (local && local.seg > 0) rec = { seg: local.seg, pos: local.pos || 0 }
  }
  if (rec) {
    let idx = rec.seg
    if (!segs[idx] || !segs[idx].done) {
      idx = segs.findIndex((s, i) => i >= rec.seg && s.done)
    }
    if (idx < 0) idx = segs.findIndex((s) => s.done)
    if (idx >= 0) {
      setSrcAndPlay(idx, rec.pos || 0)
      return
    }
  }
  const first = segs.findIndex((s) => s.done)
  if (first >= 0) {
    playerState.index = first
    nextTickPlay()
  }
}

async function loadSegments() {
  if (!playerState.task) return
  playerState.loading = true
  try {
    const res = await audiobookApi.get(playerState.task.id)
    playerState.task = res.data.data.task
    playerState.segments = res.data.data.segments || []
    autoJumpToDone()
  } catch {
    ElMessage.error('加载分段失败')
  } finally {
    playerState.loading = false
  }
}

// 边生成边播放：保证当前播放的段已生成；等待续播时新段生成自动续播
function autoJumpToDone() {
  const segs = playerState.segments
  if (!segs.length || playerState.index < 0) return
  if (segs[playerState.index].done) {
    if (playerState.waitingNext) {
      const next = segs.findIndex((s, i) => i > playerState.index && s.done)
      if (next >= 0) {
        playerState.waitingNext = false
        playerState.index = next
        nextTickPlay()
      }
    }
    return
  }
  const next = segs.findIndex((s, i) => i >= playerState.index && s.done)
  if (next >= 0) {
    playerState.index = next
    nextTickPlay()
  }
}

function nextTickPlay() {
  requestAnimationFrame(() => {
    if (audio) audio.play().catch(() => {})
  })
}

export function playSegment(idx) {
  const seg = playerState.segments[idx]
  if (!seg || !seg.done) {
    ElMessage.info(isGenerating() ? '该段正在生成中，请稍候' : '该段尚未生成，无法播放')
    return
  }
  // 点击当前正在播放的段 → 暂停 / 继续
  if (idx === playerState.index && !audio.paused) {
    audio.pause()
    return
  }
  setSrcAndPlay(idx, 0)
  saveProgress()
}

export function togglePlay() {
  if (!playerState.task || !audio) return
  if (playerState.index < 0) {
    const first = playerState.segments.findIndex((s) => s.done)
    if (first >= 0) setSrcAndPlay(first, 0)
    return
  }
  if (audio.paused) audio.play().catch(() => {})
  else audio.pause()
}

// 暂停播放（进入阅读器页等场景自动调用）：保留任务与进度，仅暂停并收起弹层
export function pausePlay() {
  if (audio && !audio.paused) {
    audio.pause()
    saveProgress()
  }
  playerState.panelVisible = false
}

export function playPrev() {
  const segs = playerState.segments
  for (let i = playerState.index - 1; i >= 0; i--) {
    if (segs[i].done) {
      setSrcAndPlay(i, 0)
      saveProgress()
      return
    }
  }
}

export function playNext() {
  const segs = playerState.segments
  for (let i = playerState.index + 1; i < segs.length; i++) {
    if (segs[i].done) {
      setSrcAndPlay(i, 0)
      saveProgress()
      return
    }
  }
  // 后面暂无已生成段：生成中则等待自动续播
  if (isGenerating()) {
    playerState.waitingNext = true
    ElMessage.info('后续段落正在生成，生成后将自动续播')
  }
}

function onEnded() {
  saveProgress()
  const segs = playerState.segments
  for (let i = playerState.index + 1; i < segs.length; i++) {
    if (segs[i].done) {
      setSrcAndPlay(i, 0)
      return
    }
  }
  if (isGenerating()) {
    playerState.playing = false
    playerState.waitingNext = true
    ElMessage.info('当前已生成段已播完，下一段生成后将自动续播')
  } else {
    playerState.playing = false
    playerState.currentTime = 0
    playerState.duration = 0
  }
  updateMediaSession()
}

function onError() {
  playerState.playing = false
  updateMediaSession()
  ElMessage.warning('音频播放失败，可能尚未生成完成')
}

function onTimeUpdate() {
  if (!audio) return
  playerState.currentTime = audio.currentTime
  playerState.duration = audio.duration || playerState.duration
  // 节流保存播放进度
  const now = Date.now()
  if (now - lastSaveAt > 8000) {
    lastSaveAt = now
    saveProgress()
  }
}

// 点击进度条定位
export function seekByClick(e, el) {
  if (!audio || !playerState.duration) return
  const rect = el.getBoundingClientRect()
  const ratio = Math.min(1, Math.max(0, (e.clientX - rect.left) / rect.width))
  try { audio.currentTime = ratio * playerState.duration } catch {}
  playerState.currentTime = audio.currentTime
}

// 轮询分段（边生成边播放）
function startPolling() {
  if (pollTimer) return
  pollTimer = setInterval(() => {
    if (playerState.task && isGenerating()) loadSegments()
    else stopPolling()
  }, 5000)
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

// 停止并清空播放器（迷你条隐藏）
export function stopPlayer() {
  if (audio) {
    audio.pause()
    audio.removeAttribute('src')
    audio.load()
  }
  saveProgress()
  stopPolling()
  playerState.playing = false
  playerState.waitingNext = false
  playerState.task = null
  playerState.segments = []
  playerState.index = -1
  playerState.currentTime = 0
  playerState.duration = 0
  playerState.panelVisible = false
  clearMediaSession()
}

// ===== 后台播放：Media Session（锁屏 / 系统媒体面板控制） =====
// - 支持系统媒体键、锁屏通知栏控制播放/暂停/上一段/下一段/拖动
// - Media Session 激活后，浏览器不会节能冻结后台标签页，保证切后台后音频持续播放
const mediaSession = typeof navigator !== 'undefined' && 'mediaSession' in navigator ? navigator.mediaSession : null

// 封面：优先任务自身封面（接收的分享任务无原书），否则书籍封面（带鉴权 token），无封面时生成渐变 SVG 占位图
function mediaArtwork() {
  if (!playerState.task) return []
  const t = playerState.task
  // 列表/详情接口已兜底填充 cover_path（含接收的分享任务）
  const url = t.cover_path
    ? audiobookApi.coverUrl(t.id)
    : (t.book_id ? bookApi.coverUrl(t.book_id) : '')
  if (url) {
    return [
      { src: url, sizes: '96x96', type: 'image/jpeg' },
      { src: url, sizes: '256x256', type: 'image/jpeg' },
      { src: url, sizes: '512x512', type: 'image/jpeg' },
    ]
  }
  const svg = `data:image/svg+xml;charset=utf-8,${encodeURIComponent(
    '<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512"><defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#6366f1"/><stop offset="1" stop-color="#8b5cf6"/></linearGradient></defs><rect width="512" height="512" fill="url(#g)"/><text x="256" y="310" font-size="220" font-family="sans-serif" font-weight="bold" fill="#fff" text-anchor="middle">声</text></svg>'
  )}`
  return [
    { src: svg, sizes: '96x96' },
    { src: svg, sizes: '256x256' },
    { src: svg, sizes: '512x512' },
  ]
}

// 同步锁屏/系统面板：元数据 + 播放状态 + 控制动作
export function updateMediaSession() {
  if (!mediaSession) return
  const task = playerState.task
  try {
    if (!task) {
      clearMediaSession()
      return
    }
    const seg = playerState.segments[playerState.index]
    mediaSession.metadata = new MediaMetadata({
      title: task.book_title || '有声书',
      artist: seg ? seg.title || '第 ' + seg.index + ' 段' : '有声书',
      album: '我的书库 · 有声书',
      artwork: mediaArtwork(),
    })
    mediaSession.setActionHandler('play', () => { if (audio) audio.play().catch(() => {}) })
    mediaSession.setActionHandler('pause', () => { if (audio) audio.pause() })
    mediaSession.setActionHandler('previoustrack', () => playPrev())
    mediaSession.setActionHandler('nexttrack', () => playNext())
    mediaSession.setActionHandler('seekto', (d) => {
      if (audio && d && typeof d.seekTime === 'number') {
        try { audio.currentTime = d.seekTime } catch {}
      }
    })
    mediaSession.playbackState = playerState.playing ? 'playing' : 'paused'
  } catch {}
}

// 清除锁屏/系统面板（播放器关闭时）
export function clearMediaSession() {
  if (!mediaSession) return
  try {
    mediaSession.playbackState = 'none'
    mediaSession.metadata = null
    mediaSession.setActionHandler('play', null)
    mediaSession.setActionHandler('pause', null)
    mediaSession.setActionHandler('previoustrack', null)
    mediaSession.setActionHandler('nexttrack', null)
    mediaSession.setActionHandler('seekto', null)
  } catch {}
}
