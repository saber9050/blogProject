import { describe, it, expect } from 'vitest'
import { buildTocItems, pickActiveId, headingIdOf, type TocItem } from '../../src/utils/toc'

describe('headingIdOf', () => {
  it('按序号生成稳定 id', () => {
    expect(headingIdOf({ index: 1 })).toBe('article-h-1')
    expect(headingIdOf({ index: 12 })).toBe('article-h-12')
  })
})

describe('buildTocItems', () => {
  it('保留 1~3 级标题，丢掉更深的层级', () => {
    const items = buildTocItems([
      { text: '一级', level: 1 },
      { text: '二级', level: 2 },
      { text: '三级', level: 3 },
      { text: '四级', level: 4 }
    ])
    expect(items.map((i) => i.text)).toEqual(['一级', '二级', '三级'])
  })

  it('空标题不进目录', () => {
    const items = buildTocItems([
      { text: '', level: 2 },
      { text: '有字', level: 2 }
    ])
    expect(items).toHaveLength(1)
    expect(items[0].text).toBe('有字')
  })

  it('id 用原始下标，不受过滤影响', () => {
    // 第 2 个标题（index=2）是 4 级、被过滤掉，
    // 第 3 个标题仍要拿到 article-h-3，而不是过滤后的 article-h-2
    const items = buildTocItems([
      { text: 'A', level: 2 },
      { text: 'B', level: 4 },
      { text: 'C', level: 2 }
    ])
    expect(items.map((i) => i.id)).toEqual(['article-h-1', 'article-h-3'])
  })

  it('depth 以最浅标题为基准，`#` 起头和 `##` 起头都从一开始', () => {
    const fromHash = buildTocItems([
      { text: 'A', level: 1 },
      { text: 'B', level: 2 },
      { text: 'C', level: 3 }
    ])
    expect(fromHash.map((i) => i.depth)).toEqual([0, 1, 2])

    const fromH2 = buildTocItems([
      { text: 'A', level: 2 },
      { text: 'B', level: 3 }
    ])
    expect(fromH2.map((i) => i.depth)).toEqual([0, 1])
  })

  it('maxLevel 可配置', () => {
    const items = buildTocItems(
      [
        { text: 'A', level: 2 },
        { text: 'B', level: 3 }
      ],
      2
    )
    expect(items.map((i) => i.text)).toEqual(['A'])
  })
})

describe('pickActiveId', () => {
  const items: TocItem[] = [
    { id: 'h1', text: 'A', level: 2, depth: 0 },
    { id: 'h2', text: 'B', level: 2, depth: 0 },
    { id: 'h3', text: 'C', level: 2, depth: 0 }
  ]

  it('还没滚过第一个标题时高亮第一个', () => {
    expect(pickActiveId(items, [300, 800, 1300], 76)).toBe('h1')
  })

  it('取最后一个越过偏移线的标题', () => {
    // 第一个在线上方 24px（76-100？这里 40 < 76），第二个在下方
    expect(pickActiveId(items, [40, 500, 900], 76)).toBe('h1')
    expect(pickActiveId(items, [-320, 60, 900], 76)).toBe('h2')
    expect(pickActiveId(items, [-900, -400, 20], 76)).toBe('h3')
  })

  it('跳过还没渲染出来的标题（null）', () => {
    expect(pickActiveId(items, [-500, null, 20], 76)).toBe('h3')
  })

  it('空列表返回空串', () => {
    expect(pickActiveId([], [], 76)).toBe('')
  })
})
