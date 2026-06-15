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
            <span>&#128065; {{ article.views }} 阅读</span>
          </div>
          <div class="article-content__body">{{ article.content }}</div>
        </section>

        <!-- 评论区 -->
        <aside class="article-comments">
          <div class="comments-header">评论 ({{ comments.length }})</div>
          <div class="comments-list">
            <div v-for="c in comments" :key="c.id" class="comment-item">
              <div class="comment-item__header">
                <img v-if="c.avatar_url" :src="c.avatar_url" class="comment-item__avatar" />
                <span v-else class="comment-item__avatar table-avatar--placeholder">&#128100;</span>
                <span class="comment-item__name">{{ c.user_name }}</span>
                <span class="comment-item__time">{{ fmt(c.created_at) }}</span>
              </div>
              <p class="comment-item__content">{{ c.content }}</p>
            </div>
            <div v-if="!comments.length" class="admin-empty">暂无评论，来写第一条吧</div>
          </div>

          <div class="comments-form">
            <textarea v-model="newComment" placeholder="写下你的评论..." rows="2"></textarea>
            <div class="comments-form__actions">
              <button class="btn btn--primary" :disabled="!newComment.trim()" @click="submitComment">
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
  likes: number
  author: string
  type_id: number
  created_at: string
}

interface Comment {
  id: number
  content: string
  user_name: string
  avatar_url: string
  created_at: string
}

const route = useRoute()
const article = ref<Article>({ id: 0, title: '', content: '', cover_url: '', summary: '', views: 0, likes: 0, author: '', type_id: 0, created_at: '' })
const comments = ref<Comment[]>([])
const newComment = ref('')

const fmt = (d: string) => {
  if (!d) return ''
  return new Date(d).toLocaleDateString('zh-CN')
}

const submitComment = async () => {
  if (!newComment.value.trim()) return
  try {
    await api.post(`/articles/${route.params.id}/comments`, { content: newComment.value })
    newComment.value = ''
    loadComments()
  } catch {
    // mock
    comments.value.push({
      id: Date.now(),
      content: newComment.value,
      user_name: '当前用户',
      avatar_url: '',
      created_at: new Date().toISOString()
    })
    newComment.value = ''
  }
}

const loadComments = async () => {
  try {
    const res = await api.get(`/articles/${route.params.id}/comments`)
    comments.value = res.data.data || res.data || []
  } catch {
    comments.value = [
      { id: 1, content: '写得很好，受益匪浅！', user_name: '读者A', avatar_url: '', created_at: '2026-06-13T10:00:00Z' },
      { id: 2, content: '希望作者能继续更新这个系列。', user_name: '读者B', avatar_url: '', created_at: '2026-06-13T14:30:00Z' }
    ]
  }
}

onMounted(async () => {
  const id = route.params.id as string
  try {
    const res = await api.get(`/articles/${id}`)
    article.value = res.data.data || res.data
  } catch {
    article.value = {
      id: Number(id),
      title: 'Go 语言并发编程深入解析',
      content: 'Go 语言以其简洁高效的并发模型著称...\n\nGoroutine 是 Go 并发模型的核心，它是一种轻量级线程，由 Go 运行时管理。\n\nChannel 则提供了 goroutine 之间的通信机制...',
      cover_url: '',
      summary: '深入理解 goroutine 调度',
      views: 2340,
      likes: 186,
      author: '张三',
      type_id: 1,
      created_at: '2026-06-10T08:00:00Z'
    }
  }
  loadComments()
})
</script>
