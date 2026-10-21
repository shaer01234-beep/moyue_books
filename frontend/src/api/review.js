import axios from 'axios'
import { API_BASE } from '@/config'

// ========== 读后感（读书笔记）API ==========
export const reviewApi = {
  // 读后感列表：支持 q（读后感标题/所关联书名搜索）、book_id（按书过滤）
  listReviews(params = {}) {
    return axios.get(`${API_BASE}/reviews`, { params })
  },

  // 某本书的读后感（阅读页使用）
  listBookReviews(bookId) {
    return axios.get(`${API_BASE}/books/${bookId}/reviews`)
  },

  // 读后感详情
  getReview(id) {
    return axios.get(`${API_BASE}/reviews/${id}`)
  },

  // 创建读后感
  createReview(data) {
    return axios.post(`${API_BASE}/reviews`, data)
  },

  // 更新读后感
  updateReview(id, data) {
    return axios.put(`${API_BASE}/reviews/${id}`, data)
  },

  // 删除读后感
  deleteReview(id) {
    return axios.delete(`${API_BASE}/reviews/${id}`)
  }
}
