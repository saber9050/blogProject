<template>
  <div class="home-page">
    <NavBar @search="onSearch" />
    <main class="page-shell">
      <div class="home-layout">
        <div class="home-main">
          <!-- 排序规则 -->
          <div class="home-bar">
            <span class="home-bar__label">排序规则</span>
            <div class="home-sort">
              <button class="sort-btn" :class="{ 'sort-btn--active': sortBy === 'latest' }" @click="sortBy = 'latest'">
                最近发布
              </button>
              <button class="sort-btn" :class="{ 'sort-btn--active': sortBy === 'popular' }" @click="sortBy = 'popular'">
                热门推荐
              </button>
            </div>
          </div>

          <!-- 分类 -->
          <div class="home-bar">
            <span class="home-bar__label">分类</span>
            <div class="category-bar">
              <button
                class="category-tag"
                :class="{ 'category-tag--active': activeCategory === null }"
                @click="activeCategory = null"
              >
                全部
              </button>
              <button
                v-for="cat in categories"
                :key="cat.id"
                class="category-tag"
                :class="{ 'category-tag--active': activeCategory === cat.id }"
                @click="activeCategory = cat.id"
              >
                {{ cat.name }}
              </button>
            </div>
          </div>

          <!-- 标签（多选） -->
          <div class="home-bar">
            <span class="home-bar__label">标签</span>
            <div class="category-bar">
              <button
                v-for="tag in tags"
                :key="tag.id"
                class="category-tag"
                :class="{ 'category-tag--active': activeTags.includes(tag.id) }"
                @click="toggleTag(tag.id)"
              >
                {{ tag.name }}
              </button>
            </div>
          </div>

          <!-- 文章列表 -->
          <div v-if="filteredArticles.length" class="article-list">
            <article
              v-for="item in filteredArticles"
              :key="item.id"
              class="article-card"
              @click="goDetail(item.id)"
            >
              <div class="article-card__body">
                <h3 class="article-card__title">{{ item.title }}</h3>
                <p class="article-card__summary">{{ item.summary || item.content.slice(0, 120) }}</p>
                <div class="article-card__meta">
                  <span class="article-card__meta-item">&#128100; {{ item.author }}</span>
                  <span class="article-card__meta-item">&#128197; {{ fmt(item.created_at) }}</span>
                  <span class="article-card__meta-item">&#128065; {{ item.views }}</span>
                  <span class="article-card__meta-item">&#128077; {{ item.likes }}</span>
                </div>
              </div>
              <img
                v-if="item.cover_url"
                :src="item.cover_url"
                :alt="item.title"
                class="article-card__cover"
              />
            </article>
          </div>
          <div v-else class="article-empty">暂无文章</div>
        </div>

        <aside class="home-sidebar">
          <div class="author-card">
            <img
              v-if="author.avatar_url"
              :src="author.avatar_url"
              class="author-card__avatar"
            />
            <span v-else class="author-card__avatar author-card__avatar--placeholder">&#128100;</span>
            <h3 class="author-card__name">{{ author.nickname || '博主' }}</h3>
            <p class="author-card__bio">{{ author.bio || '这个人很懒，什么都没写~' }}</p>
            <div class="author-card__stats">
              <div class="author-card__stat">
                <span class="author-card__stat-num">{{ author.article_count }}</span>
                <span class="author-card__stat-label">文章</span>
              </div>
              <div class="author-card__stat">
                <span class="author-card__stat-num">{{ author.total_views }}</span>
                <span class="author-card__stat-label">总浏览</span>
              </div>
              <div class="author-card__stat">
                <span class="author-card__stat-num">{{ author.total_likes }}</span>
                <span class="author-card__stat-label">总点赞</span>
              </div>
            </div>
          </div>
        </aside>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
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
  tags?: number[]
  created_at: string
}

interface Category {
  id: number
  name: string
}

interface Tag {
  id: number
  name: string
}

interface Author {
  avatar_url: string
  nickname: string
  bio: string
  article_count: number
  total_views: number
  total_likes: number
}

const router = useRouter()
const articles = ref<Article[]>([])
const categories = ref<Category[]>([])
const tags = ref<Tag[]>([])
const searchQuery = ref('')
const sortBy = ref<'latest' | 'popular'>('latest')
const activeCategory = ref<number | null>(null)
const activeTags = ref<number[]>([])

const author = ref<Author>({
  avatar_url: '',
  nickname: '博主',
  bio: '',
  article_count: 0,
  total_views: 0,
  total_likes: 0
})

const toggleTag = (id: number) => {
  const idx = activeTags.value.indexOf(id)
  if (idx >= 0) {
    activeTags.value.splice(idx, 1)
  } else {
    activeTags.value.push(id)
  }
}

const filteredArticles = computed(() => {
  let list = [...articles.value]

  if (activeCategory.value !== null) {
    list = list.filter((a) => a.type_id === activeCategory.value)
  }

  if (activeTags.value.length > 0) {
    list = list.filter((a) => {
      if (!a.tags || a.tags.length === 0) return false
      return activeTags.value.some((tid) => a.tags!.includes(tid))
    })
  }

  if (searchQuery.value.trim()) {
    const q = searchQuery.value.trim().toLowerCase()
    list = list.filter((a) => a.title.toLowerCase().includes(q))
  }

  if (sortBy.value === 'popular') {
    list.sort((a, b) => b.views - a.views)
  } else {
    list.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
  }

  return list
})

const fmt = (d: string) => {
  if (!d) return ''
  return new Date(d).toLocaleDateString('zh-CN')
}

const goDetail = (id: number) => {
  router.push(`/article/${id}`)
}

const onSearch = (q: string) => {
  searchQuery.value = q
}

onMounted(async () => {
  try {
    const [aRes, cRes, tRes, auRes] = await Promise.all([
      api.get('/articles'),
      api.get('/categories'),
      api.get('/tags'),
      api.get('/author/info')
    ])
    articles.value = aRes.data.data || aRes.data || []
    categories.value = cRes.data.data || cRes.data || []
    tags.value = tRes.data.data || tRes.data || []
    if (auRes.data.data) {
      author.value = { ...author.value, ...auRes.data.data }
    }
  } catch {
    articles.value = [
      { id: 1, title: 'Go 语言并发编程深入解析', content: 'Go 语言的 goroutine 和 channel 是其并发模型的核心...', cover_url: '', summary: '深入理解 goroutine 调度、channel 通信以及并发模式的最佳实践。', views: 2340, likes: 186, author: '张三', type_id: 1, tags: [1, 3], created_at: '2026-06-10T08:00:00Z' },
      { id: 2, title: 'Vue 3 Composition API 实战指南', content: 'Vue 3 带来了全新的 Composition API...', cover_url: '', summary: '从 setup 到响应式 API，全面掌握 Vue 3 的组合式开发方式。', views: 1850, likes: 142, author: '李四', type_id: 2, tags: [2], created_at: '2026-06-08T10:30:00Z' },
      { id: 3, title: '构建高性能 RESTful API', content: '设计一个高性能的 RESTful API 需要考虑诸多因素...', cover_url: '', summary: '涵盖路由设计、中间件、数据库优化和缓存策略。', views: 3120, likes: 257, author: '王五', type_id: 3, tags: [3, 4], created_at: '2026-06-12T14:00:00Z' },
      { id: 4, title: 'Docker 容器化部署最佳实践', content: '使用 Docker 进行应用容器化是现代 DevOps 的基石...', cover_url: '', summary: '从 Dockerfile 编写到 docker-compose 编排，一站式部署指南。', views: 1560, likes: 98, author: '张三', type_id: 4, tags: [5], created_at: '2026-06-05T09:00:00Z' },
      { id: 5, title: 'MySQL 索引优化详解', content: '索引是数据库性能优化的核心手段...', cover_url: '', summary: '深入 B+Tree 原理，掌握索引设计、优化和慢查询分析。', views: 2780, likes: 203, author: '李四', type_id: 3, tags: [3, 6], created_at: '2026-06-11T16:20:00Z' }
    ]
    categories.value = [
      { id: 1, name: 'Go语言' },
      { id: 2, name: '前端' },
      { id: 3, name: '后端' },
      { id: 4, name: '运维' }
    ]
    tags.value = [
      { id: 1, name: '并发' },
      { id: 2, name: 'Vue' },
      { id: 3, name: '数据库' },
      { id: 4, name: 'API设计' },
      { id: 5, name: 'Docker' },
      { id: 6, name: 'MySQL' }
    ]
    author.value = { avatar_url: '', nickname: '博主', bio: '', article_count: 5, total_views: 11650, total_likes: 886 }
  }
})
</script>

<style scoped>
.home-layout {
  display: grid;
  grid-template-columns: 1fr 260px;
  gap: 24px;
  align-items: start;
}

.home-main { min-width: 0; }

.home-bar {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  margin-bottom: 10px;
}

.home-bar__label {
  font-size: 0.82rem;
  font-weight: 700;
  color: var(--text-muted);
  flex-shrink: 0;
  min-width: 50px;
  line-height: 38px;
}

.home-sort {
  display: flex;
  gap: 6px;
}

.home-sidebar { position: sticky; top: 76px; }

.author-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 24px 18px 20px;
  border: 1px solid rgba(255,255,255,0.55);
  border-radius: var(--radius-lg);
  background: rgba(255,255,255,0.8);
  backdrop-filter: blur(12px);
  box-shadow: 0 4px 18px rgba(15,23,42,0.04);
}

.author-card__avatar {
  width: 72px; height: 72px;
  border-radius: 50%;
  object-fit: cover;
  border: 3px solid rgba(16,185,129,0.18);
  margin-bottom: 10px;
}

.author-card__avatar--placeholder {
  display: inline-flex;
  align-items: center; justify-content: center;
  background: var(--primary-soft);
  color: var(--primary-700);
  font-size: 1.6rem;
  border: none;
}

.author-card__name {
  font-size: 1.05rem;
  font-weight: 700;
  color: var(--text-strong);
  margin: 0 0 8px;
}

.author-card__bio {
  font-size: 0.82rem;
  color: var(--text-muted);
  text-align: center;
  line-height: 1.5;
  margin: 0 0 16px;
  padding: 0 4px;
}

.author-card__stats {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  width: 100%;
}

.author-card__stat {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 10px 4px;
  border-radius: 10px;
  background: rgba(248,250,252,0.7);
}

.author-card__stat-num {
  font-size: 1.1rem;
  font-weight: 800;
  color: var(--primary-700);
}

.author-card__stat-label {
  font-size: 0.74rem;
  color: var(--text-faint);
  margin-top: 2px;
}

@media (max-width: 768px) {
  .home-layout {
    grid-template-columns: 1fr;
  }
  .home-sidebar { position: static; order: -1; }
}
</style>
