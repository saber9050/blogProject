<template>
  <header class="navbar">
    <div class="navbar__inner">
      <!-- 品牌（不可点击） -->
      <div class="navbar__brand">
        <span class="navbar__brand-icon">&#9998;</span>
        <strong>{{ user ? user.user_name + '的博客' : '我的博客' }}</strong>
      </div>

      <!-- 导航按钮：首页 / 关于 -->
      <div class="navbar__links">
        <RouterLink to="/" class="navbar__link" active-class="navbar__link--active" exact>首页</RouterLink>
        <button class="navbar__link" @click="$router.push('/about')">关于</button>
      </div>

      <!-- 搜索栏 -->
      <div class="navbar__search-wrap">
        <input
          v-model="searchText"
          class="navbar__search"
          type="text"
          placeholder="搜索文章..."
          @keyup.enter="doSearch"
        />
      </div>

      <!-- 右侧操作区 -->
      <div class="navbar__right">
        <template v-if="user">
          <div class="navbar__bell-wrap">
            <button class="navbar__bell" @click="handleBell">&#128276;</button>
            <span v-if="notificationCount > 0" class="navbar__bell-badge">
              {{ notificationCount > 99 ? '99+' : notificationCount }}
            </span>
          </div>

          <div
            class="navbar__user-area"
            @mouseenter="openDropdown"
            @mouseleave="scheduleClose"
            @click="dropdownOpen = !dropdownOpen"
          >
            <img
              v-if="user.avatar_url"
              :src="user.avatar_url"
              alt="avatar"
              class="navbar__avatar"
            />
            <span v-else class="navbar__avatar navbar__avatar--placeholder">&#128100;</span>
            <span class="navbar__nick">{{ user.user_name }}</span>

            <div v-if="dropdownOpen" class="navbar__dropdown" @mouseenter="keepOpen" @mouseleave="scheduleClose">
              <button class="navbar__dropdown-item" @click.stop="goProfile">
                <span>&#128100;</span> 个人信息
              </button>
              <button
                v-if="user.role_id === 1"
                class="navbar__dropdown-item"
                @click.stop="$router.push('/admin')"
              >
                <span>&#9881;</span> 后台管理
              </button>
              <div class="navbar__dropdown-divider" />
              <button class="navbar__dropdown-item navbar__dropdown-item--danger" @click.stop="handleLogout">
                <span>&#10140;</span> 退出登录
              </button>
            </div>
          </div>
        </template>
        <template v-else>
          <button class="navbar__btn" @click="$router.push('/login')">登录</button>
        </template>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import api from '../api'

interface UserInfo {
  id: number
  user_name: string
  avatar_url: string
  role_id: number
}

const router = useRouter()
const user = ref<UserInfo | null>(null)
const searchText = ref('')
const notificationCount = ref(0)
const dropdownOpen = ref(false)
let closeTimer: ReturnType<typeof setTimeout> | null = null

const openDropdown = () => {
  if (closeTimer) { clearTimeout(closeTimer); closeTimer = null }
  dropdownOpen.value = true
}

const scheduleClose = () => {
  closeTimer = setTimeout(() => {
    dropdownOpen.value = false
  }, 150)
}

const keepOpen = () => {
  if (closeTimer) { clearTimeout(closeTimer); closeTimer = null }
}

const emit = defineEmits<{ search: [q: string] }>()

const doSearch = () => {
  emit('search', searchText.value.trim())
}

watch(searchText, (val) => {
  emit('search', val.trim())
})

const handleBell = () => {
  notificationCount.value = 0
}

const goProfile = () => {
  dropdownOpen.value = false
  router.push('/profile')
}

const handleLogout = async () => {
  dropdownOpen.value = false
  try { await api.post('/auth/logout') } catch { /* ignore */ }
  localStorage.removeItem('token')
  localStorage.removeItem('user')
  user.value = null
  router.push('/login')
}

onMounted(async () => {
  const token = localStorage.getItem('token')
  if (!token) return

  try {
    const res = await api.get('/user/info')
    if (res.data.data) {
      user.value = res.data.data
      localStorage.setItem('user', JSON.stringify(res.data.data))
    }
  } catch {
    // API 失败时尝试从 localStorage 读取
    const stored = localStorage.getItem('user')
    if (stored) {
      try { user.value = JSON.parse(stored) } catch { user.value = null }
    }
  }
  notificationCount.value = 3

  // 监听用户信息更新事件（来自 ProfileView）
  const handleUserInfoUpdated = (event: CustomEvent) => {
    const updatedUser = event.detail
    if (!updatedUser) return
    // 更新 NavBar 中显示的用户名和头像
    if (updatedUser.user_name && user.value) {
      user.value.user_name = updatedUser.user_name
    }
    if (updatedUser.avatar_url && user.value) {
      user.value.avatar_url = updatedUser.avatar_url
    }
    // 同时更新 localStorage 中的用户信息
    const stored = localStorage.getItem('user')
    if (stored) {
      try {
        const u = JSON.parse(stored)
        if (updatedUser.user_name) u.user_name = updatedUser.user_name
        if (updatedUser.avatar_url) u.avatar_url = updatedUser.avatar_url
        localStorage.setItem('user', JSON.stringify(u))
      } catch { /* ignore */ }
    }
  }
  window.addEventListener('user-info-updated', handleUserInfoUpdated as EventListener)
  onUnmounted(() => {
    window.removeEventListener('user-info-updated', handleUserInfoUpdated as EventListener)
  })
})
</script>

<style scoped>
.navbar {
  position: sticky;
  top: 0;
  z-index: 100;
  background: rgba(255, 255, 255, 0.85);
  backdrop-filter: blur(16px);
  border-bottom: 1px solid rgba(148, 163, 184, 0.16);
  box-shadow: 0 2px 16px rgba(15, 23, 42, 0.04);
}

.navbar__inner {
  max-width: 1200px;
  margin: 0 auto;
  height: 60px;
  display: flex;
  align-items: center;
  padding: 0 24px;
  gap: 16px;
}

/* ====== 品牌（不可点击） ====== */
.navbar__brand {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  color: var(--text-strong);
  flex-shrink: 0;
  cursor: default;
  user-select: none;
}

.navbar__brand-icon {
  width: 36px; height: 36px;
  border-radius: 12px;
  display: inline-flex;
  align-items: center; justify-content: center;
  background: linear-gradient(135deg, #059669, #3b82f6);
  color: #fff;
  font-size: 0.95rem;
  box-shadow: 0 8px 16px rgba(16, 185, 129, 0.2);
}

.navbar__brand strong {
  font-size: 1.05rem; font-weight: 700; letter-spacing: -0.01em;
}

/* ====== 导航链接 ====== */
.navbar__links {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.navbar__link {
  padding: 6px 16px;
  border: none;
  border-radius: 10px;
  background: transparent;
  font-size: 0.88rem;
  font-weight: 600;
  color: var(--text-secondary);
  cursor: pointer;
  text-decoration: none;
  transition: background 0.2s ease, color 0.2s ease;
}

.navbar__link:hover {
  background: rgba(16,185,129,0.08);
  color: var(--primary-700);
}

.navbar__link--active {
  background: rgba(16,185,129,0.12);
  color: var(--primary-700);
}

/* ====== 搜索栏 ====== */
.navbar__search-wrap {
  flex: 1;
  display: flex;
  justify-content: center;
  min-width: 0;
}

.navbar__search {
  width: 100%;
  max-width: 400px;
  height: 38px;
  padding: 0 16px 0 38px;
  border: 1px solid var(--border-strong);
  border-radius: 12px;
  background: rgba(255,255,255,0.7) url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='15' height='15' viewBox='0 0 24 24' fill='none' stroke='%2394a3b8' stroke-width='2'%3E%3Ccircle cx='11' cy='11' r='8'/%3E%3Cpath d='m21 21-4.3-4.3'/%3E%3C/svg%3E") 12px center no-repeat;
  font-size: 0.88rem;
  color: var(--text-strong);
  outline: none;
  transition: border-color 0.24s ease, box-shadow 0.24s ease;
}
.navbar__search:focus {
  border-color: var(--primary-500);
  box-shadow: 0 0 0 3px rgba(16,185,129,0.12);
}

/* ====== 右侧 ====== */
.navbar__right {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}

.navbar__bell-wrap { position: relative; }

.navbar__bell {
  width: 36px; height: 36px;
  border: none;
  border-radius: 10px;
  background: transparent;
  font-size: 1.1rem;
  cursor: pointer;
  display: inline-flex;
  align-items: center; justify-content: center;
  transition: background 0.2s ease;
}
.navbar__bell:hover { background: rgba(148,163,184,0.1); }

.navbar__bell-badge {
  position: absolute;
  top: -2px; right: -2px;
  min-width: 18px; height: 18px;
  padding: 0 4px;
  border-radius: 9px;
  background: var(--danger);
  color: #fff;
  font-size: 0.68rem;
  font-weight: 700;
  display: inline-flex;
  align-items: center; justify-content: center;
  line-height: 1;
}

.navbar__user-area {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 10px 4px 4px;
  border-radius: 12px;
  cursor: pointer;
  position: relative;
  transition: background 0.2s ease;
}
.navbar__user-area:hover { background: rgba(148,163,184,0.06); }

.navbar__avatar {
  width: 34px; height: 34px;
  border-radius: 50%;
  object-fit: cover;
  border: 2px solid rgba(16,185,129,0.2);
  flex-shrink: 0;
}
.navbar__avatar--placeholder {
  display: inline-flex;
  align-items: center; justify-content: center;
  background: var(--primary-soft);
  color: var(--primary-700);
  font-size: 0.85rem;
  border: none;
}
.navbar__nick {
  font-size: 0.9rem; font-weight: 600;
  color: var(--text-strong);
  white-space: nowrap;
}

.navbar__dropdown {
  position: absolute;
  top: calc(100% + 6px);
  right: 0;
  min-width: 160px;
  padding: 6px;
  background: rgba(255,255,255,0.96);
  backdrop-filter: blur(16px);
  border: 1px solid var(--border);
  border-radius: 14px;
  box-shadow: 0 12px 32px rgba(15,23,42,0.12);
  z-index: 110;
  display: flex;
  flex-direction: column;
}

.navbar__dropdown-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 9px 14px;
  border: none;
  border-radius: 10px;
  background: transparent;
  font-size: 0.87rem;
  font-weight: 600;
  color: var(--text-secondary);
  cursor: pointer;
  transition: background 0.2s ease;
  text-align: left;
}
.navbar__dropdown-item:hover { background: rgba(16,185,129,0.06); color: var(--primary-700); }
.navbar__dropdown-item--danger:hover { background: var(--danger-soft); color: var(--danger); }

.navbar__dropdown-divider {
  height: 1px;
  margin: 4px 8px;
  background: var(--border);
}

.navbar__btn {
  height: 36px;
  padding: 0 20px;
  border: none;
  border-radius: 12px;
  background: linear-gradient(135deg, var(--primary-600), var(--primary-500));
  color: #fff;
  font-size: 0.85rem; font-weight: 700;
  box-shadow: 0 8px 16px rgba(16,185,129,0.16);
  cursor: pointer;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}
.navbar__btn:hover { transform: translateY(-1px); box-shadow: 0 12px 20px rgba(16,185,129,0.22); }

@media (max-width: 768px) {
  .navbar__inner { padding: 0 14px; gap: 10px; }
  .navbar__search { max-width: 180px; }
  .navbar__nick { display: none; }
}
</style>