import api from './index'
import type { Channel, PaginatedData } from './types'

export const channelApi = {
  list(params?: Record<string, unknown>): Promise<Channel[] | PaginatedData<Channel>> {
    return api.get('/api/channels', { params }).then(r => r.data)
  },
  listRevealed(): Promise<Channel[]> {
    return api.get('/api/channels', { params: { reveal: true } }).then(r => r.data)
  },
  create(data: Partial<Channel>): Promise<any> {
    return api.post('/api/channels', data).then(r => r.data)
  },
  update(id: number, data: Partial<Channel>): Promise<any> {
    if (data.api_key && data.api_key.startsWith('***')) delete data.api_key
    return api.put(`/api/channels/${id}`, data).then(r => r.data)
  },
  delete(id: number): Promise<any> {
    return api.delete(`/api/channels/${id}`).then(r => r.data)
  },
  test(id: number): Promise<{ success: boolean; latency_ms: number; model: string; status: string; error?: string }> {
    return api.post(`/api/channels/${id}/test`).then(r => r.data)
  },
  fetchModels(id: number): Promise<{ success: boolean; models: string[]; message?: string }> {
    return api.get(`/api/channels/${id}/fetch-models`).then(r => r.data)
  },
  fetchModelsByCred(baseURL: string, apiKey: string): Promise<{ success: boolean; models: string[]; message?: string }> {
    return api.post(`/api/channels/0/fetch-models`, { base_url: baseURL, api_key: apiKey }).then(r => r.data)
  },
}
