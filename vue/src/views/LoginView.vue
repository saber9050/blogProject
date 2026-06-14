<template>
  <AuthShell
    hero-eyebrow="AUTH ACCESS"
    hero-title="欢迎回到你的创作空间"
    hero-description="登录后即可继续写作、管理内容和处理互动消息。界面升级后，账号登录与邮箱验证码登录保持同一体验节奏。"
    :hero-highlights="heroHighlights"
    panel-eyebrow="账号认证"
    panel-title="登录你的账号"
    panel-description="选择最适合当前场景的登录方式，快速进入博客系统。"
    :footer-text="footerText"
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

      <div class="form-note">
        <span class="form-note__icon">i</span>
        <span>{{ loginType === 'account' ? '账号登录需要输入图形验证码。' : '邮箱验证码发送后 60 秒内可再次获取。' }}</span>
      </div>

      <form v-if="loginType === 'account'" class="auth-form" @submit.prevent="handleAccountLogin">
        <div class="form-stack">
          <div class="field-header">
            <label class="field-label" for="account">账号</label>
            <span class="field-tip">输入已注册账号</span>
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
          <p class="helper-text" :class="{ 'helper-text--error': !!errors.account }">
            {{ errors.account ?? '支持账号密码登录。' }}
          </p>
        </div>

        <div class="form-stack">
          <div class="field-header">
            <label class="field-label" for="password">密码</label>
            <span class="field-tip">请输入账号对应密码</span>
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
          <p class="helper-text" :class="{ 'helper-text--error': !!errors.password }">
            {{ errors.password ?? '密码区分大小写。' }}
          </p>
        </div>

        <div class="form-stack">
          <div class="field-header">
            <label class="field-label" for="captcha-code">图形验证码</label>
            <span class="field-tip">点击右侧图片可刷新</span>
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
          <p class="helper-text" :class="{ 'helper-text--error': !!errors.captchaCode }">
            {{ errors.captchaCode ?? '验证码用于提升登录安全性。' }}
          </p>
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
            <span class="field-tip">用于接收登录验证码</span>
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
          <p class="helper-text" :class="{ 'helper-text--error': !!errors.email }">
            {{ errors.email ?? '请输入可正常接收邮件的邮箱。' }}
          </p>
        </div>

        <div class="form-stack">
          <div class="field-header">
            <label class="field-label" for="email-captcha">邮箱验证码</label>
            <span class="field-tip">发送前请确认邮箱无误</span>
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
          <p class="helper-text" :class="{ 'helper-text--error': !!errors.emailCaptcha }">
            {{ errors.emailCaptcha ?? '验证码发送后请尽快完成登录。' }}
          </p>
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

const footerText = `© ${new Date().getFullYear()} 我的博客. 统一视觉升级版认证页`
const heroHighlights = [
  {
    icon: '01',
    title: '双模式登录',
    description: '支持账号密码与邮箱验证码两种登录路径，适配不同使用场景。'
  },
  {
    icon: '02',
    title: '安全验证',
    description: '账号登录结合图形验证码，减少异常请求带来的登录风险。'
  },
  {
    icon: '03',
    title: '清晰反馈',
    description: '所有输入项、按钮状态和错误信息都按统一层级呈现。'
  }
]

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
    const { token, user_name } = response.data.data
    localStorage.setItem('token', token)
    localStorage.setItem('user_name', user_name)
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
    const { token, user_name } = response.data.data
    localStorage.setItem('token', token)
    localStorage.setItem('user_name', user_name)
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
