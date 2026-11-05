import { describe, it, expect } from 'vitest'
import { createSSEParser } from '../../src/utils/sse'

/**
 * SSE 解析是流式问答的命门，但它是个纯函数——不需要网络也不需要后端。
 *
 * 真实场景里最常见的两个坑都在这里覆盖：
 *   1. 一个 TCP 包里可能只有半个事件（必须缓冲）
 *   2. 一个包里可能有多个事件（必须循环取出）
 * 少任何一个，前端都会表现为「偶发丢字 / 卡住」，而且难以稳定复现。
 */
describe('SSE 解析器', () => {
  it('解析单帧', () => {
    const p = createSSEParser()
    const frames = p.feed('event: delta\ndata: {"content":"你好"}\n\n')
    expect(frames).toHaveLength(1)
    expect(frames[0].event).toBe('delta')
    expect(frames[0].data).toBe('{"content":"你好"}')
  })

  it('一次喂入多帧', () => {
    const p = createSSEParser()
    const frames = p.feed(
      'event: meta\ndata: {"a":1}\n\nevent: delta\ndata: {"content":"x"}\n\n'
    )
    expect(frames.map((f) => f.event)).toEqual(['meta', 'delta'])
  })

  it('跨 chunk 的半帧会被正确缓冲', () => {
    const p = createSSEParser()
    expect(p.feed('event: del')).toHaveLength(0)
    expect(p.feed('ta\ndata: {"content"')).toHaveLength(0)
    const frames = p.feed(':"hi"}\n\n')
    expect(frames).toHaveLength(1)
    expect(frames[0].event).toBe('delta')
    expect(frames[0].data).toBe('{"content":"hi"}')
  })

  it('兼容 \\r\\n 行尾（代理可能改写）', () => {
    const p = createSSEParser()
    const frames = p.feed('event: delta\r\ndata: {"content":"ok"}\r\n\r\n')
    // \r\n\r\n 不是 \n\n，所以这一帧要等 flush 才能取出
    const all = frames.concat(p.flush())
    expect(all).toHaveLength(1)
    expect(all[0].data).toBe('{"content":"ok"}')
  })

  it('流结束时没有结尾空行也能取出最后一帧', () => {
    const p = createSSEParser()
    const inFeed = p.feed('event: done\ndata: {"rounds":1}')
    expect(inFeed).toHaveLength(0)
    const tail = p.flush()
    expect(tail).toHaveLength(1)
    expect(tail[0].event).toBe('done')
  })

  it('忽略没有 data 的帧', () => {
    const p = createSSEParser()
    const frames = p.feed('event: ping\n\n')
    expect(frames).toHaveLength(0)
  })

  it('data 里的换行会被拼回单行', () => {
    const p = createSSEParser()
    const frames = p.feed('event: delta\ndata: line1\ndata: line2\n\n')
    expect(frames[0].data).toBe('line1\nline2')
  })
})
