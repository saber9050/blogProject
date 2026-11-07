import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

/**
 * token 刷新是「看起来简单、错一次就登出用户」的一块：
 * 后端 refresh 是 rotation 的（每刷一次就换掉 Redis 里的 refresh token），
 * 所以**并发刷新必然互相顶掉**——第二个请求拿旧 TID 去刷会被判无效，
 * 用户直接被登出，而且只在「多个请求恰好同时 401」时复现。
 *
 * 这几条测试守的就是「只发一次」这个约束。
 */

// 用可变函数替代 axios.post，测试里替换实现
const postMock = vi.fn()

vi.mock('axios', () => ({
  default: {
    post: (...args: unknown[]) => postMock(...args)
  }
}))

// toast 会渲染到全局容器，这里不需要它的副作用
vi.mock('../../src/utils/toast', () => ({
  showToast: vi.fn()
}))

describe('refreshAccessToken', () => {
  beforeEach(() => {
    postMock.mockReset()
    localStorage.clear()
    localStorage.setItem('token', 'stale-token')
  })

  afterEach(() => {
    vi.resetModules()
  })

  it('并发调用只发一次刷新请求', async () => {
    // 每次 import 拿到的是同一个模块实例，pending 在其中共享
    const { refreshAccessToken } = await import('../../src/utils/token')

    postMock.mockResolvedValue({ data: { data: { access_token: 'new-token' } } })

    const [a, b, c] = await Promise.all([
      refreshAccessToken(),
      refreshAccessToken(),
      refreshAccessToken()
    ])

    expect(a).toBe('new-token')
    expect(b).toBe('new-token')
    expect(c).toBe('new-token')
    // 关键断言：三次并发只打一次接口
    expect(postMock).toHaveBeenCalledTimes(1)
    expect(localStorage.getItem('token')).toBe('new-token')
  })

  it('必须带上已过期的 access token 作为 Authorization', async () => {
    const { refreshAccessToken } = await import('../../src/utils/token')
    postMock.mockResolvedValue({ data: { data: { access_token: 'n' } } })

    await refreshAccessToken()

    // 后端从旧 token 的 TID claim 里取 refresh token（auth/controller.go），
    // 所以这里必须是"过期但完整"的那个，不能省。
    const [, , config] = postMock.mock.calls[0]
    expect(config.headers.Authorization).toBe('Bearer stale-token')
  })

  it('失败后清空 pending，下一次还能重试', async () => {
    const { refreshAccessToken } = await import('../../src/utils/token')

    postMock.mockRejectedValueOnce(new Error('network'))
    await expect(refreshAccessToken()).rejects.toThrow('network')

    // 不做这个清理的话，一次网络抖动会让刷新永久卡死
    postMock.mockResolvedValueOnce({ data: { data: { access_token: 'ok' } } })
    await expect(refreshAccessToken()).resolves.toBe('ok')
    expect(postMock).toHaveBeenCalledTimes(2)
  })

  it('本地没有 token 时直接失败，不发请求', async () => {
    const { refreshAccessToken } = await import('../../src/utils/token')
    localStorage.removeItem('token')

    await expect(refreshAccessToken()).rejects.toThrow()
    expect(postMock).not.toHaveBeenCalled()
  })
})
