<template>
  <div class="about-page">
    <NavBar />
    <main class="page-shell">
      <!-- Hero Banner -->
      <div class="hero-banner">
        <div class="hero-content">
          <div class="hero-avatar">
            <img
              v-if="aboutInfo.avatar_url"
              :src="aboutInfo.avatar_url"
              class="hero-avatar__img"
            />
            <span v-else class="hero-avatar__placeholder">&#128100;</span>
          </div>
          <h1 class="hero-name">{{ aboutInfo.admin_name }}</h1>
          <p v-if="aboutInfo.introduction" class="hero-quote">"{{ aboutInfo.introduction }}"</p>
          <!-- Display: tech_stack tags -->
          <div v-if="!isEditing && aboutInfo.tech_stack" class="hero-tags">
            <span v-for="tech in aboutInfo.tech_stack.split(',')" :key="tech" class="hero-tag">{{ tech.trim() }}</span>
          </div>
          <!-- Edit: tech_stack input -->
          <div v-if="isEditing" class="hero-field">
            <label class="edit-label">技术栈</label>
            <input v-model="editForm.tech_stack" class="edit-input" placeholder="多个用逗号隔开，如: Go,Vue,AI" />
          </div>
          <div class="hero-social">
            <a v-if="!isEditing && aboutInfo.git_hub" :href="aboutInfo.git_hub" class="hero-social-btn" target="_blank">
              <span>&#128241;</span> GitHub
            </a>
            <a v-if="!isEditing && aboutInfo.email" :href="'mailto:' + aboutInfo.email" class="hero-social-btn">
              <span>&#9993;</span> Email
            </a>
            <a v-if="!isEditing && aboutInfo.csdn" :href="aboutInfo.csdn" class="hero-social-btn" target="_blank">
              <span>&#128187;</span> CSDN
            </a>
          </div>
          <!-- Edit Button -->
          <div v-if="isAdmin && !isEditing" class="hero-edit-area">
            <button @click="enterEditMode" class="hero-edit-btn">&#9998; 编辑页面</button>
          </div>
        </div>
      </div>

      <div class="about-layout">
        <div class="about-main">
          <!-- 我的故事 -->
          <div class="about-section">
            <h2 class="about-section__title"><span class="section-icon">&#128214;</span> 我的故事</h2>
            <div v-if="!isEditing" class="about-section__content">
              <div class="about-text" style="white-space: pre-wrap;">{{ aboutInfo.my_story }}</div>
            </div>
            <div v-else class="about-section__content">
              <textarea v-model="editForm.my_story" class="edit-textarea" rows="8" placeholder="输入我的故事（支持 Markdown 格式）"></textarea>
            </div>
          </div>

          <!-- 博客理念 -->
          <div class="about-section">
            <h2 class="about-section__title"><span class="section-icon">&#128196;</span> 为什么建立博客？</h2>
            <div v-if="!isEditing" class="about-section__content">
              <div class="philosophy-content">
                <p class="philosophy-text" style="white-space: pre-wrap;">{{ aboutInfo.why }}</p>
              </div>
            </div>
            <div v-else class="about-section__content">
              <textarea v-model="editForm.why" class="edit-textarea" rows="6" placeholder="输入为什么建立博客"></textarea>
            </div>
          </div>

          <!-- 兴趣领域 -->
          <div class="about-section">
            <h2 class="about-section__title"><span class="section-icon">&#10084;</span> 我感兴趣</h2>
            <div v-if="!isEditing" class="about-section__content">
              <div v-if="aboutInfo.interest" class="interest-tags">
                <span v-for="item in aboutInfo.interest.split(',')" :key="item" class="interest-tag">{{ item.trim() }}</span>
              </div>
            </div>
            <div v-else class="about-section__content">
              <label class="edit-label">感兴趣的技术（多个用逗号隔开）</label>
              <input v-model="editForm.interest" class="edit-input" placeholder="如: Go,Redis,消息队列,系统设计" />
            </div>
          </div>

          <!-- 博客统计 -->
          <div class="about-section">
            <h2 class="about-section__title"><span class="section-icon">&#128202;</span> 博客数据</h2>
            <div class="about-section__content">
              <div class="stats-grid">
                <div class="stat-card">
                  <div class="stat-card__icon">&#128196;</div>
                  <div class="stat-card__value">{{ stats.article_count }}</div>
                  <div class="stat-card__label">文章</div>
                </div>
                <div class="stat-card">
                  <div class="stat-card__icon">&#128065;</div>
                  <div class="stat-card__value">{{ formatNumber(stats.total_views) }}</div>
                  <div class="stat-card__label">阅读</div>
                </div>
                <div class="stat-card">
                  <div class="stat-card__icon">&#10084;</div>
                  <div class="stat-card__value">{{ stats.total_likes }}</div>
                  <div class="stat-card__label">点赞</div>
                </div>
                <div class="stat-card">
                  <div class="stat-card__icon">&#128483;</div>
                  <div class="stat-card__value">{{ tagCount }}</div>
                  <div class="stat-card__label">标签</div>
                </div>
                <div class="stat-card">
                  <div class="stat-card__icon">&#128193;</div>
                  <div class="stat-card__value">{{ categoryCount }}</div>
                  <div class="stat-card__label">分类</div>
                </div>
              </div>
            </div>
          </div>

          <!-- 联系我 -->
          <div class="about-section">
            <h2 class="about-section__title"><span class="section-icon">&#128236;</span> 联系我</h2>
            <div v-if="!isEditing" class="about-section__content">
              <div class="contact-cards">
                <div v-if="aboutInfo.git_hub" class="contact-card">
                  <div class="contact-card__icon">&#128241;</div>
                  <h3 class="contact-card__title">GitHub</h3>
                  <p class="contact-card__desc">查看我的开源项目</p>
                  <a :href="aboutInfo.git_hub" class="contact-card__link" target="_blank">
                    访问主页 &rarr;
                  </a>
                </div>
                <div v-if="aboutInfo.email" class="contact-card">
                  <div class="contact-card__icon">&#9993;</div>
                  <h3 class="contact-card__title">Email</h3>
                  <div class="contact-card__desc">欢迎交流技术</div>
                  <a :href="'mailto:' + aboutInfo.email" class="contact-card__link">
                    发送邮件 &rarr;
                  </a>
                </div>
                <div v-if="aboutInfo.csdn" class="contact-card">
                  <div class="contact-card__icon">&#128187;</div>
                  <h3 class="contact-card__title">CSDN</h3>
                  <p class="contact-card__desc">查看更多文章</p>
                  <a :href="aboutInfo.csdn" class="contact-card__link" target="_blank">
                    访问主页 &rarr;
                  </a>
                </div>
              </div>
            </div>
            <div v-else class="about-section__content">
              <div class="edit-social">
                <label class="edit-label">GitHub 地址</label>
                <input v-model="editForm.git_hub" class="edit-input" placeholder="https://github.com/yourusername" />
                <label class="edit-label">CSDN 地址</label>
                <input v-model="editForm.csdn" class="edit-input" placeholder="https://blog.csdn.net/yourusername" />
              </div>
            </div>
          </div>

          <!-- 编辑操作栏 -->
          <div v-if="isEditing" class="edit-actions">
            <button @click="saveEdit" class="edit-btn edit-btn--save" :disabled="isSaving">
              {{ isSaving ? '保存中...' : '保存修改' }}
            </button>
            <button @click="cancelEdit" class="edit-btn edit-btn--cancel">取消</button>
          </div>

          <!-- Footer -->
          <div class="about-footer">
            <p class="footer-text">
              感谢你的阅读。<br>
            </p>
            <p class="footer-motto">
              Keep Learning.<br>
              Keep Coding.<br>
              <span class="footer-heart">&#10084;</span>
            </p>
          </div>
        </div>


      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import NavBar from '../components/NavBar.vue'
import api from '../api'
import { showToast } from '../utils/toast'

interface AboutInfo {
  admin_name: string
  avatar_url: string
  email: string
  introduction: string
  tech_stack: string
  my_story: string
  why: string
  interest: string
  git_hub: string
  csdn: string
}

interface ArticleStats {
  article_count: number
  total_views: number
  total_likes: number
}

const aboutInfo = ref<AboutInfo>({
  admin_name: '',
  avatar_url: '',
  email: '',
  introduction: '',
  tech_stack: '',
  my_story: '',
  why: '',
  interest: '',
  git_hub: '',
  csdn: ''
})

const stats = ref<ArticleStats>({
  article_count: 0,
  total_views: 0,
  total_likes: 0
})
const tagCount = ref(0)
const categoryCount = ref(0)

// 编辑模式
const isEditing = ref(false)
const isSaving = ref(false)

interface EditForm {
  tech_stack: string
  my_story: string
  why: string
  interest: string
  git_hub: string
  csdn: string
}

const editForm = ref<EditForm>({
  tech_stack: '',
  my_story: '',
  why: '',
  interest: '',
  git_hub: '',
  csdn: ''
})

let originalData: EditForm = {
  tech_stack: '',
  my_story: '',
  why: '',
  interest: '',
  git_hub: '',
  csdn: ''
}

const isAdmin = computed(() => {
  try {
    const user = JSON.parse(localStorage.getItem('user') || '{}')
    return user.role_id === 1
  } catch {
    return false
  }
})

const enterEditMode = () => {
  editForm.value = {
    tech_stack: aboutInfo.value.tech_stack,
    my_story: aboutInfo.value.my_story,
    why: aboutInfo.value.why,
    interest: aboutInfo.value.interest,
    git_hub: aboutInfo.value.git_hub,
    csdn: aboutInfo.value.csdn
  }
  originalData = { ...editForm.value }
  isEditing.value = true
}

const cancelEdit = () => {
  isEditing.value = false
}

const saveEdit = async () => {
  const payload: Record<string, string> = {}
  const fields = ['tech_stack', 'my_story', 'why', 'interest', 'git_hub', 'csdn'] as const
  for (const field of fields) {
    if (editForm.value[field] !== originalData[field]) {
      payload[field] = editForm.value[field]
    }
  }

  if (Object.keys(payload).length === 0) {
    isEditing.value = false
    return
  }

  isSaving.value = true
  try {
    await api.put('/about', payload)
    aboutInfo.value = { ...aboutInfo.value, ...editForm.value }
    isEditing.value = false
    showToast('保存成功', 'success')
  } catch (err) {
    showToast('保存失败', 'error')
  } finally {
    isSaving.value = false
  }
}

const formatNumber = (num: number) => {
  if (num >= 1000) {
    return (num / 1000).toFixed(1) + 'K'
  }
  return num.toString()
}

const loadStats = async () => {
  try {
    const res = await api.get('/articles/stats')
    const data = res.data.data || res.data
    stats.value = {
      article_count: data.article_count ?? 0,
      total_views: data.total_views ?? 0,
      total_likes: data.total_likes ?? 0
    }
  } catch (err) {
    console.log('获取文章统计数据失败')
  }
}

const loadTagCount = async () => {
  try {
    const res = await api.get('/tags/count')
    tagCount.value = (res.data.data || res.data).count ?? 0
  } catch (err) {
    console.log('获取标签数量失败')
  }
}

const loadCategoryCount = async () => {
  try {
    const res = await api.get('/categories/count')
    categoryCount.value = (res.data.data || res.data).count ?? 0
  } catch (err) {
    console.log('获取分类数量失败')
  }
}

onMounted(async () => {
  try {
    const res = await api.get('/about')
    if (res.data.data) {
      aboutInfo.value = { ...aboutInfo.value, ...res.data.data }
    }
  } catch (err) {
    console.log('获取关于页面信息失败，使用默认值')
  }

  // 加载博客数据统计
  loadStats()
  loadTagCount()
  loadCategoryCount()
})
</script>

<style scoped>
/* ========== Hero Banner ========== */
.hero-banner {
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.1), rgba(59, 130, 246, 0.1));
  border-radius: 24px;
  padding: 48px 32px;
  margin-bottom: 32px;
  backdrop-filter: blur(20px);
  border: 1px solid rgba(255, 255, 255, 0.3);
  box-shadow: 0 8px 32px rgba(15, 23, 42, 0.1);
  position: relative;
  overflow: hidden;
}

.hero-banner::before {
  content: '';
  position: absolute;
  top: -50%;
  left: -50%;
  width: 200%;
  height: 200%;
  background: radial-gradient(circle, rgba(16, 185, 129, 0.1) 0%, transparent 70%);
  animation: rotate 20s linear infinite;
}

@keyframes rotate {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.hero-content {
  position: relative;
  z-index: 1;
  text-align: center;
  max-width: 600px;
  margin: 0 auto;
}

.hero-avatar {
  width: 120px;
  height: 120px;
  margin: 0 auto 20px;
  border-radius: 50%;
  border: 4px solid rgba(16, 185, 129, 0.3);
  overflow: hidden;
  box-shadow: 0 8px 24px rgba(16, 185, 129, 0.2);
}

.hero-avatar__img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.hero-avatar__placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  font-size: 3rem;
  background: linear-gradient(135deg, var(--primary-100), var(--primary-200));
  color: var(--primary-600);
}

.hero-name {
  font-size: 2.5rem;
  font-weight: 800;
  color: var(--text-strong);
  margin: 0 0 12px;
  background: linear-gradient(135deg, var(--primary-600), var(--primary-500));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.hero-title {
  font-size: 1.1rem;
  color: var(--text-secondary);
  margin: 0 0 16px;
  font-weight: 500;
}

.hero-quote {
  font-size: 1.2rem;
  color: var(--text-primary);
  margin: 0 0 24px;
  font-style: italic;
  font-weight: 500;
}

.hero-tags {
  display: flex;
  justify-content: center;
  gap: 8px;
  margin-bottom: 24px;
  flex-wrap: wrap;
}

.hero-tag {
  padding: 8px 16px;
  border-radius: 20px;
  background: rgba(16, 185, 129, 0.1);
  color: var(--primary-700);
  font-weight: 600;
  font-size: 0.9rem;
  border: 1px solid rgba(16, 185, 129, 0.2);
}

.hero-social {
  display: flex;
  justify-content: center;
  gap: 12px;
}

.hero-social-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 10px 20px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.8);
  color: var(--text-secondary);
  text-decoration: none;
  font-weight: 600;
  font-size: 0.9rem;
  border: 1px solid rgba(255, 255, 255, 0.5);
  transition: all 0.3s ease;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.hero-social-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 16px rgba(16, 185, 129, 0.2);
  border-color: var(--primary-300);
  color: var(--primary-700);
}

/* ========== Layout ========== */
.about-layout {
  width: 100%;
}

.about-main {
  width: 100%;
}

/* ========== Sections ========== */
.about-section {
  background: rgba(255, 255, 255, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.55);
  border-radius: var(--radius-lg);
  padding: 28px;
  margin-bottom: 24px;
  backdrop-filter: blur(12px);
  box-shadow: 0 4px 18px rgba(15, 23, 42, 0.04);
  transition: transform 0.3s ease, box-shadow 0.3s ease;
}

.about-section:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.08);
}

.about-section__title {
  font-size: 1.35rem;
  font-weight: 700;
  color: var(--text-strong);
  margin: 0 0 20px;
  padding-bottom: 16px;
  border-bottom: 2px solid var(--primary-100);
  display: flex;
  align-items: center;
  gap: 10px;
}

.section-icon {
  font-size: 1.4rem;
}

.about-section__content {
  color: var(--text-secondary);
  line-height: 1.8;
}

.about-text {
  margin: 0 0 16px;
  font-size: 1rem;
  line-height: 1.9;
}

.about-text:last-child {
  margin-bottom: 0;
}

/* ========== Philosophy ========== */
.philosophy-content {
  text-align: center;
  padding: 20px;
}

.philosophy-text {
  font-size: 1.1rem;
  line-height: 2;
  color: var(--text-primary);
  margin: 0 0 20px;
  font-weight: 500;
}

.philosophy-text:last-child {
  margin-bottom: 0;
}

/* ========== Interest Tags ========== */
.interest-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.interest-tag {
  padding: 8px 16px;
  border-radius: 20px;
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.1), rgba(59, 130, 246, 0.1));
  color: var(--primary-700);
  font-weight: 600;
  font-size: 0.9rem;
  border: 1px solid rgba(16, 185, 129, 0.2);
  transition: all 0.3s ease;
}

.interest-tag:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(16, 185, 129, 0.2);
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.15), rgba(59, 130, 246, 0.15));
}

/* ========== Stats Grid ========== */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: 16px;
}

.stat-card {
  background: rgba(248, 250, 252, 0.8);
  border-radius: 16px;
  padding: 20px 16px;
  text-align: center;
  transition: all 0.3s ease;
  border: 1px solid rgba(255, 255, 255, 0.5);
}

.stat-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 20px rgba(16, 185, 129, 0.15);
}

.stat-card__icon {
  font-size: 1.8rem;
  margin-bottom: 8px;
}

.stat-card__value {
  font-size: 1.8rem;
  font-weight: 800;
  color: var(--primary-700);
  margin-bottom: 4px;
}

.stat-card__label {
  font-size: 0.85rem;
  color: var(--text-muted);
  font-weight: 600;
}

/* ========== Contact Cards ========== */
.contact-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
}

.contact-card {
  background: rgba(248, 250, 252, 0.8);
  border-radius: 16px;
  padding: 24px 20px;
  text-align: center;
  transition: all 0.3s ease;
  border: 1px solid rgba(255, 255, 255, 0.5);
}

.contact-card:hover {
  transform: translateY(-6px);
  box-shadow: 0 12px 28px rgba(16, 185, 129, 0.2);
  background: rgba(16, 185, 129, 0.03);
}

.contact-card__icon {
  font-size: 2.2rem;
  margin-bottom: 12px;
}

.contact-card__title {
  font-size: 1.1rem;
  font-weight: 700;
  color: var(--text-strong);
  margin: 0 0 8px;
}

.contact-card__desc {
  font-size: 0.85rem;
  color: var(--text-muted);
  margin: 0 0 16px;
}

.contact-card__link {
  display: inline-block;
  color: var(--primary-600);
  text-decoration: none;
  font-weight: 600;
  font-size: 0.9rem;
  transition: color 0.2s ease;
}

.contact-card__link:hover {
  color: var(--primary-700);
  text-decoration: underline;
}

/* ========== Footer ========== */
.about-footer {
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.08), rgba(59, 130, 246, 0.08));
  border-radius: 20px;
  padding: 32px 24px;
  text-align: center;
  margin-top: 32px;
  border: 1px solid rgba(255, 255, 255, 0.3);
}

.footer-text {
  font-size: 1.1rem;
  line-height: 2;
  color: var(--text-secondary);
  margin: 0 0 24px;
}

.footer-motto {
  font-size: 1.3rem;
  font-weight: 700;
  color: var(--primary-700);
  margin: 0;
  line-height: 1.8;
}

.footer-heart {
  color: var(--danger);
  font-size: 1.5rem;
}

/* ========== Edit Mode ========== */
.hero-edit-area {
  margin-top: 20px;
}

.hero-edit-btn {
  padding: 10px 24px;
  border-radius: 12px;
  border: 1px solid var(--primary-300);
  background: rgba(255, 255, 255, 0.9);
  color: var(--primary-700);
  font-weight: 600;
  font-size: 0.95rem;
  cursor: pointer;
  transition: all 0.3s ease;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.hero-edit-btn:hover {
  background: var(--primary-600);
  color: #fff;
  border-color: var(--primary-600);
  transform: translateY(-2px);
  box-shadow: 0 4px 16px rgba(16, 185, 129, 0.3);
}

.hero-field {
  margin-bottom: 20px;
  text-align: left;
}

.edit-label {
  display: block;
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text-secondary);
  margin-bottom: 6px;
}

.edit-input {
  width: 100%;
  padding: 10px 14px;
  border-radius: 10px;
  border: 1px solid var(--border);
  background: rgba(255, 255, 255, 0.9);
  color: var(--text-strong);
  font-size: 0.95rem;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
  outline: none;
  box-sizing: border-box;
}

.edit-input:focus {
  border-color: var(--primary-400);
  box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.15);
}

.edit-textarea {
  width: 100%;
  padding: 12px 14px;
  border-radius: 10px;
  border: 1px solid var(--border);
  background: rgba(255, 255, 255, 0.9);
  color: var(--text-strong);
  font-size: 0.95rem;
  line-height: 1.7;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
  outline: none;
  resize: vertical;
  font-family: inherit;
  box-sizing: border-box;
}

.edit-textarea:focus {
  border-color: var(--primary-400);
  box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.15);
}

.edit-social {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.edit-actions {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
  margin-top: 24px;
  padding: 20px 0;
  border-top: 1px solid var(--border);
}

.edit-btn {
  padding: 10px 28px;
  border-radius: 10px;
  font-weight: 600;
  font-size: 0.95rem;
  cursor: pointer;
  transition: all 0.3s ease;
  border: 1px solid transparent;
}

.edit-btn--save {
  background: var(--primary-600);
  color: #fff;
  border-color: var(--primary-600);
}

.edit-btn--save:hover:not(:disabled) {
  background: var(--primary-700);
  transform: translateY(-2px);
  box-shadow: 0 4px 14px rgba(16, 185, 129, 0.3);
}

.edit-btn--save:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.edit-btn--cancel {
  background: rgba(255, 255, 255, 0.8);
  color: var(--text-secondary);
  border-color: var(--border);
}

.edit-btn--cancel:hover {
  background: var(--bg-muted);
  color: var(--text-strong);
}



/* ========== Responsive ========== */
@media (max-width: 1024px) {
  .about-layout { grid-template-columns: 1fr; }
  .about-sidebar { position: static; order: -1; }
}

@media (max-width: 768px) {
  .hero-banner { padding: 32px 20px; }
  .hero-name { font-size: 2rem; }
  .hero-social { flex-direction: column; align-items: center; }
  .hero-social-btn { width: 100%; justify-content: center; }
  
  .about-section { padding: 20px 16px; }
  .stats-grid { grid-template-columns: repeat(2, 1fr); }
  .contact-cards { grid-template-columns: 1fr; }
}
</style>