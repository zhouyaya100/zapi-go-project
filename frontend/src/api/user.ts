import api from './index'
import type { User, PaginatedData } from './types'

export const userApi = {
  list(params?: Record<string, unknown>): Promise<User[] | PaginatedData<User>> {
    return api.get('/api/users', { params }).then(r => r.data)
  },
  update(id: number, data: Partial<User> & { password?: string }): Promise<any> {
    return api.put(`/api/users/${id}`, data).then(r => r.data)
  },
  delete(id: number): Promise<any> {
    return api.delete(`/api/users/${id}`).then(r => r.data)
  },
  recharge(id: number, amount: number): Promise<any> {
    return api.post(`/api/users/${id}/recharge`, { amount }).then(r => r.data)
  },
  deduct(id: number, amount: number): Promise<any> {
    return api.post(`/api/users/${id}/deduct`, { amount }).then(r => r.data)
  },
}
