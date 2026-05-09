<template>
  <div>
    <div class="page-header">
      <h2 class="page-title">{{ t('channel.title') }}</h2>
      <el-button type="primary" @click="openForm()">{{ t('channel.addChannel') }}</el-button>
    </div>
    <el-table :data="items" style="width:100%" v-loading="loading" row-key="id">
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="name" :label="t('channel.name')" min-width="120" />
      <el-table-column prop="type" :label="t('channel.type')" width="100">
        <template #default="{row}">{{ channelTypes.find(ct => ct.value === row.type)?.label || row.type }}</template>
      </el-table-column>
      <el-table-column prop="base_url" :label="t('channel.baseUrl')" min-width="180" show-overflow-tooltip />
      <el-table-column :label="t('channel.models')" min-width="150" show-overflow-tooltip>
        <template #default="{row}">{{ row.models || '-' }}</template>
      </el-table-column>
      <el-table-column :label="t('channel.modelMapping')" min-width="130" show-overflow-tooltip>
        <template #default="{row}">{{ formatMapping(row.model_mapping) }}</template>
      </el-table-column>
      <el-table-column :label="t('channel.allowedGroups')" min-width="120" show-overflow-tooltip>
        <template #default="{row}">{{ formatGroupNames(row.allowed_groups) }}</template>
      </el-table-column>
      <el-table-column :label="t('channel.upstreamGroups')" min-width="120" show-overflow-tooltip>
        <template #default="{row}">{{ formatUpstreamNames(row) }}</template>
      </el-table-column>
      <el-table-column :label="t('channel.weight')" prop="weight" width="60" />
      <el-table-column :label="t('channel.priority')" prop="priority" width="60" />
      <el-table-column :label="t('common.status')" width="80">
        <template #default="{row}"><el-switch v-model="row.enabled" @change="toggleEnabled(row)" /></template>
      </el-table-column>
      <el-table-column :label="t('channel.failCount')" prop="fail_count" width="70" />
      <el-table-column :label="t('channel.responseTime')" prop="response_time" width="90" />
      <el-table-column :label="t('common.actions')" width="180" fixed="right">
        <template #default="{row}">
          <el-button link type="primary" @click="testCh(row)">{{ t('common.test') }}</el-button>
          <el-button link type="primary" @click="openForm(row)">{{ t('common.edit') }}</el-button>
          <el-button link type="danger" @click="del(row)">{{ t('common.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="formVisible" :title="editing ? t('channel.editChannel') : t('channel.addChannel')" width="700" :close-on-click-modal="false">
      <el-form :model="form" label-width="120px">
        <el-form-item :label="t('channel.name')"><el-input v-model="form.name" /></el-form-item>
        <el-form-item :label="t('channel.type')">
          <el-select v-model="form.type" :placeholder="t('channel.type')" style="width:100%">
            <el-option v-for="ct in channelTypes" :key="ct.value" :label="ct.label" :value="ct.value" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('channel.baseUrl')"><el-input v-model="form.base_url" /></el-form-item>
        <el-form-item :label="t('channel.apiKey')"><el-input v-model="form.api_key" type="password" show-password :placeholder="editing ? t('channel.apiKeyMasked') : ''" /></el-form-item>
        <el-form-item :label="t('channel.models')">
          <div style="display:flex;gap:8px;width:100%">
            <ModelMultiSelect v-model="form.models" :allModels="modelsStore.allModels" placeholder="留空=全部模型" style="flex:1" />
            <el-button type="primary" :loading="fetchingModels" @click="handleFetchModels" :disabled="!form.base_url || !form.api_key" title="从上游自动获取模型列表（需填写接口地址和API密钥）">自动获取</el-button>
          </div>
        </el-form-item>
        <el-form-item :label="t('channel.modelMapping')">
          <div style="width:100%">
            <div v-for="(item, idx) in mappingEntries" :key="idx" style="display:flex;gap:8px;margin-bottom:6px;align-items:center">
              <el-input v-model="item.key" placeholder="对外模型名(用户请求的)" style="flex:1" />
              <span style="color:var(--el-text-color-secondary)">→</span>
              <el-input v-model="item.value" placeholder="上游模型名(发给渠道的)" style="flex:1" />
              <el-button link type="danger" @click="mappingEntries.splice(idx,1)">✕</el-button>
            </div>
            <el-button size="small" @click="mappingEntries.push({key:'',value:''})">+ 添加映射</el-button>
          </div>
        </el-form-item>
        <el-form-item :label="t('channel.allowedGroups')">
          <el-select v-model="form.allowed_groups_arr" multiple placeholder="选择允许的分组（必选）" style="width:100%">
            <el-option v-for="g in groups" :key="g.id" :label="g.name" :value="String(g.id)" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('channel.upstreamGroups')">
          <div style="width:100%">
            <div v-if="form.upstream_group_ids?.length" style="margin-bottom:4px">
              <el-tag v-for="id in (form.upstream_group_ids||[])" :key="id" style="margin:2px">{{ upstreamGroups.find(g=>g.id===id)?.name || '#'+id }}</el-tag>
            </div>
            <div v-else style="color:var(--el-text-color-secondary)">未分配</div>
            <div style="color:var(--el-text-color-placeholder);font-size:12px;margin-top:4px">请在上游组管理中分配渠道</div>
          </div>
        </el-form-item>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item :label="t('channel.weight')"><el-input-number v-model="form.weight" :min="0" style="width:100%" /></el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label="t('channel.priority')"><el-input-number v-model="form.priority" :min="0" style="width:100%" /></el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item :label="t('common.enabled')"><el-switch v-model="form.enabled" /></el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label="t('channel.autoBan')"><el-switch v-model="form.auto_ban" /></el-form-item>
          </el-col>
        </el-row>
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
import { channelApi } from '@/api/channel'
import { groupApi } from '@/api/group'
import { upstreamApi } from '@/api/upstream'
import { useModelsStore } from '@/stores/models'
import ModelMultiSelect from '@/components/common/ModelMultiSelect.vue'
import type { Channel, Group, UpstreamGroup } from '@/api/types'

const { t } = useI18n()
const modelsStore = useModelsStore()
const items = ref<Channel[]>([])
const groups = ref<Group[]>([])
const upstreamGroups = ref<UpstreamGroup[]>([])
const loading = ref(false)
const formVisible = ref(false)
const saving = ref(false)
const editing = ref<number|null>(null)
const form = ref<any>({})
const mappingEntries = ref<{key:string,value:string}[]>([])
const fetchingModels = ref(false)

const channelTypes = [
  { value: 'openai', label: 'OpenAI' },
  { value: 'azure', label: 'Azure' },
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'google', label: 'Google' },
  { value: 'custom', label: 'Custom' },
  { value: 'palm', label: 'PaLM' },
  { value: 'api2d', label: 'API2D' },
  { value: 'baidu', label: 'Baidu' },
  { value: 'zhipu', label: 'Zhipu' },
  { value: 'ali', label: 'Ali' },
  { value: 'xunfei', label: 'Xunfei' },
  { value: 'ai360', label: 'AI360' },
  { value: 'tencent', label: 'Tencent' },
  { value: 'gemini', label: 'Gemini' },
  { value: 'moonshot', label: 'Moonshot' },
  { value: 'baichuan', label: 'Baichuan' },
  { value: 'minimax', label: 'MiniMax' },
  { value: 'mistral', label: 'Mistral' },
  { value: 'cohere', label: 'Cohere' },
  { value: 'deepseek', label: 'DeepSeek' },
]

onMounted(() => { load(); loadGroups(); loadUpstreamGroups(); modelsStore.fetchModels() })

async function load() {
  loading.value = true
  try { const d = await channelApi.list(); items.value = Array.isArray(d) ? d : (d as any).items || [] } catch {}
  loading.value = false
}
async function loadGroups() { try { groups.value = await groupApi.list() as any } catch {} }
async function loadUpstreamGroups() { try { const d = await upstreamApi.list(); upstreamGroups.value = Array.isArray(d) ? d : (d as any).items || [] } catch {} }

function formatMapping(s: string): string {
  if (!s) return '-'
  try { const obj = JSON.parse(s); return Object.entries(obj).map(([k,v])=>`${k}→${v}`).join(', ') || '-' } catch { return s || '-' }
}

function formatGroupNames(allowedGroups: string): string {
  if (!allowedGroups) return '-'
  return allowedGroups.split(',').map(s => {
    const g = groups.value.find(gr => String(gr.id) === s.trim() || gr.name === s.trim())
    return g ? g.name : s.trim()
  }).join(', ')
}

function formatUpstreamNames(row: Channel): string {
  if (!row.upstream_group_ids?.length) return '-'
  return row.upstream_group_ids.map(id => {
    const ug = upstreamGroups.value.find(g => g.id === id)
    return ug ? ug.name : `#${id}`
  }).join(', ')
}

function parseMapping(s: string): {key:string,value:string}[] {
  if (!s) return []
  try {
    const obj = JSON.parse(s)
    return Object.entries(obj).map(([key, value]) => ({ key, value: String(value) }))
  } catch { return [] }
}

function openForm(row?: Channel) {
  editing.value = row?.id || null
  if (row) {
    const allowedArr = row.allowed_groups ? row.allowed_groups.split(',').map(s=>s.trim()).filter(Boolean) : []
    form.value = { ...row, allowed_groups_arr: allowedArr }
    mappingEntries.value = parseMapping(row.model_mapping)
    // Fetch full api_key for editing
    channelApi.listRevealed().then((chs: any[]) => {
      const ch = chs.find((c: any) => c.id === row.id)
      if (ch && ch.api_key) form.value.api_key = ch.api_key
    }).catch(() => {})
  } else {
    form.value = { name: '', type: 'openai', base_url: '', api_key: '', models: '', model_mapping: '', allowed_groups_arr: [], weight: 1, priority: 1, enabled: true, auto_ban: true, upstream_group_ids: [] }
    mappingEntries.value = []
  }
  formVisible.value = true
}

async function save() {
  saving.value = true
  try {
    const data = { ...form.value }
    // mapping entries -> JSON string
    const mappingObj: Record<string,string> = {}
    for (const e of mappingEntries.value) {
      if (e.key.trim()) mappingObj[e.key.trim()] = e.value.trim()
    }
    data.model_mapping = Object.keys(mappingObj).length ? JSON.stringify(mappingObj) : ''
    // allowed_groups array -> comma string
    data.allowed_groups = (data.allowed_groups_arr || []).join(',')
    delete data.allowed_groups_arr
    // Remove read-only fields not accepted by backend
    delete data.upstream_group_ids
    delete data.fail_count
    delete data.test_time
    delete data.response_time
    delete data.created_at
    delete data.api_key_length
    // Edit mode: if api_key is empty, don't submit it (keep existing key)
    if (editing.value && !data.api_key) {
      delete data.api_key
    }
    if (editing.value) await channelApi.update(editing.value, data)
    else await channelApi.create(data)
    ElMessage.success(t('common.success')); formVisible.value = false; load()
  } catch {}
  saving.value = false
}

async function toggleEnabled(row: Channel) {
  try {
    await channelApi.update(row.id, { enabled: row.enabled })
    ElMessage.success(row.enabled ? t('common.enabled') : t('common.disabled'))
  } catch {
    row.enabled = !row.enabled // revert on failure
  }
}

async function del(row: Channel) {
  try { await ElMessageBox.confirm(t('common.confirmDelete'), '', { type: 'warning' }); await channelApi.delete(row.id); load() } catch {}
}

async function testCh(row: Channel) {
  try {
    const r = await channelApi.test(row.id)
    if (r.success) ElMessage.success(`${t('channel.testResult')}: OK - ${r.latency_ms}ms (${r.model})`)
    else ElMessage.error(`${t('channel.testResult')}: ${r.status}`)
  } catch {}
}

async function handleFetchModels() {
  if (!form.value.base_url || !form.value.api_key) {
    ElMessage.warning('请先填写接口地址和API密钥')
    return
  }
  fetchingModels.value = true
  try {
    let r: any
    if (editing.value && form.value.id) {
      // 编辑模式：用渠道ID
      r = await channelApi.fetchModels(editing.value)
    } else {
      // 新增模式：直接传base_url和api_key
      r = await channelApi.fetchModelsByCred(form.value.base_url, form.value.api_key)
    }
    if (r.success && r.models?.length) {
      const existing = form.value.models ? form.value.models.split(',').filter((s: string) => s.trim()) : []
      const newModels = r.models.filter((m: string) => !existing.includes(m))
      if (newModels.length === 0) {
        ElMessage.info('上游模型列表与当前配置一致，无新增模型')
      } else {
        const merged = [...existing, ...newModels].join(',')
        form.value.models = merged
        ElMessage.success(`获取到 ${r.models.length} 个模型，新增 ${newModels.length} 个`)
      }
      modelsStore.reset()
      modelsStore.fetchModels()
    } else {
      ElMessage.error(r.message || '获取模型列表失败')
    }
  } catch (e: any) {
    ElMessage.error('获取模型列表失败：' + (e.message || '未知错误'))
  } finally {
    fetchingModels.value = false
  }
}
</script>
