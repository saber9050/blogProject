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
          <div v-if="users.length" class="pagination">
            <button class="pagination__btn" :disabled="userPage <= 1" @click="loadUsers(userPage - 1)">上一页</button>
            <template v-for="p in getPageRange(userPage, calcPages(userTotal, userPageSize))" :key="p">
              <span v-if="p === '...'" class="pagination__ellipsis">...</span>
              <button v-else class="pagination__btn" :class="{ 'pagination__btn--active': p === userPage }" @click="loadUsers(p)">{{ p }}</button>
            </template>
            <button class="pagination__btn" :disabled="userPage >= calcPages(userTotal, userPageSize)" @click="loadUsers(userPage + 1)">下一页</button>
            <span class="pagination__info">共 {{ userTotal }} 条</span>
            <select class="pagination__size" v-model.number="userPageSize" @change="loadUsers(1)">
              <option :value="10">10条/页</option>
              <option :value="20">20条/页</option>
              <option :value="50">50条/页</option>
            </select>
          </div>
        </section>

        <!-- ========== 文章管理 ========== -->
        <section v-if="activeTab === 'articles'" class="admin-panel">
          <div class="admin-panel__header">
            <h2 class="admin-panel__title">文章管理</h2>
            <div class="admin-panel__actions">
              <button class="btn-sm" @click="transferModalVisible = true">一键转移</button>
              <button class="btn-sm btn-sm--primary" @click="router.push('/admin/article/new')">+ 新增</button>
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
                  <span class="cat-tag-badge">{{ a.category?.name || '-' }}</span>
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
                    <button class="btn-sm" @click="router.push('/admin/article/' + a.id + '/edit')">编辑</button>
                    <button class="btn-sm btn-sm--danger" @click="handleDelete('article', a.id)">删除</button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
          <div v-if="!adminArticles.length" class="admin-empty">暂无文章数据</div>
          <div v-if="adminArticles.length" class="pagination">
            <button class="pagination__btn" :disabled="articlePage <= 1" @click="loadArticles(articlePage - 1)">上一页</button>
            <template v-for="p in getPageRange(articlePage, calcPages(articleTotal, articlePageSize))" :key="p">
              <span v-if="p === '...'" class="pagination__ellipsis">...</span>
              <button v-else class="pagination__btn" :class="{ 'pagination__btn--active': p === articlePage }" @click="loadArticles(p)">{{ p }}</button>
            </template>
            <button class="pagination__btn" :disabled="articlePage >= calcPages(articleTotal, articlePageSize)" @click="loadArticles(articlePage + 1)">下一页</button>
            <span class="pagination__info">共 {{ articleTotal }} 条</span>
            <select class="pagination__size" v-model.number="articlePageSize" @change="loadArticles(1)">
              <option :value="10">10条/页</option>
              <option :value="20">20条/页</option>
              <option :value="50">50条/页</option>
            </select>
          </div>
        </section>

        <!-- ========== 评论管理 ========== -->
        <section v-if="activeTab === 'comments'" class="admin-panel">
          <div class="admin-panel__header">
            <h2 class="admin-panel__title">评论管理</h2>
            <div class="admin-panel__actions">
              <button v-if="!batchMode" class="btn-sm btn-sm--danger" @click="batchMode = true">批量删除</button>
              <template v-else>
                <button class="btn-sm" @click="exitBatchMode">取消</button>
                <button class="btn-sm btn-sm--danger" :disabled="!selectedCommentIds.length" @click="handleBatchDeleteComments">确认删除 ({{ selectedCommentIds.length }})</button>
              </template>
            </div>
          </div>
          <table class="admin-table">
            <thead>
              <tr>
                <th v-if="batchMode"><input type="checkbox" :checked="allCommentsSelected" @change="toggleAllComments" /></th>
                <th>ID</th>
                <th>所属文章</th>
                <th>评论者</th>
                <th>评论内容</th>
                <th>时间</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="c in adminComments" :key="c.id">
                <td v-if="batchMode"><input type="checkbox" :value="c.id" v-model="selectedCommentIds" /></td>
                <td>{{ c.id }}</td>
                <td>{{ c.article_title }}</td>
                <td>{{ c.user_name }}</td>
                <td>{{ c.content }}</td>
                <td>{{ fmt(c.created_at) }}</td>
                <td>
                  <button class="btn-sm btn-sm--danger" @click="handleDelete('comment', c.id)">删除</button>
                </td>
              </tr>
            </tbody>
          </table>
          <div v-if="!adminComments.length" class="admin-empty">暂无评论数据</div>
          <div v-if="adminComments.length" class="pagination">
            <button class="pagination__btn" :disabled="commentPage <= 1" @click="loadComments(commentPage - 1)">上一页</button>
            <template v-for="p in getPageRange(commentPage, calcPages(commentTotal, commentPageSize))" :key="p">
              <span v-if="p === '...'" class="pagination__ellipsis">...</span>
              <button v-else class="pagination__btn" :class="{ 'pagination__btn--active': p === commentPage }" @click="loadComments(p)">{{ p }}</button>
            </template>
            <button class="pagination__btn" :disabled="commentPage >= calcPages(commentTotal, commentPageSize)" @click="loadComments(commentPage + 1)">下一页</button>
            <span class="pagination__info">共 {{ commentTotal }} 条</span>
            <select class="pagination__size" v-model.number="commentPageSize" @change="loadComments(1)">
              <option :value="10">10条/页</option>
              <option :value="20">20条/页</option>
              <option :value="50">50条/页</option>
            </select>
          </div>
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
          <div v-if="adminCategories.length" class="pagination">
            <button class="pagination__btn" :disabled="categoryPage <= 1" @click="loadCategories(categoryPage - 1)">上一页</button>
            <template v-for="p in getPageRange(categoryPage, calcPages(categoryTotal, categoryPageSize))" :key="p">
              <span v-if="p === '...'" class="pagination__ellipsis">...</span>
              <button v-else class="pagination__btn" :class="{ 'pagination__btn--active': p === categoryPage }" @click="loadCategories(p)">{{ p }}</button>
            </template>
            <button class="pagination__btn" :disabled="categoryPage >= calcPages(categoryTotal, categoryPageSize)" @click="loadCategories(categoryPage + 1)">下一页</button>
            <span class="pagination__info">共 {{ categoryTotal }} 条</span>
            <select class="pagination__size" v-model.number="categoryPageSize" @change="loadCategories(1)">
              <option :value="10">10条/页</option>
              <option :value="20">20条/页</option>
              <option :value="50">50条/页</option>
            </select>
          </div>
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
          <div v-if="adminTags.length" class="pagination">
            <button class="pagination__btn" :disabled="tagPage <= 1" @click="loadTags(tagPage - 1)">上一页</button>
            <template v-for="p in getPageRange(tagPage, calcPages(tagTotal, tagPageSize))" :key="p">
              <span v-if="p === '...'" class="pagination__ellipsis">...</span>
              <button v-else class="pagination__btn" :class="{ 'pagination__btn--active': p === tagPage }" @click="loadTags(p)">{{ p }}</button>
            </template>
            <button class="pagination__btn" :disabled="tagPage >= calcPages(tagTotal, tagPageSize)" @click="loadTags(tagPage + 1)">下一页</button>
            <span class="pagination__info">共 {{ tagTotal }} 条</span>
            <select class="pagination__size" v-model.number="tagPageSize" @change="loadTags(1)">
              <option :value="10">10条/页</option>
              <option :value="20">20条/页</option>
              <option :value="50">50条/页</option>
            </select>
          </div>
        </section>
      </main>
    </div>

    <!-- ========== 编辑/新增模态框 ========== -->
    <div v-if="modalVisible" class="modal-overlay" @click.self="modalVisible = false">
      <div class="modal">
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
                <div class="modal__input-wrapper" :class="{ 'modal__input-wrapper--error': accountError || accountFormatError, 'modal__input-wrapper--ok': !accountError && !accountFormatError && accountChecked }">
                  <input v-model="modalForm.account" class="modal__input" :class="{ 'modal__input--error': accountError || accountFormatError }" placeholder="请输入 11 位数字账号" maxlength="11" @input="checkAccount" />
                  <span v-if="accountCheckLoading" class="modal__input-suffix">检查中...</span>
                  <span v-else-if="accountError" class="modal__input-suffix modal__input-suffix--error">账号已存在</span>
                  <span v-else-if="accountChecked && !accountFormatError" class="modal__input-suffix modal__input-suffix--ok">可用</span>
                </div>
                <p v-if="accountFormatError" class="modal__field-hint modal__field-hint--error">账号必须为 11 位数字</p>
              </div>
              <div class="modal__field">
                <label class="modal__label">密码</label>
                <div class="modal__input-wrapper" :class="{ 'modal__input-wrapper--error': passwordError }">
                  <input v-model="modalForm.password" class="modal__input" :class="{ 'modal__input--error': passwordError }" type="password" placeholder="11-20 位，需包含字母和数字" @input="checkPasswordFormat" />
                </div>
                <p v-if="passwordError" class="modal__field-hint modal__field-hint--error">{{ passwordErrorMsg }}</p>
                <p v-else class="modal__field-hint">密码需 11-20 位，必须同时包含字母和数字</p>
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

          <!-- ---- 分类/标签 ---- -->
          <template v-else>
            <div class="modal__field">
              <label class="modal__label"><span class="required-mark">*</span>名称</label>
              <input v-model="modalForm.name" class="modal__input" :class="{ 'modal__input--required-empty': !modalForm.name?.trim() }" placeholder="请输入名称" />
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

    <!-- ========== 删除确认模态框 ========== -->
    <div v-if="deleteModalVisible" class="modal-overlay" @click.self="deleteModalVisible = false">
      <div class="modal modal--delete">
        <h3 class="modal__title">确认删除</h3>
        <div class="modal__body">
          <p class="delete-confirm-text">确定要删除该{{ deleteTarget?.type === 'user' ? '用户' : deleteTarget?.type === 'article' ? '文章' : deleteTarget?.type === 'comment' ? '评论' : deleteTarget?.type === 'category' ? '分类' : '标签' }}吗？此操作不可恢复。</p>
        </div>
        <div class="modal__footer">
          <button class="btn btn--cancel" @click="deleteModalVisible = false">取消</button>
          <button class="btn btn--danger" @click="confirmDelete">确认删除</button>
        </div>
      </div>
    </div>

    <!-- ========== 一键转移分类模态框 ========== -->
    <div v-if="transferModalVisible" class="modal-overlay" @click.self="transferModalVisible = false">
      <div class="modal">
        <h3 class="modal__title">一键转移分类</h3>
        <div class="modal__body">
          <div class="modal__field">
            <label class="modal__label"><span class="required-mark">*</span>源分类</label>
            <select v-model.number="transferFrom" class="modal__input" :class="{ 'modal__input--required-empty': !transferFrom }">
              <option :value="0" disabled>请选择源分类</option>
              <option v-for="c in enabledCategories" :key="c.id" :value="c.id">{{ c.name }}</option>
            </select>
          </div>
          <div class="modal__field">
            <label class="modal__label"><span class="required-mark">*</span>目标分类</label>
            <select v-model.number="transferTo" class="modal__input" :class="{ 'modal__input--required-empty': !transferTo }">
              <option :value="0" disabled>请选择目标分类</option>
              <option v-for="c in enabledCategories" :key="c.id" :value="c.id">{{ c.name }}</option>
            </select>
          </div>
          <p class="transfer-hint">将源分类下的所有文章转移到目标分类。</p>
        </div>
        <div class="modal__footer">
          <button class="btn btn--cancel" @click="transferModalVisible = false">取消</button>
          <button class="btn btn--primary" :disabled="!transferFrom || !transferTo || transferFrom === transferTo" @click="handleTransfer">确认转移</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, type Ref } from 'vue'
import { useRouter } from 'vue-router'
import api from '../api'
import NavBar from '../components/NavBar.vue'
import { showToast } from '../utils/toast'

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
  category?: { id: number; name: string }
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

interface AdminCommentItem {
  id: number
  article_title: string
  user_name: string
  content: string
  created_at: string
}

// ---------- 状态 ----------
const router = useRouter()
const activeTab = ref('users')
const tabs = [
  { key: 'users', label: '用户管理', icon: '👥' },
  { key: 'articles', label: '文章管理', icon: '📝' },
  { key: 'comments', label: '评论管理', icon: '💬' },
  { key: 'categories', label: '分类管理', icon: '📁' },
  { key: 'tags', label: '标签管理', icon: '🏷️' }
]

const users = ref<UserItem[]>([])
const adminArticles = ref<ArticleItem[]>([])
const adminCategories = ref<CatTagItem[]>([])
const adminTags = ref<CatTagItem[]>([])
const adminComments = ref<AdminCommentItem[]>([])
const selectedCommentIds = ref<number[]>([])
const batchMode = ref(false)

// 分页状态
const userPage = ref(1)
const userPageSize = ref(10)
const userTotal = ref(0)
const articlePage = ref(1)
const articlePageSize = ref(10)
const articleTotal = ref(0)
const commentPage = ref(1)
const commentPageSize = ref(10)
const commentTotal = ref(0)
const categoryPage = ref(1)
const categoryPageSize = ref(10)
const categoryTotal = ref(0)
const tagPage = ref(1)
const tagPageSize = ref(10)
const tagTotal = ref(0)

// 模态框
const modalVisible = ref(false)
const modalType = ref<'user' | 'category' | 'tag'>('user')
const editingId = ref<number | null>(null)
const editingUser = ref<UserItem | null>(null)

// 删除确认模态框
const deleteModalVisible = ref(false)
const deleteTarget = ref<{ type: string; id: number } | null>(null)

// 一键转移分类
const transferModalVisible = ref(false)
const transferFrom = ref(0)
const transferTo = ref(0)
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

// 仅启用的分类和标签（用于文章表单）
const enabledCategories = computed(() => adminCategories.value.filter(c => c.status === 1))
// 唯一性校验状态
const nameCheckLoading = ref(false)
const accountCheckLoading = ref(false)
const nameError = ref(false)
const accountError = ref(false)
const nameChecked = ref(false)
const accountChecked = ref(false)
let nameCheckTimer: ReturnType<typeof setTimeout> | null = null
let accountCheckTimer: ReturnType<typeof setTimeout> | null = null

// 账号格式校验
const accountFormatError = ref(false)
// 密码格式校验
const passwordError = ref(false)
const passwordErrorMsg = ref('')

const checkPasswordFormat = () => {
  const val = modalForm.password || ''
  if (!val) {
    passwordError.value = false
    passwordErrorMsg.value = ''
    return
  }
  if (val.length < 11 || val.length > 20) {
    passwordError.value = true
    passwordErrorMsg.value = '密码长度需为 11-20 位'
    return
  }
  if (!/[a-zA-Z]/.test(val) || !/\d/.test(val)) {
    passwordError.value = true
    passwordErrorMsg.value = '密码必须同时包含字母和数字'
    return
  }
  if (/[^a-zA-Z0-9]/.test(val)) {
    passwordError.value = true
    passwordErrorMsg.value = '密码只能包含字母和数字'
    return
  }
  passwordError.value = false
  passwordErrorMsg.value = ''
}

// 是否全选评论
const allCommentsSelected = computed(() => {
  return adminComments.value.length > 0 && selectedCommentIds.value.length === adminComments.value.length
})

// 全选/取消全选评论
const toggleAllComments = () => {
  if (allCommentsSelected.value) {
    selectedCommentIds.value = []
  } else {
    selectedCommentIds.value = adminComments.value.map(c => c.id)
  }
}

// 退出批量模式
const exitBatchMode = () => {
  batchMode.value = false
  selectedCommentIds.value = []
}

// 批量删除评论
const handleBatchDeleteComments = async () => {
  if (!selectedCommentIds.value.length) return
  try {
    await api.delete('/admin/comments', { data: { ids: selectedCommentIds.value } })
    showToast('批量删除成功', 'success')
    exitBatchMode()
    await loadComments(commentPage.value)
  } catch (error: any) {
    console.error('批量删除失败:', error)
    const msg = error?.response?.data?.message || '批量删除失败，请重试'
    showToast(msg, 'error')
  }
}

// ---------- 分页加载 ----------
const loadUsers = async (page = 1) => {
  userPage.value = page
  try {
    const res = await api.get('/admin/users', { params: { page: userPage.value, page_size: userPageSize.value } })
    users.value = res.data.data?.list || []
    userTotal.value = res.data.data?.total || 0
  } catch (e) { console.error('加载用户失败:', e) }
}

const loadArticles = async (page = 1) => {
  articlePage.value = page
  try {
    const res = await api.get('/admin/articles', { params: { page: articlePage.value, page_size: articlePageSize.value } })
    adminArticles.value = res.data.data?.list || []
    articleTotal.value = res.data.data?.total || 0
  } catch (e) { console.error('加载文章失败:', e) }
}

const loadComments = async (page = 1) => {
  commentPage.value = page
  try {
    const res = await api.get('/admin/comments', { params: { page: commentPage.value, page_size: commentPageSize.value } })
    adminComments.value = res.data.data?.list || []
    commentTotal.value = res.data.data?.total || 0
  } catch (e) { console.error('加载评论失败:', e) }
}

const loadCategories = async (page = 1) => {
  categoryPage.value = page
  try {
    const res = await api.get('/admin/categories', { params: { page: categoryPage.value, page_size: categoryPageSize.value } })
    adminCategories.value = res.data.data?.list || []
    categoryTotal.value = res.data.data?.total || 0
  } catch (e) { console.error('加载分类失败:', e) }
}

const loadTags = async (page = 1) => {
  tagPage.value = page
  try {
    const res = await api.get('/admin/tags', { params: { page: tagPage.value, page_size: tagPageSize.value } })
    adminTags.value = res.data.data?.list || []
    tagTotal.value = res.data.data?.total || 0
  } catch (e) { console.error('加载标签失败:', e) }
}

const calcPages = (total: number, size: number) => Math.ceil(total / size) || 1

const getPageRange = (current: number, total: number): (number | '...')[] => {
  if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1)
  const pages: (number | '...')[] = []
  if (current <= 4) {
    for (let i = 1; i <= 5; i++) pages.push(i)
    pages.push('...'); pages.push(total)
  } else if (current >= total - 3) {
    pages.push(1); pages.push('...')
    for (let i = total - 4; i <= total; i++) pages.push(i)
  } else {
    pages.push(1); pages.push('...')
    for (let i = current - 1; i <= current + 1; i++) pages.push(i)
    pages.push('...'); pages.push(total)
  }
  return pages
}

// 保存按钮是否禁用（新增用户时，昵称或校验失败或未通过唯一性校验时禁用）
const canSaveDisabled = computed(() => {
  // 新增用户：唯一性校验通过后才可保存
  if (modalType.value === 'user' && !editingId.value) {
    return nameError.value || accountError.value || accountFormatError.value || passwordError.value || nameCheckLoading.value || accountCheckLoading.value || !nameChecked.value || !accountChecked.value
  }
  // 分类/标签：名称必填
  if (modalType.value === 'category' || modalType.value === 'tag') {
    return !modalForm.name?.trim()
  }
  return false
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
    accountFormatError.value = false
    return
  }
  // 格式校验：11 位数字
  if (!/^\d{11}$/.test(val)) {
    accountFormatError.value = true
    accountError.value = false
    accountChecked.value = false
    accountCheckLoading.value = false
    return
  }
  accountFormatError.value = false
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

const openModal = (type: 'user' | 'category' | 'tag', item?: any) => {
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
  accountFormatError.value = false
  passwordError.value = false
  passwordErrorMsg.value = ''

  // reset
  modalForm.user_name = ''
  modalForm.account = ''
  modalForm.password = ''
  modalForm.status = 1
  modalForm.name = ''

  if (item) {
    modalTitle.value = type === 'user' ? '编辑用户' : type === 'category' ? '编辑分类' : '编辑标签'
    if (type === 'user') {
      modalForm.status = item.status ?? 1
    } else {
      modalForm.name = item.name || ''
      modalForm.status = item.status ?? 1
    }
  } else {
    modalTitle.value = type === 'user' ? '新增用户' : type === 'category' ? '新增分类' : '新增标签'
  }
}

const closeModal = () => {
  modalVisible.value = false
}

const handleSave = async () => {
  try {
    if (modalType.value === 'user') {
      if (editingId.value) {
        // 更新用户状态
        await api.put(`/admin/users/${editingId.value}`, { status: modalForm.status })
      } else {
        // 创建用户
        await api.post('/admin/users', {
          user_name: modalForm.user_name,
          account: modalForm.account,
          password: modalForm.password,
          status: modalForm.status
        })
      }
      // 重新加载用户列表
      await loadUsers(editingId.value ? userPage.value : 1)
    } else if (modalType.value === 'category') {
      const categoryData = {
        name: modalForm.name,
        status: modalForm.status
      }
      
      if (editingId.value) {
        await api.put(`/admin/categories/${editingId.value}`, categoryData)
      } else {
        await api.post('/admin/categories', categoryData)
      }
      // 重新加载分类列表
      await loadCategories(editingId.value ? categoryPage.value : 1)
    } else if (modalType.value === 'tag') {
      const tagData = {
        name: modalForm.name,
        status: modalForm.status
      }
      
      if (editingId.value) {
        await api.put(`/admin/tags/${editingId.value}`, tagData)
      } else {
        await api.post('/admin/tags', tagData)
      }
      // 重新加载标签列表
      await loadTags(editingId.value ? tagPage.value : 1)
    }
    
    modalVisible.value = false
  } catch (error: any) {
    console.error('保存失败:', error)
    const msg = error?.response?.data?.message || '保存失败，请重试'
    showToast(msg, 'error')
  }
}

const handleDelete = (type: string, id: number) => {
  deleteTarget.value = { type, id }
  deleteModalVisible.value = true
}

const confirmDelete = async () => {
  if (!deleteTarget.value) return
  const { type, id } = deleteTarget.value
  deleteModalVisible.value = false
  deleteTarget.value = null

  const reloadWithFallback = async (loadFn: Function, page: Ref<number>, data: Ref<any[]>) => {
    await loadFn(page.value)
    if (data.value.length === 0 && page.value > 1) {
      await loadFn(page.value - 1)
    }
  }

  try {
    if (type === 'user') {
      await api.delete(`/admin/users/${id}`)
      await reloadWithFallback(loadUsers, userPage, users)
    } else if (type === 'article') {
      await api.delete(`/admin/articles/${id}`)
      await reloadWithFallback(loadArticles, articlePage, adminArticles)
    } else if (type === 'comment') {
      await api.delete(`/admin/comments/${id}`)
      await reloadWithFallback(loadComments, commentPage, adminComments)
    } else if (type === 'category') {
      await api.delete(`/admin/categories/${id}`)
      await reloadWithFallback(loadCategories, categoryPage, adminCategories)
    } else if (type === 'tag') {
      await api.delete(`/admin/tags/${id}`)
      await reloadWithFallback(loadTags, tagPage, adminTags)
    }
    showToast('删除成功', 'success')
  } catch (error: any) {
    console.error('删除失败:', error)
    const msg = error?.response?.data?.message || '删除失败，请重试'
    showToast(msg, 'error')
  }
}

const handleTransfer = async () => {
  if (!transferFrom.value || !transferTo.value || transferFrom.value === transferTo.value) return
  try {
    await api.put('/admin/articles/transfer', {
      from_type_id: transferFrom.value,
      to_type_id: transferTo.value
    })
    showToast('转移成功', 'success')
    transferModalVisible.value = false
    transferFrom.value = 0
    transferTo.value = 0
    // 重新加载文章和分类列表
    await Promise.all([
      loadArticles(articlePage.value),
      loadCategories(categoryPage.value)
    ])
  } catch (error: any) {
    console.error('转移失败:', error)
    const msg = error?.response?.data?.message || '转移失败，请重试'
    showToast(msg, 'error')
  }
}

onMounted(async () => {
  try {
    await Promise.all([
      loadUsers(1),
      loadArticles(1),
      loadComments(1),
      loadCategories(1),
      loadTags(1)
    ])
  } catch (error) {
    console.error('加载数据失败:', error)
    showToast('加载数据失败，请刷新页面重试', 'error')
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

.modal__field-hint {
  margin: 4px 0 0;
  font-size: 12px;
  color: #999;
  line-height: 1.4;
}
.modal__field-hint--error {
  color: #ff4d4f;
}

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

.modal__input--required-empty { border-color: #ff4d4f; }
.modal__input--required-empty:focus { border-color: #ff4d4f; box-shadow: 0 0 0 2px rgba(255,77,79,0.1); }

.required-mark { color: #ff4d4f; margin-right: 2px; font-size: 14px; }

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
.btn--danger { background: #ff4d4f; color: #fff; border-color: #ff4d4f; }
.btn--danger:hover { background: #ff7875; border-color: #ff7875; }

.modal--delete { width: 380px; }

.delete-confirm-text { font-size: 14px; color: #333; line-height: 1.6; margin: 8px 0; }

.transfer-hint {
  font-size: 12px;
  color: #999;
  margin: 8px 0 0;
}

/* ---------- 分页 ---------- */
.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 6px;
  padding: 16px 0;
  flex-wrap: wrap;
}

.pagination__btn {
  min-width: 32px;
  height: 32px;
  padding: 0 8px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  background: #fff;
  color: #333;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}

.pagination__btn:hover { border-color: #1677ff; color: #1677ff; }
.pagination__btn--active { background: #1677ff; color: #fff; border-color: #1677ff; }
.pagination__btn--active:hover { background: #4096ff; border-color: #4096ff; color: #fff; }
.pagination__btn:disabled { color: #d9d9d9; border-color: #d9d9d9; cursor: not-allowed; background: #fafafa; }

.pagination__ellipsis {
  display: inline-block;
  width: 24px;
  text-align: center;
  color: #999;
  font-size: 13px;
}

.pagination__info {
  margin-left: 8px;
  font-size: 13px;
  color: #666;
}

.pagination__size {
  margin-left: 4px;
  padding: 4px 6px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  font-size: 13px;
  color: #333;
  background: #fff;
  cursor: pointer;
}
</style>