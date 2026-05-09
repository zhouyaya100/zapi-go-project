<template>
  <div>
    <div class="page-header">
      <h2 class="page-title">{{ t('usage.title') }}</h2>
      <div style="display:flex;gap:8px">
        <el-input v-model="modelFilter" placeholder="Model" clearable style="width:160px" @clear="load" @keyup.enter="load" />
        <el-select v-model="groupBy" style="width:120px" @change="load">
          <el-option label="按天" value="day" />
          <el-option label="按模型" value="model" />
          <el-option label="按用户" value="user" />
          <el-option label="按渠道" value="channel" />
          <el-option label="按明细" value="detail" />
        </el-select>
        <el-date-picker v-model="dateRange" type="daterange" start-placeholder="日期范围" @change="load" />
        <el-button @click="exportCsv">导出CSV</el-button>
        <el-button @click="exportXlsx">导出XLSX</el-button>
      </div>
    </div>
    <div class="stat-cards" v-if="usage?.summary">
      <StatsCard :title="t('usage.totalRequests')" :value="usage.summary.total_requests" icon="TrendCharts" color="#4C6EF5" gradient-end="#7C8FF8" />
      <StatsCard :title="t('usage.successRequests')" :value="usage.summary.success_requests" icon="CircleCheck" color="#10B981" gradient-end="#34D399" />
      <StatsCard :title="t('usage.promptTokens')" :value="usage.summary.total_prompt_tokens" icon="Download" color="#F59E0B" gradient-end="#FBBF24" />
      <StatsCard :title="t('usage.completionTokens')" :value="usage.summary.total_completion_tokens" icon="Upload" color="#EF4444" gradient-end="#F87171" />
      <StatsCard title="成功率" :value="Number(usage.summary.total_requests) > 0 ? (Number(usage.summary.success_requests) / Number(usage.summary.total_requests) * 100).toFixed(1) + '%' : '-'" icon="CircleCheckFilled" color="#10B981" gradient-end="#34D399" />
      <StatsCard title="缓存命中Token" :value="usage.summary.total_cached_tokens || 0" icon="Lightning" color="#8B5CF6" gradient-end="#A78BFA" />
      <StatsCard title="缓存未命中Token" :value="usage.summary.total_uncached_tokens || 0" icon="Lightning" color="#F59E0B" gradient-end="#FBBF24" />
    </div>
    <el-table :data="usage?.items || []" style="width:100%;margin-top:16px" v-loading="loading">
      <el-table-column prop="key" label="维度" min-width="120" />
      <el-table-column prop="requests" label="请求数" width="100" />
      <el-table-column prop="success" label="成功数" width="100" />
      <el-table-column prop="fail" label="失败数" width="100" />
      <el-table-column prop="success_rate" label="成功率" width="80" />
      <el-table-column prop="prompt_tokens" label="输入Token" width="120" />
      <el-table-column prop="completion_tokens" label="输出Token" width="120" />
      <el-table-column prop="cached_tokens" label="缓存命中" width="120" />
      <el-table-column prop="uncached_tokens" label="缓存未命中" width="120" />
      <el-table-column prop="total_tokens" label="总Token" width="120" />
      <el-table-column prop="avg_latency_ms" label="平均延迟(ms)" width="100" />
    </el-table>
    <el-pagination style="margin-top:16px" v-model:current-page="page" :page-size="20" :total="total" layout="total, prev, pager, next" @current-change="load" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { statsApi } from '@/api/stats'
import { exportApi } from '@/api/export'
import StatsCard from '@/components/common/StatsCard.vue'
import type { UsageResponse } from '@/api/types'

const { t } = useI18n()
const usage = ref<UsageResponse | null>(null)
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const groupBy = ref('day')
const dateRange = ref<Date[] | null>(null)
const modelFilter = ref('')

onMounted(() => load())
async function load() {
  loading.value = true
  try {
    const params: any = { group_by: groupBy.value, page: page.value, page_size: 20 }
    if (modelFilter.value) params.model = modelFilter.value
    if (dateRange.value?.length === 2) { params.date_from = dateRange.value[0].toISOString().slice(0,10); params.date_to = dateRange.value[1].toISOString().slice(0,10) }
    usage.value = await statsApi.operatorUsage(params); total.value = usage.value?.total || 0
  } catch {}
  loading.value = false
}
function exportCsv() {
  const params: any = { group_by: groupBy.value }
  if (modelFilter.value) params.model = modelFilter.value
  if (dateRange.value?.length === 2) { params.date_from = dateRange.value![0].toISOString().slice(0,10); params.date_to = dateRange.value![1].toISOString().slice(0,10) }
  exportApi.operatorCsv(params)
}
function exportXlsx() {
  const params: any = { group_by: groupBy.value }
  if (modelFilter.value) params.model = modelFilter.value
  if (dateRange.value?.length === 2) { params.date_from = dateRange.value![0].toISOString().slice(0,10); params.date_to = dateRange.value![1].toISOString().slice(0,10) }
  exportApi.operatorXlsx(params)
}
</script>
