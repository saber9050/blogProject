<template>
  <div class="article-page">
    <NavBar />
    <main class="page-shell">
      <div class="article-detail">
        <!-- 文章内容 -->
        <section class="article-content">
          <h1 class="article-content__title">{{ article.title }}</h1>
          <div class="article-content__meta">
            <span>&#128100; {{ article.author }}</span>
            <span>&#128197; {{ fmt(article.created_at) }}</span>
            <span>&#128065; {{ article.view_count || article.views }} 阅读</span>
            <button
              class="article-content__like-btn"
              :class="{ 'article-content__like-btn--liked': article.is_liked }"
              @click="toggleDetailLike"
            >
              <span v-if="article.is_liked">&#10084;</span>
              <span v-else>&#9825;</span>
              {{ article.like_count || article.likes }} 点赞
            </button>
            <span>&#128172; {{ article.comment_count || 0 }} 评论</span>
          </div>
          <div class="article-content__body">{{ article.content }}</div>
        </section>

        <!-- 评论区 -->
        <aside class="article-comments">
          <div class="comments-header">评论 ({{ commentTotal }})</div>

          <div class="comments-list">
            <!-- 一级评论列表 -->
            <div v-for="c in comments" :key="c.id" class="comment-item">
              <div class="comment-item__header">
                <img v-if="c.avatar_url" :src="c.avatar_url" class="comment-item__avatar" />
                <span v-else class="comment-item__avatar table-avatar--placeholder">&#128100;</span>
                <span class="comment-item__name">{{ c.user_name }}</span>
                <span class="comment-item__time">{{ fmt(c.created_at) }}</span>
              </div>
              <p v-if="c.is_deleted" class="comment-item__content comment-item__content--deleted">
                该评论已删除
              </p>
              <p v-else class="comment-item__content">{{ c.content }}</p>

              <!-- 操作按钮 -->
              <div class="comment-item__actions">
                <button
                  v-if="!c.is_deleted"
                  class="comment-btn"
                  @click="startReply(c.id, c.user_name)"
                >
                  回复
                </button>
                <button
                  v-if="canDelete(c.user_id)"
                  class="comment-btn comment-btn--danger"
                  @click="doDelete(c.id, null)"
                >
                  撤回
                </button>
              </div>

              <!-- 回复输入框 -->
              <div v-if="replyingTo?.commentId === c.id" class="comment-reply-form">
                <textarea
                  v-model="replyText"
                  :placeholder="`回复 ${replyingTo?.userName}...`"
                  rows="2"
                />
                <div class="comment-reply-form__actions">
                  <button class="comment-btn" @click="cancelReply">取消</button>
                  <button
                    class="comment-btn comment-btn--primary"
                    :disabled="!replyText.trim()"
                    @click="submitReply(c.id)"
                  >
                    发表
                  </button>
                </div>
              </div>

              <!-- 二级评论 -->
              <div v-if="c._childrenLoaded" class="comment-children">
                <div
                  v-for="sub in c._children"
                  :key="sub.id"
                  class="comment-item comment-item--child"
                >
                  <div class="comment-item__header">
                    <img v-if="sub.avatar_url" :src="sub.avatar_url" class="comment-item__avatar" />
                    <span v-else class="comment-item__avatar table-avatar--placeholder">&#128100;</span>
                    <span class="comment-item__name">{{ sub.user_name }}</span>
                    <span v-if="sub.reply_to_name" class="comment-item__reply-to">
                      回复 {{ sub.reply_to_name }}
                    </span>
                    <span class="comment-item__time">{{ fmt(sub.created_at) }}</span>
                  </div>
                  <p v-if="sub.is_deleted" class="comment-item__content comment-item__content--deleted">
                    该评论已删除
                  </p>
                  <p v-else class="comment-item__content">{{ sub.content }}</p>

                  <div class="comment-item__actions">
                    <button
                      v-if="!sub.is_deleted"
                      class="comment-btn"
                      @click="startReply(c.id, sub.user_name)"
                    >
                      回复
                    </button>
                    <button
                      v-if="canDelete(sub.user_id)"
                      class="comment-btn comment-btn--danger"
                      @click="doDelete(sub.id, c.id)"
                    >
                      撤回
                    </button>
                  </div>
                </div>

                <!-- 加载更多二级评论 -->
                <div v-if="c._childrenHasMore" class="comment-load-more">
                  <button class="comment-btn" @click="loadMoreChildren(c)">
                    查看更多回复
                  </button>
                </div>
              </div>

              <!-- 展开二级评论 -->
              <div v-if="!c._childrenLoaded && c._childrenTotal > 0" class="comment-load-more">
                <button class="comment-btn" @click="toggleChildren(c)">
                  展开 {{ c._childrenTotal }} 条回复
                </button>
              </div>
            </div>

            <div v-if="!comments.length" class="admin-empty">暂无评论，来写第一条吧</div>

            <!-- 一级评论分页 -->
            <div v-if="commentHasMore" class="comment-load-more">
              <button class="comment-btn" @click="loadNextPage">加载更多评论</button>
            </div>
          </div>

          <!-- 发表评论表单 -->
          <div class="comments-form">
            <textarea
              v-model="newComment"
              placeholder="写下你的评论..."
              rows="2"
            />
            <div class="comments-form__actions">
              <button
                class="btn btn--primary"
                :disabled="!newComment.trim()"
                @click="submitComment"
              >
                发表评论
              </button>
            </div>
          </div>
        </aside>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import api from '../api'
import NavBar from '../components/NavBar.vue'

interface Article {
  id: number
  title: string
  content: string
  cover_url: string
  summary: string
  views: number
  view_count: number
  likes: number
  like_count: number
  comment_count: number
  is_liked: boolean
  author: string
  type_id: number
  created_at: string
}

interface Comment {
  id: number
  content: string
  user_name: string
  avatar_url: string
  user_id: number
  parent_id: number | null
  reply_to_name: string
  is_deleted: boolean
  created_at: string
  _childrenLoaded: boolean
  _children: Comment[]
  _childrenTotal: number
  _childrenPage: number
  _childrenHasMore: boolean
}

const route = useRoute()
const article = ref<Article>({
  id: 0,
  title: '',
  content: '',
  cover_url: '',
  summary: '',
  views: 0,
  view_count: 0,
  likes: 0,
  like_count: 0,
  comment_count: 0,
  is_liked: false,
  author: '',
  type_id: 0,
  created_at: ''
})
const comments = ref<Comment[]>([])
const newComment = ref('')
const replyText = ref('')
const replyingTo = ref<{ commentId: number; userName: string } | null>(null)
const commentPage = ref(1)
const commentTotal = ref(0)
const commentHasMore = ref(false)

// 当前登录用户信息
const currentUser = ref<{ id: number; user_name: string; role_id: number; avatar_url?: string } | null>(null)

const loadCurrentUser = () => {
  try {
    const raw = localStorage.getItem('user')
    if (raw) {
      currentUser.value = JSON.parse(raw)
    }
  } catch {
    currentUser.value = null
  }
}

const canDelete = (commentUserId: number) => {
  if (!currentUser.value) return false
  // 管理员可以删除任何人的评论
  if (currentUser.value.role_id === 1) return true
  // 普通用户只能删除自己的
  return currentUser.value.id === commentUserId
}

const fmt = (d: string) => {
  if (!d) return ''
  return new Date(d).toLocaleDateString('zh-CN')
}

// 详情页点赞/取消点赞（乐观更新）
const toggleDetailLike = async () => {
  if (!currentUser.value) {
    alert('请先登录')
    return
  }

  const wasLiked = article.value.is_liked
  const oldLikeCount = article.value.like_count || article.value.likes || 0

  // 乐观更新 UI
  article.value.is_liked = !wasLiked
  if (article.value.like_count !== undefined) article.value.like_count += wasLiked ? -1 : 1
  if (article.value.likes !== undefined) article.value.likes += wasLiked ? -1 : 1

  try {
    if (wasLiked) {
      await api.delete(`/articles/${article.value.id}/like`)
    } else {
      await api.post(`/articles/${article.value.id}/like`)
    }
  } catch {
    // 失败回滚
    article.value.is_liked = wasLiked
    if (article.value.like_count !== undefined) article.value.like_count = oldLikeCount
    if (article.value.likes !== undefined) article.value.likes = oldLikeCount
  }
}

const PAGE_SIZE = 20
const CHILD_PAGE_SIZE = 10
const CHILD_INITIAL = 2

// 发表一级评论
const submitComment = async () => {
  if (!newComment.value.trim()) return
  if (!currentUser.value) {
    alert('请先登录')
    return
  }
  try {
    const res = await api.post(`/articles/${route.params.id}/comments`, {
      content: newComment.value,
      parent_id: null
    })
    const data = res.data.data || res.data
    // 用 API 返回的 id/时间 + 本地用户信息拼接新评论
    const newC: Comment = {
      id: data.id,
      content: newComment.value,
      user_name: currentUser.value.user_name,
      avatar_url: currentUser.value.avatar_url || '',
      user_id: currentUser.value.id,
      parent_id: null,
      reply_to_name: '',
      is_deleted: false,
      created_at: data.created_at || new Date().toISOString(),
      _childrenLoaded: false,
      _children: [],
      _childrenTotal: 0,
      _childrenPage: 1,
      _childrenHasMore: false
    }
    comments.value.unshift(newC)
    commentTotal.value++
    newComment.value = ''
  } catch {
    alert('发表评论失败，请稍后重试')
  }
}

// 开始回复
const startReply = (commentId: number, userName: string) => {
  replyingTo.value = { commentId, userName }
  replyText.value = ''
}

const cancelReply = () => {
  replyingTo.value = null
  replyText.value = ''
}

// 提交回复（二级评论，parent_id 始终是根评论 ID）
const submitReply = async (rootCommentId: number) => {
  if (!replyText.value.trim() || !replyingTo.value) return
  if (!currentUser.value) {
    alert('请先登录')
    return
  }

  try {
    await api.post(`/articles/${route.params.id}/comments`, {
      content: replyText.value,
      parent_id: rootCommentId,
      reply_to_user_name: replyingTo.value.userName
    })
    replyText.value = ''
    replyingTo.value = null
    // 重新加载该一级评论的子评论
    const parent = comments.value.find((c) => c.id === rootCommentId)
    if (parent) refreshChildren(parent)
  } catch {
    alert('回复失败，请稍后重试')
  }
}

// 删除评论
const doDelete = async (commentId: number, parentId: number | null) => {
  try {
    await api.delete(`/articles/${route.params.id}/comments/${commentId}`)
  } catch {
    alert('删除失败，请稍后重试')
    return
  }

  if (parentId) {
    // 删除二级评论
    const parent = comments.value.find((c) => c.id === parentId)
    if (parent) {
      const sub = parent._children.find((s) => s.id === commentId)
      if (sub) {
        sub.is_deleted = true
        sub.content = ''
      }
      parent._childrenTotal = Math.max(0, parent._childrenTotal - 1)
    }
  } else {
    // 删除一级评论
    const c = comments.value.find((c) => c.id === commentId)
    if (c) {
      c.is_deleted = true
      c.content = ''
    }
    commentTotal.value = Math.max(0, commentTotal.value - 1)
  }
}

// 展开/收起二级评论
const toggleChildren = async (c: Comment) => {
  if (c._childrenLoaded) {
    // 已加载，只是收起
    c._childrenLoaded = false
    c._children = []
    return
  }
  await loadChildren(c, 1)
}

// 加载二级评论
const loadChildren = async (c: Comment, page: number) => {
  try {
    const res = await api.get(
      `/articles/${route.params.id}/comments/${c.id}/replies`,
      { params: { page, page_size: CHILD_PAGE_SIZE } }
    )
    const data = res.data.data || res.data
    const list = (data.list || data || []).map((item: any) => ({
      ...item,
      _childrenLoaded: true,
      _children: [],
      _childrenTotal: 0,
      _childrenPage: 1,
      _childrenHasMore: false
    }))
    if (page === 1) {
      c._children = list
    } else {
      c._children.push(...list)
    }
    c._childrenLoaded = true
    c._childrenTotal = data.total || 0
    c._childrenPage = page
    c._childrenHasMore = (data.total || 0) > page * CHILD_PAGE_SIZE
  } catch {
    // 加载失败，不做降级处理
  }
}

// 加载更多二级评论
const loadMoreChildren = async (c: Comment) => {
  await loadChildren(c, (c._childrenPage || 0) + 1)
}

// 刷新二级评论
const refreshChildren = async (c: Comment) => {
  c._childrenPage = 1
  await loadChildren(c, 1)
}

// 加载一级评论下一页
const loadNextPage = async () => {
  commentPage.value++
  await loadComments()
}

// 加载一级评论
const loadComments = async () => {
  try {
    const res = await api.get(`/articles/${route.params.id}/comments`, {
      params: { page: commentPage.value, page_size: PAGE_SIZE }
    })
    const data = res.data.data || res.data
    const list = (data.list || data || []).map((item: any) => ({
      ...item,
      _childrenLoaded: false,
      _children: [],
      _childrenTotal: item.children_total || 0,
      _childrenPage: 1,
      _childrenHasMore: (item.children_total || 0) > CHILD_INITIAL
    }))
    if (commentPage.value === 1) {
      comments.value = list
    } else {
      comments.value.push(...list)
    }
    commentTotal.value = data.total || 0
    commentHasMore.value = (data.total || 0) > commentPage.value * PAGE_SIZE
  } catch {
    // 加载失败，保持现有数据
  }
}

onMounted(async () => {
  loadCurrentUser()
  const id = route.params.id as string
  try {
    const res = await api.get(`/articles/${id}`)
    const data = res.data.data || res.data
    article.value = {
      ...article.value,
      ...data,
      // 兼容旧字段名
      view_count: data.view_count ?? data.views ?? 0,
      like_count: data.like_count ?? data.likes ?? 0,
      comment_count: data.comment_count ?? 0,
      is_liked: data.is_liked ?? false
    }
  } catch {
    // 文章加载失败
  }
  loadComments()
})
</script>

<style scoped>
/* article-detail layout */
.article-detail {
  display: grid;
  grid-template-columns: 1fr 340px;
  gap: 24px;
  align-items: start;
}

.article-content {
  padding: 28px 32px;
  border: 1px solid rgba(255, 255, 255, 0.55);
  border-radius: var(--radius-lg);
  background: rgba(255, 255, 255, 0.85);
  backdrop-filter: blur(12px);
  box-shadow: 0 4px 18px rgba(15, 23, 42, 0.04);
  overflow-wrap: break-word;
}

.article-content__title {
  font-size: 1.5rem;
  font-weight: 800;
  color: var(--text-strong);
  line-height: 1.35;
  margin-bottom: 10px;
}

.article-content__meta {
  display: flex;
  align-items: center;
  gap: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border);
  margin-bottom: 18px;
  font-size: 0.85rem;
  color: var(--text-muted);
}

.article-content__body {
  font-size: 0.95rem;
  line-height: 1.8;
  color: var(--text-secondary);
  white-space: pre-wrap;
}

/* ========== 详情页点赞按钮 ========== */
.article-content__like-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: rgba(255,255,255,0.7);
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text-muted);
  cursor: pointer;
  transition: all 0.2s ease;
}

.article-content__like-btn:hover {
  border-color: var(--danger);
  color: var(--danger);
  transform: scale(1.05);
}

.article-content__like-btn--liked {
  border-color: var(--danger);
  color: var(--danger);
  background: rgba(220,38,38,0.06);
}

/* ========== 评论区 ========== */
.article-comments {
  border: 1px solid rgba(255, 255, 255, 0.55);
  border-radius: var(--radius-lg);
  background: rgba(255, 255, 255, 0.85);
  backdrop-filter: blur(12px);
  box-shadow: 0 4px 18px rgba(15, 23, 42, 0.04);
  overflow: hidden;
  display: flex;
  flex-direction: column;
  max-height: calc(100vh - 100px);
}

.comments-header {
  padding: 14px 18px;
  font-size: 0.95rem;
  font-weight: 700;
  color: var(--text-strong);
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.comments-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px 18px;
}

.comment-item {
  padding: 10px 0;
  border-bottom: 1px solid var(--border);
}

.comment-item:last-child {
  border-bottom: none;
}

.comment-item__header {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 4px;
}

.comment-item__avatar {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  object-fit: cover;
  background: var(--primary-soft);
  flex-shrink: 0;
}

.comment-item__name {
  font-size: 0.82rem;
  font-weight: 700;
  color: var(--text-strong);
}

.comment-item__reply-to {
  font-size: 0.78rem;
  color: var(--text-muted);
}

.comment-item__time {
  margin-left: auto;
  font-size: 0.72rem;
  color: var(--text-faint);
  flex-shrink: 0;
}

.comment-item__content {
  font-size: 0.85rem;
  color: var(--text-secondary);
  line-height: 1.55;
  margin: 0;
  padding-left: 30px;
}

.comment-item__content--deleted {
  color: var(--text-faint);
  font-style: italic;
}

/* 操作按钮行 */
.comment-item__actions {
  display: flex;
  gap: 8px;
  padding-left: 30px;
  margin-top: 4px;
}

.comment-btn {
  border: none;
  background: none;
  font-size: 0.76rem;
  color: var(--text-muted);
  cursor: pointer;
  padding: 2px 6px;
  border-radius: 4px;
  transition: color 0.2s, background 0.2s;
}

.comment-btn:hover {
  color: var(--primary-600);
  background: rgba(16, 185, 129, 0.06);
}

.comment-btn--primary {
  color: var(--primary-600);
  font-weight: 600;
}

.comment-btn--danger {
  color: var(--danger);
}

.comment-btn--danger:hover {
  color: var(--danger);
  background: rgba(220, 38, 38, 0.06);
}

/* 二级评论容器 */
.comment-children {
  margin-top: 6px;
  padding-left: 30px;
  border-left: 2px solid rgba(16, 185, 129, 0.15);
}

.comment-item--child {
  padding: 8px 0 8px 8px;
  border-bottom: 1px solid var(--border);
}

.comment-item--child:last-child {
  border-bottom: none;
}

/* 回复输入框 */
.comment-reply-form {
  margin-top: 8px;
  padding-left: 30px;
}

.comment-reply-form textarea {
  width: 100%;
  min-height: 60px;
  padding: 8px 12px;
  border: 1px solid var(--border-strong);
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.7);
  font-size: 0.82rem;
  color: var(--text-strong);
  resize: vertical;
  outline: none;
  transition: border-color 0.24s ease, box-shadow 0.24s ease;
}

.comment-reply-form textarea:focus {
  border-color: var(--primary-500);
  box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.1);
}

.comment-reply-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: 6px;
  margin-top: 6px;
}

/* 加载更多按钮 */
.comment-load-more {
  padding: 10px 0;
  text-align: center;
}

/* ========== 发表评论表单 ========== */
.comments-form {
  padding: 14px 18px;
  border-top: 1px solid var(--border);
  flex-shrink: 0;
}

.comments-form textarea {
  width: 100%;
  min-height: 64px;
  padding: 8px 12px;
  border: 1px solid var(--border-strong);
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.7);
  font-size: 0.85rem;
  color: var(--text-strong);
  resize: vertical;
  outline: none;
  transition: border-color 0.24s ease, box-shadow 0.24s ease;
}

.comments-form textarea:focus {
  border-color: var(--primary-500);
  box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.12);
}

.comments-form__actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 8px;
}

@media (max-width: 768px) {
  .article-detail {
    grid-template-columns: 1fr;
  }
}
</style>
