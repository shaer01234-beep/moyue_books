import axios from 'axios'
import { getToken } from './auth'
import { API_BASE } from '@/config'

// 媒体库相关接口
export const libraryApi = {
  // 媒体库列表
  list() {
    return axios.get(`${API_BASE}/libraries`)
  },

  // 媒体库详情
  get(id) {
    return axios.get(`${API_BASE}/libraries/${id}`)
  },

  // 新建媒体库
  create(data) {
    return axios.post(`${API_BASE}/libraries`, data)
  },

  // 更新媒体库
  update(id, data) {
    return axios.put(`${API_BASE}/libraries/${id}`, data)
  },

  // 删除媒体库
  remove(id) {
    return axios.delete(`${API_BASE}/libraries/${id}`)
  },

  // 扫描记录
  logs(id, params = {}) {
    return axios.get(`${API_BASE}/libraries/${id}/logs`, { params })
  },

  // 触发扫描的 SSE 地址（POST，需 Authorization 头）
  scanUrl(id) {
    return `${API_BASE}/libraries/${id}/scan`
  }
}

export default libraryApi
