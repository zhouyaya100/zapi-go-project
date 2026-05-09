<template>
  <div class="sidenav">
    <div class="sidenav-logo" :class="{ 'sidenav-logo-collapsed': collapsed }">
      <span v-if="!collapsed" class="logo-text">ZAPI</span>
      <span v-else class="logo-text-short">Z</span>
    </div>
    <el-menu :default-active="activeMenu" :collapse="collapsed" @select="handleSelect" class="sidenav-menu">
      <template v-if="isAdmin">
        <el-menu-item index="/admin/dashboard"><el-icon><DataBoard /></el-icon><span>{{ t('nav.dashboard') }}</span></el-menu-item>
        <el-menu-item index="/admin/channels"><el-icon><Connection /></el-icon><span>{{ t('nav.channels') }}</span></el-menu-item>
        <el-menu-item index="/admin/upstream-groups"><el-icon><Share /></el-icon><span>{{ t('nav.upstreamGroups') }}</span></el-menu-item>
        <el-menu-item index="/admin/groups"><el-icon><UserFilled /></el-icon><span>{{ t('nav.groups') }}</span></el-menu-item>
        <el-menu-item index="/admin/users"><el-icon><Avatar /></el-icon><span>{{ t('nav.users') }}</span></el-menu-item>
        <el-menu-item index="/admin/tokens"><el-icon><Key /></el-icon><span>{{ t('nav.tokens') }}</span></el-menu-item>
        <el-menu-item index="/admin/logs"><el-icon><Document /></el-icon><span>{{ t('nav.logs') }}</span></el-menu-item>
        <el-menu-item index="/admin/usage"><el-icon><TrendCharts /></el-icon><span>{{ t('nav.usage') }}</span></el-menu-item>
        <el-menu-item index="/admin/lb-status"><el-icon><Odometer /></el-icon><span>负载均衡</span></el-menu-item>
        <el-menu-item v-if="isSuper" index="/admin/settings"><el-icon><Tools /></el-icon><span>{{ t('nav.settings') }}</span></el-menu-item>
        <el-menu-item index="/admin/notifications"><el-icon><Bell /></el-icon><span>{{ t('nav.notifications') }}</span></el-menu-item>
        <el-menu-item-group v-if="collapsed" :title="''" />
        <el-menu-item index="/user/my-tokens"><el-icon><Key /></el-icon><span>{{ t('nav.myTokens') }}</span></el-menu-item>
        <el-menu-item index="/user/my-logs"><el-icon><Document /></el-icon><span>{{ t('nav.myLogs') }}</span></el-menu-item>
      </template>
      <template v-else-if="isOperator">
        <el-menu-item index="/operator/dashboard"><el-icon><DataBoard /></el-icon><span>{{ t('nav.dashboard') }}</span></el-menu-item>
        <el-menu-item index="/operator/logs"><el-icon><Document /></el-icon><span>{{ t('nav.logs') }}</span></el-menu-item>
        <el-menu-item index="/operator/usage"><el-icon><TrendCharts /></el-icon><span>{{ t('nav.usage') }}</span></el-menu-item>
        <el-menu-item-group v-if="collapsed" :title="''" />
        <el-menu-item index="/user/my-tokens"><el-icon><Key /></el-icon><span>{{ t('nav.myTokens') }}</span></el-menu-item>
        <el-menu-item index="/user/my-logs"><el-icon><Document /></el-icon><span>{{ t('nav.myLogs') }}</span></el-menu-item>
        <el-menu-item index="/user/my-usage"><el-icon><TrendCharts /></el-icon><span>{{ t('nav.myUsage') }}</span></el-menu-item>
      </template>
      <template v-else>
        <el-menu-item index="/user/dashboard"><el-icon><DataBoard /></el-icon><span>{{ t('nav.dashboard') }}</span></el-menu-item>
        <el-menu-item index="/user/my-tokens"><el-icon><Key /></el-icon><span>{{ t('nav.myTokens') }}</span></el-menu-item>
        <el-menu-item index="/user/my-logs"><el-icon><Document /></el-icon><span>{{ t('nav.myLogs') }}</span></el-menu-item>
        <el-menu-item index="/user/my-usage"><el-icon><TrendCharts /></el-icon><span>{{ t('nav.myUsage') }}</span></el-menu-item>
      </template>
      <el-menu-item index="/guide"><el-icon><Reading /></el-icon><span>{{ t('nav.guide') }}</span></el-menu-item>
    </el-menu>
    <div class="sidenav-version">
      {{ collapsed ? '' : 'v' + version }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'

defineProps<{ collapsed: boolean }>()
const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const app = useAppStore()
const isAdmin = computed(() => auth.isAdmin)
const isOperator = computed(() => auth.isOperator && !auth.isAdmin)
const isSuper = computed(() => auth.isSuper)
const version = computed(() => app.version || '4.5.9')
const activeMenu = computed(() => route.path)

function handleSelect(index: string) {
  router.push(index)
}
</script>

<style scoped>
.sidenav {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--el-bg-color);
  border-right: none;
  box-shadow: 1px 0 4px rgba(0,0,0,0.04);
}
.sidenav-logo {
  padding: 18px 16px;
  text-align: center;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.sidenav-logo-collapsed {
  padding: 18px 8px;
}
.logo-text {
  font-size: 22px;
  font-weight: 800;
  letter-spacing: 4px;
  background: linear-gradient(135deg, #10B981, #34D399);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}
.logo-text-short {
  font-size: 22px;
  font-weight: 800;
  background: linear-gradient(135deg, #10B981, #34D399);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}
.sidenav-menu {
  border-right: none;
  flex: 1;
  overflow-y: auto;
}
.sidenav-menu .el-menu-item {
  border-radius: 6px;
  margin: 2px 8px;
  transition: all 0.2s ease;
}
.sidenav-menu .el-menu-item:hover {
  background: rgba(76,110,245,0.06);
}
html.dark .sidenav-menu .el-menu-item:hover {
  background: rgba(108,143,248,0.08);
}
.sidenav-menu .el-menu-item.is-active {
  background: rgba(76,110,245,0.1);
  color: var(--zapi-primary);
  font-weight: 600;
  position: relative;
}
html.dark .sidenav-menu .el-menu-item.is-active {
  background: rgba(108,143,248,0.12);
}
.sidenav-menu .el-menu-item.is-active::before {
  content: '';
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 3px;
  height: 60%;
  border-radius: 0 3px 3px 0;
  background: linear-gradient(180deg, var(--zapi-primary), var(--zapi-violet));
}
.sidenav-version {
  padding: 8px 16px;
  font-size: 11px;
  color: var(--el-text-color-secondary);
  text-align: center;
  white-space: nowrap;
  overflow: hidden;
  opacity: 0.7;
}
html.dark .sidenav {
  box-shadow: 1px 0 6px rgba(0,0,0,0.15);
}
</style>
