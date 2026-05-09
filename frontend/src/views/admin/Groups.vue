<template>
  <div>
    <div class="page-header">
      <h2 class="page-title">{{ t('group.title') }}</h2>
      <el-button type="primary" @click="openForm()">{{ t('group.addGroup') }}</el-button>
    </div>
    <el-table :data="items" style="width:100%" v-loading="loading">
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="name" :label="t('group.name')" />
      <el-table-column prop="comment" :label="t('group.comment')" />
      <el-table-column :label="t('group.rateMode')" width="90">
        <template #default="{row}">{{ row.rate_mode === 'per_model' ? '按模型' : '全局' }}</template>
      </el-table-column>
      <el-table-column prop="rpm" label="RPM" width="80">
        <template #default="{row}">{{ row.rpm === -1 ? '∞' : row.rpm === 0 ? '禁止' : row.rpm }}</template>
      </el-table-column>
      <el-table-column prop="tpm" label="TPM" width="80">
        <template #default="{row}">{{ row.tpm === -1 ? '∞' : row.tpm === 0 ? '禁止' : row.tpm }}</template>
      </el-table-column>
      <el-table-column :label="t('group.modelRateLimits')" min-width="150" show-overflow-tooltip>
        <template #default="{row}">{{ formatRateLimits(row.model_rate_limits) }}</template>
      </el-table-column>
      <el-table-column label="可选模型" min-width="120" show-overflow-tooltip>
        <template #default="{row}">{{ row.allowed_models || '无权限' }}</template>
      </el-table-column>
      <el-table-column :label="t('group.userCount')" prop="user_count" width="80" />
      <el-table-column :label="t('common.actions')" width="150">
        <template #default="{row}">
          <el-button link type="primary" @click="openForm(row)">{{ t('common.edit') }}</el-button>
          <el-button link type="danger" @click="del(row)">{{ t('common.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="formVisible" :title="editing ? t('group.editGroup') : t('group.addGroup')" width="650" :close-on-click-modal="false">
      <el-form :model="form" label-width="100px">
        <el-form-item :label="t('group.name')"><el-input v-model="form.name" /></el-form-item>
        <el-form-item :label="t('group.comment')"><el-input v-model="form.comment" /></el-form-item>
        <el-form-item label="可选模型">
          <ModelMultiSelect v-model="form.allowed_models" :allModels="modelsStore.allModels" placeholder="留空表示无权限，请选择可用模型" />
        </el-form-item>
        <el-form-item :label="t('group.rateMode')">
          <el-select v-model="form.rate_mode" style="width:100%">
            <el-option label="全局" value="global" /><el-option label="按模型" value="per_model" />
          </el-select>
        </el-form-item>
        <el-form-item label="RPM"><el-input-number v-model="form.rpm" :min="-1" style="width:100%" /><span style="margin-left:8px;color:var(--el-text-color-secondary);font-size:12px">-1=不限, 0=禁止</span></el-form-item>
        <el-form-item label="TPM"><el-input-number v-model="form.tpm" :min="-1" style="width:100%" /><span style="margin-left:8px;color:var(--el-text-color-secondary);font-size:12px">-1=不限, 0=禁止</span></el-form-item>
        <el-form-item v-if="form.rate_mode === 'per_model'" :label="t('group.modelRateLimits')">
          <div style="width:100%">
            <el-table :data="rateLimitEntries" size="small" border style="margin-bottom:8px">
              <el-table-column label="模型" min-width="150">
                <template #default="{row}"><el-select v-model="row.model" filterable allow-create default-first-option placeholder="选择模型" size="small" style="width:100%"><el-option v-for="m in modelsStore.allModels" :key="m" :label="m" :value="m" /></el-select></template>
              </el-table-column>
              <el-table-column label="RPM" width="100">
                <template #default="{row}"><el-input-number v-model="row.rpm" :min="-1" size="small" controls-position="right" /></template>
              </el-table-column>
              <el-table-column label="TPM" width="100">
                <template #default="{row}"><el-input-number v-model="row.tpm" :min="-1" size="small" controls-position="right" /></template>
              </el-table-column>
              <el-table-column label="禁止" width="60">
                <template #default="{row}"><el-switch v-model="row.blocked" size="small" /></template>
              </el-table-column>
              <el-table-column width="50">
                <template #default="{$index}"><el-button link type="danger" size="small" @click="rateLimitEntries.splice($index,1)">✕</el-button></template>
              </el-table-column>
            </el-table>
            <el-button size="small" @click="rateLimitEntries.push({model:'',rpm:0,tpm:0,blocked:false})">+ 添加模型限流</el-button>
          </div>
        </el-form-item>
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
import { groupApi } from '@/api/group'
import { useModelsStore } from '@/stores/models'
import ModelMultiSelect from '@/components/common/ModelMultiSelect.vue'
import type { Group } from '@/api/types'

const { t } = useI18n()
const modelsStore = useModelsStore()
const items = ref<Group[]>([])
const loading = ref(false)
const formVisible = ref(false)
const saving = ref(false)
const editing = ref<number|null>(null)
const form = ref<any>({})
const rateLimitEntries = ref<{model:string,rpm:number,tpm:number,blocked:boolean}[]>([])

onMounted(() => { load(); modelsStore.fetchModels() })
async function load() { loading.value = true; try { items.value = await groupApi.list() } catch {}; loading.value = false }

function formatRateLimits(s: string): string {
  if (!s) return '-'
  try {
    const obj = JSON.parse(s)
    return Object.entries(obj).map(([k,v]: any[]) => `${k}:${v.blocked?'禁止':`${v.rpm===-1?'∞':v.rpm===0?'禁止':v.rpm}rpm/${v.tpm===-1?'∞':v.tpm===0?'禁止':v.tpm}tpm`}`).join('; ')
  } catch { return s || '-' }
}

function parseRateLimits(s: string): {model:string,rpm:number,tpm:number,blocked:boolean}[] {
  if (!s) return []
  try {
    const obj = JSON.parse(s)
    return Object.entries(obj).map(([k, v]: [string, any]) => ({ model: k, rpm: v.rpm || 0, tpm: v.tpm || 0, blocked: !!v.blocked }))
  } catch { return [] }
}

function openForm(row?: Group) {
  editing.value = row?.id || null
  if (row) {
    form.value = { ...row }
    rateLimitEntries.value = parseRateLimits(row.model_rate_limits)
  } else {
    form.value = { name: '', comment: '', rate_mode: 'global', rpm: -1, tpm: -1, model_rate_limits: '', allowed_models: '' }
    rateLimitEntries.value = []
  }
  formVisible.value = true
}

async function save() {
  saving.value = true
  try {
    const data = { ...form.value }
    // rate limit entries -> JSON
    if (form.value.rate_mode === 'per_model' && rateLimitEntries.value.length) {
      const obj: Record<string, any> = {}
      for (const e of rateLimitEntries.value) {
        if (e.model.trim()) obj[e.model.trim()] = { rpm: e.rpm, tpm: e.tpm, blocked: e.blocked }
      }
      data.model_rate_limits = Object.keys(obj).length ? JSON.stringify(obj) : ''
    } else {
      data.model_rate_limits = ''
    }
    if (editing.value) await groupApi.update(editing.value, data)
    else await groupApi.create(data)
    ElMessage.success(t('common.success')); formVisible.value = false; load()
  } catch {}
  saving.value = false
}

async function del(row: Group) {
  try { await ElMessageBox.confirm(t('common.confirmDelete'), '', {type:'warning'}); await groupApi.delete(row.id); load() } catch {}
}
</script>
