<template>
  <div class="page">
    <!-- 顶部标题区 -->
    <div class="page-hero">
      <span class="hero-icon">
        <el-icon :size="24"><Headset /></el-icon>
      </span>
      <div class="hero-text">
        <h2 class="hero-title">有声书</h2>
        <p class="hero-sub">将图书自动生成语音，随时聆听。支持 EPUB / AZW3 / PDF</p>
      </div>
      <div class="hero-badge">
        <el-icon :size="16"><Headset /></el-icon>
        <span>AI 语音合成</span>
      </div>
    </div>

    <!-- 统计卡片：点击筛选列表，再次点击取消 -->
    <div class="stats-row">
      <div
        v-for="s in statCards"
        :key="s.key"
        class="stat-card"
        :class="{ active: activeFilter === s.key }"
        @click="toggleFilter(s.key)"
      >
        <div class="stat-icon" :style="{ background: s.gradient, boxShadow: `0 10px 24px ${s.glow}` }">
          <el-icon :size="22"><component :is="s.icon" /></el-icon>
        </div>
        <div>
          <div class="stat-value">{{ s.value }}</div>
          <div class="stat-label">{{ s.label }}</div>
        </div>
        <span class="stat-shine" :style="{ background: s.gradient }"></span>
      </div>
    </div>

    <!-- 视图切换 -->
    <div class="list-toolbar">
      <ViewModeSwitch v-model="viewMode" />
    </div>

    <!-- 任务列表 -->
    <div v-loading="loading" class="list-wrap">
      <div v-if="!loading && tasks.length === 0" class="empty">
        <el-icon :size="56"><Headset /></el-icon>
        <p>还没有有声书，快去书架选一本书生成吧</p>
        <el-button type="primary" round @click="goBookshelf">去书架选书</el-button>
      </div>

      <!-- 筛选后该分类下为空 -->
      <div v-else-if="!loading && filteredTasks.length === 0" class="empty">
        <el-icon :size="56"><FolderOpened /></el-icon>
        <p>该分类下暂无有声书</p>
        <el-button type="primary" round plain @click="activeFilter = 'all'">查看全部</el-button>
      </div>

      <div v-else :class="viewMode === 'grid' ? 'task-grid' : 'task-list'">
        <div v-for="t in pagedTasks" :key="t.id" class="task-card" :class="['card-' + t.status, { playing: isNowPlaying(t) }]">
          <div class="task-head">
            <div class="book-cover">
              <img v-if="coverUrl(t)" :src="coverUrl(t)" alt="" />
              <el-icon v-else :size="30"><Notebook /></el-icon>
              <span v-if="isNowPlaying(t)" class="cover-badge playing">
                <span class="eq-bar-wrap"><i class="eq-bar"></i><i class="eq-bar"></i><i class="eq-bar"></i></span>
              </span>
              <span v-else-if="t.status === 'done'" class="cover-badge done">
                <el-icon :size="12"><Check /></el-icon>
              </span>
              <span v-else-if="t.status === 'processing'" class="cover-badge">
                <span class="eq-bar-wrap"><i class="eq-bar"></i><i class="eq-bar"></i><i class="eq-bar"></i></span>
              </span>
              <span v-else-if="t.status === 'pending'" class="cover-badge"><span class="pulse-dot"></span></span>
              <span v-else-if="t.status === 'paused'" class="cover-badge paused">
                <el-icon :size="12"><VideoPause /></el-icon>
              </span>
            </div>
            <div class="task-info">
              <div class="book-title" :title="t.book_title || t.book?.title">{{ t.book_title || t.book?.title || '未知书名' }}</div>
              <div class="book-meta">
                <el-tag v-if="t.book && t.book.file_type" size="small" effect="plain" class="fmt-tag">{{ (t.book.file_type || '').toUpperCase() }}</el-tag>
                <el-tag :type="statusMeta(t.status).type" size="small" effect="light" class="status-tag" :class="'tag-' + t.status">
                  <el-icon v-if="t.status === 'done'" :size="12"><CircleCheck /></el-icon>
                  <el-icon v-else-if="t.status === 'processing' || t.status === 'pending'" :size="12" class="tag-spin"><Loading /></el-icon>
                  <el-icon v-else-if="t.status === 'paused'" :size="12"><VideoPause /></el-icon>
                  {{ statusMeta(t.status).label }}
                </el-tag>
              </div>
            </div>
            <el-dropdown trigger="click" class="task-more" @command="(cmd) => onMore(cmd, t)">
              <el-icon :size="18" class="more-icon"><MoreFilled /></el-icon>
              <template #dropdown>
                <el-dropdown-menu>
                <template v-if="t.status === 'processing' || t.status === 'pending'">
                  <el-dropdown-item command="pause">
                    <el-icon><VideoPause /></el-icon><span>暂停生成</span>
                  </el-dropdown-item>
                  <el-dropdown-item command="cancel">
                    <el-icon><CircleClose /></el-icon><span>取消生成</span>
                  </el-dropdown-item>
                </template>
                  <template v-else>
                    <el-dropdown-item command="regenerate">
                      <el-icon><RefreshRight /></el-icon><span>重新生成</span>
                    </el-dropdown-item>
                    <el-dropdown-item v-if="t.status === 'done'" command="share">
                      <el-icon><Share /></el-icon><span>分享有声书</span>
                    </el-dropdown-item>
                  </template>
                  <el-dropdown-item command="delete" divided>
                    <el-icon><Delete /></el-icon><span>删除有声书</span>
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>

          <!-- 生成中：进度条 -->
          <div v-if="t.status === 'processing' || t.status === 'pending'" class="task-body">
            <div class="progress-row">
              <span class="gen-spinner"></span>
              <el-progress
                :percentage="progressPercent(t)"
                :stroke-width="8"
                :color="progressColor"
              />
            </div>
            <div class="progress-meta">
              <span v-if="t.status === 'pending'">
                <el-icon :size="12" class="tag-spin"><Loading /></el-icon> 排队中…
              </span>
              <span v-else>正在生成第 {{ t.done_segments }}/{{ t.total_segments }} 段 · {{ progressPercent(t) }}%</span>
              <span class="voice-name" :title="t.voice">
                <el-icon :size="12"><Microphone /></el-icon>{{ taskVoiceLabel(t) }}
              </span>
            </div>
            <div v-if="t.done_segments > 0" class="task-actions" style="margin-top: 10px">
              <el-button size="small" type="success" round plain :icon="Headset" @click="openPlay(t)">
                试听已生成 {{ t.done_segments }} 段
              </el-button>
            </div>
          </div>

          <!-- 完成：可播放 -->
          <div v-else-if="t.status === 'done'" class="task-body">
            <div class="done-meta">
              <span class="done-item"><el-icon><FolderOpened /></el-icon>{{ t.total_segments }} 段</span>
              <span class="done-item"><el-icon><Clock /></el-icon>约 {{ formatMinutes(t.total_segments) }} 分钟</span>
              <span class="voice-name" :title="t.voice"><el-icon><Microphone /></el-icon>{{ taskVoiceLabel(t) }}</span>
            </div>
            <div class="task-actions">
              <span v-if="isNowPlaying(t)" class="now-playing-tip">
                <span class="eq-bar-wrap"><i class="eq-bar"></i><i class="eq-bar"></i><i class="eq-bar"></i></span>
                正在播放
              </span>
              <el-button v-else type="primary" round class="gradient-btn shine-btn" :icon="Headset" @click="openPlay(t, false)">开始聆听</el-button>
            </div>
          </div>

          <!-- 失败 / 取消 / 暂停 -->
          <div v-else class="task-body">
            <div class="error-text" :title="t.error">{{ errorText(t) }}</div>
            <div class="task-actions">
              <el-button
                v-if="t.done_segments > 0"
                type="success"
                round
                plain
                :icon="Headset"
                @click="openPlay(t)"
              >
                试听已生成 {{ t.done_segments }} 段
              </el-button>
              <el-button v-if="t.status === 'paused'" type="success" round :icon="VideoPlay" @click="doResume(t)">继续生成</el-button>
              <el-button v-else-if="t.status === 'cancelled'" type="success" round :icon="VideoPlay" @click="openGenerate(t, true)">继续生成</el-button>
              <el-button v-else type="primary" round :icon="RefreshRight" @click="openGenerate(t, true)">重新生成</el-button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-if="filteredTasks.length > pageSize" class="pagination">
      <PaginationBar
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="filteredTasks.length"
        :page-sizes="pageSizesOptions"
        @size-change="onSizeChange"
      />
    </div>

    <!-- 生成设置弹窗 -->
    <el-dialog v-model="generateVisible" title="生成有声书" width="480px" :close-on-click-modal="false">
      <div v-if="generateBook" class="gen-desc">
        <img v-if="generateBook.id && generateBook.cover_path" :src="bookApi.coverUrl(generateBook.id)" class="gen-cover" alt="" />
        <div>
          <div class="gen-book-title">{{ generateBook.title }}</div>
          <div class="gen-book-sub">{{ (generateBook.file_type || '').toUpperCase() }} · 将按章节智能分段（每段约 10 分钟朗读量）</div>
        </div>
      </div>
      <el-form label-position="top">
        <el-form-item label="朗读音色">
          <el-select v-model="generateForm.voice" filterable placeholder="请选择音色" style="width: 100%">
            <el-option-group v-if="femaleVoices.length" label="女声">
              <el-option v-for="v in femaleVoices" :key="v.short_name" :label="v.friendly_name" :value="v.short_name" />
            </el-option-group>
            <el-option-group v-if="maleVoices.length" label="男声">
              <el-option v-for="v in maleVoices" :key="v.short_name" :label="v.friendly_name" :value="v.short_name" />
            </el-option-group>
            <el-option-group v-if="otherVoices.length" label="其他">
              <el-option v-for="v in otherVoices" :key="v.short_name" :label="v.friendly_name" :value="v.short_name" />
            </el-option-group>
            <template #empty>
              <div class="voice-empty">音色列表加载失败，将使用默认音色</div>
            </template>
          </el-select>
        </el-form-item>
        <el-form-item label="生成引擎">
          <el-select v-model="generateForm.engine" style="width: 100%">
            <el-option v-for="e in engineOptions" :key="e.key" :label="e.label" :value="e.key">
              <span class="engine-opt-label">{{ e.label }}</span>
              <span v-if="e.desc" class="engine-opt-desc">{{ e.desc }}</span>
            </el-option>
          </el-select>
          <div class="engine-opt-tip">
            微软 Edge TTS 在受限网络下可能不可用（403 / 握手失败）：后端已内置时钟偏移校正自动重试；若仍失败，请在启动后端的环境中设置环境变量 EDGE_TTS_PROXY 或 HTTPS_PROXY 让其经代理出网。「自动」会在 Edge TTS 失败时回退到已配置的本地引擎。
          </div>
        </el-form-item>
        <el-form-item label="朗读语速">
          <div class="speed-row">
            <el-slider v-model="generateForm.speed_factor" :min="0.5" :max="2" :step="0.1" style="flex: 1" />
            <span class="speed-val">{{ generateForm.speed_factor.toFixed(1) }}x</span>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="generateVisible = false">取消</el-button>
        <el-button type="primary" :loading="generating" :disabled="!generateForm.voice" class="gradient-btn" @click="submitGenerate">
          开始生成
        </el-button>
      </template>
    </el-dialog>

    <!-- 分享弹窗 -->
    <el-dialog v-model="shareVisible" title="分享有声书" width="560px" align-center append-to-body class="detail-dialog">
      <div class="share-desc">
        <el-icon :size="16"><Share /></el-icon>
        <span>生成分享链接，好友打开链接即可查看并把这本有声书接收（复制音频文件）到自己账户。</span>
      </div>

      <!-- 创建分享 -->
      <div class="share-create">
        <el-radio-group v-model="shareExpiresDays" size="default">
          <el-radio-button :value="0">永久有效</el-radio-button>
          <el-radio-button :value="7">7 天</el-radio-button>
          <el-radio-button :value="30">30 天</el-radio-button>
        </el-radio-group>
        <el-button
          type="primary"
          :loading="creatingShare"
          :disabled="shares.length >= 5"
          @click="createShare"
        >
          <el-icon><Plus /></el-icon> 生成分享链接
        </el-button>
      </div>

      <!-- 分享列表 -->
      <div v-loading="loadingShares" class="share-list">
        <div v-if="!loadingShares && shares.length === 0" class="share-empty">
          还没有分享链接
        </div>
        <div v-for="s in shares" :key="s.id" class="share-item">
          <div class="share-item-info">
            <div class="share-item-link">{{ shareUrl(s.token) }}</div>
            <div class="share-item-meta">
              <el-tag v-if="s.expires_at" size="small" type="warning" effect="plain">
                至 {{ formatDate(s.expires_at) }}
              </el-tag>
              <el-tag v-else size="small" type="success" effect="plain">永久有效</el-tag>
              <span class="share-item-time">{{ formatDate(s.created_at) }} 创建</span>
            </div>
          </div>
          <div class="share-item-actions">
            <el-tooltip content="复制链接" placement="top">
              <el-button size="small" circle @click="copyShare(s.token)">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
            </el-tooltip>
            <el-tooltip content="撤销分享" placement="top">
              <el-button size="small" circle type="danger" plain @click="revokeShare(s)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </el-tooltip>
          </div>
        </div>
      </div>
    </el-dialog>

  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { audiobookApi } from '@/api/audiobook'
import { audiobookShareApi, getShareConfig } from '@/api/share'
import { copyText } from '@/utils/clipboard'
import { bookApi } from '@/api/book'
import { useViewPagination } from '@/composables/useViewPagination'
import ViewModeSwitch from '@/components/ViewModeSwitch.vue'
import PaginationBar from '@/components/PaginationBar.vue'
import axios from 'axios'
import { API_BASE } from '@/config'
import { openPlay, stopPlayer, playerState } from '@/player'

const router = useRouter()
const route = useRoute()

// ===== 列表与轮询 =====
const tasks = ref([])
const loading = ref(false)
let pollTimer = null

const counts = computed(() => {
  const c = { processing: 0, done: 0, failed: 0 }
  for (const t of tasks.value) {
    if (t.status === 'processing' || t.status === 'pending') c.processing++
    else if (t.status === 'done') c.done++
    else c.failed++
  }
  return c
})

// 封面地址：优先任务自身封面（接收的分享任务无原书），否则回退原书封面
const coverUrl = (t) => {
  if (t.cover_path) return audiobookApi.coverUrl(t.id)
  if (t.book && t.book.cover_path) return bookApi.coverUrl(t.book_id)
  return ''
}

// 统计筛选：all / processing / done / failed（点击卡片切换，再次点击取消）
const activeFilter = ref('all')
const filteredTasks = computed(() => {
  if (activeFilter.value === 'all') return tasks.value
  return tasks.value.filter((t) => {
    if (activeFilter.value === 'processing') return t.status === 'processing' || t.status === 'pending'
    if (activeFilter.value === 'done') return t.status === 'done'
    return t.status === 'failed' || t.status === 'cancelled' || t.status === 'paused'
  })
})

// 分页（后端返回全部任务，前端切片）；书架风格：两行自动 + 倍数选项 + 按视图持久化 + 缩放重算
const { viewMode, page, pageSize, pageSizesOptions, onSizeChange } =
  useViewPagination('audiobooks', () => {})
const pagedTasks = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return filteredTasks.value.slice(start, start + pageSize.value)
})
watch(activeFilter, () => { page.value = 1 })

function toggleFilter(key) {
  activeFilter.value = activeFilter.value === key ? 'all' : key
}

// 统计卡片（与书架页同款结构）
const statCards = computed(() => [
  {
    key: 'all',
    label: '全部',
    value: tasks.value.length,
    icon: 'Collection',
    gradient: 'linear-gradient(135deg,#6366f1,#8b5cf6)',
    glow: 'rgba(99,102,241,0.45)'
  },
  {
    key: 'processing',
    label: '生成中',
    value: counts.value.processing,
    icon: 'Loading',
    gradient: 'linear-gradient(135deg,#a855f7,#6366f1)',
    glow: 'rgba(168,85,247,0.45)'
  },
  {
    key: 'done',
    label: '已完成',
    value: counts.value.done,
    icon: 'Headset',
    gradient: 'linear-gradient(135deg,#10b981,#34d399)',
    glow: 'rgba(16,185,129,0.45)'
  },
  {
    key: 'failed',
    label: '暂停/失败',
    value: counts.value.failed,
    icon: 'Warning',
    gradient: 'linear-gradient(135deg,#fb7185,#f43f5e)',
    glow: 'rgba(244,63,94,0.45)'
  }
])

const hasActive = computed(() => tasks.value.some((t) => t.status === 'processing' || t.status === 'pending'))

const statusMeta = (s) => {
  const map = {
    pending: { label: '排队中', type: 'info' },
    processing: { label: '生成中', type: 'primary' },
    done: { label: '已完成', type: 'success' },
    failed: { label: '失败', type: 'danger' },
    cancelled: { label: '已取消', type: 'warning' },
    paused: { label: '已暂停', type: 'warning' }
  }
  return map[s] || { label: s, type: 'info' }
}

// 当前正在播放的有声书
const isNowPlaying = (t) => !!(playerState.task && playerState.task.id === t.id)

// 失败/取消/暂停卡片的说明文案
const errorText = (t) => {
  if (t.status === 'paused') {
    return `已暂停生成，已完成 ${t.done_segments}/${t.total_segments} 段，可随时继续`
  }
  if (t.status === 'cancelled') {
    return t.error || '已取消生成，已生成的分段已保留'
  }
  return t.error || '生成失败'
}

const progressPercent = (t) => {
  if (!t.total_segments) return t.status === 'pending' ? 0 : 100
  return Math.min(100, Math.round((t.done_segments / t.total_segments) * 100))
}
const progressColor = (p) => (p >= 100 ? '#34d399' : '#6366f1')

const voiceDisplay = (v) => (v || '默认音色').replace(/^Microsoft Server Speech Text to Speech Voice \(/i, '').replace(/\)$/, '')
// 卡片音色展示：浏览器离线任务显示「本机语音」（voice 字段存的是本机 voiceURI，不宜直出）
const taskVoiceLabel = (t) =>
  t.engine === 'browser' ? '本机语音' : voiceDisplay(t.voice)
const formatMinutes = (segments) => {
  const total = playDurationOf(segments)
  return total >= 60 ? Math.round(total / 60) + '+' : Math.max(1, Math.round(total)) + ''
}
// 每段约 10 分钟估算（3000 字符 @5字/秒）
const playDurationOf = (segments) => segments * 600

async function refresh(showLoading = true) {
  if (showLoading) loading.value = true
  try {
    const res = await audiobookApi.list()
    tasks.value = res.data.data || []
    recoverStaleBrowserTasks()
  } catch (e) {
    // 静默失败，轮询会再试
  } finally {
    loading.value = false
    // 无进行中任务时自动停止轮询（播放器自身的轮询独立维护）
    if (!hasActive.value) stopPolling()
  }
}

// 页面加载时本标签页不可能有存活中的生成会话：processing 的浏览器任务一律回退暂停，
// 避免页面意外关闭导致任务卡在 processing、阻塞后续生成
const recoveredBrowserIds = new Set()
function recoverStaleBrowserTasks() {
  const stale = tasks.value.filter(
    (t) =>
      (t.engine === 'browser') &&
      t.status === 'processing' &&
      !recoveredBrowserIds.has(t.id)
  )
  for (const t of stale) {
    recoveredBrowserIds.add(t.id)
    audiobookApi.pause(t.id).catch(() => {})
  }
}

function startPolling() {
  stopPolling()
  pollTimer = setInterval(() => {
    refresh(false)
  }, 5000)
}
function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

// ===== 生成设置 =====
const generateVisible = ref(false)
const generateBook = ref(null)
const generating = ref(false)
const voices = ref([])
const generateForm = reactive({ voice: '', speed_factor: 1.0, engine: 'auto' })
// 引擎选项：auto（edge-tts 失败自动回退本地引擎）+ 后端下发的云端引擎列表
const engineOptions = ref([
  { key: 'auto', label: '自动（推荐）', desc: 'Edge TTS 优先，失败自动切换本地引擎' }
])

const femaleVoices = computed(() => voices.value.filter((v) => v.gender === 'Female'))
const maleVoices = computed(() => voices.value.filter((v) => v.gender === 'Male'))
const otherVoices = computed(() => voices.value.filter((v) => v.gender !== 'Female' && v.gender !== 'Male'))

async function loadVoices() {
  try {
    const res = await axios.get(`${API_BASE}/tts/settings`)
    const data = res.data?.data || {}
    voices.value = data.voices || []
    const cfg = data.config || {}
    engineOptions.value = [
      { key: 'auto', label: '自动（推荐）', desc: 'Edge TTS 优先，失败自动切换本地引擎' },
      ...(data.engines || [])
    ]
    if (!generateForm.voice) generateForm.voice = cfg.voice || (voices.value[0] ? voices.value[0].short_name : '')
    if (!generateForm.speed_factor || generateForm.speed_factor <= 0) generateForm.speed_factor = cfg.speed_factor || 1.0
  } catch (e) {
    voices.value = []
    generateForm.voice = generateForm.voice || ''
  }
}

function openGenerate(task, isRetry = false) {
  if (isRetry && task) {
    // 重新生成/继续生成：填充书籍信息，否则弹窗内生成按钮无任务可提交
    generateBook.value = {
      id: task.book_id,
      title: task.book_title,
      cover_path: task.book?.cover_path,
      file_type: task.book?.file_type
    }
    generateForm.voice = task.voice || generateForm.voice
    generateForm.speed_factor = task.speed_factor || 1.0
    generateForm.engine = task.engine || 'auto'
  }
  generateVisible.value = true
}

async function submitGenerate() {
  if (!generateBook.value) return
  generating.value = true
  try {
    await audiobookApi.create({
      book_id: generateBook.value.id,
      engine: generateForm.engine,
      voice: generateForm.voice,
      speed_factor: generateForm.speed_factor
    })
    generateVisible.value = false
    await refresh()
    ElMessage.success('已开始生成，可在下方查看进度')
    startPolling()
  } catch (e) {
    const msg = e.response?.data?.error || '生成失败'
    if (msg.includes('已有生成任务')) {
      await refresh()
      startPolling()
      ElMessage.info(msg)
    } else {
      ElMessage.error(msg)
    }
  } finally {
    generating.value = false
  }
}

// 从书架选书入口（书架侧边调用时打开选书）
function goBookshelf() {
  router.push('/bookshelf')
}

// 从书架页跳转携带书籍参数（?book_id=...）自动打开生成弹窗
function openFromQuery() {
  const q = route.query
  if (!q.book_id) return
  generateBook.value = {
    id: Number(q.book_id),
    title: q.title || '',
    cover_path: q.cover_path || '',
    file_type: q.file_type || ''
  }
  generateVisible.value = true
  // 清除 query，避免刷新页面时重复弹出
  router.replace({ path: '/audiobooks', query: {} })
}

// ===== 操作 =====
function onMore(cmd, task) {
  if (cmd === 'pause') doPause(task)
  else if (cmd === 'cancel') doCancel(task)
  else if (cmd === 'regenerate') doRegenerate(task)
  else if (cmd === 'share') openShareDialog(task)
  else if (cmd === 'delete') doDelete(task)
}

// ===== 分享 =====
const shareVisible = ref(false)
const shares = ref([])
const shareExpiresDays = ref(7)
const creatingShare = ref(false)
const loadingShares = ref(false)
let shareTask = null

const loadShares = async () => {
  if (!shareTask) return
  loadingShares.value = true
  try {
    const { data } = await audiobookShareApi.listShares(shareTask.id)
    shares.value = data.data || []
  } catch (e) {
    shares.value = []
  } finally {
    loadingShares.value = false
  }
}

const openShareDialog = (task) => {
  shareTask = task
  shareVisible.value = true
  shareExpiresDays.value = 7
  loadServicePort()
  loadShares()
}

// 服务端口（飞牛 service_port）：分享链接用端口直连，不走统一网关
const servicePort = ref('')

const loadServicePort = async () => {
  try {
    const { data } = await getShareConfig()
    servicePort.value = String(data.port || '')
  } catch (e) {
    servicePort.value = ''
  }
}

// 分享链接基础：优先端口直连（http://NAS_IP:service_port/#/share/audio/xxx）
const shareBase = () => {
  const p = servicePort.value
  const { protocol, hostname, port, origin } = window.location
  if (p) {
    // 当前已在服务端口直连访问（开发环境 / 直连部署），直接用同源
    if (port === p || (!port && (p === '80' || p === '443'))) return origin
    // 飞牛统一网关场景：用 hostname + 服务端口直连
    return `${protocol}//${hostname}:${p}`
  }
  // 端口信息未获取到时回退：同源 + 网关前缀
  return `${origin}${import.meta.env.BASE_URL}`
}

// 如 http://NAS_IP:service_port/#/share/audio/xxx（端口直连，不走统一网关）
const shareUrl = (token) => `${shareBase()}#/share/audio/${token}`

const createShare = async () => {
  creatingShare.value = true
  try {
    const { data } = await audiobookShareApi.createShare(shareTask.id, shareExpiresDays.value)
    shares.value.unshift(data)
    copyShare(data.token)
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '生成分享链接失败')
  } finally {
    creatingShare.value = false
  }
}

const copyShare = async (token) => {
  const url = shareUrl(token)
  const ok = await copyText(url)
  if (ok) {
    ElMessage.success('链接已复制到剪贴板')
  } else {
    window.prompt('浏览器限制自动复制，请手动复制链接：', url)
  }
}

const revokeShare = async (s) => {
  try {
    await ElMessageBox.confirm('撤销后该链接将立即失效，确定撤销这条分享吗？', '撤销分享', { type: 'warning', confirmButtonText: '确定', cancelButtonText: '取消' })
    await audiobookShareApi.revokeShare(s.token)
    shares.value = shares.value.filter((x) => x.token !== s.token)
    ElMessage.success('已撤销分享')
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(e.response?.data?.error || '撤销失败')
  }
}

const formatDate = (d) => {
  const date = new Date(d)
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

async function doPause(task) {
  try {
    await audiobookApi.pause(task.id)
    ElMessage.success('已请求暂停，当前分段完成后生效')
    await refresh()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '暂停失败')
  }
}

async function doResume(task) {
  try {
    await audiobookApi.resume(task.id)
    ElMessage.success('已恢复生成')
    await refresh()
    if (hasActive.value) {
      startPolling()
    }
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '恢复失败')
  }
}

async function doCancel(task) {
  try {
    await audiobookApi.cancel(task.id)
    ElMessage.success('已请求取消，稍后生效')
    await refresh()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '取消失败')
  }
}

function doRegenerate(task) {
  openGenerate(task, true)
}

async function doDelete(task) {
  try {
    await ElMessageBox.confirm('将删除该有声书的全部音频文件与记录，确定删除？', '删除有声书', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消'
    })
  } catch (e) {
    return
  }
  try {
    await audiobookApi.remove(task.id)
    ElMessage.success('已删除')
    // 若删除的是正在播放的任务，关闭全局播放器
    if (playerState.task && playerState.task.id === task.id) {
      stopPlayer()
    }
    await refresh()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '删除失败')
  }
}

// ===== 生命周期 =====
onMounted(() => {
  refresh()
  // 等音色列表就绪后再打开书架跳转携带的生成弹窗，保证音色可选
  loadVoices().then(() => openFromQuery())
  if (hasActive.value) startPolling()
})
onBeforeUnmount(() => {
  // 仅停止页面轮询，不停止全局播放（后台继续播放）
  stopPolling()
})

// 供外部（如书架页）通过全局事件选择书籍生成
if (typeof window !== 'undefined') {
  window.addEventListener('audiobook-generate', (e) => {
    const book = e.detail?.book
    if (!book) return
    generateBook.value = { id: book.id, title: book.title, cover_path: book.cover_path, file_type: book.file_type }
    loadVoices()
    generateVisible.value = true
  })
}
</script>

<style scoped>
.page {
  max-width: 1400px;
  margin: 0 auto;
}

/* ===== 顶部标题区 ===== */
.page-hero {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 22px 26px;
  border-radius: 20px;
  background: var(--app-gradient);
  box-shadow: 0 14px 40px color-mix(in srgb, var(--app-primary) 30%, transparent);
  color: #fff;
  margin-bottom: 18px;
}
.hero-icon {
  flex-shrink: 0;
  width: 50px;
  height: 50px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  background: rgba(255, 255, 255, 0.2);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.25);
}
.hero-text {
  flex: 1;
  min-width: 0;
}
.hero-title {
  margin: 0;
  font-size: 22px;
  font-weight: 700;
  letter-spacing: 1px;
}
.hero-sub {
  margin: 6px 0 0;
  font-size: 13px;
  opacity: 0.85;
}
.hero-badge {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.18);
  backdrop-filter: blur(8px);
  font-size: 13px;
  font-weight: 600;
}

/* ===== 统计卡片（与书架页统一） ===== */
.stats-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
  margin-bottom: 24px;
}
.stat-card {
  position: relative;
  overflow: hidden;
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 22px 24px;
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.85);
  backdrop-filter: blur(12px);
  border: 1px solid rgba(255, 255, 255, 0.6);
  box-shadow: 0 8px 32px rgba(31, 38, 135, 0.08);
  transition: transform 0.3s ease, box-shadow 0.3s ease;
  cursor: pointer;
}
.stat-card:hover {
  transform: translateY(-4px) scale(1.02);
  box-shadow: 0 16px 40px rgba(31, 38, 135, 0.16);
}
/* 筛选激活态：主题描边高亮 */
.stat-card.active {
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--app-primary) 60%, transparent), 0 12px 32px color-mix(in srgb, var(--app-primary) 20%, transparent);
  transform: translateY(-4px) scale(1.02);
}
.stat-icon {
  flex-shrink: 0;
  width: 52px;
  height: 52px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
}
.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: #0f172a;
  line-height: 1.1;
  font-variant-numeric: tabular-nums;
}
.stat-label {
  font-size: 13px;
  color: #64748b;
  margin-top: 4px;
}
.stat-shine {
  position: absolute;
  top: -60px;
  right: -60px;
  width: 120px;
  height: 120px;
  border-radius: 50%;
  opacity: 0.08;
  transform: scale(1);
  transition: transform 0.5s ease;
}
.stat-card:hover .stat-shine {
  transform: scale(1.8);
}

/* ===== 任务列表 ===== */
.list-wrap {
  min-height: 200px;
}
.empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
  padding: 70px 20px;
  color: #94a3b8;
}
.empty p {
  margin: 0;
  font-size: 14px;
}

.list-toolbar {
  display: flex;
  justify-content: flex-end;
  margin: 4px 0 14px;
}
.task-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 14px;
}
.task-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.task-card {
  position: relative;
  min-width: 0;
  border-radius: 14px;
  padding: 16px;
  background: #fff;
  box-shadow: 0 4px 16px rgba(31, 38, 135, 0.08);
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1), box-shadow 0.3s ease;
}
.task-card::before {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: 14px;
  padding: 1.5px;
  background: linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 50%, transparent), rgba(34, 211, 238, 0.5), rgba(139, 92, 246, 0.5));
  -webkit-mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
  -webkit-mask-composite: xor;
  mask-composite: exclude;
  opacity: 0;
  transition: opacity 0.3s ease;
  pointer-events: none;
}
.task-card:hover {
  transform: translateY(-4px) scale(1.02);
  box-shadow: 0 20px 48px color-mix(in srgb, var(--app-primary) 28%, transparent);
}
.task-card:hover::before {
  opacity: 1;
}
/* 当前播放中的卡片：细主题描边高亮（与 hover 描边同粗细） */
.task-card.playing {
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--app-primary) 55%, transparent), 0 12px 32px color-mix(in srgb, var(--app-primary) 22%, transparent);
}
.task-card.playing::before {
  opacity: 1;
}

.task-head {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 14px;
}
.book-cover {
  position: relative;
  flex-shrink: 0;
  width: 52px;
  height: 68px;
  border-radius: 8px;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  background: var(--app-gradient);
  box-shadow: 0 6px 16px color-mix(in srgb, var(--app-primary) 30%, transparent);
}
.book-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 0.3s ease;
}
.task-card:hover .book-cover img {
  transform: scale(1.07);
}

/* 封面状态角标 */
.cover-badge {
  position: absolute;
  top: -6px;
  right: -6px;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  background: var(--app-gradient);
  box-shadow: 0 4px 10px rgba(99, 102, 241, 0.4);
  z-index: 2;
}
.cover-badge.done {
  background: linear-gradient(135deg, #34d399, #10b981);
  box-shadow: 0 4px 10px rgba(16, 185, 129, 0.4);
}
.cover-badge.paused {
  background: linear-gradient(135deg, #fbbf24, #f59e0b);
  box-shadow: 0 4px 10px rgba(245, 158, 11, 0.4);
}
/* 播放中角标：主题色 + 呼吸光圈 */
.cover-badge.playing {
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  box-shadow: 0 4px 10px rgba(99, 102, 241, 0.4);
  animation: playingPulse 1.6s ease-in-out infinite;
}
@keyframes playingPulse {
  0%, 100% { box-shadow: 0 0 0 0 rgba(99, 102, 241, 0.45); }
  50% { box-shadow: 0 0 0 6px rgba(99, 102, 241, 0); }
}
.cover-badge .eq-bar-wrap {
  transform: scale(0.72);
}
.cover-badge .eq-bar {
  background: #fff;
}
.pulse-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #fbbf24;
  animation: pulseDot 1.2s ease-in-out infinite;
}
@keyframes pulseDot {
  0%, 100% { transform: scale(1); opacity: 1; }
  50% { transform: scale(1.5); opacity: 0.5; }
}
.task-info {
  flex: 1;
  min-width: 0;
}
.book-title {
  font-size: 15px;
  font-weight: 700;
  color: #1e293b;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-bottom: 6px;
}
.book-meta {
  display: flex;
  gap: 6px;
  align-items: center;
}
.fmt-tag {
  border: none;
  color: #64748b;
  background: #f1f5f9;
}
.task-more {
  flex-shrink: 0;
  color: #94a3b8;
  cursor: pointer;
}
.more-icon:hover {
  color: var(--app-primary);
}

.task-body {
  padding-top: 2px;
}
.progress-row {
  display: flex;
  align-items: center;
  min-width: 0;
}
.progress-row :deep(.el-progress) {
  flex: 1;
  min-width: 0;
}
/* 进度条左侧生成中标识：旋转圆环（区别于播放中的均衡条） */
.gen-spinner {
  flex-shrink: 0;
  width: 16px;
  height: 16px;
  margin-right: 8px;
  border-radius: 50%;
  border: 2.5px solid color-mix(in srgb, var(--app-primary) 22%, transparent);
  border-top-color: var(--app-primary);
  animation: genSpin 0.9s linear infinite;
}
@keyframes genSpin {
  to { transform: rotate(360deg); }
}
.progress-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 8px;
  font-size: 12px;
  color: #64748b;
}
.progress-meta > span {
  min-width: 0;
}
.voice-name {
  display: flex;
  align-items: center;
  gap: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 45%;
  min-width: 0;
  color: #94a3b8;
}

/* 状态标签美化 */
.status-tag {
  border: none;
  font-weight: 600;
}
.tag-done {
  background: linear-gradient(135deg, #d1fae5, #a7f3d0) !important;
  color: #059669 !important;
}
.tag-processing {
  background: linear-gradient(135deg, #e0e7ff, #c7d2fe) !important;
  color: #4f46e5 !important;
}
.tag-spin {
  animation: tagSpin 1s linear infinite;
}
@keyframes tagSpin {
  to { transform: rotate(360deg); }
}

.done-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 14px;
  align-items: center;
  font-size: 13px;
  color: #64748b;
  margin-bottom: 12px;
}
.done-item {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
}
.task-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

/* 列表视图：横向紧凑布局，状态信息与操作按钮移到底部右侧一行，整体不再过高 */
.task-list .task-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 12px 16px;
}
/* 列表视图下悬停不缩放宽度，避免整行抖动 */
.task-list .task-card:hover {
  transform: translateY(-2px);
}
.task-list .task-head {
  flex: 1 1 auto;
  min-width: 0;
  margin-bottom: 0;
}
/* 状态信息 + 操作按钮整体靠右，按钮在最右 */
.task-list .task-body {
  flex: 0 0 auto;
  margin: 0;
  padding: 0;
  display: flex;
  align-items: center;
  gap: 16px;
}
/* 进度条在列表视图下固定窄宽，横向排列不撑高 */
.task-list .progress-row {
  flex: 0 0 160px;
  width: 160px;
}
.task-list .progress-meta {
  margin-top: 0;
}
.task-list .done-meta {
  margin-bottom: 0;
}
/* 覆盖处理中文内联 margin-top，保持横向对齐 */
.task-list .task-actions {
  flex-shrink: 0;
  margin-top: 0 !important;
}
.task-list .error-text {
  margin-bottom: 0;
  max-width: 320px;
}
.error-text {
  font-size: 12px;
  color: #f43f5e;
  margin-bottom: 10px;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.gradient-btn {
  background: var(--app-gradient) !important;
  border: none !important;
  box-shadow: 0 6px 18px color-mix(in srgb, var(--app-primary) 35%, transparent);
}

/* "开始聆听"按钮流光扫过动效 */
.shine-btn {
  position: relative;
  overflow: hidden;
}
.shine-btn::after {
  content: '';
  position: absolute;
  top: 0;
  left: -80%;
  width: 45%;
  height: 100%;
  background: linear-gradient(120deg, transparent, rgba(255, 255, 255, 0.5), transparent);
  transform: skewX(-20deg);
  transition: left 0.5s ease;
}
.shine-btn:hover::after {
  left: 130%;
}

/* 当前播放中的书块：正在播放状态提示（替代"开始聆听"按钮） */
.now-playing-tip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 7px 14px;
  border-radius: 999px;
  font-size: 13px;
  font-weight: 600;
  color: var(--app-primary);
  background: color-mix(in srgb, var(--app-primary) 10%, #fff);
  border: 1px solid color-mix(in srgb, var(--app-primary) 22%, transparent);
}
.now-playing-tip .eq-bar-wrap {
  height: 12px;
}

/* ===== 生成弹窗 ===== */
.gen-desc {
  display: flex;
  gap: 14px;
  align-items: center;
  padding: 12px 14px;
  border-radius: 12px;
  background: #f1f5f9;
  margin-bottom: 16px;
}
.gen-cover {
  width: 42px;
  height: 56px;
  border-radius: 6px;
  object-fit: cover;
  box-shadow: 0 4px 12px rgba(31, 38, 135, 0.18);
}
.gen-book-title {
  font-size: 14px;
  font-weight: 700;
  color: #1e293b;
  margin-bottom: 4px;
}
.gen-book-sub {
  font-size: 12px;
  color: #94a3b8;
}
.speed-row {
  display: flex;
  align-items: center;
  gap: 14px;
  width: 100%;
}
.speed-val {
  flex-shrink: 0;
  font-size: 14px;
  font-weight: 700;
  color: var(--app-primary);
  min-width: 34px;
  text-align: right;
}
.voice-empty {
  font-size: 12px;
  color: #94a3b8;
  padding: 8px;
}
.engine-opt-label {
  font-size: 14px;
  color: var(--el-text-color-primary);
}
.engine-opt-desc {
  display: block;
  margin-top: 2px;
  font-size: 12px;
  color: #94a3b8;
  white-space: normal;
  line-height: 1.4;
}
.engine-opt-tip {
  margin-top: 6px;
  font-size: 12px;
  line-height: 1.5;
  color: #94a3b8;
}

/* ===== 播放弹窗 ===== */
.play-wrap {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.segment-list {
  max-height: 46vh;
  overflow-y: auto;
  border-radius: 12px;
  border: 1px solid #eef2f7;
  background: #f8fafc;
  padding: 6px;
}
.segment-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 10px;
  cursor: pointer;
  transition: background 0.15s ease;
}
.segment-item:hover {
  background: rgba(99, 102, 241, 0.07);
}
.segment-item.active {
  background: linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 12%, transparent), color-mix(in srgb, var(--app-primary) 4%, transparent));
}
/* 未生成分段（边生成边播放） */
.segment-item.pending {
  opacity: 0.55;
  cursor: not-allowed;
}
.segment-item.pending:hover {
  background: transparent;
}
.seg-pending {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: #94a3b8;
}

/* 边生成边播放状态提示条 */
.play-status {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  margin-bottom: 10px;
  border-radius: 10px;
  font-size: 12px;
  color: #4f46e5;
  background: linear-gradient(135deg, #eef2ff, #ecfeff);
  border: 1px solid rgba(99, 102, 241, 0.22);
}
.play-status .status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #6366f1;
  animation: pulseDot 1.2s ease-in-out infinite;
}
.waiting-tip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  margin-left: 8px;
  font-size: 12px;
  font-weight: 500;
  color: #6366f1;
}
.seg-left {
  display: flex;
  gap: 10px;
  align-items: center;
  min-width: 0;
  flex: 1;
}
.seg-idx {
  flex-shrink: 0;
  width: 30px;
  height: 30px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 700;
  color: #6366f1;
  background: #eef2ff;
}
.segment-item.active .seg-idx {
  color: #fff;
  background: var(--app-gradient);
}
.seg-info {
  min-width: 0;
}
.seg-title {
  font-size: 13px;
  font-weight: 600;
  color: #334155;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.seg-text {
  font-size: 12px;
  color: #94a3b8;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-top: 2px;
}
.seg-right {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}
.seg-dur {
  font-size: 12px;
  color: #94a3b8;
}
.seg-play {
  color: #94a3b8;
}
.segment-item.active .seg-play {
  color: var(--app-primary);
}

/* 播放中动态均衡条 */
.eq-bar-wrap {
  display: flex;
  align-items: flex-end;
  gap: 2px;
  height: 14px;
}
.eq-bar {
  width: 3px;
  border-radius: 2px;
  background: var(--app-primary);
  animation: eq 0.9s ease-in-out infinite;
}
.eq-bar:nth-child(2) { animation-delay: 0.2s; }
.eq-bar:nth-child(3) { animation-delay: 0.4s; }
@keyframes eq {
  0%, 100% { height: 4px; }
  50% { height: 14px; }
}

/* 播放控制条 */
.player-bar {
  border-radius: 14px;
  padding: 12px 16px;
  background: linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 10%, #fff), #fff);
  border: 1px solid color-mix(in srgb, var(--app-primary) 14%, transparent);
}
.player-title {
  font-size: 13px;
  font-weight: 600;
  color: #334155;
  text-align: center;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-bottom: 10px;
}
.player-title .waiting-tip {
  vertical-align: middle;
}
.player-controls {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 26px;
}
.ctrl-btn {
  color: #64748b;
  cursor: pointer;
  transition: color 0.2s;
}
.ctrl-btn:hover:not(.disabled) {
  color: var(--app-primary);
}
.ctrl-btn.disabled {
  opacity: 0.3;
  cursor: not-allowed;
}
.play-btn {
  width: 52px;
  height: 52px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  background: var(--app-gradient);
  box-shadow: 0 8px 22px color-mix(in srgb, var(--app-primary) 40%, transparent);
  cursor: pointer;
  transition: transform 0.2s cubic-bezier(0.34, 1.56, 0.64, 1);
}
.play-btn:active {
  transform: scale(0.9);
}
.player-progress {
  margin-top: 10px;
  cursor: pointer;
}
.pbar {
  height: 6px;
  border-radius: 3px;
  background: #e2e8f0;
  overflow: hidden;
}
.pbar-fill {
  height: 100%;
  border-radius: 3px;
  background: var(--app-gradient);
  transition: width 0.2s linear;
}
.ptime {
  display: block;
  text-align: right;
  font-size: 11px;
  color: #94a3b8;
  margin-top: 4px;
}

.hidden-audio {
  display: none;
}

/* 响应式 */
@media (max-width: 768px) {
  .stats-row {
    grid-template-columns: repeat(2, 1fr);
    gap: 12px;
  }
  .stat-card {
    padding: 14px 12px;
    gap: 10px;
  }
  .stat-icon {
    width: 40px;
    height: 40px;
    border-radius: 12px;
  }
  .stat-value {
    font-size: 22px;
  }
  .task-grid {
    grid-template-columns: minmax(0, 1fr);
  }
  /* 顶部标题区：手机端紧凑横排，副标题自然折行，badge 靠右不换行挤压 */
  .page-hero {
    padding: 14px 16px;
    flex-direction: row;
    align-items: center;
    gap: 10px;
    border-radius: 16px;
    margin-bottom: 12px;
  }
  .hero-text {
    flex: 1;
    min-width: 0;
  }
  .hero-icon {
    width: 42px;
    height: 42px;
    border-radius: 12px;
  }
  .hero-title {
    font-size: 18px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .hero-sub {
    font-size: 12px;
    line-height: 1.5;
    margin-top: 3px;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    overflow: hidden;
  }
  .hero-badge {
    padding: 5px 10px;
    font-size: 12px;
    gap: 4px;
  }
  .play-hint {
    margin-top: 6px;
  }
}

/* ===== 分享弹窗 ===== */
.share-desc {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 12px 14px;
  margin-bottom: 16px;
  border-radius: 12px;
  font-size: 13px;
  line-height: 1.7;
  color: #475569;
  background: linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 7%, transparent), rgba(34, 211, 238, 0.05));
  border: 1px solid color-mix(in srgb, var(--app-primary) 15%, transparent);
}

.share-desc .el-icon {
  margin-top: 3px;
  flex-shrink: 0;
  color: var(--app-primary);
}

.share-create {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 16px;
}

.share-list {
  min-height: 60px;
  max-height: 300px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.share-empty {
  padding: 24px;
  text-align: center;
  font-size: 13px;
  color: #94a3b8;
  border: 1px dashed color-mix(in srgb, var(--app-primary) 25%, transparent);
  border-radius: 12px;
}

.share-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  border-radius: 12px;
  background: color-mix(in srgb, var(--app-primary) 4%, transparent);
  border: 1px solid color-mix(in srgb, var(--app-primary) 12%, transparent);
  transition: background 0.2s ease;
}

.share-item:hover {
  background: color-mix(in srgb, var(--app-primary) 8%, transparent);
}

.share-item-info {
  min-width: 0;
  flex: 1;
}

.share-item-link {
  font-size: 13px;
  font-weight: 500;
  color: var(--app-primary);
  word-break: break-all;
  margin-bottom: 6px;
}

.share-item-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.share-item-time {
  font-size: 12px;
  color: #94a3b8;
}

.share-item-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.share-item-actions .el-button + .el-button {
  margin-left: 0;
}
</style>
