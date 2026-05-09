import api from './index'
import type { Settings, PublicSettings, ErrorLogEntry } from './types'

export const settingsApi = {
  get(): Promise<Settings> {
    return api.get('/api/settings').then(r => r.data)
  },
  update(data: Partial<Settings>): Promise<any> {
    return api.put('/api/settings', data).then(r => r.data)
  },
  public(): Promise<PublicSettings> {
    return api.get('/api/settings/public').then(r => r.data)
  },
  errorLog(): Promise<{ items: ErrorLogEntry[] }> {
    return api.get('/api/settings/error-log').then(r => r.data)
  },
  clearErrorLog(): Promise<any> {
    return api.delete('/api/settings/error-log').then(r => r.data)
  },
}
