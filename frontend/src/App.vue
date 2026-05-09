<template>
  <el-config-provider :locale="elementLocale">
    <router-view />
  </el-config-provider>
</template>

<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import { ElConfigProvider } from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import en from 'element-plus/es/locale/lang/en'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

const { locale } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const elementLocale = computed(() => locale.value === 'zh' ? zhCn : en)

watch(() => appStore.isDark, (v) => {
  document.documentElement.classList.toggle('dark', v)
}, { immediate: true })

onMounted(() => {
  authStore.restoreSession()
  appStore.fetchVersion()
})
</script>
