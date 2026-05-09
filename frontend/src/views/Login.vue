<template>
  <div class="login-container">
    <div class="login-bg-shapes">
      <div class="shape shape-1"></div>
      <div class="shape shape-2"></div>
      <div class="shape shape-3"></div>
      <div class="shape shape-4"></div>
    </div>
    <div class="login-card">
      <div class="login-header">
        <div class="login-logo">Z</div>
        <h2 class="login-title">ZAPI</h2>
        <p class="login-subtitle">AI Gateway Management Platform</p>
      </div>
      <el-form :model="form" @submit.prevent="handleLogin" label-position="top">
        <el-form-item :label="t('login.username')">
          <el-input v-model="form.username" :prefix-icon="'User'" size="large" />
        </el-form-item>
        <el-form-item :label="t('login.password')">
          <el-input v-model="form.password" type="password" show-password :prefix-icon="'Lock'" size="large" />
        </el-form-item>
        <el-form-item :label="t('login.captcha')">
          <div class="captcha-row">
            <el-input v-model="form.captcha_code" @keyup.enter="handleLogin" size="large" />
            <img v-if="captchaImg" :src="captchaImg" class="captcha-img" @click="loadCaptcha" :alt="'点击刷新'" />
          </div>
        </el-form-item>
        <el-button type="primary" style="width: 100%; margin-top: 4px" size="large" @click="handleLogin" :loading="loading">{{ t('login.submit') }}</el-button>
        <div v-if="allowRegister" style="text-align: center; margin-top: 16px">
          <router-link to="/register">{{ t('login.noAccount') }} {{ t('login.register') }}</router-link>
        </div>
      </el-form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { settingsApi } from '@/api/settings'

const { t } = useI18n()
const router = useRouter()
const auth = useAuthStore()
const form = ref({ username: '', password: '', captcha_id: '', captcha_code: '' })
const loading = ref(false)
const captchaImg = ref('')
const allowRegister = ref(true)

async function loadCaptcha() {
  try {
    const r = await fetch('/api/auth/captcha')
    const id = r.headers.get('x-captcha-id')
    const blob = await r.blob()
    if (id) form.value.captcha_id = id
    captchaImg.value = URL.createObjectURL(blob)
  } catch (e) { console.error('captcha error', e) }
}

onMounted(async () => {
  try { const d = await settingsApi.public(); allowRegister.value = d.allow_register } catch {}
  loadCaptcha()
})

async function handleLogin() {
  if (!form.value.username || !form.value.password) { ElMessage.warning('请输入用户名和密码'); return }
  if (!form.value.captcha_code) { ElMessage.warning('请输入验证码'); return }
  loading.value = true
  try {
    await auth.login(form.value.username, form.value.password, form.value.captcha_id, form.value.captcha_code)
    const u = auth.currentUser
    if (u?.role === 'admin') router.push('/admin/dashboard')
    else if (u?.role === 'operator') router.push('/operator/dashboard')
    else router.push('/user/dashboard')
  } catch { 
    ElMessage.error('登录失败，请检查用户名、密码和验证码')
    form.value.captcha_code = ''
    loadCaptcha() 
  }
  loading.value = false
}
</script>

<style scoped>
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #0f172a 0%, #1e293b 50%, #0f172a 100%);
  position: relative;
  overflow: hidden;
}
.login-bg-shapes {
  position: absolute;
  inset: 0;
  pointer-events: none;
}
.shape {
  position: absolute;
  border-radius: 50%;
  filter: blur(80px);
  opacity: 0.15;
  animation: float 20s ease-in-out infinite;
}
.shape-1 {
  width: 500px; height: 500px;
  background: #4C6EF5;
  top: -10%; left: -5%;
  animation-delay: 0s;
}
.shape-2 {
  width: 400px; height: 400px;
  background: #10B981;
  bottom: -15%; right: -5%;
  animation-delay: -5s;
}
.shape-3 {
  width: 300px; height: 300px;
  background: #8B5CF6;
  top: 50%; left: 60%;
  animation-delay: -10s;
}
.shape-4 {
  width: 250px; height: 250px;
  background: #14B8A6;
  top: 20%; right: 20%;
  animation-delay: -15s;
}
@keyframes float {
  0%, 100% { transform: translate(0, 0) scale(1); }
  25% { transform: translate(30px, -40px) scale(1.05); }
  50% { transform: translate(-20px, 20px) scale(0.95); }
  75% { transform: translate(15px, 35px) scale(1.02); }
}
html.dark .login-container {
  background: linear-gradient(135deg, #0a0a1a 0%, #111827 50%, #0a0a1a 100%);
}

.login-card {
  width: 420px;
  padding: 48px 40px 36px;
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.9);
  backdrop-filter: blur(24px);
  -webkit-backdrop-filter: blur(24px);
  border: 1px solid rgba(255, 255, 255, 0.3);
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.15), 0 0 0 1px rgba(255, 255, 255, 0.1) inset;
  position: relative;
  z-index: 1;
  animation: cardIn 0.6s ease-out;
}
@keyframes cardIn {
  from { opacity: 0; transform: translateY(24px) scale(0.97); }
  to { opacity: 1; transform: translateY(0) scale(1); }
}
html.dark .login-card {
  background: rgba(30, 41, 59, 0.85);
  border: 1px solid rgba(255, 255, 255, 0.06);
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.4), 0 0 0 1px rgba(255, 255, 255, 0.04) inset;
}

.login-header {
  text-align: center;
  margin-bottom: 32px;
}
.login-logo {
  width: 56px; height: 56px;
  border-radius: 14px;
  background: linear-gradient(135deg, #4C6EF5, #8B5CF6);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 28px;
  font-weight: 800;
  color: #fff;
  margin-bottom: 16px;
  box-shadow: 0 4px 16px rgba(76, 110, 245, 0.3);
}
.login-title {
  font-size: 26px;
  font-weight: 800;
  letter-spacing: 4px;
  margin: 0 0 6px 0;
  background: linear-gradient(135deg, #4C6EF5, #8B5CF6);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}
.login-subtitle {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  margin: 0;
  letter-spacing: 1px;
}

.captcha-row {
  display: flex;
  gap: 10px;
  align-items: center;
}
.captcha-row .el-input { flex: 1; }
.captcha-img {
  height: 40px;
  cursor: pointer;
  border-radius: 8px;
  border: 1px solid var(--el-border-color);
  transition: transform 0.2s;
}
.captcha-img:hover { transform: scale(1.05); }
</style>
