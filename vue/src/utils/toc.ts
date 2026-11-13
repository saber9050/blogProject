/**
 * 文章目录（TOC）的数据构造与锚点规则。
 *
 * 抽成独立文件而不是写在 ArticleDetailView 里，是因为「id 规则」必须与
 * MdPreview 共用同一个函数：两边各写一套，改了一边另一边不会报错，
 * 只会表现为「点目录没反应」这种很难排查的静默故障。
 */

/** 目录里最多展示到几级标题 */
export const TOC_MAX_LEVEL = 3

export interface TocItem {
  /** DOM 里对应标题的 id */
  id: string
  text: string
  /** 1~6 原始层级 */
  level: number
  /**
   * 相对「本文最浅标题」的缩进层级（0 起）。
   *
   * 不能直接用 level 缩进：有的文章用 `#` 当章节、有的用 `##`，
   * 按绝对层级算，后者的所有条目都会莫名其妙缩进一格。
   */
  depth: number
}

/** MdPreview 的 onGetCatalog 回调参数（只取用得到的字段） */
export interface CatalogHeading {
  text: string
  level: number
}

/**
 * 标题 id 规则：article-h-1、article-h-2 ……
 *
 * index 是标题在全文里的序号（1 起），由 md-editor-v3 自己递增，
 * 所以「渲染时的 id」与「目录里的 id」天然一致。
 *
 * 不用默认的 slug 规则：它会对中文标点、重复标题做各种处理，
 * 自己再实现一遍必然与库的实现漂移。
 */
export function headingIdOf({ index }: { index: number }): string {
  return `article-h-${index}`
}

/**
 * 把解析出的标题列表转成目录项。
 *
 * 过滤规则：
 *   - 空标题丢掉（`## ` 后面没字的，点了也不知道跳哪）
 *   - 超过 TOC_MAX_LEVEL 的丢掉（4 级往下常是「1. 2. 3.」这种颗粒度，
 *     全塞进侧栏会变成一面墙）
 *
 * ⚠️ id 的序号用的是**原始列表下标**（i+1），不是过滤后的下标：
 * 一旦过滤掉某些标题，两者就不相等了，用过滤后的下标会指向错误的节点。
 */
export function buildTocItems(
  list: CatalogHeading[],
  maxLevel: number = TOC_MAX_LEVEL
): TocItem[] {
  const kept: TocItem[] = []
  list.forEach((h, i) => {
    if (!h.text || h.level > maxLevel) return
    kept.push({ id: headingIdOf({ index: i + 1 }), text: h.text, level: h.level, depth: 0 })
  })
  if (kept.length === 0) return kept
  // 以本文实际用到的最浅层级为基准，`#` 起头和 `##` 起头的文章都能对齐
  const minLevel = Math.min(...kept.map((i) => i.level))
  kept.forEach((i) => {
    i.depth = Math.min(i.level - minLevel, 2)
  })
  return kept
}

/**
 * 找出「当前读到的小节」：从上往下最后一个已经越过头部偏移线的标题。
 *
 * 不用 IntersectionObserver：它的回调是批量的，还得维护一整套可见集合；
 * 这里只关心「谁在线上方」，线性扫一遍（最多几十项）更简单也更准。
 *
 * @param positions 与 items 一一对应的标题顶部坐标（相对视口）；缺失的传 null
 * @param offset    头部偏移线（吸顶导航高度 + 留白）
 */
export function pickActiveId(
  items: TocItem[],
  positions: Array<number | null>,
  offset: number
): string {
  if (items.length === 0) return ''
  let current = items[0].id
  for (let i = 0; i < items.length; i++) {
    const top = positions[i]
    if (top == null) continue
    if (top - offset <= 0) current = items[i].id
    else break
  }
  return current
}
