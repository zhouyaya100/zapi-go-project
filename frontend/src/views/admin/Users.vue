<template>
  <div>
    <div class="page-header">
      <h2 class="page-title">{{ t('user.title') }}</h2>
    </div>
    <div style="margin-bottom:12px;display:flex;align-items:center;gap:12px;flex-wrap:wrap">
      <el-input v-model="searchText" placeholder="搜索用户名" clearable style="width:220px" prefix-icon="Search" />
      <el-select v-model="roleFilter" clearable placeholder="角色过滤" style="width:140px">
        <el-option label="管理员" value="admin" /><el-option label="运营" value="operator" /><el-option label="普通用户" value="user" />
      </el-select>
      <el-select v-model="groupFilter" clearable placeholder="分组过滤" style="width:160px">
        <el-option v-for="g in groups" :key="g.id" :label="g.name" :value="g.id" />
      </el-select>
    </div>
    <el-table :data="filteredItems" style="width:100%" v-loading="loading">
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="username" :label="t('user.username')" min-width="100" />
      <el-table-column prop="role" :label="t('user.role')" width="80">
        <template #default="{row}"><el-tag :type="row.role==='admin'?'danger':row.role==='operator'?'warning':'info'" size="small">{{roleLabel(row.role)}}</el-tag></template>
      </el-table-column>
      <el-table-column prop="group_name" :label="t('user.group')" width="100" />
      <el-table-column label="绑定模式" width="90">
        <template #default="{row}">
          <el-tag :type="row.bind_mode==='custom'?'warning':''" size="small">{{ row.bind_mode==='custom'?'自定义':'继承分组' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="可用模型" min-width="120" show-overflow-tooltip>
        <template #default="{row}">{{ row.authed_models?.length ? row.authed_models.join(', ') : '-' }}</template>
      </el-table-column>
      <el-table-column :label="t('common.enabled')" width="70"><template #default="{row}"><el-switch v-model="row.enabled" @change="toggleEnabled(row)" size="small" /></template></el-table-column>
      <el-table-column :label="t('user.tokenQuota')" width="110"><template #default="{row}">{{row.token_quota===-1?t('common.unlimited'):row.token_quota?.toLocaleString()}}</template></el-table-column>
      <el-table-column :label="t('user.tokenQuotaUsed')" prop="token_quota_used" width="110"><template #default="{row}">{{row.token_quota_used?.toLocaleString()}}</template></el-table-column>
      <el-table-column :label="t('common.actions')" width="260" fixed="right">
        <template #default="{row}">
          <el-button link type="primary" @click="openForm(row)">{{ t('common.edit') }}</el-button>
          <el-button link type="success" @click="openRecharge(row)">{{ t('user.recharge') }}</el-button>
          <el-button link type="warning" @click="openDeduct(row)">{{ t('user.deduct') }}</el-button>
          <el-button link type="info" @click="resetPwd(row)">{{ t('user.resetPassword') }}</el-button>
          <el-button v-if="row.id!==1" link type="danger" @click="del(row)">{{ t('common.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="formVisible" :title="t('user.editUser')" width="650" :close-on-click-modal="false">
      <el-form :model="form" label-width="120px">
        <el-form-item :label="t('user.role')">
          <el-select v-model="form.role" style="width:100%"><el-option label="管理员" value="admin" /><el-option label="运营" value="operator" /><el-option label="普通用户" value="user" /></el-select>
        </el-form-item>
        <el-form-item :label="t('user.group')">
          <el-select v-model="form.group_id" clearable placeholder="无分组" style="width:100%"><el-option v-for="g in groups" :key="g.id" :label="g.name" :value="g.id" /></el-select>
        </el-form-item>
        <el-form-item label="绑定模式">
          <el-radio-group v-model="form.bind_mode" @change="onBindModeChange">
            <el-radio value="inherit">继承分组</el-radio>
            <el-radio value="custom">自定义</el-radio>
          </el-radio-group>
        </el-form-item>
        <!-- 继承分组时：显示分组配置的只读信息 -->
        <el-form-item v-if="form.bind_mode==='inherit'" label="分组模型">
          <span style="color:var(--el-text-color-regular)">{{ form._group_allowed_models || '无权限' }}</span>
          <span style="margin-left:8px;color:var(--el-text-color-secondary);font-size:12px">（来自分组）</span>
        </el-form-item>
        <!-- 自定义模式时：提示 -->
        <el-form-item v-if="form.bind_mode==='custom'">
          <el-alert type="warning" :closable="false" show-icon style="width:100%">自定义模式下，留空表示无权限</el-alert>
        </el-form-item>
        <el-form-item :label="t('user.allowedModels')">
          <ModelMultiSelect v-model="form.allowed_models" :allModels="modelsStore.allModels" :disabled="form.bind_mode==='inherit'" :placeholder="form.bind_mode==='inherit'?'继承分组，不可编辑':'留空表示无模型权限'" />
        </el-form-item>
        <el-form-item :label="t('user.maxTokens')"><el-input-number v-model="form.max_tokens" :min="0" style="width:100%" /></el-form-item>
        <el-form-item :label="t('user.tokenQuota')"><el-input-number v-model="form.token_quota" :min="-1" style="width:100%" /><span style="margin-left:8px;color:var(--el-text-color-secondary)">-1={{t('common.unlimited')}}</span></el-form-item>
        <el-form-item :label="t('user.rateMode')">
          <el-select v-model="form.rate_mode" style="width:100%">
            <el-option label="继承分组" value="inherit" /><el-option label="全局" value="global" /><el-option label="按模型" value="per_model" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="form.rate_mode!=='inherit'" label="RPM"><el-input-number v-model="form.rpm" :min="-1" style="width:100%" /><span style="margin-left:8px;color:var(--el-text-color-secondary);font-size:12px">-1=不限, 0=禁止</span></el-form-item>
        <el-form-item v-if="form.rate_mode!=='inherit'" label="TPM"><el-input-number v-model="form.tpm" :min="-1" style="width:100%" /><span style="margin-left:8px;color:var(--el-text-color-secondary);font-size:12px">-1=不限, 0=禁止</span></el-form-item>
        <el-form-item v-if="form.rate_mode==='per_model'" :label="t('user.modelRateLimits')">
          <div style="width:100%">
            <el-table :data="modelRateEntries" size="small" border style="margin-bottom:8px">
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
                <template #default="{$index}"><el-button link type="danger" size="small" @click="modelRateEntries.splice($index,1)">✕</el-button></template>
              </el-table-column>
            </el-table>
            <el-button size="small" @click="modelRateEntries.push({model:'',rpm:0,tpm:0,blocked:false})">+ 添加模型限流</el-button>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="formVisible=false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="save" :loading="saving">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="rechargeVisible" :title="t('user.recharge')" width="400">
      <el-form :model="rechargeForm" label-width="80px">
        <el-form-item label="金额"><el-input-number v-model="rechargeForm.amount" :min="1" style="width:100%" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="rechargeVisible=false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="doRecharge" :loading="rechargeLoading">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="deductVisible" title="扣除额度" width="400">
      <el-form :model="deductForm" label-width="80px">
        <el-form-item label="金额"><el-input-number v-model="deductForm.amount" :min="1" style="width:100%" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="deductVisible=false">{{ t('common.cancel') }}</el-button>
        <el-button type="warning" @click="doDeduct" :loading="deductLoading">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { userApi } from '@/api/user'
import { groupApi } from '@/api/group'
import { useModelsStore } from '@/stores/models'
import ModelMultiSelect from '@/components/common/ModelMultiSelect.vue'
import type { User, Group } from '@/api/types'

const { t } = useI18n()
const modelsStore = useModelsStore()
const items = ref<User[]>([])
const groups = ref<Group[]>([])
const loading = ref(false)
const formVisible = ref(false)
const saving = ref(false)
const editing = ref<number|null>(null)
const form = ref<any>({})
const modelRateEntries = ref<{model:string,rpm:number,tpm:number,blocked:boolean}[]>([])
const rechargeVisible = ref(false)
const rechargeLoading = ref(false)
const rechargeForm = ref({ id: 0, amount: 100 })
const deductVisible = ref(false)
const deductLoading = ref(false)
const deductForm = ref({ id: 0, amount: 100 })

// Search & filters
const searchText = ref('')
const roleFilter = ref('')
const groupFilter = ref<number | string>('')

const filteredItems = computed(() => {
  return items.value.filter((u: User) => {
    if (searchText.value) {
      const kw = searchText.value.toLowerCase()
      if (!(u.username && u.username.toLowerCase().includes(kw))) return false
    }
    if (roleFilter.value && u.role !== roleFilter.value) return false
    if (groupFilter.value !== '' && groupFilter.value !== null && u.group_id !== groupFilter.value) return false
    return true
  })
})

function roleLabel(role: string) { return role === 'admin' ? t('user.admin') : role === 'operator' ? t('user.operator') : t('user.normalUser') }

onMounted(async () => {
  load()
  try { groups.value = await groupApi.list() } catch {}
  modelsStore.fetchModels()
})

async function load() { loading.value = true; try { const d = await userApi.list(); items.value = Array.isArray(d) ? d : (d as any).items || [] } catch {}; loading.value = false }

function parseModelRateLimits(s: string): {model:string,rpm:number,tpm:number,blocked:boolean}[] {
  if (!s) return []
  try {
    const obj = JSON.parse(s)
    return Object.entries(obj).map(([k, v]: [string, any]) => ({ model: k, rpm: v.rpm || 0, tpm: v.tpm || 0, blocked: !!v.blocked }))
  } catch { return [] }
}

function onBindModeChange(val: string) {
  if (val === 'inherit') {
    form.value.allowed_models = ''
  }
}

function openForm(row: User) {
  editing.value = row.id
  form.value = {
    role: row.role, group_id: row.group_id || 0, max_tokens: row.max_tokens, token_quota: row.token_quota,
    bind_mode: row.bind_mode || 'inherit',
    allowed_models: row.allowed_models || '', rate_mode: row.rate_mode || 'inherit', rpm: row.rpm || 0, tpm: row.tpm || 0,
    // 只读展示字段（以下划线开头标记，不提交到后端）
    _group_allowed_models: row.group_allowed_models || ''
  }
  modelRateEntries.value = parseModelRateLimits(row.model_rate_limits || '')
  formVisible.value = true
}

async function save() {
  saving.value = true
  try {
    const data: any = {
      role: form.value.role,
      group_id: form.value.group_id,
      max_tokens: form.value.max_tokens,
      token_quota: form.value.token_quota,
      bind_mode: form.value.bind_mode,
      allowed_models: form.value.allowed_models,
      rate_mode: form.value.rate_mode,
      rpm: form.value.rpm,
      tpm: form.value.tpm
    }
    // Ensure group_id is 0 when cleared (null from el-select clearable)
    if (data.group_id === null || data.group_id === undefined) data.group_id = 0
    if (form.value.rate_mode === 'per_model' && modelRateEntries.value.length) {
      const obj: Record<string, any> = {}
      for (const e of modelRateEntries.value) {
        if (e.model.trim()) obj[e.model.trim()] = { rpm: e.rpm, tpm: e.tpm, blocked: e.blocked }
      }
      data.model_rate_limits = Object.keys(obj).length ? JSON.stringify(obj) : ''
    } else {
      data.model_rate_limits = ''
    }
    await userApi.update(editing.value!, data); ElMessage.success(t('common.success')); formVisible.value = false; load()
  } catch {}
  saving.value = false
}

async function toggleEnabled(row: User) { try { await userApi.update(row.id, { enabled: row.enabled }) } catch { row.enabled = !row.enabled } }

function openRecharge(row: User) { rechargeForm.value = { id: row.id, amount: 100 }; rechargeVisible.value = true }
async function doRecharge() {
  rechargeLoading.value = true
  try { await userApi.recharge(rechargeForm.value.id, rechargeForm.value.amount); ElMessage.success(t('common.success')); rechargeVisible.value = false; load() } catch {}
  rechargeLoading.value = false
}

function openDeduct(row: User) { deductForm.value = { id: row.id, amount: 100 }; deductVisible.value = true }
async function doDeduct() {
  deductLoading.value = true
  try { await userApi.deduct(deductForm.value.id, deductForm.value.amount); ElMessage.success(t('common.success')); deductVisible.value = false; load() } catch {}
  deductLoading.value = false
}

async function resetPwd(row: User) {
  try {
    const { value } = await ElMessageBox.prompt('输入新密码', t('user.resetPassword'), { inputType: 'password' })
    if (value) { await userApi.update(row.id, { password: value }); ElMessage.success(t('common.success')) }
  } catch {}
}

async function del(row: User) {
  if (row.id === 1) { ElMessage.error('无法删除超级管理员'); return }
  try { await ElMessageBox.confirm(t('common.confirmDelete'), '', { type: 'warning' }); await userApi.delete(row.id); load() } catch {}
}
</script>
