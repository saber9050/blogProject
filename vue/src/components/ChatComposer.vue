<template>
  <div class="composer" :class="{ 'composer--streaming': streaming }">
    <!-- 候选浮层：贴在输入框上方。面板本身靠页面底部，向下弹会被视口裁掉 -->
    <div v-if="picker.open" class="composer__picker" role="listbox">
      <p v-if="picker.loading" class="composer__picker-hint">搜索中…</p>
      <p v-else-if="picker.items.length === 0" class="composer__picker-hint">没有匹配的文章</p>
      <button
        v-for="(item, i) in picker.items"
        v-else
        :key="item.id"
        type="button"
        role="option"
        :aria-selected="i === picker.active"
        class="composer__picker-item"
        :class="{ 'is-active': i === picker.active, 'is-picked': isPicked(item.id) }"
        :disabled="isPicked(item.id)"
        @mousedown.prevent="pick(i)"
      >
        <!-- 标题来自数据库，highlightKeyword 内部已做 HTML 转义，不是裸 v-html -->
        <span class="composer__picker-title" v-html="highlightKeyword(item.title, picker.keyword)"></span>
        <span v-if="isPicked(item.id)" class="composer__picker-tag">已引用</span>
      </button>
    </div>

    <div
      ref="editorEl"
      class="composer__editor"
      role="textbox"
      aria-multiline="true"
      :contenteditable="editable"
      :data-placeholder="placeholder"
      @input="onInput"
      @keydown="onKeydown"
      @beforeinput="onBeforeInput"
      @paste="onPaste"
      @compositionstart="composing = true"
      @compositionend="onCompositionEnd"
      @mousedown="onClickInside"
    ></div>

    <p v-if="hint" class="composer__hint">{{ hint }}</p>

    <div class="composer__actions">
      <button
        v-if="!streaming"
        class="composer__send"
        type="button"
        :disabled="!canSend"
        @click="submit"
      >
        发送
      </button>
      <button v-else class="composer__send composer__send--stop" type="button" @click="$emit('stop')">
        停止
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { fetchArticleCandidates, type ArticleCandidate } from '../api/chat'
import {
  createCursorPad,
  createPill,
  highlightKeyword,
  serializeComposer,
  ZWSP,
  type ComposerPayload
} from '../utils/editor-serialize'

const props = defineProps<{
  /** 服务端说不可聊时禁用输入 */
  disabled?: boolean
  /** 流式生成中：不可编辑，只显示「停止」 */
  streaming?: boolean
  /** 单次提问可引用的其他文章数上限（不含当前文章） */
  maxRefs: number
  /** 当前文章 id，从候选里排除 */
  currentArticleId: number
}>()

const emit = defineEmits<{
  (e: 'send', payload: ComposerPayload): void
  (e: 'stop'): void
}>()

const editorEl = ref<HTMLElement | null>(null)
const composing = ref(false)
const hint = ref('')
const canSend = ref(false)

// 流式中彻底禁止编辑：否则用户会以为「我改了但已经发出去了」
const editable = computed(() => !props.disabled && !props.streaming)
const placeholder = computed(() =>
  props.maxRefs > 0 ? '就这篇文章提问，输入 / 可引用其他文章…' : '就这篇文章提问…'
)

// ===== 候选浮层 =====

const picker = reactive({
  open: false,
  loading: false,
  items: [] as ArticleCandidate[],
  active: 0,
  keyword: ''
})

// 组件级缓存（不落 localStorage）：同一个 keyword 反复开关不该重复请求
const cache = new Map<string, { at: number; items: ArticleCandidate[] }>()
const CACHE_TTL = 60_000
let searchTimer: number | null = null

// `/` 的位置。选中候选时要从这里开始删，把 `/` 和后续关键字一起吃掉，
// 不然 pill 自己带 `/`，会变成 `//标题`
let slashAnchor: { node: Text; offset: number } | null = null

const pickedIds = ref<number[]>([])

function isPicked(id: number): boolean {
  return pickedIds.value.includes(id)
}

function firstSelectable(): number {
  const idx = picker.items.findIndex((it) => !isPicked(it.id))
  return idx < 0 ? 0 : idx
}

async function search(keyword: string) {
  const cached = cache.get(keyword)
  if (cached && Date.now() - cached.at < CACHE_TTL) {
    picker.items = cached.items
    picker.active = firstSelectable()
    picker.loading = false
    return
  }

  picker.loading = true
  try {
    const items = await fetchArticleCandidates(keyword, props.currentArticleId, 20)
    cache.set(keyword, { at: Date.now(), items })
    picker.items = items
    picker.active = firstSelectable()
  } catch {
    picker.items = []
  } finally {
    picker.loading = false
  }
}

function scheduleSearch(keyword: string) {
  if (searchTimer !== null) window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(() => {
    searchTimer = null
    // 请求回来时用户可能已经关掉浮层了，别把过期的结果塞回去
    if (!picker.open) return
    void search(keyword)
  }, 200)
}

function openPicker(keyword: string) {
  if (pickedIds.value.length >= props.maxRefs) {
    // 到上限不弹浮层，只提示。继续弹的话用户选了也插不进去，更困惑
    hint.value = `最多引用 ${props.maxRefs} 篇其他文章`
    picker.open = false
    return
  }
  picker.open = true
  picker.keyword = keyword
  picker.items = []
  void search(keyword)
}

function closePicker() {
  picker.open = false
  picker.items = []
  slashAnchor = null
  if (searchTimer !== null) {
    window.clearTimeout(searchTimer)
    searchTimer = null
  }
}

// ===== 输入处理 =====

function currentCaret(): { node: Node; offset: number } | null {
  const sel = window.getSelection()
  if (!sel || sel.rangeCount === 0) return null
  const range = sel.getRangeAt(0)
  if (!range.collapsed) return null
  return { node: range.startContainer, offset: range.startOffset }
}

function onInput() {
  syncHeight()
  clearPendingDelete()
  hint.value = ''

  // 流式中途由外部把 contenteditable 关掉，但仍要更新发送按钮状态
  canSend.value = serializeComposer(editorEl.value!).text.length > 0

  // 重新采集 pill 列表：用于发送上限与「已引用置灰」
  syncPickedIds()

  const caret = currentCaret()
  if (!caret) return

  // 刚打进一个 `/`
  if (
    caret.node.nodeType === Node.TEXT_NODE &&
    caret.offset > 0 &&
    (caret.node.textContent || '')[caret.offset - 1] === '/' &&
    !picker.open
  ) {
    slashAnchor = { node: caret.node as Text, offset: caret.offset }
    openPicker('')
    return
  }

  if (!picker.open) return

  // 浮层已开：`/` 之后到光标的连续字符就是关键字。
  // ⚠️ 不因为「看起来像路径」就关闭浮层——用户可能真想搜标题含 redis 的文章
  const kw = keywordAfterSlash()
  if (kw === null) {
    closePicker()
    return
  }
  if (kw !== picker.keyword) {
    picker.keyword = kw
    scheduleSearch(kw)
  }
}

/** 取 `/` 之后到光标之间的文本；跨节点或含空白则返回 null（表示不再是候选输入） */
function keywordAfterSlash(): string | null {
  if (!slashAnchor) return null
  const caret = currentCaret()
  if (!caret || caret.node !== slashAnchor.node) return null
  const text = slashAnchor.node.textContent || ''
  const kw = text.slice(slashAnchor.offset, caret.offset)
  // 出现空白说明用户已经在写别的内容了
  if (/[\s\u200B]/.test(kw)) return null
  return kw
}

function onCompositionEnd() {
  composing.value = false
}

function onKeydown(e: KeyboardEvent) {
  // 中文输入法：拼音还没上屏时按 Enter 不是「发送」，是「上屏」
  if (composing.value || e.isComposing || e.keyCode === 229) return

  if (picker.open) {
    // 浮层打开时，键盘事件全部交给浮层——否则用户按 Enter 想选中，消息却发出去了
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      moveActive(1)
      return
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault()
      moveActive(-1)
      return
    }
    if (e.key === 'Enter' || e.key === 'Tab') {
      e.preventDefault()
      pick(picker.active)
      return
    }
    if (e.key === 'Escape') {
      e.preventDefault()
      // 关闭后 `/关键字` 原样保留为普通文本
      closePicker()
      return
    }
  }

  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    submit()
    return
  }

  if (e.key === 'Enter' && e.shiftKey) {
    // 统一插 <br>，避免浏览器插 <div> 导致序列化时多出换行
    e.preventDefault()
    insertNodes([document.createElement('br')])
  }
}

function moveActive(delta: number) {
  const total = picker.items.length
  if (total === 0) return
  let next = picker.active
  for (let i = 0; i < total; i++) {
    next = (next + delta + total) % total
    if (!isPicked(picker.items[next].id)) break
  }
  picker.active = next
}

/**
 * beforeinput 处理退格。
 *
 * 为什么不用 keydown：部分安卓输入法的软键盘退格不派发 keydown，
 * 那边写「两次确认」会直接退化成「一次删光」。beforeinput 覆盖面更广。
 */
function onBeforeInput(e: Event) {
  const ev = e as InputEvent
  if (ev.inputType !== 'deleteContentBackward' && ev.inputType !== 'deleteContentForward') return

  const caret = currentCaret()
  if (!caret) return

  const pill = pillLeftOfCaret(caret)
  if (!pill) {
    clearPendingDelete()
    return
  }

  // 无论第几次都先拦住：不拦的话浏览器默认行为会立刻删掉整个原子节点
  ev.preventDefault()

  if (pendingDelete?.el === pill) {
    removePill(pill)
    clearPendingDelete()
    afterMutate()
    return
  }

  markPendingDelete(pill)
}

type PendingDelete = { el: HTMLElement; timer: number }

let pendingDelete: PendingDelete | null = null
const CONFIRM_MS = 2000

/** 光标左侧紧贴的 pill（含 pill 后的零宽空格的情况） */
function pillLeftOfCaret(caret: { node: Node; offset: number }): HTMLElement | null {
  let node: Node | null = caret.node
  let offset = caret.offset

  // 光标可能在 pill 后面的零宽空格里（offset=1）
  if (node.nodeType === Node.TEXT_NODE && offset > 0) {
    const text = node.textContent || ''
    // 零宽空格之后没有其它内容时，视为紧贴 pill 右侧
    if (text.slice(0, offset).replace(/\u200B/g, '').length > 0) return null
  }

  // 光标前面若不是任何实体内容（只有零宽空格），就往前找 pill
  let prev: Node | null = node.previousSibling
  while (prev && prev.nodeType === Node.TEXT_NODE && (prev.textContent || '').replace(/\u200B/g, '') === '') {
    prev = prev.previousSibling
  }
  if (prev && prev.nodeType === Node.ELEMENT_NODE) {
    const el = prev as HTMLElement
    if (el.classList.contains('ref-pill')) return el
  }
  return null
}

function markPendingDelete(pill: HTMLElement) {
  clearPendingDelete()
  pill.classList.add('ref-pill--pending')
  const title = pill.dataset.title || ''
  hint.value = `再按一次退格删除「/${title}」`
  pendingDelete = {
    el: pill,
    timer: window.setTimeout(() => clearPendingDelete(), CONFIRM_MS)
  }
}

function clearPendingDelete() {
  if (pendingDelete) {
    // 必须清 timer：只清样式的话，2 秒后会去操作一个可能已被删除的节点
    window.clearTimeout(pendingDelete.timer)
    pendingDelete.el.classList.remove('ref-pill--pending')
    pendingDelete = null
  }
  if (hint.value.startsWith('再按一次退格')) hint.value = ''
}

function removePill(pill: HTMLElement) {
  // 连它后面的零宽空格一起删：pill 没了那个字符就是垃圾，视觉上留个空位
  const next = pill.nextSibling
  pill.remove()
  if (next && next.nodeType === Node.TEXT_NODE && next.textContent === ZWSP) {
    next.parentNode?.removeChild(next)
  }
  syncPickedIds()
}

function onPaste(e: ClipboardEvent) {
  // 只取纯文本。这也是安全防线：否则外部网页可以伪造
  // <span data-article-id="8"> 混进 payload，等于伪造引用
  e.preventDefault()
  const text = e.clipboardData?.getData('text/plain') || ''
  if (!text) return
  insertPlainText(text)
}

function insertPlainText(text: string) {
  // 多行粘贴：用 <br> 分段，避免塞进一个含 \n 的文本节点（不会渲染成换行）
  const parts = text.split(/\r?\n/)
  const nodes: Node[] = []
  parts.forEach((line, i) => {
    if (i > 0) nodes.push(document.createElement('br'))
    if (line) nodes.push(document.createTextNode(line.replace(/\u200B/g, '')))
  })
  insertNodes(nodes)
}

function insertNodes(nodes: Node[]) {
  const sel = window.getSelection()
  if (!sel || sel.rangeCount === 0) return
  const range = sel.getRangeAt(0)
  range.deleteContents()
  const frag = document.createDocumentFragment()
  nodes.forEach((n) => frag.appendChild(n))
  range.insertNode(frag)
  // 光标放到最后一个节点之后
  const last = nodes[nodes.length - 1]
  if (last) {
    range.setStartAfter(last)
    range.collapse(true)
    sel.removeAllRanges()
    sel.addRange(range)
  }
  afterMutate()
}

// ===== 选中候选 =====

function pick(index: number) {
  const item = picker.items[index]
  if (!item || isPicked(item.id)) return

  const pill = createPill(item.id, item.title)

  // 优先用 `/` 的位置删掉 `/关键字`；拿不到（DOM 已变）就在光标处插入
  const range = rangeOfSlash()
  if (range) {
    range.deleteContents()
    const pad = createCursorPad()
    const frag = document.createDocumentFragment()
    frag.appendChild(pill)
    frag.appendChild(pad)
    range.insertNode(frag)
    // 光标放到零宽空格之后，用户可以直接接着打字
    const after = document.createRange()
    after.setStart(pad, 1)
    after.collapse(true)
    const sel = window.getSelection()
    sel?.removeAllRanges()
    sel?.addRange(after)
  } else {
    insertNodes([pill, createCursorPad()])
  }

  closePicker()
  syncPickedIds()
  afterMutate()
  syncHeight()
  editorEl.value?.focus()
}

function rangeOfSlash(): Range | null {
  if (!slashAnchor || !slashAnchor.node.isConnected) return null
  const caret = currentCaret()
  // 只有光标还在同一个文本节点里、且在 `/` 之后，才能精确删掉这一段
  if (!caret || caret.node !== slashAnchor.node || caret.offset < slashAnchor.offset) return null
  const range = document.createRange()
  range.setStart(slashAnchor.node, slashAnchor.offset - 1) // 连同 `/` 一起删
  range.setEnd(slashAnchor.node, caret.offset)
  return range
}

function syncPickedIds() {
  const el = editorEl.value
  if (!el) {
    pickedIds.value = []
    return
  }
  const ids: number[] = []
  el.querySelectorAll<HTMLElement>('.ref-pill').forEach((p) => {
    const id = Number(p.dataset.articleId)
    if (id > 0) ids.push(id)
  })
  pickedIds.value = ids
}

function afterMutate() {
  canSend.value = serializeComposer(editorEl.value!).text.length > 0
}

function onClickInside(e: MouseEvent) {
  // 点 pill 不删除，只把光标挪到它后面（删除统一走两次退格）
  const target = e.target as HTMLElement
  const pill = target.closest?.('.ref-pill') as HTMLElement | null
  if (!pill) return
  e.preventDefault()
  const range = document.createRange()
  range.setStartAfter(pill)
  range.collapse(true)
  const sel = window.getSelection()
  sel?.removeAllRanges()
  sel?.addRange(range)
  editorEl.value?.focus()
}

// ===== 高度与提交 =====

function syncHeight() {
  const el = editorEl.value
  if (!el) return
  el.style.height = 'auto'
  const line = 22
  el.style.height = `${Math.min(el.scrollHeight, line * 5)}px`
}

function submit() {
  if (!canSend.value || props.streaming || props.disabled) return
  const el = editorEl.value
  if (!el) return

  const payload = serializeComposer(el)
  if (!payload.text.trim()) return

  emit('send', payload)
  clear()
}

function clear() {
  const el = editorEl.value
  if (!el) return
  // 必须显式清 innerHTML：只删子节点时浏览器可能留下一个 <br>，
  // 那样 CSS 的 :empty::before 占位符就永远不显示了
  el.innerHTML = ''
  el.style.height = 'auto'
  canSend.value = false
  pickedIds.value = []
  hint.value = ''
  clearPendingDelete()
  closePicker()
}

function focus() {
  editorEl.value?.focus()
}

defineExpose({ clear, focus })
</script>

<style scoped>
.composer {
  position: relative;
  border-top: 1px solid var(--border);
  background: var(--surface-strong);
  padding: 10px 12px 12px;
}

.composer__editor {
  min-height: 40px;
  max-height: 110px;
  overflow-y: auto;
  padding: 9px 11px;
  border: 1px solid var(--border-strong);
  border-radius: 10px;
  font-size: 0.85rem;
  line-height: 22px;
  color: var(--text-strong);
  outline: none;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.composer__editor:focus {
  border-color: var(--primary-500);
  box-shadow: 0 0 0 3px var(--primary-soft);
}

/* contenteditable 没有原生 placeholder；只有真的空（无子节点）才触发 */
.composer__editor:empty::before {
  content: attr(data-placeholder);
  color: var(--text-faint);
  pointer-events: none;
}

.composer--streaming .composer__editor {
  background: var(--surface-soft);
  color: var(--text-secondary);
}

.composer__hint {
  margin: 6px 2px 0;
  font-size: 0.72rem;
  color: var(--text-faint);
}

.composer__actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 8px;
}

.composer__send {
  height: 34px;
  padding: 0 18px;
  border: none;
  border-radius: 10px;
  background: var(--primary-500);
  color: #fff;
  font-size: 0.82rem;
  cursor: pointer;
}

.composer__send:disabled {
  background: var(--border-strong);
  cursor: not-allowed;
}

.composer__send--stop {
  background: var(--text-muted);
}

/* ===== 候选浮层 ===== */

.composer__picker {
  position: absolute;
  left: 12px;
  right: 12px;
  bottom: calc(100% + 6px);
  max-height: 240px;
  overflow-y: auto;
  padding: 4px;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--surface-strong);
  box-shadow: var(--shadow-soft);
  z-index: 5;
}

.composer__picker-hint {
  margin: 0;
  padding: 10px;
  text-align: center;
  font-size: 0.78rem;
  color: var(--text-faint);
}

.composer__picker-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  width: 100%;
  height: 36px;
  padding: 0 10px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--text-secondary);
  font-size: 0.82rem;
  text-align: left;
  cursor: pointer;
}

.composer__picker-item.is-active {
  background: var(--accent-soft);
  color: var(--accent-500);
}

.composer__picker-item.is-picked {
  color: var(--text-faint);
  cursor: not-allowed;
}

.composer__picker-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.composer__picker-title :deep(mark) {
  background: rgba(59, 130, 246, 0.22);
  color: inherit;
  border-radius: 3px;
}

.composer__picker-tag {
  flex-shrink: 0;
  font-size: 0.68rem;
  color: var(--text-faint);
}

/* ===== 引用 pill（照截图）===== */

.composer__editor :deep(.ref-pill) {
  display: inline-block;
  vertical-align: baseline;
  margin: 0 2px;
  padding: 1px 6px;
  border-radius: 6px;
  background: var(--accent-soft);
  color: var(--accent-500);
  font-weight: 600;
  font-size: inherit;
  cursor: default;
  user-select: none;
  white-space: nowrap;
}

.composer__editor :deep(.ref-pill:hover) {
  background: rgba(59, 130, 246, 0.2);
}

/* 待删态：删除线 + 虚线红框，2 秒后自动复位 */
.composer__editor :deep(.ref-pill--pending) {
  background: rgba(59, 130, 246, 0.28);
  text-decoration: line-through;
  outline: 1px dashed var(--danger);
}
</style>
