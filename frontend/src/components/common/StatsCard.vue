<template>
  <div class="stat-card" :style="{ '--accent': color || '#4C6EF5', '--accent-end': gradientEnd || color || '#8B5CF6' }">
    <div class="stat-card-icon">
      <el-icon :size="22"><component :is="icon" /></el-icon>
    </div>
    <div class="stat-card-body">
      <div class="stat-card-title">{{ title }}</div>
      <div class="stat-card-value" :style="{ color: color || '#4C6EF5' }">{{ displayValue }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted } from 'vue'

const props = defineProps<{ title: string; value: number | string; icon?: string; color?: string; gradientEnd?: string }>()

const animatedValue = ref<number>(0)

const formatNum = (v: number): string => {
  if (v === -1) return '∞'
  if (v >= 1000000000) return (v / 1000000000).toFixed(1) + 'B'
  if (v >= 1000000) return (v / 1000000).toFixed(1) + 'M'
  if (v >= 10000) return (v / 1000).toFixed(1) + 'K'
  return v.toLocaleString()
}

const displayValue = computed(() => {
  let v = props.value
  if (typeof v === 'string') {
    // Handle "已用/总额" format — no animation
    if (v.includes('/')) return v
    const n = Number(v)
    if (!isNaN(n) && isFinite(n)) v = n
    else return v
  }
  if (typeof v === 'number') return formatNum(Math.round(animatedValue.value))
  return String(v)
})

onMounted(() => {
  const target = typeof props.value === 'number' ? props.value : Number(props.value)
  if (isNaN(target) || !isFinite(target)) return
  const duration = 400
  const start = performance.now()
  const tick = (now: number) => {
    const elapsed = now - start
    const progress = Math.min(elapsed / duration, 1)
    // ease-out cubic
    const ease = 1 - Math.pow(1 - progress, 3)
    animatedValue.value = target * ease
    if (progress < 1) requestAnimationFrame(tick)
    else animatedValue.value = target
  }
  requestAnimationFrame(tick)
})
</script>

<style scoped>
.stat-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 18px;
  border-radius: 10px;
  background: var(--el-bg-color);
  border: none;
  box-shadow: var(--zapi-shadow);
  transition: box-shadow 0.25s ease, transform 0.2s ease;
  min-width: 0;
  overflow: hidden;
  animation: fadeUp 0.35s ease-out;
}
.stat-card:hover {
  box-shadow: var(--zapi-shadow-hover);
  transform: translateY(-2px);
}
.stat-card:hover .stat-card-icon {
  transform: rotate(3deg) scale(1.05);
}
.stat-card-icon {
  flex-shrink: 0;
  width: 44px;
  height: 44px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--accent), var(--accent-end));
  color: #fff;
  transition: transform 0.25s ease;
}
.stat-card-body {
  overflow: hidden;
  flex: 1;
  min-width: 0;
}
.stat-card-title {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  letter-spacing: 0.3px;
}
.stat-card-value {
  font-size: 24px;
  font-weight: 700;
  line-height: 1.2;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-variant-numeric: tabular-nums;
}
html.dark .stat-card-icon {
  opacity: 0.9;
}
</style>
