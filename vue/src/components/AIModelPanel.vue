<template>
  <div class="ai-panel">
    <div class="ai-panel__header">
      <h2 class="ai-panel__title">AI 模型配置</h2>
      <div class="ai-panel__actions">
        <button class="btn-sm btn-sm--primary" @click="openCreate">+ 新增配置</button>
      </div>
    </div>

    <p class="ai-panel__tip">
      大模型走 OpenAI 兼容协议。新增配置需先「测试连接」通过才能保存；编辑仅可修改超时、最大 token、
      温度与状态，保存不再测试连接。使用 AI 功能（如文章「一键生成摘要」）时，选择要使用的模型配置。
    </p>

    <table class="ai-table">
      <thead>
        <tr>
          <th>ID</th>
          <th>模型 id</th>
          <th>Base URL</th>
          <th>密钥</th>
          <th>状态</th>
          <th>操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="row in list" :key="row.id">
          <td>{{ row.id }}</td>
          <td>{{ row.model }}</td>
          <td class="ai-table__url" :title="row.base_url">{{ row.base_url }}</td>
          <td class="ai-table__mono">{{ row.api_key_masked || '—' }}</td>
          <td>
            <span class="ai-badge" :class="row.status === 1 ? 'ai-badge--on' : 'ai-badge--off'">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </span>
          </td>
          <td>
            <div class="ai-actions">
              <button class="btn-sm" :disabled="busyId === row.id" @click="testRow(row)">
                {{ busyId === row.id ? '测试中' : '测试' }}
              </button>
              <button class="btn-sm" @click="openEdit(row)">编辑</button>
              <button class="btn-sm btn-sm--danger" @click="remove(row)">删除</button>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
    <div v-if="!loading && !list.length" class="ai-empty">暂无模型配置，点击「+ 新增配置」开始</div>
    <div v-if="loading" class="ai-empty">加载中...</div>

    <!--
      新增 / 编辑弹窗。
      必须 Teleport 到 body：本组件位于 .admin-main 内，而 .admin-main 有 overflow-x:auto
      （会让 overflow-y 一并变成 auto）会把弹窗裁掉，导致显示不全、底部点不到「保存」。
    -->
    <Teleport to="body">
      <div v-if="modalVisible" class="modal-overlay" @click.self="closeModal">
        <div class="modal" role="dialog" aria-modal="true">
          <h3 class="modal__title">{{ editingId === null ? '新增模型配置' : '编辑模型配置' }}</h3>

          <div class="modal__body">
            <div class="modal__field">
              <label class="modal__label">
                <span v-if="editingId === null" class="required-mark">*</span>Base URL
              </label>
              <input
                v-model="form.base_url"
                class="modal__input"
                :disabled="editingId !== null"
                placeholder="openai格式，如 https://api.xxx.com/v1"
              />
            </div>

            <div v-if="editingId === null" class="modal__field">
              <label class="modal__label"><span class="required-mark">*</span>API Key</label>
              <input
                v-model="form.api_key"
                class="modal__input"
                type="password"
                placeholder="请输入密钥"
                autocomplete="new-password"
              />
            </div>

            <div class="modal__field">
              <label class="modal__label">
                <span v-if="editingId === null" class="required-mark">*</span>模型 id
              </label>
              <input
                v-model="form.model"
                class="modal__input"
                :disabled="editingId !== null"
                placeholder="如：deepseek-chat"
                maxlength="128"
              />
            </div>

            <div class="modal__row">
              <div class="modal__field">
                <label class="modal__label">超时（秒）</label>
                <input v-model.number="form.timeout_sec" class="modal__input" type="number" min="1" max="300" />
              </div>
              <div class="modal__field">
                <label class="modal__label">max_tokens</label>
                <input v-model.number="form.max_tokens" class="modal__input" type="number" min="1" max="32768" />
              </div>
              <div class="modal__field">
                <label class="modal__label">temperature</label>
                <input v-model.number="form.temperature" class="modal__input" type="number" min="0" max="2" step="0.1" />
              </div>
            </div>

            <div class="modal__row">
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
              {{ saveLabel }}
            </button>
          </div>
        </div>
      </div>

      <!-- 删除确认：与其他管理页保持一致的弹窗样式 -->
      <div v-if="deleteModalVisible" class="modal-overlay" @click.self="closeDelete">
        <div class="modal modal--delete">
          <h3 class="modal__title">确认删除</h3>
          <div class="modal__body">
            <p class="delete-confirm-text">
              确定要删除模型配置「{{ deleteTarget?.model }}」吗？此操作不可恢复。
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

interface ModelConfig {
  id: number
  base_url: string
  api_key_masked: string
  model: string
  timeout_sec: number
  max_tokens: number
  temperature: number
  status: number
}

const list = ref<ModelConfig[]>([])
const loading = ref(false)
const busyId = ref<number | null>(null)

const modalVisible = ref(false)
const editingId = ref<number | null>(null)
const saving = ref(false)

const deleteModalVisible = ref(false)
const deleteTarget = ref<ModelConfig | null>(null)

const form = reactive({
  base_url: '',
  api_key: '',
  model: '',
  timeout_sec: 30,
  max_tokens: 2048,
  temperature: 0.3,
  status: 1
})

// 新增：base_url、api_key、model 均填完后才可保存；编辑：仅改运行参数与状态，随时可保存
const canSave = computed(() => {
  if (editingId.value !== null) return true
  return !!(form.base_url.trim() && form.api_key.trim() && form.model.trim())
})

const saveLabel = computed(() => {
  if (saving.value) return editingId.value === null ? '测试连接并保存…' : '保存中…'
  return '保存'
})

const errMsg = (err: any, fallback: string) => err?.response?.data?.message || fallback

const load = async () => {
  loading.value = true
  try {
    const res = await api.get('/admin/llm-configs')
    list.value = res.data?.data || []
  } catch (err: any) {
    // 给出可诊断的提示：404 多半是后端未更新/未重启
    if (err?.response?.status === 404) {
      showToast('接口不存在（404）：请确认后端已更新并重启', 'error')
    } else {
      showToast(errMsg(err, '加载模型配置失败'), 'error')
    }
  } finally {
    loading.value = false
  }
}

const openCreate = () => {
  editingId.value = null
  Object.assign(form, {
    base_url: '',
    api_key: '',
    model: '',
    timeout_sec: 30,
    max_tokens: 2048,
    temperature: 0.3,
    status: 1
  })
  modalVisible.value = true
}

const openEdit = (row: ModelConfig) => {
  editingId.value = row.id
  Object.assign(form, {
    base_url: row.base_url,
    api_key: '', // 编辑态不展示、也不允许改密钥
    model: row.model,
    timeout_sec: row.timeout_sec,
    max_tokens: row.max_tokens,
    temperature: row.temperature,
    status: row.status
  })
  modalVisible.value = true
}

const closeModal = () => {
  modalVisible.value = false
}

// 列表行快捷测试（复用已存密钥）
const testRow = async (row: ModelConfig) => {
  busyId.value = row.id
  try {
    const res = await api.post('/admin/llm-configs/test', {
      id: row.id,
      base_url: row.base_url,
      model: row.model,
      timeout_sec: row.timeout_sec
    })
    const data = res.data?.data || {}
    if (data.ok) {
      showToast(`连接正常 · ${data.latency_ms || 0}ms`, 'success')
    } else {
      showToast(data.error || '连接失败', 'error')
    }
  } catch (err: any) {
    showToast(errMsg(err, '测试请求失败'), 'error')
  } finally {
    busyId.value = null
  }
}

const save = async () => {
  saving.value = true
  try {
    if (editingId.value === null) {
      await api.post('/admin/llm-configs', {
        base_url: form.base_url.trim(),
        api_key: form.api_key.trim(),
        model: form.model.trim(),
        timeout_sec: form.timeout_sec,
        max_tokens: form.max_tokens,
        temperature: form.temperature,
        status: form.status
      })
      showToast('配置已保存', 'success')
    } else {
      await api.put(`/admin/llm-configs/${editingId.value}`, {
        timeout_sec: form.timeout_sec,
        max_tokens: form.max_tokens,
        temperature: form.temperature,
        status: form.status
      })
      showToast('配置已更新', 'success')
    }
    closeModal()
    await load()
  } catch (err: any) {
    showToast(errMsg(err, '保存失败'), 'error')
  } finally {
    saving.value = false
  }
}

const remove = (row: ModelConfig) => {
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
    await api.delete(`/admin/llm-configs/${row.id}`)
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
.ai-panel {
  background: #fff;
  border-radius: var(--radius-md);
  padding: 20px 24px 24px;
}

.ai-panel__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.ai-panel__title {
  font-size: 1rem;
  font-weight: 700;
  color: var(--text-strong);
  margin: 0;
}

.ai-panel__tip {
  font-size: 0.78rem;
  color: #999;
  margin: 0 0 14px;
  line-height: 1.6;
}

.ai-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.85rem;
}

.ai-table th,
.ai-table td {
  padding: 10px 12px;
  text-align: left;
  border-bottom: 1px solid var(--border);
  vertical-align: middle;
}

.ai-table th {
  font-weight: 600;
  color: var(--text-strong);
  background: #fafafa;
  white-space: nowrap;
}

.ai-table__url {
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-strong);
}

.ai-table__mono {
  font-family: 'JetBrains Mono', 'Consolas', monospace;
  font-size: 0.78rem;
  color: #666;
}

.ai-actions {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.ai-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 0.72rem;
  font-weight: 600;
}

.ai-badge--on {
  color: var(--primary-600);
  background: var(--primary-soft);
}

.ai-badge--off {
  color: #888;
  background: rgba(148, 163, 184, 0.18);
}

.ai-empty {
  padding: 24px 0;
  text-align: center;
  color: #999;
  font-size: 0.85rem;
}

/* ===== 弹窗（Teleport 到 body）===== */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 42, 0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  box-sizing: border-box;
  z-index: 1000;
}

/* 标题与底部固定、中间内容滚动 —— 保证「保存」按钮始终可见可点 */
.modal {
  display: flex;
  flex-direction: column;
  width: 600px;
  max-width: 100%;
  max-height: 100%;
  background: #fff;
  border-radius: var(--radius-md);
  overflow: hidden;
  box-shadow: 0 12px 40px rgba(15, 23, 42, 0.22);
}

.modal__title {
  flex-shrink: 0;
  margin: 0;
  padding: 16px 24px;
  font-size: 1rem;
  font-weight: 700;
  color: var(--text-strong);
  border-bottom: 1px solid var(--border);
}

.modal__body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 16px 24px;
}

.modal__field {
  margin-bottom: 12px;
  flex: 1;
  min-width: 0;
}

.modal__row {
  display: flex;
  gap: 12px;
}

.modal__label {
  display: block;
  margin-bottom: 6px;
  font-size: 0.8rem;
  color: #555;
}

.required-mark {
  color: var(--danger);
  margin-right: 2px;
}

.modal__input {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--border-strong);
  border-radius: 8px;
  font-size: 0.85rem;
  color: var(--text-strong);
  outline: none;
  box-sizing: border-box;
}

.modal__input:focus {
  border-color: var(--primary-500);
}

/* ===== 删除确认弹窗：与后台其他管理页样式保持一致 ===== */
.modal--delete {
  width: 380px;
}

.delete-confirm-text {
  margin: 8px 0;
  font-size: 0.88rem;
  color: #333;
  line-height: 1.6;
}

.btn--danger {
  background: #ff4d4f;
  color: #fff;
  border-color: #ff4d4f;
}

.btn--danger:hover {
  background: #ff7875;
  border-color: #ff7875;
}

.btn--primary:disabled {
  background: var(--primary-400);
  box-shadow: none;
  cursor: not-allowed;
  transform: none;
}

.modal__input:disabled {
  background: rgba(148, 163, 184, 0.12);
  color: #888;
  cursor: not-allowed;
}

.modal__footer {
  flex-shrink: 0;
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding: 12px 24px;
  border-top: 1px solid var(--border);
}
</style>
