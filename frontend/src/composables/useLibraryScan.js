import { API_BASE } from '@/config'
import { getToken } from '@/api/auth'

// 以 fetch + StreamReader 消费后端媒体库扫描的 SSE 进度（与上传页扫描一致）。
// handlers: { onProgress(data), onDone(data), onError(msg) }
// 返回 Promise，扫描流结束后 resolve。
export function runLibraryScan(id, handlers = {}) {
  const url = `${API_BASE}/libraries/${id}/scan`
  return fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${getToken()}`
    },
    body: JSON.stringify({})
  }).then(async (resp) => {
    if (!resp.ok) {
      let msg = `扫描失败 (${resp.status})`
      try {
        const err = await resp.json()
        if (err && err.error) msg = err.error
      } catch (e) { /* ignore */ }
      if (handlers.onError) handlers.onError(msg)
      return Promise.reject(new Error(msg))
    }
    const reader = resp.body.getReader()
    const decoder = new TextDecoder()
    let buffer = ''

    return new Promise((resolve, reject) => {
      const pump = () => {
        reader.read().then(({ done, value }) => {
          if (done) {
            resolve()
            return
          }
          buffer += decoder.decode(value, { stream: true })
          let idx
          while ((idx = buffer.indexOf('\n\n')) !== -1) {
            const raw = buffer.slice(0, idx)
            buffer = buffer.slice(idx + 2)
            parseScanEvent(raw, handlers)
          }
          pump()
        }).catch(reject)
      }
      pump()
    })
  })
}

function parseScanEvent(raw, handlers) {
  let event = 'message'
  const dataLines = []
  for (const line of raw.split('\n')) {
    if (line.startsWith('event:')) event = line.slice(6).trim()
    else if (line.startsWith('data:')) dataLines.push(line.slice(5).trim())
  }
  if (dataLines.length === 0) return
  let data
  try {
    data = JSON.parse(dataLines.join('\n'))
  } catch (e) {
    return
  }
  if (event === 'progress' && handlers.onProgress) handlers.onProgress(data)
  else if (event === 'done' && handlers.onDone) handlers.onDone(data)
  else if (event === 'error' && handlers.onError) handlers.onError(data.error || '扫描失败')
}
