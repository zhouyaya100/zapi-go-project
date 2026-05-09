<template>
  <div>
    <!-- 我的信息 -->
    <h3 class="section-title">我的信息</h3>
    <el-row :gutter="14" v-if="dashboard">
      <el-col :span="6"><StatsCard :title="t('dashboard.totalRequests')" :value="dashboard.total_requests" icon="TrendCharts" color="#4C6EF5" gradient-end="#7C8FF8" /></el-col>
      <el-col :span="6"><StatsCard :title="t('dashboard.successRequests')" :value="dashboard.success_requests" icon="CircleCheck" color="#10B981" gradient-end="#34D399" /></el-col>
      <el-col :span="6"><StatsCard :title="t('dashboard.promptTokens')" :value="dashboard.total_prompt_tokens" icon="Download" color="#F59E0B" gradient-end="#FBBF24" /></el-col>
      <el-col :span="6"><StatsCard :title="t('dashboard.completionTokens')" :value="dashboard.total_completion_tokens" icon="Upload" color="#EF4444" gradient-end="#F87171" /></el-col>
    </el-row>
    <el-row :gutter="14" style="margin-top:14px" v-if="dashboard">
      <el-col :span="6"><StatsCard title="成功率" :value="Number(dashboard.total_requests) > 0 ? (Number(dashboard.success_requests) / Number(dashboard.total_requests) * 100).toFixed(1) + '%' : '-'" icon="CircleCheckFilled" color="#10B981" gradient-end="#34D399" /></el-col>
      <el-col :span="6"><StatsCard title="缓存命中Token" :value="dashboard.total_cached_tokens || 0" icon="Lightning" color="#8B5CF6" gradient-end="#A78BFA" /></el-col>
      <el-col :span="6"><StatsCard title="缓存未命中Token" :value="dashboard.total_uncached_tokens || 0" icon="Lightning" color="#F59E0B" gradient-end="#FBBF24" /></el-col>
      <el-col :span="6"><StatsCard :title="t('dashboard.tokenCount')" :value="dashboard.token_count" icon="Key" color="#14B8A6" gradient-end="#2DD4BF" /></el-col>
      <el-col :span="6"><StatsCard title="额度用量" :value="formatQuota(dashboard.token_quota_used, dashboard.token_quota)" icon="Coin" color="#10B981" gradient-end="#34D399" /></el-col>
    </el-row>
    <el-row :gutter="14" style="margin-top:16px" v-if="dashboard">
      <el-col :span="12">
        <el-card class="info-card">
          <h4>授权模型</h4>
          <div class="model-tags"><el-tag v-for="m in (dashboard.authorized_models || [])" :key="m" style="margin:2px" size="small">{{ m }}</el-tag></div>
          <div v-if="!dashboard.authorized_models?.length" style="color:var(--el-text-color-secondary);font-size:13px">无权限</div>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card class="info-card">
          <h4>限流信息</h4>
          <div class="info-rows">
            <div class="info-row"><span class="info-label">分组</span><span>{{ dashboard.group_name || '默认' }}</span></div>
            <div class="info-row"><span class="info-label">限流模式</span><span>{{ rateModeLabel(dashboard.rate_mode) }}</span></div>
            <div class="info-row"><span class="info-label">RPM</span><span>{{ formatLimit(dashboard.rpm) }}</span></div>
            <div class="info-row"><span class="info-label">TPM</span><span>{{ formatLimit(dashboard.tpm) }}</span></div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 平台统计 -->
    <h3 class="section-title">平台统计</h3>
    <el-row :gutter="14" v-if="stats">
      <el-col :span="6"><StatsCard title="平台总请求" :value="stats.total_requests" icon="TrendCharts" color="#4C6EF5" gradient-end="#7C8FF8" /></el-col>
      <el-col :span="6"><StatsCard title="平台成功请求" :value="stats.success_requests" icon="CircleCheck" color="#10B981" gradient-end="#34D399" /></el-col>
      <el-col :span="6"><StatsCard title="平台总Tokens" :value="stats.total_tokens" icon="Coin" color="#F59E0B" gradient-end="#FBBF24" /></el-col>
      <el-col :span="6"><StatsCard title="平台成功率" :value="Number(stats.total_requests) > 0 ? (Number(stats.success_requests) / Number(stats.total_requests) * 100).toFixed(1) + '%' : '-'" icon="CircleCheckFilled" color="#10B981" gradient-end="#34D399" /></el-col>
    </el-row>
    <el-row :gutter="14" style="margin-top:14px" v-if="stats">
      <el-col :span="6"><StatsCard title="平台缓存命中Token" :value="stats.total_cached_tokens || 0" icon="Lightning" color="#8B5CF6" gradient-end="#A78BFA" /></el-col>
      <el-col :span="6"><StatsCard title="平台缓存未命中Token" :value="stats.total_uncached_tokens || 0" icon="Lightning" color="#F59E0B" gradient-end="#FBBF24" /></el-col>
      <el-col :span="6"><StatsCard :title="t('dashboard.activeChannels')" :value="`${stats.channels_enabled}/${stats.channels}`" icon="Connection" color="#10B981" gradient-end="#34D399" /></el-col>
      <el-col :span="6"><StatsCard :title="t('dashboard.activeUsers')" :value="`${stats.users_enabled}/${stats.users}`" icon="User" color="#14B8A6" gradient-end="#2DD4BF" /></el-col>
    </el-row>
    <el-row :gutter="14" style="margin-top:14px" v-if="stats">
      <el-col :span="12"><StatsCard :title="t('dashboard.recent24h')" :value="stats.recent_24h_requests" icon="Timer" color="#EF4444" gradient-end="#F87171" /></el-col>
      <el-col :span="12"><StatsCard title="24h用量" :value="stats.recent_24h_tokens" icon="Coin" color="#8B5CF6" gradient-end="#A78BFA" /></el-col>
    </el-row>

    <!-- 趋势图 -->
    <el-card style="margin-top:16px">
      <h3 style="font-size:15px;font-weight:600;margin:0 0 12px">请求趋势</h3>
      <div ref="chartRef" style="height:300px;width:100%"></div>
    </el-card>

    <!-- 模型统计 + 最近日志 -->
    <el-row :gutter="14" style="margin-top:16px;display:flex" v-if="dashboard?.model_stats?.length">
      <el-col :span="12" style="display:flex">
        <el-card style="flex:1">
          <h4>模型统计</h4>
          <el-table :data="dashboard.model_stats" size="small" style="width:100%">
            <el-table-column prop="model" label="Model" />
            <el-table-column prop="count" label="请求数" width="80" />
            <el-table-column prop="avg_latency" label="平均延迟" width="80" />
          </el-table>
        </el-card>
      </el-col>
      <el-col :span="12" style="display:flex">
        <el-card style="flex:1">
          <h4>最近日志</h4>
          <el-table :data="dashboard?.recent_logs || []" size="small" style="width:100%">
            <el-table-column prop="model" label="Model" />
            <el-table-column prop="latency_ms" label="延迟" width="60" />
            <el-table-column :label="t('common.status')" width="60"><template #default="{row}"><el-tag :type="row.success?'success':'danger'" size="small">{{row.success?'OK':'✕'}}</el-tag></template></el-table-column>
            <el-table-column prop="created_at" label="时间" width="140" />
          </el-table>
        </el-card>
      </el-col>
    </el-row>
    <el-card style="margin-top:16px" v-else>
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

function rateModeLabel(mode: string) { return mode === 'inherit' ? '继承分组' : mode === 'per_model' ? '按模型' : mode === 'admin' ? '管理员' : '全局' }
function formatLimit(val: number | undefined) { return val === -1 ? '∞' : val === 0 ? '禁止' : val ?? '-' }
function formatQuota(used: number, total: number) {
  const u = Number(used) || 0
  const t = total === -1 ? '∞' : Number(total)
  return `${u.toLocaleString()} / ${t === '∞' ? '∞' : (t as number).toLocaleString()}`
}

onMounted(async () => {
  try {
    const [s, d, u] = await Promise.all([statsApi.stats(), statsApi.adminDashboard(), statsApi.usage({ group_by: 'day', page_size: 30 })])
    stats.value = s; dashboard.value = d
    await nextTick()
    setTimeout(() => {
      if (chartRef.value) {
        const chart = echarts.init(chartRef.value)
        const items = (u as any).items || []
        const sorted = [...items].sort((a: UsageItem, b: UsageItem) => a.key.localeCompare(b.key))
        chart.setOption({
          tooltip: { trigger: 'axis', confine: true },
          grid: { left: 70, right: 70, top: 40, bottom: 50 },
          xAxis: { type: 'category', data: sorted.map((i: UsageItem) => i.key), axisLabel: { fontSize: 11, rotate: 30 } },
          yAxis: [
            { type: 'value', name: 'Requests', nameTextStyle: { fontSize: 12, padding: [0, 0, 0, 20] }, axisLabel: { fontSize: 11, formatter: (v: number) => v >= 1000000 ? (v / 1000000).toFixed(1) + 'M' : v >= 1000 ? (v / 1000).toFixed(1) + 'K' : String(v) } },
            { type: 'value', name: 'Tokens', nameTextStyle: { fontSize: 12, padding: [0, 20, 0, 0] }, axisLabel: { fontSize: 11, formatter: (v: number) => v >= 1000000 ? (v / 1000000).toFixed(1) + 'M' : v >= 1000 ? (v / 1000).toFixed(1) + 'K' : String(v) } }
          ],
          legend: { top: 6, textStyle: { fontSize: 11 } },
          series: [
            { name: 'Requests', type: 'bar', data: sorted.map((i: UsageItem) => i.requests), itemStyle: { color: '#4C6EF5', borderRadius: [3, 3, 0, 0] }, barMaxWidth: 28 },
            { name: '缓存命中', type: 'bar', stack: 'tokens', data: sorted.map((i: UsageItem) => i.cached_tokens || 0), itemStyle: { color: '#8B5CF6', borderRadius: [0, 0, 0, 0] }, barMaxWidth: 28 },
            { name: '缓存未命中', type: 'bar', stack: 'tokens', data: sorted.map((i: UsageItem) => i.uncached_tokens || 0), itemStyle: { color: '#F59E0B', borderRadius: [3, 3, 0, 0] }, barMaxWidth: 28 },
            { name: 'Tokens', type: 'line', yAxisIndex: 1, data: sorted.map((i: UsageItem) => i.total_tokens), itemStyle: { color: '#F59E0B' }, lineStyle: { width: 2 }, symbol: 'circle', symbolSize: 4, areaStyle: { color: 'rgba(245,158,11,0.08)' } }
          ]
        })
        const ro = new ResizeObserver(() => chart.resize())
        ro.observe(chartRef.value)
        window.addEventListener('resize', () => chart.resize())
      }
    }, 100)
  } catch {}
})
</script>

<style scoped>
.info-card { height: 100%; }
.info-card h4 { margin: 0 0 10px 0; font-size: 14px; color: var(--el-text-color-secondary); font-weight: 500; }
.info-rows { display: flex; flex-direction: column; gap: 6px; }
.info-row { display: flex; justify-content: space-between; font-size: 13px; }
.info-label { color: var(--el-text-color-secondary); }
.model-tags { display: flex; flex-wrap: wrap; gap: 4px; }
</style>
