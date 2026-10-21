// ===== 语音朗读引擎（Edge TTS，Go 后端直连）=====
// 从 PdfReader 的成熟实现抽取的阅读器无关组合式函数，供连续滚动阅读器等复用：
//   - WebSocket 合成（连接复用 + 并发合成），失败自动回退 HTTP
//   - 逐句播放 / 预取缓存 / 暂停继续 / 上句下句 / 停止
//   - 移动端自动播放策略拦截后，等待用户手势自动恢复
// 文本来源、句子高亮、章节切换由调用方通过回调注入（onSentence / onEnd）。
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { bookApi } from '@/api/book'
import { getToken } from '@/api/auth'
import { wsApiBase } from '@/config'
import { loadReaderPrefs, saveReaderPrefs } from './readerPrefs'

// 文本切分为朗读句子（中英标点分段，过长再按逗号截断）
export const splitSentences = (text) => {
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

// 归一化文本（忽略全部空白），用于跨 span/节点的朗读文本匹配
export const normTtsText = (s) => (s || '').replace(/\s+/g, '')

// 判断 play() 是否因移动端自动播放策略被拒绝（而非真正的合成/解码失败）
const isAutoplayBlocked = (e) => {
  if (!e) return false
  if (e.name === 'NotAllowedError') return true
  return /notallowed|autoplay|interact with the document/i.test(e.message || '')
}

const TTS_PREFETCH_WINDOW = 6 // 当前句 + 后 5 句

export function useTtsPlayer({ onSentence, onEnd } = {}) {
  // ===== 状态 =====
  const active = ref(false)      // 朗读会话进行中
  const playing = ref(false)     // 正在播放（非暂停）
  const paused = ref(false)      // 暂停 / 等待手势解锁
  const loading = ref(false)     // 当前句合成中
  const index = ref(-1)          // 当前句下标
  const sentences = ref([])      // 当前章节的句子列表
  const currentText = ref('')    // 当前句文本
  const speed = ref(1.0)
  const emotion = ref('平静')
  const emotions = ref(['平静'])
  const voices = ref([])
  const voice = ref('zh-CN-XiaoxiaoNeural') // Edge TTS 音色 short name
  const engines = ref([
    { key: 'edge-tts', label: 'Edge TTS（在线，免费）', desc: '微软在线语音，无需本地模型，需联网。情感/语速为近似模拟。', available: true, default: true },
    { key: 'chattts', label: 'ChatTTS（本地模型）', desc: '本地 ChatTTS 模型合成，支持参考音色、固定音色种子与语速控制。需安装 ChatTTS 并下载模型。', available: true, default: false }
  ])
  const config = ref({
    engine: 'edge-tts',
    emotion_enabled: true,
    speed_factor: 1.0
  })

  // 浏览器是否支持原生 SpeechSynthesis（离线朗读）
  const browserSupported = typeof window !== 'undefined' && 'speechSynthesis' in window
  // 朗读引擎默认策略：浏览器支持则优先「浏览器离线」（避免受限网络下微软 403），
  // 否则回退云端（WS / HTTP）。本地已保存的偏好（ttsEngine）始终优先于默认值。
  const resolveDefaultEngine = () => {
    const saved = loadReaderPrefs().ttsEngine
    if (saved) return saved
    return browserSupported ? 'browser' : 'edge-tts'
  }
  // 朗读引擎：'edge-tts' / 'gpt-sovits' / 'fish-speech' / 'chattts'（云端，走后端）
  // 或 'browser'（浏览器原生 SpeechSynthesis，离线、不依赖微软端点）
  const engine = ref(resolveDefaultEngine())
  // 浏览器引擎可用音色与选中项
  const browserVoices = ref([])
  const browserVoiceURI = ref('')

  // ===== 内部 =====
  let audioEl = null
  let objectUrl = null
  let pendingPlay = null
  let unlockBound = false
  let failCount = 0
  const cache = new Map()
  // 浏览器引擎当前 utterance
  let browserUtter = null

  // 加载浏览器可用音色（异步 voiceschanged）。优先中文音色
  const loadBrowserVoices = () => {
    if (!browserSupported) return
    const pick = () => {
      const vs = window.speechSynthesis.getVoices() || []
      browserVoices.value = vs
      // 优先使用本地保存的本机音色，否则回退到首个中文音色
      const saved = loadReaderPrefs().ttsBrowserVoice
      if (saved && vs.some(v => v.voiceURI === saved)) {
        browserVoiceURI.value = saved
      } else if (!browserVoiceURI.value) {
        const zh = vs.find(v => /^(zh|cmn|zh-CN|zh-TW)/i.test(v.lang)) ||
                  vs.find(v => /中文|普通话|Chinese|zh/i.test(v.name)) ||
                  vs[0]
        browserVoiceURI.value = zh ? zh.voiceURI : ''
      }
    }
    pick()
    if ('onvoiceschanged' in window.speechSynthesis) {
      window.speechSynthesis.onvoiceschanged = pick
    }
  }

  // ===== WebSocket（连接复用 + 并发合成）=====
  let ws = null
  let wsSeq = 0
  let wsDisabled = false // 连接级失败后本会话不再尝试 WS（自动回退 HTTP）
  let wsConnecting = null
  const wsPending = new Map() // id -> {resolve, reject}

  const rejectAllWsPending = (err) => {
    wsPending.forEach(p => p.reject(err))
    wsPending.clear()
  }

  const ensureWs = () => {
    if (wsDisabled) return Promise.reject(new Error('TTS WebSocket 已禁用'))
    if (ws && ws.readyState === WebSocket.OPEN) return Promise.resolve(ws)
    if (wsConnecting) return wsConnecting
    wsConnecting = new Promise((resolve, reject) => {
      const sock = new WebSocket(`${wsApiBase()}/tts/ws?token=${getToken()}`)
      sock.binaryType = 'arraybuffer'
      let settled = false
      // 关键：连接建立失败/断开时必须 reject，否则调用方永久挂起，HTTP 回退也不会触发
      const fail = (msg) => {
        if (settled) return
        settled = true
        wsConnecting = null
        reject(new Error(msg))
      }
      sock.onopen = () => {
        settled = true
        wsConnecting = null
        ws = sock
        resolve(sock)
      }
      sock.onerror = () => {
        wsDisabled = true
        ws = null
        fail('TTS WebSocket 连接失败')
        rejectAllWsPending(new Error('TTS WebSocket 连接失败'))
      }
      sock.onclose = () => {
        wsConnecting = null
        if (ws === sock) ws = null
        fail('TTS WebSocket 连接已断开')
        rejectAllWsPending(new Error('TTS WebSocket 连接已断开'))
      }
      sock.onmessage = (ev) => {
        if (typeof ev.data === 'string') {
          let msg = null
          try { msg = JSON.parse(ev.data) } catch (e) { return }
          const p = wsPending.get(msg.id)
          if (p) { wsPending.delete(msg.id); p.reject(new Error(msg.error || '合成失败')) }
          return
        }
        // 二进制：[int32 BE id][MP3]
        const buf = new DataView(ev.data)
        const id = buf.getInt32(0)
        const p = wsPending.get(id)
        if (!p) return
        wsPending.delete(id)
        p.resolve(new Blob([ev.data.slice(4)], { type: 'audio/mpeg' }))
      }
    })
    return wsConnecting
  }

  const closeWs = () => {
    rejectAllWsPending(new Error('朗读已停止'))
    if (ws) {
      ws.onclose = null
      ws.onerror = null
      ws.close()
      ws = null
    }
  }

  // 通过 WebSocket 合成一句
  const synthesizeWs = (i) => {
    return ensureWs().then(sock => new Promise((resolve, reject) => {
      const id = ++wsSeq
      wsPending.set(id, { resolve, reject })
      sock.send(JSON.stringify({
        id,
        text: sentences.value[i],
        emotion: emotion.value,
        speed_factor: speed.value,
        voice: voice.value
      }))
    }))
  }

  // 合成单句（返回 Promise<Blob>，播放与预取共用）：优先 WS，失败回退 HTTP
  const synthesizeOne = (i) => {
    return synthesizeWs(i)
      .catch(err => {
        console.warn('TTS WebSocket 合成失败，回退 HTTP：', err.message)
        return bookApi.synthesizeTts({
          text: sentences.value[i],
          emotion: emotion.value,
          speed_factor: speed.value,
          media_type: 'wav',
          voice: voice.value
        }).then(resp => new Blob([resp.data], { type: resp.headers['content-type'] || 'audio/wav' }))
      })
  }

  // ===== 音频元素（复用单例，避免移动端自动播放策略拒绝新建元素）=====
  const getPlayElement = () => {
    if (!audioEl) {
      audioEl = new Audio()
      audioEl.preload = 'auto'
      audioEl.setAttribute('playsinline', '')
      audioEl.setAttribute('webkit-playsinline', '')
    }
    return audioEl
  }

  const stopAudio = () => {
    if (audioEl) {
      audioEl.pause()
      audioEl.onended = null
      audioEl.onerror = null
    }
    if (objectUrl) {
      URL.revokeObjectURL(objectUrl)
      objectUrl = null
    }
  }

  // 播放被自动播放策略拒绝后，等待用户轻触屏幕（产生用户手势）自动恢复播放
  const resumeAfterUnlock = () => {
    cleanupUnlock()
    const idx = pendingPlay
    pendingPlay = null
    if (idx == null || !active.value) return
    // 元素 src 已就绪：直接在有手势的上下文中恢复播放，无需重新合成
    if (audioEl && audioEl.src) {
      audioEl.play().then(() => {
        playing.value = true
        paused.value = false
      }).catch(() => {
        playSentence(idx) // 极少数 WebView 仍拒绝：走完整重播流程
      })
      return
    }
    playSentence(idx)
  }
  const cleanupUnlock = () => {
    if (!unlockBound) return
    unlockBound = false
    window.removeEventListener('pointerdown', resumeAfterUnlock, { capture: true })
  }
  const registerUnlock = () => {
    if (unlockBound) return
    unlockBound = true
    window.addEventListener('pointerdown', resumeAfterUnlock, { capture: true, once: true })
  }

  // ===== 预取缓存 =====
  const prefetch = (i) => {
    const n = sentences.value.length
    if (!active.value || paused.value || i < 0 || i >= n) return
    for (let k = i; k < i + TTS_PREFETCH_WINDOW && k < n; k++) {
      if (cache.has(k)) continue
      cache.set(k, synthesizeOne(k).catch(() => null))
    }
  }

  const takeCache = async (i) => {
    const item = cache.get(i)
    cache.delete(i)
    return item ? await item : null
  }

  // ===== 播放控制 =====
  const playSentence = async (i) => {
    const n = sentences.value.length
    if (!active.value || i < 0) return
    if (i >= n) { handleEnd(); return }
    index.value = i
    currentText.value = sentences.value[i]
    onSentence?.(i, sentences.value[i])
    stopAudio()
    // 浏览器离线引擎：直接用 SpeechSynthesis 朗读，无需后端合成
    if (engine.value === 'browser') return playSentenceBrowser(i)
    loading.value = true
    prefetch(i) // 并发预取当前句与后续句，利用 WS 复用连接
    try {
      let blob = await takeCache(i)
      if (!blob && index.value === i) blob = await synthesizeOne(i)
      if (index.value !== i) return // 已被跳过/停止
      const url = URL.createObjectURL(blob)
      objectUrl = url
      const el = getPlayElement()
      el.onended = () => {
        if (objectUrl === url) objectUrl = null
        failCount = 0
        if (!paused.value && active.value) playSentence(i + 1)
      }
      el.onerror = () => {
        if (objectUrl === url) objectUrl = null
        failCount += 1
        if (failCount >= 3) {
          stop()
        } else if (active.value) {
          playSentence(i + 1)
        }
      }
      el.src = url
      await el.play()
      playing.value = true
      paused.value = false
    } catch (e) {
      // 移动端自动播放策略拒绝无手势 play()：不是合成失败，等待用户轻触屏幕恢复
      if (isAutoplayBlocked(e)) {
        pendingPlay = i
        playing.value = false
        paused.value = true
        registerUnlock()
        return
      }
      // 真正的合成/解码失败：释放未播放成功的 blob URL 再跳过
      if (objectUrl) {
        URL.revokeObjectURL(objectUrl)
        objectUrl = null
      }
      failCount += 1
      if (failCount >= 3) {
        stop()
      } else if (active.value) {
        playSentence(i + 1)
      }
    } finally {
      loading.value = false
    }
  }

  // 浏览器离线朗读：直接调用 SpeechSynthesis 朗读当前句（参考 hanzi.html 的用法）
  const playSentenceBrowser = (i) => {
    if (!browserSupported) {
      ElMessage?.error?.('当前浏览器不支持语音合成（SpeechSynthesis）')
      return
    }
    const synth = window.speechSynthesis
    synth.cancel() // 取消上句，避免 Chrome 串句
    try { if (synth.paused) synth.resume() } catch (e) { /* noop */ }
    const text = sentences.value[i]
    const u = new SpeechSynthesisUtterance(text)
    u.lang = 'zh-CN'
    u.rate = speed.value
    u.pitch = 1.0
    const v = browserVoices.value.find(v => v.voiceURI === browserVoiceURI.value)
    if (v) u.voice = v
    loading.value = false
    playing.value = true
    paused.value = false
    browserUtter = u
    // 看门狗：Chrome 偶发 onend 不触发导致卡住，超时则强制续播下一句
    const watchdog = setTimeout(() => {
      if (active.value && index.value === i && !paused.value) {
        try { synth.cancel() } catch (e) {}
        browserUtter = null
        playSentence(i + 1)
      }
    }, 8000 + text.length * 400)
    const clearWatch = () => clearTimeout(watchdog)
    u.onend = () => {
      clearWatch()
      browserUtter = null
      failCount = 0
      if (!paused.value && active.value) playSentence(i + 1)
    }
    u.onerror = (e) => {
      clearWatch()
      browserUtter = null
      failCount += 1
      if (failCount >= 3) {
        stop()
        ElMessage?.error?.('连续多句朗读失败，已停止')
      } else if (active.value) {
        playSentence(i + 1)
      }
    }
    // Chrome 偶发「cancel 后首句被吞」，延迟一拍再 speak 规避
    setTimeout(() => {
      if (active.value && index.value === i && !paused.value) synth.speak(u)
    }, 10)
  }

  // 当前内容播放完毕：由调用方决定加载下一章或结束
  const handleEnd = () => {
    if (!active.value) return
    if (onEnd) onEnd()
    else stop()
  }

  // 开始朗读：sentences 为句子列表，startIdx 为起始句下标
  const start = async (list, startIdx = 0) => {
    if (active.value) stop()
    // 浏览器支持则默认离线引擎：同步恢复偏好，避免后端网络阻塞；无偏好时按浏览器能力选默认
    const prefEngine = loadReaderPrefs().ttsEngine
    if (prefEngine) {
      engine.value = prefEngine
    } else if (browserSupported) {
      engine.value = 'browser'
    }
    if (engine.value === 'browser') {
      loadBrowserVoices()
    } else {
      await loadSettings()
    }
    sentences.value = list || []
    active.value = true
    playing.value = true
    paused.value = false
    index.value = -1
    failCount = 0
    cache.clear()
    if (!sentences.value.length) {
      active.value = false
      playing.value = false
      return false
    }
    playSentence(Math.max(0, startIdx))
    return true
  }

  // 章节切换：替换句子列表并从首句继续（调用方需保证 active）
  const setSentences = (list) => {
    if (!active.value) return
    cache.clear()
    sentences.value = list || []
    index.value = -1
    failCount = 0
    if (sentences.value.length) {
      playSentence(0)
    } else {
      handleEnd()
    }
  }

  const stop = () => {
    active.value = false
    playing.value = false
    paused.value = false
    index.value = -1
    currentText.value = ''
    sentences.value = []
    cache.clear()
    stopAudio()
    if (browserSupported) { try { window.speechSynthesis.cancel() } catch (e) {} }
    browserUtter = null
    closeWs()
  }

  const togglePlay = () => {
    if (!active.value) return
    // 浏览器引擎：直接 pause/resume SpeechSynthesis
    if (engine.value === 'browser') {
      const synth = window.speechSynthesis
      if (paused.value) {
        try { synth.resume() } catch (e) {}
        paused.value = false
        playing.value = true
      } else {
        try { synth.pause() } catch (e) {}
        paused.value = true
        playing.value = false
      }
      return
    }
    if (!audioEl) return
    // 处于"等待解锁"状态：由全局 pointerdown 手势监听统一恢复，这里不重复处理
    if (pendingPlay != null) return
    if (paused.value) {
      audioEl.play().catch(() => {})
      paused.value = false
    } else {
      audioEl.pause()
      paused.value = true
    }
  }

  const prev = () => {
    if (index.value > 0) playSentence(index.value - 1)
  }

  const next = () => {
    if (index.value < sentences.value.length - 1) playSentence(index.value + 1)
  }

  // 切换音色/语速后，若正在播放则用新设置重播当前句
  const restartIfPlaying = () => {
    if (active.value && index.value >= 0) {
      cache.clear() // 清掉旧设置的预取缓存，确保后续句也按新设置重新合成
      playSentence(index.value)
    }
  }

  // ===== 设置 =====
  const loadSettings = async () => {
    // 引擎偏好优先取本地保存（即使离线也能用浏览器引擎）
    const savedPrefs = loadReaderPrefs()
    if (savedPrefs.ttsEngine) {
      engine.value = savedPrefs.ttsEngine
      if (engine.value === 'browser') { loadBrowserVoices(); return } // 离线引擎无需后端
    } else if (browserSupported) {
      // 无偏好且浏览器支持：默认离线引擎，避免受限网络下微软 403
      engine.value = 'browser'
      loadBrowserVoices()
      return
    }
    try {
      const { data } = await bookApi.getTtsSettings()
      if (data?.data?.config) {
        config.value = { ...config.value, ...data.data.config }
        if (data.data.config.speed_factor > 0) speed.value = data.data.config.speed_factor
      }
      if (data?.data?.emotions?.length) emotions.value = data.data.emotions
      if (data?.data?.engines?.length) engines.value = data.data.engines
      if (data?.data?.voices?.length) {
        voices.value = data.data.voices
        if (!voice.value && voices.value.length) voice.value = voices.value[0].short_name
      } else {
        // 音色列表加载失败（离线/网络），仍保留默认晓晓
        voices.value = [{ short_name: voice.value, friendly_name: '晓晓（默认）', gender: 'Female' }]
      }
      // 音色偏好优先取本地保存值（Go 后端不保存朗读配置），其次用默认配置
      const savedVoice = savedPrefs.ttsVoice
      if (savedVoice) voice.value = savedVoice
      else if (data?.data?.config?.voice) voice.value = data.data.config.voice
      // 本地未保存引擎偏好且不支持浏览器时，回退到后端默认引擎
      if (!savedPrefs.ttsEngine && !browserSupported && data?.data?.config?.engine) engine.value = data.data.config.engine
    } catch (e) {
      // 静默失败，使用默认配置（含本地已保存/默认的浏览器引擎）
    }
  }

  // 切换朗读引擎（云端 / 浏览器离线）。浏览器引擎会预载本地可用音色
  const setEngine = (e) => {
    engine.value = e
    const prefs = loadReaderPrefs()
    saveReaderPrefs({ ...prefs, ttsEngine: e })
    if (e === 'browser') {
      loadBrowserVoices()
    } else {
      // 切到云端时确保音色/语气/语速等配置已加载（失败则静默保留默认）
      loadSettings()
    }
    if (active.value && index.value >= 0) restartIfPlaying()
  }

  // 浏览器引擎音色选择
  const onBrowserVoiceChange = (uri) => {
    browserVoiceURI.value = uri
    const prefs = loadReaderPrefs()
    saveReaderPrefs({ ...prefs, ttsBrowserVoice: uri })
    if (active.value && index.value >= 0 && engine.value === 'browser') restartIfPlaying()
  }

  // 切换人声：更新当前音色并本地记住偏好，朗读中则用新音色重播当前句
  const onVoiceChange = (v) => {
    config.value.voice = v
    if (active.value && index.value >= 0) restartIfPlaying()
    saveReaderPrefs({ ...loadReaderPrefs(), ttsVoice: v })
  }

  // 音色显示名（下拉用）
  const voiceLabel = (v) => {
    let name = (v.friendly_name || v.short_name || '').replace(/^Microsoft\s*/, '')
    name = name.replace(/\s*-\s*Chinese\s*\(Mainland\)\s*$/, '')
    const g = v.gender === 'Female' ? '女声' : v.gender === 'Male' ? '男声' : ''
    return g ? `${name}（${g}）` : name
  }

  return {
    // 状态
    active, playing, paused, loading, index, sentences, currentText,
    speed, emotion, emotions, voices, voice, engines, config,
    engine, browserVoices, browserVoiceURI, browserSupported,
    // 方法
    start, setSentences, stop, togglePlay, prev, next,
    loadSettings, onVoiceChange, restartIfPlaying, voiceLabel,
    setEngine, onBrowserVoiceChange
  }
}
