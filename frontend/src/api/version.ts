import api from './index'
import type { VersionInfo } from './types'

export const versionApi = {
  get(): Promise<VersionInfo> {
    return api.get('/api/version').then(r => r.data)
  },
}
