import axios from 'axios'
import { getToken } from './auth'
import { API_BASE } from '@/config'

// 拼接鉴权 query 参数：<audio> 无法携带 Authorization 头，需通过 ?token= 完成鉴权
function withToken(url) {
  const token = getToken()
  if (!token) return url
  const sep = url.includes('?') ? '&' : '?'
  return `${url}${sep}token=${encodeURIComponent(token)}`
}

export const audiobookApi = {
  // 有声书任务列表（含图书信息）
  list() {
    return axios.get(`${API_BASE}/audiobooks`)
  },

  // 创建生成任务（异步）
  create(data) {
    return axios.post(`${API_BASE}/audiobooks`, data)
  },

  // 任务详情 + 分段列表
  get(id) {
    return axios.get(`${API_BASE}/audiobooks/${id}`)
  },

  // 取消生成（保留已生成分段，可续传）
  cancel(id) {
    return axios.post(`${API_BASE}/audiobooks/${id}/cancel`)
  },

  // 暂停生成（保留已生成分段，可恢复）
  pause(id) {
    return axios.post(`${API_BASE}/audiobooks/${id}/pause`)
  },

  // 恢复暂停的生成任务（续传）
  resume(id) {
    return axios.post(`${API_BASE}/audiobooks/${id}/resume`)
  },

  // 删除（取消 + 删除全部音频文件与记录）
  remove(id) {
    return axios.delete(`${API_BASE}/audiobooks/${id}`)
  },

  // 分段音频播放地址（需 token 鉴权）
  audioUrl(taskId, file) {
    return withToken(`${API_BASE}/audiobooks/${taskId}/audio/${encodeURIComponent(file)}`)
  },

  // 任务封面地址（优先任务自身封面，回退原书封面；原生 <img> 加载，需 ?token= 鉴权）
  coverUrl(taskId) {
    return withToken(`${API_BASE}/audiobooks/${taskId}/cover`)
  },

  // 保存播放进度（用于下次继续播放）
  saveProgress(id, segment, position) {
    return axios.put(`${API_BASE}/audiobooks/${id}/progress`, { segment, position })
  }
}
