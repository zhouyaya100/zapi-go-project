<template>
  <div>
    <div class="page-header">
      <h2 class="page-title">{{ t('nav.myTokens') }}</h2>
      <el-button type="primary" @click="openForm()">{{ t('token.addToken') }}</el-button>
    </div>
    <el-table :data="items" style="width:100%" v-loading="loading">
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="name" :label="t('token.name')" min-width="120" />
      <el-table-column :label="t('token.key')" min-width="220">
        <template #default="{row}"><code>{{row.key.substring(0,16)}}...</code><el-button link type="primary" size="small" @click="copy(row.key)">{{ t('common.copy') }}</el-button></template>
      </el-table-column>
      <el-table-column prop="models" :label="t('token.models')" min-width="120" show-overflow-tooltip />
      <el-table-column :label="t('common.enabled')" width="80"><template #default="{row}"><el-switch v-model="row.enabled" @change="toggleEnabled(row)" size="small" /></template></el-table-column>
      <el-table-column :label="t('token.quotaLimit')" width="120"><template #default="{row}">{{row.quota_limit===-1?t('common.unlimited'):row.quota_limit?.toLocaleString()}}</template></el-table-column>
      <el-table-column :label="t('token.quotaUsed')" prop="quota_used" width="120"><template #default="{row}">{{row.quota_used?.toLocaleString()}}</template></el-table-column>
      <el-table-column :label="t('token.expiresAt')" width="170"><template #default="{row}">{{row.expires_at||'永不过期'}}</template></el-table-column>
      <el-table-column :label="t('common.actions')" width="120" fixed="right">
        <template #default="{row}">
          <el-button link type="primary" @click="openForm(row)">{{ t('common.edit') }}</el-button>
          <el-button link type="danger" @click="del(row)">{{ t('common.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-dialog v-model="formVisible" :title="editing ? t('token.editToken') : t('token.addToken')" width="500" :close-on-click-modal="false">
      <el-form :model="form" label-width="100px">
        <el-form-item :label="t('token.name')"><el-input v-model="form.name" /></el-form-item>
        <el-form-item :label="t('token.models')">
          <ModelMultiSelect v-model="form.models" :allModels="myModels" placeholder="留空=全部授权模型" />
        </el-form-item>
        <el-form-item :label="t('token.quotaLimit')"><el-input-number v-model="form.quota_limit" :min="-1" style="width:100%" /><span style="margin-left:8px;color:var(--el-text-color-secondary)">-1={{t('common.unlimited')}}</span></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="formVisible=false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="save" :loading="saving">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { myTokenApi } from '@/api/token'
import { statsApi } from '@/api/stats'
import ModelMultiSelect from '@/components/common/ModelMultiSelect.vue'
import type { Token } from '@/api/types'

const { t } = useI18n()
const items = ref<Token[]>([])
const loading = ref(false)
const formVisible = ref(false)
const saving = ref(false)
const editing = ref<number|null>(null)
const form = ref<any>({})
const myModels = ref<string[]>([])

onMounted(async () => {
  load()
  // 获取用户授权的模型列表
  try {
    const d = await statsApi.myModels()
    myModels.value = d.models || []
  } catch {}
})

async function load() { loading.value = true; try { const d = await myTokenApi.list(); items.value = Array.isArray(d) ? d : (d as any).items || [] } catch {}; loading.value = false }
function openForm(row?: Token) { editing.value = row?.id || null; form.value = row ? { name: row.name, models: row.models, quota_limit: row.quota_limit } : { name: '', models: '', quota_limit: -1 }; formVisible.value = true }
async function save() { saving.value = true; try { if (editing.value) await myTokenApi.update(editing.value, { name: form.value.name, models: form.value.models, quota_limit: form.value.quota_limit }); else await myTokenApi.create(form.value); ElMessage.success(t('common.success')); formVisible.value = false; load() } catch {}; saving.value = false }
async function toggleEnabled(row: Token) { try { await myTokenApi.update(row.id, { enabled: row.enabled }) } catch { row.enabled = !row.enabled } }
function copy(key: string) { navigator.clipboard.writeText(key); ElMessage.success(t('common.copied')) }
async function del(row: Token) { try { await ElMessageBox.confirm(t('common.confirmDelete'), '', {type:'warning'}); await myTokenApi.delete(row.id); load() } catch {} }
</script>
