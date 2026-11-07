<template>
  <div class="acs">
    <div class="acs__header">
      <h2 class="acs__title">助手会话</h2>
      <div class="acs__actions">
        <!-- 刷新不回第 1 页：管理员停在第 3 页看数据，刷新后跳回第 1 页会很难受 -->
        <button class="btn-sm" type="button" @click="load(page)">刷新</button>
      </div>
    </div>

    <div class="acs__filter">
      <input
        v-model="filter.user"
        class="acs__input"
        placeholder="用户 ID 或用户名"
        @keyup.enter="load(1)"
      />
      <input
        v-model="filter.article"
        class="acs__input"
        placeholder="文章标题"
        @keyup.enter="load(1)"
      />
      <button class="btn-sm" type="button" @click="load(1)">查询</button>
      <button class="btn-sm" type="button" @click="resetFilter">重置</button>
    </div>

    <div class="acs__table-wrap">
      <table class="acs-table">
        <thead>
          <tr>
            <th>会话 ID</th>
            <th>文章</th>
            <th>用户</th>
            <th>消息数</th>
            <th>最后活跃</th>
            <th>会话摘要</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading">
            <td colspan="7" class="acs__empty">加载中…</td>
          </tr>
          <tr v-else-if="list.length === 0">
            <td colspan="7" class="acs__empty">没有符合条件的会话</td>
          </tr>
          <template v-else>
            <tr v-for="s in list" :key="s.id">
            <td class="acs__mono">{{ s.id }}</td>
            <td class="acs__article">
              <span v-if="s.article_deleted" class="acs__deleted">文章已删除</span>
              <span v-else :title="s.article_title">{{ s.article_title || '—' }}</span>
            </td>
            <td>
              <span class="acs__mono">#{{ s.user_id }}</span>
              <span class="acs__username">{{ s.user_name || '已注销' }}</span>
            </td>
            <td class="acs__mono">{{ s.message_count }}</td>
            <td class="acs__time">{{ s.last_message_at ? fmt(s.last_message_at) : '—' }}</td>
            <td class="acs__summary" :title="s.summary">{{ s.summary || '（还没有提问）' }}</td>
            <td class="acs__ops">
              <button class="btn-sm" type="button" @click="openDetail(s.id)">查看</button>
              <button class="btn-sm btn-sm--danger" type="button" @click="askDelete(s)">删除</button>
            </td>
            </tr>
          </template>
        </tbody>
      </table>
    </div>

    <div v-if="total > 0" class="pagination">
      <button class="pagination__btn" :disabled="page <= 1" @click="load(page - 1)">上一页</button>
      <template v-for="p in pageList" :key="p">
        <span v-if="p === '...'" class="pagination__ellipsis">...</span>
        <button
          v-else
          class="pagination__btn"
          :class="{ 'pagination__btn--active': p === page }"
          @click="load(Number(p))"
        >
          {{ p }}
        </button>
      </template>
      <button class="pagination__btn" :disabled="page >= totalPages" @click="load(page + 1)">
        下一页
      </button>
      <span class="pagination__info">共 {{ total }} 条</span>
      <select v-model.number="pageSize" class="pagination__size" @change="load(1)">
        <option :value="10">10 / 页</option>
        <option :value="20">20 / 页</option>
        <option :value="50">50 / 页</option>
      </select>
    </div>

    <!-- 详情弹窗：必须 Teleport 到 body。
         .admin-main 有 overflow-x: auto，会把子元素的 fixed 定位裁掉 -->
    <Teleport to="body">
      <div v-if="detail" class="modal-overlay" @click.self="closeDetail">
        <div class="acs-modal">
          <header class="acs-modal__header">
            <h3 class="acs-modal__title">会话 #{{ detail.session.id }}</h3>
            <button class="acs-modal__close" type="button" @click="closeDetail">×</button>
          </header>

          <div class="acs-modal__meta">
            <span>文章：{{ detail.session.article_title || '（已删除）' }}</span>
            <span>用户：#{{ detail.session.user_id }} {{ detail.session.user_name }}</span>
            <span>消息：{{ detail.session.message_count }}</span>
            <span>创建：{{ fmt(detail.session.created_at) }}</span>
          </div>

          <div class="acs-modal__body">
            <p v-if="detailLoading" class="acs__empty">加载中…</p>
            <template v-else>
            <div v-for="m in detail.messages" :key="m.id" class="acs-msg" :class="`acs-msg--${m.role}`">
              <div class="acs-msg__head">
                <span class="acs-msg__role">{{ m.role === 'user' ? '用户' : '助手' }}</span>
                <span class="acs-msg__time">{{ fmt(m.created_at) }}</span>
                <span v-if="m.model" class="acs-msg__model">{{ m.model }}</span>
                <!-- 这三个徽章是后台特有的：要能一眼看出哪句被拒、哪句失败、哪句被打断 -->
                <!-- 检索次数：后台判断「答得不好」的第一现场信息——是没查，还是查了没命中 -->
                <span
                  v-if="m.role === 'assistant'"
                  class="acs-badge"
                  :class="m.steps?.length ? 'acs-badge--info' : 'acs-badge--muted'"
                >
                  {{ m.steps?.length ? `检索 ${m.steps.length} 次` : '未检索' }}
                </span>
                <span v-if="m.refused" class="acs-badge acs-badge--refused">拒答</span>
                <span v-if="m.error" class="acs-badge acs-badge--error" :title="m.error">失败</span>
                <span v-if="m.interrupted" class="acs-badge acs-badge--warn">中断</span>
              </div>
              <!-- user 消息按原文显示（不渲染 pill）：后台要看的是用户到底打了什么 -->
              <p class="acs-msg__content">{{ m.content || '（无内容）' }}</p>
              <p v-if="m.error" class="acs-msg__error">{{ m.error }}</p>
              <p v-if="m.article_refs?.length" class="acs-msg__refs">
                引用文章：{{ m.article_refs.map((r) => `#${r.article_id} ${r.title}`).join('、') }}
              </p>
              <!-- 检索过程：后台默认展开，排查时不用再点一次 -->
              <ol v-if="m.steps?.length" class="acs-steps">
                <li v-for="st in m.steps" :key="st.round" class="acs-steps__item">
                  <span class="acs-steps__round">#{{ st.round }}</span>
                  检索「{{ st.query || '（无检索词）' }}」
                  <span v-if="st.hits > 0">→ 命中 {{ st.hits }} 段</span>
                  <span v-else class="acs-steps__none">→ 未命中</span>
                  <span class="acs-steps__meta">{{ st.chars }} 字 · {{ st.ms }}ms</span>
                  <span v-if="st.degraded" class="acs-steps__warn">
                    · {{ st.note || '降级' }}
                  </span>
                  <div v-if="st.articles?.length" class="acs-steps__from">
                    来自：{{ st.articles.join('、') }}
                  </div>
                </li>
              </ol>
              <p v-if="m.refs?.length" class="acs-msg__refs">
                命中片段：{{ m.refs.map((r) => `#${r.chunk_id} ${r.heading_path}`).join('、') }}
              </p>
            </div>
            </template>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- 删除确认：硬删不可恢复，二次确认不是可选的 UI 装饰 -->
    <Teleport to="body">
      <div v-if="deleting" class="modal-overlay" @click.self="deleting = null">
        <div class="acs-confirm">
          <h3 class="acs-confirm__title">删除会话 #{{ deleting.id }}？</h3>
          <p class="acs-confirm__desc">
            删除后该用户的对话记录会立即消失，<strong>不可恢复</strong>。
          </p>
          <div class="acs-confirm__actions">
            <button class="btn-sm" type="button" @click="deleting = null">取消</button>
            <button class="btn-sm btn-sm--danger" type="button" :disabled="deleteLoading" @click="confirmDelete">
              {{ deleteLoading ? '删除中…' : '确认删除' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  adminDeleteChatSession,
  adminGetChatSession,
  adminListChatSessions,
  type AdminMessageItem,
  type AdminSessionItem
} from '../api/chat'
import { showToast } from '../utils/toast'
import { calcPages, getPageRange } from '../utils/pagination'

const list = ref<AdminSessionItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)
const filter = ref({ user: '', article: '' })

const detail = ref<{ session: AdminSessionItem; messages: AdminMessageItem[] } | null>(null)
const detailLoading = ref(false)
const deleting = ref<AdminSessionItem | null>(null)
const deleteLoading = ref(false)

const totalPages = computed(() => calcPages(total.value, pageSize.value))
const pageList = computed(() => getPageRange(page.value, totalPages.value))

function fmt(v: string | null) {
  if (!v) return '—'
  const d = new Date(v)
  if (Number.isNaN(d.getTime())) return v
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

async function load(p: number) {
  loading.value = true
  try {
    const res = await adminListChatSessions({
      user: filter.value.user.trim(),
      article: filter.value.article.trim(),
      page: p,
      page_size: pageSize.value
    })
    list.value = res.list || []
    total.value = res.total
    page.value = res.page || p
  } catch (e: any) {
    showToast(e?.response?.data?.message || '加载会话列表失败', 'error')
    list.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

function resetFilter() {
  filter.value = { user: '', article: '' }
  load(1)
}

async function openDetail(id: number) {
  detailLoading.value = true
  // 先开弹窗再拉数据：拉取有延迟，先开能立刻给出「点了有反应」的反馈
  detail.value = { session: { id } as AdminSessionItem, messages: [] }
  try {
    detail.value = await adminGetChatSession(id)
  } catch (e: any) {
    detail.value = null
    showToast(e?.response?.data?.message || '加载会话详情失败', 'error')
  } finally {
    detailLoading.value = false
  }
}

function closeDetail() {
  detail.value = null
}

function askDelete(s: AdminSessionItem) {
  deleting.value = s
}

async function confirmDelete() {
  if (!deleting.value) return
  const id = deleting.value.id
  deleteLoading.value = true
  try {
    await adminDeleteChatSession(id)
    showToast('已删除', 'success')
    deleting.value = null
    if (detail.value?.session.id === id) detail.value = null
    // 删空当前页且不是第 1 页 → 回上一页，否则会停在空页
    const nextPage = list.value.length === 1 && page.value > 1 ? page.value - 1 : page.value
    await load(nextPage)
  } catch (e: any) {
    showToast(e?.response?.data?.message || '删除失败', 'error')
  } finally {
    deleteLoading.value = false
  }
}

onMounted(() => load(1))
</script>

<style scoped>
.acs {
  display: flex;
  flex-direction: column;
  gap: 12px;
  /* .admin-panel 是纯卡片容器、自身没有内边距，留白由面板自己负责 */
  padding: 20px 24px 24px;
  /* 撑满父级 + 表格区内部滚动：页面本身不滚，侧边栏才不会被顶出视口 */
  flex: 1;
  min-height: 0;
}

.acs__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-shrink: 0;
}

.acs__title {
  margin: 0;
  font-size: 1.05rem;
  color: var(--text-strong);
}

.acs__actions {
  display: flex;
  gap: 8px;
}

.acs__filter {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  flex-shrink: 0;
}

.acs__input {
  height: 32px;
  padding: 0 10px;
  border: 1px solid var(--border-strong);
  border-radius: 8px;
  background: var(--surface-strong);
  font-size: 0.82rem;
  color: var(--text-strong);
  outline: none;
}

.acs__input:focus {
  border-color: var(--primary-500);
}

.acs__table-wrap {
  flex: 1;
  min-height: 0;
  overflow: auto;
  border: 1px solid var(--border);
  border-radius: 12px;
}

.acs-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.8rem;
}

.acs-table th,
.acs-table td {
  padding: 9px 12px;
  text-align: left;
  border-bottom: 1px solid rgba(148, 163, 184, 0.2);
  vertical-align: middle;
}

.acs-table th {
  position: sticky;
  top: 0;
  z-index: 1;
  font-weight: 600;
  color: var(--text-muted);
  background: var(--surface-soft);
  white-space: nowrap;
}

.acs-table tbody tr:hover {
  background: rgba(148, 163, 184, 0.06);
}

.acs__empty {
  padding: 28px;
  text-align: center;
  color: var(--text-faint);
  font-size: 0.85rem;
}

.acs__mono {
  font-variant-numeric: tabular-nums;
  color: var(--text-secondary);
}

.acs__article {
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.acs__deleted {
  color: var(--text-faint);
  font-style: italic;
}

.acs__username {
  margin-left: 6px;
  color: var(--text-secondary);
}

.acs__time {
  white-space: nowrap;
  font-size: 0.8rem;
  color: var(--text-muted);
}

.acs__summary {
  max-width: 260px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-muted);
}

.acs__ops {
  display: flex;
  gap: 6px;
  white-space: nowrap;
}

/* ===== 详情弹窗 ===== */

.acs-modal {
  width: min(860px, 92vw);
  max-height: 84vh;
  display: flex;
  flex-direction: column;
  border-radius: var(--radius-lg);
  background: var(--surface-strong);
  box-shadow: var(--shadow-md);
  overflow: hidden;
}

.acs-modal__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 18px;
  border-bottom: 1px solid var(--border);
}

.acs-modal__title {
  margin: 0;
  font-size: 1rem;
  color: var(--text-strong);
}

.acs-modal__close {
  border: none;
  background: transparent;
  font-size: 1.3rem;
  line-height: 1;
  color: var(--text-muted);
  cursor: pointer;
}

.acs-modal__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  padding: 10px 18px;
  font-size: 0.78rem;
  color: var(--text-muted);
  background: var(--surface-soft);
  border-bottom: 1px solid var(--border);
}

.acs-modal__body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 14px 18px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.acs-msg {
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface-soft);
}

.acs-msg--user {
  background: var(--primary-soft);
}

.acs-msg__head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
  font-size: 0.72rem;
  color: var(--text-faint);
}

.acs-msg__role {
  font-weight: 600;
  color: var(--text-secondary);
}

.acs-msg__content {
  margin: 0;
  font-size: 0.85rem;
  line-height: 1.65;
  color: var(--text-strong);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.acs-msg__error {
  margin: 6px 0 0;
  font-size: 0.75rem;
  color: var(--danger);
}

/* ===== 检索过程（后台默认展开） ===== */

.acs-steps {
  margin: 8px 0 0;
  padding: 8px 10px;
  list-style: none;
  border-left: 2px solid var(--accent-soft);
  background: rgba(59, 130, 246, 0.04);
  border-radius: 0 8px 8px 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.acs-steps__item {
  font-size: 0.75rem;
  color: var(--text-secondary);
  overflow-wrap: anywhere;
}

.acs-steps__round {
  display: inline-block;
  min-width: 16px;
  margin-right: 4px;
  padding: 0 4px;
  border-radius: 4px;
  background: var(--accent-soft);
  color: var(--accent-500);
  font-size: 0.68rem;
  text-align: center;
}

.acs-steps__none {
  color: var(--warning);
}

.acs-steps__meta {
  margin-left: 6px;
  color: var(--text-faint);
}

.acs-steps__warn {
  color: var(--warning);
}

.acs-steps__from {
  margin-top: 2px;
  color: var(--text-faint);
  font-size: 0.72rem;
}

.acs-msg__refs {
  margin: 6px 0 0;
  font-size: 0.72rem;
  color: var(--text-faint);
  overflow-wrap: anywhere;
}

.acs-badge {
  padding: 1px 6px;
  border-radius: 6px;
  font-size: 0.68rem;
}

.acs-badge--refused {
  background: var(--surface-strong);
  color: var(--text-muted);
  border: 1px dashed var(--border-strong);
}

.acs-badge--error {
  background: var(--danger-soft);
  color: var(--danger);
}

.acs-badge--info {
  background: var(--accent-soft);
  color: var(--accent-500);
}

.acs-badge--muted {
  background: var(--surface-strong);
  color: var(--text-faint);
  border: 1px solid var(--border);
}

.acs-badge--warn {
  background: rgba(217, 119, 6, 0.12);
  color: var(--warning);
}

/* ===== 删除确认 ===== */

.acs-confirm {
  width: min(420px, 92vw);
  padding: 20px;
  border-radius: var(--radius-lg);
  background: var(--surface-strong);
  box-shadow: var(--shadow-md);
}

.acs-confirm__title {
  margin: 0 0 8px;
  font-size: 1rem;
  color: var(--text-strong);
}

.acs-confirm__desc {
  margin: 0 0 16px;
  font-size: 0.82rem;
  color: var(--text-muted);
}

.acs-confirm__actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
