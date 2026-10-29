/**
 * 分页计算与页码序列。
 *
 * 后台每个列表（用户/文章/评论/标签/分类/切分块）的分页逻辑完全一样，
 * 之前是写在 AdminView 里的局部函数，切分管理的面板拿不到就得再抄一份。
 * 抽出来共用，改规则只改一处。
 */

/** 总页数。0 条也返回 1，避免出现「第 1 页 / 共 0 页」这种自相矛盾的文案 */
export const calcPages = (total: number, size: number) => Math.ceil(total / size) || 1

/**
 * 生成要渲染的页码序列，超过 7 页时用 '...' 折叠。
 *
 * 5 个可见页码是刻意的：再多就会出现「页码把分页条挤到第二行」的情况，
 * 而这种列表的翻页需求通常只是「上一页/下一页 + 偶尔跳首末页」。
 */
export const getPageRange = (current: number, total: number): (number | '...')[] => {
  if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1)

  const pages: (number | '...')[] = []
  if (current <= 4) {
    // 靠前：1..5 ... 末页
    for (let i = 1; i <= 5; i++) pages.push(i)
    pages.push('...')
    pages.push(total)
  } else if (current >= total - 3) {
    // 靠后：首页 ... 末 5 页
    pages.push(1)
    pages.push('...')
    for (let i = total - 4; i <= total; i++) pages.push(i)
  } else {
    // 中间：首页 ... 当前±1 ... 末页
    pages.push(1)
    pages.push('...')
    for (let i = current - 1; i <= current + 1; i++) pages.push(i)
    pages.push('...')
    pages.push(total)
  }
  return pages
}
