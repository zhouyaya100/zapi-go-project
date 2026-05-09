<template>
  <div>
    <div class="page-header">
      <h2 class="page-title">{{ t('upstream.title') }}</h2>
      <el-button type="primary" @click="openForm()">{{ t('upstream.addGroup') }}</el-button>
    </div>
    <el-table :data="items" style="width:100%" v-loading="loading">
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="name" :label="t('upstream.name')" min-width="120" />
      <el-table-column prop="alias" :label="t('upstream.alias')" min-width="120" />
      <el-table-column prop="strategy" :label="t('upstream.strategy')" width="100">
        <template #default="{row}">{{ strategyLabel(row.strategy) }}</template>
      </el-table-column>
      <el-table-column :label="t('upstream.allowedGroups')" min-width="120" show-overflow-tooltip>
        <template #default="{row}">{{ formatGroupNames(row.allowed_groups) }}</template>
      </el-table-column>
      <el-table-column :label="t('common.enabled')" width="70">
        <template #default="{row}"><el-tag :type="row.enabled?'success':'danger'" size="small">{{ row.enabled ? t('common.enabled') : t('common.disabled') }}</el-tag></template>
      </el-table-column>
      <el-table-column :label="t('upstream.channels')" width="70"><template #default="{row}">{{ row.channels?.length || 0 }}</template></el-table-column>
      <el-table-column :label="t('upstream.maxFails')" prop="max_fails" width="70" />
      <el-table-column :label="t('upstream.failTimeout')" prop="fail_timeout" width="90" />
      <el-table-column :label="t('common.actions')" width="150" fixed="right">
        <template #default="{row}">
          <el-button link type="primary" @click="openForm(row)">{{ t('common.edit') }}</el-button>
          <el-button link type="danger" @click="del(row)">{{ t('common.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="formVisible" :title="editing ? t('upstream.editGroup') : t('upstream.addGroup')" width="700" :close-on-click-modal="false">
      <el-form :model="form" label-width="140px">
        <el-form-item :label="t('upstream.name')"><el-input v-model="form.name" /></el-form-item>
        <el-form-item :label="t('upstream.alias')"><el-input v-model="form.alias" placeholder="模型别名" /></el-form-item>
        <el-form-item :label="t('upstream.strategy')">
          <el-select v-model="form.strategy" style="width:100%">
            <el-option label="优先级" value="priority" /><el-option label="轮询" value="round_robin" />
            <el-option label="加权" value="weighted" /><el-option label="最低延迟" value="least_latency" />
            <el-option label="最少请求" value="least_requests" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('upstream.allowedGroups')">
          <el-select v-model="form.allowed_groups_arr" multiple placeholder="留空=所有分组可用" style="width:100%">
            <el-option v-for="g in groups" :key="g.id" :label="g.name" :value="String(g.id)" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('common.enabled')"><el-switch v-model="form.enabled" /></el-form-item>
        <el-form-item :label="t('upstream.healthCheck')"><el-input-number v-model="form.health_check_interval" :min="0" style="width:100%" /></el-form-item>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item :label="t('upstream.maxFails')"><el-input-number v-model="form.max_fails" :min="0" style="width:100%" /></el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label="t('upstream.failTimeout')"><el-input-number v-model="form.fail_timeout" :min="1" style="width:100%" /></el-form-item>
          </el-col>
        </el-row>
        <el-form-item :label="t('upstream.retryOnFail')"><el-switch v-model="form.retry_on_fail" /></el-form-item>
        <el-form-item :label="t('upstream.channels')">
          <el-select v-model="form.channel_ids" multiple placeholder="选择渠道" style="width:100%">
            <el-option v-for="c in allChannels" :key="c.id" :label="`${c.name} (ID:${c.id})`" :value="c.id" />
          </el-select>
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
import { upstreamApi } from '@/api/upstream'
import { channelApi } from '@/api/channel'
import { groupApi } from '@/api/group'
import type { UpstreamGroup, Channel, Group } from '@/api/types'

const { t } = useI18n()
const items = ref<UpstreamGroup[]>([])
const allChannels = ref<Channel[]>([])
const groups = ref<Group[]>([])
const loading = ref(false)
const formVisible = ref(false)
const saving = ref(false)
const editing = ref<number|null>(null)
const form = ref<any>({})

const strategyLabels: Record<string,string> = { priority:'优先级', round_robin:'轮询', weighted:'加权', least_latency:'最低延迟', least_requests:'最少请求' }
function strategyLabel(s: string) { return strategyLabels[s] || s }

function formatGroupNames(allowedGroups: string): string {
  if (!allowedGroups) return '全部'
  return allowedGroups.split(',').map(s => {
    const g = groups.value.find(gr => String(gr.id) === s.trim() || gr.name === s.trim())
    return g ? g.name : s.trim()
  }).join(', ')
}

onMounted(() => { load(); loadChannels(); loadGroups() })

async function load() {
  loading.value = true
  try { const d = await upstreamApi.list(); items.value = Array.isArray(d) ? d : (d as any).items || [] } catch {}
  loading.value = false
}
async function loadChannels() {
  try { const d = await channelApi.list(); allChannels.value = Array.isArray(d) ? d : (d as any).items || [] } catch {}
}
async function loadGroups() { try { groups.value = await groupApi.list() as any } catch {} }

function openForm(row?: UpstreamGroup) {
  editing.value = row?.id || null
  if (row) {
    const allowedArr = row.allowed_groups ? row.allowed_groups.split(',').map(s=>s.trim()).filter(Boolean) : []
    form.value = { ...row, channel_ids: row.channels?.map(c=>c.channel_id||c.id)||[], allowed_groups_arr: allowedArr }
  } else {
    form.value = { name:'', alias:'', strategy:'priority', allowed_groups_arr: [], enabled:true, health_check_interval:60, max_fails:5, fail_timeout:30, retry_on_fail:false, channel_ids:[] }
  }
  formVisible.value = true
}

async function save() {
  saving.value = true
  try {
    const data = { ...form.value }
    data.allowed_groups = (data.allowed_groups_arr || []).join(',')
    delete data.allowed_groups_arr
    delete data.channels
    if (editing.value) await upstreamApi.update(editing.value, data)
    else await upstreamApi.create(data)
    ElMessage.success(t('common.success')); formVisible.value = false; load()
  } catch {}
  saving.value = false
}

async function del(row: UpstreamGroup) {
  try { await ElMessageBox.confirm(t('common.confirmDelete'), '', { type: 'warning' }); await upstreamApi.delete(row.id); load() } catch {}
}
</script>
