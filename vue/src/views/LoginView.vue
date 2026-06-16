<template>
  <AuthShell
    panel-title="登录你的账号"
  >
    <div class="auth-form">
      <div class="tab-switch">
        <button
          type="button"
          class="tab-option"
          :class="{ 'is-active': loginType === 'account' }"
          @click="loginType = 'account'"
        >
          <span>&#128100;</span>
          <span>账号登录</span>
        </button>
        <button
          type="button"
          class="tab-option"
          :class="{ 'is-active': loginType === 'email' }"
          @click="loginType = 'email'"
        >
          <span>&#9993;</span>
          <span>邮箱登录</span>
        </button>
      </div>

      <form v-if="loginType === 'account'" class="auth-form" @submit.prevent="handleAccountLogin">
        <div class="form-stack">
          <div class="field-header">
            <label class="field-label" for="account">账号</label>
          </div>
          <div class="input-shell" :class="{ 'input-shell--error': !!errors.account }">
            <span class="input-shell__icon">&#128100;</span>
            <input
              id="account"
              v-model="accountForm.account"
              class="input-shell__control"
              type="text"
              placeholder="请输入账号"
              @input="clearError('account')"
            />
          </div>
          <p v-if="errors.account" class="helper-text helper-text--error">{{ errors.account }}</p>
        </div>

        <div class="form-stack">
          <div class="field-header">
            <label class="field-label" for="password">密码</label>
          </div>
          <div class="input-shell" :class="{ 'input-shell--error': !!errors.password }">
            <span class="input-shell__icon">&#128274;</span>
            <input
              id="password"
              v-model="accountForm.password"
              class="input-shell__control"
              type="password"
              placeholder="请输入密码"
              @input="clearError('password')"
            />
          </div>
          <p v-if="errors.password" class="helper-text helper-text--error">{{ errors.password }}</p>
        </div>

        <div class="form-stack">
          <div class="field-header">
            <label class="field-label" for="captcha-code">图形验证码</label>
          </div>
          <div class="input-shell" :class="{ 'input-shell--error': !!errors.captchaCode }">
            <span class="input-shell__icon">&#128394;</span>
            <input
              id="captcha-code"
              v-model="accountForm.captcha_code"
              class="input-shell__control"
              type="text"
              placeholder="请输入 6 位图形验证码"
              maxlength="6"
              @input="clearError('captchaCode')"
            />
            <div class="input-shell__addon">
              <button
                type="button"
                class="captcha-preview"
                title="点击刷新验证码"
                @click="refreshCaptcha"
              >
                <img
                  v-if="captchaImage"
                  :src="captchaImage"
                  alt="验证码"
                  class="captcha-preview__image"
                  @error="captchaImage = ''"
                />
                <span v-else class="captcha-preview__placeholder">点击获取</span>
              </button>
            </div>
          </div>
          <input v-model="accountForm.captcha_key" type="hidden" />
          <p v-if="errors.captchaCode" class="helper-text helper-text--error">{{ errors.captchaCode }}</p>
        </div>

        <button type="submit" class="submit-primary" :disabled="loading">
          <span v-if="loading" class="spinner"></span>
          <span>{{ loading ? '登录中...' : '立即登录' }}</span>
        </button>
      </form>

      <form v-else class="auth-form" @submit.prevent="handleEmailLogin">
        <div class="form-stack">
          <div class="field-header">
            <label class="field-label" for="email">邮箱</label>
          </div>
          <div class="input-shell" :class="{ 'input-shell--error': !!errors.email }">
            <span class="input-shell__icon">&#9993;</span>
            <input
              id="email"
              v-model="emailForm.email"
              class="input-shell__control"
              type="email"
              placeholder="请输入邮箱地址"
              @input="clearError('email')"
            />
          </div>
          <p v-if="errors.email" class="helper-text helper-text--error">{{ errors.email }}</p>
        </div>

        <div class="form-stack">
          <div class="field-header">
            <label class="field-label" for="email-captcha">邮箱验证码</label>
          </div>
          <div class="input-shell" :class="{ 'input-shell--error': !!errors.emailCaptcha }">
            <span class="input-shell__icon">&#128276;</span>
            <input
              id="email-captcha"
              v-model="emailForm.captcha"
              class="input-shell__control"
              type="text"
              placeholder="请输入 6 位邮箱验证码"
              maxlength="6"
              @input="clearError('emailCaptcha')"
            />
            <div class="input-shell__addon">
              <button
                type="button"
                class="inline-action"
                :disabled="isSending || countdown > 0"
                @click="sendEmailCaptcha"
              >
                {{ countdown > 0 ? `${countdown}s` : (isSending ? '发送中' : '获取验证码') }}
              </button>
            </div>
          </div>
          <p v-if="errors.emailCaptcha" class="helper-text helper-text--error">{{ errors.emailCaptcha }}</p>
        </div>

        <button type="submit" class="submit-primary" :disabled="loading">
          <span v-if="loading" class="spinner"></span>
          <span>{{ loading ? '登录中...' : '邮箱快捷登录' }}</span>
        </button>
      </form>
    </div>

    <template #footer>
      <div class="form-links">
        <a href="#" class="text-link" @click.prevent="goRegister">还没有账号？立即注册</a>
        <a href="#" class="text-link" @click.prevent="goForgotPassword">忘记密码？</a>
      </div>
    </template>
  </AuthShell>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import AuthShell from '../components/AuthShell.vue'

interface AccountForm {
  account: string
  password: string
  captcha_key: string
  captcha_code: string
}

interface EmailForm {
  email: string
  captcha: string
}

interface Errors {
  account?: string
  password?: string
  captchaCode?: string
  email?: string
  emailCaptcha?: string
}

const router = useRouter()

const loginType = ref<'account' | 'email'>('account')

const accountForm = ref<AccountForm>({
  account: '',
  password: '',
  captcha_key: '',
  captcha_code: ''
})

const emailForm = ref<EmailForm>({
  email: '',
  captcha: ''
})

const captchaImage = ref('')
const loading = ref(false)
const isSending = ref(false)
const countdown = ref(0)
const errors = reactive<Errors>({})

// 获取图形验证码
const fetchCaptcha = async () => {
  try {
    const response = await axios.get('/api/v1/auth/image_captcha')
    const { captcha_id, base_64 } = response.data.data
    accountForm.value.captcha_key = captcha_id
    captchaImage.value = base_64
  } catch (error) {
    console.error('获取验证码失败:', error)
  }
}

const refreshCaptcha = () => {
  fetchCaptcha()
}

const clearError = (field: keyof Errors) => {
  errors[field] = undefined
}

// 验证账号密码表单
const validateAccountForm = (): boolean => {
  let valid = true
  const form = accountForm.value

  if (!form.account.trim()) {
    errors.account = '账号不能为空'
    valid = false
  }

  if (!form.password) {
    errors.password = '密码不能为空'
    valid = false
  } else if (form.password.length < 6) {
    errors.password = '密码长度至少6位'
    valid = false
  }

  if (!form.captcha_code) {
    errors.captchaCode = '验证码不能为空'
    valid = false
  } else if (form.captcha_code.length !== 6) {
    errors.captchaCode = '验证码必须为6位'
    valid = false
  }

  return valid
}

// 验证邮箱登录表单
const validateEmailForm = (): boolean => {
  let valid = true
  const form = emailForm.value

  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  if (!form.email) {
    errors.email = '邮箱不能为空'
    valid = false
  } else if (!emailRegex.test(form.email)) {
    errors.email = '邮箱格式不正确'
    valid = false
  }

  if (!form.captcha) {
    errors.emailCaptcha = '验证码不能为空'
    valid = false
  } else if (form.captcha.length !== 6) {
    errors.emailCaptcha = '验证码必须为6位'
    valid = false
  }

  return valid
}

// 发送邮箱验证码
const sendEmailCaptcha = async () => {
  const email = emailForm.value.email
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  if (!email) {
    errors.email = '请输入邮箱'
    return
  }
  if (!emailRegex.test(email)) {
    errors.email = '邮箱格式不正确'
    return
  }

  isSending.value = true
  try {
    await axios.post('/api/v1/auth/captcha', {
      email: email,
      purpose: 'login'
    })

    countdown.value = 60
    const timer = setInterval(() => {
      countdown.value--
      if (countdown.value <= 0) {
        clearInterval(timer)
      }
    }, 1000)

    alert('验证码已发送到您的邮箱，请注意查收')
  } catch (error: any) {
    console.error('发送验证码失败:', error)
    if (error.response?.data?.message) {
      alert(error.response.data.message)
    } else {
      alert('发送验证码失败，请稍后重试')
    }
  } finally {
    isSending.value = false
  }
}

// 处理账号密码登录
const handleAccountLogin = async () => {
  if (!validateAccountForm()) return

  loading.value = true
  try {
    const response = await axios.post('/api/v1/auth/login', accountForm.value)
    const { token } = response.data.data
    localStorage.setItem('token', token)
    // 获取完整用户信息并存储
    try {
      const userRes = await axios.get('/api/v1/user/info', {
        headers: { Authorization: `Bearer ${token}` }
      })
      if (userRes.data.data) {
        localStorage.setItem('user', JSON.stringify(userRes.data.data))
      }
    } catch { /* 忽略，NavBar 会自行获取 */ }
    router.push('/')
  } catch (error: any) {
    console.error('登录失败:', error)
    if (error.response?.data?.message) {
      alert(error.response.data.message)
    } else {
      alert('登录失败，请检查账号密码和验证码')
    }
    refreshCaptcha()
  } finally {
    loading.value = false
  }
}

// 处理邮箱登录
const handleEmailLogin = async () => {
  if (!validateEmailForm()) return

  loading.value = true
  try {
    const response = await axios.post('/api/v1/auth/email_login', {
      email: emailForm.value.email,
      captcha: emailForm.value.captcha,
      purpose: 'login'
    })
    const { token } = response.data.data
    localStorage.setItem('token', token)
    // 获取完整用户信息并存储
    try {
      const userRes = await axios.get('/api/v1/user/info', {
        headers: { Authorization: `Bearer ${token}` }
      })
      if (userRes.data.data) {
        localStorage.setItem('user', JSON.stringify(userRes.data.data))
      }
    } catch { /* 忽略，NavBar 会自行获取 */ }
    router.push('/')
  } catch (error: any) {
    console.error('邮箱登录失败:', error)
    if (error.response?.data?.message) {
      alert(error.response.data.message)
    } else {
      alert('登录失败，请检查邮箱和验证码')
    }
  } finally {
    loading.value = false
  }
}

const goRegister = () => {
  router.push('/register')
}

const goForgotPassword = () => {
  router.push('/forgot-password')
}

onMounted(() => {
  fetchCaptcha()
})
</script>
