import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { showToast, getToastState, type ToastItem } from '../../src/utils/toast'

describe('toast 工具', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    // 清空共享状态，保证用例之间互不干扰
    getToastState().value.splice(0)
  })

  afterEach(() => {
    vi.clearAllTimers()
    vi.useRealTimers()
  })

  it('默认类型为 info', () => {
    showToast('保存成功')

    const toasts = getToastState().value
    expect(toasts).toHaveLength(1)
    expect(toasts[0].message).toBe('保存成功')
    expect(toasts[0].type).toBe('info')
  })

  it('支持指定 success / error 类型', () => {
    showToast('操作成功', 'success')
    showToast('操作失败', 'error')

    const types = getToastState().value.map((t: ToastItem) => t.type)
    expect(types).toEqual(['success', 'error'])
  })

  it('多条消息的 id 唯一且递增', () => {
    showToast('第一条')
    showToast('第二条')
    showToast('第三条')

    const ids = getToastState().value.map((t: ToastItem) => t.id)
    expect(new Set(ids).size).toBe(3)
    expect(ids).toEqual([...ids].sort((a, b) => a - b))
  })

  it('3 秒后自动移除', () => {
    showToast('稍纵即逝')

    expect(getToastState().value).toHaveLength(1)

    vi.advanceTimersByTime(2999)
    expect(getToastState().value).toHaveLength(1)

    vi.advanceTimersByTime(1)
    expect(getToastState().value).toHaveLength(0)
  })

  it('只移除到期的那条，不影响其他消息', () => {
    showToast('先消失')
    vi.advanceTimersByTime(1000)
    showToast('后消失')

    vi.advanceTimersByTime(2000) // 第 3 秒：第一条到期
    const remaining = getToastState().value.map((t: ToastItem) => t.message)
    expect(remaining).toEqual(['后消失'])

    vi.advanceTimersByTime(1000) // 第 4 秒：第二条到期
    expect(getToastState().value).toHaveLength(0)
  })
})
