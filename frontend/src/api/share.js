import axios from 'axios'
import { API_BASE } from '@/config'

export const shareApi = {
  // 创建分享
  createShare(bookId, expiresDays) {
    return axios.post(`${API_BASE}/books/${bookId}/share`, { expires_days: expiresDays })
  },

  // 查看某本书的分享列表
  listShares(bookId) {
    return axios.get(`${API_BASE}/books/${bookId}/shares`)
  },

  // 撤销分享
  revokeShare(token) {
    return axios.delete(`${API_BASE}/shares/${token}`)
  },

  // 公开：分享信息（无需登录）
  getShareInfo(token) {
    return axios.get(`${API_BASE}/shares/${token}`)
  },

  // 公开：分享封面 URL
  shareCoverUrl(token) {
    return `${API_BASE}/shares/${token}/cover`
  },

  // 判断是否为分享者本人
  checkOwn(token) {
    return axios.get(`${API_BASE}/shares/${token}/own`)
  },

  // 接收分享（登录）
  acceptShare(token) {
    return axios.post(`${API_BASE}/shares/${token}/accept`)
  }
}

// 有声书分享
export const audiobookShareApi = {
  // 创建分享
  createShare(taskId, expiresDays) {
    return axios.post(`${API_BASE}/audiobooks/${taskId}/share`, { expires_days: expiresDays })
  },

  // 查看某本有声书的分享列表
  listShares(taskId) {
    return axios.get(`${API_BASE}/audiobooks/${taskId}/shares`)
  },

  // 撤销分享
  revokeShare(token) {
    return axios.delete(`${API_BASE}/shares/audiobook/${token}`)
  },

  // 公开：分享信息（无需登录）
  getShareInfo(token) {
    return axios.get(`${API_BASE}/shares/audiobook/${token}`)
  },

  // 公开：分享封面 URL
  shareCoverUrl(token) {
    return `${API_BASE}/shares/audiobook/${token}/cover`
  },

  // 判断是否为分享者本人
  checkOwn(token) {
    return axios.get(`${API_BASE}/shares/audiobook/${token}/own`)
  },

  // 接收分享（登录）
  acceptShare(token) {
    return axios.post(`${API_BASE}/shares/audiobook/${token}/accept`)
  }
}

// 公开：服务端口配置（用于构造端口直连的分享链接，飞牛部署不走统一网关）
export const getShareConfig = () => axios.get(`${API_BASE}/config/public`)

export default shareApi
