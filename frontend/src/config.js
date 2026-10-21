// ===== 前后端服务地址配置 =====
// 兼容两种访问方式（同一份构建均适用）：
//   1) 飞牛统一网关：页面经 /app/moyue 前缀由网关反代，API 走 /app/moyue/api
//   2) 直接 TCP 访问（IP:端口，局域网直连 / 分享链接）：根路径，API 走 /api
// 优先使用构建时注入的 BASE_URL（fnOS 打包脚本已将 base 设为 /app/moyue/），
// 未注入（默认相对路径构建）时按运行时路径自动探测网关前缀，保证两种方式都不需要单独构建。
const KNOWN_GATEWAY_PREFIXES = ['/app/moyue']

function detectGatewayPrefix() {
  const p = window.location.pathname || '/'
  for (const prefix of KNOWN_GATEWAY_PREFIXES) {
    if (p === prefix || p.startsWith(prefix + '/')) return prefix
  }
  return ''
}

// BASE_URL 由 vite 的 base 注入：fnOS 构建为 /app/moyue/，默认构建为 ./
const injectedBase = (import.meta.env.BASE_URL || '/').replace(/\/+$/, '')
const BASE = injectedBase && injectedBase !== '.' && injectedBase !== ''
  ? injectedBase
  : detectGatewayPrefix()

// API 地址：优先用构建时注入的 VITE_API_BASE，否则按 BASE 推导（网关 /app/moyue/api 或直连 /api）
const API_BASE = (import.meta.env.VITE_API_BASE || (BASE ? BASE + '/api' : '/api')).replace(/\/+$/, '')

// 静态资源（图书文件 / 封面等）基础地址：由 API_BASE 去掉 /api 后缀得到
// 例：API_BASE=https://host/api  ->  staticBase=https://host
//     API_BASE=/app/moyue/api    ->  staticBase=/app/moyue
//     API_BASE=/api（同源）      ->  staticBase=''
const staticBase = () => (API_BASE ? API_BASE.replace(/\/api$/, '') : '')

// 由 API_BASE 推导 WebSocket 基础地址（TTS 朗读走 /api/tts/ws）
// 例：API_BASE=https://host/api  ->  wss://host/api
//     API_BASE=/app/moyue/api    ->  ws(s)://当前host/app/moyue/api
//     API_BASE=/api（同源）      ->  ws(s)://当前host/api
const wsApiBase = () => {
  if (/^https?:\/\//i.test(API_BASE)) {
    const u = new URL(API_BASE)
    const proto = u.protocol === 'https:' ? 'wss:' : 'ws:'
    return `${proto}//${u.host}${u.pathname.replace(/\/$/, '')}`
  }
  const proto = location.protocol === 'https:' ? 'wss://' : 'ws://'
  return `${proto}${location.host}${API_BASE || '/api'}`
}

export { API_BASE, staticBase, wsApiBase }
