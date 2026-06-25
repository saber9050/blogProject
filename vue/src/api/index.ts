import axios from 'axios'

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 15000,
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
  (err) => {
    if (err.response?.status === 401) {
      // 清除token，但保留用户数据以便显示
      localStorage.removeItem('token')
      
      const currentPath = window.location.pathname
      // 需要认证的页面路径
      const protectedPaths = ['/admin', '/profile']
      const isProtectedPath = protectedPaths.some(path => currentPath.startsWith(path))
      
      // 如果当前在需要认证的页面，则重定向到登录页
      if (isProtectedPath && currentPath !== '/login') {
        window.location.href = '/login'
      }
      // 对于公开页面（如首页），不清除用户数据，让页面以未登录状态显示
    }
    return Promise.reject(err)
  }
)

export default api
