<template>
  <div>
    <el-row :gutter="14" v-if="stats">
      <el-col :span="6"><StatsCard title="总请求数" :value="stats.total_requests" icon="TrendCharts" color="#4C6EF5" gradient-end="#7C8FF8" /></el-col>
      <el-col :span="6"><StatsCard title="成功请求数" :value="stats.success_requests" icon="CircleCheck" color="#10B981" gradient-end="#34D399" /></el-col>
      <el-col :span="6"><StatsCard title="总Token" :value="stats.total_tokens" icon="Coin" color="#F59E0B" gradient-end="#FBBF24" /></el-col>
      <el-col :span="6"><StatsCard title="成功率" :value="Number(stats.total_requests) > 0 ? (Number(stats.success_requests) / Number(stats.total_requests) * 100).toFixed(1) + '%' : '-'" icon="CircleCheckFilled" color="#10B981" gradient-end="#34D399" /></el-col>
    </el-row>
    <el-row :gutter="14" style="margin-top:14px" v-if="stats">
      <el-col :span="6"><StatsCard title="缓存命中Token" :value="stats.total_cached_tokens || 0" icon="Lightning" color="#8B5CF6" gradient-end="#A78BFA" /></el-col>
      <el-col :span="6"><StatsCard title="缓存未命中Token" :value="stats.total_uncached_tokens || 0" icon="Lightning" color="#F59E0B" gradient-end="#FBBF24" /></el-col>
      <el-col :span="6"><StatsCard title="24h请求" :value="stats.recent_24h_requests" icon="Timer" color="#EF4444" gradient-end="#F87171" /></el-col>
      <el-col :span="6"><StatsCard title="24h用量" :value="stats.recent_24h_tokens" icon="Coin" color="#8B5CF6" gradient-end="#A78BFA" /></el-col>
    </el-row>
    <el-row :gutter="16" style="margin-top:16px">
      <el-col :span="16">
        <el-card><h3 style="font-size:15px;font-weight:600;margin:0 0 12px">用量趋势</h3><div ref="chartRef" style="height:300px"></div></el-card>
      </el-col>
      <el-col :span="8">
        <el-card><h3 style="font-size:15px;font-weight:600;margin:0 0 12px">{{ t('dashboard.modelStats') }}</h3>
          <el-table :data="dashboard?.model_stats || []" size="small" style="width:100%">
            <el-table-column prop="model" label="Model" />
            <el-table-column prop="count" label="Count" width="80" />
            <el-table-column prop="avg_latency" label="Latency" width="80" />
          </el-table>
        </el-card>
      </el-col>
    </el-row>
    <el-card style="margin-top:16px">
      <h3 style="font-size:15px;font-weight:600;margin:0 0 12px">{{ t('dashboard.recentLogs') }}</h3>
      <el-table :data="dashboard?.recent_logs || []" size="small" style="width:100%">
        <el-table-column prop="model" label="Model" />
        <el-table-column prop="latency_ms" label="Latency(ms)" width="100" />
        <el-table-column :label="t('common.status')" width="80"><template #default="{row}"><el-tag :type="row.success?'success':'danger'" size="small">{{row.success?'OK':'Fail'}}</el-tag></template></el-table-column>
        <el-table-column prop="created_at" :label="t('common.createdAt')" width="180" />
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import * as echarts from 'echarts'
import { statsApi } from '@/api/stats'
import StatsCard from '@/components/common/StatsCard.vue'
import type { DashboardData, StatsData, UsageItem } from '@/api/types'

const { t } = useI18n()
const stats = ref<StatsData | null>(null)
const dashboard = ref<DashboardData | null>(null)
const chartRef = ref<HTMLElement>()

onMounted(async () => {
  try {
    const [s, d, u] = await Promise.all([statsApi.stats(), statsApi.operatorDashboard(), statsApi.operatorUsage({ group_by: 'day', page_size: 30 })])
    stats.value = s; dashboard.value = d
    await nextTick()
    setTimeout(() => {
      if (chartRef.value) {
        const chart = echarts.init(chartRef.value)
        const items = [...(u as any).items || []].sort((a: UsageItem, b: UsageItem) => a.key.localeCompare(b.key))
        chart.setOption({
          tooltip: { trigger: 'axis', confine: true },
          grid: { left: 70, right: 70, top: 40, bottom: 50 },
          xAxis: { type: 'category', data: items.map((i: UsageItem) => i.key), axisLabel: { fontSize: 11, rotate: 30 } },
          yAxis: { type: 'value', axisLabel: { fontSize: 11, formatter: (v: number) => v >= 1000000 ? (v / 1000000).toFixed(1) + 'M' : v >= 1000 ? (v / 1000).toFixed(1) + 'K' : String(v) } },
          series: [{ name: 'Requests', type: 'bar', data: items.map((i: UsageItem) => i.requests), itemStyle: { color: '#4C6EF5', borderRadius: [3, 3, 0, 0] }, barMaxWidth: 28 }]
        })
        const ro = new ResizeObserver(() => chart.resize())
        ro.observe(chartRef.value)
        window.addEventListener('resize', () => chart.resize())
      }
    }, 100)
  } catch {}
})
</script>
