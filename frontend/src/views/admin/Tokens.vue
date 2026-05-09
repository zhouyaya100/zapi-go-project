<template>
  <div>
    <div class="page-header">
      <h2 class="page-title">{{ t('token.title') }}</h2>
      <el-button type="primary" @click="openForm()">{{ t('token.addToken') }}</el-button>
    </div>
    <div style="margin-bottom:12px;display:flex;align-items:center;gap:12px">
      <el-input v-model="searchText" placeholder="搜索名称/Key" clearable style="width:260px" prefix-icon="Search" />
    </div>
    <el-table :data="pagedItems" style="width:100%" v-loading="loading">
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="username" :label="t('token.user')" width="100" />
      <el-table-column prop="name" :label="t('token.name')" min-width="120" />
      <el-table-column :label="t('token.key')" min-width="260">
        <template #default="{row}">
          <code style="font-size:13px">{{ visibleKeys[row.id] ? row.key : row.key.substring(0,12) + '...' }}</code>
          <el-button link :type="visibleKeys[row.id]?'warning':'primary'" size="small" @click="visibleKeys[row.id]=!visibleKeys[row.id]">
            <el-icon><View v-if="!visibleKeys[row.id]" /><Hide v-else /></el-icon>
          </el-button>
          <el-button link type="primary" size="small" @click="copy(row.key)">{{ t('common.copy') }}</el-button>
        </template>
      </el-table-column>
      <el-table-column prop="models" :label="t('token.models')" min-width="120" show-overflow-tooltip />
      <el-table-column :label="t('common.enabled')" width="70"><template #default="{row}"><el-switch v-model="row.enabled" @change="toggleEnabled(row)" size="small" /></template></el-table-column>
      <el-table-column :label="t('token.quotaLimit')" width="110"><template #default="{row}">{{row.quota_limit===-1?t('common.unlimited'):row.quota_limit?.toLocaleString()}}</template></el-table-column>
      <el-table-column :label="t('token.quotaUsed')" prop="quota_used" width="110"><template #default="{row}">{{row.quota_used?.toLocaleString()}}</template></el-table-column>
      <el-table-column :label="t('token.expiresAt')" width="170"><template #default="{row}">{{row.expires_at||'永不过期'}}</template></el-table-column>
      <el-table-column :label="t('common.actions')" width="220" fixed="right">
        <template #default="{row}">
          <el-button link type="primary" @click="openForm(row)">{{ t('common.edit') }}</el-button>
          <el-button link type="success" @click="openRecharge(row)">{{ t('token.recharge') }}</el-button>
          <el-button v-if="row.quota_limit!==-1" link type="warning" @click="setUnlimited(row)">设为无限额度</el-button>
          <el-button link type="danger" @click="del(row)">{{ t('common.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>
    <div style="margin-top:12px;display:flex;justify-content:flex-end">
      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :page-sizes="[10,20,50,100]"
        :total="filteredItems.length"
        layout="total, sizes, prev, pager, next"
        background
      />
    </div>

    <el-dialog v-model="formVisible" :title="editing ? t('token.editToken') : t('token.addToken')" width="500" :close-on-click-modal="false">
      <el-form :model="form" label-width="100px">
        <el-form-item v-if="!editing" :label="t('token.user')">
          <el-select v-model="form.user_id" filterable placeholder="选择用户" style="width:100%">
            <el-option v-for="u in users" :key="u.id" :label="`${u.username} (ID:${u.id})`" :value="u.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('token.name')"><el-input v-model="form.name" /></el-form-item>
        <el-form-item :label="t('token.models')">
          <ModelMultiSelect v-model="form.models" :allModels="availableModelsForUser" :placeholder="availableModelsForUser.length < modelsStore.allModels.length ? '只能在用户权限范围内选择模型' : '留空=全部模型'" />
        </el-form-item>
        <el-form-item :label="t('token.quotaLimit')"><el-input-number v-model="form.quota_limit" :min="-1" style="width:100%" /><span style="margin-left:8px;color:var(--el-text-color-secondary)">-1={{t('common.unlimited')}}</span></el-form-item>
        <el-form-item v-if="!editing" :label="t('token.expiresAt')"><el-date-picker v-model="form.expires_at" type="datetime" placeholder="留空=永不过期" style="width:100%" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="formVisible=false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="save" :loading="saving">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="rechargeVisible" :title="t('token.recharge')" width="400">
      <el-form :model="rechargeForm" label-width="80px">
        <el-form-item label="金额"><el-input-number v-model="rechargeForm.amount" :min="1" style="width:100%" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="rechargeVisible=false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="doRecharge" :loading="rechargeLoading">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { View, Hide } from '@element-plus/icons-vue'
import { tokenApi } from '@/api/token'
import { userApi } from '@/api/user'
import { useModelsStore } from '@/stores/models'
import ModelMultiSelect from '@/components/common/ModelMultiSelect.vue'
import type { Token, User } from '@/api/types'

const { t } = useI18n()
const modelsStore = useModelsStore()
const items = ref<Token[]>([])
const users = ref<User[]>([])
const loading = ref(false)
const formVisible = ref(false)
const saving = ref(false)
const editing = ref<number|null>(null)
const form = ref<any>({})
const rechargeVisible = ref(false)
const rechargeLoading = ref(false)
const rechargeForm = ref({ id: 0, amount: 100 })

// Search & Pagination
const searchText = ref('')
const currentPage = ref(1)
const pageSize = ref(20)

// Key visibility toggle
const visibleKeys = reactive<Record<number, boolean>>({})

const filteredItems = computed(() => {
  if (!searchText.value) return items.value
  const kw = searchText.value.toLowerCase()
  return items.value.filter((item: Token) =>
    (item.name && item.name.toLowerCase().includes(kw)) ||
    (item.key && item.key.toLowerCase().includes(kw))
  )
})

const pagedItems = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  return filteredItems.value.slice(start, start + pageSize.value)
})

onMounted(() => { load(); loadUsers(); modelsStore.fetchModels() })

async function load() { loading.value = true; try { const d = await tokenApi.list(); items.value = Array.isArray(d) ? d : (d as any).items || [] } catch {}; loading.value = false }
async function loadUsers() { try { const d = await userApi.list(); users.value = Array.isArray(d) ? d : (d as any).items || [] } catch {} }

// When creating a token, filter available models by the selected user's effective models
const availableModelsForUser = computed(() => {
  if (editing.value) return modelsStore.allModels
  if (!form.value.user_id) return modelsStore.allModels
  const u = users.value.find((u: any) => u.id === form.value.user_id)
  if (!u) return modelsStore.allModels
  const authed: string[] = u.authed_models || []
  if (authed.length === 0) return [] as string[]
  return authed
})

function openForm(row?: Token) {
  editing.value = row?.id || null
  if (row) {
    form.value = { name: row.name, models: row.models, quota_limit: row.quota_limit, expires_at: row.expires_at || null }
  } else {
    form.value = { user_id: null, name: '', models: '', quota_limit: -1, expires_at: null }
  }
  formVisible.value = true
}

async function save() {
  saving.value = true
  try {
    const data: any = { ...form.value }
    if (data.expires_at) data.expires_at = new Date(data.expires_at).toISOString()
    else delete data.expires_at
    if (editing.value) {
      // PUT only supports: name, models, enabled, quota_limit
      await tokenApi.update(editing.value, { name: data.name, models: data.models, enabled: data.enabled, quota_limit: data.quota_limit })
    } else {
      if (!data.user_id) { ElMessage.warning('请选择用户'); saving.value = false; return }
      await tokenApi.create(data)
    }
    ElMessage.success(t('common.success')); formVisible.value = false; load()
  } catch {}
  saving.value = false
}

async function setUnlimited(row: Token) {
  try {
    await tokenApi.update(row.id, { quota_limit: -1 })
    ElMessage.success('已设为无限额度')
    load()
  } catch {}
}

async function toggleEnabled(row: Token) { try { await tokenApi.update(row.id, { enabled: row.enabled }) } catch { row.enabled = !row.enabled } }
function copy(key: string) { navigator.clipboard.writeText(key); ElMessage.success(t('common.copied')) }
function openRecharge(row: Token) { rechargeForm.value = { id: row.id, amount: 100 }; rechargeVisible.value = true }
async function doRecharge() { rechargeLoading.value = true; try { await tokenApi.recharge(rechargeForm.value.id, rechargeForm.value.amount); ElMessage.success(t('common.success')); rechargeVisible.value = false; load() } catch {}; rechargeLoading.value = false }
async function del(row: Token) { try { await ElMessageBox.confirm(t('common.confirmDelete'), '', {type:'warning'}); await tokenApi.delete(row.id); load() } catch {} }
</script>
