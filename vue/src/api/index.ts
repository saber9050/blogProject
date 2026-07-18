import axios from 'axios'

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

    // 刷新接口自身失败 或 已重试过 → 直接登出
    if (originalRequest._retry || originalRequest.url === '/auth/refresh') {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      redirectToLogin()
      return Promise.reject(err)
    }

    // 尝试用 Cookie 中的 refresh_token 刷新
    originalRequest._retry = true
    try {
      const response = await api.post('/auth/refresh')
      const { access_token } = response.data.data
      localStorage.setItem('token', access_token)
      // 用新 token 重试原请求
      originalRequest.headers.Authorization = `Bearer ${access_token}`
      return api(originalRequest)
    } catch {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      redirectToLogin()
      return Promise.reject(err)
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
