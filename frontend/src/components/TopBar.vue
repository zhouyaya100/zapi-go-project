<template>
  <div class="topbar">
    <div class="topbar-left">
      <el-button :icon="collapsed ? 'Expand' : 'Fold'" text @click="$emit('toggle')" class="topbar-btn" />
      <el-breadcrumb separator="/">
        <el-breadcrumb-item><span class="topbar-route">{{ currentRoute }}</span></el-breadcrumb-item>
      </el-breadcrumb>
    </div>
    <div class="topbar-right">
      <el-badge :value="unreadCount" :hidden="unreadCount === 0" :max="99">
        <el-button :icon="'Bell'" text @click="$router.push('/notifications')" class="topbar-btn" />
      </el-badge>
      <el-button :icon="appStore.isDark ? 'Sunny' : 'Moon'" text @click="appStore.toggleDark()" :title="t(appStore.isDark ? 'nav.lightMode' : 'nav.darkMode')" class="topbar-btn" />
      <el-dropdown @command="handleLang">
        <el-button text class="topbar-btn">{{ appStore.locale === 'zh' ? '中文' : 'EN' }}</el-button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="zh">中文</el-dropdown-item>
            <el-dropdown-item command="en">English</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
      <el-dropdown @command="handleCommand">
        <el-button text class="topbar-btn">
          <el-icon><User /></el-icon>
          {{ auth.currentUser?.username }}
        </el-button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="password">{{ t('nav.changePassword') }}</el-dropdown-item>
            <el-dropdown-item command="logout" divided>{{ t('nav.logout') }}</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
    <el-dialog v-model="pwdDialog" :title="t('nav.changePassword')" width="400">
      <el-form :model="pwdForm" label-width="100px">
        <el-form-item :label="t('login.password')"><el-input v-model="pwdForm.old_password" type="password" show-password /></el-form-item>
        <el-form-item :label="t('login.password')"><el-input v-model="pwdForm.new_password" type="password" show-password /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="pwdDialog = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="changePwd" :loading="pwdLoading">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { notificationApi } from '@/api/notification'

defineProps<{ collapsed: boolean }>()
defineEmits(['toggle'])

const { t } = useI18n()
const route = useRoute()
const auth = useAuthStore()
const appStore = useAppStore()
const unreadCount = ref(0)
const pwdDialog = ref(false)
const pwdLoading = ref(false)
const pwdForm = ref({ old_password: '', new_password: '' })
const currentRoute = computed(() => route.meta.title || route.name?.toString() || '')

onMounted(async () => {
  try { const d = await notificationApi.unreadCount(); unreadCount.value = d.count } catch {}
})

function handleLang(lang: string) { appStore.setLocale(lang); location.reload() }
function handleCommand(cmd: string) {
  if (cmd === 'logout') auth.logout()
  else if (cmd === 'password') { pwdForm.value = { old_password: '', new_password: '' }; pwdDialog.value = true }
}
async function changePwd() {
  pwdLoading.value = true
  try { await auth.changePassword(pwdForm.value.old_password, pwdForm.value.new_password); ElMessage.success(t('common.success')); pwdDialog.value = false } catch {}
  pwdLoading.value = false
}
</script>

<style scoped>
.topbar {
  display: flex;
  align-items: center;
  height: 56px;
  padding: 0 20px;
  justify-content: space-between;
  background: var(--el-bg-color);
  box-shadow: 0 1px 4px rgba(0,0,0,0.04);
}
html.dark .topbar {
  box-shadow: 0 1px 4px rgba(0,0,0,0.15);
}
.topbar-left {
  display: flex;
  align-items: center;
  gap: 12px;
}
.topbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
}
.topbar-route {
  font-weight: 600;
  font-size: 14px;
}
.topbar-btn {
  border-radius: 8px;
  transition: all 0.2s ease;
}
.topbar-btn:hover {
  background: rgba(76,110,245,0.08);
}
html.dark .topbar-btn:hover {
  background: rgba(108,143,248,0.1);
}
</style>
