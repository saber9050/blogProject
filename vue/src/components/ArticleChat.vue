<template>
  <div class="ai-chat">
    <!-- 悬浮入口 -->
    <button
      v-if="!open"
      class="ai-chat__fab"
      type="button"
      aria-label="打开 AI 答疑助手"
      title="AI 答疑助手"
      @click="openPanel"
    >
      <img src="/logo-mark.svg" alt="" class="ai-chat__fab-icon" />
    </button>

    <!-- 对话面板 -->
    <section v-else class="ai-chat__panel" aria-label="AI 答疑助手">
      <header class="ai-chat__header">
        <div class="ai-chat__title">
          <img src="/logo-mark.svg" alt="" class="ai-chat__logo" />
          <span>AI 答疑</span>
          <span v-if="currentModel" class="ai-chat__model">{{ currentModel }}</span>
        </div>
        <div class="ai-chat__actions">
          <button
            class="ai-chat__icon-btn"
            type="button"
            title="新建会话"
            :disabled="streaming"
            @click="startNewSession"
          >
            &#65291;
          </button>
          <button class="ai-chat__icon-btn" type="button" title="收起" @click="closePanel">
            &#8722;
          </button>
        </div>
      </header>

      <!-- 未登录：入口照常显示，点了告诉他要先登录 -->
      <div v-if="!loggedIn" class="ai-chat__notice">
        <p class="ai-chat__notice-title">登录后才能使用 AI 答疑助手</p>
        <p class="ai-chat__notice-desc">登录后可保存对话记录，随时回来接着聊。</p>
        <button class="btn-sm ai-chat__notice-btn" type="button" @click="goLogin">去登录</button>
      </div>

      <!-- 服务端说不可用（没配模型 / 文章不可聊） -->
      <div v-else-if="context && !context.enabled" class="ai-chat__notice">
        <p class="ai-chat__notice-title">AI 助手暂不可用</p>
        <p class="ai-chat__notice-desc">{{ context.reason || '请稍后再试' }}</p>
      </div>

      <template v-else>
        <div ref="listEl" class="ai-chat__messages">
          <div v-if="loadingHistory" class="ai-chat__hint">正在加载会话…</div>

          <!-- 空态引导 -->
          <div v-else-if="messages.length === 0" class="ai-chat__empty">
            <p class="ai-chat__empty-title">
              就{{ context?.article_title ? `《${context.article_title}》` : '这篇文章' }}提问
            </p>
            <p class="ai-chat__empty-desc">AI 会基于文章内容回答，文章里没有的会明确说没提到。</p>
            <div class="ai-chat__samples">
              <button
                v-for="s in sampleQuestions"
                :key="s"
                class="ai-chat__sample"
                type="button"
                @click="useSample(s)"
              >
                {{ s }}
              </button>
            </div>
          </div>

          <!-- 消息 -->
          <div
            v-for="m in messages"
            :key="m.key"
            class="ai-chat__msg"
            :class="`ai-chat__msg--${m.role}`"
          >
            <div
              class="ai-chat__bubble"
              :class="{
                'ai-chat__bubble--refused': m.refused,
                'ai-chat__bubble--error': m.error
              }"
            >
              <!-- 助手消息渲染 Markdown；用户消息保留换行 -->
              <MdPreview
                v-if="m.role === 'assistant' && !m.refused"
                :model-value="m.content || (m.streaming ? '' : '')"
                class="ai-chat__markdown"
              />
              <p v-else class="ai-chat__plain">{{ m.content }}</p>

              <span v-if="m.streaming" class="ai-chat__caret" aria-hidden="true"></span>

              <p v-if="m.toolHint" class="ai-chat__tool-hint">{{ m.toolHint }}</p>
              <p v-if="m.error" class="ai-chat__error-text">{{ m.error }}</p>
              <p v-if="m.interrupted" class="ai-chat__interrupted">（已停止生成）</p>

              <p v-if="m.refs && m.refs.length" class="ai-chat__refs">
                参考 {{ m.refs.length }} 段
              </p>
            </div>
          </div>
        </div>

        <footer class="ai-chat__composer">
          <textarea
            v-model="draft"
            class="ai-chat__input"
            :disabled="!context?.enabled"
            placeholder="就这篇文章提问…（Enter 发送，Shift+Enter 换行）"
            rows="1"
            @keydown.enter.exact.prevent="onEnter"
          ></textarea>
          <button
            v-if="!streaming"
            class="ai-chat__send"
            type="button"
            :disabled="!draft.trim()"
            @click="send"
          >
            发送
          </button>
          <button v-else class="ai-chat__send ai-chat__send--stop" type="button" @click="stop">
            停止
          </button>
        </footer>
      </template>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { MdPreview } from 'md-editor-v3'
import 'md-editor-v3/lib/preview.css'
import {
  createChatSession,
  fetchChatContext,
  fetchChatMessages,
  sendChatMessageStream,
  type ChatContext,
  type RefChunk
} from '../api/chat'

const props = defineProps<{ articleId: number }>()

const route = useRoute()
const router = useRouter()

interface LocalMessage {
  key: string
  id: number
  role: 'user' | 'assistant'
  content: string
  model?: string
  refused?: boolean
  interrupted?: boolean
  error?: string
  streaming?: boolean
  toolHint?: string
  refs?: RefChunk[]
}

const open = ref(false)
const context = ref<ChatContext | null>(null)
const messages = ref<LocalMessage[]>([])
const draft = ref('')
const streaming = ref(false)
const loadingHistory = ref(false)
const listEl = ref<HTMLElement | null>(null)

const sessionId = ref(0)
let abortController: AbortController | null = null
// 流式节流：把高频 delta 合并成每 60ms 一次渲染。
// 不节流的话每个 token 都触发一次 Markdown 重渲染，长回答会明显卡顿。
let flushTimer: number | null = null
let pendingText = ''

const loggedIn = computed(() => !!localStorage.getItem('token'))
const currentModel = computed(() => {
  const m = messages.value.find((x) => x.role === 'assistant' && x.model)
  return m?.model || ''
})
const sampleQuestions = computed(() => [
  '这篇文章主要讲了什么？',
  '文中的关键结论是怎么得出来的？',
  '有没有容易踩坑的地方？'
])

const lastSessionKey = () => {
  // key 带 userId：同一台设备换账号时不会拿上一个账号的会话 id 去请求
  let uid = 'u'
  try {
    const raw = localStorage.getItem('user')
    if (raw) uid = String(JSON.parse(raw).id ?? 'u')
  } catch {
    /* user 不是合法 JSON 时用兜底 key */
  }
  return `chat_last_session:${uid}:${props.articleId}`
}

function scrollToBottom() {
  nextTick(() => {
    const el = listEl.value
    if (el) el.scrollTop = el.scrollHeight
  })
}

async function openPanel() {
  open.value = true
  if (!loggedIn.value) return
  if (!context.value) await loadContext()
}

function closePanel() {
  open.value = false
  // 面板收起不销毁会话：下次展开还是原来的对话
}

function goLogin() {
  router.push({ path: '/login', query: { redirect: route.fullPath } })
}

async function loadContext() {
  try {
    context.value = await fetchChatContext(props.articleId)
  } catch (e: any) {
    context.value = null
    // 401 说明 token 失效，提示重新登录即可；其它错误不打断页面使用
    if (e?.response?.status === 401) {
      localStorage.removeItem('token')
    }
    return
  }
  if (!context.value.enabled) return

  // 恢复上次会话：优先 localStorage（记录的是「这台设备上次真正打开的那个」），
  // 没有再退回服务端给的最近活跃会话
  const saved = Number(localStorage.getItem(lastSessionKey()) || 0)
  const target = saved || context.value.last_session_id
  if (target) {
    sessionId.value = target
    await loadHistory(target)
  }
}

async function loadHistory(id: number) {
  loadingHistory.value = true
  try {
    const res = await fetchChatMessages(id, 0, 30)
    messages.value = res.list.map((m) => ({
      key: `db-${m.id}`,
      id: m.id,
      role: m.role,
      content: m.content,
      model: m.model,
      refused: m.refused,
      interrupted: m.interrupted,
      error: m.error,
      refs: m.refs || undefined
    }))
    scrollToBottom()
  } catch {
    // 会话可能已在别处被删除：清掉本地记录并回到空态，不弹错误（不是用户操作失误）
    localStorage.removeItem(lastSessionKey())
    sessionId.value = 0
    messages.value = []
  } finally {
    loadingHistory.value = false
  }
}

function startNewSession() {
  if (streaming.value) return
  sessionId.value = 0
  messages.value = []
  localStorage.removeItem(lastSessionKey())
  draft.value = ''
}

function useSample(text: string) {
  draft.value = text
  nextTick(() => send())
}

function onEnter() {
  // 中文输入法：拼音还没上屏时按 Enter，compositionend 还没触发，不能当发送
  const composing = (window.event as any)?.isComposing
  if (composing) return
  send()
}

async function send() {
  const text = draft.value.trim()
  if (!text || streaming.value) return

  draft.value = ''
  streaming.value = true

  const userKey = `local-u-${Date.now()}`
  messages.value.push({ key: userKey, id: 0, role: 'user', content: text })

  const asstKey = `local-a-${Date.now()}`
  const assistant: LocalMessage = {
    key: asstKey,
    id: 0,
    role: 'assistant',
    content: '',
    streaming: true
  }
  messages.value.push(assistant)
  scrollToBottom()

  try {
    // 没有会话就先建一个
    if (!sessionId.value) {
      const session = await createChatSession(props.articleId, context.value?.default_model_id || 0)
      sessionId.value = session.id
      localStorage.setItem(lastSessionKey(), String(session.id))
    }

    abortController = new AbortController()
    await sendChatMessageStream(
      sessionId.value,
      { content: text },
      {
        onMeta: (meta) => {
          assistant.id = meta.assistant_message_id
          assistant.model = meta.model
        },
        onTool: (tool) => {
          // 检索进度：3~5 秒空白没有任何反馈时，用户一定以为卡死了
          if (tool.phase === 'start') {
            assistant.toolHint = `正在检索资料…（第 ${tool.round} 轮）`
          } else {
            assistant.toolHint = `已检索 ${tool.hits ?? 0} 段`
          }
          scrollToBottom()
        },
        onRefs: (chunks) => {
          assistant.refs = chunks
        },
        onDelta: (delta) => {
          assistant.toolHint = ''
          pendingText += delta
          if (flushTimer === null) {
            flushTimer = window.setTimeout(() => {
              flushTimer = null
              assistant.content += pendingText
              pendingText = ''
              scrollToBottom()
            }, 60)
          }
        },
        onDone: (done) => {
          if (pendingText) {
            assistant.content += pendingText
            pendingText = ''
          }
          assistant.id = done.assistant_message_id
          assistant.refused = done.refused
          assistant.interrupted = done.interrupted
          assistant.streaming = false
          assistant.toolHint = ''
          scrollToBottom()
        },
        onError: (_code, message) => {
          // partial 时也走这里：已生成的内容必须保留，只在气泡里补一行错误说明
          if (pendingText) {
            assistant.content += pendingText
            pendingText = ''
          }
          assistant.streaming = false
          assistant.toolHint = ''
          assistant.error = message
        }
      },
      abortController.signal
    )
  } catch (e: any) {
    if (pendingText) {
      assistant.content += pendingText
      pendingText = ''
    }
    assistant.streaming = false
    assistant.toolHint = ''
    // 用户主动停止不算错误，保留已生成的内容
    if (e?.name === 'AbortError') {
      assistant.interrupted = true
    } else {
      assistant.error = e?.message || '网络异常，请稍后重试'
    }
  } finally {
    if (flushTimer !== null) {
      window.clearTimeout(flushTimer)
      flushTimer = null
    }
    streaming.value = false
    abortController = null
    scrollToBottom()
  }
}

function stop() {
  abortController?.abort()
}

// 切文章：面板内容必须重置，否则会把上一篇文章的对话带过来
watch(
  () => props.articleId,
  async () => {
    abortController?.abort()
    messages.value = []
    sessionId.value = 0
    context.value = null
    draft.value = ''
    if (open.value && loggedIn.value) await loadContext()
  }
)

onBeforeUnmount(() => {
  abortController?.abort()
  if (flushTimer !== null) window.clearTimeout(flushTimer)
})
</script>

<style scoped>
/* 入口按钮：与详情页浮层同一层级（NavBar 是 100/110，弹窗 200） */
.ai-chat__fab {
  position: fixed;
  right: 24px;
  bottom: 24px;
  z-index: 900;
  width: 52px;
  height: 52px;
  border: 1px solid var(--border);
  border-radius: 50%;
  background: var(--surface-strong);
  box-shadow: var(--shadow-soft);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: transform var(--transition-base), box-shadow var(--transition-base);
}

.ai-chat__fab:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.ai-chat__fab-icon {
  width: 28px;
  height: 28px;
}

.ai-chat__panel {
  position: fixed;
  right: 24px;
  bottom: 24px;
  z-index: 910;
  width: 388px;
  height: min(640px, calc(100vh - 120px));
  display: flex;
  flex-direction: column;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--surface-strong);
  box-shadow: var(--shadow-md);
  overflow: hidden;
}

.ai-chat__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 14px;
  border-bottom: 1px solid var(--border);
  background: var(--surface-soft);
}

.ai-chat__title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  color: var(--text-strong);
  font-size: 0.9rem;
}

.ai-chat__logo {
  width: 20px;
  height: 20px;
}

.ai-chat__model {
  font-weight: 400;
  font-size: 0.72rem;
  color: var(--text-faint);
  max-width: 110px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ai-chat__actions {
  display: flex;
  gap: 4px;
}

.ai-chat__icon-btn {
  width: 26px;
  height: 26px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface-strong);
  color: var(--text-secondary);
  cursor: pointer;
  line-height: 1;
}

.ai-chat__icon-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.ai-chat__messages {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  background: var(--surface-soft);
}

.ai-chat__hint,
.ai-chat__empty-desc {
  color: var(--text-muted);
  font-size: 0.8rem;
  text-align: center;
}

.ai-chat__empty {
  margin: auto 0;
  text-align: center;
  padding: 12px;
}

.ai-chat__empty-title {
  font-weight: 600;
  color: var(--text-strong);
  margin-bottom: 6px;
}

.ai-chat__samples {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: 12px;
}

.ai-chat__sample {
  padding: 7px 10px;
  border: 1px dashed var(--border-strong);
  border-radius: 10px;
  background: var(--surface-strong);
  color: var(--text-secondary);
  font-size: 0.8rem;
  cursor: pointer;
  text-align: left;
}

.ai-chat__sample:hover {
  border-color: var(--primary-400);
  color: var(--primary-700);
}

.ai-chat__msg {
  display: flex;
}

.ai-chat__msg--user {
  justify-content: flex-end;
}

.ai-chat__bubble {
  max-width: 88%;
  padding: 9px 12px;
  border-radius: 12px;
  font-size: 0.85rem;
  line-height: 1.6;
  color: var(--text-strong);
  background: var(--surface-strong);
  border: 1px solid var(--border);
  overflow-wrap: anywhere;
}

.ai-chat__msg--user .ai-chat__bubble {
  background: var(--primary-soft);
  border-color: rgba(16, 185, 129, 0.3);
}

/* 拒答不是错误，是设计好的边界：用低调灰条，不要红框 */
.ai-chat__bubble--refused {
  background: var(--surface-soft);
  border-style: dashed;
  color: var(--text-muted);
}

.ai-chat__bubble--error {
  border-color: rgba(220, 38, 38, 0.3);
}

.ai-chat__plain {
  white-space: pre-wrap;
  margin: 0;
}

.ai-chat__markdown :deep(.md-editor-preview) {
  background: transparent;
  font-size: 0.85rem;
}

.ai-chat__caret {
  display: inline-block;
  width: 6px;
  height: 14px;
  margin-left: 2px;
  vertical-align: text-bottom;
  background: var(--primary-500);
  animation: ai-chat-blink 1s steps(2, start) infinite;
}

@keyframes ai-chat-blink {
  to {
    visibility: hidden;
  }
}

.ai-chat__tool-hint {
  margin: 6px 0 0;
  font-size: 0.75rem;
  color: var(--text-faint);
}

.ai-chat__error-text {
  margin: 6px 0 0;
  font-size: 0.75rem;
  color: var(--danger);
}

.ai-chat__interrupted {
  margin: 6px 0 0;
  font-size: 0.75rem;
  color: var(--text-faint);
}

.ai-chat__refs {
  margin: 6px 0 0;
  font-size: 0.72rem;
  color: var(--text-faint);
}

.ai-chat__composer {
  display: flex;
  gap: 8px;
  padding: 10px 12px 12px;
  border-top: 1px solid var(--border);
  background: var(--surface-strong);
}

.ai-chat__input {
  flex: 1;
  min-height: 40px;
  max-height: 120px;
  padding: 9px 11px;
  border: 1px solid var(--border-strong);
  border-radius: 10px;
  font-size: 0.85rem;
  font-family: inherit;
  color: var(--text-strong);
  resize: none;
  outline: none;
}

.ai-chat__input:focus {
  border-color: var(--primary-500);
  box-shadow: 0 0 0 3px var(--primary-soft);
}

.ai-chat__send {
  align-self: flex-end;
  height: 40px;
  padding: 0 16px;
  border: none;
  border-radius: 10px;
  background: var(--primary-500);
  color: #fff;
  font-size: 0.85rem;
  cursor: pointer;
}

.ai-chat__send:disabled {
  background: var(--border-strong);
  cursor: not-allowed;
}

.ai-chat__send--stop {
  background: var(--text-muted);
}

.ai-chat__notice {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 24px;
  text-align: center;
}

.ai-chat__notice-title {
  font-weight: 600;
  color: var(--text-strong);
}

.ai-chat__notice-desc {
  font-size: 0.8rem;
  color: var(--text-muted);
}

.ai-chat__notice-btn {
  margin-top: 8px;
}

@media (max-width: 768px) {
  .ai-chat__panel {
    right: 0;
    bottom: 0;
    width: 100%;
    height: 100%;
    border-radius: 0;
  }
}
</style>
