<template>
  <div>
    <div class="page-header">
      <h2 class="page-title">{{ t('log.title') }}</h2>
    </div>
    <div class="filter-bar">
      <el-input v-model="filters.username" :placeholder="t('log.username')" clearable style="width:150px" @clear="load" @keyup.enter="load" />
      <el-input v-model="filters.model" placeholder="Model" clearable style="width:150px" @clear="load" @keyup.enter="load" />
      <el-input v-model="filters.channel_id" :placeholder="t('log.channelId')" clearable style="width:120px" @clear="load" @keyup.enter="load" />
      <el-select v-model="filters.success" :placeholder="t('common.status')" clearable style="width:120px" @change="load">
        <el-option :label="t('common.all')" value="" />
        <el-option :label="t('common.succeeded')" :value="true" />
        <el-option :label="t('common.failed')" :value="false" />
      </el-select>
      <el-date-picker v-model="filters.dateRange" type="daterange" :start-placeholder="t('log.dateRange')" style="width:240px" @change="load" />
      <el-button type="primary" @click="load">{{ t('common.search') }}</el-button>
      <el-button @click="exportCsv">CSV</el-button>
      <el-button @click="exportXlsx">XLSX</el-button>
    </div>
    <el-table :data="items" style="width:100%" v-loading="loading">
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column prop="username" :label="t('log.username')" width="100" />
      <el-table-column prop="token_name" :label="t('log.tokenName')" width="120" show-overflow-tooltip />
      <el-table-column prop="model" :label="t('log.model')" width="150" />
      <el-table-column prop="channel_name" label="渠道" width="120" />
      <el-table-column :label="t('log.promptTokens')" prop="prompt_tokens" width="90" />
      <el-table-column :label="t('log.completionTokens')" prop="completion_tokens" width="100" />
      <el-table-column label="缓存命中" prop="cached_tokens" width="80" />
      <el-table-column label="缓存未命中" width="90"><template #default="{row}">{{ (row.prompt_tokens || 0) - (row.cached_tokens || 0) }}</template></el-table-column>
      <el-table-column :label="t('log.latency')" prop="latency_ms" width="90" />
      <el-table-column :label="t('common.status')" width="70"><template #default="{row}"><el-tag :type="row.success?'success':'danger'" size="small">{{row.success?'OK':'✕'}}</el-tag></template></el-table-column>
      <el-table-column prop="error_msg" :label="t('log.errorMsg')" min-width="150" show-overflow-tooltip />
      <el-table-column :label="t('log.isStream')" width="60"><template #default="{row}">{{row.is_stream?'✓':''}}</template></el-table-column>
      <el-table-column prop="client_ip" :label="t('log.clientIp')" width="120" />
      <el-table-column prop="created_at" :label="t('common.createdAt')" width="170" />
    </el-table>
    <el-pagination style="margin-top:16px" v-model:current-page="page" :page-size="pageSize" :total="total" :page-sizes="[20,50,100]" layout="total, sizes, prev, pager, next" @current-change="load" @size-change="handleSizeChange" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { logApi } from '@/api/log'
import { exportApi } from '@/api/export'
import type { LogEntry } from '@/api/types'

const { t } = useI18n()
const items = ref<LogEntry[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)
const filters = ref<{ username: string; model: string; channel_id: string; success: boolean | ''; dateRange: Date[] | null }>({ username: '', model: '', channel_id: '', success: '', dateRange: null })

onMounted(() => load())

async function load() {
  loading.value = true
  try {
    const params: any = { page: page.value, page_size: pageSize.value }
    if (filters.value.username) params.username = filters.value.username
    if (filters.value.model) params.model = filters.value.model
    if (filters.value.channel_id) params.channel_id = filters.value.channel_id
    if (filters.value.success !== '') {
      if (filters.value.success === true) params.success = '1'
      else if (filters.value.success === false) params.success = '0'
    }
    if (filters.value.dateRange?.length === 2) { params.date_from = filters.value.dateRange[0].toISOString().slice(0,10); params.date_to = filters.value.dateRange[1].toISOString().slice(0,10) }
    const d = await logApi.list(params)
    items.value = (d as any).items || []; total.value = (d as any).total || 0
  } catch {}
  loading.value = false
}

function handleSizeChange(size: number) { pageSize.value = size; page.value = 1; load() }

function buildExportParams() {
  const params: Record<string, unknown> = {}
  if (filters.value.username) params.username = filters.value.username
  if (filters.value.model) params.model = filters.value.model
  if (filters.value.channel_id) params.channel_id = filters.value.channel_id
  if (filters.value.success !== '') {
    if (filters.value.success === true) params.success = '1'
    else if (filters.value.success === false) params.success = '0'
  }
  if (filters.value.dateRange?.length === 2) {
    params.date_from = filters.value.dateRange[0].toISOString().slice(0,10)
    params.date_to = filters.value.dateRange[1].toISOString().slice(0,10)
  }
  return params
}

function exportCsv() { exportApi.adminCsv(buildExportParams()) }
function exportXlsx() { exportApi.adminXlsx(buildExportParams()) }
</script>
