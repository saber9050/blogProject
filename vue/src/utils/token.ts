import axios from 'axios'
import { showToast } from './toast'

/**
 * access token 的刷新入口。
 *
 * 为什么必须集中成一个模块（而不是 axios 与流式各写一套）：
 * 后端的 refresh 是 **rotation** 的——每刷一次就生成新的 refresh token 覆盖 Redis。
 * 两个请求并发刷新时，第二个拿着旧的 TID 去刷会被判定无效，用户直接被登出。
 * 所以这里做**单飞**：无论多少处同时发现 token 过期，都只发一次刷新请求。
 */

/** 进行中的刷新。非 null 时后续调用直接复用同一个 promise。 */
let pending: Promise<string> | null = null

/**
 * 用已过期的 access token 换一个新的。
 *
 * 注意：refresh 接口需要带**当前这个已过期的 access token**，
 * 后端从它的 TID claim 里取 refresh token（见 auth/controller.go）。
 * 所以不能因为"它过期了"就丢掉它。
 */
export function refreshAccessToken(): Promise<string> {
  if (pending) return pending

  pending = doRefresh()
  // 结束即清空：成功时后续调用会拿到已写入的新 token；
  // 失败时清空才能让下一次 401 再试一次（否则一次失败会永久卡死）。
  const reset = () => {
    pending = null
  }
  pending.then(reset, reset)
  return pending
}

async function doRefresh(): Promise<string> {
  const stale = localStorage.getItem('token')
  if (!stale) throw new Error('没有可用的登录凭证')

  // 用裸 axios 而不是 api 单例：单例的 401 拦截器会再次进入刷新流程，形成递归。
  const res = await axios.post('/api/v1/auth/refresh', null, {
    headers: { Authorization: `Bearer ${stale}` }
  })
  const token = res.data?.data?.access_token
  if (!token) throw new Error('刷新登录凭证失败')
  localStorage.setItem('token', token)
  return token
}

/**
 * 刷新失败（长期凭证也过期 / 服务端异常）后的统一收尾。
 *
 * 必须清干净 token 与 user：只清 token 会让界面停留在"已登录"的样子，
 * 下一个请求继续 401，用户看到一连串报错却不知道为什么。
 */
export function handleAuthFailure() {
  localStorage.removeItem('token')
  localStorage.removeItem('user')
  showToast('登录信息已过期，请重新登录', 'error')
  redirectToLogin()
}

function redirectToLogin() {
  const currentPath = window.location.pathname
  const protectedPaths = ['/admin', '/profile']
  const isProtectedPath = protectedPaths.some((path) => currentPath.startsWith(path))
  if (isProtectedPath && currentPath !== '/login') {
    window.location.href = '/login'
  }
}
