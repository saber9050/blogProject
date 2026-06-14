<template>
  <div class="auth-page">
    <div class="auth-page__glow auth-page__glow--left"></div>
    <div class="auth-page__glow auth-page__glow--right"></div>

    <header class="auth-topbar">
      <div class="auth-topbar__inner">
        <button type="button" class="brand-mark" @click="goHome">
          <span class="brand-mark__icon">&#9998;</span>
          <span class="brand-mark__text">
            <strong>{{ brandName }}</strong>
            <small>{{ brandTagline }}</small>
          </span>
        </button>

        <nav class="auth-topnav" aria-label="认证页面导航">
          <RouterLink
            v-for="item in navItems"
            :key="item.to"
            :to="item.to"
            class="auth-topnav__link"
            :class="{ 'is-active': route.path === item.to }"
          >
            {{ item.label }}
          </RouterLink>
        </nav>
      </div>
    </header>

    <main class="auth-stage">
      <section class="auth-shell">
        <aside class="auth-hero">
          <span class="auth-hero__eyebrow">{{ heroEyebrow }}</span>
          <h1 class="auth-hero__title">{{ heroTitle }}</h1>
          <p class="auth-hero__description">{{ heroDescription }}</p>

          <div class="auth-hero__list">
            <article
              v-for="item in heroHighlights"
              :key="`${item.title}-${item.description}`"
              class="auth-hero__item"
            >
              <span class="auth-hero__item-icon">{{ item.icon }}</span>
              <div>
                <h3>{{ item.title }}</h3>
                <p>{{ item.description }}</p>
              </div>
            </article>
          </div>
        </aside>

        <section class="auth-panel">
          <span class="auth-panel__eyebrow">{{ panelEyebrow }}</span>
          <h2 class="auth-panel__title">{{ panelTitle }}</h2>
          <p class="auth-panel__description">{{ panelDescription }}</p>

          <slot />

          <div class="auth-panel__footer">
            <slot name="footer" />
          </div>
        </section>
      </section>
    </main>

    <footer class="auth-footer">
      <p>{{ footerText }}</p>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'

interface HighlightItem {
  icon: string
  title: string
  description: string
}

interface Props {
  heroEyebrow: string
  heroTitle: string
  heroDescription: string
  heroHighlights: HighlightItem[]
  panelEyebrow: string
  panelTitle: string
  panelDescription: string
  footerText?: string
  brandName?: string
  brandTagline?: string
}

withDefaults(defineProps<Props>(), {
  footerText: `© ${new Date().getFullYear()} 我的博客. 保留所有权利.`,
  brandName: '我的博客',
  brandTagline: '现代创作空间'
})

const route = useRoute()
const router = useRouter()

const navItems = computed(() => [
  { to: '/login', label: '登录' },
  { to: '/register', label: '注册' },
  { to: '/forgot-password', label: '找回密码' }
])

const goHome = () => {
  router.push('/login')
}
</script>

<style scoped>
.auth-page {
  position: relative;
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.auth-page__glow {
  position: absolute;
  inset: auto;
  border-radius: 999px;
  filter: blur(40px);
  opacity: 0.7;
  pointer-events: none;
}

.auth-page__glow--left {
  top: 80px;
  left: -120px;
  width: 320px;
  height: 320px;
  background: rgba(16, 185, 129, 0.22);
}

.auth-page__glow--right {
  right: -80px;
  bottom: 100px;
  width: 300px;
  height: 300px;
  background: rgba(59, 130, 246, 0.18);
}

.auth-topbar {
  position: relative;
  z-index: 1;
  padding: 24px 24px 0;
}

.auth-topbar__inner {
  max-width: 1180px;
  margin: 0 auto;
  padding: 14px 18px;
  border: 1px solid rgba(255, 255, 255, 0.55);
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.66);
  backdrop-filter: blur(18px);
  box-shadow: 0 24px 48px rgba(15, 23, 42, 0.08);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.brand-mark {
  display: inline-flex;
  align-items: center;
  gap: 14px;
  border: none;
  background: transparent;
  cursor: pointer;
  color: var(--text-strong);
}

.brand-mark__icon {
  width: 44px;
  height: 44px;
  border-radius: 16px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--primary-600), var(--accent-500));
  color: #fff;
  box-shadow: 0 12px 24px rgba(16, 185, 129, 0.24);
  font-size: 1.1rem;
}

.brand-mark__text {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  line-height: 1.15;
}

.brand-mark__text strong {
  font-size: 1rem;
  font-weight: 700;
}

.brand-mark__text small {
  font-size: 0.76rem;
  color: var(--text-muted);
}

.auth-topnav {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.auth-topnav__link {
  min-height: 40px;
  padding: 0 16px;
  border-radius: 999px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
  font-size: 0.92rem;
  font-weight: 600;
  transition:
    background-color var(--transition-base),
    color var(--transition-base),
    transform var(--transition-base),
    box-shadow var(--transition-base);
}

.auth-topnav__link:hover {
  color: var(--primary-700);
  background: rgba(255, 255, 255, 0.8);
  transform: translateY(-1px);
}

.auth-topnav__link.is-active {
  color: var(--primary-700);
  background: rgba(255, 255, 255, 0.92);
  box-shadow: 0 12px 24px rgba(15, 23, 42, 0.08);
}

.auth-stage {
  position: relative;
  z-index: 1;
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 28px 24px 40px;
}

.auth-shell {
  width: min(1180px, 100%);
  display: grid;
  grid-template-columns: 1.05fr 0.95fr;
  gap: 24px;
  align-items: stretch;
}

.auth-hero,
.auth-panel {
  border: 1px solid rgba(255, 255, 255, 0.55);
  border-radius: 32px;
  box-shadow: 0 28px 70px rgba(15, 23, 42, 0.1);
  backdrop-filter: blur(18px);
}

.auth-hero {
  position: relative;
  overflow: hidden;
  padding: 40px;
  background:
    radial-gradient(circle at top left, rgba(255, 255, 255, 0.26), transparent 42%),
    linear-gradient(160deg, #0f766e 0%, #059669 48%, #34d399 100%);
  color: #fff;
  display: flex;
  flex-direction: column;
  justify-content: center;
  min-height: 620px;
}

.auth-hero::after {
  content: '';
  position: absolute;
  inset: auto -90px -100px auto;
  width: 260px;
  height: 260px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.12);
}

.auth-hero__eyebrow,
.auth-panel__eyebrow {
  display: inline-flex;
  align-items: center;
  width: fit-content;
  min-height: 34px;
  padding: 0 14px;
  border-radius: 999px;
  font-size: 0.78rem;
  font-weight: 700;
  letter-spacing: 0.08em;
}

.auth-hero__eyebrow {
  color: rgba(255, 255, 255, 0.96);
  background: rgba(255, 255, 255, 0.14);
  margin-bottom: 18px;
}

.auth-hero__title {
  max-width: 12ch;
  font-size: clamp(2.2rem, 5vw, 3.4rem);
  line-height: 1.08;
  font-weight: 800;
  letter-spacing: -0.03em;
  margin-bottom: 18px;
}

.auth-hero__description {
  max-width: 38rem;
  font-size: 1rem;
  line-height: 1.8;
  color: rgba(255, 255, 255, 0.84);
  margin-bottom: 32px;
}

.auth-hero__list {
  display: grid;
  gap: 14px;
}

.auth-hero__item {
  position: relative;
  z-index: 1;
  display: grid;
  grid-template-columns: 52px 1fr;
  gap: 14px;
  align-items: flex-start;
  padding: 18px;
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.12);
  border: 1px solid rgba(255, 255, 255, 0.18);
}

.auth-hero__item-icon {
  width: 52px;
  height: 52px;
  border-radius: 18px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.16);
  font-size: 1.25rem;
}

.auth-hero__item h3 {
  font-size: 1rem;
  font-weight: 700;
  margin-bottom: 6px;
}

.auth-hero__item p {
  color: rgba(255, 255, 255, 0.76);
  font-size: 0.92rem;
  line-height: 1.6;
}

.auth-panel {
  padding: 34px 32px 28px;
  background: rgba(255, 255, 255, 0.8);
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.auth-panel__eyebrow {
  color: var(--primary-700);
  background: var(--primary-soft);
  margin-bottom: 18px;
}

.auth-panel__title {
  font-size: clamp(1.7rem, 3vw, 2.3rem);
  line-height: 1.2;
  letter-spacing: -0.03em;
  color: var(--text-strong);
  margin-bottom: 12px;
}

.auth-panel__description {
  color: var(--text-secondary);
  line-height: 1.75;
  margin-bottom: 26px;
}

.auth-panel__footer {
  margin-top: 22px;
}

.auth-footer {
  position: relative;
  z-index: 1;
  padding: 0 24px 24px;
  text-align: center;
}

.auth-footer p {
  color: var(--text-muted);
  font-size: 0.88rem;
}

@media (max-width: 1024px) {
  .auth-shell {
    grid-template-columns: 1fr;
  }

  .auth-hero {
    min-height: auto;
    padding: 32px;
  }

  .auth-hero__title {
    max-width: none;
  }
}

@media (max-width: 640px) {
  .auth-topbar {
    padding: 18px 16px 0;
  }

  .auth-topbar__inner {
    padding: 14px;
    border-radius: 22px;
    flex-direction: column;
    align-items: flex-start;
  }

  .auth-topnav {
    width: 100%;
  }

  .auth-topnav__link {
    flex: 1;
  }

  .auth-stage {
    padding: 20px 16px 28px;
  }

  .auth-hero,
  .auth-panel {
    border-radius: 26px;
  }

  .auth-hero,
  .auth-panel {
    padding-left: 20px;
    padding-right: 20px;
  }

  .auth-hero {
    padding-top: 28px;
    padding-bottom: 28px;
  }

  .auth-panel {
    padding-top: 26px;
    padding-bottom: 22px;
  }

  .auth-hero__item {
    grid-template-columns: 44px 1fr;
    padding: 16px;
  }

  .auth-hero__item-icon {
    width: 44px;
    height: 44px;
    border-radius: 14px;
  }
}
</style>
