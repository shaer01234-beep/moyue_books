import axios from 'axios'
import { API_BASE } from '@/config'

// ===== 问题反馈（QQ / 邮件）=====

// 获取展示给使用者的反馈方式（QQ 号与邮件通道状态）
export function getFeedbackInfo() {
  return axios.get(`${API_BASE}/feedback/info`)
}

// 提交反馈：填写内容与联系方式即可，由后端内置 163 通道自动发送至开发者邮箱
export function sendFeedback(contact, content) {
  return axios.post(`${API_BASE}/feedback/send`, { contact, content })
}
