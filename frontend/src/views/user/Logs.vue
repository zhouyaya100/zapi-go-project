<template>
  <div>
    <div class="page-header"><h2 class="page-title">{{ t('nav.myLogs') }}</h2></div>
    <div class="filter-bar">
      <el-input v-model="filters.model" placeholder="Model" clearable style="width:200px" @clear="load" @keyup.enter="load" />
      <el-select v-model="filters.success" clearable placeholder="状态" style="width:120px" @change="load">
        <el-option label="全部" value="" />
        <el-option label="成功" :value="true" />
        <el-option label="失败" :value="false" />
      </el-select>
      <el-date-picker v-model="filters.dateRange" type="daterange" start-placeholder="日期范围" style="width:240px" @change="load" />
      <el-button type="primary" @click="load">{{ t('common.search') }}</el-button>
      <el-button @click="exportCsv">导出CSV</el-button>
      <el-button @click="exportXlsx">导出XLSX</el-button>
    </div>
    <el-table :data="items" style="width:100%" v-loading="loading">
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column prop="model" :label="t('log.model')" width="150" />
      <el-table-column prop="token_name" :label="t('log.tokenName')" width="120" show-overflow-tooltip />
      <el-table-column :label="t('log.promptTokens')" prop="prompt_tokens" width="90" />
      <el-table-column :label="t('log.completionTokens')" prop="completion_tokens" width="100" />
      <el-table-column label="缓存命中" prop="cached_tokens" width="80" />
      <el-table-column label="缓存未命中" width="90"><template #default="{row}">{{ (row.prompt_tokens || 0) - (row.cached_tokens || 0) }}</template></el-table-column>
      <el-table-column :label="t('log.latency')" prop="latency_ms" width="90" />
      <el-table-column :label="t('common.status')" width="70"><template #default="{row}"><el-tag :type="row.success?'success':'danger'" size="small">{{row.success?'OK':'✕'}}</el-tag></template></el-table-column>
      <el-table-column prop="error_msg" :label="t('log.errorMsg')" min-width="150" show-overflow-tooltip />
      <el-table-column label="流式" width="50"><template #default="{row}">{{row.is_stream?'✓':''}}</template></el-table-column>
      <el-table-column prop="channel_name" label="渠道" width="100" show-overflow-tooltip />
      <el-table-column prop="created_at" :label="t('common.createdAt')" width="170" />
    </el-table>
    <el-pagination style="margin-top:16px" v-model:current-page="page" :page-size="20" :total="total" layout="total, prev, pager, next" @current-change="load" />
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
const loading = ref(false)
const filters = ref<{ model: string; success: boolean | string; dateRange: Date[] | null }>({ model: '', success: '', dateRange: null })

onMounted(() => load())
async function load() {
  loading.value = true
  try {
    const params: any = { page: page.value, page_size: 20 }
    if (filters.value.model) params.model = filters.value.model
    if (filters.value.success !== '') {
      if (filters.value.success === true) params.success = '1'
      else if (filters.value.success === false) params.success = '0'
    }
    if (filters.value.dateRange?.length === 2) { params.date_from = filters.value.dateRange[0].toISOString().slice(0,10); params.date_to = filters.value.dateRange[1].toISOString().slice(0,10) }
    const d = await logApi.myLogs(params)
    items.value = (d as any).items || []; total.value = (d as any).total || 0
  } catch {}
  loading.value = false
}
function getExportParams() {
  const params: any = {}
  if (filters.value.model) params.model = filters.value.model
  if (filters.value.success !== '') {
    if (filters.value.success === true) params.success = '1'
    else if (filters.value.success === false) params.success = '0'
  }
  if (filters.value.dateRange?.length === 2) { params.date_from = filters.value.dateRange![0].toISOString().slice(0,10); params.date_to = filters.value.dateRange![1].toISOString().slice(0,10) }
  return params
}
function exportCsv() { exportApi.myCsv(getExportParams()) }
function exportXlsx() { exportApi.myXlsx(getExportParams()) }
</script>
