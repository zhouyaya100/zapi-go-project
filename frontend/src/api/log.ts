import api from './index'
import type { LogEntry, PaginatedData } from './types'

export const logApi = {
  list(params?: Record<string, unknown>): Promise<PaginatedData<LogEntry>> {
    return api.get('/api/logs', { params }).then(r => r.data)
  },
  operatorLogs(params?: Record<string, unknown>): Promise<PaginatedData<LogEntry>> {
    return api.get('/api/logs/operator', { params }).then(r => r.data)
  },
  myLogs(params?: Record<string, unknown>): Promise<PaginatedData<LogEntry>> {
    return api.get('/api/my/logs', { params }).then(r => r.data)
  },
}
