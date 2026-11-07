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
          <!-- 多个模型时才给下拉：只有一个模型时下拉没有可选项，
               反而让人以为还能选别的。 -->
          <select
            v-if="models.length > 1"
            class="ai-chat__model-select"
            :value="selectedModelID"
            :disabled="streaming || !context?.enabled"
            title="切换模型只影响后续消息，已有回答不受影响"
            @change="onModelChange"
          >
            <option v-for="m in models" :key="m.id" :value="m.id">{{ m.model }}</option>
          </select>
          <span v-else-if="currentModel" class="ai-chat__model">{{ currentModel }}</span>
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
            <!-- 过程面板：像编程工具那样，把「查了什么、查到没有、花了多久」摊开。
                 只在 assistant 侧显示；user 侧没有过程可言 -->
            <div v-if="m.role === 'assistant'" class="ai-chat__steps">
              <button class="ai-chat__steps-head" type="button" @click="toggleSteps(m)">
                <span
                  class="ai-chat__steps-dot"
                  :class="{
                    'ai-chat__steps-dot--live': hasRunning(m),
                    'ai-chat__steps-dot--empty': !hasRunning(m) && !m.steps?.length
                  }"
                ></span>
                <span class="ai-chat__steps-title">{{ stepsTitle(m) }}</span>
                <span class="ai-chat__steps-chevron" :class="{ 'is-open': m.stepsOpen }">▸</span>
              </button>

              <ol v-if="m.stepsOpen && m.steps?.length" class="ai-chat__step-list">
                <li v-for="st in m.steps" :key="st.round" class="ai-chat__step">
                  <span class="ai-chat__step-round">{{ st.round }}</span>
                  <div class="ai-chat__step-body">
                    <p class="ai-chat__step-query">
                      检索「<span class="ai-chat__step-q">{{ st.query || '（无检索词）' }}</span>」
                    </p>
                    <p class="ai-chat__step-meta">
                      <span v-if="st.running" class="ai-chat__step-live">检索中…</span>
                      <template v-else>
                        <span v-if="st.hits > 0">命中 {{ st.hits }} 段</span>
                        <span v-else class="ai-chat__step-none">未命中</span>
                        <span> · {{ st.chars }} 字 · {{ st.ms }}ms</span>
                      </template>
                      <span v-if="st.degraded" class="ai-chat__step-degraded">
                        · {{ st.note || '降级' }}
                      </span>
                    </p>
                    <p v-if="st.articles?.length" class="ai-chat__step-articles">
                      来自：{{ st.articles.join('、') }}
                    </p>
                  </div>
                </li>
              </ol>
            </div>

            <div
              class="ai-chat__bubble"
              :class="{
                'ai-chat__bubble--refused': m.refused || m.noRetrieval,
                'ai-chat__bubble--error': m.error
              }"
            >
              <!-- 助手消息渲染 Markdown；用户消息保留换行 -->
              <MdPreview
                v-if="m.role === 'assistant' && !m.refused && !m.noRetrieval"
                :model-value="m.content || (m.streaming ? '' : '')"
                class="ai-chat__markdown"
              />
              <p v-else class="ai-chat__plain">
                <span
                  v-for="(t, i) in tokensOf(m)"
                  :key="i"
                  :class="{ 'ai-chat__pill': t.type === 'pill' }"
                  >{{ t.text }}</span
                >
              </p>

              <span v-if="m.streaming" class="ai-chat__caret" aria-hidden="true"></span>

              <p v-if="m.error" class="ai-chat__error-text">{{ m.error }}</p>
              <p v-if="m.interrupted" class="ai-chat__interrupted">（已停止生成）</p>

              <p v-if="m.refs && m.refs.length" class="ai-chat__refs">
                参考 {{ m.refs.length }} 段
              </p>
            </div>
          </div>
        </div>

        <ChatComposer
          ref="composerRef"
          :disabled="!context?.enabled"
          :streaming="streaming"
          :max-refs="maxRefs"
          :current-article-id="articleId"
          @send="onComposerSend"
          @stop="stop"
        />
      </template>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { MdPreview } from 'md-editor-v3'
import 'md-editor-v3/lib/preview.css'
import ChatComposer from './ChatComposer.vue'
import {
  createChatSession,
  fetchChatContext,
  fetchChatMessages,
  sendChatMessageStream,
  type ArticleRef,
  type ChatContext,
  updateChatSession,
  type ChatStep,
  type RefChunk
} from '../api/chat'
import { tokenizeMessage, type ComposerPayload, type MessageToken } from '../utils/editor-serialize'
import { showToast } from '../utils/toast'

const props = defineProps<{ articleId: number }>()

const route = useRoute()
const router = useRouter()

/** 前端本地步骤：在后端结构上补「进行中」状态与开始时间，用于实时显示「检索中…」和耗时 */
interface LocalStep extends ChatStep {
  running?: boolean
  startedAt?: number
}

interface LocalMessage {
  key: string
  id: number
  role: 'user' | 'assistant'
  content: string
  model?: string
  refused?: boolean
  /** 一次检索都没成功，content 是后端固定文案 */
  noRetrieval?: boolean
  interrupted?: boolean
  error?: string
  streaming?: boolean
  refs?: RefChunk[]
  articleRefs?: ArticleRef[]
  /** 检索过程（仅 assistant） */
  steps?: LocalStep[]
  /** 过程面板是否展开 */
  stepsOpen?: boolean
  /** 实际检索轮数（来自 done 事件），用于区分「没检索」和「检索了没命中」 */
  rounds?: number
}

const open = ref(false)
const context = ref<ChatContext | null>(null)
const messages = ref<LocalMessage[]>([])
const streaming = ref(false)
const loadingHistory = ref(false)
const listEl = ref<HTMLElement | null>(null)
const composerRef = ref<InstanceType<typeof ChatComposer> | null>(null)

const sessionId = ref(0)
// 当前会话（或即将创建的那个）使用的模型配置 id。
//
// 为什么要有这个状态而不是每次都读 context.default_model_id：
//   ① 会话已存在时，它才是真正生效的那个（会话级绑定）；
//   ② 还没建会话（刚打开面板）时，用户选的模型得先记在这里，
//      等第一次发消息建会话时带进去——否则「选了模型但第一次提问还是默认模型」。
const selectedModelID = ref(0)
let abortController: AbortController | null = null
// delta 节流：把高频增量合并成每 60ms 一次渲染。
// 不节流的话每个 token 都触发一次 Markdown 重渲染，长回答明显卡顿。
let flushTimer: number | null = null
let pendingText = ''

const loggedIn = computed(() => !!localStorage.getItem('token'))
const models = computed(() => context.value?.models || [])

// 优先按「当前选中的 id」映射名字：刚切完模型还没发消息时，
// 历史消息里的 model 是旧模型名，用它显示会让人以为切换没生效。
const currentModel = computed(() => {
  const hit = models.value.find((m) => m.id === selectedModelID.value)
  if (hit) return hit.model
  return messages.value.find((x) => x.role === 'assistant' && x.model)?.model || ''
})
// 后端说「最多 8 篇（含当前文章）」，前端算 pill 上限时减 1。
// 不写死 7：将来后端调这个数，前端自动跟上。
const maxRefs = computed(() => Math.max((context.value?.max_ref_articles || 8) - 1, 0))
const sampleQuestions = computed(() => [
  '这篇文章主要讲了什么？',
  '文中的关键结论是怎么得出来的？',
  '有没有容易踩坑的地方？'
])

/**
 * 按 key 修改消息。
 *
 * ⚠️ 必须通过 messages.value[idx] 拿到**响应式代理**再改。
 * 直接改 push 进去时的原始对象引用不会触发渲染——表现是「流式时文字不刷新，
 * 直到流结束才一次性出现」，而且不报错，极难定位。
 */
function patchMsg(key: string, fn: (m: LocalMessage) => void) {
  const idx = messages.value.findIndex((x) => x.key === key)
  if (idx < 0) return
  fn(messages.value[idx])
}

// 历史消息里的 `/标题` 还原成 pill（按 article_refs 匹配，匹配不上退化成纯文本）
function tokensOf(m: LocalMessage): MessageToken[] {
  if (m.role !== 'user') return [{ type: 'text', text: m.content }]
  return tokenizeMessage(m.content, m.articleRefs || null)
}

// ===== 过程面板 =====

const hasRunning = (m: LocalMessage) => !!m.steps?.some((s) => s.running)

function stepsTitle(m: LocalMessage): string {
  const steps = m.steps || []
  if (steps.some((s) => s.running)) {
    return `正在检索资料…（第 ${steps.length} 轮）`
  }
  if (steps.length > 0) {
    const total = steps.reduce((n, s) => n + (s.ms || 0), 0)
    const hits = steps.reduce((n, s) => n + (s.hits || 0), 0)
    return `检索 ${steps.length} 次 · 命中 ${hits} 段 · ${total}ms`
  }
  if (m.streaming) return '正在思考…'
  if (m.refused) return '未检索（判定与文章无关）'
  // 零检索 = 模型手里只有标题没有正文，回答已被后端替换成固定提示。
  // 不能再写「直接作答」——那正是它编内容时给人的错觉。
  if (m.noRetrieval) return '未能检索到资料（未作答）'
  return '未检索资料'
}

function toggleSteps(m: LocalMessage) {
  m.stepsOpen = !m.stepsOpen
}

// ===== 会话 =====

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
  nextTick(() => composerRef.value?.focus())
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
    if (e?.response?.status === 401) {
      localStorage.removeItem('token')
    }
    return
  }
  if (!context.value.enabled) return

  // 优先 localStorage（记录「这台设备上次真正打开的那个」），服务端只作兜底
  const saved = Number(localStorage.getItem(lastSessionKey()) || 0)
  const target = saved || context.value.last_session_id
  if (target) {
    sessionId.value = target
    // 模型以**该会话实际绑定的**为准，不能用 default_model_id：
    // 后者是「最近活跃会话的模型」，而用户可能恰好停在另一个更早的会话上。
    // 列表被 50 条截断时查不到，才退回默认。
    const hit = context.value.sessions.find((x) => x.id === target)
    selectedModelID.value = hit?.model_config_id || context.value.default_model_id || 0
    await loadHistory(target)
  } else {
    // 还没建会话：先把默认模型填上，用户随时可以改，改的值会在建会话时带进去
    selectedModelID.value = context.value.default_model_id || 0
  }
}

/**
 * 切换模型。
 *
 * 两种情形必须分开处理：
 *   - 会话已存在 → PATCH 落库，后续消息立即用新模型
 *   - 会话还没建 → 只改本地，等第一次发消息创建会话时带上
 *
 * 失败要回滚并提示。静默失败最糟：用户以为换了模型，实际还在用旧的，
 * 而且是「同一句话问两遍答案质量不同」这种没法复现的现象。
 */
async function onModelChange(e: Event) {
  const el = e.target as HTMLSelectElement
  const next = Number(el.value)
  const prev = selectedModelID.value
  if (!next || next === prev) return
  selectedModelID.value = next

  if (!sessionId.value) return // 还没建会话，建的时候带进去
  try {
    await updateChatSession(sessionId.value, { model_config_id: next })
  } catch {
    selectedModelID.value = prev
    // 受控回滚：select 是 :value 绑定的，改回 ref 后需要手工同步 DOM，
    // 否则界面还停在新选项上，和实际生效的模型不一致。
    el.value = String(prev)
    showToast('切换模型失败，请稍后重试', 'error')
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
      noRetrieval: m.no_retrieval,
      interrupted: m.interrupted,
      error: m.error,
      refs: m.refs || undefined,
      articleRefs: m.article_refs || undefined,
      steps: (m.steps || []) as LocalStep[],
      // 历史消息默认收起：翻旧对话时一屏塞满过程细节会盖住答案本身
      stepsOpen: false,
      rounds: m.steps?.length || 0
    }))
    scrollToBottom()
  } catch {
    // 会话可能已在别处被删除：清掉本地记录回到空态，不弹错误（不是用户操作失误）
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
  composerRef.value?.clear()
  // 模型保持用户当前的选择，不重置回默认：
  // 他刚刚可能就是为了换模型才新建会话的。
}

function useSample(text: string) {
  void onComposerSend({ text, articleIds: [], refs: [] })
}

// ===== 发消息 =====

async function onComposerSend(payload: ComposerPayload) {
  const text = payload.text.trim()
  if (!text || streaming.value) return

  streaming.value = true

  const userKey = `local-u-${Date.now()}`
  messages.value.push({
    key: userKey,
    id: 0,
    role: 'user',
    content: text,
    // 回显时用序列化时收集的 refs（含完整标题），刚发出去的消息也能显示 pill
    articleRefs: payload.refs
  })

  const asstKey = `local-a-${Date.now()}`
  messages.value.push({
    key: asstKey,
    id: 0,
    role: 'assistant',
    content: '',
    streaming: true,
    steps: [],
    // 新消息默认展开：用户提问后第一件事就是想知道「它在干什么」
    stepsOpen: true
  })
  scrollToBottom()

  try {
    if (!sessionId.value) {
      const session = await createChatSession(
        props.articleId,
        selectedModelID.value || context.value?.default_model_id || 0
      )
      sessionId.value = session.id
      localStorage.setItem(lastSessionKey(), String(session.id))
    }

    abortController = new AbortController()
    await sendChatMessageStream(
      sessionId.value,
      { content: text, articleIds: payload.articleIds },
      {
        onMeta: (meta) => {
          patchMsg(asstKey, (m) => {
            m.id = meta.assistant_message_id
            m.model = meta.model
          })
          // 服务端会把校验通过的引用回传：以它为准（被下架的文章会被剔除）
          if (meta.article_refs?.length) {
            patchMsg(userKey, (m) => {
              m.articleRefs = meta.article_refs
            })
          }
        },
        onTool: (tool) => {
          patchMsg(asstKey, (m) => {
            if (!m.steps) m.steps = []
            if (tool.phase === 'start') {
              m.steps.push({
                round: tool.round,
                tool: tool.name || '',
                query: tool.query || '',
                hits: 0,
                chars: 0,
                ms: 0,
                running: true,
                startedAt: Date.now()
              })
            } else {
              // 用 running 标记找，不靠 round 比对：一轮里可能有多个并行调用
              const target = [...m.steps].reverse().find((x) => x.running)
              if (target) {
                target.hits = tool.hits ?? 0
                target.chars = tool.chars ?? 0
                target.articles = tool.articles || []
                target.ms = target.startedAt ? Date.now() - target.startedAt : 0
                target.running = false
              }
            }
          })
          scrollToBottom()
        },
        onRefs: (chunks) => {
          patchMsg(asstKey, (m) => {
            m.refs = chunks
          })
        },
        onDelta: (delta) => {
          pendingText += delta
          if (flushTimer === null) {
            flushTimer = window.setTimeout(() => {
              flushTimer = null
              const chunk = pendingText
              pendingText = ''
              patchMsg(asstKey, (m) => {
                m.content += chunk
              })
              scrollToBottom()
            }, 60)
          }
        },
        onDone: (done) => {
          const rest = pendingText
          pendingText = ''
          patchMsg(asstKey, (m) => {
            if (rest) m.content += rest
            m.id = done.assistant_message_id
            m.refused = done.refused
            m.noRetrieval = done.no_retrieval
            m.interrupted = done.interrupted
            m.streaming = false
            m.rounds = done.rounds
            // 以服务端记录为准覆盖本地拼的步骤：
            // 流式期间是靠 tool 事件一帧帧拼的，任何一帧丢了过程就残缺，
            // 而 done 里的 steps 是 tool 自己写的权威数据（含耗时、命中数、降级原因）。
            if (done.steps) {
              m.steps = done.steps.map((st) => ({ ...st, running: false }))
            }
          })
          scrollToBottom()
        },
        onError: (_code, message) => {
          // partial 时也走这里：已生成的内容必须保留，只补一行错误说明
          const rest = pendingText
          pendingText = ''
          patchMsg(asstKey, (m) => {
            if (rest) m.content += rest
            m.streaming = false
            m.error = message
          })
        }
      },
      abortController.signal
    )
  } catch (e: any) {
    const rest = pendingText
    pendingText = ''
    patchMsg(asstKey, (m) => {
      if (rest) m.content += rest
      m.streaming = false
      // 用户主动停止不算错误，保留已生成的内容
      if (e?.name === 'AbortError') {
        m.interrupted = true
      } else {
        m.error = e?.message || '网络异常，请稍后重试'
      }
    })
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
    selectedModelID.value = 0
    context.value = null
    composerRef.value?.clear()
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

/* 模型选择：用原生 select 而不是自绘浮层。
   380px 宽的面板里，自绘下拉要处理定位、层级、点击外部关闭，
   而原生 select 在移动端直接唤起系统选择器，键盘也能操作，成本最低。 */
.ai-chat__model-select {
  max-width: 140px;
  padding: 1px 2px 1px 6px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--surface-strong);
  color: var(--text-muted);
  font-size: 0.72rem;
  font-family: inherit;
  cursor: pointer;
  /* 保留原生下拉箭头：这是「可以点」的唯一视觉线索。
     用 appearance:none 抹掉它等于让这个控件看起来像一段静态文字。 */
}

.ai-chat__model-select:hover:not(:disabled) {
  border-color: var(--border-strong);
  color: var(--text-secondary);
}

.ai-chat__model-select:disabled {
  opacity: 0.55;
  cursor: not-allowed;
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
  flex-direction: column;
  gap: 6px;
}

.ai-chat__msg--user {
  align-items: flex-end;
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

/* ===== 过程面板 ===== */

.ai-chat__steps {
  max-width: 88%;
  width: 100%;
  font-size: 0.74rem;
}

.ai-chat__steps-head {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  padding: 5px 8px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface-strong);
  color: var(--text-muted);
  font-size: 0.74rem;
  text-align: left;
  cursor: pointer;
}

.ai-chat__steps-head:hover {
  border-color: var(--border-strong);
  color: var(--text-secondary);
}

/* 状态点：进行中时闪烁，让用户知道它还在跑 */
.ai-chat__steps-dot {
  flex-shrink: 0;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--primary-500);
}

.ai-chat__steps-dot--live {
  animation: ai-chat-pulse 1s ease-in-out infinite;
}

.ai-chat__steps-dot--empty {
  background: var(--text-faint);
}

@keyframes ai-chat-pulse {
  50% {
    opacity: 0.25;
  }
}

.ai-chat__steps-title {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ai-chat__steps-chevron {
  flex-shrink: 0;
  transition: transform 0.18s ease;
}

.ai-chat__steps-chevron.is-open {
  transform: rotate(90deg);
}

.ai-chat__step-list {
  margin: 6px 0 0;
  padding: 8px 10px;
  list-style: none;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface-soft);
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.ai-chat__step {
  display: flex;
  gap: 8px;
}

.ai-chat__step-round {
  flex-shrink: 0;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: var(--accent-soft);
  color: var(--accent-500);
  font-size: 0.66rem;
  font-weight: 600;
  line-height: 16px;
  text-align: center;
}

.ai-chat__step-body {
  flex: 1;
  min-width: 0;
}

.ai-chat__step-query {
  margin: 0;
  color: var(--text-secondary);
  overflow-wrap: anywhere;
}

.ai-chat__step-q {
  color: var(--text-strong);
  font-weight: 600;
}

.ai-chat__step-meta {
  margin: 2px 0 0;
  color: var(--text-faint);
}

.ai-chat__step-live {
  color: var(--primary-600);
}

.ai-chat__step-none {
  color: var(--warning);
}

.ai-chat__step-degraded {
  color: var(--warning);
}

.ai-chat__step-articles {
  margin: 2px 0 0;
  color: var(--text-faint);
  overflow-wrap: anywhere;
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

/* 历史消息里的引用 pill（只读展示，样式与输入框里的保持一致） */
.ai-chat__pill {
  display: inline-block;
  margin: 0 2px;
  padding: 1px 6px;
  border-radius: 6px;
  background: var(--accent-soft);
  color: var(--accent-500);
  font-weight: 600;
  font-size: 0.95em;
  white-space: nowrap;
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
