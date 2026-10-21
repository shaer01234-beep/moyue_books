// 阅读偏好本地持久化工具（设置页与阅读器共享同一份配置）
const KEY = 'reader_prefs_v1'

export const DEFAULT_PREFS = {
  fontSize: 16,
  lineHeight: 1.8,
  pageMargin: 40,
  fontFamily: 'system-ui, -apple-system, "PingFang SC", "Microsoft YaHei", sans-serif',
  theme: 'light',
  pdfTwoPage: false,
  pdfScale: 1.5,
  epubTwoPage: false
}

export function loadReaderPrefs() {
  try {
    const raw = localStorage.getItem(KEY)
    if (!raw) return { ...DEFAULT_PREFS }
    return { ...DEFAULT_PREFS, ...JSON.parse(raw) }
  } catch {
    return { ...DEFAULT_PREFS }
  }
}

export function saveReaderPrefs(prefs) {
  try {
    localStorage.setItem(KEY, JSON.stringify(prefs))
  } catch {
    /* 忽略存储异常 */
  }
}

export function resetReaderPrefs() {
  try {
    localStorage.removeItem(KEY)
  } catch {
    /* 忽略存储异常 */
  }
  return { ...DEFAULT_PREFS }
}
