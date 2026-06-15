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
        <!-- 用户管理 -->
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
                <th>角色</th>
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
                <td>
                  <span class="role-badge" :class="u.role_id === 1 ? 'role-badge--admin' : 'role-badge--user'">
                    {{ u.role_id === 1 ? '管理员' : '普通用户' }}
                  </span>
                </td>
                <td>
                  <span class="status-badge" :class="u.status === 1 ? 'status-badge--normal' : 'status-badge--banned'">
                    {{ u.status === 1 ? '正常' : '封禁' }}
                  </span>
                </td>
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

        <!-- 文章管理 -->
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
                <th>浏览量</th>
                <th>点赞数</th>
                <th>状态</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="a in adminArticles" :key="a.id">
                <td>{{ a.id }}</td>
                <td>{{ a.title }}</td>
                <td>{{ a.views }}</td>
                <td>{{ a.likes }}</td>
                <td>
                  <span class="status-badge status-badge--normal">已发布</span>
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

        <!-- 分类管理 -->
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

        <!-- 标签管理 -->
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

    <!-- 编辑/新增模态框 -->
    <div v-if="modalVisible" class="modal-overlay" @click.self="modalVisible = false">
      <div class="modal">
        <h3 class="modal__title">{{ modalTitle }}</h3>
        <div class="modal__body">
          <template v-if="modalType === 'user'">
            <div class="modal__field">
              <label class="modal__label">昵称</label>
              <input v-model="modalForm.user_name" class="modal__input" placeholder="请输入昵称" />
            </div>
            <div class="modal__field">
              <label class="modal__label">角色</label>
              <select v-model.number="modalForm.role_id" class="modal__input">
                <option :value="0">普通用户</option>
                <option :value="1">管理员</option>
              </select>
            </div>
            <div class="modal__field">
              <label class="modal__label">状态</label>
              <select v-model.number="modalForm.status" class="modal__input">
                <option :value="1">正常</option>
                <option :value="0">封禁</option>
              </select>
            </div>
          </template>
          <template v-else-if="modalType === 'article'">
            <div class="modal__field">
              <label class="modal__label">标题</label>
              <input v-model="modalForm.title" class="modal__input" placeholder="请输入标题" />
            </div>
            <div class="modal__field">
              <label class="modal__label">分类</label>
              <select v-model.number="modalForm.type_id" class="modal__input">
                <option v-for="c in adminCategories" :key="c.id" :value="c.id">{{ c.name }}</option>
              </select>
            </div>
            <div class="modal__field">
              <label class="modal__label">摘要</label>
              <input v-model="modalForm.summary" class="modal__input" placeholder="请输入摘要" />
            </div>
            <div class="modal__field">
              <label class="modal__label">内容</label>
              <textarea v-model="modalForm.content" class="modal__input" rows="4" placeholder="请输入文章内容" style="min-height:88px;resize:vertical;"></textarea>
            </div>
          </template>
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
          <button class="btn btn--cancel" @click="modalVisible = false">取消</button>
          <button class="btn btn--primary" @click="handleSave">保存</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import api from '../api'
import NavBar from '../components/NavBar.vue'

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
  views: number
  likes: number
  type_id: number
  summary: string
  content: string
}

interface CatTagItem {
  id: number
  name: string
  status: number
  created_at: string
}

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

const modalVisible = ref(false)
const modalType = ref<'user' | 'article' | 'category' | 'tag'>('user')
const editingId = ref<number | null>(null)
const modalForm = reactive<Record<string, any>>({
  user_name: '',
  role_id: 0,
  status: 1,
  title: '',
  type_id: 0,
  summary: '',
  content: '',
  name: ''
})

const modalTitle = ref('')

const fmt = (d: string) => {
  if (!d) return ''
  return new Date(d).toLocaleDateString('zh-CN')
}

const openModal = (type: 'user' | 'article' | 'category' | 'tag', item?: any) => {
  modalType.value = type
  modalVisible.value = true
  editingId.value = item?.id || null

  // reset
  modalForm.user_name = ''
  modalForm.role_id = 0
  modalForm.status = 1
  modalForm.title = ''
  modalForm.type_id = 0
  modalForm.summary = ''
  modalForm.content = ''
  modalForm.name = ''

  if (item) {
    modalTitle.value = type === 'user' ? '编辑用户' : type === 'article' ? '编辑文章' : type === 'category' ? '编辑分类' : '编辑标签'
    if (type === 'user') {
      modalForm.user_name = item.user_name || ''
      modalForm.role_id = item.role_id ?? 0
      modalForm.status = item.status ?? 1
    } else if (type === 'article') {
      modalForm.title = item.title || ''
      modalForm.type_id = item.type_id || 0
      modalForm.summary = item.summary || ''
      modalForm.content = item.content || ''
    } else {
      modalForm.name = item.name || ''
      modalForm.status = item.status ?? 1
    }
  } else {
    modalTitle.value = type === 'user' ? '新增用户' : type === 'article' ? '新增文章' : type === 'category' ? '新增分类' : '新增标签'
  }
}

const handleSave = () => {
  // mock: 直接关闭
  modalVisible.value = false
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
      { id: 1, title: 'Go 语言并发编程深入解析', views: 2340, likes: 186, type_id: 1, summary: '深入理解 goroutine', content: '...' },
      { id: 2, title: 'Vue 3 Composition API 实战指南', views: 1850, likes: 142, type_id: 2, summary: 'setup 到响应式', content: '...' },
      { id: 3, title: '构建高性能 RESTful API', views: 3120, likes: 257, type_id: 3, summary: '路由设计', content: '...' }
    ]
    adminCategories.value = [
      { id: 1, name: 'Go语言', status: 1, created_at: '2026-04-01T08:00:00Z' },
      { id: 2, name: '前端', status: 1, created_at: '2026-04-02T08:00:00Z' },
      { id: 3, name: '后端', status: 0, created_at: '2026-04-03T08:00:00Z' }
    ]
    adminTags.value = [
      { id: 1, name: '并发', status: 1, created_at: '2026-04-01T08:00:00Z' },
      { id: 2, name: 'Docker', status: 1, created_at: '2026-04-02T08:00:00Z' }
    ]
  }
})
</script>
