import axios from 'axios'
import { getToken } from './auth'
import { API_BASE, staticBase } from '@/config'

// 拼接鉴权 query 参数：<img>/<iframe> 无法携带 Authorization 头，需通过 ?token= 完成鉴权
function withToken(url) {
  const token = getToken()
  if (!token) return url
  const sep = url.includes('?') ? '&' : '?'
  return `${url}${sep}token=${encodeURIComponent(token)}`
}

export const bookApi = {
  // 图书列表
  listBooks(params = {}) {
    return axios.get(`${API_BASE}/books`, { params })
  },

  // 获取图书详情
  getBook(id) {
    return axios.get(`${API_BASE}/books/${id}`)
  },

  // 获取图书分类及数量
  getCategories() {
    return axios.get(`${API_BASE}/books/categories`)
  },

  // 创建分类（可不选图书，仅创建空分类；也可同时为图书设置分类）
  createCategory(data) {
    return axios.post(`${API_BASE}/books/categories`, data)
  },

  // 重命名分类
  renameCategory(oldName, newName) {
    return axios.put(`${API_BASE}/categories/rename`, { old_name: oldName, new_name: newName })
  },

  // 删除分类（可选将图书移动到其他分类）
  deleteCategory(name, moveTo = '') {
    return axios.delete(`${API_BASE}/categories/${encodeURIComponent(name)}`, {
      params: moveTo ? { move_to: moveTo } : {}
    })
  },

  // 向已有分类追加图书（分类不存在则自动创建）
  addBooksToCategory(name, bookIds) {
    return axios.post(`${API_BASE}/categories/${encodeURIComponent(name)}/books`, { book_ids: bookIds })
  },

  // 从分类移除图书（仅移除当前属于该分类的图书，将其 category 置空）
  removeBooksFromCategory(name, bookIds) {
    return axios.delete(`${API_BASE}/categories/${encodeURIComponent(name)}/books`, {
      data: { book_ids: bookIds }
    })
  },

  // 更新图书
  updateBook(id, data) {
    return axios.put(`${API_BASE}/books/${id}`, data)
  },

  // 删除图书
  deleteBook(id) {
    return axios.delete(`${API_BASE}/books/${id}`)
  },

  // 上传图书
  uploadBook(file, category = '') {
    const formData = new FormData()
    formData.append('file', file)
    if (category) formData.append('category', category)
    return axios.post(`${API_BASE}/books`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
  },

  // 批量上传
  uploadBooks(files, category = '') {
    const formData = new FormData()
    files.forEach(f => formData.append('files', f))
    if (category) formData.append('category', category)
    return axios.post(`${API_BASE}/books/batch`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
  },

  // 上传封面
  uploadCover(id, file) {
    const formData = new FormData()
    formData.append('cover', file)
    return axios.post(`${API_BASE}/books/${id}/cover`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
  },

  // 更新进度（ttsLocation 为朗读断点 JSON 字符串，可省略）
  updateProgress(id, progress, currentPage, ttsLocation = '') {
    return axios.put(`${API_BASE}/books/${id}/progress`, { progress, current_page: currentPage, tts_location: ttsLocation })
  },

  // 重新提取元数据
  refreshMetadata(id) {
    return axios.patch(`${API_BASE}/books/${id}/metadata`)
  },

  // 下载图书
  downloadBook(id) {
    return axios.get(`${API_BASE}/books/${id}/download`, { responseType: 'blob' })
  },

  // 获取封面 URL（原生 <img> 加载，通过 ?token= 鉴权）
  coverUrl(id) {
    return withToken(`${API_BASE}/books/${id}/cover`)
  },

  // 获取 EPUB 内嵌资源 URL（图片/CSS 等，<img> 直接加载，通过 ?token= 鉴权）
  resourceUrl(id, path) {
    return withToken(`${API_BASE}/books/${id}/resource?path=${encodeURIComponent(path)}`)
  },

  // 获取 EPUB 内容
  getBookContent(id, chapter = '') {
    return axios.get(`${API_BASE}/books/${id}/content`, {
      params: chapter ? { chapter } : {},
      responseType: chapter ? 'text' : 'json'
    })
  },

  // 统一目录接口（EPUB/KEPUB/PDF/MD 等，异步解析）
  getBookTOC(id) {
    return axios.get(`${API_BASE}/books/${id}/toc`)
  },

  // 获取统计
  getStats() {
    return axios.get(`${API_BASE}/stats`)
  },

  // ========== 阅读统计 / 计划 API ==========
  // 上报阅读/朗读会话；extra 可带 { type:'read'|'tts', engine, start_page, end_page }
  reportReading(bookId, duration, startTime, endTime, extra = {}) {
    return axios.post(`${API_BASE}/reading-history`, {
      book_id: bookId,
      duration,
      start_time: startTime,
      end_time: endTime,
      type: extra.type || 'read',
      engine: extra.engine || '',
      start_page: extra.start_page || 0,
      end_page: extra.end_page || 0
    })
  },

  // 按月日历统计
  getCalendarStats(year, month) {
    return axios.get(`${API_BASE}/stats/calendar`, { params: { year, month } })
  },

  // 统计总览
  getStatsOverview() {
    return axios.get(`${API_BASE}/stats/overview`)
  },

  // 某天阅读详情
  getDailyStats(date) {
    return axios.get(`${API_BASE}/stats/daily`, { params: { date } })
  },

  // 阅读计划列表
  listPlans() {
    return axios.get(`${API_BASE}/plans`)
  },

  // 创建阅读计划
  createPlan(data) {
    return axios.post(`${API_BASE}/plans`, data)
  },

  // 更新阅读计划
  updatePlan(id, data) {
    return axios.put(`${API_BASE}/plans/${id}`, data)
  },

  // 删除阅读计划
  deletePlan(id) {
    return axios.delete(`${API_BASE}/plans/${id}`)
  },

  // ========== 批注/划线 API ==========
  // 获取图书批注列表
  listAnnotations(bookId, type = '') {
    return axios.get(`${API_BASE}/books/${bookId}/annotations`, {
      params: type ? { type } : {}
    })
  },

  // 创建批注
  createAnnotation(bookId, data) {
    return axios.post(`${API_BASE}/books/${bookId}/annotations`, data)
  },

  // 更新批注
  updateAnnotation(annotationId, data) {
    return axios.put(`${API_BASE}/annotations/${annotationId}`, data)
  },

  // 删除批注
  deleteAnnotation(annotationId) {
    return axios.delete(`${API_BASE}/annotations/${annotationId}`)
  },

  // 获取图书文件静态 URL（用于 PDF.js 直接加载；App 内需绝对地址）
  bookFileUrl(filePath) {
    if (!filePath) return ''
    const filename = filePath.split(/[\\/]/).pop()
    return `${staticBase()}/static/books/${filename}`
  },

  // 按图书 ID 出书（内联）：支持媒体库等非 uploads 路径，统一经后端鉴权读取源文件
  fileUrl(id) {
    return withToken(`${API_BASE}/books/${id}/file`)
  },

  // ========== 收藏夹 API ==========
  // 收藏夹列表
  listCollections() {
    return axios.get(`${API_BASE}/collections`)
  },

  // 收藏夹详情（含图书，分页）
  getCollection(id, params = {}) {
    return axios.get(`${API_BASE}/collections/${id}`, { params })
  },

  // 创建收藏夹
  createCollection(data) {
    return axios.post(`${API_BASE}/collections`, data)
  },

  // 更新收藏夹
  updateCollection(id, data) {
    return axios.put(`${API_BASE}/collections/${id}`, data)
  },

  // 删除收藏夹
  deleteCollection(id) {
    return axios.delete(`${API_BASE}/collections/${id}`)
  },

  // 向收藏夹添加图书
  addBooksToCollection(id, bookIds) {
    return axios.post(`${API_BASE}/collections/${id}/books`, { book_ids: bookIds })
  },

  // 从收藏夹移除图书
  removeBookFromCollection(id, bookId) {
    return axios.delete(`${API_BASE}/collections/${id}/books/${bookId}`)
  },

  // 获取图书所属收藏夹
  getBookCollections(bookId) {
    return axios.get(`${API_BASE}/books/${bookId}/collections`)
  },

  // 设置图书所属收藏夹（全量替换）
  setBookCollections(bookId, collectionIds) {
    return axios.put(`${API_BASE}/books/${bookId}/collections`, { collection_ids: collectionIds })
  },

  // ========== 语音朗读 API ==========
  // 获取 TTS 引擎/情感/音色列表
  getTtsSettings() {
    return axios.get(`${API_BASE}/tts/settings`)
  },

  // 语音合成（返回音频 blob）
  synthesizeTts(payload) {
    return axios.post(`${API_BASE}/tts/synthesize`, payload, { responseType: 'blob' })
  },

  // 获取 EPUB 章节纯文本（用于朗读）
  getBookChapterText(bookId, chapter) {
    return axios.get(`${API_BASE}/books/${bookId}/text`, { params: { chapter } })
  }
}

export default bookApi
