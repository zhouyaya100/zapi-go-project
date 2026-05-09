import api from './index'
import type { LBGroupStatus } from './types'

export const lbApi = {
  status(): Promise<{ groups: LBGroupStatus[] }> {
    return api.get('/api/lb/status').then(r => r.data)
  },
  resetCircuit(channelId: number): Promise<any> {
    return api.post(`/api/lb/reset-circuit/${channelId}`).then(r => r.data)
  },
}
