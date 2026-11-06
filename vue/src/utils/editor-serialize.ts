/**
 * contenteditable 输入框与后端载荷之间的转换（纯函数，配套单测）。
 *
 * 单独成文件的原因：这里全是「不报错但结果不对」的逻辑——标题截断、零宽空格、
 * 浏览器插入的 <div>、pill 的完整标题与显示文本不一致。这些混在组件里只能靠手点验证。
 */

import type { ArticleRef } from '../api/chat'

/** 序列化结果，即发消息时的 content + article_ids */
export interface ComposerPayload {
  text: string
  articleIds: number[]
  /**
   * 引用文章 id + 完整标题。
   *
   * 带上标题是为了「刚发出去的消息」能立刻显示 pill：此时还没请求后端，
   * 只有 id 没法渲染。不要试图从 text 里用正则反推标题——标题里可能有 `/`，
   * 反推必然出错。
   */
  refs: ArticleRef[]
}

/** 历史消息渲染用的 token：普通文本 / 引用 pill */
export interface MessageToken {
  type: 'text' | 'pill'
  text: string
  articleId?: number
}

/** pill 显示时标题截断长度，超过部分用省略号（完整标题放 data-title） */
export const PILL_TITLE_MAX = 20

/** 零宽空格：垫在 pill 后面让光标有落脚点 */
export const ZWSP = '\u200B'

export function truncateTitle(title: string, max = PILL_TITLE_MAX): string {
  const chars = Array.from(title)
  if (chars.length <= max) return title
  return chars.slice(0, max).join('') + '…'
}

/**
 * 序列化 contenteditable 的内容。
 *
 * 规则：
 *   - 文本节点原样拼接，零宽空格丢弃（它只是光标垫片）
 *   - .ref-pill 拼成 /完整标题，并把 article_id 收进列表
 *     ⚠️ 用 data-title 而不是可见文本：超过 20 字的标题在界面上是截断的
 *   - <br> 和块级元素（浏览器回车会插 <div>）转成换行
 *   - 首尾空白 trim
 */
export function serializeComposer(root: HTMLElement): ComposerPayload {
  const articleIds: number[] = []
  const refs: ArticleRef[] = []
  let text = ''

  const appendText = (s: string) => {
    text += s.replace(/\u200B/g, '')
  }

  const walk = (node: Node) => {
    if (node.nodeType === Node.TEXT_NODE) {
      appendText(node.textContent || '')
      return
    }
    if (node.nodeType !== Node.ELEMENT_NODE) return

    const el = node as HTMLElement
    if (el.classList.contains('ref-pill')) {
      const id = Number(el.dataset.articleId)
      const title = el.dataset.title || ''
      if (id > 0) {
        articleIds.push(id)
        refs.push({ article_id: id, title })
        text += `/${title}`
      } else {
        // 数据坏了就按可见文本降级，至少别把用户输入吃掉
        text += el.textContent || ''
      }
      return
    }
    if (el.tagName === 'BR') {
      text += '\n'
      return
    }

    const isBlock = el.tagName === 'DIV' || el.tagName === 'P'
    if (isBlock && text && !text.endsWith('\n')) text += '\n'
    el.childNodes.forEach(walk)
    if (isBlock && text && !text.endsWith('\n')) text += '\n'
  }

  root.childNodes.forEach(walk)

  // 去重：同一篇文章理论上一轮只引一次，但粘贴/异常操作下可能出现重复
  const uniqIds: number[] = []
  const uniqRefs: ArticleRef[] = []
  refs.forEach((r) => {
    if (uniqIds.includes(r.article_id)) return
    uniqIds.push(r.article_id)
    uniqRefs.push(r)
  })
  return { text: text.trim(), articleIds: uniqIds, refs: uniqRefs }
}

/** 创建一个原子 pill 节点 */
export function createPill(articleId: number, title: string): HTMLSpanElement {
  const span = document.createElement('span')
  span.className = 'ref-pill'
  span.setAttribute('contenteditable', 'false')
  span.dataset.articleId = String(articleId)
  // 完整标题存在这里：序列化与 tooltip 都用它，显示文本才做截断
  span.dataset.title = title
  span.title = title
  span.textContent = `/${truncateTitle(title)}`
  return span
}

/** 创建一个零宽空格文本节点（pill 后垫光标用） */
export function createCursorPad(): Text {
  return document.createTextNode(ZWSP)
}

/**
 * 把历史消息的纯文本 content 还原成 token 列表。
 *
 * 为什么不在 content 里存占位符（如 [[a:12]]）：那样历史消息在接口层就不可读了，
 * 而且用户手打的 `/标题` 与真正的引用在文本上无法区分——反正精确 id 已在 article_refs 里。
 *
 * 按 article_refs 的顺序在 content 里找 `/${title}`，每个只替换第一处。
 * 找不到（用户后来改了文章标题）就退化成纯文本，不影响阅读。
 */
export function tokenizeMessage(content: string, refs: ArticleRef[] | null): MessageToken[] {
  if (!content) return []

  const marks: { start: number; end: number; ref: ArticleRef }[] = []
  let cursor = 0
  for (const ref of refs || []) {
    if (!ref?.title) continue
    const needle = `/${ref.title}`
    const idx = content.indexOf(needle, cursor)
    if (idx < 0) continue
    marks.push({ start: idx, end: idx + needle.length, ref })
    cursor = idx + needle.length
  }
  if (marks.length === 0) {
    return [{ type: 'text', text: content }]
  }

  const tokens: MessageToken[] = []
  let pos = 0
  for (const m of marks) {
    if (m.start > pos) {
      tokens.push({ type: 'text', text: content.slice(pos, m.start) })
    }
    tokens.push({ type: 'pill', text: `/${m.ref.title}`, articleId: m.ref.article_id })
    pos = m.end
  }
  if (pos < content.length) {
    tokens.push({ type: 'text', text: content.slice(pos) })
  }
  return tokens
}

/**
 * 在文本里给 keyword 加 <mark> 高亮，返回可 v-html 的字符串。
 *
 * ⚠️ 必须先转义再拼标签：文章标题来自数据库（管理员可编辑），
 * 直接拼进去就是一个存储型 XSS 的口子。
 */
export function highlightKeyword(title: string, keyword: string): string {
  const safe = escapeHtml(title)
  const kw = keyword.trim()
  if (!kw) return safe

  const lowerTitle = safe.toLowerCase()
  const lowerKw = escapeHtml(kw).toLowerCase()
  const idx = lowerTitle.indexOf(lowerKw)
  if (idx < 0) return safe

  const len = lowerKw.length
  return (
    safe.slice(0, idx) +
    '<mark>' +
    safe.slice(idx, idx + len) +
    '</mark>' +
    safe.slice(idx + len)
  )
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}
