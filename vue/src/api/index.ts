import axios from 'axios'
import { handleAuthFailure, refreshAccessToken } from '../utils/token'

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

    // 刷新接口自身失败 → 交给调用方收尾，不能再递归刷新
    if (originalRequest.url === '/auth/refresh') {
      return Promise.reject(err)
    }

    // 已经重试过一次仍然 401：说明问题不在 token 过期（可能被服务端拒绝），
    // 再刷一次只会打成死循环。原实现只写不读这个标记，等于没有防重入。
    if (originalRequest._retry) {
      handleAuthFailure()
      return Promise.reject(err)
    }
    originalRequest._retry = true

    try {
      // 单飞：并发请求同时 401 时只发一次刷新（后端 rotation，并发刷会互相顶掉）
      const token = await refreshAccessToken()
      originalRequest.headers.Authorization = `Bearer ${token}`
      return api(originalRequest)
    } catch {
      handleAuthFailure()
      return Promise.reject(err)
    }
  }
)

export default api
