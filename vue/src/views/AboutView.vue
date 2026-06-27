<template>
  <div class="about-page">
    <NavBar />
    <main class="page-shell">
      <!-- Hero Banner -->
      <div class="hero-banner">
        <div class="hero-content">
          <div class="hero-avatar">
            <img
              v-if="author.avatar_url"
              :src="author.avatar_url"
              class="hero-avatar__img"
            />
            <span v-else class="hero-avatar__placeholder">&#128100;</span>
          </div>
          <h1 class="hero-name">{{ author.user_name}}</h1>
          <p class="hero-quote">"热爱构建高性能、高可维护性的系统"</p>
          <div class="hero-tags">
            <span class="hero-tag">Go</span>
            <span class="hero-tag">Vue</span>
            <span class="hero-tag">AI</span>
            <span class="hero-tag">Docker</span>
            <span class="hero-tag">Redis</span>
            <span class="hero-tag">MySQL</span>
          </div>
          <div class="hero-social">
            <a href="https://github.com/yourusername" class="hero-social-btn" target="_blank">
              <span>&#128241;</span> GitHub
            </a>
            <a href="mailto:contact@example.com" class="hero-social-btn">
              <span>&#9993;</span> Email
            </a>
          </div>
        </div>
      </div>

      <div class="about-layout">
        <div class="about-main">
          <!-- 我的故事 -->
          <div class="about-section">
            <h2 class="about-section__title"><span class="section-icon">&#128214;</span> 我的故事</h2>
            <div class="about-section__content">
              <p class="about-text">
                大学开始接触编程，最初学习 Java。后来接触 Go，逐渐喜欢上它的简洁、高效以及优秀的并发模型。
              </p>
              <p class="about-text">
                目前主要从事：Go 后端、微服务、Redis、Docker、AI Agent 开发。
              </p>
              <p class="about-text">
                我喜欢把复杂问题拆解成简单模块，也喜欢研究系统架构背后的设计思想。
              </p>
              <p class="about-text">
                这个博客记录我的成长，希望未来回头看时，仍然能够感谢现在坚持学习的自己。
              </p>
            </div>
          </div>

          <!-- 当前研究方向 -->
          <div class="about-section">
            <h2 class="about-section__title"><span class="section-icon">&#128640;</span> 我目前正在研究</h2>
            <div class="about-section__content">
              <div class="focus-cards">
                <div class="focus-card">
                  <div class="focus-card__icon">&#129302;</div>
                  <h3 class="focus-card__title">AI Agent</h3>
                  <p class="focus-card__desc">智能代理开发与应用</p>
                </div>
                <div class="focus-card">
                  <div class="focus-card__icon">&#128187;</div>
                  <h3 class="focus-card__title">MCP</h3>
                  <p class="focus-card__desc">模型上下文协议</p>
                </div>
                <div class="focus-card">
                  <div class="focus-card__icon">&#9881;</div>
                  <h3 class="focus-card__title">Vue3</h3>
                  <p class="focus-card__desc">现代前端框架</p>
                </div>
              </div>
            </div>
          </div>

          <!-- 博客理念 -->
          <div class="about-section">
            <h2 class="about-section__title"><span class="section-icon">&#128196;</span> 为什么建立博客？</h2>
            <div class="about-section__content">
              <div class="philosophy-content">
                <p class="philosophy-text">
                  学习最大的敌人不是不会，<br>
                  而是遗忘。
                </p>
                <p class="philosophy-text">
                  因此，<br>
                  我希望把每一次踩坑、<br>
                  每一次成长、<br>
                  每一次思考，<br>
                  都记录下来。
                </p>
                <p class="philosophy-text">
                  既帮助别人，<br>
                  也帮助未来的自己。
                </p>
              </div>
            </div>
          </div>

          <!-- 兴趣领域 -->
          <div class="about-section">
            <h2 class="about-section__title"><span class="section-icon">&#10084;</span> 我感兴趣</h2>
            <div class="about-section__content">
              <div class="interest-tags">
                <span class="interest-tag">Go</span>
                <span class="interest-tag">Redis</span>
                <span class="interest-tag">消息队列</span>
                <span class="interest-tag">系统设计</span>
                <span class="interest-tag">AI Agent</span>
                <span class="interest-tag">Docker</span>
                <span class="interest-tag">Linux</span>
                <span class="interest-tag">MySQL</span>
                <span class="interest-tag">Vue3</span>
                <span class="interest-tag">TypeScript</span>
                <span class="interest-tag">数据库优化</span>
                <span class="interest-tag">高并发</span>
                <span class="interest-tag">性能优化</span>
              </div>
            </div>
          </div>

          <!-- 博客统计 -->
          <div class="about-section">
            <h2 class="about-section__title"><span class="section-icon">&#128202;</span> 博客数据</h2>
            <div class="about-section__content">
              <div class="stats-grid">
                <div class="stat-card">
                  <div class="stat-card__icon">&#128196;</div>
                  <div class="stat-card__value">{{ author.article_count || 32 }}</div>
                  <div class="stat-card__label">文章</div>
                </div>
                <div class="stat-card">
                  <div class="stat-card__icon">&#128065;</div>
                  <div class="stat-card__value">{{ formatNumber(author.total_views || 102000) }}</div>
                  <div class="stat-card__label">阅读</div>
                </div>
                <div class="stat-card">
                  <div class="stat-card__icon">&#10084;</div>
                  <div class="stat-card__value">{{ author.total_likes || 580 }}</div>
                  <div class="stat-card__label">点赞</div>
                </div>
                <div class="stat-card">
                  <div class="stat-card__icon">&#128483;</div>
                  <div class="stat-card__value">25</div>
                  <div class="stat-card__label">标签</div>
                </div>
                <div class="stat-card">
                  <div class="stat-card__icon">&#128193;</div>
                  <div class="stat-card__value">8</div>
                  <div class="stat-card__label">分类</div>
                </div>
              </div>
            </div>
          </div>

          <!-- 联系我 -->
          <div class="about-section">
            <h2 class="about-section__title"><span class="section-icon">&#128236;</span> 联系我</h2>
            <div class="about-section__content">
              <div class="contact-cards">
                <div class="contact-card">
                  <div class="contact-card__icon">&#128241;</div>
                  <h3 class="contact-card__title">GitHub</h3>
                  <p class="contact-card__desc">查看我的开源项目</p>
                  <a href="https://github.com/yourusername" class="contact-card__link" target="_blank">
                    访问主页 &rarr;
                  </a>
                </div>
                <div class="contact-card">
                  <div class="contact-card__icon">&#9993;</div>
                  <h3 class="contact-card__title">Email</h3>
                  <div class="contact-card__desc">欢迎交流技术</div>
                  <a href="mailto:contact@example.com" class="contact-card__link">
                    发送邮件 &rarr;
                  </a>
                </div>
                <div class="contact-card">
                  <div class="contact-card__icon">&#128187;</div>
                  <h3 class="contact-card__title">CSDN</h3>
                  <p class="contact-card__desc">查看更多文章</p>
                  <a href="#" class="contact-card__link" target="_blank">
                    访问主页 &rarr;
                  </a>
                </div>
              </div>
            </div>
          </div>

          <!-- Footer -->
          <div class="about-footer">
            <p class="footer-text">
              感谢你的阅读。<br>
              希望这里的内容，<br>
              能够帮助到正在学习路上的你。
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
import { ref, onMounted } from 'vue'
import NavBar from '../components/NavBar.vue'
import api from '../api'

interface Author {
  user_id: number
  user_name: string
  account: string
  email: string
  avatar_url: string
  introduction: string
  role_id: number
  status: number
  create_at: string
  article_count?: number
  total_views?: number
  total_likes?: number
}

const author = ref<Author>({
  user_id: 0,
  user_name: '',
  account: '',
  email: '',
  avatar_url: '',
  introduction: '',
  role_id: 0,
  status: 0,
  create_at: '',
  article_count: 0,
  total_views: 0,
  total_likes: 0
})

const formatNumber = (num: number) => {
  if (num >= 1000) {
    return (num / 1000).toFixed(1) + 'K'
  }
  return num.toString()
}

onMounted(async () => {
  try {
    const res = await api.get('/user/info')
    if (res.data.data) {
      author.value = { ...author.value, ...res.data.data }
    }
  } catch (err) {
    console.log('获取博主信息失败，使用默认值')
  }
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

/* ========== Focus Cards ========== */
.focus-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 16px;
}

.focus-card {
  background: rgba(248, 250, 252, 0.8);
  border-radius: 16px;
  padding: 20px 16px;
  text-align: center;
  transition: all 0.3s ease;
  border: 1px solid rgba(255, 255, 255, 0.5);
}

.focus-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 20px rgba(16, 185, 129, 0.15);
  background: rgba(16, 185, 129, 0.05);
}

.focus-card__icon {
  font-size: 2rem;
  margin-bottom: 12px;
}

.focus-card__title {
  font-size: 1rem;
  font-weight: 700;
  color: var(--text-strong);
  margin: 0 0 6px;
}

.focus-card__desc {
  font-size: 0.8rem;
  color: var(--text-muted);
  margin: 0;
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
  .focus-cards { grid-template-columns: repeat(2, 1fr); }
  .stats-grid { grid-template-columns: repeat(2, 1fr); }
  .contact-cards { grid-template-columns: 1fr; }
}
</style>