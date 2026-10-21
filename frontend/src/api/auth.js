import axios from 'axios'
import { API_BASE } from '@/config'

// ===== token 与用户信息存取 =====
export function getToken() {
  return localStorage.getItem('mb_token') || ''
}

export function setToken(token) {
  localStorage.setItem('mb_token', token)
}

export function clearToken() {
  localStorage.removeItem('mb_token')
}

export function getUser() {
  try {
    return JSON.parse(localStorage.getItem('mb_user') || 'null')
  } catch {
    return null
  }
}

export function setUser(user) {
  if (user) {
    localStorage.setItem('mb_user', JSON.stringify(user))
  } else {
    localStorage.removeItem('mb_user')
  }
}

// ===== 认证接口 =====
export function register(username, password, nickname) {
  return axios.post(`${API_BASE}/auth/register`, { username, password, nickname })
}

export function login(username, password) {
  return axios.post(`${API_BASE}/auth/login`, { username, password })
}

export function getMe() {
  return axios.get(`${API_BASE}/auth/me`)
}

export function updateProfile(nickname, avatar) {
  return axios.put(`${API_BASE}/auth/me`, { nickname, avatar })
}

export function changePassword(oldPassword, newPassword) {
  return axios.put(`${API_BASE}/auth/password`, {
    old_password: oldPassword,
    new_password: newPassword
  })
}
