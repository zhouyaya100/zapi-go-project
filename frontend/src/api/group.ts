import api from './index'
import type { Group } from './types'

export const groupApi = {
  list(): Promise<Group[]> {
    return api.get('/api/groups').then(r => r.data)
  },
  create(data: Partial<Group>): Promise<any> {
    return api.post('/api/groups', data).then(r => r.data)
  },
  update(id: number, data: Partial<Group>): Promise<any> {
    return api.put(`/api/groups/${id}`, data).then(r => r.data)
  },
  delete(id: number): Promise<any> {
    return api.delete(`/api/groups/${id}`).then(r => r.data)
  },
}
