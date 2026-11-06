import { describe, it, expect, beforeEach } from 'vitest'
import {
  createPill,
  highlightKeyword,
  serializeComposer,
  tokenizeMessage,
  truncateTitle,
  ZWSP
} from '../../src/utils/editor-serialize'

/**
 * 这层逻辑全是「不报错但结果不对」的类型：标题截断、零宽空格混进正文、
 * 浏览器回车插 <div> 导致多出换行、pill 的完整标题与显示文本不一致。
 * 靠手点验证成本极高，所以全部用纯函数测。
 */

let root: HTMLElement

beforeEach(() => {
  root = document.createElement('div')
})

describe('serializeComposer', () => {
  it('纯文本原样返回', () => {
    root.textContent = '这篇文章讲了什么'
    const payload = serializeComposer(root)
    expect(payload.text).toBe('这篇文章讲了什么')
    expect(payload.articleIds).toEqual([])
  })

  it('pill 序列化成 /完整标题，并收集 id', () => {
    root.appendChild(createPill(8, 'Redis 发布订阅和 Stream'))
    root.appendChild(document.createTextNode(' 和这篇文章的区别'))
    const payload = serializeComposer(root)
    expect(payload.text).toBe('/Redis 发布订阅和 Stream 和这篇文章的区别')
    expect(payload.articleIds).toEqual([8])
    expect(payload.refs[0]).toEqual({ article_id: 8, title: 'Redis 发布订阅和 Stream' })
  })

  it('超长标题用完整标题序列化（不用户界面上截断后的）', () => {
    const long = '这是一个特别特别长的文章标题用来测试截断逻辑是否正确'
    const pill = createPill(3, long)
    // 界面上是截断的
    expect(pill.textContent).not.toBe(`/${long}`)
    root.appendChild(pill)
    // 但序列化必须用完整标题，否则模型拿到的资料名对不上
    expect(serializeComposer(root).text).toBe(`/${long}`)
  })

  it('零宽空格被丢弃', () => {
    root.appendChild(document.createTextNode(`你好${ZWSP}世界`))
    expect(serializeComposer(root).text).toBe('你好世界')
  })

  it('pill 后面的零宽空格不进正文', () => {
    root.appendChild(createPill(8, 'A'))
    root.appendChild(document.createTextNode(ZWSP))
    root.appendChild(document.createTextNode('问题'))
    expect(serializeComposer(root).text).toBe('/A问题')
  })

  it('<br> 转成换行', () => {
    root.appendChild(document.createTextNode('第一行'))
    root.appendChild(document.createElement('br'))
    root.appendChild(document.createTextNode('第二行'))
    expect(serializeComposer(root).text).toBe('第一行\n第二行')
  })

  it('<div> 块级元素转成换行（浏览器按 Enter 会插它）', () => {
    const d1 = document.createElement('div')
    d1.textContent = '第一行'
    const d2 = document.createElement('div')
    d2.textContent = '第二行'
    root.appendChild(d1)
    root.appendChild(d2)
    expect(serializeComposer(root).text).toBe('第一行\n第二行')
  })

  it('首尾空白被 trim', () => {
    root.textContent = '  只有问题  '
    expect(serializeComposer(root).text).toBe('只有问题')
  })

  it('只有 pill 时文本非空（发送按钮要亮）', () => {
    root.appendChild(createPill(8, 'A'))
    expect(serializeComposer(root).text.length).toBeGreaterThan(0)
  })

  it('重复 pill 去重', () => {
    root.appendChild(createPill(8, 'A'))
    root.appendChild(createPill(8, 'A'))
    expect(serializeComposer(root).articleIds).toEqual([8])
  })
})

describe('truncateTitle', () => {
  it('不超过上限时原样返回', () => {
    expect(truncateTitle('短标题', 20)).toBe('短标题')
  })
  it('超过按字符（不是字节）截断', () => {
    const s = '一二三四五六七八九十一二三四五六七八九十一二三'
    const got = truncateTitle(s, 5)
    expect(got).toBe('一二三四五…')
  })
})

describe('tokenizeMessage', () => {
  it('按 article_refs 还原 pill', () => {
    const tokens = tokenizeMessage('/A 和 /B 有什么区别', [
      { article_id: 1, title: 'A' },
      { article_id: 2, title: 'B' }
    ])
    expect(tokens.map((t) => t.type)).toEqual(['pill', 'text', 'pill', 'text'])
    expect(tokens[0]).toEqual({ type: 'pill', text: '/A', articleId: 1 })
    expect(tokens[2]).toEqual({ type: 'pill', text: '/B', articleId: 2 })
  })

  it('没有 refs 时退化成纯文本', () => {
    const tokens = tokenizeMessage('/A 问题', null)
    expect(tokens).toEqual([{ type: 'text', text: '/A 问题' }])
  })

  it('标题对不上（文章改过名）时退化成纯文本，不影响阅读', () => {
    const tokens = tokenizeMessage('/旧标题 问题', [{ article_id: 1, title: '新标题' }])
    expect(tokens).toEqual([{ type: 'text', text: '/旧标题 问题' }])
  })

  it('每个 ref 只替换第一处', () => {
    const tokens = tokenizeMessage('/A 和 /A', [{ article_id: 1, title: 'A' }])
    const pills = tokens.filter((t) => t.type === 'pill')
    expect(pills).toHaveLength(1)
  })
})

describe('highlightKeyword', () => {
  it('命中时加 mark', () => {
    expect(highlightKeyword('Redis 持久化', 'redis')).toBe('<mark>Redis</mark> 持久化')
  })

  it('没命中时原样返回', () => {
    expect(highlightKeyword('Redis 持久化', 'mysql')).toBe('Redis 持久化')
  })

  it('标题里的 HTML 会被转义（防 XSS）', () => {
    const got = highlightKeyword('<img src=x onerror=alert(1)>', 'img')
    expect(got).not.toContain('<img')
    expect(got).toContain('&lt;')
  })

  it('关键字为空时不加 mark', () => {
    expect(highlightKeyword('标题', '')).toBe('标题')
  })
})
