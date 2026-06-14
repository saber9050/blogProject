<template>
  <AuthShell
    hero-eyebrow="CREATE ACCOUNT"
    hero-title="创建一个更好看的博客身份"
    hero-description="注册页沿用统一设计规范，突出输入节奏、校验反馈与结果状态，让首次注册过程更清晰、更稳定。"
    :hero-highlights="heroHighlights"
    panel-eyebrow="注册新账号"
    panel-title="完成你的基础资料"
    panel-description="昵称和账号支持唯一性检查，密码字段在输入过程中实时反馈格式和一致性。"
    :footer-text="footerText"
  >
    <div class="auth-form">
      <div class="form-note">
        <span class="form-note__icon">i</span>
        <span>字段格式要求来自后端注册结构体，提交前必须全部通过校验。</span>
      </div>

      <form class="auth-form" @submit.prevent="handleRegister">
        <div class="form-stack">
          <div class="field-header">
            <label class="field-label" for="nickname">昵称</label>
            <span class="field-tip">1-15 个字符，需唯一</span>
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
          <p
            class="helper-text"
            :class="{
              'helper-text--error': !!errors.nickname,
              'helper-text--success': !errors.nickname && nicknameStatus === 'valid'
            }"
          >
            {{ errors.nickname ?? (nicknameStatus === 'valid' ? '昵称可用。' : '完成输入后会自动检测是否已存在。') }}
          </p>
        </div>

        <div class="form-stack">
          <div class="field-header">
            <label class="field-label" for="register-account">账号</label>
            <span class="field-tip">11 位数字，需唯一</span>
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
          <p
            class="helper-text"
            :class="{
              'helper-text--error': !!errors.account,
              'helper-text--success': !errors.account && accountStatus === 'valid'
            }"
          >
            {{ errors.account ?? (accountStatus === 'valid' ? '账号可用。' : '账号将作为系统登录标识。') }}
          </p>
        </div>

        <div class="form-stack">
          <div class="field-header">
            <label class="field-label" for="register-password">密码</label>
            <span class="field-tip">11-20 位，需同时包含数字和字母</span>
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
          <p
            class="helper-text"
            :class="{
              'helper-text--error': !!errors.password,
              'helper-text--success': !errors.password && passwordStatus === 'valid'
            }"
          >
            {{ errors.password ?? (passwordStatus === 'valid' ? '密码格式已通过。' : '仅支持数字与字母组合。') }}
          </p>
        </div>

        <div class="form-stack">
          <div class="field-header">
            <label class="field-label" for="register-password-ack">确认密码</label>
            <span class="field-tip">需与上方密码保持一致</span>
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
          <p
            class="helper-text"
            :class="{
              'helper-text--error': !!errors.passwordAck,
              'helper-text--success': !errors.passwordAck && passwordAckStatus === 'valid'
            }"
          >
            {{ errors.passwordAck ?? (passwordAckStatus === 'valid' ? '两次密码输入一致。' : '确认密码会实时检查一致性。') }}
          </p>
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

const footerText = `© ${new Date().getFullYear()} 我的博客. 注册与校验体验已统一升级`
const heroHighlights = [
  {
    icon: '01',
    title: '实时格式校验',
    description: '昵称、账号、密码和确认密码都按统一规范即时反馈。'
  },
  {
    icon: '02',
    title: '唯一性检测',
    description: '昵称和账号在输入完成后即可检查是否已被占用。'
  },
  {
    icon: '03',
    title: '可视化状态',
    description: '通过成功、校验中、错误三种状态，让用户快速理解当前结果。'
  }
]

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
