import api from './index'
import type { DashboardData, MyDashboardData, StatsData, UsageResponse } from './types'

export const statsApi = {
  adminDashboard(): Promise<DashboardData> {
    return api.get('/api/dashboard').then(r => r.data)
  },
  myDashboard(): Promise<MyDashboardData> {
    return api.get('/api/my/dashboard').then(r => r.data)
  },
  stats(): Promise<StatsData> {
    return api.get('/api/stats').then(r => r.data)
  },
  usage(params?: Record<string, unknown>): Promise<UsageResponse> {
    return api.get('/api/stats/usage', { params }).then(r => r.data)
  },
  operatorUsage(params?: Record<string, unknown>): Promise<UsageResponse> {
    return api.get('/api/stats/usage/operator', { params }).then(r => r.data)
  },
  myUsage(params?: Record<string, unknown>): Promise<UsageResponse> {
    return api.get('/api/my/usage', { params }).then(r => r.data)
  },
  myModels(): Promise<{ models: string[] }> {
    return api.get('/api/my/models').then(r => r.data)
  },
  operatorDashboard(): Promise<DashboardData> {
    return api.get('/api/dashboard').then(r => r.data)
  },
}
