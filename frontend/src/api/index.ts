import axios from 'axios'
import { ElMessage } from 'element-plus'
import router from '@/router'

const api = axios.create({
  baseURL: '',
  timeout: 30000,
  headers: { 'Content-Type': 'application/json' },
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

api.interceptors.response.use(
  (response) => response,
  (error) => {
    const status = error.response?.status
    // 401: 未登录，跳转登录页（不弹消息，页面会跳走）
    if (status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      if (!window.location.pathname.includes('/login')) router.push('/login')
      return Promise.reject(error)
    }
    // 403: 权限不足，不全局弹消息，让页面自己处理（静默失败）
    // 页面层面的catch已经处理了，全局弹消息太烦人
    if (status === 403) {
      return Promise.reject(error)
    }
    // 其他错误：弹消息提示
    const msg = error.response?.data?.error?.message || error.response?.data?.message || error.message || '请求失败'
    ElMessage.error(msg)
    return Promise.reject(error)
  },
)

export default api
