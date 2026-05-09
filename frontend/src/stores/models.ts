import { defineStore } from 'pinia'
import { ref } from 'vue'
import { settingsApi } from '@/api/settings'

export const useModelsStore = defineStore('models', () => {
  const allModels = ref<string[]>([])
  const loaded = ref(false)

  async function fetchModels() {
    if (loaded.value) return
    try {
      // Try the admin settings API first (returns all_models)
      const data = await settingsApi.get()
      allModels.value = data.all_models || []
      loaded.value = true
    } catch {
      // Fallback to public settings (accessible to all users)
      try {
        const data = await settingsApi.public()
        allModels.value = data.all_models || []
        loaded.value = true
      } catch {}
    }
  }

  function reset() {
    allModels.value = []
    loaded.value = false
  }

  return { allModels, loaded, fetchModels, reset }
})
