import api from './index'
import type { LoginResponse, User, CaptchaData } from './types'

export const authApi = {
  login(data: { username: string; password: string; captcha_id?: string; captcha_code?: string }): Promise<LoginResponse> {
    return api.post('/api/auth/login', data).then(r => r.data)
  },
  register(data: { username: string; password: string; captcha_id: string; captcha_code: string }): Promise<LoginResponse> {
    return api.post('/api/auth/register', data).then(r => r.data)
  },
  captcha(): Promise<CaptchaData> {
    return api.get('/api/auth/captcha').then(r => ({ captcha_id: r.headers['x-captcha-id'], captcha_image: '' }))
  },
  captchaImage(id: string): Promise<string> {
    return api.get('/api/auth/captcha', { params: { t: Date.now() }, responseType: 'blob' }).then(r => {
      const captchaId = r.headers['x-captcha-id']
      const url = URL.createObjectURL(r.data)
      return { captcha_id: captchaId || id, captcha_image: url }
    })
  },
  me(): Promise<User> {
    return api.get('/api/auth/me').then(r => r.data)
  },
  changePassword(data: { old_password: string; new_password: string }): Promise<any> {
    return api.put('/api/auth/password', data).then(r => r.data)
  },
  refresh(): Promise<{ token: string }> {
    return api.post('/api/auth/refresh').then(r => r.data)
  },
}
