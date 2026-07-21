<template>
  <div class="home-page">
    <NavBar />
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
            <div class="category-bar" ref="categoryBarRef">
              <button
                class="category-tag"
                :class="{ 'category-tag--active': activeCategory === null }"
                @click="changeCategory(null)"
              >
                全部
              </button>
              <button
                v-for="(cat, index) in categories"
                :key="cat.id"
                class="category-tag"
                :class="{
                  'category-tag--active': activeCategory === cat.id,
                  'category-tag--hidden': !categoryExpanded && categoryHasMore && index >= categoryVisibleCount
                }"
                @click="changeCategory(cat.id)"
              >
                {{ cat.name }}
              </button>
              <button v-if="categoryHasMore" class="category-tag category-toggle" @click="toggleCategory">
                {{ categoryExpanded ? '收起' : '更多' }}
              </button>
            </div>
          </div>

          <!-- 标签（多选） -->
          <div class="home-bar">
            <span class="home-bar__label">标签</span>
            <div class="category-bar" ref="tagBarRef">
              <button
                v-for="(tag, index) in tags"
                :key="tag.id"
                class="category-tag"
                :class="{
                  'category-tag--active': activeTags.includes(tag.id),
                  'category-tag--hidden': !tagExpanded && tagHasMore && index >= tagVisibleCount
                }"
                @click="toggleTag(tag.id)"
              >
                {{ tag.name }}
              </button>
              <button v-if="tagHasMore" class="category-tag category-toggle" @click="toggleTagSection">
                {{ tagExpanded ? '收起' : '更多' }}
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
                @error="handleImgError"
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
          <div class="sidebar-card">
            <div class="sidebar-avatar">
              <img
                v-if="author.avatar_url"
                :src="author.avatar_url"
                class="sidebar-avatar__img"
              />
              <span v-else class="sidebar-avatar__placeholder">&#128100;</span>
            </div>
            <h3 class="sidebar-name">{{ author.admin_name || '邹鑫鹏' }}</h3>
            <p class="sidebar-title">{{ author.introduction || 'Backend Developer' }}</p>
            
            
            <div v-if="author.tech_stack" class="sidebar-tags">
              <span v-for="tech in author.tech_stack.split(',')" :key="tech" class="sidebar-tag">{{ tech.trim() }}</span>
            </div>
            
            <div class="sidebar-stats">
              <div class="sidebar-stat">
                <span class="sidebar-stat__value">{{ stats.article_count }}</span>
                <span class="sidebar-stat__label">文章</span>
              </div>
              <div class="sidebar-stat">
                <span class="sidebar-stat__value">{{ formatNumber(stats.total_views) }}</span>
                <span class="sidebar-stat__label">阅读</span>
              </div>
              <div class="sidebar-stat">
                <span class="sidebar-stat__value">{{ stats.total_likes }}</span>
                <span class="sidebar-stat__label">点赞</span>
              </div>
            </div>
            
            <div class="sidebar-section">
              <h4 class="sidebar-section__title">最近文章</h4>
              <div class="sidebar-articles">
                <div v-for="article in sidebarArticles" :key="article.id" class="sidebar-article" @click="goDetail(article.id)">
                  {{ article.title }}
                </div>
                <div v-if="!sidebarArticles.length" class="sidebar-article sidebar-article--empty">暂无文章</div>
              </div>
            </div>
            
            <div class="sidebar-motto">
              <p class="sidebar-motto__text">Stay Hungry.</p>
              <p class="sidebar-motto__text">Stay Foolish.</p>
            </div>
          </div>
        </aside>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
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
  admin_name: string
  introduction: string
  tech_stack: string
}

interface ArticleStats {
  article_count: number
  total_views: number
  total_likes: number
}

const router = useRouter()
const route = useRoute()
const articles = ref<Article[]>([])
const sidebarArticles = ref<Article[]>([])  // 侧边栏最近文章
const stats = ref<ArticleStats>({
  article_count: 0,
  total_views: 0,
  total_likes: 0
})
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

// 展开/收起状态
const categoryBarRef = ref<HTMLElement | null>(null)
const tagBarRef = ref<HTMLElement | null>(null)
const categoryExpanded = ref(false)
const tagExpanded = ref(false)
const categoryHasMore = ref(false)
const tagHasMore = ref(false)
const categoryVisibleCount = ref(0)
const tagVisibleCount = ref(0)

const measureAndCollapse = (container: HTMLElement | null, expanded: boolean, hasMore: { value: boolean }, visibleCount: { value: number }, extraBefore: number) => {
  if (!container) return
  const allTags = Array.from(container.querySelectorAll('.category-tag')) as HTMLElement[]
  const toggle = container.querySelector('.category-toggle') as HTMLElement | null
  const realTags = allTags.filter(t => t !== toggle)

  if (realTags.length < 2) {
    hasMore.value = false
    visibleCount.value = realTags.length
    return
  }

  if (expanded) {
    hasMore.value = true
    visibleCount.value = realTags.length
    return
  }

  // 第一次测量：所有标签可见
  // 找到首行项目数
  const firstTop = realTags[0].offsetTop
  const firstLineCount = realTags.filter(t => t.offsetTop === firstTop).length

  if (firstLineCount < realTags.length) {
    hasMore.value = true
    // 保留位置：extraBefore 个固定按钮（如"全部"）+ 1 个 toggle
    visibleCount.value = Math.max(0, firstLineCount - extraBefore - 1)
  } else {
    hasMore.value = false
    visibleCount.value = realTags.length
  }
}

const updateOverflow = () => {
  measureAndCollapse(categoryBarRef.value, categoryExpanded.value, categoryHasMore, categoryVisibleCount, 1)
  measureAndCollapse(tagBarRef.value, tagExpanded.value, tagHasMore, tagVisibleCount, 0)
}

const toggleCategory = () => {
  categoryExpanded.value = !categoryExpanded.value
  nextTick(() => measureAndCollapse(categoryBarRef.value, categoryExpanded.value, categoryHasMore, categoryVisibleCount, 1))
}

const toggleTagSection = () => {
  tagExpanded.value = !tagExpanded.value
  nextTick(() => measureAndCollapse(tagBarRef.value, tagExpanded.value, tagHasMore, tagVisibleCount, 0))
}

watch([categories, tags], () => {
  nextTick(updateOverflow)
})

const PAGE_SIZE = 10

const author = ref<Author>({
  avatar_url: '',
  admin_name: '博主',
  introduction: '',
  tech_stack: ''
})

const formatNumber = (num: number) => {
  if (num >= 1000) {
    return (num / 1000).toFixed(1) + 'K'
  }
  return num.toString()
}

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
    const list = (data.list || data || []).map((item: any) => {
      // 保留原始数据，只添加兼容字段
      return {
        ...item,
        views: item.view_count ?? item.views ?? 0,
        likes: item.like_count ?? item.likes ?? 0,
        author: item.author_name ?? item.author ?? '',
        cover_url: item.cover_url && item.cover_url.trim() ? item.cover_url : null
      }
    })

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

  const wasLiked = !!item.is_liked
  const oldLikeCount = item.like_count || item.likes || 0

  // 乐观更新 UI
  item.is_liked = !wasLiked
  if (item.like_count !== undefined) {
    item.like_count = wasLiked ? oldLikeCount - 1 : oldLikeCount + 1
  }
  if (item.likes !== undefined) {
    item.likes = wasLiked ? oldLikeCount - 1 : oldLikeCount + 1
  }

  try {
    if (wasLiked) {
      await api.delete(`/articles/${item.id}/like`)
    } else {
      await api.post(`/articles/${item.id}/like`)
    }
  } catch (error) {
    console.error('点赞操作失败:', error)
    // 失败回滚
    item.is_liked = wasLiked
    if (item.like_count !== undefined) {
      item.like_count = oldLikeCount
    }
    if (item.likes !== undefined) {
      item.likes = oldLikeCount
    }
  }
}

const fmt = (d: string) => {
  if (!d) return ''
  return new Date(d).toLocaleDateString('zh-CN')
}

const goDetail = (id: number) => {
  router.push(`/article/${id}`)
}

const handleImgError = (e: Event) => {
  const el = e.target as HTMLElement | null
  if (el) el.style.display = 'none'
}


// 加载侧边栏最近文章（按时间排序，最新的在前面）
const loadSidebarArticles = async () => {
  try {
    const res = await api.get('/articles', {
      params: {
        page: 1,
        page_size: 5,
        sort: 'latest'
      }
    })
    const data = res.data.data || res.data
    const list = (data.list || data || []).map((item: any) => {
      return {
        ...item,
        views: item.view_count ?? item.views ?? 0,
        likes: item.like_count ?? item.likes ?? 0,
        author: item.author_name ?? item.author ?? '',
        cover_url: item.cover_url && item.cover_url.trim() ? item.cover_url : null
      }
    })
    sidebarArticles.value = list
  } catch (error) {
    console.error('加载侧边栏文章失败:', error)
    sidebarArticles.value = []
  }
}

// 加载统计数据（文章统计、标签数量、分类数量）
const loadStats = async () => {
  try {
    const res = await api.get('/articles/stats')
    const data = res.data.data || res.data
    stats.value = {
      article_count: data.article_count ?? 0,
      total_views: data.total_views ?? 0,
      total_likes: data.total_likes ?? 0
    }
  } catch (error) {
    console.error('加载文章统计数据失败:', error)
  }
}

onMounted(async () => {
  // 独立加载分类、标签、作者信息（互不影响）
  try {
    const cRes = await api.get('/categories')
    categories.value = cRes.data?.data || []
  } catch {
    // 分类API失败，不降级，保持空数组
  }

  try {
    const tRes = await api.get('/tags')
    tags.value = tRes.data?.data || []
  } catch {
    // 标签API失败，不降级，保持空数组
  }

  try {
    const aboutRes = await api.get('/about')
    if (aboutRes.data.data) {
      author.value = { ...author.value, ...aboutRes.data.data }
    }
  } catch (err) {
    console.log('获取关于页面信息失败，使用默认值')
  }

  // 加载文章统计数据
  loadStats()
  // 加载侧边栏最近文章
  loadSidebarArticles()
  // 从路由参数读取搜索词
  if (route.query.q) {
    searchQuery.value = route.query.q as string
  }
  // 加载文章列表
  loadArticles()
})

// 监听路由搜索参数变化（用户在其他页面搜索跳转过来）
watch(() => route.query.q, (q) => {
  searchQuery.value = q || ''
  resetAndLoad()
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

.category-tag--hidden {
  display: none;
}

.category-toggle {
  color: var(--primary-600) !important;
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


/* ========== Sidebar ========== */
.home-sidebar { position: sticky; top: 76px; }

.sidebar-card {
  background: rgba(255, 255, 255, 0.85);
  border: 1px solid rgba(255, 255, 255, 0.55);
  border-radius: var(--radius-lg);
  padding: 24px 20px;
  backdrop-filter: blur(12px);
  box-shadow: 0 4px 18px rgba(15, 23, 42, 0.04);
}

.sidebar-avatar {
  width: 80px;
  height: 80px;
  margin: 0 auto 16px;
  border-radius: 50%;
  border: 3px solid rgba(16, 185, 129, 0.3);
  overflow: hidden;
}

.sidebar-avatar__img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.sidebar-avatar__placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  font-size: 2rem;
  background: linear-gradient(135deg, var(--primary-100), var(--primary-200));
  color: var(--primary-600);
}

.sidebar-name {
  font-size: 1.3rem;
  font-weight: 700;
  color: var(--text-strong);
  margin: 0 0 8px;
  text-align: center;
}

.sidebar-title {
  font-size: 0.95rem;
  color: var(--text-secondary);
  margin: 0 0 16px;
  text-align: center;
  font-weight: 500;
}

.sidebar-location {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  margin-bottom: 16px;
}

.sidebar-location__icon {
  font-size: 1rem;
}

.sidebar-location__text {
  font-size: 0.9rem;
  color: var(--text-secondary);
}

.sidebar-tags {
  display: flex;
  justify-content: center;
  gap: 6px;
  margin-bottom: 20px;
  flex-wrap: wrap;
}

.sidebar-tag {
  padding: 4px 10px;
  border-radius: 12px;
  background: rgba(16, 185, 129, 0.1);
  color: var(--primary-700);
  font-size: 0.8rem;
  font-weight: 600;
}

.sidebar-stats {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  margin-bottom: 20px;
}

.sidebar-stat {
  text-align: center;
  padding: 12px 8px;
  background: rgba(248, 250, 252, 0.7);
  border-radius: 10px;
}

.sidebar-stat__value {
  display: block;
  font-size: 1.1rem;
  font-weight: 800;
  color: var(--primary-700);
  margin-bottom: 2px;
}

.sidebar-stat__label {
  font-size: 0.7rem;
  color: var(--text-faint);
  font-weight: 600;
}

.sidebar-section {
  margin-bottom: 20px;
}

.sidebar-section__title {
  font-size: 0.9rem;
  font-weight: 700;
  color: var(--text-strong);
  margin: 0 0 10px;
}

.sidebar-articles {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.sidebar-article {
  padding: 8px 12px;
  background: rgba(248, 250, 252, 0.6);
  border-radius: 8px;
  font-size: 0.85rem;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
}

.sidebar-article:hover {
  background: rgba(16, 185, 129, 0.1);
  color: var(--primary-700);
}

.sidebar-update {
  font-size: 0.9rem;
  color: var(--text-secondary);
  font-weight: 600;
}

.sidebar-motto {
  text-align: center;
  padding-top: 16px;
  border-top: 1px solid var(--border);
}

.sidebar-motto__text {
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--text-secondary);
  margin: 0 0 4px;
  font-style: italic;
}

@media (max-width: 768px) {
  .home-layout { grid-template-columns: 1fr; }
  .home-sidebar { position: static; order: -1; }
}
</style>