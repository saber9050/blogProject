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
              <button class="sort-btn" :class="{ 'sort-btn--active': sortBy === 'latest' }" @click="changeSort('latest')">
                最近发布
              </button>
              <button class="sort-btn" :class="{ 'sort-btn--active': sortBy === 'popular' }" @click="changeSort('popular')">
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
                @click="changeCategory(null)"
              >
                全部
              </button>
              <button
                v-for="cat in categories"
                :key="cat.id"
                class="category-tag"
                :class="{ 'category-tag--active': activeCategory === cat.id }"
                @click="changeCategory(cat.id)"
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
          <div v-if="articles.length" class="article-list">
            <article
              v-for="item in articles"
              :key="item.id"
              class="article-card"
              @click="goDetail(item.id)"
            >
              <div class="article-card__body">
                <h3 class="article-card__title">{{ item.title }}</h3>
                <p class="article-card__summary">{{ item.summary || '' }}</p>
                <div class="article-card__meta">
                  <span class="article-card__meta-item">&#128100; {{ item.author_name || item.author }}</span>
                  <span class="article-card__meta-item">&#128197; {{ fmt(item.created_at) }}</span>
                  <span class="article-card__meta-item">&#128065; {{ item.view_count || item.views }}</span>
                  <button
                    class="article-card__meta-item article-card__like-btn"
                    :class="{ 'article-card__like-btn--liked': item.is_liked }"
                    @click.stop="toggleLike(item)"
                  >
                    <span v-if="item.is_liked" class="like-icon like-icon--filled">&#10084;</span>
                    <span v-else class="like-icon">&#9825;</span>
                    {{ item.like_count || item.likes || 0 }}
                  </button>
                  <span class="article-card__meta-item">&#128172; {{ item.comment_count || 0 }}</span>
                </div>
              </div>
              <img
                v-if="item.cover_url"
                :src="item.cover_url"
                :alt="item.title"
                class="article-card__cover"
              />
            </article>

            <!-- 加载更多 -->
            <div v-if="hasMore" class="article-load-more">
              <button
                class="comment-btn"
                :disabled="loadingMore"
                @click="loadNextPage"
              >
                {{ loadingMore ? '加载中...' : '加载更多' }}
              </button>
            </div>
          </div>
          <div v-else-if="!loading" class="article-empty">暂无文章</div>
          <div v-else class="article-empty">加载中...</div>
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
import { ref, onMounted } from 'vue'
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
  view_count: number
  likes: number
  like_count: number
  comment_count: number
  is_liked: boolean
  author: string
  author_name: string
  author_avatar: string
  type_id: number
  category?: { id: number; name: string }
  tags?: { id: number; name: string }[]
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
const currentPage = ref(1)
const totalCount = ref(0)
const hasMore = ref(false)
const loading = ref(false)
const loadingMore = ref(false)
const searchQuery = ref('')
const sortBy = ref<'latest' | 'popular'>('latest')
const activeCategory = ref<number | null>(null)
const activeTags = ref<number[]>([])

const PAGE_SIZE = 10

const author = ref<Author>({
  avatar_url: '',
  nickname: '博主',
  bio: '',
  article_count: 0,
  total_views: 0,
  total_likes: 0
})

// 获取当前登录用户信息
const getCurrentUser = () => {
  try {
    const raw = localStorage.getItem('user')
    return raw ? JSON.parse(raw) : null
  } catch {
    return null
  }
}

const toggleTag = (id: number) => {
  const idx = activeTags.value.indexOf(id)
  if (idx >= 0) {
    activeTags.value.splice(idx, 1)
  } else {
    activeTags.value.push(id)
  }
  resetAndLoad()
}

const changeSort = (s: 'latest' | 'popular') => {
  sortBy.value = s
  resetAndLoad()
}

const changeCategory = (id: number | null) => {
  activeCategory.value = id
  resetAndLoad()
}

const resetAndLoad = () => {
  currentPage.value = 1
  articles.value = []
  hasMore.value = false
  loadArticles()
}

const buildParams = () => {
  const params: Record<string, any> = {
    page: currentPage.value,
    page_size: PAGE_SIZE,
    sort: sortBy.value
  }
  if (activeCategory.value !== null) {
    params.category_id = activeCategory.value
  }
  if (activeTags.value.length > 0) {
    params.tag_ids = activeTags.value.join(',')
  }
  if (searchQuery.value.trim()) {
    params.keyword = searchQuery.value.trim()
  }
  return params
}

const loadArticles = async () => {
  if (currentPage.value === 1) {
    loading.value = true
  } else {
    loadingMore.value = true
  }

  try {
    const res = await api.get('/articles', { params: buildParams() })
    const data = res.data.data || res.data
    const list = (data.list || data || []).map((item: any) => ({
      ...item,
      // 兼容旧字段名
      views: item.view_count ?? item.views ?? 0,
      likes: item.like_count ?? item.likes ?? 0,
      author: item.author_name ?? item.author ?? ''
    }))

    if (currentPage.value === 1) {
      articles.value = list
    } else {
      articles.value.push(...list)
    }

    totalCount.value = data.total || list.length
    hasMore.value = currentPage.value * PAGE_SIZE < totalCount.value
  } catch {
    // API 失败，保持现有数据
    if (currentPage.value === 1) {
      articles.value = []
    }
    hasMore.value = false
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

const loadNextPage = () => {
  currentPage.value++
  loadArticles()
}

// 点赞/取消点赞（乐观更新）
const toggleLike = async (item: Article) => {
  const user = getCurrentUser()
  if (!user) {
    router.push('/login')
    return
  }

  const wasLiked = item.is_liked
  const oldLikeCount = item.like_count || item.likes || 0

  // 乐观更新 UI
  item.is_liked = !wasLiked
  if (item.like_count !== undefined) item.like_count += wasLiked ? -1 : 1
  if (item.likes !== undefined) item.likes += wasLiked ? -1 : 1

  try {
    if (wasLiked) {
      await api.delete(`/articles/${item.id}/like`)
    } else {
      await api.post(`/articles/${item.id}/like`)
    }
  } catch {
    // 失败回滚
    item.is_liked = wasLiked
    if (item.like_count !== undefined) item.like_count = oldLikeCount
    if (item.likes !== undefined) item.likes = oldLikeCount
  }
}

const fmt = (d: string) => {
  if (!d) return ''
  return new Date(d).toLocaleDateString('zh-CN')
}

const goDetail = (id: number) => {
  router.push(`/article/${id}`)
}

const onSearch = (q: string) => {
  searchQuery.value = q
  resetAndLoad()
}

onMounted(async () => {
  // 独立加载分类、标签、作者信息（互不影响）
  try {
    const cRes = await api.get('/categories')
    categories.value = cRes.data.data || cRes.data || []
  } catch {
    // 分类API失败，不降级，保持空数组
  }

  try {
    const tRes = await api.get('/tags')
    tags.value = tRes.data.data || tRes.data || []
  } catch {
    // 标签API失败，不降级，保持空数组
  }

  try {
    const auRes = await api.get('/user/info')
    if (auRes.data.data) {
      author.value = { ...author.value, ...auRes.data.data }
    }
  } catch (err) {
    // 用户未登录或API失败，使用默认作者信息
    console.log('用户未登录或获取作者信息失败，使用默认值')
  }

  // // 加载文章列表
  loadArticles()
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

.home-sort { display: flex; gap: 6px; }

.home-sidebar { position: sticky; top: 76px; }

/* ========== 点赞按钮（卡片内） ========== */
.article-card__like-btn {
  background: none;
  border: none;
  padding: 0;
  cursor: pointer;
  transition: color 0.2s ease, transform 0.15s ease;
}

.article-card__like-btn:hover {
  transform: scale(1.18);
  color: var(--danger);
}

.article-card__like-btn--liked {
  color: var(--danger);
}

.like-icon {
  font-size: 0.95rem;
  color: var(--text-faint);
  transition: color 0.2s ease;
}

.like-icon--filled {
  color: var(--danger);
}

/* ========== 加载更多 ========== */
.article-load-more {
  display: flex;
  justify-content: center;
  padding: 16px 0 8px;
}

.comment-btn {
  padding: 8px 24px;
  border: 1px solid var(--border-strong);
  border-radius: 10px;
  background: rgba(255,255,255,0.8);
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
}

.comment-btn:hover {
  border-color: var(--primary-400);
  color: var(--primary-700);
  background: rgba(16,185,129,0.06);
}

.comment-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* ========== 作者卡片 ========== */
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
  .home-layout { grid-template-columns: 1fr; }
  .home-sidebar { position: static; order: -1; }
}
</style>