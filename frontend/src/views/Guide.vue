<template>
  <div>
    <h2 class="page-title">{{ t('guide.title') }}</h2>
    <el-card style="margin-bottom: 16px">
      <h3>{{ t('guide.baseUrl') }}</h3>
      <el-input :model-value="baseUrl" readonly><template #append><el-button @click="copy(baseUrl)">{{ t('common.copy') }}</el-button></template></el-input>
      <h3 style="margin-top: 16px">{{ t('guide.apiKey') }}</h3>
      <p style="color: var(--el-text-color-secondary)">在"我的令牌"页面创建令牌获取API密钥</p>
      <h3 style="margin-top: 16px">{{ t('guide.example') }}</h3>
      <pre style="background: var(--el-fill-color-light); padding: 12px; border-radius: 4px; overflow-x: auto">curl {{ baseUrl }}/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4","messages":[{"role":"user","content":"Hello"}]}'</pre>
    </el-card>
    <el-card>
      <h3>{{ t('guide.supportedModels') }}</h3>
      <el-tag v-for="m in models" :key="m" style="margin: 4px">{{ m }}</el-tag>
      <div v-if="!models.length" style="color: var(--el-text-color-secondary)">Loading...</div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { statsApi } from '@/api/stats'

const { t } = useI18n()
const baseUrl = window.location.origin
const models = ref<string[]>([])

onMounted(async () => {
  try { const d = await statsApi.myModels(); models.value = d.models || [] } catch {}
})

function copy(text: string) {
  navigator.clipboard.writeText(text)
  ElMessage.success(t('common.copied'))
}
</script>
