<template>
  <div class="auth-page">
    <div class="auth-page__glow auth-page__glow--left"></div>
    <div class="auth-page__glow auth-page__glow--right"></div>

    <main class="auth-stage">
      <section class="auth-shell auth-shell--single">
        <section class="auth-panel">
          <div class="auth-inlinebar">
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
          <h2 class="auth-panel__title">{{ panelTitle }}</h2>

          <slot />

          <div class="auth-panel__footer">
            <slot name="footer" />
          </div>
        </section>
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'

interface Props {
  panelTitle: string
  brandName?: string
  brandTagline?: string
}

withDefaults(defineProps<Props>(), {
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
  padding: 16px 16px 0;
}

.auth-topbar__inner {
  max-width: 1180px;
  margin: 0 auto;
  padding: 10px 14px;
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
  padding: 16px 16px 18px;
}

.auth-shell {
  width: min(1180px, 100%);
  display: grid;
  grid-template-columns: 1fr;
  gap: 18px;
  align-items: stretch;
}

.auth-shell--single {
  width: min(620px, 100%);
  margin: 0 auto;
}

.auth-hero,
.auth-panel {
  border: 1px solid rgba(255, 255, 255, 0.55);
  border-radius: 28px;
  box-shadow: 0 28px 70px rgba(15, 23, 42, 0.1);
  backdrop-filter: blur(18px);
}

.auth-hero {
  position: relative;
  overflow: hidden;
  padding: 30px;
  background:
    radial-gradient(circle at top left, rgba(255, 255, 255, 0.26), transparent 42%),
    linear-gradient(160deg, #0f766e 0%, #059669 48%, #34d399 100%);
  color: #fff;
  display: flex;
  flex-direction: column;
  justify-content: center;
  min-height: 520px;
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
  margin-bottom: 12px;
}

.auth-hero__title {
  max-width: 12ch;
  font-size: clamp(1.9rem, 4vw, 2.9rem);
  line-height: 1.08;
  font-weight: 800;
  letter-spacing: -0.03em;
  margin-bottom: 12px;
}

.auth-hero__description {
  max-width: 38rem;
  font-size: 0.94rem;
  line-height: 1.65;
  color: rgba(255, 255, 255, 0.84);
  margin-bottom: 20px;
}

.auth-hero__list {
  display: grid;
  gap: 10px;
}

.auth-hero__item {
  position: relative;
  z-index: 1;
  display: grid;
  grid-template-columns: 44px 1fr;
  gap: 12px;
  align-items: flex-start;
  padding: 14px;
  border-radius: 20px;
  background: rgba(255, 255, 255, 0.12);
  border: 1px solid rgba(255, 255, 255, 0.18);
}

.auth-hero__item-icon {
  width: 44px;
  height: 44px;
  border-radius: 14px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.16);
  font-size: 1rem;
}

.auth-hero__item h3 {
  font-size: 0.95rem;
  font-weight: 700;
  margin-bottom: 4px;
}

.auth-hero__item p {
  color: rgba(255, 255, 255, 0.76);
  font-size: 0.84rem;
  line-height: 1.45;
}

.auth-panel {
  padding: 24px 24px 18px;
  background: rgba(255, 255, 255, 0.8);
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.auth-panel__eyebrow {
  color: var(--primary-700);
  background: var(--primary-soft);
  margin-bottom: 12px;
}

.auth-panel__title {
  font-size: clamp(1.5rem, 2.8vw, 2rem);
  line-height: 1.2;
  letter-spacing: -0.03em;
  color: var(--text-strong);
  margin-bottom: 8px;
}

.auth-panel__description {
  color: var(--text-secondary);
  line-height: 1.55;
  margin-bottom: 16px;
}

.auth-panel__footer {
  margin-top: 14px;
}

.auth-footer {
  position: relative;
  z-index: 1;
  padding: 0 16px 12px;
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
    padding: 24px;
  }

  .auth-hero__title {
    max-width: none;
  }
}

@media (min-width: 1025px) and (max-height: 900px) {
  .auth-page {
    overflow: hidden;
  }

  .auth-topbar {
    padding-top: 12px;
  }

  .auth-stage {
    padding-top: 12px;
    padding-bottom: 12px;
  }

  .auth-hero {
    min-height: 0;
    padding: 24px;
  }

  .auth-hero__description {
    margin-bottom: 14px;
  }

  .auth-hero__item p {
    display: none;
  }

  .auth-panel {
    padding: 22px 22px 16px;
  }

  .auth-footer {
    display: none;
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
