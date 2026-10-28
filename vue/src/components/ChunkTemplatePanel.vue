<template>
  <div class="ct-panel">
    <div class="ct-panel__header">
      <h2 class="ct-panel__title">切分模板</h2>
      <div class="ct-panel__actions">
        <button class="btn-sm btn-sm--primary" @click="openCreate">+ 新增模板</button>
      </div>
    </div>

    <p class="ct-panel__tip">
      模板决定文章正文怎么切成块。切分按「段落 → 换行 → 句末 → 逗号 → 硬切」逐级降级，
      三个数值参数控制块大小、相邻块重叠与碎块合并。
      <strong>已被使用的模板不能改切分参数</strong>——改了之后库里已存的块与新参数不再对应，只能全部重新向量化。
    </p>

    <table class="ct-table">
      <thead>
        <tr>
          <th>ID</th>
          <th>名称</th>
          <th>块大小</th>
          <th>重叠</th>
          <th>碎块阈值</th>
          <th>剥离HTML</th>
          <th>已用文章</th>
          <th>块数</th>
          <th>状态</th>
          <th>操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="row in list" :key="row.id">
          <td>{{ row.id }}</td>
          <td>
            <div class="ct-name">{{ row.name }}</div>
            <div v-if="row.description" class="ct-desc">{{ row.description }}</div>
          </td>
          <td class="ct-mono">{{ row.chunk_size }}</td>
          <td class="ct-mono">{{ row.chunk_overlap }}</td>
          <td class="ct-mono">{{ row.min_chunk_size }}</td>
          <td>{{ row.strip_html ? '是' : '否' }}</td>
          <td class="ct-mono">{{ row.used_article_count }}</td>
          <td class="ct-mono">{{ row.chunk_count }}</td>
          <td>
            <span class="ct-badge" :class="row.status === 1 ? 'ct-badge--on' : 'ct-badge--off'">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </span>
          </td>
          <td>
            <div class="ct-actions">
              <button class="btn-sm" @click="toggleStatus(row)">
                {{ row.status === 1 ? '禁用' : '启用' }}
              </button>
              <button class="btn-sm" @click="openEdit(row)">编辑</button>
              <button class="btn-sm btn-sm--danger" @click="askDelete(row)">删除</button>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
    <div v-if="!loading && !list.length" class="ct-empty">暂无切分模板，点击「+ 新增模板」开始</div>
    <div v-if="loading" class="ct-empty">加载中...</div>

    <!-- 必须 Teleport：本组件在 .admin-main 内，那里 overflow-x:auto 会连带裁剪弹窗 -->
    <Teleport to="body">
      <div v-if="modalVisible" class="modal-overlay" @click.self="closeModal">
        <div class="modal" role="dialog" aria-modal="true">
          <h3 class="modal__title">{{ editingId === null ? '新增切分模板' : '编辑切分模板' }}</h3>

          <div class="modal__body">
            <div class="modal__field">
              <label class="modal__label"><span class="required-mark">*</span>模板名称</label>
              <input v-model="form.name" class="modal__input" maxlength="64" placeholder="如：长文粗切" />
            </div>

            <div class="modal__field">
              <label class="modal__label">用途说明</label>
              <input v-model="form.description" class="modal__input" maxlength="255" placeholder="选填，如：适合 5000 字以上的技术长文" />
            </div>

            <div v-if="paramsLocked" class="modal__alert">
              该模板已被 {{ editingRow?.used_article_count }} 篇文章使用（共 {{ editingRow?.chunk_count }} 块），
              切分参数不可修改。如需调整请新建一个模板。
            </div>

            <div class="modal__row">
              <div class="modal__field">
                <label class="modal__label"><span class="required-mark">*</span>块大小（字符）</label>
                <input
                  v-model.number="form.chunk_size"
                  class="modal__input"
                  type="number"
                  min="100"
                  max="4000"
                  :disabled="paramsLocked"
                />
              </div>
              <div class="modal__field">
                <label class="modal__label">相邻块重叠（字符）</label>
                <input
                  v-model.number="form.chunk_overlap"
                  class="modal__input"
                  type="number"
                  min="0"
                  :max="Math.floor(form.chunk_size / 2)"
                  :disabled="paramsLocked"
                />
              </div>
              <div class="modal__field">
                <label class="modal__label">碎块合并阈值（字符）</label>
                <input
                  v-model.number="form.min_chunk_size"
                  class="modal__input"
                  type="number"
                  min="0"
                  :max="form.chunk_size"
                  :disabled="paramsLocked"
                />
              </div>
            </div>

            <p class="modal__hint">
              块大小 100~4000；重叠不超过块大小的一半（避免相邻块大量重复）；碎块阈值以下的内容会并入相邻块。
            </p>

            <div class="modal__row">
              <div class="modal__field">
                <label class="modal__label">剥离 HTML 标签</label>
                <select v-model="form.strip_html" class="modal__input" :disabled="paramsLocked">
                  <option :value="true">是（正文是富文本，推荐）</option>
                  <option :value="false">否（正文是纯文本/Markdown）</option>
                </select>
                <p v-if="paramsLocked" class="modal__hint">已使用的模板该项也不可修改</p>
              </div>
              <div class="modal__field">
                <label class="modal__label">状态</label>
                <select v-model.number="form.status" class="modal__input">
                  <option :value="1">启用</option>
                  <option :value="0">禁用</option>
                </select>
              </div>
            </div>
          </div>

          <div class="modal__footer">
            <button class="btn btn--cancel" @click="closeModal">取消</button>
            <button class="btn btn--primary" :disabled="!canSave || saving" @click="save">
              {{ saving ? '保存中…' : '保存' }}
            </button>
          </div>
        </div>
      </div>

      <div v-if="deleteModalVisible" class="modal-overlay" @click.self="closeDelete">
        <div class="modal modal--delete">
          <h3 class="modal__title">确认删除</h3>
          <div class="modal__body">
            <p class="delete-confirm-text">
              确定要删除模板「{{ deleteTarget?.name }}」吗？此操作不可恢复。
            </p>
          </div>
          <div class="modal__footer">
            <button class="btn btn--cancel" @click="closeDelete">取消</button>
            <button class="btn btn--danger" @click="confirmDelete">确认删除</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import api from '../api'
import { showToast } from '../utils/toast'

interface ChunkTemplate {
  id: number
  name: string
  description: string
  chunk_size: number
  chunk_overlap: number
  min_chunk_size: number
  strip_html: boolean
  status: number
  used_article_count: number
  chunk_count: number
}

const list = ref<ChunkTemplate[]>([])
const loading = ref(false)

const modalVisible = ref(false)
const editingId = ref<number | null>(null)
const editingRow = ref<ChunkTemplate | null>(null)
const saving = ref(false)

const deleteModalVisible = ref(false)
const deleteTarget = ref<ChunkTemplate | null>(null)

const form = reactive({
  name: '',
  description: '',
  chunk_size: 800,
  chunk_overlap: 80,
  min_chunk_size: 100,
  strip_html: true,
  status: 1
})

// 已被使用的模板，参数锁死。后端也会拦，这里提前置灰是为了不让用户白填一遍
const paramsLocked = computed(
  () => editingId.value !== null && (editingRow.value?.used_article_count || 0) > 0
)

const canSave = computed(() => {
  if (!form.name.trim()) return false
  if (form.chunk_size < 100 || form.chunk_size > 4000) return false
  if (form.chunk_overlap < 0 || form.chunk_overlap > Math.floor(form.chunk_size / 2)) return false
  if (form.min_chunk_size < 0 || form.min_chunk_size > form.chunk_size) return false
  return true
})

const errMsg = (err: any, fallback: string) => err?.response?.data?.message || fallback

const load = async () => {
  loading.value = true
  try {
    const res = await api.get('/admin/chunk-templates')
    list.value = res.data?.data || []
  } catch (err: any) {
    if (err?.response?.status === 404) {
      showToast('接口不存在（404）：请确认后端已更新并重启', 'error')
    } else {
      showToast(errMsg(err, '加载切分模板失败'), 'error')
    }
  } finally {
    loading.value = false
  }
}

const openCreate = () => {
  editingId.value = null
  editingRow.value = null
  Object.assign(form, {
    name: '',
    description: '',
    chunk_size: 800,
    chunk_overlap: 80,
    min_chunk_size: 100,
    strip_html: true,
    status: 1
  })
  modalVisible.value = true
}

const openEdit = (row: ChunkTemplate) => {
  editingId.value = row.id
  editingRow.value = row
  Object.assign(form, {
    name: row.name,
    description: row.description || '',
    chunk_size: row.chunk_size,
    chunk_overlap: row.chunk_overlap,
    min_chunk_size: row.min_chunk_size,
    strip_html: row.strip_html,
    status: row.status
  })
  modalVisible.value = true
}

const closeModal = () => {
  modalVisible.value = false
  editingRow.value = null
}

const save = async () => {
  saving.value = true
  try {
    if (editingId.value === null) {
      await api.post('/admin/chunk-templates', {
        name: form.name.trim(),
        description: form.description.trim(),
        chunk_size: form.chunk_size,
        chunk_overlap: form.chunk_overlap,
        min_chunk_size: form.min_chunk_size,
        strip_html: form.strip_html,
        status: form.status
      })
      showToast('模板已创建', 'success')
    } else {
      // 参数锁定时只提交名称/说明/状态：把锁定的数字也传上去，
      // 万一用户通过控制台解除了 disabled，后端会因「值没变」放行，语义一致
      const payload: Record<string, any> = {
        name: form.name.trim(),
        description: form.description.trim(),
        status: form.status
      }
      if (!paramsLocked.value) {
        payload.chunk_size = form.chunk_size
        payload.chunk_overlap = form.chunk_overlap
        payload.min_chunk_size = form.min_chunk_size
        payload.strip_html = form.strip_html
      }
      await api.put(`/admin/chunk-templates/${editingId.value}`, payload)
      showToast('模板已更新', 'success')
    }
    closeModal()
    await load()
  } catch (err: any) {
    showToast(errMsg(err, '保存失败'), 'error')
  } finally {
    saving.value = false
  }
}

// 列表里直接切状态，比进编辑页改更顺手
const toggleStatus = async (row: ChunkTemplate) => {
  try {
    await api.put(`/admin/chunk-templates/${row.id}`, { status: row.status === 1 ? 0 : 1 })
    showToast(row.status === 1 ? '已禁用' : '已启用', 'success')
    await load()
  } catch (err: any) {
    showToast(errMsg(err, '操作失败'), 'error')
  }
}

const askDelete = (row: ChunkTemplate) => {
  // 已被使用的模板直接提示，避免点了删除才看到 409
  if (row.chunk_count > 0) {
    showToast(
      `该模板已被 ${row.used_article_count} 篇文章使用（共 ${row.chunk_count} 块），无法删除；` +
        `请先将相关文章改用其他模板重新向量化`,
      'error'
    )
    return
  }
  deleteTarget.value = row
  deleteModalVisible.value = true
}

const closeDelete = () => {
  deleteModalVisible.value = false
  deleteTarget.value = null
}

const confirmDelete = async () => {
  const row = deleteTarget.value
  if (!row) return
  try {
    await api.delete(`/admin/chunk-templates/${row.id}`)
    showToast('已删除', 'success')
    closeDelete()
    await load()
  } catch (err: any) {
    showToast(errMsg(err, '删除失败'), 'error')
  }
}

onMounted(load)
</script>

<style scoped>
.ct-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.ct-panel__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.ct-panel__title {
  margin: 0;
  font-size: 1rem;
  font-weight: 600;
}

.ct-panel__tip {
  margin: 0;
  padding: 10px 12px;
  border-radius: 8px;
  font-size: 0.78rem;
  line-height: 1.6;
  color: #666;
  background: rgba(22, 119, 255, 0.06);
  border: 1px solid rgba(22, 119, 255, 0.15);
}

.ct-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.82rem;
}

.ct-table th,
.ct-table td {
  padding: 8px 10px;
  text-align: left;
  border-bottom: 1px solid rgba(148, 163, 184, 0.2);
  vertical-align: middle;
}

.ct-table th {
  font-weight: 600;
  color: #666;
  white-space: nowrap;
  background: rgba(148, 163, 184, 0.06);
}

.ct-name {
  font-weight: 500;
}

.ct-desc {
  margin-top: 2px;
  font-size: 0.72rem;
  color: #999;
}

.ct-mono {
  font-family: var(--font-mono, monospace);
  font-size: 0.78rem;
}

.ct-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 0.72rem;
  font-weight: 600;
}

.ct-badge--on {
  color: var(--primary-600);
  background: var(--primary-soft);
}

.ct-badge--off {
  color: #888;
  background: rgba(148, 163, 184, 0.18);
}

.ct-actions {
  display: flex;
  gap: 6px;
  flex-wrap: nowrap;
}

.ct-empty {
  padding: 24px 0;
  text-align: center;
  color: #999;
  font-size: 0.85rem;
}

/* ===== 弹窗（Teleport 到 body）===== */
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
  max-width: 620px;
  max-height: 86vh;
  display: flex;
  flex-direction: column;
  border-radius: 12px;
  background: #fff;
  box-shadow: 0 12px 32px rgba(15, 23, 42, 0.18);
}

.modal--delete {
  max-width: 420px;
}

.modal__title {
  margin: 0;
  padding: 16px 20px;
  font-size: 0.95rem;
  font-weight: 600;
  border-bottom: 1px solid rgba(148, 163, 184, 0.2);
}

.modal__body {
  padding: 16px 20px;
  overflow-y: auto;
}

.modal__field {
  margin-bottom: 12px;
}

.modal__row {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
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
  background: #fff;
  box-sizing: border-box;
}

.modal__input:disabled {
  background: #f5f5f5;
  color: #999;
  cursor: not-allowed;
}

.modal__hint {
  margin: 4px 0 0;
  font-size: 0.75rem;
  line-height: 1.5;
  color: #888;
}

.modal__alert {
  margin: 4px 0 12px;
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

.required-mark {
  color: #ff4d4f;
  margin-right: 2px;
}

.delete-confirm-text {
  margin: 0;
  font-size: 0.85rem;
  line-height: 1.6;
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

.btn-sm--primary {
  background: #1677ff;
  color: #fff;
  border-color: #1677ff;
}

.btn-sm--primary:hover {
  background: #4096ff;
  border-color: #4096ff;
  color: #fff;
}

.btn-sm--danger {
  color: #ff4d4f;
  border-color: #ff4d4f;
}

.btn-sm--danger:hover {
  background: #ff4d4f;
  color: #fff;
}

.btn--primary {
  background: #1677ff;
  color: #fff;
  border-color: #1677ff;
}

.btn--primary:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.btn--danger {
  background: #ff4d4f;
  color: #fff;
  border-color: #ff4d4f;
}
</style>
