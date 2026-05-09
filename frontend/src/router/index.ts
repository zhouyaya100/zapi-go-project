import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import AppLayout from '@/components/AppLayout.vue'

const routes: RouteRecordRaw[] = [
  { path: '/login', name: 'Login', component: () => import('@/views/Login.vue'), meta: { requiresAuth: false } },
  { path: '/register', name: 'Register', component: () => import('@/views/Register.vue'), meta: { requiresAuth: false } },
  {
    path: '/',
    component: AppLayout,
    meta: { requiresAuth: true },
    children: [
      { path: 'dashboard', redirect: () => {
        const stored = localStorage.getItem('user')
        if (stored) { try { const u = JSON.parse(stored); if (u.role === 'admin') return '/admin/dashboard'; if (u.role === 'operator') return '/operator/dashboard'; return '/user/dashboard' } catch {} }
        return '/login'
      }},
      { path: 'admin/dashboard', name: 'AdminDashboard', component: () => import('@/views/admin/Dashboard.vue'), meta: { requiredRole: ['admin'] } },
      { path: 'admin/channels', name: 'AdminChannels', component: () => import('@/views/admin/Channels.vue'), meta: { requiredRole: ['admin'] } },
      { path: 'admin/upstream-groups', name: 'AdminUpstreamGroups', component: () => import('@/views/admin/UpstreamGroups.vue'), meta: { requiredRole: ['admin'] } },
      { path: 'admin/groups', name: 'AdminGroups', component: () => import('@/views/admin/Groups.vue'), meta: { requiredRole: ['admin'] } },
      { path: 'admin/users', name: 'AdminUsers', component: () => import('@/views/admin/Users.vue'), meta: { requiredRole: ['admin'] } },
      { path: 'admin/tokens', name: 'AdminTokens', component: () => import('@/views/admin/Tokens.vue'), meta: { requiredRole: ['admin'] } },
      { path: 'admin/logs', name: 'AdminLogs', component: () => import('@/views/admin/Logs.vue'), meta: { requiredRole: ['admin'] } },
      { path: 'admin/usage', name: 'AdminUsage', component: () => import('@/views/admin/Usage.vue'), meta: { requiredRole: ['admin'] } },
      { path: 'admin/settings', name: 'AdminSettings', component: () => import('@/views/admin/Settings.vue'), meta: { requiredRole: ['admin'], superAdminOnly: true } },
      { path: 'admin/lb-status', name: 'AdminLBStatus', component: () => import('@/views/admin/LBStatus.vue'), meta: { requiredRole: ['admin'] } },
      { path: 'admin/notifications', name: 'AdminNotifications', component: () => import('@/views/admin/Notifications.vue'), meta: { requiredRole: ['admin'] } },
      { path: 'operator/dashboard', name: 'OperatorDashboard', component: () => import('@/views/operator/Dashboard.vue'), meta: { requiredRole: ['operator', 'admin'] } },
      { path: 'operator/logs', name: 'OperatorLogs', component: () => import('@/views/operator/Logs.vue'), meta: { requiredRole: ['operator', 'admin'] } },
      { path: 'operator/usage', name: 'OperatorUsage', component: () => import('@/views/operator/Usage.vue'), meta: { requiredRole: ['operator', 'admin'] } },
      { path: 'user/dashboard', name: 'UserDashboard', component: () => import('@/views/user/Dashboard.vue'), meta: { requiredRole: ['user', 'operator', 'admin'] } },
      { path: 'user/my-tokens', name: 'UserTokens', component: () => import('@/views/user/Tokens.vue'), meta: { requiredRole: ['user', 'operator', 'admin'] } },
      { path: 'user/my-logs', name: 'UserLogs', component: () => import('@/views/user/Logs.vue'), meta: { requiredRole: ['user', 'operator', 'admin'] } },
      { path: 'user/my-usage', name: 'UserUsage', component: () => import('@/views/user/Usage.vue'), meta: { requiredRole: ['user', 'operator', 'admin'] } },
      { path: 'notifications', name: 'Notifications', component: () => import('@/views/Notifications.vue') },
      { path: 'guide', name: 'Guide', component: () => import('@/views/Guide.vue') },
    ]
  },
  { path: '/:pathMatch(.*)*', redirect: '/dashboard' },
]

const router = createRouter({ history: createWebHistory(import.meta.env.BASE_URL), routes })

router.beforeEach((to, _from, next) => {
  const auth = useAuthStore()
  if (!auth.currentUser && auth.token) auth.restoreSession()
  const requiresAuth = to.matched.some(r => r.meta.requiresAuth !== false)
  const requiredRole = to.meta.requiredRole as string[] | undefined
  const superAdminOnly = to.meta.superAdminOnly as boolean | false
  if (!requiresAuth) {
    if (auth.isLoggedIn && (to.name === 'Login' || to.name === 'Register')) {
      if (auth.isAdmin) return next('/admin/dashboard')
      if (auth.isOperator) return next('/operator/dashboard')
      return next('/user/dashboard')
    }
    return next()
  }
  if (!auth.isLoggedIn) return next({ path: '/login', query: { redirect: to.fullPath } })
  if (superAdminOnly && !auth.isSuper) return next('/admin/dashboard')
  if (requiredRole?.length && !requiredRole.includes(auth.userRole)) {
    if (auth.isAdmin) return next('/admin/dashboard')
    if (auth.isOperator) return next('/operator/dashboard')
    return next('/user/dashboard')
  }
  next()
})

router.afterEach((to) => {
  document.title = to.meta.title ? `${to.meta.title as string} - ZAPI` : 'ZAPI'
})

export default router
