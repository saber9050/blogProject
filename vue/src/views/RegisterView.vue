<template>
  <AuthShell
    panel-title="完成你的基础资料"
  >
    <div class="auth-form">
      <form class="auth-form" @submit.prevent="handleRegister">
        <div class="form-stack">
          <div class="field-header">
            <label class="field-label" for="nickname">昵称</label>
          </div>
          <div
            class="input-shell"
            :class="{
              'input-shell--error': nicknameStatus === 'error',
              'input-shell--valid': nicknameStatus === 'valid'
            }"
          >
            <span class="input-shell__icon">&#128100;</span>
            <input
              id="nickname"
              v-model="form.nickname"
              class="input-shell__control"
              type="text"
              placeholder="请输入昵称"
              @input="onNicknameInput"
              @blur="checkNicknameUnique"
            />
            <span v-if="nicknameStatus === 'valid'" class="status-mark status-mark--valid">&#10003;</span>
            <span v-if="nicknameStatus === 'checking'" class="status-mark status-mark--checking">&#8635;</span>
          </div>
          <p v-if="errors.nickname" class="helper-text helper-text--error">{{ errors.nickname }}</p>
        </div>

        <div class="form-stack">
          <div class="field-header">
            <label class="field-label" for="register-account">账号</label>
          </div>
          <div
            class="input-shell"
            :class="{
              'input-shell--error': accountStatus === 'error',
              'input-shell--valid': accountStatus === 'valid'
            }"
          >
            <span class="input-shell__icon">&#128273;</span>
            <input
              id="register-account"
              v-model="form.account"
              class="input-shell__control"
              type="text"
              placeholder="请输入 11 位数字账号"
              maxlength="11"
              @input="onAccountInput"
              @blur="checkAccountUnique"
            />
            <span v-if="accountStatus === 'valid'" class="status-mark status-mark--valid">&#10003;</span>
            <span v-if="accountStatus === 'checking'" class="status-mark status-mark--checking">&#8635;</span>
          </div>
          <p v-if="errors.account" class="helper-text helper-text--error">{{ errors.account }}</p>
        </div>

        <div class="form-stack">
          <div class="field-header">
            <label class="field-label" for="register-password">密码</label>
          </div>
          <div
            class="input-shell"
            :class="{
              'input-shell--error': passwordStatus === 'error',
              'input-shell--valid': passwordStatus === 'valid'
            }"
          >
            <span class="input-shell__icon">&#128274;</span>
            <input
              id="register-password"
              v-model="form.password"
              class="input-shell__control"
              type="password"
              placeholder="请输入密码"
              @input="onPasswordInput"
            />
            <span v-if="passwordStatus === 'valid'" class="status-mark status-mark--valid">&#10003;</span>
          </div>
          <p v-if="errors.password" class="helper-text helper-text--error">{{ errors.password }}</p>
        </div>

        <div class="form-stack">
          <div class="field-header">
            <label class="field-label" for="register-password-ack">确认密码</label>
          </div>
          <div
            class="input-shell"
            :class="{
              'input-shell--error': passwordAckStatus === 'error',
              'input-shell--valid': passwordAckStatus === 'valid'
            }"
          >
            <span class="input-shell__icon">&#128274;</span>
            <input
              id="register-password-ack"
              v-model="form.passwordAck"
              class="input-shell__control"
              type="password"
              placeholder="请再次输入密码"
              @input="onPasswordAckInput"
            />
            <span v-if="passwordAckStatus === 'valid'" class="status-mark status-mark--valid">&#10003;</span>
          </div>
          <p v-if="errors.passwordAck" class="helper-text helper-text--error">{{ errors.passwordAck }}</p>
        </div>

        <button type="submit" class="submit-primary" :disabled="loading || !canSubmit">
          <span v-if="loading" class="spinner"></span>
          <span>{{ loading ? '注册中...' : '创建账号' }}</span>
        </button>
      </form>
    </div>

    <template #footer>
      <div class="form-links">
        <a href="#" class="text-link" @click.prevent="goLogin">已有账号？立即登录</a>
        <a href="#" class="text-link" @click.prevent="goForgotPassword">忘记密码？</a>
      </div>
    </template>
  </AuthShell>
</template>

<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import AuthShell from '../components/AuthShell.vue'

interface RegisterForm {
  nickname: string
  account: string
  password: string
  passwordAck: string
}

interface Errors {
  nickname?: string
  account?: string
  password?: string
  passwordAck?: string
}

type FieldStatus = '' | 'valid' | 'error' | 'checking'

const router = useRouter()

const form = ref<RegisterForm>({
  nickname: '',
  account: '',
  password: '',
  passwordAck: ''
})

const loading = ref(false)
const errors = reactive<Errors>({})

const nicknameStatus = ref<FieldStatus>('')
const accountStatus = ref<FieldStatus>('')
const passwordStatus = ref<FieldStatus>('')
const passwordAckStatus = ref<FieldStatus>('')

// ===== 昵称：实时格式校验 =====
const validateNickname = (): boolean => {
  const val = form.value.nickname.trim()
  if (!val) {
    errors.nickname = '昵称不能为空'
    nicknameStatus.value = 'error'
    return false
  }
  if (val.length < 1 || val.length > 15) {
    errors.nickname = '昵称长度需为1-15个字符'
    nicknameStatus.value = 'error'
    return false
  }
  errors.nickname = undefined
  return true
}

const onNicknameInput = () => {
  validateNickname()
  // 格式变化后重置唯一性状态
  if (nicknameStatus.value !== 'error') {
    nicknameStatus.value = ''
  }
}

// ===== 昵称：失焦后检测唯一性 =====
const checkNicknameUnique = async () => {
  if (!validateNickname()) return
  nicknameStatus.value = 'checking'
  try {
    const res = await axios.get('/api/v1/auth/is_exists_name', {
      params: { user_name: form.value.nickname.trim() }
    })
    if (res.data.data?.is_exists) {
      errors.nickname = '该昵称已被使用，请更换'
      nicknameStatus.value = 'error'
    } else {
      errors.nickname = undefined
      nicknameStatus.value = 'valid'
    }
  } catch {
    errors.nickname = '检测昵称失败，请稍后重试'
    nicknameStatus.value = 'error'
  }
}

// ===== 账号：实时格式校验 =====
const validateAccount = (): boolean => {
  const val = form.value.account.trim()
  if (!val) {
    errors.account = '账号不能为空'
    accountStatus.value = 'error'
    return false
  }
  if (val.length !== 11) {
    errors.account = '账号必须为11位数字'
    accountStatus.value = 'error'
    return false
  }
  if (!/^\d{11}$/.test(val)) {
    errors.account = '账号必须为11位数字'
    accountStatus.value = 'error'
    return false
  }
  errors.account = undefined
  return true
}

const onAccountInput = () => {
  validateAccount()
  if (accountStatus.value !== 'error') {
    accountStatus.value = ''
  }
}

// ===== 账号：失焦后检测唯一性 =====
const checkAccountUnique = async () => {
  if (!validateAccount()) return
  accountStatus.value = 'checking'
  try {
    const res = await axios.get('/api/v1/auth/is_exists_account', {
      params: { account: form.value.account.trim() }
    })
    if (res.data.data?.is_exists) {
      errors.account = '该账号已被注册，请更换'
      accountStatus.value = 'error'
    } else {
      errors.account = undefined
      accountStatus.value = 'valid'
    }
  } catch {
    errors.account = '检测账号失败，请稍后重试'
    accountStatus.value = 'error'
  }
}

// ===== 密码：实时格式校验 =====
const validatePassword = (): boolean => {
  const val = form.value.password
  if (!val) {
    errors.password = '密码不能为空'
    passwordStatus.value = 'error'
    return false
  }
  if (val.length < 11 || val.length > 20) {
    errors.password = '密码长度需为11-20位'
    passwordStatus.value = 'error'
    return false
  }
  // 必须同时包含数字和字母
  if (!/[a-zA-Z]/.test(val) || !/\d/.test(val)) {
    errors.password = '密码必须同时包含数字和字母'
    passwordStatus.value = 'error'
    return false
  }
  if (/[^a-zA-Z0-9]/.test(val)) {
    errors.password = '密码只能包含数字和字母'
    passwordStatus.value = 'error'
    return false
  }
  errors.password = undefined
  passwordStatus.value = 'valid'
  // 密码变更后重新校验确认密码
  if (form.value.passwordAck) {
    validatePasswordAck()
  }
  return true
}

const onPasswordInput = () => {
  validatePassword()
}

// ===== 密码确认：实时格式 + 一致性校验 =====
const validatePasswordAck = (): boolean => {
  const val = form.value.passwordAck
  if (!val) {
    errors.passwordAck = '请再次输入密码'
    passwordAckStatus.value = 'error'
    return false
  }
  if (val.length < 11 || val.length > 20) {
    errors.passwordAck = '密码长度需为11-20位'
    passwordAckStatus.value = 'error'
    return false
  }
  if (form.value.password && val !== form.value.password) {
    errors.passwordAck = '两次密码输入不一致'
    passwordAckStatus.value = 'error'
    return false
  }
  errors.passwordAck = undefined
  passwordAckStatus.value = 'valid'
  return true
}

const onPasswordAckInput = () => {
  validatePasswordAck()
}

// ===== 是否可提交 =====
const canSubmit = computed(() => {
  return (
    nicknameStatus.value === 'valid' &&
    accountStatus.value === 'valid' &&
    passwordStatus.value === 'valid' &&
    passwordAckStatus.value === 'valid' &&
    !errors.nickname &&
    !errors.account &&
    !errors.password &&
    !errors.passwordAck
  )
})

// ===== 提交注册 =====
const handleRegister = async () => {
  // 最终全量校验
  if (!validateNickname()) return
  if (!validateAccount()) return
  if (!validatePassword()) return
  if (!validatePasswordAck()) return

  loading.value = true
  try {
    await axios.post('/api/v1/auth/register', {
      user_name: form.value.nickname.trim(),
      account: form.value.account.trim(),
      password: form.value.password,
      ack: form.value.passwordAck
    })
    alert('注册成功，请登录')
    router.push('/login')
  } catch (error: any) {
    console.error('注册失败:', error)
    if (error.response?.data?.message) {
      alert(error.response.data.message)
    } else {
      alert('注册失败，请稍后重试')
    }
  } finally {
    loading.value = false
  }
}

const goLogin = () => router.push('/login')
const goForgotPassword = () => router.push('/forgot-password')
</script>
