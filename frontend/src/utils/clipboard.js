/**
 * 剪贴板复制工具。
 *
 * navigator.clipboard 仅在安全上下文（HTTPS 或 localhost）可用；
 * 通过 IP / 域名以 HTTP 访问（NAS、局域网等）时该 API 不存在，
 * 需回退到"临时 textarea + execCommand" 方案复制。
 */

// 兼容复制：优先 Clipboard API，失败自动回退 execCommand。
// 返回 Promise<boolean>，true 表示复制成功。
export async function copyText(text) {
  if (window.isSecureContext && navigator.clipboard && navigator.clipboard.writeText) {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch (e) {
      /* 权限被拒等，回退下方 execCommand 方案 */
    }
  }
  return legacyCopy(text)
}

function legacyCopy(text) {
  const ta = document.createElement('textarea')
  ta.value = text
  ta.setAttribute('readonly', '')
  ta.style.position = 'fixed'
  ta.style.top = '0'
  ta.style.left = '-9999px'
  ta.style.opacity = '0'
  document.body.appendChild(ta)

  // 记住并恢复当前选区，避免复制破坏阅读器/输入框选择状态
  const sel = window.getSelection()
  const prevRange = sel && sel.rangeCount > 0 ? sel.getRangeAt(0) : null

  ta.focus()
  ta.select()
  ta.setSelectionRange(0, text.length)

  let ok = false
  try {
    ok = document.execCommand('copy')
  } catch (e) {
    ok = false
  }

  document.body.removeChild(ta)
  if (prevRange) {
    sel.removeAllRanges()
    sel.addRange(prevRange)
  }
  return ok
}
