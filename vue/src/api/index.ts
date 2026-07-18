import axios from 'axios'
import { showToast } from '../utils/toast'

let isRefreshing = false
let failedQueue: Array<{ resolve: (token: string) => void; reject: (error: unknown) => void }> = []

function processQueue(error: unknown, token: string | null = null) {
  for (const prom of failedQueue) {
    if (error) {
      prom.reject(error)
    } else {
      prom.resolve(token!)
    }
  }
  failedQueue = []
}

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 60000,
  headers: { 'Content-Type': 'application/json' }
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (res) => res,
  async (err) => {
    if (err.response?.status !== 401) {
      return Promise.reject(err)
    }

    const originalRequest = err.config

    // 刷新接口自身失败 → 直接登出
    if (originalRequest.url === '/auth/refresh') {
      return Promise.reject(err)
    }

    // 已有刷新请求进行中 → 排队等待新 token
    if (isRefreshing) {
      return new Promise<string>((resolve, reject) => {
        failedQueue.push({ resolve, reject })
      }).then((token) => {
        originalRequest.headers.Authorization = `Bearer ${token}`
        return api(originalRequest)
      })
    }

    originalRequest._retry = true
    isRefreshing = true

    try {
      const response = await api.post('/auth/refresh')
      const { access_token } = response.data.data
      localStorage.setItem('token', access_token)

      // 按序唤醒所有排队请求
      processQueue(null, access_token)

      originalRequest.headers.Authorization = `Bearer ${access_token}`
      return api(originalRequest)
    } catch (refreshError) {
      processQueue(refreshError, null)
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      showToast('登录信息已过期，请重新登录', 'error')
      redirectToLogin()
      return Promise.reject(err)
    } finally {
      isRefreshing = false
    }
  }
)

function redirectToLogin() {
  const currentPath = window.location.pathname
  const protectedPaths = ['/admin', '/profile']
  const isProtectedPath = protectedPaths.some(path => currentPath.startsWith(path))
  if (isProtectedPath && currentPath !== '/login') {
    window.location.href = '/login'
  }
}

export default api
