<template>
  <div class="admin-page">
    <NavBar />
    <div class="admin-layout">
      <!-- 侧边栏 -->
      <aside class="admin-sidebar">
        <div class="admin-sidebar__title">后台管理</div>
        <button
          v-for="tab in tabs"
          :key="tab.key"
          class="admin-sidebar__link"
          :class="{ 'admin-sidebar__link--active': activeTab === tab.key }"
          @click="activeTab = tab.key"
        >
          <span>{{ tab.icon }}</span>
          <span>{{ tab.label }}</span>
        </button>
      </aside>

      <!-- 主内容区 -->
      <main class="admin-main">
        <!-- ========== 用户管理 ========== -->
        <section v-if="activeTab === 'users'" class="admin-panel">
          <div class="admin-panel__header">
            <h2 class="admin-panel__title">用户管理</h2>
            <div class="admin-panel__actions">
              <button class="btn-sm btn-sm--primary" @click="openModal('user')">+ 新增</button>
            </div>
          </div>
          <table class="admin-table">
            <thead>
              <tr>
                <th>ID</th>
                <th>昵称</th>
                <th>头像</th>
                <th>状态</th>
                <th>注册时间</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="u in users" :key="u.id">
                <td>{{ u.id }}</td>
                <td>{{ u.user_name }}</td>
                <td>
                  <img v-if="u.avatar_url" :src="u.avatar_url" class="table-avatar" />
                  <span v-else class="table-avatar table-avatar--placeholder">&#128100;</span>
                </td>
                <td>{{ u.status === 1 ? '正常' : '封禁' }}</td>
                <td>{{ fmt(u.created_at) }}</td>
                <td>
                  <div class="table-actions">
                    <button class="btn-sm" @click="openModal('user', u)">编辑</button>
                    <button class="btn-sm btn-sm--danger" @click="handleDelete('user', u.id)">删除</button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
          <div v-if="!users.length" class="admin-empty">暂无用户数据</div>
        </section>

        <!-- ========== 文章管理 ========== -->
        <section v-if="activeTab === 'articles'" class="admin-panel">
          <div class="admin-panel__header">
            <h2 class="admin-panel__title">文章管理</h2>
            <div class="admin-panel__actions">
              <button class="btn-sm btn-sm--primary" @click="openModal('article')">+ 新增</button>
            </div>
          </div>
          <table class="admin-table">
            <thead>
              <tr>
                <th>ID</th>
                <th>文章标题</th>
                <th>分类</th>
                <th>标签</th>
                <th>状态</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="a in adminArticles" :key="a.id">
                <td>{{ a.id }}</td>
                <td>{{ a.title }}</td>
                <td>
                  <span class="cat-tag-badge">{{ a.category_name || '-' }}</span>
                </td>
                <td>
                  <template v-if="a.tags && a.tags.length">
                    <span v-for="t in a.tags" :key="t.id" class="cat-tag-badge cat-tag-badge--tag">{{ t.name }}</span>
                  </template>
                  <span v-else>-</span>
                </td>
                <td>
                  <span class="status-badge" :class="a.status === 1 ? 'status-badge--normal' : 'status-badge--draft'">
                    {{ a.status === 1 ? '已发布' : '草稿' }}
                  </span>
                </td>
                <td>
                  <div class="table-actions">
                    <button class="btn-sm" @click="openModal('article', a)">编辑</button>
                    <button class="btn-sm btn-sm--danger" @click="handleDelete('article', a.id)">删除</button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
          <div v-if="!adminArticles.length" class="admin-empty">暂无文章数据</div>
        </section>

        <!-- ========== 分类管理 ========== -->
        <section v-if="activeTab === 'categories'" class="admin-panel">
          <div class="admin-panel__header">
            <h2 class="admin-panel__title">分类管理</h2>
            <div class="admin-panel__actions">
              <button class="btn-sm btn-sm--primary" @click="openModal('category')">+ 新增</button>
            </div>
          </div>
          <table class="admin-table">
            <thead>
              <tr>
                <th>ID</th>
                <th>名称</th>
                <th>状态</th>
                <th>创建时间</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="c in adminCategories" :key="c.id">
                <td>{{ c.id }}</td>
                <td>{{ c.name }}</td>
                <td>
                  <span class="status-badge" :class="c.status === 1 ? 'status-badge--normal' : 'status-badge--banned'">
                    {{ c.status === 1 ? '启用' : '禁用' }}
                  </span>
                </td>
                <td>{{ fmt(c.created_at) }}</td>
                <td>
                  <div class="table-actions">
                    <button class="btn-sm" @click="openModal('category', c)">编辑</button>
                    <button class="btn-sm btn-sm--danger" @click="handleDelete('category', c.id)">删除</button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
          <div v-if="!adminCategories.length" class="admin-empty">暂无分类数据</div>
        </section>

        <!-- ========== 标签管理 ========== -->
        <section v-if="activeTab === 'tags'" class="admin-panel">
          <div class="admin-panel__header">
            <h2 class="admin-panel__title">标签管理</h2>
            <div class="admin-panel__actions">
              <button class="btn-sm btn-sm--primary" @click="openModal('tag')">+ 新增</button>
            </div>
          </div>
          <table class="admin-table">
            <thead>
              <tr>
                <th>ID</th>
                <th>名称</th>
                <th>状态</th>
                <th>创建时间</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="t in adminTags" :key="t.id">
                <td>{{ t.id }}</td>
                <td>{{ t.name }}</td>
                <td>
                  <span class="status-badge" :class="t.status === 1 ? 'status-badge--normal' : 'status-badge--banned'">
                    {{ t.status === 1 ? '启用' : '禁用' }}
                  </span>
                </td>
                <td>{{ fmt(t.created_at) }}</td>
                <td>
                  <div class="table-actions">
                    <button class="btn-sm" @click="openModal('tag', t)">编辑</button>
                    <button class="btn-sm btn-sm--danger" @click="handleDelete('tag', t.id)">删除</button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
          <div v-if="!adminTags.length" class="admin-empty">暂无标签数据</div>
        </section>
      </main>
    </div>

    <!-- ========== 编辑/新增模态框 ========== -->
    <div v-if="modalVisible" class="modal-overlay" @click.self="modalVisible = false">
      <div class="modal" :class="{ 'modal--wide': modalType === 'article' }">
        <h3 class="modal__title">{{ modalTitle }}</h3>
        <div class="modal__body">
          <!-- ---- 用户 ---- -->
          <template v-if="modalType === 'user'">
            <!-- 新增：显示昵称、账号、密码、状态 -->
            <template v-if="!editingId">
              <div class="modal__field">
                <label class="modal__label">昵称</label>
                <div class="modal__input-wrapper" :class="{ 'modal__input-wrapper--error': nameError, 'modal__input-wrapper--ok': !nameError && nameChecked }">
                  <input v-model="modalForm.user_name" class="modal__input" :class="{ 'modal__input--error': nameError }" placeholder="请输入昵称" @input="checkName" />
                  <span v-if="nameCheckLoading" class="modal__input-suffix">检查中...</span>
                  <span v-else-if="nameError" class="modal__input-suffix modal__input-suffix--error">昵称已存在</span>
                  <span v-else-if="nameChecked" class="modal__input-suffix modal__input-suffix--ok">可用</span>
                </div>
              </div>
              <div class="modal__field">
                <label class="modal__label">账号</label>
                <div class="modal__input-wrapper" :class="{ 'modal__input-wrapper--error': accountError, 'modal__input-wrapper--ok': !accountError && accountChecked }">
                  <input v-model="modalForm.account" class="modal__input" :class="{ 'modal__input--error': accountError }" placeholder="请输入账号" @input="checkAccount" />
                  <span v-if="accountCheckLoading" class="modal__input-suffix">检查中...</span>
                  <span v-else-if="accountError" class="modal__input-suffix modal__input-suffix--error">账号已存在</span>
                  <span v-else-if="accountChecked" class="modal__input-suffix modal__input-suffix--ok">可用</span>
                </div>
              </div>
              <div class="modal__field">
                <label class="modal__label">密码</label>
                <input v-model="modalForm.password" class="modal__input" type="password" placeholder="请输入密码" />
              </div>
              <div class="modal__field">
                <label class="modal__label">状态</label>
                <select v-model.number="modalForm.status" class="modal__input">
                  <option :value="1">正常</option>
                  <option :value="0">封禁</option>
                </select>
              </div>
            </template>
            <!-- 编辑：只保留状态 -->
            <template v-else>
              <div class="modal__field">
                <label class="modal__label">状态</label>
                <select v-model.number="modalForm.status" class="modal__input">
                  <option :value="1">正常</option>
                  <option :value="0">封禁</option>
                </select>
              </div>
            </template>
          </template>

          <!-- ---- 文章 ---- -->
          <template v-else-if="modalType === 'article'">
            <div class="modal__field">
              <label class="modal__label">标题</label>
              <input v-model="modalForm.title" class="modal__input" placeholder="请输入文章标题" />
            </div>
            <div class="modal__field">
              <label class="modal__label">分类</label>
              <select v-model.number="modalForm.type_id" class="modal__input">
                <option :value="0" disabled>请选择分类</option>
                <option v-for="c in adminCategories" :key="c.id" :value="c.id">{{ c.name }}</option>
              </select>
            </div>
            <div class="modal__field">
              <label class="modal__label">标签</label>
              <div class="tag-selector">
                <span
                  v-for="t in adminTags"
                  :key="t.id"
                  class="tag-option"
                  :class="{ 'tag-option--selected': selectedTags.includes(t.id) }"
                  @click="toggleTag(t.id)"
                >
                  {{ t.name }}
                </span>
              </div>
            </div>
            <div class="modal__field">
              <label class="modal__label">封面图片</label>
              <div class="cover-upload">
                <input ref="coverInputRef" type="file" accept="image/*" style="display:none" @change="onCoverChange" />
                <button class="btn-sm" @click="selectCover">
                  {{ modalForm.cover_url ? '更换封面' : '选择封面' }}
                </button>
                <img v-if="modalForm.cover_url" :src="modalForm.cover_url" class="cover-preview" />
              </div>
            </div>
            <div class="modal__field">
              <label class="modal__label">摘要</label>
              <textarea v-model="modalForm.summary" class="modal__input" rows="2" placeholder="请输入文章摘要" style="resize:vertical;min-height:44px;"></textarea>
            </div>
            <div class="modal__field">
              <label class="modal__label">状态</label>
              <select v-model.number="modalForm.status" class="modal__input">
                <option :value="1">已发布</option>
                <option :value="0">草稿箱</option>
              </select>
            </div>
            <div class="modal__field">
              <label class="modal__label">内容</label>
              <div ref="editorContainer" class="editor-container"></div>
            </div>
          </template>

          <!-- ---- 分类/标签 ---- -->
          <template v-else>
            <div class="modal__field">
              <label class="modal__label">名称</label>
              <input v-model="modalForm.name" class="modal__input" placeholder="请输入名称" />
            </div>
            <div class="modal__field">
              <label class="modal__label">状态</label>
              <select v-model.number="modalForm.status" class="modal__input">
                <option :value="1">启用</option>
                <option :value="0">禁用</option>
              </select>
            </div>
          </template>
        </div>
        <div class="modal__footer">
          <button class="btn btn--cancel" @click="closeModal">取消</button>
          <button class="btn btn--primary" :disabled="canSaveDisabled" @click="handleSave">保存</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, nextTick } from 'vue'
import api from '../api'
import NavBar from '../components/NavBar.vue'

// ---------- 类型 ----------
interface UserItem {
  id: number
  user_name: string
  avatar_url: string
  role_id: number
  status: number
  created_at: string
}

interface ArticleItem {
  id: number
  title: string
  type_id: number
  category_name?: string
  tags?: { id: number; name: string }[]
  summary: string
  content: string
  cover_url: string
  status: number
  views?: number
  likes?: number
}

interface CatTagItem {
  id: number
  name: string
  status: number
  created_at: string
}

// ---------- 状态 ----------
const activeTab = ref('users')
const tabs = [
  { key: 'users', label: '用户管理', icon: '👥' },
  { key: 'articles', label: '文章管理', icon: '📝' },
  { key: 'categories', label: '分类管理', icon: '📁' },
  { key: 'tags', label: '标签管理', icon: '🏷️' }
]

const users = ref<UserItem[]>([])
const adminArticles = ref<ArticleItem[]>([])
const adminCategories = ref<CatTagItem[]>([])
const adminTags = ref<CatTagItem[]>([])

// 模态框
const modalVisible = ref(false)
const modalType = ref<'user' | 'article' | 'category' | 'tag'>('user')
const editingId = ref<number | null>(null)
const editingUser = ref<UserItem | null>(null)
const modalTitle = ref('')
const modalForm = reactive<Record<string, any>>({
  user_name: '',
  account: '',
  password: '',
  status: 1,
  title: '',
  type_id: 0,
  summary: '',
  content: '',
  cover_url: '',
  name: ''
})

// 文章标签多选
const selectedTags = ref<number[]>([])

// 编辑器
const editorContainer = ref<HTMLDivElement | null>(null)
let editorInstance: any = null

// 封面文件上传
const coverInputRef = ref<HTMLInputElement | null>(null)

// 唯一性校验状态
const nameCheckLoading = ref(false)
const accountCheckLoading = ref(false)
const nameError = ref(false)
const accountError = ref(false)
const nameChecked = ref(false)
const accountChecked = ref(false)
let nameCheckTimer: ReturnType<typeof setTimeout> | null = null
let accountCheckTimer: ReturnType<typeof setTimeout> | null = null

// 保存按钮是否禁用（新增用户时，昵称或校验失败或未通过唯一性校验时禁用）
const canSaveDisabled = computed(() => {
  if (modalType.value !== 'user' || editingId.value) return false
  return nameError.value || accountError.value || nameCheckLoading.value || accountCheckLoading.value || !nameChecked.value || !accountChecked.value
})

const checkName = () => {
  nameCheckTimer && clearTimeout(nameCheckTimer)
  nameChecked.value = false
  const val = modalForm.user_name?.trim()
  if (!val) {
    nameError.value = false
    nameCheckLoading.value = false
    return
  }
  nameCheckLoading.value = true
  nameCheckTimer = setTimeout(async () => {
    try {
      const res = await api.get('/auth/is_exists_name', { params: { user_name: val } })
      const exists = res.data?.data?.is_exists
      nameError.value = exists === true
      nameChecked.value = true
    } catch {
      nameError.value = false
      nameChecked.value = false
    } finally {
      nameCheckLoading.value = false
    }
  }, 500)
}

const checkAccount = () => {
  accountCheckTimer && clearTimeout(accountCheckTimer)
  accountChecked.value = false
  const val = modalForm.account?.trim()
  if (!val) {
    accountError.value = false
    accountCheckLoading.value = false
    return
  }
  accountCheckLoading.value = true
  accountCheckTimer = setTimeout(async () => {
    try {
      const res = await api.get('/auth/is_exists_account', { params: { account: val } })
      const exists = res.data?.data?.is_exists
      accountError.value = exists === true
      accountChecked.value = true
    } catch {
      accountError.value = false
      accountChecked.value = false
    } finally {
      accountCheckLoading.value = false
    }
  }, 500)
}

// ---------- 方法 ----------
const fmt = (d: string) => {
  if (!d) return ''
  return new Date(d).toLocaleDateString('zh-CN')
}

const toggleTag = (id: number) => {
  const idx = selectedTags.value.indexOf(id)
  if (idx >= 0) {
    selectedTags.value.splice(idx, 1)
  } else {
    selectedTags.value.push(id)
  }
}

const selectCover = () => {
  coverInputRef.value?.click()
}

const onCoverChange = (e: Event) => {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  // 读取本地预览 URL
  modalForm.cover_url = URL.createObjectURL(file)
}

const openModal = async (type: 'user' | 'article' | 'category' | 'tag', item?: any) => {
  modalType.value = type
  modalVisible.value = true
  editingId.value = item?.id || null
  editingUser.value = item || null

  // 重置唯一性校验状态
  nameError.value = false
  accountError.value = false
  nameChecked.value = false
  accountChecked.value = false
  nameCheckLoading.value = false
  accountCheckLoading.value = false

  // reset
  modalForm.user_name = ''
  modalForm.account = ''
  modalForm.password = ''
  modalForm.status = 1
  modalForm.title = ''
  modalForm.type_id = 0
  modalForm.summary = ''
  modalForm.content = ''
  modalForm.cover_url = ''
  modalForm.name = ''
  selectedTags.value = []

  if (item) {
    modalTitle.value = type === 'user' ? '编辑用户' : type === 'article' ? '编辑文章' : type === 'category' ? '编辑分类' : '编辑标签'
    if (type === 'user') {
      modalForm.status = item.status ?? 1
    } else if (type === 'article') {
      modalForm.title = item.title || ''
      modalForm.type_id = item.type_id || 0
      modalForm.summary = item.summary || ''
      modalForm.content = item.content || ''
      modalForm.cover_url = item.cover_url || ''
      modalForm.status = item.status ?? 0
      if (item.tags) {
        selectedTags.value = item.tags.map((t: any) => t.id)
      }
    } else {
      modalForm.name = item.name || ''
      modalForm.status = item.status ?? 1
    }
  } else {
    modalTitle.value = type === 'user' ? '新增用户' : type === 'article' ? '新增文章' : type === 'category' ? '新增分类' : '新增标签'
  }

  // 文章编辑器：等 DOM 渲染后初始化
  if (type === 'article') {
    await nextTick()
    initEditor(item?.content || '')
  }
}

const closeModal = () => {
  modalVisible.value = false
  destroyEditor()
}

const initEditor = async (content: string) => {
  if (!editorContainer.value) return
  // 动态导入 wangEditor
  const wangEditor = await import('@wangeditor/editor')
  const editorConfig: wangEditor.IEditorConfig = {
    placeholder: '请输入文章内容...',
    onChange: (editor: any) => {
      modalForm.content = editor.getHtml()
    }
  }
  // 销毁旧实例
  if (editorInstance) {
    editorInstance.destroy()
    editorInstance = null
  }
  const editor = wangEditor.createEditor({
    selector: editorContainer.value,
    config: editorConfig,
    content: content ? [{ type: 'paragraph', children: [{ text: '' }] }] : undefined,
    html: content || undefined,
    mode: 'default'
  })
  wangEditor.createToolbar({
    editor,
    selector: editorContainer.value
  })
  editorInstance = editor
}

const destroyEditor = () => {
  if (editorInstance) {
    try {
      editorInstance.destroy()
    } catch {}
    editorInstance = null
  }
}

const handleSave = () => {
  // mock: 直接关闭
  modalVisible.value = false
  destroyEditor()
}

const handleDelete = (type: string, id: number) => {
  if (type === 'user') users.value = users.value.filter((u) => u.id !== id)
  else if (type === 'article') adminArticles.value = adminArticles.value.filter((a) => a.id !== id)
  else if (type === 'category') adminCategories.value = adminCategories.value.filter((c) => c.id !== id)
  else adminTags.value = adminTags.value.filter((t) => t.id !== id)
}

onMounted(async () => {
  try {
    const [uRes, aRes, cRes, tRes] = await Promise.all([
      api.get('/admin/users'),
      api.get('/admin/articles'),
      api.get('/admin/categories'),
      api.get('/admin/tags')
    ])
    users.value = uRes.data.data || uRes.data || []
    adminArticles.value = aRes.data.data || aRes.data || []
    adminCategories.value = cRes.data.data || cRes.data || []
    adminTags.value = tRes.data.data || tRes.data || []
  } catch {
    // mock data
    users.value = [
      { id: 1, user_name: '张三', avatar_url: '', role_id: 1, status: 1, created_at: '2026-05-01T08:00:00Z' },
      { id: 2, user_name: '李四', avatar_url: '', role_id: 0, status: 1, created_at: '2026-05-10T10:30:00Z' },
      { id: 3, user_name: '王五', avatar_url: '', role_id: 0, status: 0, created_at: '2026-05-15T14:00:00Z' }
    ]
    adminArticles.value = [
      { id: 1, title: 'Go 语言并发编程深入解析', type_id: 1, category_name: '后端', tags: [{ id: 1, name: 'Go' }], summary: '深入理解 goroutine', content: '<p>...</p>', cover_url: '', status: 1 },
      { id: 2, title: 'Vue 3 Composition API 实战指南', type_id: 2, category_name: '前端', tags: [{ id: 2, name: 'Vue' }], summary: 'setup 到响应式', content: '<p>...</p>', cover_url: '', status: 1 },
      { id: 3, title: '构建高性能 RESTful API', type_id: 1, category_name: '后端', tags: [], summary: '路由设计', content: '<p>...</p>', cover_url: '', status: 0 }
    ]
    adminCategories.value = [
      { id: 1, name: 'Go语言', status: 1, created_at: '2026-04-01T08:00:00Z' },
      { id: 2, name: '前端', status: 1, created_at: '2026-04-02T08:00:00Z' },
      { id: 3, name: '后端', status: 0, created_at: '2026-04-03T08:00:00Z' }
    ]
    adminTags.value = [
      { id: 1, name: 'Go', status: 1, created_at: '2026-04-01T08:00:00Z' },
      { id: 2, name: 'Vue', status: 1, created_at: '2026-04-02T08:00:00Z' },
      { id: 3, name: 'Docker', status: 1, created_at: '2026-04-03T08:00:00Z' }
    ]
  }
})
</script>

<style scoped>
.admin-layout {
  display: flex;
  min-height: calc(100vh - 50px);
}

.admin-sidebar {
  width: 200px;
  background: #f5f7fa;
  border-right: 1px solid #e8e8e8;
  padding: 16px 0;
  flex-shrink: 0;
}

.admin-sidebar__title {
  font-size: 14px;
  font-weight: 600;
  color: #666;
  padding: 0 16px 12px;
  border-bottom: 1px solid #e8e8e8;
  margin-bottom: 8px;
}

.admin-sidebar__link {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 10px 16px;
  border: none;
  background: transparent;
  color: #333;
  font-size: 14px;
  cursor: pointer;
  transition: background 0.2s;
  text-align: left;
}

.admin-sidebar__link:hover { background: #e8f4ff; }
.admin-sidebar__link--active { background: #e8f4ff; color: #1677ff; font-weight: 500; }

.admin-main { flex: 1; padding: 20px; overflow-x: auto; }

.admin-panel__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.admin-panel__title { font-size: 18px; font-weight: 600; }

.admin-table {
  width: 100%;
  border-collapse: collapse;
  background: #fff;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 1px 4px rgba(0,0,0,0.06);
}

.admin-table th,
.admin-table td {
  padding: 12px 14px;
  text-align: left;
  border-bottom: 1px solid #f0f0f0;
  font-size: 13px;
}

.admin-table th {
  background: #fafafa;
  font-weight: 500;
  color: #555;
}

.admin-table tbody tr:hover { background: #fafafa; }

.table-avatar { width: 36px; height: 36px; border-radius: 50%; object-fit: cover; display: inline-block; }
.table-avatar--placeholder { font-size: 24px; line-height: 36px; text-align: center; display: inline-block; width: 36px; }

.role-badge, .status-badge, .cat-tag-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
}

.role-badge--admin { background: #e6f0ff; color: #1677ff; }
.role-badge--user { background: #f0f0f0; color: #666; }

.status-badge--normal { background: #f6ffed; color: #52c41a; }
.status-badge--banned { background: #fff2f0; color: #ff4d4f; }
.status-badge--draft { background: #fffbe6; color: #faad14; }

.cat-tag-badge { background: #f0f5ff; color: #2f54eb; margin-right: 4px; }
.cat-tag-badge--tag { background: #f0f0f0; color: #666; }

.table-actions { display: flex; gap: 6px; }

.btn-sm {
  padding: 4px 10px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  background: #fff;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-sm:hover { border-color: #1677ff; color: #1677ff; }
.btn-sm--primary { background: #1677ff; color: #fff; border-color: #1677ff; }
.btn-sm--primary:hover { background: #4096ff; border-color: #4096ff; color: #fff; }
.btn-sm--danger { color: #ff4d4f; border-color: #ff4d4f; }
.btn-sm--danger:hover { background: #ff4d4f; color: #fff; }

.admin-empty {
  text-align: center;
  padding: 40px;
  color: #999;
  font-size: 14px;
}

/* ---------- 输入框唯一性校验 ---------- */
.modal__input-wrapper {
  display: flex;
  align-items: center;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  transition: border-color 0.2s;
  overflow: hidden;
}

.modal__input-wrapper:focus-within { border-color: #1677ff; box-shadow: 0 0 0 2px rgba(22,119,255,0.1); }
.modal__input-wrapper--error { border-color: #ff4d4f; }
.modal__input-wrapper--error:focus-within { border-color: #ff4d4f; box-shadow: 0 0 0 2px rgba(255,77,79,0.1); }
.modal__input-wrapper--ok { border-color: #52c41a; }
.modal__input-wrapper--ok:focus-within { border-color: #52c41a; box-shadow: 0 0 0 2px rgba(82,196,26,0.1); }

.modal__input-wrapper .modal__input {
  flex: 1;
  border: none;
  box-shadow: none;
}

.modal__input-wrapper .modal__input:focus { box-shadow: none; }

.modal__input-wrapper .modal__input--error { color: #ff4d4f; }

.modal__input-suffix {
  flex-shrink: 0;
  padding: 0 11px;
  font-size: 12px;
  color: #999;
  white-space: nowrap;
}

.modal__input-suffix--error { color: #ff4d4f; }
.modal__input-suffix--ok { color: #52c41a; }

.btn--primary:disabled { background: #a0c4ff; border-color: #a0c4ff; cursor: not-allowed; }

/* ---------- 模态框 ---------- */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.45);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.modal {
  background: #fff;
  border-radius: 8px;
  width: 460px;
  max-height: 80vh;
  overflow-y: auto;
  box-shadow: 0 6px 16px rgba(0,0,0,0.12);
}

.modal--wide { width: 680px; }

.modal__title {
  font-size: 16px;
  font-weight: 600;
  padding: 16px 20px;
  border-bottom: 1px solid #f0f0f0;
}

.modal__body { padding: 16px 20px; }

.modal__field { margin-bottom: 14px; }

.modal__label {
  display: block;
  margin-bottom: 4px;
  font-size: 13px;
  font-weight: 500;
  color: #333;
}

.modal__input {
  width: 100%;
  padding: 7px 11px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  font-size: 13px;
  transition: border-color 0.2s;
  box-sizing: border-box;
}

.modal__input:focus { border-color: #1677ff; outline: none; box-shadow: 0 0 0 2px rgba(22,119,255,0.1); }

select.modal__input { appearance: auto; }

.modal__footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding: 12px 20px;
  border-top: 1px solid #f0f0f0;
}

.btn {
  padding: 8px 20px;
  border-radius: 4px;
  font-size: 13px;
  cursor: pointer;
  border: 1px solid #d9d9d9;
  transition: all 0.2s;
}

.btn--primary { background: #1677ff; color: #fff; border-color: #1677ff; }
.btn--primary:hover { background: #4096ff; }
.btn--cancel { background: #fff; }
.btn--cancel:hover { border-color: #1677ff; color: #1677ff; }

/* ---------- 标签选择器 ---------- */
.tag-selector {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tag-option {
  display: inline-block;
  padding: 4px 12px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
  user-select: none;
}

.tag-option:hover { border-color: #1677ff; color: #1677ff; }
.tag-option--selected { background: #1677ff; color: #fff; border-color: #1677ff; }

/* ---------- 封面 ---------- */
.cover-upload { display: flex; align-items: center; gap: 12px; }

.cover-preview {
  width: 80px;
  height: 52px;
  object-fit: cover;
  border-radius: 4px;
  border: 1px solid #f0f0f0;
}

/* ---------- 编辑器 ---------- */
.editor-container {
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  min-height: 300px;
}

.editor-container :deep(.w-e-toolbar) {
  border-bottom: 1px solid #d9d9d9;
}

.editor-container :deep(.w-e-text-container) {
  min-height: 260px;
}
</style>