<template>
  <div class="login-container">
    <el-card class="login-card">
      <h2 style="text-align: center; margin-bottom: 24px; color: #18a058">ZAPI - {{ t('login.register') }}</h2>
      <el-form :model="form" @submit.prevent="handleRegister" label-position="top">
        <el-form-item :label="t('login.username')"><el-input v-model="form.username" /></el-form-item>
        <el-form-item :label="t('login.password')"><el-input v-model="form.password" type="password" show-password /></el-form-item>
        <el-form-item :label="t('login.confirmPassword')"><el-input v-model="form.confirm" type="password" show-password /></el-form-item>
        <el-form-item :label="t('login.captcha')">
          <div class="captcha-row">
            <el-input v-model="form.captcha_code" @keyup.enter="handleRegister" />
            <img v-if="captchaImg" :src="captchaImg" class="captcha-img" @click="loadCaptcha" />
          </div>
        </el-form-item>
        <el-button type="primary" style="width: 100%" @click="handleRegister" :loading="loading">{{ t('login.register') }}</el-button>
        <div style="text-align: center; margin-top: 12px">
          <router-link to="/login">{{ t('login.hasAccount') }} {{ t('login.submit') }}</router-link>
        </div>
      </el-form>
    </el-card>
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
const form = ref({ username: '', password: '', confirm: '', captcha_id: '', captcha_code: '' })
const loading = ref(false)
const captchaImg = ref('')

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
  await loadCaptcha()
  // Check if registration is allowed
  try {
    const pub = await settingsApi.public()
    if (!pub.allow_register) {
      ElMessage.warning('系统已关闭注册')
      router.push('/login')
    }
  } catch { /* ignore */ }
})

async function handleRegister() {
  if (!form.value.username || !form.value.password) { ElMessage.warning('请填写所有字段'); return }
  if (form.value.password !== form.value.confirm) { ElMessage.warning('密码不一致'); return }
  loading.value = true
  try {
    await auth.register(form.value.username, form.value.password, form.value.captcha_id, form.value.captcha_code)
    ElMessage.success(t('common.success'))
    router.push('/user/dashboard')
  } catch {
    form.value.captcha_code = ''
    loadCaptcha()
  }
  loading.value = false
}
</script>
