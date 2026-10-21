// ===== 全局外观配置（主题色 / 网站名称 / 网站图标 / 页面动效 / 菜单透光） =====

const STORAGE_KEY = 'moyue.appPrefs'

export const DEFAULT_APP_PREFS = {
  themeColor: '#10b981',   // 主题色（默认翠绿，与品牌绿色 logo 一致）
  siteName: '墨阅',         // 网站名称
  siteIcon: 'Collection',  // 网站图标（Element Plus 图标名，或 http/data 开头的自定义 URL）
  pageTransition: true,    // 页面切换动效
  menuGlow: true           // 菜单透光效果
}

// 候选图标（Element Plus 全局注册，可直接按名渲染）
export const APP_ICON_OPTIONS = ['Collection', 'Reading', 'Notebook', 'Star', 'Sunny', 'Moon', 'MagicStick', 'Fire']

// 预设主题色（首位为推荐/默认绿色）
export const APP_COLOR_PRESETS = ['#10b981', '#6366f1', '#3b82f6', '#06b6d4', '#f59e0b', '#ec4899', '#ef4444', '#8b5cf6']

// 旧版默认主题色（靛蓝），用于迁移判断
const LEGACY_THEME_COLOR = '#6366f1'

export function loadAppPrefs() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return { ...DEFAULT_APP_PREFS }
    const prefs = { ...DEFAULT_APP_PREFS, ...JSON.parse(raw) }
    // 旧默认名迁移：曾默认“我的书架”，更名后统一为“墨阅”，已自定义的名称不受影响
    if (prefs.siteName === '我的书架') {
      prefs.siteName = DEFAULT_APP_PREFS.siteName
      saveAppPrefs(prefs)
    }
    // 旧默认主题色迁移：靛蓝 → 品牌绿（仅替换旧默认值，刻意自定义靛蓝的用户可再点回）
    if (prefs.themeColor === LEGACY_THEME_COLOR) {
      prefs.themeColor = DEFAULT_APP_PREFS.themeColor
      saveAppPrefs(prefs)
    }
    return prefs
  } catch (e) {
    return { ...DEFAULT_APP_PREFS }
  }
}

export function saveAppPrefs(prefs) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(prefs))
}

export function resetAppPrefs() {
  return { ...DEFAULT_APP_PREFS }
}

// ===== 颜色工具 =====
function hexToRgb(hex) {
  const h = hex.replace('#', '')
  const full = h.length === 3 ? h.split('').map((c) => c + c).join('') : h
  const num = parseInt(full, 16)
  return { r: (num >> 16) & 255, g: (num >> 8) & 255, b: num & 255 }
}

function mixColor(hex, targetHex, weight) {
  const a = hexToRgb(hex)
  const b = hexToRgb(targetHex)
  const toHex = (n) => n.toString(16).padStart(2, '0')
  const r = Math.round(a.r + (b.r - a.r) * weight)
  const g = Math.round(a.g + (b.g - a.g) * weight)
  const bl = Math.round(a.b + (b.b - a.b) * weight)
  return `#${toHex(r)}${toHex(g)}${toHex(bl)}`
}

// 是否为自定义图标 URL
export function isCustomIcon(icon) {
  return typeof icon === 'string' && /^(https?:|data:|\.|\/)/.test(icon)
}

// 根据主题色生成 favicon SVG（主题色圆角方块 + 网站名首字符）
function buildFaviconHref(prefs) {
  const icon = prefs.siteIcon
  if (isCustomIcon(icon)) {
    return icon + (icon.includes('?') ? '&' : '?') + 'v=' + Date.now()
  }
  const ch = (prefs.siteName || '书').charAt(0)
  const color = prefs.themeColor || DEFAULT_APP_PREFS.themeColor
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64"><defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="${color}"/><stop offset="1" stop-color="${mixColor(color, '#22d3ee', 0.55)}"/></linearGradient></defs><rect width="64" height="64" rx="14" fill="url(#g)"/><text x="32" y="42" font-size="30" font-family="'PingFang SC','Microsoft YaHei',sans-serif" font-weight="bold" fill="#fff" text-anchor="middle">${ch}</text></svg>`
  return 'data:image/svg+xml;charset=utf-8,' + encodeURIComponent(svg)
}

// 将外观配置应用到全局：CSS 变量（主题色/渐变）、Element Plus 主色、document.title、favicon
export function applyAppPrefs(prefs = loadAppPrefs()) {
  const root = document.documentElement
  const color = prefs.themeColor || DEFAULT_APP_PREFS.themeColor

  // 主题色语义变量
  root.style.setProperty('--app-primary', color)
  root.style.setProperty('--app-primary-dark', mixColor(color, '#000000', 0.2))
  root.style.setProperty('--app-primary-light-3', mixColor(color, '#ffffff', 0.3))
  root.style.setProperty('--app-primary-light-5', mixColor(color, '#ffffff', 0.5))
  root.style.setProperty('--app-primary-light-7', mixColor(color, '#ffffff', 0.7))
  root.style.setProperty('--app-primary-light-9', mixColor(color, '#ffffff', 0.9))
  root.style.setProperty('--app-gradient', `linear-gradient(135deg, ${color}, ${mixColor(color, '#22d3ee', 0.55)})`)

  // 同步 Element Plus 主色（按钮、滑块、表单等自动跟随）
  root.style.setProperty('--el-color-primary', color)
  root.style.setProperty('--el-color-primary-dark-2', mixColor(color, '#000000', 0.2))
  root.style.setProperty('--el-color-primary-light-3', mixColor(color, '#ffffff', 0.3))
  root.style.setProperty('--el-color-primary-light-5', mixColor(color, '#ffffff', 0.5))
  root.style.setProperty('--el-color-primary-light-7', mixColor(color, '#ffffff', 0.7))
  root.style.setProperty('--el-color-primary-light-8', mixColor(color, '#ffffff', 0.8))
  root.style.setProperty('--el-color-primary-light-9', mixColor(color, '#ffffff', 0.9))

  // 网站名称与图标
  document.title = prefs.siteName || DEFAULT_APP_PREFS.siteName
  let link = document.querySelector("link[rel='icon']")
  if (!link) {
    link = document.createElement('link')
    link.rel = 'icon'
    document.head.appendChild(link)
  }
  link.href = buildFaviconHref(prefs)

  // 动效与透光开关：供 CSS 通过属性选择器控制
  root.dataset.pageTransition = prefs.pageTransition ? 'on' : 'off'
  root.dataset.menuGlow = prefs.menuGlow ? 'on' : 'off'
}
