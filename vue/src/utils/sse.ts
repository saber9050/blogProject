/**
 * SSE 解析器（纯函数，可单测）。
 *
 * 为什么不用 EventSource：
 *   1. EventSource 只能发 GET，而发消息是 POST（要带 body 和 Authorization 头）
 *   2. EventSource 无法手动带 token（项目用 Bearer，不是 Cookie）
 * 所以只能 fetch + ReadableStream 自己解协议。
 */

export interface SSEFrame {
  event: string
  data: string
}

export interface SSEParser {
  /** 喂入一段解码后的文本，返回本次能完整解析出的帧 */
  feed(chunk: string): SSEFrame[]
  /** 流结束时调用：把残留在缓冲里的最后一帧吐出来 */
  flush(): SSEFrame[]
}

export function createSSEParser(): SSEParser {
  let buffer = ''

  const parseFrame = (raw: string): SSEFrame | null => {
    let event = 'message'
    const dataLines: string[] = []

    for (const rawLine of raw.split('\n')) {
      // 代理可能把行尾改成 \r\n，这里统一去掉
      const line = rawLine.replace(/\r$/, '')
      if (line.startsWith('event:')) {
        event = line.slice(6).trim()
      } else if (line.startsWith('data:')) {
        // data 后面可能有一个空格，只去掉一个（保留内容本身的缩进）
        dataLines.push(line.slice(5).replace(/^ /, ''))
      }
      // 其它字段（id: / retry:）本项目不用，直接忽略
    }

    if (dataLines.length === 0) return null
    return { event, data: dataLines.join('\n') }
  }

  const drain = (allowTail: boolean): SSEFrame[] => {
    const frames: SSEFrame[] = []
    let idx = buffer.indexOf('\n\n')
    while (idx !== -1) {
      const raw = buffer.slice(0, idx)
      buffer = buffer.slice(idx + 2)
      const frame = parseFrame(raw)
      if (frame) frames.push(frame)
      idx = buffer.indexOf('\n\n')
    }
    // 流结束时最后可能没有结尾空行（服务端写完就关连接），也要解出来
    if (allowTail && buffer.trim() !== '') {
      const frame = parseFrame(buffer)
      buffer = ''
      if (frame) frames.push(frame)
    }
    return frames
  }

  return {
    feed: (chunk: string) => {
      buffer += chunk
      return drain(false)
    },
    flush: () => drain(true)
  }
}
