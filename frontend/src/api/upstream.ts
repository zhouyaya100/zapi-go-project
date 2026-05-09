import api from './index'
import type { UpstreamGroup, UpstreamGroupChannel } from './types'

export const upstreamApi = {
  list(params?: Record<string, unknown>): Promise<UpstreamGroup[]> {
    return api.get('/api/upstream-groups', { params }).then(r => r.data)
  },
  get(id: number): Promise<UpstreamGroup> {
    return api.get(`/api/upstream-groups/${id}`).then(r => r.data)
  },
  create(data: any): Promise<any> {
    return api.post('/api/upstream-groups', data).then(r => r.data)
  },
  update(id: number, data: any): Promise<any> {
    return api.put(`/api/upstream-groups/${id}`, data).then(r => r.data)
  },
  delete(id: number): Promise<any> {
    return api.delete(`/api/upstream-groups/${id}`).then(r => r.data)
  },
  addChannel(groupId: number, channelId: number): Promise<any> {
    return api.post(`/api/upstream-groups/${groupId}/channels`, { channel_id: channelId }).then(r => r.data)
  },
  removeChannel(groupId: number, channelId: number): Promise<any> {
    return api.delete(`/api/upstream-groups/${groupId}/channels/${channelId}`).then(r => r.data)
  },
}
