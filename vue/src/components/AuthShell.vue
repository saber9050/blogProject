<template>
  <div class="auth-page">
    <div class="auth-page__glow auth-page__glow--left"></div>
    <div class="auth-page__glow auth-page__glow--right"></div>

    <main class="auth-stage">
      <section class="auth-shell auth-shell--single">
        <section class="auth-panel">
          <!-- 品牌标识区域：已移除右侧导航，整体居中 -->
          <div class="auth-inlinebar">
            <button type="button" class="brand-mark" @click="goHome">
              <span class="brand-mark__icon">&#9998;</span>
              <span class="brand-mark__text">
                <strong>{{ brandName }}</strong>
                <small>{{ brandTagline }}</small>
              </span>
            </button>
          </div>

          <!-- 标题保持左对齐（不移动） -->
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
import { useRouter } from 'vue-router'

interface Props {
  panelTitle: string
  brandName?: string
  brandTagline?: string
}

withDefaults(defineProps<Props>(), {
  brandName: '我的博客',
  brandTagline: '现代创作空间'
})

const router = useRouter()
const goHome = () => {
  router.push('/login')
}
</script>

<style scoped>
/* ========== 光晕效果 ========== */
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

/* ========== 品牌标识 - 居中 ========== */
.auth-inlinebar {
  display: flex;
  justify-content: center;
  align-items: center;
  width: 100%;
  margin-bottom: 8px;
}

.brand-mark {
  display: inline-flex;
  align-items: center;
  gap: 14px;
  border: none;
  background: transparent;
  cursor: pointer;
  color: var(--text-strong, #1e293b);
  transition: transform 0.2s ease;
}

.brand-mark:hover {
  transform: translateY(-1px);
}

.brand-mark__icon {
  width: 44px;
  height: 44px;
  border-radius: 16px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #059669, #3b82f6);
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
  color: var(--text-muted, #64748b);
}

/* ========== 认证卡片主体 ========== */
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

.auth-panel {
  border: 1px solid rgba(255, 255, 255, 0.55);
  border-radius: 28px;
  box-shadow: 0 28px 70px rgba(15, 23, 42, 0.1);
  backdrop-filter: blur(18px);
  padding: 24px 24px 18px;
  background: rgba(255, 255, 255, 0.8);
  display: flex;
  flex-direction: column;
  justify-content: center;
}

/* 标题保持左对齐（不移动） */
.auth-panel__title {
  font-size: clamp(1.5rem, 2.8vw, 2rem);
  line-height: 1.2;
  letter-spacing: -0.03em;
  color: var(--text-strong, #0f172a);
  margin-bottom: 8px;
  text-align: left;   /* 关键：左对齐，不居中 */
}

.auth-panel__footer {
  margin-top: 14px;
}

/* ========== 响应式适配 ========== */
@media (max-width: 640px) {
  .auth-stage {
    padding: 20px 16px 28px;
  }

  .auth-panel {
    border-radius: 26px;
    padding: 26px 20px 22px;
  }

  .auth-inlinebar {
    margin-bottom: 12px;
  }

  .brand-mark__icon {
    width: 40px;
    height: 40px;
    border-radius: 14px;
  }

  .brand-mark__text strong {
    font-size: 0.95rem;
  }

  .brand-mark__text small {
    font-size: 0.7rem;
  }
}

@media (min-width: 1025px) and (max-height: 900px) {
  .auth-stage {
    padding-top: 12px;
    padding-bottom: 12px;
  }

  .auth-panel {
    padding: 22px 22px 16px;
  }

  .auth-inlinebar {
    margin-bottom: 4px;
  }
}
</style>