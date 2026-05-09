<template>
  <div>
    <div class="page-header">
      <h2 class="page-title">负载均衡状态</h2>
      <div style="display:flex;align-items:center;gap:8px">
        <el-tag type="info" size="small">自动刷新: {{ autoRefresh ? '30s' : '关' }}</el-tag>
        <el-switch v-model="autoRefresh" @change="toggleAutoRefresh" />
        <el-button @click="load" :icon="'Refresh'">{{ t('common.refresh') }}</el-button>
      </div>
    </div>
    <div v-for="group in groups" :key="group.id" style="margin-bottom: 16px">
      <el-card>
        <template #header>
          <div style="display: flex; align-items: center; justify-content: space-between">
            <div>
              <span style="font-weight: 600; font-size: 16px">{{ group.name }}</span>
              <el-tag size="small" style="margin-left: 8px">{{ strategyLabel(group.strategy) }}</el-tag>
              <el-tag size="small" type="info" style="margin-left: 4px">{{ group.channels?.length || 0 }} 个渠道</el-tag>
            </div>
          </div>
        </template>
        <el-table :data="group.channels || []" style="width: 100%" size="small">
          <el-table-column prop="id" label="ID" width="60" />
          <el-table-column prop="name" label="渠道名称" min-width="120" />
          <el-table-column prop="weight" label="权重" width="70" />
          <el-table-column prop="priority" label="优先级" width="70" />
          <el-table-column label="状态" width="90">
            <template #default="{ row }">
              <el-tag :type="statusType(row.status)" size="small">{{ statusText(row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="熔断" width="90">
            <template #default="{ row }">
              <el-tag :type="circuitType(row.circuit)" size="small">{{ circuitText(row.circuit) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="active_requests" label="活跃请求" width="90" />
          <el-table-column prop="total_requests" label="总请求" width="80" />
          <el-table-column prop="global_total_requests" label="全局总请求" width="100" />
          <el-table-column prop="success_rate" label="成功率" width="80" />
          <el-table-column prop="global_success_rate" label="全局成功率" width="100" />
          <el-table-column prop="avg_latency_ms" label="平均延迟(ms)" width="110" />
          <el-table-column prop="fail_count" label="失败次数" width="90" />
          <el-table-column prop="response_time" label="响应时间(ms)" width="110" />
          <el-table-column label="共享" width="60">
            <template #default="{ row }"><span>{{ row.shared ? '✓' : '' }}</span></template>
          </el-table-column>
          <el-table-column label="操作" width="100" fixed="right">
            <template #default="{ row }">
              <el-button v-if="row.circuit === 'open'" link type="warning" @click="resetCircuit(row.id)">重置熔断</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </div>
    <el-empty v-if="!groups.length" description="暂无上游组" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { lbApi } from '@/api/lb'
import type { LBGroupStatus } from '@/api/types'

const { t } = useI18n()
const groups = ref<LBGroupStatus[]>([])
const autoRefresh = ref(true)
let timer: ReturnType<typeof setInterval> | null = null

const strategyLabels: Record<string, string> = { priority: '优先级', round_robin: '轮询', weighted: '加权', least_latency: '最低延迟', least_requests: '最少请求' }
function strategyLabel(s: string) { return strategyLabels[s] || s }
function statusText(s: string) { return { healthy: '健康', unhealthy: '不健康', unknown: '未知' }[s] || s }
function statusType(s: string) { return { healthy: 'success', unhealthy: 'danger', unknown: 'info' }[s] || 'info' }
function circuitText(s: string) { return { closed: '关闭', open: '熔断', half_open: '半开' }[s] || s }
function circuitType(s: string) { return { closed: 'success', open: 'danger', half_open: 'warning' }[s] || 'info' }

onMounted(() => {
  load()
  startAutoRefresh()
})

onUnmounted(() => {
  stopAutoRefresh()
})

function startAutoRefresh() {
  if (timer) clearInterval(timer)
  timer = setInterval(() => load(), 30000)
}

function stopAutoRefresh() {
  if (timer) { clearInterval(timer); timer = null }
}

function toggleAutoRefresh(val: boolean) {
  if (val) startAutoRefresh()
  else stopAutoRefresh()
}

async function load() {
  try {
    const d = await lbApi.status()
    groups.value = d.groups || []
  } catch {}
}

async function resetCircuit(channelId: number) {
  try {
    await lbApi.resetCircuit(channelId)
    ElMessage.success('熔断已重置')
    load()
  } catch {}
}
</script>
