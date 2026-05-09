import { defineStore } from 'pinia'
import { ref } from 'vue'
import { versionApi } from '@/api/version'

export const useAppStore = defineStore('app', () => {
  const isDark = ref(localStorage.getItem('zapi_dark') === 'true')
  const version = ref('')
  const locale = ref(localStorage.getItem('zapi_lang') || 'zh')

  function toggleDark() {
    isDark.value = !isDark.value
    localStorage.setItem('zapi_dark', String(isDark.value))
  }

  async function fetchVersion() {
    try {
      const data = await versionApi.get()
      version.value = data.version
    } catch {}
  }

  function setLocale(lang: string) {
    locale.value = lang
    localStorage.setItem('zapi_lang', lang)
  }

  return { isDark, version, locale, toggleDark, fetchVersion, setLocale }
})
