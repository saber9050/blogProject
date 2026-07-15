<template>
  <div class="article-editor-page">
    <NavBar />
    <div class="editor-layout">
      <div class="editor-container">
        <!-- 顶部导航 -->
        <div class="editor-topbar">
          <button class="btn-back" @click="goBack">← 返回列表</button>
          <h2 class="editor-title">{{ isEdit ? '编辑文章' : '新增文章' }}</h2>
          <div class="editor-topbar__right">
            <button class="btn btn--primary" :disabled="saving || !canSave" @click="handleSave">
              {{ saving ? '保存中...' : '保存' }}
            </button>
          </div>
        </div>

        <!-- 表单区域 -->
        <div class="editor-form">
          <div class="editor-form__main">
            <!-- 标题 -->
            <div class="form-group">
              <label class="form-label"><span class="required">*</span>标题</label>
              <input v-model="form.title" class="form-input" :class="{ 'input-error': !form.title?.trim() && submitted }" placeholder="请输入文章标题" />
            </div>

            <!-- 内容 -->
            <div class="form-group">
              <label class="form-label"><span class="required">*</span>内容</label>
              <div ref="toolbarContainer" class="toolbar-container"></div>
              <div ref="editorContainer" class="editor-content" :class="{ 'input-error': contentEmpty && submitted }"></div>
            </div>
          </div>

          <div class="editor-form__side">
            <!-- 分类 -->
            <div class="form-group">
              <label class="form-label"><span class="required">*</span>分类</label>
              <select v-model.number="form.type_id" class="form-input" :class="{ 'input-error': !form.type_id && submitted }">
                <option :value="0" disabled>请选择分类</option>
                <option v-for="c in categories" :key="c.id" :value="c.id">{{ c.name }}</option>
              </select>
            </div>

            <!-- 标签 -->
            <div class="form-group">
              <label class="form-label">标签</label>
              <div class="tag-selector">
                <span
                  v-for="t in tags"
                  :key="t.id"
                  class="tag-option"
                  :class="{ 'tag-option--selected': form.tag_ids.includes(t.id) }"
                  @click="toggleTag(t.id)"
                >
                  {{ t.name }}
                </span>
                <span v-if="!tags.length" class="tag-empty">暂无可用标签</span>
              </div>
            </div>

            <!-- 封面 -->
            <div class="form-group">
              <label class="form-label">封面图片</label>
              <div class="cover-upload">
                <input ref="coverInputRef" type="file" accept="image/*" style="display:none" @change="onCoverChange" />
                <button class="btn-sm" type="button" @click="selectCover">
                  {{ form.cover_url ? '更换封面' : '选择封面' }}
                </button>
                <img v-if="coverPreviewUrl || form.cover_url" :src="coverPreviewUrl || form.cover_url" class="cover-preview" />
              </div>
            </div>

            <!-- 摘要 -->
            <div class="form-group">
              <label class="form-label">摘要</label>
              <textarea v-model="form.summary" class="form-input form-textarea" rows="3" placeholder="请输入文章摘要"></textarea>
            </div>

            <!-- 状态 -->
            <div class="form-group">
              <label class="form-label">状态</label>
              <select v-model.number="form.status" class="form-input">
                <option :value="1">已发布</option>
                <option :value="0">草稿箱</option>
              </select>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import api from '../api'
import NavBar from '../components/NavBar.vue'
import { showToast } from '../utils/toast'
import type { IEditorConfig } from '@wangeditor/editor'

const route = useRoute()
const router = useRouter()

const isEdit = computed(() => route.name === 'article-edit')
const articleId = computed(() => (route.params.id ? Number(route.params.id) : null))

// ---------- 表单数据 ----------
const form = reactive({
  title: '',
  type_id: 0,
  tag_ids: [] as number[],
  cover_url: '',
  summary: '',
  content: '',
  status: 0
})

const originalData = ref<Record<string, any> | null>(null)
const submitted = ref(false)
const saving = ref(false)

// ---------- 分类 & 标签 ----------
interface CatTagItem {
  id: number
  name: string
  status: number
}
const categories = ref<CatTagItem[]>([])
const tags = ref<CatTagItem[]>([])

// ---------- 编辑器 ----------
const toolbarContainer = ref<HTMLDivElement | null>(null)
const editorContainer = ref<HTMLDivElement | null>(null)
let editorInstance: any = null

// ---------- 封面 ----------
const coverInputRef = ref<HTMLInputElement | null>(null)
const coverPreviewUrl = ref('')

// ---------- 计算属性 ----------
const contentEmpty = computed(() => {
  const c = form.content
  return !c || c === '<p><br></p>' || c === '<p></p>'
})

const canSave = computed(() => {
  return form.title?.trim() && form.type_id && !contentEmpty.value
})

// ---------- 方法 ----------
const toggleTag = (id: number) => {
  const idx = form.tag_ids.indexOf(id)
  if (idx >= 0) {
    form.tag_ids.splice(idx, 1)
  } else {
    form.tag_ids.push(id)
  }
}

const selectCover = () => {
  coverInputRef.value?.click()
}

const onCoverChange = async (e: Event) => {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  coverPreviewUrl.value = URL.createObjectURL(file)
  try {
    const fd = new FormData()
    fd.append('file', file)
    const res = await api.post('/admin/upload', fd, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
    form.cover_url = res.data?.data?.url || res.data?.url || ''
    coverPreviewUrl.value = form.cover_url
  } catch (err: any) {
    console.error('封面上传失败:', err)
    showToast(err?.response?.data?.message || '封面上传失败', 'error')
    coverPreviewUrl.value = ''
    if (coverInputRef.value) coverInputRef.value.value = ''
  }
}

const loadArticleDetail = async (id: number): Promise<Record<string, any> | null> => {
  try {
    const res = await api.get(`/articles/${id}`)
    return res.data?.data || null
  } catch (err) {
    console.error('获取文章详情失败:', err)
    return null
  }
}

const initEditor = async (content: string) => {
  if (!toolbarContainer.value || !editorContainer.value) return
  const wangEditor = await import('@wangeditor/editor')

  const uploadImage = (editor: any, file: File) => {
    const fd = new FormData()
    fd.append('file', file)
    api.post('/admin/upload', fd, {
      headers: { 'Content-Type': 'multipart/form-data' }
    }).then(res => {
      const url = res.data?.data?.url || res.data?.url || ''
      if (url) {
        editor.restoreSelection()
        editor.dangerouslyInsertHtml(`<img src="${url}" alt="" />`)
      }
    }).catch(err => {
      console.error('图片上传失败:', err)
    })
  }

  const editorConfig: Partial<IEditorConfig> = {
    placeholder: '请输入文章内容...',
    onChange: (editor: any) => {
      form.content = editor.getHtml()
    },
    customPaste: (editor: any, event: ClipboardEvent) => {
      const cd = event.clipboardData
      if (!cd) return true

      if (cd.items) {
        for (let i = 0; i < cd.items.length; i++) {
          const item = cd.items[i]
          if (item.type.startsWith('image/')) {
            const file = item.getAsFile()
            if (file) {
              event.preventDefault()
              uploadImage(editor, file)
              return false
            }
          }
        }
      }

      if (cd.files && cd.files.length > 0) {
        for (let i = 0; i < cd.files.length; i++) {
          const file = cd.files[i]
          if (file.type.startsWith('image/')) {
            event.preventDefault()
            uploadImage(editor, file)
            return false
          }
        }
      }

      const text = cd.getData('text/plain')
      if (text) {
        const trimmed = text.trim()
        if (/^https?:\/\/.+\.(jpe?g|png|gif|webp|svg|bmp|ico)(\?.*)?$/i.test(trimmed)) {
          event.preventDefault()
          editor.restoreSelection()
          editor.dangerouslyInsertHtml(`<img src="${trimmed}" alt="" />`)
          return false
        }
      }

      return true
    },
    MENU_CONF: {
      uploadImage: {
        allowedFileTypes: ['image/jpeg', 'image/png', 'image/gif', 'image/webp'],
        customUpload: (file: File, insertFn: (src: string, alt: string, href: string) => void) => {
          const fd = new FormData()
          fd.append('file', file)
          api.post('/admin/upload', fd, {
            headers: { 'Content-Type': 'multipart/form-data' }
          }).then(res => {
            const url = res.data?.data?.url || res.data?.url || ''
            if (url) insertFn(url, '', '')
          }).catch(err => {
            console.error('图片上传失败:', err)
          })
        }
      }
    }
  }

  if (editorInstance) {
    editorInstance.destroy()
    editorInstance = null
  }
  const editor = wangEditor.createEditor({
    selector: editorContainer.value,
    config: editorConfig,
    html: content || undefined,
    mode: 'default'
  })
  wangEditor.createToolbar({
    editor,
    selector: toolbarContainer.value,
    config: {
      excludeKeys: ['group-video', 'insertImage']
    }
  })
  editorInstance = editor
}

const destroyEditor = () => {
  if (editorInstance) {
    try { editorInstance.destroy() } catch {}
    editorInstance = null
  }
}

const goBack = () => {
  destroyEditor()
  router.push('/admin')
}

const handleSave = async () => {
  submitted.value = true
  if (!canSave.value) return
  saving.value = true

  try {
    const cleanContent = form.content === '<p><br></p>' || form.content === '<p></p>' ? '' : form.content

    if (isEdit.value && articleId.value) {
      let updateData: Record<string, any> = {}
      const orig = originalData.value

      if (orig) {
        if (form.title !== orig.title) updateData.title = form.title
        if (form.type_id !== orig.type_id) updateData.type_id = form.type_id
        if (JSON.stringify(form.tag_ids.slice().sort()) !== JSON.stringify((orig.tag_ids || []).slice().sort())) {
          updateData.tag_ids = form.tag_ids
        }
        if (form.cover_url !== orig.cover_url) updateData.cover_url = form.cover_url
        if (form.summary !== orig.summary) updateData.summary = form.summary
        if (cleanContent !== orig.content) updateData.content = cleanContent
        updateData.status = form.status

        if (Object.keys(updateData).length === 0 || (Object.keys(updateData).length === 1 && 'status' in updateData && updateData.status === orig.status)) {
          showToast('未做任何修改', 'info')
          goBack()
          return
        }
      } else {
        updateData = {
          title: form.title,
          type_id: form.type_id,
          tag_ids: form.tag_ids,
          cover_url: form.cover_url,
          summary: form.summary,
          content: cleanContent,
          status: form.status
        }
      }

      await api.put(`/admin/articles/${articleId.value}`, updateData)
    } else {
      await api.post('/admin/articles', {
        title: form.title,
        type_id: form.type_id,
        tag_ids: form.tag_ids,
        cover_url: form.cover_url,
        summary: form.summary,
        content: cleanContent,
        status: form.status
      })
    }

    showToast('保存成功', 'success')
    goBack()
  } catch (error: any) {
    console.error('保存失败:', error)
    showToast(error?.response?.data?.message || '保存失败，请重试', 'error')
  } finally {
    saving.value = false
  }
}

// ---------- 生命周期 ----------
onMounted(async () => {
  // 加载分类和标签
  try {
    const [catRes, tagRes] = await Promise.all([
      api.get('/admin/categories', { params: { page: 1, page_size: 100 } }),
      api.get('/admin/tags', { params: { page: 1, page_size: 100 } })
    ])
    const allCats: CatTagItem[] = catRes.data.data?.list || []
    const allTags: CatTagItem[] = tagRes.data.data?.list || []
    categories.value = allCats.filter(c => c.status === 1)
    tags.value = allTags.filter(t => t.status === 1)
  } catch (e) {
    console.error('加载分类/标签失败:', e)
  }

  // 编辑模式：加载文章详情
  if (isEdit.value && articleId.value) {
    const data = await loadArticleDetail(articleId.value)
    if (data) {
      form.title = data.title || ''
      form.type_id = data.category?.id || data.type_id || 0
      form.summary = data.summary || ''
      form.content = data.content || ''
      form.cover_url = data.cover_url || ''
      coverPreviewUrl.value = data.cover_url || ''
      form.status = data.status ?? 0
      const tagList = data.tags || []
      form.tag_ids = tagList.map((t: any) => t.id)

      originalData.value = {
        title: form.title,
        type_id: form.type_id,
        summary: form.summary,
        content: form.content,
        cover_url: form.cover_url,
        status: form.status,
        tag_ids: form.tag_ids.slice()
      }
    }
  }

  await nextTick()
  initEditor(form.content || '')
})

onBeforeUnmount(() => {
  destroyEditor()
})
</script>

<style scoped>
.article-editor-page {
  min-height: 100vh;
  background: #f5f7fa;
}

.editor-layout {
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px;
}

.editor-container {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 1px 4px rgba(0,0,0,0.06);
}

/* ---------- 顶部栏 ---------- */
.editor-topbar {
  display: flex;
  align-items: center;
  padding: 16px 24px;
  border-bottom: 1px solid #f0f0f0;
  gap: 16px;
}

.btn-back {
  padding: 6px 14px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  background: #fff;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
}

.btn-back:hover {
  border-color: #1677ff;
  color: #1677ff;
}

.editor-title {
  flex: 1;
  font-size: 18px;
  font-weight: 600;
  margin: 0;
}

.editor-topbar__right {
  flex-shrink: 0;
}

/* ---------- 表单 ---------- */
.editor-form {
  display: flex;
  gap: 24px;
  padding: 24px;
}

.editor-form__main {
  flex: 1;
  min-width: 0;
}

.editor-form__side {
  width: 320px;
  flex-shrink: 0;
}

.form-group {
  margin-bottom: 20px;
}

.form-label {
  display: block;
  margin-bottom: 6px;
  font-size: 13px;
  font-weight: 500;
  color: #333;
}

.required {
  color: #ff4d4f;
  margin-right: 2px;
}

.form-input {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  font-size: 13px;
  transition: border-color 0.2s;
  box-sizing: border-box;
  background: #fff;
}

.form-input:focus {
  border-color: #1677ff;
  outline: none;
  box-shadow: 0 0 0 2px rgba(22,119,255,0.1);
}

select.form-input {
  appearance: auto;
}

.form-textarea {
  resize: vertical;
  min-height: 70px;
  line-height: 1.5;
}

.input-error {
  border-color: #ff4d4f !important;
}

/* ---------- 标签选择器 ---------- */
.tag-selector {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.tag-option {
  display: inline-block;
  padding: 4px 10px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
  user-select: none;
}

.tag-option:hover {
  border-color: #1677ff;
  color: #1677ff;
}

.tag-option--selected {
  background: #1677ff;
  color: #fff;
  border-color: #1677ff;
}

.tag-empty {
  font-size: 12px;
  color: #999;
}

/* ---------- 封面 ---------- */
.cover-upload {
  display: flex;
  align-items: center;
  gap: 10px;
}

.cover-preview {
  width: 80px;
  height: 52px;
  object-fit: cover;
  border-radius: 4px;
  border: 1px solid #f0f0f0;
}

.btn-sm {
  padding: 4px 10px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  background: #fff;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-sm:hover {
  border-color: #1677ff;
  color: #1677ff;
}

/* ---------- 编辑器 ---------- */
.toolbar-container {
  position: sticky;
  top: 60px;          /* NavBar 高度，吸顶时保持在导航下方 */
  z-index: 10;
  background: #fff;
  border: 1px solid #d9d9d9;
  border-bottom: none;
  border-radius: 4px 4px 0 0;
}

.editor-content {
  border: 1px solid #d9d9d9;
  border-radius: 0 0 4px 4px;
  min-height: 400px;
}

.editor-content :deep(.w-e-text-container) {
  min-height: 360px;
}

/* ---------- 按钮 ---------- */
.btn {
  padding: 8px 20px;
  border-radius: 4px;
  font-size: 13px;
  cursor: pointer;
  border: 1px solid #d9d9d9;
  transition: all 0.2s;
}

.btn--primary {
  background: #1677ff;
  color: #fff;
  border-color: #1677ff;
}

.btn--primary:hover {
  background: #4096ff;
}

.btn--primary:disabled {
  background: #a0c4ff;
  border-color: #a0c4ff;
  cursor: not-allowed;
}

/* ---------- 响应式 ---------- */
@media (max-width: 900px) {
  .editor-form {
    flex-direction: column;
  }
  .editor-form__side {
    width: 100%;
  }
  .editor-layout {
    padding: 12px;
  }
  .editor-form {
    padding: 16px;
  }
}
</style>
