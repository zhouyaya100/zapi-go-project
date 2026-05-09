import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authApi } from '@/api/auth'
import type { User } from '@/api/types'
import router from '@/router'

export const useAuthStore = defineStore('auth', () => {
  const currentUser = ref<User | null>(null)
  const token = ref(localStorage.getItem('token') || '')

  const isLoggedIn = computed(() => !!token.value && !!currentUser.value)
  const userRole = computed(() => currentUser.value?.role || '')
  const isAdmin = computed(() => userRole.value === 'admin' || isSuper.value)
  const isOperator = computed(() => userRole.value === 'operator')
  const isSuper = computed(() => currentUser.value?.id === 1 || currentUser.value?.is_super === true)

  function setAuth(t: string, user: any) {
    token.value = t
    localStorage.setItem('token', t)
    currentUser.value = user
    localStorage.setItem('user', JSON.stringify(user))
  }

  async function login(username: string, password: string, captcha_id?: string, captcha_code?: string) {
    const data = await authApi.login({ username, password, captcha_id, captcha_code })
    setAuth(data.token, data.user)
    return data
  }

  async function register(username: string, password: string, captcha_id: string, captcha_code: string) {
    const data = await authApi.register({ username, password, captcha_id, captcha_code })
    setAuth(data.token, data.user)
    return data
  }

  function logout() {
    token.value = ''
    currentUser.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    router.push('/login')
  }

  async function fetchMe() {
    try {
      const data = await authApi.me()
      currentUser.value = data
      localStorage.setItem('user', JSON.stringify(data))
    } catch {
      logout()
    }
  }

  function restoreSession() {
    const stored = localStorage.getItem('user')
    if (stored && token.value) {
      try { currentUser.value = JSON.parse(stored) } catch { logout() }
    }
    if (token.value && !currentUser.value) {
      fetchMe()
    }
  }

  async function changePassword(old_password: string, new_password: string) {
    return authApi.changePassword({ old_password, new_password })
  }

  return { currentUser, token, isLoggedIn, userRole, isAdmin, isOperator, isSuper, login, register, logout, fetchMe, restoreSession, changePassword, setAuth }
})
