<template>
  <div class="stats-page">
    <div class="bg-glow bg-glow-1"></div>
    <div class="bg-glow bg-glow-2"></div>

    <!-- 顶部统计卡 -->
    <div class="stats-row">
      <div v-for="(s, index) in statCards" :key="s.label" class="stat-card" :style="{ '--i': index }">
        <div class="stat-icon" :style="{ background: s.gradient, boxShadow: `0 10px 24px ${s.glow}` }">
          <el-icon :size="20"><component :is="s.icon" /></el-icon>
        </div>
        <div class="stat-body">
          <div class="stat-value">{{ s.value }}</div>
          <div class="stat-label">{{ s.label }}</div>
        </div>
      </div>
    </div>

    <!-- 主体：日历 + 右侧面板 -->
    <div class="stats-main">
      <!-- 左：日历 -->
      <div class="calendar-panel">
        <div class="panel-title">
          <span class="title-dot"></span>
          阅读日历
          <span class="title-sub">每日阅读时长</span>
        </div>
        <el-calendar v-model="selectedDate" ref="calendarRef" class="read-calendar">
          <template #header="{ date }">
            <span class="el-calendar__title">{{ fmtCalTitle(date) }}</span>
            <div class="el-calendar__button-group">
              <el-button-group>
                <el-button size="small" :class="{ 'is-active': calActive === 'prev' }" @click="setCalAction('prev')">上月</el-button>
                <el-button size="small" :class="{ 'is-active': calActive === 'today' }" @click="setCalAction('today')">今天</el-button>
                <el-button size="small" :class="{ 'is-active': calActive === 'next' }" @click="setCalAction('next')">下月</el-button>
              </el-button-group>
            </div>
          </template>
          <template #date-cell="{ data }">
            <div class="cal-cell" :class="cellClass(data.day)">
              <span class="cal-day">{{ Number(data.day.split('-')[2]) }}</span>
              <template v-if="dayMap[data.day]">
                <span class="cal-duration">{{ fmtShort(dayMap[data.day].duration) }}</span>
                <span v-if="dayMap[data.day].completed" class="cal-done"><el-icon><Check /></el-icon></span>
              </template>
              <template v-else-if="isToday(data.day)">
                <span class="cal-today-tag">今天</span>
              </template>
            </div>
          </template>
        </el-calendar>
        <div class="legend">
          <span class="legend-item"><i class="lg-bar lg-0"></i>未读</span>
          <span class="legend-item"><i class="lg-bar lg-1"></i>&lt;30分</span>
          <span class="legend-item"><i class="lg-bar lg-2"></i>30-60分</span>
          <span class="legend-item"><i class="lg-bar lg-3"></i>&gt;60分</span>
          <span class="legend-item"><i class="lg-check"></i>达标</span>
        </div>
        <!-- 最近 7 天 -->
        <div class="recent7">
          <div class="panel-title small">
            <span class="title-dot cyan"></span>
            最近 7 天
          </div>
          <div class="week-bars">
            <div v-for="d in recent7" :key="d.date" class="week-bar-col">
              <div class="week-bar" :class="weekBarClass(d)">
                <span class="week-min">{{ fmtShort(d.duration) }}</span>
              </div>
              <span class="week-day">{{ weekDayName(d.date) }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 右：当日详情 + 计划 -->
      <div class="side-panel">
        <!-- 当日详情 -->
        <div class="side-card">
          <div class="panel-title">
            <span class="title-dot green"></span>
            {{ selectedDateLabel }} 阅读记录
          </div>
          <div v-if="dailyLoading" class="side-loading">
            <el-icon class="is-loading"><Loading /></el-icon>
          </div>
          <template v-else-if="dailyRecords.length > 0">
            <div class="daily-total">
              <span class="dt-value">{{ fmtDuration(dailyTotal) }}</span>
              <span class="dt-label">
                当天共 {{ dailyRecords.length }} 次
                <template v-if="dailyTts > 0"> · 其中朗读 {{ fmtDuration(dailyTts) }}</template>
              </span>
              <span v-if="dayMap[selectedDate]?.completed" class="dt-goal">
                <el-icon><CircleCheck /></el-icon> 达标
              </span>
            </div>
            <div v-for="r in dailyRecords" :key="r.id" class="daily-item" @click="$router.push('/book/' + r.book_id)">
              <div class="di-cover">
                <img v-if="r.book_cover" :src="bookApi.coverUrl(r.book_id)" />
                <div v-else class="di-cover-ph"><el-icon><Reading /></el-icon></div>
              </div>
              <div class="di-body">
                <div class="di-title">{{ r.book_title }}</div>
                <div class="di-meta">
                  <span v-if="r.type === 'tts'" class="di-tts-badge">朗读</span>
                  <span class="di-time">{{ fmtClock(r.start_time) }}</span>
                  <span class="di-dur">{{ fmtDuration(r.duration) }}</span>
                  <span v-if="r.type === 'tts' && r.engine" class="di-engine">{{ engineLabel(r.engine) }}</span>
                  <span class="di-prog" v-if="r.progress > 0">{{ Math.round(r.progress) }}%</span>
                </div>
              </div>
            </div>
          </template>
          <div v-else class="side-empty">
            <el-icon :size="32"><Sunny /></el-icon>
            <p>这一天没有阅读记录</p>
            <p v-if="!isToday(selectedDate)" class="side-empty-sub">点击日历其他日期查看</p>
          </div>
        </div>

        <!-- 阅读计划 -->
        <div class="side-card">
          <div class="panel-title">
            <span class="title-dot orange"></span>
            阅读计划
            <el-button size="small" type="primary" round class="add-plan-btn" @click="openPlanDialog">
              <el-icon><Plus /></el-icon> 新建
            </el-button>
          </div>
          <div v-if="plans.length === 0" class="side-empty">
            <el-icon :size="32"><Calendar /></el-icon>
            <p>还没有阅读计划</p>
            <p class="side-empty-sub">创建计划设定每日阅读目标</p>
          </div>
          <div v-else class="plan-list">
            <div v-for="p in plans" :key="p.id" class="plan-item" :class="{ inactive: !p.active }">
              <div class="plan-head">
                <span class="plan-name">{{ p.name }}</span>
                <el-switch
                  v-model="p.active"
                  size="small"
                  @change="togglePlan(p)"
                />
              </div>
              <div class="plan-meta">
                <span class="plan-goal"><el-icon><Timer /></el-icon> {{ p.daily_goal_min }} 分钟/天</span>
                <span class="plan-date">{{ p.start_date }} ~ {{ p.end_date }}</span>
              </div>
              <div class="plan-progress" v-if="planStats[p.id]">
                <el-progress
                  :percentage="planStats[p.id].percent"
                  :stroke-width="6"
                  :show-text="false"
                  color="linear-gradient(90deg,#f59e0b,#fbbf24)"
                />
                <span class="plan-stats-text">
                  达标 {{ planStats[p.id].done }} / {{ planStats[p.id].total }} 天
                </span>
              </div>
              <div class="plan-actions">
                <el-button size="small" text @click="editPlan(p)">编辑</el-button>
                <el-button size="small" text type="danger" @click="removePlan(p.id)">删除</el-button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 计划编辑对话框 -->
    <el-dialog v-model="planDialogVisible" :title="editingPlan ? '编辑计划' : '新建阅读计划'" width="480px">
      <el-form :model="planForm" label-width="96px">
        <el-form-item label="计划名称" required>
          <el-input v-model="planForm.name" placeholder="如：本月读完《百年孤独》" />
        </el-form-item>
        <el-form-item label="每日目标" required>
          <el-input-number v-model="planForm.daily_goal_min" :min="5" :max="480" :step="5" />
          <span class="form-unit">分钟/天</span>
        </el-form-item>
        <el-form-item label="开始日期">
          <el-date-picker v-model="planForm.start_date" type="date" value-format="YYYY-MM-DD" placeholder="选择日期" />
        </el-form-item>
        <el-form-item label="结束日期">
          <el-date-picker v-model="planForm.end_date" type="date" value-format="YYYY-MM-DD" placeholder="选择日期" />
        </el-form-item>
        <el-form-item label="关联图书" v-if="books.length > 0">
          <el-select v-model="planForm.book_id" clearable placeholder="可选，关联某本图书" style="width: 100%">
            <el-option v-for="b in books" :key="b.id" :label="b.title" :value="b.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="planDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="savePlan">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { bookApi } from '@/api/book'

// ===== 状态 =====
const selectedDate = ref(new Date())
const calendarRef = ref(null)
// 当前激活的日历操作按钮（上月/今天/下月），用于选中态样式
const calActive = ref('')
const setCalAction = (act) => {
  calActive.value = act
  calendarRef.value?.selectDate(act === 'prev' ? 'prev-month' : act === 'next' ? 'next-month' : 'today')
}
const fmtCalTitle = (d) => {
  const date = d instanceof Date ? d : new Date(d)
  // 无效日期（undefined / null / 非法字符串）会让 getFullYear() 返回 NaN，
  // 界面上表现为 "NaN年NaN月"，这里回退到当前月份
  if (!date || Number.isNaN(date.getTime())) {
    const now = new Date()
    return `${now.getFullYear()}年${now.getMonth() + 1}月`
  }
  return `${date.getFullYear()}年${date.getMonth() + 1}月`
}
const fmtLocal = (d) =>
  `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
const todayStr = ref(fmtLocal(new Date()))
const calendar = ref(null) // { year, month, days }
const overview = ref({})
const dailyRecords = ref([])
const dailyTotal = ref(0)
const dailyTts = ref(0) // 当天朗读时长（秒）
const dailyLoading = ref(false)
const plans = ref([])
const books = ref([])
const planStats = ref({})

const planDialogVisible = ref(false)
const editingPlan = ref(null)
const planForm = ref({})

// ===== 派生 =====
const selectedDateLabel = computed(() => {
  const d = new Date(selectedDate.value)
  // 同上：无效日期回退到今天，避免出现 "NaN年NaN月NaN日"
  if (Number.isNaN(d.getTime())) {
    const now = new Date()
    return `${now.getFullYear()}年${now.getMonth() + 1}月${now.getDate()}日`
  }
  return `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日`
})

const dayMap = computed(() => {
  const m = {}
  calendar.value?.days?.forEach(d => { m[d.date] = d })
  return m
})

const recent7 = computed(() => overview.value.recent_7_days || [])

const statCards = computed(() => {
  const o = overview.value || {}
  const goal = o.daily_goal_min || 0
  return [
    {
      label: goal > 0 ? `本月阅读（目标 ${goal}分/天）` : '本月阅读',
      value: fmtDuration(o.month_duration),
      icon: 'Calendar',
      gradient: 'linear-gradient(135deg,var(--app-primary),#8b5cf6)',
      glow: 'rgba(99,102,241,0.45)'
    },
    {
      label: '连续阅读',
      value: `${o.current_streak || 0} 天`,
      icon: 'Sunny',
      gradient: 'linear-gradient(135deg,#f59e0b,#fbbf24)',
      glow: 'rgba(245,158,11,0.45)'
    },
    {
      label: '累计阅读',
      value: fmtDuration(o.total_duration),
      icon: 'Clock',
      gradient: 'linear-gradient(135deg,#06b6d4,#22d3ee)',
      glow: 'rgba(6,182,212,0.45)'
    },
    {
      label: '本月达标',
      value: `${o.month_goal_days || 0} 天`,
      icon: 'CircleCheck',
      gradient: 'linear-gradient(135deg,#10b981,#34d399)',
      glow: 'rgba(16,185,129,0.45)'
    }
  ]
})

// ===== 工具函数 =====
const fmtDuration = (sec) => {
  sec = sec || 0
  if (sec < 60) return `${sec} 秒`
  if (sec < 3600) return `${Math.floor(sec / 60)} 分钟`
  const h = Math.floor(sec / 3600)
  const m = Math.floor((sec % 3600) / 60)
  return m > 0 ? `${h} 小时 ${m} 分` : `${h} 小时`
}

const fmtShort = (sec) => {
  sec = sec || 0
  if (sec < 60) return `${sec}s`
  if (sec < 3600) return `${Math.floor(sec / 60)}m`
  return `${(sec / 3600).toFixed(1)}h`
}

const fmtClock = (t) => {
  if (!t) return ''
  const d = new Date(t)
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

// 朗读引擎中文名
const engineLabel = (e) => (e === 'browser' ? '浏览器离线' : e === 'edge-tts' ? '云端' : e || '')

const isToday = (dayStr) => dayStr === todayStr.value

const weekDayName = (dateStr) => {
  const d = new Date(dateStr + 'T00:00:00')
  const names = ['日', '一', '二', '三', '四', '五', '六']
  return names[d.getDay()]
}

// 日历格子颜色分级
const cellClass = (day) => {
  const info = dayMap.value[day]
  if (!info || info.duration <= 0) return isToday(day) ? 'today' : ''
  if (info.duration < 30 * 60) return 'lv-1'
  if (info.duration < 60 * 60) return 'lv-2'
  return 'lv-3'
}

const weekBarClass = (d) => {
  if (!d.duration || d.duration <= 0) return ''
  if (d.duration < 30 * 60) return 'bar-1'
  if (d.duration < 60 * 60) return 'bar-2'
  return 'bar-3'
}

// ===== 数据加载 =====
const loadCalendar = async (year, month) => {
  try {
    const { data } = await bookApi.getCalendarStats(year, month)
    calendar.value = data
  } catch (e) {
    // 列表加载失败不弹错误提醒（无阅读记录属正常情况），静默降级为空日历
    console.warn('加载日历统计失败', e)
  }
}

const loadOverview = async () => {
  try {
    const { data } = await bookApi.getStatsOverview()
    overview.value = data
  } catch (e) {
    // ignore
  }
}

const loadDaily = async () => {
  const key = fmtLocal(new Date(selectedDate.value))
  dailyLoading.value = true
  try {
    const { data } = await bookApi.getDailyStats(key)
    dailyRecords.value = data.records || []
    dailyTotal.value = data.total || 0
    dailyTts.value = data.tts_total || 0
  } catch (e) {
    dailyRecords.value = []
    dailyTotal.value = 0
    dailyTts.value = 0
  } finally {
    dailyLoading.value = false
  }
}

const loadPlans = async () => {
  try {
    const { data } = await bookApi.listPlans()
    plans.value = data.data || []
    computePlanStats()
  } catch (e) {
    // ignore
  }
}

// 计算每个计划在其日期范围内的达标天数
const computePlanStats = () => {
  planStats.value = {}
  const days = calendar.value?.days || []
  plans.value.forEach(p => {
    const start = p.start_date || '0000-00-00'
    const end = p.end_date || '9999-12-31'
    const inRange = days.filter(d => d.date >= start && d.date <= end)
    const done = inRange.filter(d => d.completed).length
    const total = inRange.length
    planStats.value[p.id] = {
      done,
      total,
      percent: total > 0 ? Math.round((done / total) * 100) : 0
    }
  })
}

// ===== 交互 =====
watch(selectedDate, () => {
  loadDaily()
  const d = selectedDate.value
  loadCalendar(d.getFullYear(), d.getMonth() + 1)
})

// 监听月份切换（el-calendar 切换时 calendar.value 会变化）
watch(() => calendar.value?.month, () => {
  computePlanStats()
})

const openPlanDialog = () => {
  editingPlan.value = null
  planForm.value = {
    name: '',
    daily_goal_min: 30,
    start_date: todayStr.value,
    end_date: '',
    book_id: null
  }
  planDialogVisible.value = true
}

const editPlan = (p) => {
  editingPlan.value = p
  planForm.value = {
    name: p.name,
    daily_goal_min: p.daily_goal_min,
    start_date: p.start_date,
    end_date: p.end_date,
    book_id: p.book_id || null
  }
  planDialogVisible.value = true
}

const savePlan = async () => {
  if (!planForm.value.name || !planForm.value.daily_goal_min) {
    ElMessage.warning('请填写计划名称和每日目标')
    return
  }
  const payload = {
    name: planForm.value.name,
    daily_goal_min: planForm.value.daily_goal_min,
    start_date: planForm.value.start_date,
    end_date: planForm.value.end_date,
    book_id: planForm.value.book_id
  }
  try {
    if (editingPlan.value) {
      await bookApi.updatePlan(editingPlan.value.id, payload)
    } else {
      await bookApi.createPlan(payload)
    }
    ElMessage.success('保存成功')
    planDialogVisible.value = false
    loadPlans()
    loadCalendar(selectedDate.value.getFullYear(), selectedDate.value.getMonth() + 1)
    loadOverview()
  } catch (e) {
    ElMessage.error('保存失败')
  }
}

const togglePlan = async (p) => {
  try {
    await bookApi.updatePlan(p.id, { active: p.active })
    loadPlans()
    loadOverview()
    loadCalendar(selectedDate.value.getFullYear(), selectedDate.value.getMonth() + 1)
  } catch (e) {
    p.active = !p.active
    ElMessage.error('操作失败')
  }
}

const removePlan = async (id) => {
  try {
    await bookApi.deletePlan(id)
    ElMessage.success('已删除')
    loadPlans()
    loadOverview()
    loadCalendar(selectedDate.value.getFullYear(), selectedDate.value.getMonth() + 1)
  } catch (e) {
    ElMessage.error('删除失败')
  }
}

// 加载图书列表（计划关联图书用）
const loadBooks = async () => {
  try {
    const { data } = await bookApi.listBooks({ page: 1, page_size: 100 })
    books.value = data.data || []
  } catch (e) {
    // ignore
  }
}

onMounted(() => {
  const d = new Date()
  loadCalendar(d.getFullYear(), d.getMonth() + 1)
  loadOverview()
  loadDaily()
  loadPlans()
  loadBooks()
})
</script>

<style scoped>
.stats-page {
  position: relative;
  max-width: 1400px;
  margin: 0 auto;
}

.bg-glow {
  position: fixed;
  border-radius: 50%;
  filter: blur(90px);
  pointer-events: none;
  z-index: 0;
}
.bg-glow-1 {
  width: 460px;
  height: 460px;
  background: radial-gradient(circle, color-mix(in srgb, var(--app-primary) 16%, transparent), transparent 70%);
  top: -100px;
  left: -100px;
  animation: float 12s ease-in-out infinite;
}
.bg-glow-2 {
  width: 400px;
  height: 400px;
  background: radial-gradient(circle, rgba(34, 211, 238, 0.12), transparent 70%);
  bottom: -80px;
  right: -80px;
  animation: float 14s ease-in-out infinite reverse;
}
@keyframes float {
  0%, 100% { transform: translate(0, 0); }
  50% { transform: translate(30px, -20px); }
}

.stats-row,
.stats-main {
  position: relative;
  z-index: 1;
}

/* ===== 统计卡 ===== */
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
  animation: statIn 0.5s cubic-bezier(0.22, 1, 0.36, 1) both;
  animation-delay: calc(var(--i, 0) * 60ms);
}

@keyframes statIn {
  from {
    opacity: 0;
    transform: translateY(16px) scale(0.96);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

.stat-card:hover {
  transform: translateY(-4px) scale(1.02);
  box-shadow: 0 16px 40px rgba(31, 38, 135, 0.16);
}

.stat-icon {
  flex-shrink: 0;
  width: 48px;
  height: 48px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
}

.stat-value {
  font-size: 24px;
  font-weight: 700;
  color: #0f172a;
  line-height: 1.2;
  font-variant-numeric: tabular-nums;
}

.stat-label {
  font-size: 12px;
  color: #64748b;
  margin-top: 4px;
}

/* ===== 主体布局 ===== */
/* minmax(0, 1fr)：grid 轨道最小宽度归零，防止日历表格等固有宽度撑破容器 */
.stats-main {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 360px;
  gap: 20px;
  align-items: start;
}

.calendar-panel,
.side-card {
  background: rgba(255, 255, 255, 0.88);
  backdrop-filter: blur(14px);
  border: 1px solid rgba(255, 255, 255, 0.7);
  box-shadow: 0 12px 40px rgba(31, 38, 135, 0.1);
  border-radius: 18px;
  padding: 20px;
  flex-shrink: 0;
  min-width: 0; /* 允许收缩，防止内容固有宽度撑破 grid 轨道 */
  animation: statIn 0.5s cubic-bezier(0.22, 1, 0.36, 1) both;
  animation-delay: 200ms;
}

.stats-main > .calendar-panel {
  animation-delay: 240ms;
}

.side-panel {
  position: sticky;
  top: 0;
  max-height: calc(100vh - 100px); /* 减去 header 60px 与主区域上下 padding 40px */
  display: flex;
  flex-direction: column;
  gap: 20px;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding-right: 4px;
  scrollbar-width: thin;
  scrollbar-color: color-mix(in srgb, var(--app-primary) 35%, transparent) transparent;
}

.side-panel::-webkit-scrollbar {
  width: 6px;
}
.side-panel::-webkit-scrollbar-track {
  background: transparent;
}
.side-panel::-webkit-scrollbar-thumb {
  background: color-mix(in srgb, var(--app-primary) 30%, transparent);
  border-radius: 3px;
}
.side-panel::-webkit-scrollbar-thumb:hover {
  background: color-mix(in srgb, var(--app-primary) 55%, transparent);
}

.panel-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 700;
  color: #0f172a;
  margin-bottom: 16px;
}

.panel-title.small {
  font-size: 13px;
  margin-bottom: 12px;
}

.title-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--app-primary), color-mix(in srgb, var(--app-primary) 55%, #8b5cf6));
  box-shadow: 0 0 8px color-mix(in srgb, var(--app-primary) 60%, transparent);
}
.title-dot.cyan {
  background: linear-gradient(135deg, #06b6d4, #22d3ee);
  box-shadow: 0 0 8px rgba(6, 182, 212, 0.6);
}
.title-dot.green {
  background: linear-gradient(135deg, #10b981, #34d399);
  box-shadow: 0 0 8px rgba(16, 185, 129, 0.6);
}
.title-dot.orange {
  background: linear-gradient(135deg, #f59e0b, #fbbf24);
  box-shadow: 0 0 8px rgba(245, 158, 11, 0.6);
}

.title-sub {
  font-size: 12px;
  font-weight: 400;
  color: #94a3b8;
  margin-left: auto;
}

/* ===== 日历 ===== */
.read-calendar {
  background: transparent;
  width: 100%;
  min-width: 0;
}

/* 日历表格固定布局 + 宽度 100%：7 列均分，窄屏不再被列内容撑宽 */
.read-calendar :deep(.el-calendar__body) {
  min-width: 0;
  padding: 0;
}

.read-calendar :deep(.el-calendar-table) {
  table-layout: fixed;
  width: 100%;
  border-collapse: collapse;
}

.read-calendar :deep(.el-calendar__header) {
  border-bottom: 1px solid color-mix(in srgb, var(--app-primary) 15%, transparent);
  padding: 4px 0 14px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px;
}

.read-calendar :deep(.el-calendar__title) {
  font-size: 16px;
  font-weight: 700;
  color: #0f172a;
  white-space: nowrap;
}

.read-calendar :deep(.el-button-group) {
  border-radius: 8px;
}

.read-calendar :deep(.el-button-group .el-button) {
  border-radius: 8px;
  margin-left: 4px;
  border-color: #e2e8f0;
  color: #475569;
  background: transparent;
}

.read-calendar :deep(.el-button-group .el-button:hover) {
  color: var(--app-primary);
  border-color: var(--app-primary);
  background: transparent;
}

.read-calendar :deep(.el-button-group .el-button.is-active) {
  color: var(--app-primary);
  border-color: var(--app-primary);
  background: transparent;
}

.read-calendar :deep(.el-calendar-table) {
  border-top: none;
}

.read-calendar :deep(.el-calendar-table .el-calendar-day) {
  height: 76px;
  padding: 4px;
  min-width: 0;
  overflow: hidden;
  border-color: color-mix(in srgb, var(--app-primary) 8%, transparent);
  transition: background 0.2s;
}

/* 表头（日一二三四五六）同样收缩，避免撑宽 */
.read-calendar :deep(.el-calendar-table thead th) {
  padding: 6px 2px;
  font-size: 12px;
  color: #64748b;
}

.read-calendar :deep(.el-calendar-table .el-calendar-day:hover) {
  background: color-mix(in srgb, var(--app-primary) 6%, transparent);
}

.read-calendar :deep(.el-calendar-table td.is-today) {
  background: color-mix(in srgb, var(--app-primary) 8%, transparent);
}

.read-calendar :deep(.el-calendar-table td.is-selected) {
  background: linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 14%, transparent), rgba(34, 211, 238, 0.08));
}

.cal-cell {
  position: relative;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.2s;
  overflow: hidden;
  white-space: nowrap;
}

.cal-cell.today {
  box-shadow: inset 0 0 0 1.5px color-mix(in srgb, var(--app-primary) 50%, transparent);
}

.cal-day {
  font-size: 13px;
  font-weight: 600;
  color: #334155;
}

.lv-1 { background: color-mix(in srgb, var(--app-primary) 12%, transparent); }
.lv-2 { background: color-mix(in srgb, var(--app-primary) 28%, transparent); }
.lv-3 { background: color-mix(in srgb, var(--app-primary) 50%, transparent); }
.lv-1 .cal-day, .lv-2 .cal-day, .lv-3 .cal-day { color: #fff; }

.cal-duration {
  font-size: 10px;
  color: #fff;
  margin-top: 1px;
  font-variant-numeric: tabular-nums;
}

.cal-done {
  position: absolute;
  top: 2px;
  right: 2px;
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: linear-gradient(135deg, #10b981, #34d399);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 0 6px rgba(16, 185, 129, 0.5);
}

.cal-done :deep(.el-icon) {
  font-size: 10px;
}

.cal-today-tag {
  font-size: 10px;
  color: var(--app-primary);
  margin-top: 2px;
}

/* 图例 */
.legend {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-top: 14px;
  padding-top: 12px;
  border-top: 1px solid color-mix(in srgb, var(--app-primary) 12%, transparent);
  flex-wrap: wrap;
}

.legend-item {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 11px;
  color: #64748b;
}

.lg-bar {
  width: 14px;
  height: 10px;
  border-radius: 3px;
}
.lg-0 { background: #f1f5f9; border: 1px solid #e2e8f0; }
.lg-1 { background: color-mix(in srgb, var(--app-primary) 12%, transparent); }
.lg-2 { background: color-mix(in srgb, var(--app-primary) 28%, transparent); }
.lg-3 { background: color-mix(in srgb, var(--app-primary) 50%, transparent); }

.lg-check {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: linear-gradient(135deg, #10b981, #34d399);
  box-shadow: 0 0 4px rgba(16, 185, 129, 0.5);
}

/* 最近 7 天 */
.recent7 {
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid color-mix(in srgb, var(--app-primary) 12%, transparent);
}

.week-bars {
  display: flex;
  gap: 10px;
  justify-content: space-between;
  align-items: flex-end;
}

.week-bar-col {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
}

.week-bar {
  width: 100%;
  max-width: 34px;
  min-height: 26px;
  border-radius: 8px;
  background: #f1f5f9;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: height 0.3s ease, background 0.3s ease;
}
.week-bar.bar-1 { background: color-mix(in srgb, var(--app-primary) 14%, transparent); }
.week-bar.bar-2 { background: color-mix(in srgb, var(--app-primary) 30%, transparent); }
.week-bar.bar-3 { background: linear-gradient(135deg, var(--app-primary), color-mix(in srgb, var(--app-primary) 55%, #8b5cf6)); box-shadow: 0 4px 12px color-mix(in srgb, var(--app-primary) 35%, transparent); }

.week-min {
  font-size: 10px;
  color: #64748b;
}
.bar-3 .week-min { color: #fff; }

.week-day {
  font-size: 11px;
  color: #94a3b8;
}

/* ===== 右侧面板 ===== */
.side-loading {
  display: flex;
  justify-content: center;
  padding: 30px 0;
  color: var(--app-primary);
}

.side-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 30px 0;
  color: #a3b3c6;
  text-align: center;
}

.side-empty p {
  margin: 8px 0 0;
  font-size: 13px;
  color: #64748b;
}

.side-empty-sub {
  font-size: 12px !important;
  color: #94a3b8 !important;
  margin-top: 2px !important;
}

.daily-total {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  border-radius: 12px;
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.08), rgba(34, 211, 238, 0.06));
  border: 1px solid rgba(16, 185, 129, 0.2);
  margin-bottom: 12px;
}

.dt-value {
  font-size: 20px;
  font-weight: 700;
  color: #059669;
  font-variant-numeric: tabular-nums;
}

.dt-label {
  font-size: 12px;
  color: #64748b;
}

.dt-goal {
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 12px;
  font-weight: 600;
  color: #10b981;
}

.daily-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px;
  border-radius: 12px;
  cursor: pointer;
  transition: background 0.2s, transform 0.2s;
}

.daily-item:hover {
  background: color-mix(in srgb, var(--app-primary) 6%, transparent);
  transform: translateX(3px);
}

.di-cover {
  flex-shrink: 0;
  width: 40px;
  height: 54px;
  border-radius: 6px;
  overflow: hidden;
  box-shadow: 0 4px 12px rgba(31, 38, 135, 0.2);
}

.di-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.di-cover-ph {
  width: 100%;
  height: 100%;
  background: linear-gradient(160deg, #334155, #1e293b);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #94a3b8;
}

.di-body {
  min-width: 0;
  flex: 1;
}

.di-title {
  font-size: 13px;
  font-weight: 600;
  color: #0f172a;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.di-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
  font-size: 12px;
}

.di-time {
  color: #94a3b8;
}

.di-dur {
  color: var(--app-primary);
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.di-prog {
  padding: 1px 6px;
  border-radius: 6px;
  background: color-mix(in srgb, var(--app-primary) 10%, transparent);
  color: var(--app-primary);
  font-weight: 600;
}

/* 朗读徽标与引擎标签 */
.di-tts-badge {
  padding: 1px 7px;
  border-radius: 6px;
  font-weight: 600;
  color: #b45309;
  background: linear-gradient(135deg, rgba(250, 204, 21, 0.28), rgba(251, 191, 36, 0.34));
  border: 1px solid rgba(245, 158, 11, 0.35);
}

.di-engine {
  color: #94a3b8;
}

/* ===== 计划 ===== */
.add-plan-btn {
  margin-left: auto;
  background: linear-gradient(135deg, #f59e0b, #fbbf24);
  border: none;
  box-shadow: 0 6px 16px rgba(245, 158, 11, 0.35);
}

.plan-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.plan-item {
  padding: 14px;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.7);
  border: 1px solid color-mix(in srgb, var(--app-primary) 12%, transparent);
  box-shadow: 0 4px 16px rgba(31, 38, 135, 0.06);
  transition: all 0.25s ease;
}

.plan-item:hover {
  box-shadow: 0 8px 24px rgba(31, 38, 135, 0.12);
  transform: translateY(-2px);
}

.plan-item.inactive {
  opacity: 0.55;
}

.plan-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.plan-name {
  font-size: 14px;
  font-weight: 700;
  color: #0f172a;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.plan-meta {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-bottom: 10px;
  font-size: 12px;
  color: #64748b;
}

.plan-goal {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: #f59e0b;
  font-weight: 600;
}

.plan-date {
  color: #94a3b8;
  font-variant-numeric: tabular-nums;
}

.plan-progress {
  display: flex;
  align-items: center;
  gap: 10px;
}

.plan-progress .el-progress {
  flex: 1;
}

.plan-stats-text {
  font-size: 11px;
  color: #94a3b8;
  white-space: nowrap;
}

.plan-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 6px;
  border-top: 1px dashed color-mix(in srgb, var(--app-primary) 15%, transparent);
  padding-top: 6px;
}

.form-unit {
  margin-left: 10px;
  font-size: 13px;
  color: #64748b;
}

/* ===== 响应式 ===== */
@media (max-width: 1024px) {
  .stats-main {
    grid-template-columns: minmax(0, 1fr);
  }
  .side-panel {
    position: static;
    max-height: none;
    overflow: visible;
    padding-right: 0;
  }
  .plan-list {
    max-height: 460px;
    overflow-y: auto;
    overscroll-behavior: contain;
    padding-right: 4px;
  }
}

@media (max-width: 768px) {
  .stats-row {
    grid-template-columns: repeat(2, 1fr);
    gap: 12px;
  }
  .stat-value {
    font-size: 20px;
  }

  .calendar-panel,
  .side-card {
    padding: 16px 12px;
  }

  /* 日历单元格压缩 */
  .read-calendar :deep(.el-calendar-table .el-calendar-day) {
    height: 56px;
    padding: 2px;
  }

  .read-calendar :deep(.el-calendar__title) {
    font-size: 14px;
  }

  /* 上月/今天/下月按钮组在窄屏紧凑化，避免挤压换行 */
  .read-calendar :deep(.el-calendar__header) {
    padding: 2px 0 12px;
  }
  .read-calendar :deep(.el-button-group .el-button) {
    padding: 6px 9px;
    margin-left: 3px;
    font-size: 12px;
  }
  .read-calendar :deep(.el-calendar__button-group) {
    margin-left: auto;
  }

  .cal-day {
    font-size: 12px;
  }

  .cal-duration {
    font-size: 10px;
  }

  .legend {
    flex-wrap: wrap;
    gap: 8px;
  }

  /* 最近 7 天柱状图 */
  .week-bars {
    height: 120px;
  }

  .week-min {
    font-size: 9px;
  }

  /* 当日详情 */
  .daily-list {
    max-height: 220px;
    overflow-y: auto;
  }

  /* 计划卡片 */
  .plan-item {
    padding: 12px;
  }
}
</style>
