<template>
  <div>
    <div class="page-header">
      <h2 class="page-title">{{ t('usage.title') }}</h2>
      <div style="display:flex;gap:8px">
        <el-input v-model="modelFilter" :placeholder="t('log.model')" clearable style="width:150px" @clear="load" @keyup.enter="load" />
        <el-select v-model="groupBy" style="width:140px" @change="load">
          <el-option :label="t('usage.byDay')" value="day" /><el-option :label="t('usage.byModel')" value="model" />
          <el-option :label="t('usage.byUser')" value="user" /><el-option :label="t('usage.byChannel')" value="channel" />
          <el-option :label="t('usage.byDetail')" value="detail" />
        </el-select>
        <el-date-picker v-model="dateRange" type="daterange" :start-placeholder="t('log.dateRange')" @change="load" />
        <el-button @click="exportCsv">{{ t('usage.exportCsv') }}</el-button>
        <el-button @click="exportXlsx">{{ t('usage.exportXlsx') }}</el-button>
      </div>
    </div>
    <div class="stat-cards" v-if="usage?.summary">
      <StatsCard :title="t('usage.totalRequests')" :value="usage.summary.total_requests" icon="TrendCharts" color="#4C6EF5" gradient-end="#7C8FF8" />
      <StatsCard :title="t('usage.successRequests')" :value="usage.summary.success_requests" icon="CircleCheck" color="#10B981" gradient-end="#34D399" />
      <StatsCard :title="t('usage.promptTokens')" :value="usage.summary.total_prompt_tokens" icon="Download" color="#F59E0B" gradient-end="#FBBF24" />
      <StatsCard :title="t('usage.completionTokens')" :value="usage.summary.total_completion_tokens" icon="Upload" color="#EF4444" gradient-end="#F87171" />
      <StatsCard :title="t('usage.avgLatency')" :value="usage.summary.avg_latency_ms + 'ms'" icon="Timer" color="#14B8A6" gradient-end="#2DD4BF" />
      <StatsCard title="成功率" :value="Number(usage.summary.total_requests) > 0 ? (Number(usage.summary.success_requests) / Number(usage.summary.total_requests) * 100).toFixed(1) + '%' : '-'" icon="CircleCheckFilled" color="#10B981" gradient-end="#34D399" />
      <StatsCard title="缓存命中Token" :value="usage.summary.total_cached_tokens || 0" icon="Lightning" color="#8B5CF6" gradient-end="#A78BFA" />
      <StatsCard title="缓存未命中Token" :value="usage.summary.total_uncached_tokens || 0" icon="Lightning" color="#F59E0B" gradient-end="#FBBF24" />
    </div>
    <el-card style="margin-top:16px">
      <div ref="chartRef" style="height:300px"></div>
    </el-card>
    <el-table :data="usage?.items || []" style="width:100%;margin-top:16px" v-loading="loading">
      <el-table-column v-if="groupBy==='detail'" prop="user" :label="t('usage.byUser')" width="120" />
      <el-table-column v-if="groupBy==='detail'" prop="model" :label="t('usage.byModel')" width="150" />
      <el-table-column v-if="groupBy==='detail'" prop="channel" :label="t('usage.byChannel')" width="120" />
      <el-table-column v-if="groupBy!=='detail'" prop="key" :label="groupByLabel" min-width="120" />
      <el-table-column prop="requests" :label="t('usage.totalRequests')" width="100" />
      <el-table-column prop="success" :label="t('usage.successRequests')" width="100" />
      <el-table-column prop="fail" :label="t('usage.failCount')" width="80" />
      <el-table-column prop="success_rate" :label="t('usage.successRate')" width="80" />
      <el-table-column prop="prompt_tokens" :label="t('usage.promptTokens')" width="120" />
      <el-table-column prop="completion_tokens" :label="t('usage.completionTokens')" width="120" />
      <el-table-column prop="cached_tokens" label="缓存命中" width="120" />
      <el-table-column prop="uncached_tokens" label="缓存未命中" width="120" />
      <el-table-column prop="total_tokens" :label="t('usage.totalTokens')" width="120" />
      <el-table-column prop="avg_latency_ms" :label="t('usage.avgLatency')" width="100" />
    </el-table>
    <el-pagination style="margin-top:16px" v-model:current-page="page" :page-size="20" :total="total" layout="total, prev, pager, next" @current-change="load" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import * as echarts from 'echarts'
import { statsApi } from '@/api/stats'
import { exportApi } from '@/api/export'
import StatsCard from '@/components/common/StatsCard.vue'
import type { UsageResponse, UsageItem } from '@/api/types'

const { t } = useI18n()
const usage = ref<UsageResponse | null>(null)
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const groupBy = ref('day')
const modelFilter = ref('')
const dateRange = ref<Date[] | null>(null)
const chartRef = ref<HTMLElement>()
const groupByLabel = computed(() => ({ day: t('usage.byDay'), model: t('usage.byModel'), user: t('usage.byUser'), channel: t('usage.byChannel'), detail: t('usage.byDetail') }[groupBy.value] || groupBy.value))

onMounted(() => load())

async function load() {
  loading.value = true
  try {
    const params: any = { group_by: groupBy.value, page: page.value, page_size: 20 }
    if (modelFilter.value) params.model = modelFilter.value
    if (dateRange.value?.length === 2) { params.date_from = dateRange.value[0].toISOString().slice(0,10); params.date_to = dateRange.value[1].toISOString().slice(0,10) }
    usage.value = await statsApi.usage(params)
    total.value = usage.value?.total || 0
    await nextTick()
    setTimeout(() => {
      if (chartRef.value && usage.value?.items?.length) {
        const chart = echarts.init(chartRef.value)
        const items = [...usage.value.items].sort((a: UsageItem, b: UsageItem) => a.key.localeCompare(b.key))
        chart.setOption({
          tooltip: { trigger: 'axis', confine: true },
          grid: { left: 70, right: 70, top: 40, bottom: 50 },
          xAxis: { type: 'category', data: items.map((i: UsageItem) => i.key), axisLabel: { fontSize: 11, rotate: 30 } },
          yAxis: [
            { type: 'value', name: 'Requests', nameTextStyle: { fontSize: 12, padding: [0, 0, 0, 20] }, axisLabel: { fontSize: 11, formatter: (v: number) => v >= 1000000 ? (v / 1000000).toFixed(1) + 'M' : v >= 1000 ? (v / 1000).toFixed(1) + 'K' : String(v) } },
            { type: 'value', name: 'Tokens', nameTextStyle: { fontSize: 12, padding: [0, 20, 0, 0] }, axisLabel: { fontSize: 11, formatter: (v: number) => v >= 1000000 ? (v / 1000000).toFixed(1) + 'M' : v >= 1000 ? (v / 1000).toFixed(1) + 'K' : String(v) } }
          ],
          legend: { top: 6, textStyle: { fontSize: 11 } },
          series: [
            { name: 'Requests', type: 'bar', data: items.map((i: UsageItem) => i.requests), itemStyle: { color: '#4C6EF5', borderRadius: [3, 3, 0, 0] }, barMaxWidth: 28 },
            { name: '缓存命中', type: 'bar', stack: 'tokens', data: items.map((i: UsageItem) => i.cached_tokens || 0), itemStyle: { color: '#8B5CF6', borderRadius: [0, 0, 0, 0] }, barMaxWidth: 28 },
            { name: '缓存未命中', type: 'bar', stack: 'tokens', data: items.map((i: UsageItem) => i.uncached_tokens || 0), itemStyle: { color: '#F59E0B', borderRadius: [3, 3, 0, 0] }, barMaxWidth: 28 },
            { name: 'Tokens', type: 'line', yAxisIndex: 1, data: items.map((i: UsageItem) => i.total_tokens), itemStyle: { color: '#F59E0B' }, lineStyle: { width: 2 }, symbol: 'circle', symbolSize: 4, areaStyle: { color: 'rgba(245,158,11,0.08)' } }
          ]
        })
        const ro = new ResizeObserver(() => chart.resize())
        ro.observe(chartRef.value)
        window.addEventListener('resize', () => chart.resize())
      }
    }, 100)
  } catch {}
  loading.value = false
}

function exportCsv() {
  const params: any = { group_by: groupBy.value }
  if (modelFilter.value) params.model = modelFilter.value
  if (dateRange.value?.length === 2) { params.date_from = dateRange.value[0].toISOString().slice(0,10); params.date_to = dateRange.value[1].toISOString().slice(0,10) }
  else if (groupBy.value === 'detail') { const now = new Date(); params.date_to = now.toISOString().slice(0,10); now.setDate(now.getDate()-30); params.date_from = now.toISOString().slice(0,10) }
  exportApi.adminCsv(params)
}
function exportXlsx() {
  const params: any = { group_by: groupBy.value }
  if (modelFilter.value) params.model = modelFilter.value
  if (dateRange.value?.length === 2) { params.date_from = dateRange.value[0].toISOString().slice(0,10); params.date_to = dateRange.value[1].toISOString().slice(0,10) }
  else if (groupBy.value === 'detail') { const now = new Date(); params.date_to = now.toISOString().slice(0,10); now.setDate(now.getDate()-30); params.date_from = now.toISOString().slice(0,10) }
  exportApi.adminXlsx(params)
}
</script>
