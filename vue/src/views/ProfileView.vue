<template>
  <div class="profile-page">
    <NavBar />
    <main class="page-shell">
      <div class="profile-card">
        <!-- 头像区域 - 可点击更换 -->
        <div class="profile-card__avatar" @click="triggerAvatarUpload" title="点击更换头像">
          <img
              v-if="avatarPreview || userInfo.avatar_url"
              :src="avatarPreview || userInfo.avatar_url"
              class="profile-card__avatar-img"
              alt="头像"
          />
          <span v-else class="profile-card__avatar-placeholder">&#128100;</span>
          <div class="profile-card__avatar-overlay">
            <span>&#128247;</span>
          </div>
        </div>
        <input
            ref="avatarInput"
            type="file"
            accept="image/*"
            style="display: none"
            @change="handleAvatarChange"
        />

        <!-- 用户信息列表 -->
        <div class="profile-card__info-list">
          <!-- 用户名【已修改：移除输入框外壳】 -->
          <div class="info-item">
            <span class="info-item__label">用户名</span>
            <input
                v-model="profileForm.user_name"
                class="plain-input"
                type="text"
                placeholder="请输入用户名"
            />
          </div>

          <!-- 分隔线 -->
          <div class="info-divider" />

          <!-- 邮箱 -->
          <div class="info-item">
            <span class="info-item__label">邮箱</span>
            <span class="info-item__value">{{ userInfo.email || '未设置' }}</span>
            <button
                v-if="userInfo.email"
                class="info-item__action"
                @click="showEmailModal = true"
            >修改邮箱</button>
            <button
                v-else
                class="info-item__action"
                @click="showAddEmailModal = true"
            >添加邮箱</button>
          </div>

          <!-- 分隔线 -->
          <div class="info-divider" />

          <!-- 密码 -->
          <div class="info-item">
            <span class="info-item__label">密码</span>
            <span class="info-item__value info-item__value--masked">********</span>
            <button class="info-item__action" @click="showPasswordModal = true">修改密码</button>
          </div>

          <template v-if="userInfo.role_id === 1">
            <!-- 分隔线 -->
            <div class="info-divider" />

            <!-- 个人简介【已修改：移除输入框外壳】 -->
            <div class="info-item info-item--bio">
              <span class="info-item__label">个人简介</span>
              <input
                  v-model="profileForm.introduction"
                  class="plain-input"
                  type="text"
                  placeholder="这个人很懒，什么都没写~"
              />
            </div>

            <!-- 分隔线 -->
            <div class="info-divider" />
          </template>

          <!-- 保存修改按钮 -->
          <div class="profile-card__save-row">
            <button class="save-btn" :disabled="saving" @click="handleSaveProfile">
              <span v-if="saving" class="spinner"></span>
              <span>{{ saving ? '保存中...' : '保存修改' }}</span>
            </button>
          </div>
        </div>
      </div>
    </main>

    <!-- ==================== 修改邮箱弹窗 ==================== -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="showEmailModal" class="modal-overlay" @click.self="closeEmailModal">
          <div class="modal-panel">
            <div class="modal-panel__header">
              <h3 class="modal-panel__title">修改邮箱</h3>
              <button class="modal-panel__close" @click="closeEmailModal">&times;</button>
            </div>

            <form class="modal-panel__body" @submit.prevent="handleChangeEmail">
              <!-- 当前邮箱（只读） -->
              <div class="form-stack">
                <div class="field-header">
                  <label class="field-label">当前邮箱</label>
                </div>
                <div class="input-shell input-shell--readonly">
                  <span class="input-shell__icon">&#9993;</span>
                  <input
                      class="input-shell__control"
                      type="email"
                      :value="userInfo.email"
                      readonly
                      disabled
                  />
                </div>
              </div>

              <!-- 邮箱验证码 -->
              <div class="form-stack">
                <div class="field-header">
                  <label class="field-label">邮箱验证码</label>
                </div>
                <div class="input-shell" :class="{ 'input-shell--error': !!emailErrors.captcha }">
                  <span class="input-shell__icon">&#128276;</span>
                  <input
                      v-model="emailForm.captcha"
                      class="input-shell__control"
                      type="text"
                      placeholder="请输入6位邮箱验证码"
                      maxlength="6"
                      @input="clearEmailError('captcha')"
                  />
                  <div class="input-shell__addon">
                    <button
                        type="button"
                        class="inline-action"
                        :disabled="emailSending || emailCountdown > 0 || emailErrors.newEmail === '该邮箱已被使用，请更换'"
                        @click="sendEmailCaptcha('reset_email')"
                    >
                      {{ emailCountdown > 0 ? `${emailCountdown}s` : (emailSending ? '发送中' : '获取验证码') }}
                    </button>
                  </div>
                </div>
                <p v-if="emailErrors.captcha" class="helper-text helper-text--error">{{ emailErrors.captcha }}</p>
              </div>

              <!-- 当前密码 -->
              <div class="form-stack">
                <div class="field-header">
                  <label class="field-label">当前密码</label>
                </div>
                <div class="input-shell" :class="{ 'input-shell--error': !!emailErrors.password }">
                  <span class="input-shell__icon">&#128274;</span>
                  <input
                      v-model="emailForm.password"
                      class="input-shell__control"
                      type="password"
                      placeholder="请输入当前密码"
                      @input="clearEmailError('password')"
                  />
                </div>
                <p v-if="emailErrors.password" class="helper-text helper-text--error">{{ emailErrors.password }}</p>
              </div>

              <!-- 新邮箱 -->
              <div class="form-stack">
                <div class="field-header">
                  <label class="field-label">新邮箱</label>
                </div>
                <div class="input-shell" :class="{ 'input-shell--error': !!emailErrors.newEmail }">
                  <span class="input-shell__icon">&#9993;</span>
                  <input
                      v-model="emailForm.newEmail"
                      class="input-shell__control"
                      type="email"
                      placeholder="请输入新邮箱"
                      @input="clearEmailError('newEmail')"
                      @blur="checkEmailUnique"
                  />
                </div>
                <p v-if="emailErrors.newEmail" class="helper-text helper-text--error">{{ emailErrors.newEmail }}</p>
              </div>

              <button type="submit" class="submit-primary" :disabled="emailLoading || emailErrors.newEmail === '该邮箱已被使用，请更换'">
                <span v-if="emailLoading" class="spinner"></span>
                <span>{{ emailLoading ? '提交中...' : '确认修改' }}</span>
              </button>
            </form>
          </div>
        </div>
      </Transition>
    </Teleport>

    <!-- ==================== 添加邮箱弹窗（邮箱为空时使用） ==================== -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="showAddEmailModal" class="modal-overlay" @click.self="closeAddEmailModal">
          <div class="modal-panel">
            <div class="modal-panel__header">
              <h3 class="modal-panel__title">添加邮箱</h3>
              <button class="modal-panel__close" @click="closeAddEmailModal">&times;</button>
            </div>

            <form class="modal-panel__body" @submit.prevent="handleAddEmail">
              <!-- 新邮箱 -->
              <div class="form-stack">
                <div class="field-header">
                  <label class="field-label">新邮箱</label>
                </div>
                <div class="input-shell" :class="{ 'input-shell--error': !!addEmailErrors.newEmail }">
                  <span class="input-shell__icon">&#9993;</span>
                  <input
                      v-model="addEmailForm.newEmail"
                      class="input-shell__control"
                      type="email"
                      placeholder="请输入邮箱地址"
                      @input="clearAddEmailError('newEmail')"
                      @blur="checkAddEmailUnique"
                  />
                </div>
                <p v-if="addEmailErrors.newEmail" class="helper-text helper-text--error">{{ addEmailErrors.newEmail }}</p>
              </div>

              <!-- 邮箱验证码 -->
              <div class="form-stack">
                <div class="field-header">
                  <label class="field-label">邮箱验证码</label>
                </div>
                <div class="input-shell" :class="{ 'input-shell--error': !!addEmailErrors.captcha }">
                  <span class="input-shell__icon">&#128276;</span>
                  <input
                      v-model="addEmailForm.captcha"
                      class="input-shell__control"
                      type="text"
                      placeholder="请输入6位邮箱验证码"
                      maxlength="6"
                      @input="clearAddEmailError('captcha')"
                  />
                  <div class="input-shell__addon">
                    <button
                        type="button"
                        class="inline-action"
                        :disabled="addEmailSending || addEmailCountdown > 0 || addEmailErrors.newEmail === '该邮箱已被使用，请更换'"
                        @click="sendAddEmailCaptcha"
                    >
                      {{ addEmailCountdown > 0 ? `${addEmailCountdown}s` : (addEmailSending ? '发送中' : '获取验证码') }}
                    </button>
                  </div>
                </div>
                <p v-if="addEmailErrors.captcha" class="helper-text helper-text--error">{{ addEmailErrors.captcha }}</p>
              </div>

              <button type="submit" class="submit-primary" :disabled="addEmailLoading || addEmailErrors.newEmail === '该邮箱已被使用，请更换'">
                <span v-if="addEmailLoading" class="spinner"></span>
                <span>{{ addEmailLoading ? '提交中...' : '确认添加' }}</span>
              </button>
            </form>
          </div>
        </div>
      </Transition>
    </Teleport>

    <!-- ==================== 修改密码弹窗 ==================== -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="showPasswordModal" class="modal-overlay" @click.self="closePasswordModal">
          <div class="modal-panel">
            <div class="modal-panel__header">
              <h3 class="modal-panel__title">修改密码</h3>
              <button class="modal-panel__close" @click="closePasswordModal">&times;</button>
            </div>

            <!-- 无邮箱：提示先添加邮箱 -->
            <div v-if="!userInfo.email" class="modal-panel__body">
              <div class="no-email-prompt">
                <p class="no-email-prompt__text">请先添加邮箱，才能修改密码</p>
                <button type="button" class="submit-primary" @click="openAddEmailFromPassword">
                  去添加邮箱
                </button>
              </div>
            </div>

            <!-- 有邮箱：正常表单 -->
            <form v-else class="modal-panel__body" @submit.prevent="handleChangePassword">
              <!-- 邮箱验证码 -->
              <div class="form-stack">
                <div class="field-header">
                  <label class="field-label">邮箱验证码</label>
                </div>
                <div class="input-shell" :class="{ 'input-shell--error': !!passwordErrors.captcha }">
                  <span class="input-shell__icon">&#128276;</span>
                  <input
                      v-model="passwordForm.captcha"
                      class="input-shell__control"
                      type="text"
                      placeholder="请输入6位邮箱验证码"
                      maxlength="6"
                      @input="clearPasswordError('captcha')"
                  />
                  <div class="input-shell__addon">
                    <button
                        type="button"
                        class="inline-action"
                        :disabled="passwordSending || passwordCountdown > 0"
                        @click="sendPasswordCaptcha"
                    >
                      {{ passwordCountdown > 0 ? `${passwordCountdown}s` : (passwordSending ? '发送中' : '获取验证码') }}
                    </button>
                  </div>
                </div>
                <p v-if="passwordErrors.captcha" class="helper-text helper-text--error">{{ passwordErrors.captcha }}</p>
              </div>

              <!-- 新密码（带实时校验） -->
              <div class="form-stack">
                <div class="field-header">
                  <label class="field-label">新密码</label>
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
                      v-model="passwordForm.newPassword"
                      class="input-shell__control"
                      type="password"
                      placeholder="请输入新密码"
                      @input="onPasswordInput"
                  />
                  <span v-if="passwordStatus === 'valid'" class="status-mark status-mark--valid">&#10003;</span>
                </div>
                <p v-if="passwordErrors.newPassword" class="helper-text helper-text--error">{{ passwordErrors.newPassword }}</p>
                <p v-else class="helper-text helper-text--hint">密码需 11-20 位，必须同时包含数字和字母</p>
              </div>

              <!-- 确认新密码 -->
              <div class="form-stack">
                <div class="field-header">
                  <label class="field-label">确认新密码</label>
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
                      v-model="passwordForm.ack"
                      class="input-shell__control"
                      type="password"
                      placeholder="请再次输入新密码"
                      @input="onPasswordAckInput"
                  />
                  <span v-if="passwordAckStatus === 'valid'" class="status-mark status-mark--valid">&#10003;</span>
                </div>
                <p v-if="passwordErrors.ack" class="helper-text helper-text--error">{{ passwordErrors.ack }}</p>
              </div>

              <button type="submit" class="submit-primary" :disabled="passwordLoading">
                <span v-if="passwordLoading" class="spinner"></span>
                <span>{{ passwordLoading ? '提交中...' : '确认修改' }}</span>
              </button>
            </form>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import api from '../api'
import NavBar from '../components/NavBar.vue'
import { showToast } from '../utils/toast'

interface UserInfo {
  user_id: number
  user_name: string
  account: string
  email: string
  avatar_url: string
  introduction: string
  role_id: number
  staus: number
  create_at: string
}

interface ProfileForm {
  user_name: string
  introduction: string
}

interface EmailForm {
  captcha: string
  password: string
  newEmail: string
}

interface PasswordForm {
  captcha: string
  newPassword: string
  ack: string
}

interface EmailErrors {
  captcha?: string
  password?: string
  newEmail?: string
}

interface PasswordErrors {
  captcha?: string
  newPassword?: string
  ack?: string
}

const router = useRouter()

const userInfo = ref<UserInfo>({
  user_id: 0,
  user_name: '',
  account: '',
  email: '',
  avatar_url: '',
  introduction: '',
  role_id: 0,
  staus: 0,
  create_at: ''
})

const profileForm = ref<ProfileForm>({
  user_name: '',
  introduction: ''
})

const saving = ref(false)

// 触发用户信息更新事件，通知其他组件（如 NavBar）
const emitUserInfoUpdated = () => {
  window.dispatchEvent(new CustomEvent('user-info-updated', {
    detail: userInfo.value
  }))
}

// ==================== 头像上传 ====================
const avatarInput = ref<HTMLInputElement | null>(null)
const avatarPreview = ref('')
const avatarUploading = ref(false)

const triggerAvatarUpload = () => {
  avatarInput.value?.click()
}

const handleAvatarChange = async (e: Event) => {
  const target = e.target as HTMLInputElement
  const file = target.files?.[0]
  if (!file) return

  // 本地预览
  avatarPreview.value = URL.createObjectURL(file)

  avatarUploading.value = true
  try {
    const formData = new FormData()
    formData.append('file', file)
    const res = await api.post('/user/avatar', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
    if (res.data.data) {
      userInfo.value.avatar_url = res.data.data.avatar_url
      // 同步更新 localStorage
      const stored = localStorage.getItem('user')
      if (stored) {
        try {
          const u = JSON.parse(stored)
          u.avatar_url = res.data.data.avatar_url
          localStorage.setItem('user', JSON.stringify(u))
          // 触发事件通知其他组件（如 NavBar）
          emitUserInfoUpdated()
        } catch { /* ignore */ }
      }
    }
  } catch (error: any) {
    const msg = error.response?.data?.message || '头像上传失败'
    showToast(msg, 'error')
    avatarPreview.value = ''
  } finally {
    avatarUploading.value = false
    target.value = ''
  }
}

// ==================== 保存修改（用户名 + 个人简介） ====================
const handleSaveProfile = async () => {
  saving.value = true
  try {
    await api.post('/user/profile', {
      nick_name: profileForm.value.user_name,
      introduction: profileForm.value.introduction
    })
    userInfo.value.user_name = profileForm.value.user_name
    userInfo.value.introduction = profileForm.value.introduction
    // 同步更新 localStorage
    const stored = localStorage.getItem('user')
    if (stored) {
      try {
        const u = JSON.parse(stored)
        u.user_name = profileForm.value.user_name
        localStorage.setItem('user', JSON.stringify(u))
        // 触发事件通知其他组件（如 NavBar）
        emitUserInfoUpdated()
      } catch { /* ignore */ }
    }
    showToast('保存成功', 'success')
  } catch (error: any) {
    const msg = error.response?.data?.message || '保存失败，请稍后重试'
    showToast(msg, 'error')
  } finally {
    saving.value = false
  }
}

// ==================== 修改邮箱 ====================
const showEmailModal = ref(false)
const emailLoading = ref(false)
const emailSending = ref(false)
const emailCountdown = ref(0)
const emailForm = ref<EmailForm>({ captcha: '', password: '', newEmail: '' })
const emailErrors = reactive<EmailErrors>({})

const clearEmailError = (field: keyof EmailErrors) => {
  emailErrors[field] = undefined
}

const sendEmailCaptcha = async (purpose: string) => {
  if (!userInfo.value.email) {
    emailErrors.captcha = '邮箱信息异常'
    return
  }

  emailSending.value = true
  try {
    await api.post('/auth/captcha', {
      email: userInfo.value.email,
      purpose: purpose
    })

    emailCountdown.value = 60
    const timer = setInterval(() => {
      emailCountdown.value--
      if (emailCountdown.value <= 0) {
        clearInterval(timer)
      }
    }, 1000)

    showToast('验证码已发送到您的邮箱，请注意查收', 'success')
  } catch (error: any) {
    const msg = error.response?.data?.message || '发送验证码失败，请稍后重试'
    showToast(msg, 'error')
  } finally {
    emailSending.value = false
  }
}

const checkEmailUnique = async () => {
  const val = emailForm.value.newEmail.trim()
  if (!val) return
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  if (!emailRegex.test(val)) return

  try {
    const res = await api.get('/user/is_exists_email', { params: { new_email: val } })
    if (res.data.data?.is_exists) {
      emailErrors.newEmail = '该邮箱已被使用，请更换'
    } else {
      emailErrors.newEmail = undefined
    }
  } catch {
    // 接口异常时跳过检查
  }
}

const validateEmailForm = (): boolean => {
  let valid = true

  if (!emailForm.value.captcha) {
    emailErrors.captcha = '验证码不能为空'
    valid = false
  } else if (emailForm.value.captcha.length !== 6) {
    emailErrors.captcha = '验证码必须为6位'
    valid = false
  }

  if (!emailForm.value.password) {
    emailErrors.password = '当前密码不能为空'
    valid = false
  }

  // 如果失焦检查已经发现邮箱被占用，直接阻止提交
  if (emailErrors.newEmail === '该邮箱已被使用，请更换') {
    valid = false
  }

  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  if (!emailForm.value.newEmail) {
    emailErrors.newEmail = '新邮箱不能为空'
    valid = false
  } else if (!emailRegex.test(emailForm.value.newEmail)) {
    emailErrors.newEmail = '邮箱格式不正确'
    valid = false
  }

  return valid
}

const handleChangeEmail = async () => {
  if (!validateEmailForm()) return

  emailLoading.value = true
  try {
    await api.post('/user/email_ack', {
      captcha: emailForm.value.captcha,
      password: emailForm.value.password,
      new_email: emailForm.value.newEmail
    })
    showToast('修改邮箱请求已提交，请前往新邮箱查收确认邮件完成修改', 'success')
    closeEmailModal()
  } catch (error: any) {
    const msg = error.response?.data?.message || '修改邮箱失败，请稍后重试'
    showToast(msg, 'error')
  } finally {
    emailLoading.value = false
  }
}

const closeEmailModal = () => {
  showEmailModal.value = false
  emailForm.value = { captcha: '', password: '', newEmail: '' }
  emailErrors.captcha = undefined
  emailErrors.password = undefined
  emailErrors.newEmail = undefined
}

// ==================== 添加邮箱 ====================
const showAddEmailModal = ref(false)
const addEmailLoading = ref(false)
const addEmailSending = ref(false)
const addEmailCountdown = ref(0)
const addEmailForm = ref<{ newEmail: string; captcha: string }>({ newEmail: '', captcha: '' })
const addEmailErrors = reactive<{ newEmail?: string; captcha?: string }>({})

const clearAddEmailError = (field: keyof typeof addEmailErrors) => {
  addEmailErrors[field] = undefined
}

const sendAddEmailCaptcha = async () => {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  if (!addEmailForm.value.newEmail) {
    addEmailErrors.newEmail = '请先输入邮箱地址'
    return
  }
  if (!emailRegex.test(addEmailForm.value.newEmail)) {
    addEmailErrors.newEmail = '邮箱格式不正确'
    return
  }

  addEmailSending.value = true
  try {
    await api.post('/auth/captcha', {
      email: addEmailForm.value.newEmail,
      purpose: 'add_email'
    })

    addEmailCountdown.value = 60
    const timer = setInterval(() => {
      addEmailCountdown.value--
      if (addEmailCountdown.value <= 0) {
        clearInterval(timer)
      }
    }, 1000)

    showToast('验证码已发送到您的邮箱，请注意查收', 'success')
  } catch (error: any) {
    const msg = error.response?.data?.message || '发送验证码失败，请稍后重试'
    showToast(msg, 'error')
  } finally {
    addEmailSending.value = false
  }
}

const checkAddEmailUnique = async () => {
  const val = addEmailForm.value.newEmail.trim()
  if (!val) return
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  if (!emailRegex.test(val)) return

  try {
    const res = await api.get('/user/is_exists_email', { params: { new_email: val } })
    if (res.data.data?.is_exists) {
      addEmailErrors.newEmail = '该邮箱已被使用，请更换'
    } else {
      addEmailErrors.newEmail = undefined
    }
  } catch {
    // 接口异常时跳过检查
  }
}

const validateAddEmailForm = (): boolean => {
  let valid = true

  // 如果失焦检查已经发现邮箱被占用，直接阻止提交
  if (addEmailErrors.newEmail === '该邮箱已被使用，请更换') {
    valid = false
  }

  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  if (!addEmailForm.value.newEmail) {
    addEmailErrors.newEmail = '邮箱不能为空'
    valid = false
  } else if (!emailRegex.test(addEmailForm.value.newEmail)) {
    addEmailErrors.newEmail = '邮箱格式不正确'
    valid = false
  }

  if (!addEmailForm.value.captcha) {
    addEmailErrors.captcha = '验证码不能为空'
    valid = false
  } else if (addEmailForm.value.captcha.length !== 6) {
    addEmailErrors.captcha = '验证码必须为6位'
    valid = false
  }

  return valid
}

const handleAddEmail = async () => {
  if (!validateAddEmailForm()) return

  addEmailLoading.value = true
  try {
    await api.post('/user/add_email', {
      new_email: addEmailForm.value.newEmail,
      captcha: addEmailForm.value.captcha
    })
    userInfo.value.email = addEmailForm.value.newEmail
    showToast('邮箱添加成功', 'success')
    closeAddEmailModal()
  } catch (error: any) {
    const msg = error.response?.data?.message || '添加邮箱失败，请稍后重试'
    showToast(msg, 'error')
  } finally {
    addEmailLoading.value = false
  }
}

const closeAddEmailModal = () => {
  showAddEmailModal.value = false
  addEmailForm.value = { newEmail: '', captcha: '' }
  addEmailErrors.newEmail = undefined
  addEmailErrors.captcha = undefined
}

// ==================== 修改密码 ====================
const showPasswordModal = ref(false)
const passwordLoading = ref(false)
const passwordSending = ref(false)
const passwordCountdown = ref(0)
const passwordStatus = ref<'' | 'valid' | 'error'>('')
const passwordAckStatus = ref<'' | 'valid' | 'error'>('')
const passwordForm = ref<PasswordForm>({ captcha: '', newPassword: '', ack: '' })
const passwordErrors = reactive<PasswordErrors>({})

const clearPasswordError = (field: keyof PasswordErrors) => {
  passwordErrors[field] = undefined
}

// ===== 新密码：实时校验（同注册页规则） =====
const validatePasswordStrength = (): boolean => {
  const val = passwordForm.value.newPassword
  if (!val) {
    passwordErrors.newPassword = '新密码不能为空'
    passwordStatus.value = 'error'
    return false
  }
  if (val.length < 11 || val.length > 20) {
    passwordErrors.newPassword = '密码长度需为11-20位'
    passwordStatus.value = 'error'
    return false
  }
  if (!/[a-zA-Z]/.test(val) || !/\d/.test(val)) {
    passwordErrors.newPassword = '密码必须同时包含数字和字母'
    passwordStatus.value = 'error'
    return false
  }
  if (/[^a-zA-Z0-9]/.test(val)) {
    passwordErrors.newPassword = '密码只能包含数字和字母'
    passwordStatus.value = 'error'
    return false
  }
  passwordErrors.newPassword = undefined
  passwordStatus.value = 'valid'
  // 密码变更后重新校验确认密码
  if (passwordForm.value.ack) {
    validatePasswordAck()
  }
  return true
}

const onPasswordInput = () => {
  validatePasswordStrength()
}

// ===== 确认密码：实时校验 =====
const validatePasswordAck = (): boolean => {
  const val = passwordForm.value.ack
  if (!val) {
    passwordErrors.ack = '请再次输入新密码'
    passwordAckStatus.value = 'error'
    return false
  }
  if (val.length < 11 || val.length > 20) {
    passwordErrors.ack = '密码长度需为11-20位'
    passwordAckStatus.value = 'error'
    return false
  }
  if (passwordForm.value.newPassword && val !== passwordForm.value.newPassword) {
    passwordErrors.ack = '两次密码输入不一致'
    passwordAckStatus.value = 'error'
    return false
  }
  passwordErrors.ack = undefined
  passwordAckStatus.value = 'valid'
  return true
}

const onPasswordAckInput = () => {
  validatePasswordAck()
}

const sendPasswordCaptcha = async () => {
  if (!userInfo.value.email) {
    passwordErrors.captcha = '邮箱信息异常'
    return
  }

  passwordSending.value = true
  try {
    await api.post('/auth/captcha', {
      email: userInfo.value.email,
      purpose: 'reset_password'
    })

    passwordCountdown.value = 60
    const timer = setInterval(() => {
      passwordCountdown.value--
      if (passwordCountdown.value <= 0) {
        clearInterval(timer)
      }
    }, 1000)

    showToast('验证码已发送到您的邮箱，请注意查收', 'success')
  } catch (error: any) {
    const msg = error.response?.data?.message || '发送验证码失败，请稍后重试'
    showToast(msg, 'error')
  } finally {
    passwordSending.value = false
  }
}

const validatePasswordForm = (): boolean => {
  let valid = true

  if (!passwordForm.value.captcha) {
    passwordErrors.captcha = '验证码不能为空'
    valid = false
  } else if (passwordForm.value.captcha.length !== 6) {
    passwordErrors.captcha = '验证码必须为6位'
    valid = false
  }

  if (!validatePasswordStrength()) valid = false
  if (!validatePasswordAck()) valid = false

  return valid
}

const handleChangePassword = async () => {
  if (!validatePasswordForm()) return

  passwordLoading.value = true
  try {
    await api.post('/auth/reset_password', {
      email: userInfo.value.email,
      captcha: passwordForm.value.captcha,
      new_password: passwordForm.value.newPassword,
      ack: passwordForm.value.ack
    })
    showToast('密码修改成功', 'success')
    closePasswordModal()
    router.push('/login')
  } catch (error: any) {
    const msg = error.response?.data?.message || '修改密码失败，请稍后重试'
    showToast(msg, 'error')
  } finally {
    passwordLoading.value = false
  }
}

const openAddEmailFromPassword = () => {
  showPasswordModal.value = false
  showAddEmailModal.value = true
}

const closePasswordModal = () => {
  showPasswordModal.value = false
  passwordForm.value = { captcha: '', newPassword: '', ack: '' }
  passwordErrors.captcha = undefined
  passwordErrors.newPassword = undefined
  passwordErrors.ack = undefined
  passwordStatus.value = ''
  passwordAckStatus.value = ''
}

// ==================== 获取用户信息 ====================
onMounted(async () => {
  try {
    const res = await api.get('/user/info')
    if (res.data.data) {
      userInfo.value = res.data.data
      profileForm.value.user_name = res.data.data.user_name
      profileForm.value.introduction = res.data.data.introduction || ''
    }
  } catch {
    console.error('获取用户信息失败')
  }
})
</script>

<style scoped>
/* ========== 页面外壳 ========== */
.page-shell {
  max-width: 600px;
  margin: 32px auto;
  padding: 0 24px;
}

/* ========== 个人信息卡片 ========== */
.profile-card {
  background: rgba(255, 255, 255, 0.85);
  backdrop-filter: blur(16px);
  border: 1px solid rgba(255, 255, 255, 0.55);
  border-radius: var(--radius-xl);
  box-shadow: 0 4px 18px rgba(15, 23, 42, 0.04);
  padding: 36px 32px 28px;
}

/* ========== 头像 - 可点击 ========== */
.profile-card__avatar {
  display: flex;
  justify-content: center;
  margin-bottom: 28px;
  position: relative;
  cursor: pointer;
}

.profile-card__avatar-img {
  width: 88px;
  height: 88px;
  border-radius: 50%;
  object-fit: cover;
  border: 3px solid rgba(16, 185, 129, 0.18);
  box-shadow: 0 4px 14px rgba(16, 185, 129, 0.1);
  transition: opacity 0.24s ease;
}

.profile-card__avatar-placeholder {
  width: 88px;
  height: 88px;
  border-radius: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--primary-soft);
  color: var(--primary-700);
  font-size: 2rem;
}

.profile-card__avatar-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.4rem;
  color: #fff;
  opacity: 0;
  transition: opacity 0.24s ease;
}

.profile-card__avatar-overlay::before {
  content: '';
  position: absolute;
  width: 88px;
  height: 88px;
  border-radius: 50%;
  background: rgba(15, 23, 42, 0.35);
}

.profile-card__avatar-overlay span {
  position: relative;
  z-index: 1;
}

.profile-card__avatar:hover .profile-card__avatar-overlay {
  opacity: 1;
}

.profile-card__avatar:hover .profile-card__avatar-img {
  opacity: 0.75;
}

/* ========== 信息列表 ========== */
.profile-card__info-list {
  display: flex;
  flex-direction: column;
}

.info-item {
  display: flex;
  align-items: center;
  padding: 14px 0;
  gap: 14px;
}

.info-item--bio {
  align-items: center;
}

.info-item__label {
  font-size: 0.88rem;
  font-weight: 700;
  color: var(--text-muted);
  flex-shrink: 0;
  min-width: 56px;
}

.info-item__value {
  font-size: 0.95rem;
  color: var(--text-strong);
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.info-item__value--masked {
  letter-spacing: 0.12em;
}

.info-item__action {
  flex-shrink: 0;
  height: 34px;
  padding: 0 16px;
  border: none;
  border-radius: 12px;
  background: linear-gradient(135deg, var(--primary-600), var(--primary-500));
  color: #fff;
  font-size: 0.8rem;
  font-weight: 700;
  box-shadow: 0 8px 18px rgba(16, 185, 129, 0.16);
  transition: transform var(--transition-base), box-shadow var(--transition-base);
}

.info-item__action:hover {
  transform: translateY(-1px);
  box-shadow: 0 12px 22px rgba(16, 185, 129, 0.22);
}

.info-divider {
  height: 1px;
  background: rgba(148, 163, 184, 0.14);
}

/* ========== 无框单行输入框【新增样式】 ========== */
.plain-input {
  flex: 1;
  min-height: 42px;
  padding: 0 4px;
  font-size: 0.95rem;
  color: var(--text-strong);
  background: transparent;
  border: none;
  border-bottom: 1px solid transparent;
  outline: none;
  transition: border-color 0.24s ease;
}
/* 聚焦时下划线高亮，提示可编辑 */
.plain-input:focus {
  border-bottom-color: var(--primary-500);
}
.plain-input::placeholder {
  color: var(--text-muted);
}

/* ========== 保存按钮 ========== */
.profile-card__save-row {
  display: flex;
  justify-content: center;
  padding-top: 10px;
}

.save-btn {
  min-height: 48px;
  min-width: 200px;
  padding: 0 32px;
  border: none;
  border-radius: var(--radius-lg);
  background: linear-gradient(135deg, var(--primary-600), var(--primary-500));
  color: #fff;
  font-size: 0.95rem;
  font-weight: 700;
  box-shadow: 0 12px 28px rgba(16, 185, 129, 0.2);
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  transition: transform var(--transition-base), box-shadow var(--transition-base), opacity var(--transition-base);
}

.save-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 18px 32px rgba(16, 185, 129, 0.28);
}

.save-btn:disabled {
  opacity: 0.6;
  box-shadow: none;
  cursor: not-allowed;
}

/* ========== 弹窗覆盖层 ========== */
.modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 200;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(15, 23, 42, 0.32);
  backdrop-filter: blur(6px);
  padding: 20px;
}

.modal-panel {
  width: 100%;
  max-width: 440px;
  background: var(--surface-strong);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-md);
  overflow: hidden;
}

.modal-panel__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px 24px 0;
}

.modal-panel__title {
  font-size: 1.1rem;
  font-weight: 700;
  color: var(--text-strong);
  margin: 0;
}

.modal-panel__close {
  width: 36px;
  height: 36px;
  border: none;
  border-radius: 12px;
  background: rgba(148, 163, 184, 0.1);
  color: var(--text-muted);
  font-size: 1.2rem;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition: background var(--transition-base), color var(--transition-base);
}

.modal-panel__close:hover {
  background: rgba(220, 38, 38, 0.1);
  color: var(--danger);
}

.modal-panel__body {
  padding: 20px 24px 28px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

/* ========== 无邮箱提示 ========== */
.no-email-prompt {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
  padding: 32px 0;
}

.no-email-prompt__text {
  margin: 0;
  color: var(--text-muted);
  font-size: 0.95rem;
  text-align: center;
}

.no-email-prompt .submit-primary {
  align-self: center;
}

/* ========== 只读输入框 ========== */
.input-shell--readonly {
  opacity: 0.7;
}

.input-shell--readonly .input-shell__control {
  color: var(--text-muted);
  cursor: not-allowed;
}

/* ========== 提交按钮 ========== */
.submit-primary {
  min-height: 50px;
  width: 100%;
  border: none;
  border-radius: var(--radius-lg);
  background: linear-gradient(135deg, var(--primary-600), var(--primary-500));
  color: #fff;
  font-size: 0.98rem;
  font-weight: 700;
  box-shadow: 0 12px 28px rgba(16, 185, 129, 0.2);
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  transition: transform var(--transition-base), box-shadow var(--transition-base), opacity var(--transition-base);
  margin-top: 6px;
}

.submit-primary:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 18px 32px rgba(16, 185, 129, 0.28);
}

.submit-primary:disabled {
  opacity: 0.6;
  box-shadow: none;
  cursor: not-allowed;
}

/* ========== 加载旋转器 ========== */
.spinner {
  width: 18px;
  height: 18px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* ========== 弹窗过渡动画 ========== */
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.24s ease;
}

.modal-enter-active .modal-panel,
.modal-leave-active .modal-panel {
  transition: transform 0.24s ease, opacity 0.24s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .modal-panel {
  transform: scale(0.95) translateY(8px);
  opacity: 0;
}

.modal-leave-to .modal-panel {
  transform: scale(0.95) translateY(8px);
  opacity: 0;
}
</style>