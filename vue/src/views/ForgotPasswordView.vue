<template>
  <AuthShell
    panel-title="验证邮箱并设置新密码"
  >
    <div class="auth-form">
      <form class="auth-form" @submit.prevent="handleResetPassword">
        <div class="form-stack">
          <div class="field-header">
            <label class="field-label" for="reset-email">邮箱</label>
          </div>
          <div class="input-shell" :class="{ 'input-shell--error': !!errors.email }">
            <span class="input-shell__icon">&#9993;</span>
            <input
              id="reset-email"
              v-model="form.email"
              class="input-shell__control"
              type="email"
              placeholder="请输入邮箱"
              @input="clearError('email')"
            />
          </div>
          <p v-if="errors.email" class="helper-text helper-text--error">{{ errors.email }}</p>
        </div>

        <div class="form-stack">
          <div class="field-header">
            <label class="field-label" for="reset-captcha">邮箱验证码</label>
          </div>
          <div class="input-shell" :class="{ 'input-shell--error': !!errors.captcha }">
            <span class="input-shell__icon">&#128276;</span>
            <input
              id="reset-captcha"
              v-model="form.captcha"
              class="input-shell__control"
              type="text"
              placeholder="请输入 6 位邮箱验证码"
              maxlength="6"
              @input="clearError('captcha')"
            />
            <div class="input-shell__addon">
              <button
                type="button"
                class="inline-action"
                :disabled="isSending || countdown > 0"
                @click="sendCaptcha"
              >
                {{ countdown > 0 ? `${countdown}s` : (isSending ? '发送中' : '获取验证码') }}
              </button>
            </div>
          </div>
          <p v-if="errors.captcha" class="helper-text helper-text--error">{{ errors.captcha }}</p>
        </div>

        <div class="form-stack">
          <div class="field-header">
            <label class="field-label" for="reset-password">新密码</label>
          </div>
          <div class="input-shell" :class="{ 'input-shell--error': !!errors.newPassword }">
            <span class="input-shell__icon">&#128274;</span>
            <input
              id="reset-password"
              v-model="form.newPassword"
              class="input-shell__control"
              type="password"
              placeholder="请输入新密码"
              @input="clearError('newPassword')"
            />
          </div>
          <p v-if="errors.newPassword" class="helper-text helper-text--error">{{ errors.newPassword }}</p>
        </div>

        <div class="form-stack">
          <div class="field-header">
            <label class="field-label" for="reset-password-ack">确认新密码</label>
          </div>
          <div class="input-shell" :class="{ 'input-shell--error': !!errors.ack }">
            <span class="input-shell__icon">&#128274;</span>
            <input
              id="reset-password-ack"
              v-model="form.ack"
              class="input-shell__control"
              type="password"
              placeholder="请再次输入新密码"
              @input="clearError('ack')"
            />
          </div>
          <p v-if="errors.ack" class="helper-text helper-text--error">{{ errors.ack }}</p>
        </div>

        <button type="submit" class="submit-primary" :disabled="loading">
          <span v-if="loading" class="spinner"></span>
          <span>{{ loading ? '提交中...' : '重置密码' }}</span>
        </button>
      </form>
    </div>

    <template #footer>
      <div class="form-links">
        <a href="#" class="text-link" @click.prevent="goLogin">想起密码了？立即登录</a>
        <a href="#" class="text-link" @click.prevent="goRegister">还没有账号？去注册</a>
      </div>
    </template>
  </AuthShell>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import AuthShell from '../components/AuthShell.vue'

interface ResetPasswordForm {
  email: string
  captcha: string
  newPassword: string
  ack: string
}

interface Errors {
  email?: string
  captcha?: string
  newPassword?: string
  ack?: string
}

const router = useRouter()

const form = ref<ResetPasswordForm>({
  email: '',
  captcha: '',
  newPassword: '',
  ack: ''
})

const loading = ref(false)
const isSending = ref(false)
const countdown = ref(0)
const errors = reactive<Errors>({})

const clearError = (field: keyof Errors) => {
  errors[field] = undefined
}

// 发送邮箱验证码
const sendCaptcha = async () => {
  const email = form.value.email
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
      purpose: 'reset_password'
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

const validateForm = (): boolean => {
  let valid = true

  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  if (!form.value.email) {
    errors.email = '邮箱不能为空'
    valid = false
  } else if (!emailRegex.test(form.value.email)) {
    errors.email = '邮箱格式不正确'
    valid = false
  }

  if (!form.value.captcha) {
    errors.captcha = '验证码不能为空'
    valid = false
  } else if (form.value.captcha.length !== 6) {
    errors.captcha = '验证码必须为6位'
    valid = false
  }

  if (!form.value.newPassword) {
    errors.newPassword = '新密码不能为空'
    valid = false
  } else if (form.value.newPassword.length < 6) {
    errors.newPassword = '密码长度至少6位'
    valid = false
  }

  if (!form.value.ack) {
    errors.ack = '请再次输入新密码'
    valid = false
  } else if (form.value.newPassword !== form.value.ack) {
    errors.ack = '两次密码输入不一致'
    valid = false
  }

  return valid
}

const handleResetPassword = async () => {
  if (!validateForm()) return

  loading.value = true
  try {
    await axios.post('/api/v1/auth/reset_password', {
      email: form.value.email,
      captcha: form.value.captcha,
      new_password: form.value.newPassword,
      ack: form.value.ack
    })
    alert('密码重置成功，请使用新密码登录')
    router.push('/login')
  } catch (error: any) {
    console.error('重置密码失败:', error)
    if (error.response?.data?.message) {
      alert(error.response.data.message)
    } else {
      alert('重置密码失败，请稍后重试')
    }
  } finally {
    loading.value = false
  }
}

const goLogin = () => router.push('/login')
const goRegister = () => router.push('/register')
</script>
