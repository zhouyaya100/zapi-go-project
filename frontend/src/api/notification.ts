import api from './index'
import type { Notification, PaginatedData } from './types'

export const notificationApi = {
  list(params?: Record<string, unknown>): Promise<PaginatedData<Notification>> {
    return api.get('/api/notifications', { params }).then(r => r.data)
  },
  unreadCount(): Promise<{ count: number }> {
    return api.get('/api/notifications/unread_count').then(r => r.data)
  },
  markRead(id: number): Promise<any> {
    return api.put(`/api/notifications/${id}/read`).then(r => r.data)
  },
  markAllRead(): Promise<any> {
    return api.put('/api/notifications/read_all').then(r => r.data)
  },
  delete(id: number): Promise<any> {
    return api.delete(`/api/notifications/${id}`).then(r => r.data)
  },
  send(data: { title: string; content: string; category: string; receiver_id?: number | null }): Promise<any> {
    return api.post('/api/notifications', data).then(r => r.data)
  },
  batchSend(data: { receiver_ids: number[]; category: string; title: string; content: string }): Promise<any> {
    return api.post('/api/notifications/batch', data).then(r => r.data)
  },
  sent(params?: Record<string, unknown>): Promise<PaginatedData<Notification>> {
    return api.get('/api/notifications/sent', { params }).then(r => r.data)
  },
  deleteSent(id: number): Promise<any> {
    return api.delete(`/api/notifications/sent/${id}`).then(r => r.data)
  },
  deleteOldSent(): Promise<any> {
    return api.delete('/api/notifications/sent').then(r => r.data)
  },
}
