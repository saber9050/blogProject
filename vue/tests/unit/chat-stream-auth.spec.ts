import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

/**
 * 流式请求的 401 重试。
 *
 * 原来这里直接用原生 fetch，**没有 axios 那套 401 自动刷新**，
 * 所以 access token 一过期，聊天就弹出「token 已过期」红框并中断——
 * 明明长期凭证还有效，用户却得重新登录。
 *
 * 这里的断言有两条关键约束：
 *   1. 401 → 刷新 → 重试，用户全程无感
 *   2. 只重试一次（刷新成功但仍 401 时不能再刷，后端 rotation 会越刷越乱）
 */

const refreshMock = vi.fn()
const authFailureMock = vi.fn()

vi.mock('../../src/utils/token', () => ({
  refreshAccessToken: () => refreshMock(),
  handleAuthFailure: () => authFailureMock()
}))

vi.mock('../../src/api/index', () => ({
  default: { get: vi.fn(), post: vi.fn(), patch: vi.fn(), delete: vi.fn() }
}))

/** 造一个最小的 SSE 响应体：一个 done 事件就够，测试不关心内容 */
function doneStreamHead() {
  return {
    ok: true,
    status: 200,
    body: new ReadableStream({
      start(c) {
        c.enqueue(new TextEncoder().encode('event: done\ndata: {"assistant_message_id":1}\n\n'))
        c.close()
      }
    })
  }
}

function unauthorized() {
  return {
    ok: false,
    status: 401,
    json: async () => ({ code: 401, message: 'token 已过期' })
  }
}

describe('sendChatMessageStream 的 401 重试', () => {
  const fetchMock = vi.fn()

  beforeEach(() => {
    fetchMock.mockReset()
    refreshMock.mockReset()
    authFailureMock.mockReset()
    localStorage.clear()
    localStorage.setItem('token', 'stale')
    vi.stubGlobal('fetch', fetchMock)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.resetModules()
  })

  it('access token 过期时刷新并继续，不中断', async () => {
    const { sendChatMessageStream } = await import('../../src/api/chat')
    refreshMock.mockResolvedValue('fresh')
    fetchMock.mockResolvedValueOnce(unauthorized()).mockResolvedValueOnce(doneStreamHead())

    const onDone = vi.fn()
    await sendChatMessageStream(1, { content: 'hi' }, { onDone })

    expect(refreshMock).toHaveBeenCalledTimes(1)
    expect(fetchMock).toHaveBeenCalledTimes(2)
    // 第二次必须带新 token，否则重试也会 401
    const secondHeaders = fetchMock.mock.calls[1][1].headers
    expect(secondHeaders.Authorization).toBe('Bearer fresh')
    // 重试成功后照常走完流程
    expect(onDone).toHaveBeenCalledTimes(1)
    expect(authFailureMock).not.toHaveBeenCalled()
  })

  it('初次就成功时不刷新', async () => {
    const { sendChatMessageStream } = await import('../../src/api/chat')
    fetchMock.mockResolvedValueOnce(doneStreamHead())

    await sendChatMessageStream(1, { content: 'hi' }, {})

    expect(refreshMock).not.toHaveBeenCalled()
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('刷新后仍 401 → 登出，且只重试一次', async () => {
    const { sendChatMessageStream } = await import('../../src/api/chat')
    refreshMock.mockResolvedValue('fresh')
    fetchMock.mockResolvedValue(unauthorized())

    await expect(sendChatMessageStream(1, { content: 'hi' }, {})).rejects.toThrow('登录信息已过期')

    // 关键：不能再刷第二次，否则后端 rotation 会让整个会话乱掉
    expect(refreshMock).toHaveBeenCalledTimes(1)
    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(authFailureMock).toHaveBeenCalledTimes(1)
  })

  it('刷新失败（长期凭证也过期）→ 登出', async () => {
    const { sendChatMessageStream } = await import('../../src/api/chat')
    refreshMock.mockRejectedValue(new Error('refresh expired'))
    fetchMock.mockResolvedValueOnce(unauthorized())

    await expect(sendChatMessageStream(1, { content: 'hi' }, {})).rejects.toThrow('登录信息已过期')
    expect(authFailureMock).toHaveBeenCalledTimes(1)
  })
})
