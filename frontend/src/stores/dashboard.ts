import { defineStore } from 'pinia'
import { ref } from 'vue'
import { statsApi } from '@/api/stats'
import type { DashboardData, MyDashboardData, StatsData } from '@/api/types'

export const useDashboardStore = defineStore('dashboard', () => {
  const adminDashboard = ref<DashboardData | null>(null)
  const myDashboard = ref<MyDashboardData | null>(null)
  const stats = ref<StatsData | null>(null)

  async function fetchAdminDashboard() {
    adminDashboard.value = await statsApi.adminDashboard()
  }

  async function fetchMyDashboard() {
    myDashboard.value = await statsApi.myDashboard()
  }

  async function fetchStats() {
    stats.value = await statsApi.stats()
  }

  return { adminDashboard, myDashboard, stats, fetchAdminDashboard, fetchMyDashboard, fetchStats }
})
