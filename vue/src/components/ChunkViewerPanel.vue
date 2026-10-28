<template>
  <div class="cv-panel">
    <div class="cv-panel__header">
      <h2 class="cv-panel__title">切分管理</h2>
      <div class="cv-panel__actions">
        <button class="btn-sm btn-sm--danger" @click="openReset">清空向量数据</button>
        <button class="btn-sm btn-sm--danger" @click="openDeleteArticles">删除文章向量</button>
        <button class="btn-sm" @click="loadAll">刷新</button>
      </div>
    </div>

    <div class="cv-schema">
      <span class="cv-schema__item">
        向量表：
        <strong v-if="schema?.table_exists">{{ schema.dimension }} 维</strong>
        <strong v-else>未创建</strong>
      </span>
      <span class="cv-schema__item">
        向量模型：
        <strong v-if="schema?.model">{{ schema.model.model }}</strong>
        <strong v-else>未配置</strong>
      </span>
      <span class="cv-schema__msg" :class="{ 'cv-schema__msg--warn': schema && !schema.match && schema.model }">
        {{ schema?.message }}
      </span>
    </div>

    <div class="cv-cards">
      <div class="cv-card">
        <div class="cv-card__value">{{ summary.chunk_total }}</div>
        <div class="cv-card__label">切分块总数</div>
      </div>
      <div class="cv-card">
        <div class="cv-card__value">{{ summary.article_total }}</div>
        <div class="cv-card__label">已向量化文章</div>
      </div>
    </div>

    <div class="cv-tabs">
      <button class="cv-tab" :class="{ 'cv-tab--active': view === 'detail' }" @click="view = 'detail'">
        块明细
      </button>
      <button class="cv-tab" :class="{ 'cv-tab--active': view === 'stat' }" @click="switchToStat">
        聚合统计
      </button>
    </div>

    <!-- ===== 块明细 ===== -->
    <template v-if="view === 'detail'">
      <div class="cv-filter">
        <input
          v-model="filter.article_id"
          class="cv-input cv-input--sm"
          placeholder="文章 ID"
          @keyup.enter="loadChunks(1)"
        />
        <input
          v-model="filter.keyword"
          class="cv-input"
          placeholder="搜索块内容"
          @keyup.enter="loadChunks(1)"
        />
        <button class="btn-sm" @click="resetFilter">重置</button>
      </div>

      <table class="cv-table">
        <thead>
          <tr>
            <th>块 ID</th>
            <th>文章</th>
            <th>序号</th>
            <th>字符数</th>
            <th>原文预览</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="c in chunks" :key="c.id">
            <td class="cv-mono">{{ c.id }}</td>
            <td class="cv-ellipsis" :title="c.article_title">
              <span class="cv-muted">#{{ c.article_id }}</span> {{ c.article_title || '（已删除）' }}
            </td>
            <td class="cv-mono">{{ c.chunk_index }}</td>            <td class="cv-mono">{{ c.char_count }}</td>
            <td class="cv-preview">
              {{ c.content }}<span v-if="c.content_truncated" class="cv-more">…</span>
            </td>
            <td>
              <button class="btn-sm" @click="openFull(c)">查看全文</button>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-if="!loading && !chunks.length" class="cv-empty">
        暂无切分数据。到「文章管理」选一篇文章点「向量化」，或检查筛选条件。
      </div>

      <div v-if="total > 0" class="cv-pager">
        <button class="btn-sm" :disabled="page <= 1" @click="loadChunks(page - 1)">上一页</button>
        <span class="cv-pager__info">第 {{ page }} 页 / 共 {{ total }} 块</span>
        <button class="btn-sm" :disabled="page * pageSize >= total" @click="loadChunks(page + 1)">下一页</button>
      </div>
    </template>

    <!-- ===== 聚合统计 ===== -->
    <template v-else>
      <table class="cv-table">
        <thead>
          <tr>
            <th>文章</th>
            <th>块数</th>
            <th>平均块长</th>
            <th>最近切分</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="s in artStats" :key="s.article_id">
            <td class="cv-ellipsis" :title="s.article_title">
              <span class="cv-muted">#{{ s.article_id }}</span> {{ s.article_title }}
            </td>
            <td class="cv-mono">{{ s.chunk_count }}</td>
            <td class="cv-mono">{{ s.avg_chars }} 字</td>
            <td class="cv-time">{{ s.last_chunk_at || '—' }}</td>
          </tr>
        </tbody>
      </table>
      <div v-if="!artStats.length" class="cv-empty">暂无统计数据</div>
    </template>

    <Teleport to="body">
      <!-- 全文：white-space: pre-wrap 保留换行，切分后的换行就是段落边界 -->
      <div v-if="fullChunk" class="modal-overlay" @click.self="fullChunk = null">
        <div class="modal" role="dialog" aria-modal="true">
          <h3 class="modal__title">
            块 #{{ fullChunk.id }} · {{ fullChunk.article_title }} · 第 {{ fullChunk.chunk_index }} 块 ·
            {{ fullChunk.char_count }} 字
          </h3>
          <div class="modal__body">
            <pre class="cv-full">{{ fullChunk.content }}</pre>
          </div>
          <div class="modal__footer">
            <button class="btn" @click="fullChunk = null">关闭</button>
          </div>
        </div>
      </div>

      <!-- 删除文章向量：多选，一篇一列 -->
      <div v-if="delVisible" class="modal-overlay" @click.self="delVisible = false">
        <div class="modal modal--wide" role="dialog" aria-modal="true">
          <h3 class="modal__title">删除文章向量</h3>
          <div class="modal__body">
            <p class="modal__hint" style="margin: 0 0 10px">
              只删除选中文章的全部分块，并把它们的向量化状态重置为「未处理」。删除后可以重新向量化。
            </p>

            <div v-if="delLoading" class="cv-empty">加载中…</div>
            <div v-else-if="!delOptions.length" class="cv-empty">
              还没有任何文章完成向量化
            </div>

            <template v-else>
              <div class="del-toolbar">
                <span class="del-toolbar__count">
                  已选 <strong>{{ delSelected.length }}</strong> / {{ delOptions.length }} 篇
                </span>
                <div class="del-toolbar__btns">
                  <button class="btn-sm" @click="selectAll">全选</button>
                  <button class="btn-sm" @click="clearSelect">取消全选</button>
                </div>
              </div>

              <div class="del-grid">
                <label
                  v-for="a in delOptions"
                  :key="a.article_id"
                  class="del-item"
                  :class="{ 'del-item--on': delSelected.includes(a.article_id) }"
                >
                  <input v-model="delSelected" type="checkbox" :value="a.article_id" />
                  <span class="del-item__id">#{{ a.article_id }}</span>
                  <span class="del-item__title" :title="a.article_title">
                    {{ a.article_title || '（无标题）' }}
                  </span>
                  <span class="del-item__count">{{ a.chunk_count }} 块</span>
                </label>
              </div>
            </template>
          </div>
          <div class="modal__footer">
            <button class="btn" @click="delVisible = false">取消</button>
            <button
              class="btn btn--danger"
              :disabled="!delSelected.length || deleting"
              @click="doDeleteArticles"
            >
              {{ deleting ? '删除中…' : `删除选中的 ${delSelected.length} 篇` }}
            </button>
          </div>
        </div>
      </div>

      <!-- 清空向量数据：二次确认，且必须手输 RESET -->
      <div v-if="resetVisible" class="modal-overlay" @click.self="resetVisible = false">
        <div class="modal modal--delete" role="dialog" aria-modal="true">
          <h3 class="modal__title">清空向量数据</h3>
          <div class="modal__body">
            <div class="modal__alert">
              将永久删除全部 {{ summary.chunk_total }} 个向量块，并把 {{ summary.article_total }} 篇文章的向量化状态归零。
              此操作不可恢复。
            </div>
            <p class="modal__hint">
              用于「换向量模型且维度不同」的场景。清空后向量表会被删除，需要重新配置向量模型来按新维度重建。
            </p>
            <div class="modal__field" style="margin-top: 12px">
              <label class="modal__label">输入 RESET 确认</label>
              <input v-model="resetConfirm" class="modal__input" placeholder="RESET" />
            </div>
          </div>
          <div class="modal__footer">
            <button class="btn" @click="resetVisible = false">取消</button>
            <button class="btn btn--danger" :disabled="resetConfirm !== 'RESET' || resetting" @click="doReset">
              {{ resetting ? '清空中…' : '确认清空' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import api from '../api'
import { showToast } from '../utils/toast'

interface ChunkItem {
  id: number
  article_id: number
  article_title: string
  chunk_index: number
  char_count: number
  content: string
  content_truncated: boolean
  created_at: string
}

interface VectorSchema {
  table_exists: boolean
  dimension: number
  chunk_total: number
  article_total: number
  extension_ready: boolean
  model: { id: number; model: string; dimension: number; status: number } | null
  match: boolean
  message: string
}

const schema = ref<VectorSchema | null>(null)
const summary = reactive({ chunk_total: 0, article_total: 0 })

const view = ref<'detail' | 'stat'>('detail')

const filter = reactive({ article_id: '', keyword: '' })
const chunks = ref<ChunkItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const loading = ref(false)

const artStats = ref<any[]>([])

const fullChunk = ref<ChunkItem | null>(null)

// 删除文章向量：多选面板
interface DelOption {
  article_id: number
  article_title: string
  chunk_count: number
}
const delVisible = ref(false)
const delOptions = ref<DelOption[]>([])
const delSelected = ref<number[]>([])
const delLoading = ref(false)
const deleting = ref(false)

const resetVisible = ref(false)
const resetConfirm = ref('')
const resetting = ref(false)

const errMsg = (err: any, fallback: string) => err?.response?.data?.message || fallback

const loadSchema = async () => {
  try {
    const res = await api.get('/admin/vector/schema')
    schema.value = res.data?.data || null
  } catch {
    schema.value = null
  }
}

// summary 是全局的，不受筛选影响，所以单独从 stat 接口取一次
const loadSummary = async () => {
  try {
    const res = await api.get('/admin/chunks/stat', { params: { group_by: 'article' } })
    const s = res.data?.data?.summary || {}
    summary.chunk_total = s.chunk_total || 0
    summary.article_total = s.article_total || 0
  } catch {
    summary.chunk_total = 0
    summary.article_total = 0
  }
}

const loadChunks = async (p = 1) => {
  loading.value = true
  try {
    const params: Record<string, any> = { page: p, page_size: pageSize }
    if (filter.article_id) params.article_id = filter.article_id
    if (filter.keyword.trim()) params.keyword = filter.keyword.trim()

    const res = await api.get('/admin/chunks', { params })
    const data = res.data?.data || {}
    chunks.value = data.list || []
    total.value = data.total || 0
    page.value = data.page || p
  } catch (err: any) {
    if (err?.response?.status === 404) {
      showToast('接口不存在（404）：请确认后端已更新并重启', 'error')
    } else {
      showToast(errMsg(err, '加载切分数据失败'), 'error')
    }
    chunks.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

const loadStat = async () => {
  try {
    const res = await api.get('/admin/chunks/stat', { params: { group_by: 'article' } })
    artStats.value = res.data?.data?.list || []
  } catch (err: any) {
    artStats.value = []
    showToast(errMsg(err, '加载统计失败'), 'error')
  }
}

const switchToStat = async () => {
  view.value = 'stat'
  await loadStat()
}

const resetFilter = async () => {
  filter.article_id = ''
  filter.keyword = ''
  await loadChunks(1)
}

const openFull = (row: ChunkItem) => {
  // 列表里的 content 是截断的，必须再查一次拿全文
  api
    .get(`/admin/chunks/${row.id}`)
    .then((res) => {
      fullChunk.value = res.data?.data || row
    })
    .catch(() => {
      fullChunk.value = row
    })
}

const loadAll = async () => {
  await Promise.all([loadSchema(), loadSummary()])
  if (view.value === 'detail') {
    await loadChunks(page.value)
  } else {
    await loadStat()
  }
}

const openReset = () => {
  resetConfirm.value = ''
  resetVisible.value = true
}

/**
 * 删除文章向量：多选后批量删除。
 *
 * 候选列表直接复用「按文章统计」——它列出的就是有块的文章，也就是已向量化的文章，
 * 不需要再加一个专门接口。
 *
 * 只做整篇删除：块是「整篇重新切分、整体替换」的产物，删掉其中一块之后
 * 这篇文章的块集合既不是这次切分的结果、也不是上次的。
 */
const openDeleteArticles = async () => {
  delVisible.value = true
  delSelected.value = []
  delLoading.value = true
  try {
    const res = await api.get('/admin/chunks/stat', { params: { group_by: 'article' } })
    delOptions.value = res.data?.data?.list || []
  } catch (err: any) {
    delOptions.value = []
    showToast(errMsg(err, '加载已向量化文章失败'), 'error')
  } finally {
    delLoading.value = false
  }
}

const selectAll = () => {
  delSelected.value = delOptions.value.map((a) => a.article_id)
}

const clearSelect = () => {
  delSelected.value = []
}

const doDeleteArticles = async () => {
  if (!delSelected.value.length) return
  deleting.value = true
  try {
    const res = await api.post('/admin/chunks/articles/delete', {
      article_ids: delSelected.value
    })
    const d = res.data?.data || {}
    const skipped = d.skipped || []
    if (skipped.length) {
      // 有跳过就说清楚原因，不能只报「成功」——用户会发现少了文章
      showToast(
        `已删除 ${d.deleted_articles} 篇（${d.deleted_chunks} 块），跳过 ${skipped.length} 篇：${skipped[0].reason}`,
        'error'
      )
    } else {
      showToast(`已删除 ${d.deleted_articles} 篇共 ${d.deleted_chunks} 块`, 'success')
    }
    delVisible.value = false
    await loadAll()
  } catch (err: any) {
    showToast(errMsg(err, '删除失败'), 'error')
  } finally {
    deleting.value = false
  }
}

const doReset = async () => {
  resetting.value = true
  try {
    const res = await api.post('/admin/vector/reset', { confirm: 'RESET' })
    const d = res.data?.data || {}
    showToast(`已清空 ${d.deleted_chunks || 0} 个块，重置 ${d.reset_articles || 0} 篇文章`, 'success')
    resetVisible.value = false
    await loadAll()
  } catch (err: any) {
    showToast(errMsg(err, '清空失败'), 'error')
  } finally {
    resetting.value = false
  }
}

onMounted(loadAll)
</script>

<style scoped>
.cv-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.cv-panel__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.cv-panel__title {
  margin: 0;
  font-size: 1rem;
  font-weight: 600;
}

.cv-panel__actions {
  display: flex;
  gap: 8px;
}

.cv-schema {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 16px;
  padding: 8px 12px;
  border-radius: 8px;
  font-size: 0.78rem;
  color: #555;
  background: rgba(148, 163, 184, 0.08);
}

.cv-schema__item strong {
  font-weight: 600;
  color: #222;
}

.cv-schema__msg {
  color: #888;
}

.cv-schema__msg--warn {
  color: #d46b08;
  font-weight: 500;
}

.cv-cards {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.cv-card {
  padding: 14px 16px;
  border-radius: 10px;
  background: rgba(22, 119, 255, 0.06);
  border: 1px solid rgba(22, 119, 255, 0.15);
}

.cv-card__value {
  font-size: 1.4rem;
  font-weight: 600;
  color: #1677ff;
  line-height: 1.2;
}

.cv-card__label {
  margin-top: 2px;
  font-size: 0.75rem;
  color: #666;
}

.cv-tabs {
  display: flex;
  gap: 4px;
  border-bottom: 1px solid rgba(148, 163, 184, 0.25);
}

.cv-tab {
  padding: 7px 14px;
  border: none;
  background: transparent;
  font-size: 0.82rem;
  color: #666;
  cursor: pointer;
  border-bottom: 2px solid transparent;
}

.cv-tab--active {
  color: #1677ff;
  font-weight: 600;
  border-bottom-color: #1677ff;
}

.cv-filter {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.cv-input {
  padding: 6px 10px;
  border: 1px solid rgba(148, 163, 184, 0.5);
  border-radius: 6px;
  font-size: 0.8rem;
  min-width: 160px;
}

.cv-input--sm {
  min-width: 100px;
  width: 100px;
}

.cv-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.8rem;
  table-layout: fixed;
}

.cv-table th,
.cv-table td {
  padding: 8px 10px;
  text-align: left;
  border-bottom: 1px solid rgba(148, 163, 184, 0.2);
  vertical-align: top;
}

.cv-table th {
  font-weight: 600;
  color: #666;
  background: rgba(148, 163, 184, 0.06);
  white-space: nowrap;
}

.cv-mono {
  font-family: var(--font-mono, monospace);
  font-size: 0.76rem;
  white-space: nowrap;
}

.cv-muted {
  color: #aaa;
}

/* ---------- 删除文章向量：多选面板 ---------- */
.modal.modal--wide {
  width: min(860px, 94vw);
  max-width: none;
}

.del-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}

.del-toolbar__count {
  font-size: 0.8rem;
  color: #666;
}

.del-toolbar__btns {
  display: flex;
  gap: 6px;
}

/* 一篇一列：auto-fill 让宽屏并排多列、窄屏自动减列，不用写媒体查询 */
.del-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(190px, 1fr));
  gap: 8px;
  max-height: 46vh;
  overflow-y: auto;
  padding: 2px;
}

.del-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 10px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  font-size: 0.8rem;
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
}

.del-item:hover {
  border-color: #c7d2fe;
  background: #f8faff;
}

/* 选中态整格变色，而不是只有小方框：勾选后要一眼看出选了哪些 */
.del-item--on {
  border-color: #6366f1;
  background: #eef2ff;
}

.del-item input {
  margin: 0;
  flex: 0 0 auto;
}

.del-item__id {
  flex: 0 0 auto;
  color: #999;
  font-size: 0.72rem;
}

.del-item__title {
  flex: 1 1 auto;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #333;
}

.del-item__count {
  flex: 0 0 auto;
  color: #999;
  font-size: 0.72rem;
}

.cv-ellipsis {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.cv-preview {
  color: #555;
  line-height: 1.5;
  max-height: 64px;
  overflow: hidden;
  word-break: break-all;
}

.cv-more {
  color: #aaa;
}

.cv-time {
  font-size: 0.74rem;
  color: #888;
  white-space: nowrap;
}

.cv-empty {
  padding: 24px 0;
  text-align: center;
  color: #999;
  font-size: 0.85rem;
}

.cv-pager {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding-top: 4px;
}

.cv-pager__info {
  font-size: 0.78rem;
  color: #666;
}

/* ===== 弹窗 ===== */
.modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: rgba(15, 23, 42, 0.45);
}

.modal {
  width: 100%;
  max-width: 760px;
  max-height: 86vh;
  display: flex;
  flex-direction: column;
  border-radius: 12px;
  background: #fff;
  box-shadow: 0 12px 32px rgba(15, 23, 42, 0.18);
}

.modal--delete {
  max-width: 460px;
}

.modal__title {
  margin: 0;
  padding: 14px 20px;
  font-size: 0.9rem;
  font-weight: 600;
  border-bottom: 1px solid rgba(148, 163, 184, 0.2);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.modal__body {
  padding: 16px 20px;
  overflow-y: auto;
}

.modal__field {
  margin-bottom: 12px;
}

.modal__label {
  display: block;
  margin-bottom: 4px;
  font-size: 0.78rem;
  color: #555;
}

.modal__input {
  width: 100%;
  padding: 7px 10px;
  border: 1px solid rgba(148, 163, 184, 0.5);
  border-radius: 6px;
  font-size: 0.82rem;
  box-sizing: border-box;
}

.modal__hint {
  margin: 8px 0 0;
  font-size: 0.76rem;
  line-height: 1.55;
  color: #888;
}

.modal__alert {
  padding: 8px 10px;
  border-radius: 6px;
  font-size: 0.78rem;
  line-height: 1.5;
  color: #a8071a;
  background: #fff1f0;
  border: 1px solid #ffa39e;
}

.modal__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 20px;
  border-top: 1px solid rgba(148, 163, 184, 0.2);
}

/* 切分后的换行就是段落边界，必须保留 */
.cv-full {
  margin: 0;
  padding: 12px;
  border-radius: 8px;
  background: #fafafa;
  border: 1px solid rgba(148, 163, 184, 0.3);
  font-family: var(--font-mono, monospace);
  font-size: 0.78rem;
  line-height: 1.7;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 60vh;
  overflow-y: auto;
}

.btn,
.btn-sm {
  cursor: pointer;
  border-radius: 6px;
  font-size: 0.8rem;
  border: 1px solid rgba(148, 163, 184, 0.5);
  background: #fff;
  padding: 5px 12px;
  transition: all 0.15s;
}

.btn:hover,
.btn-sm:hover {
  border-color: #1677ff;
  color: #1677ff;
}

.btn-sm:disabled,
.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-sm--danger {
  color: #ff4d4f;
  border-color: #ff4d4f;
}

.btn-sm--danger:hover {
  background: #ff4d4f;
  color: #fff;
}

.btn--danger {
  background: #ff4d4f;
  color: #fff;
  border-color: #ff4d4f;
}
</style>
