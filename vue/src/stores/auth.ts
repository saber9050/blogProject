import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import axios from 'axios'

interface User {
  user_id: number
  user_name: string
  user_role_id: number
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const user = ref<User | null>(null)

  const isAuthenticated = computed(() => !!token.value)

  const setToken = (newToken: string) => {
    token.value = newToken
    localStorage.setItem('token', newToken)
  }

  const setUser = (userData: User) => {
    user.value = userData
    localStorage.setItem('user_name', userData.user_name)
  }

  const login = async (account: string, password: string, captchaKey: string, captchaCode: string) => {
    const response = await axios.post('/api/auth/login', {
      account,
      password,
      captcha_key: captchaKey,
      captcha_code: captchaCode
    })
    const { token: newToken, user_name, user_id, user_role_id } = response.data.data
    setToken(newToken)
    setUser({ user_id, user_name, user_role_id })
    return response.data
  }

  const logout = async () => {
    try {
      await axios.post('/api/auth/logout', {}, {
        headers: { Authorization: token.value }
      })
    } catch (error) {
      console.error('登出失败:', error)
    } finally {
      token.value = ''
      user.value = null
      localStorage.removeItem('token')
      localStorage.removeItem('user_name')
    }
  }

  const fetchCaptcha = async () => {
    const response = await axios.get('/api/auth/image_captcha')
    return response.data.data
  }

  return {
    token,
    user,
    isAuthenticated,
    login,
    logout,
    fetchCaptcha
  }
})