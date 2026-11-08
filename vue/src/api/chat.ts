import api from './index'
import { createSSEParser } from '../utils/sse'
import { handleAuthFailure, refreshAccessToken } from '../utils/token'

/**
 * 文章页 AI 助手的接口封装。
 *
 * 会话 CRUD 走项目的 axios 单例（有 401 自动刷新）；**发消息走原生 fetch**：
 *   - axios 单例 timeout 是 60s，而 agent 最多检索 5 次，60s 会误杀正常请求
 *   - 拦截器拿不到 ReadableStream，做不了逐字上屏
 *   - 它的响应拦截器还会对 401 自动刷新，流式场景下语义已经不匹配
 */

export interface ChatModel {
  id: number
  model: string
}

/** 一次工具调用的完整记录（含检索到的原文） */
export interface ChatToolMessage {
  id: number
  round: number
  tool_name: string
  query: string
  /** 工具返回给模型的资料原文（含 `[1] 《标题》· 小节` 段头） */
  content: string
  hits: number
  chars: number
  ms: number
  articles: string[]
  degraded: boolean
  note: string
  created_at: string
}

export interface ChatSession {
  id: number
  title: string
  model_config_id: number
  model: string
  message_count: number
  last_message_at: string | null
  created_at: string
}

export interface ChatContext {
  enabled: boolean
  reason: string
  article_id: number
  article_title: string
  rag_ready: boolean
  max_ref_articles: number
  models: ChatModel[]
  default_model_id: number
  last_session_id: number
  sessions: ChatSession[]
}

export interface RefChunk {
  chunk_id: number
  article_id: number
  section_index: number
  heading_path: string
  rrf_score: number
}

export interface ArticleRef {
  article_id: number
  title: string
}

/** 一次工具调用的过程记录（后端 chat_messages.steps） */
export interface ChatStep {
  round: number
  tool: string
  query: string
  hits: number
  chars: number
  /** 耗时毫秒 */
  ms: number
  articles?: string[]
  degraded?: boolean
  note?: string
}

export interface ChatMessage {
  id: number
  role: 'user' | 'assistant'
  content: string
  model: string
  refs: RefChunk[] | null
  article_refs: ArticleRef[] | null
  /** 仅 assistant：检索过程。nil/空表示这次没检索 */
  steps: ChatStep[] | null
  interrupted: boolean
  refused: boolean
  /** 一次检索都没成功，content 是后端固定文案（不是模型说的） */
  no_retrieval: boolean
  error: string
  created_at: string
}

export interface ArticleCandidate {
  id: number
  title: string
}

/** 取上下文（能不能聊、可用模型、该文章下的会话列表） */
export async function fetchChatContext(articleId: number): Promise<ChatContext> {
  const res = await api.get(`/articles/${articleId}/chat/context`)
  return res.data.data
}

/** 新建会话。modelConfigId 传 0 表示用服务端默认模型 */
export async function createChatSession(
  articleId: number,
  modelConfigId = 0
): Promise<ChatSession> {
  const res = await api.post('/chat/sessions', {
    article_id: articleId,
    model_config_id: modelConfigId
  })
  return res.data.data
}

/** 历史消息（before_id 游标分页，返回正序） */
export async function fetchChatMessages(
  sessionId: number,
  beforeId = 0,
  limit = 30
): Promise<{ list: ChatMessage[]; has_more: boolean }> {
  const res = await api.get(`/chat/sessions/${sessionId}/messages`, {
    params: { before_id: beforeId, limit }
  })
  return res.data.data
}

/**
 * 改会话（标题 / 模型）。
 *
 * 换模型只影响**后续消息**：历史消息保留各自生成时的 model 名，不会重算。
 * 这是「会话级绑定」而不是「全局绑定」的意义——A 会话用便宜模型、
 * B 会话用强模型，互相不受影响。
 */
export async function updateChatSession(
  sessionId: number,
  patch: { title?: string; model_config_id?: number }
): Promise<void> {
  await api.patch(`/chat/sessions/${sessionId}`, patch)
}

/**
 * 取某条助手消息的工具调用记录（含原文）。
 *
 * 按需拉而不是跟着历史列表走：单条原文 6KB 起，一次拉 30 条历史
 * 会把几百 KB 全塞进响应，而用户通常只看其中一轮。
 */
export async function fetchToolMessages(
  sessionId: number,
  messageId: number
): Promise<ChatToolMessage[]> {
  const res = await api.get(`/chat/sessions/${sessionId}/messages/${messageId}/tools`)
  return res.data.data.list || []
}

/** 删除会话（硬删，不可恢复） */
export async function deleteChatSession(sessionId: number): Promise<void> {
  await api.delete(`/chat/sessions/${sessionId}`)
}

/** `/` 选择器的候选文章（P7 用，先留好接口） */
export async function fetchArticleCandidates(
  keyword: string,
  excludeId: number,
  limit = 20
): Promise<ArticleCandidate[]> {
  const res = await api.get('/chat/article-candidates', {
    params: { keyword, exclude_id: excludeId, limit }
  })
  return res.data.data || []
}

// ===== 流式发消息 =====

export interface StreamMeta {
  session_id: number
  user_message_id: number
  assistant_message_id: number
  model: string
  model_fallback: boolean
  article_refs: ArticleRef[]
}

export interface StreamTool {
  phase: 'start' | 'end'
  round: number
  name?: string
  query?: string
  hits?: number
  chars?: number
  articles?: string[]
}

export interface StreamDone {
  assistant_message_id: number
  prompt_tokens: number
  completion_tokens: number
  refused: boolean
  no_retrieval: boolean
  interrupted: boolean
  rounds: number
  tool_calls: number
  searched_articles: number[]
  degraded: boolean
  /**
   * 服务端记录的完整检索过程。
   *
   * 流式期间前端是靠 tool 事件现拼的；一旦某帧丢了，过程就永久缺一块。
   * done 里带一份权威数据，前端以它为准覆盖本地 —— 这是「过程一定会显示出来」的保证。
   */
  steps: ChatStep[]
}

export interface StreamHandlers {
  onMeta?: (meta: StreamMeta) => void
  onTool?: (tool: StreamTool) => void
  onRefs?: (chunks: RefChunk[]) => void
  onDelta?: (text: string) => void
  onDone?: (done: StreamDone) => void
  /** partial=true 表示之前已经推过内容，调用方必须保留已显示的文字 */
  onError?: (code: number, message: string, partial: boolean) => void
}

export interface SendOptions {
  content: string
  articleIds?: number[]
}

/**
 * 流式发送消息。
 *
 * 正常返回表示流已结束（含服务端发来的 error 事件）；
 * 抛异常只发生在「HTTP 层就没成功」或「网络层中断」。
 * 用户主动 abort 时抛 AbortError，调用方需要识别并静默处理。
 */
/**
 * 发一个流式 POST，遇到 401 自动刷新 token 并重试一次。
 *
 * 为什么重试是安全的：401 由 Auth 中间件在 handler **之前**返回，
 * 后端一个字都没执行（没落库、没调模型、没花 token）。
 * 而流一旦开出来（200），token 的事就已经过去了，不会再出现 401——
 * 所以不存在「重试导致重复提问」。
 *
 * ⚠️ 只能重试一次。刷新成功但仍 401 说明不是过期问题，
 * 再刷只会打成死循环（后端 refresh 是 rotation，越刷越乱）。
 */
async function postStream(
  url: string,
  payload: unknown,
  signal?: AbortSignal
): Promise<Response> {
  const send = (token: string | null) =>
    fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { Authorization: `Bearer ${token}` } : {})
      },
      body: JSON.stringify(payload),
      signal
    })

  let res = await send(localStorage.getItem('token'))
  if (res.status !== 401) return res

  // 用户已经点了停止 / 关了面板：没必要再刷新，直接按中断处理
  if (signal?.aborted) return res

  let token: string
  try {
    // 单飞刷新：和 axios 那侧共用同一个 promise，
    // 避免两个请求同时刷导致 refresh token 互相顶掉（后端会轮换）
    token = await refreshAccessToken()
  } catch {
    handleAuthFailure()
    throw new Error('登录信息已过期，请重新登录')
  }

  res = await send(token)
  if (res.status === 401) {
    handleAuthFailure()
    throw new Error('登录信息已过期，请重新登录')
  }
  return res
}

export async function sendChatMessageStream(
  sessionId: number,
  options: SendOptions,
  handlers: StreamHandlers,
  signal?: AbortSignal
): Promise<void> {
  const res = await postStream(
    `/api/v1/chat/sessions/${sessionId}/messages`,
    {
      content: options.content,
      article_ids: options.articleIds || [],
      stream: true
    },
    signal
  )

  // 非 200：服务端还没开始推流（meta 之前就失败了），返回的是普通 JSON 错误。
  // 后端刻意延迟写 SSE header，就是为了这种情况还能给出正确的状态码。
  if (!res.ok) {
    let message = `请求失败（${res.status}）`
    try {
      const body = await res.json()
      if (body?.message) message = body.message
    } catch {
      /* 响应体不是 JSON 时保留默认文案 */
    }
    throw new Error(message)
  }

  const reader = res.body?.getReader()
  if (!reader) throw new Error('当前浏览器不支持流式响应')

  const decoder = new TextDecoder()
  const parser = createSSEParser()

  const dispatch = (frame: { event: string; data: string }) => {
    let payload: any
    try {
      payload = JSON.parse(frame.data)
    } catch {
      return
    }
    switch (frame.event) {
      case 'meta':
        handlers.onMeta?.(payload)
        break
      case 'tool':
        handlers.onTool?.(payload)
        break
      case 'refs':
        handlers.onRefs?.(payload.chunks || [])
        break
      case 'delta':
        handlers.onDelta?.(payload.content || '')
        break
      case 'done':
        handlers.onDone?.(payload)
        break
      case 'error':
        handlers.onError?.(payload.code || 500, payload.message || '服务异常', !!payload.partial)
        break
    }
  }

  try {
    for (;;) {
      const { done, value } = await reader.read()
      if (done) break
      for (const frame of parser.feed(decoder.decode(value, { stream: true }))) {
        dispatch(frame)
      }
    }
    for (const frame of parser.flush()) {
      dispatch(frame)
    }
  } finally {
    reader.releaseLock()
  }
}

// ===== 后台「助手会话」 =====

export interface AdminSessionItem {
  id: number
  article_id: number
  article_title: string
  article_deleted: boolean
  user_id: number
  user_name: string
  title: string
  summary: string
  message_count: number
  last_message_at: string | null
  created_at: string
}

export interface AdminMessageItem {
  id: number
  role: 'user' | 'assistant'
  content: string
  model: string
  refs: RefChunk[] | null
  article_refs: ArticleRef[] | null
  steps: ChatStep[] | null
  tool_messages: ChatToolMessage[] | null
  prompt_tokens: number
  completion_tokens: number
  interrupted: boolean
  refused: boolean
  /** 一次检索都没成功，content 是后端固定文案（不是模型说的） */
  no_retrieval: boolean
  error: string
  created_at: string
}

export interface AdminSessionListParams {
  user?: string
  article?: string
  page?: number
  page_size?: number
}

/** 后台会话列表（需要管理员 token） */
export async function adminListChatSessions(params: AdminSessionListParams): Promise<{
  list: AdminSessionItem[]
  total: number
  page: number
  page_size: number
}> {
  const res = await api.get('/admin/chat-sessions', { params })
  return res.data.data
}

/** 后台会话详情 + 全部消息 */
export async function adminGetChatSession(
  id: number
): Promise<{ session: AdminSessionItem; messages: AdminMessageItem[] }> {
  const res = await api.get(`/admin/chat-sessions/${id}`)
  return res.data.data
}

/** 后台硬删会话（不可恢复） */
export async function adminDeleteChatSession(id: number): Promise<void> {
  await api.delete(`/admin/chat-sessions/${id}`)
}
