import api from './index'
import type { Token, PaginatedData } from './types'

// Admin token API (requires admin/operator role)
export const tokenApi = {
  list(params?: Record<string, unknown>): Promise<Token[] | PaginatedData<Token>> {
    return api.get('/api/tokens', { params }).then(r => r.data)
  },
  create(data: Partial<Token> & { user_id?: number; expires_at?: string }): Promise<any> {
    return api.post('/api/tokens', data).then(r => r.data)
  },
  update(id: number, data: Partial<Token>): Promise<any> {
    return api.put(`/api/tokens/${id}`, data).then(r => r.data)
  },
  delete(id: number): Promise<any> {
    return api.delete(`/api/tokens/${id}`).then(r => r.data)
  },
  recharge(id: number, amount: number): Promise<any> {
    return api.post(`/api/tokens/${id}/recharge`, { amount }).then(r => r.data)
  },
}

// User's own token API (any logged-in user)
export const myTokenApi = {
  list(): Promise<Token[] | PaginatedData<Token>> {
    return api.get('/api/my/tokens').then(r => r.data)
  },
  create(data: Partial<Token>): Promise<any> {
    return api.post('/api/my/tokens', data).then(r => r.data)
  },
  update(id: number, data: Partial<Token>): Promise<any> {
    return api.put(`/api/my/tokens/${id}`, data).then(r => r.data)
  },
  delete(id: number): Promise<any> {
    return api.delete(`/api/my/tokens/${id}`).then(r => r.data)
  },
}
