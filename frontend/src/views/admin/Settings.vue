<template>
  <div>
    <div class="page-header"><h2 class="page-title">{{ t('settings.title') }}</h2></div>
    <el-tabs v-model="activeTab">
      <el-tab-pane :label="t('settings.security')" name="security">
        <el-form :model="form" label-width="180px" style="max-width:600px">
          <el-form-item :label="t('settings.jwtExpire')"><el-input-number v-model="form.jwt_expire_hours" :min="1" /></el-form-item>
          <el-form-item :label="t('settings.corsOrigins')"><el-input v-model="form.cors_origins" placeholder="*" /></el-form-item>
          <el-form-item :label="t('settings.minPasswordLength')"><el-input-number v-model="form.min_password_length" :min="6" /></el-form-item>
          <el-form-item label="时区偏移(小时)"><el-input-number v-model="form.timezone_offset" :step="0.5" /></el-form-item>
        </el-form>
      </el-tab-pane>
      <el-tab-pane :label="t('settings.proxy')" name="proxy">
        <el-form :model="form" label-width="180px" style="max-width:600px">
          <el-form-item :label="t('settings.proxyTimeout')"><el-input-number v-model="form.proxy_timeout" :min="5" /></el-form-item>
          <el-form-item :label="t('settings.maxConnections')"><el-input-number v-model="form.proxy_max_connections" :min="0" /></el-form-item>
          <el-form-item :label="t('settings.retryCount')"><el-input-number v-model="form.proxy_retry_count" :min="0" /></el-form-item>
          <el-form-item :label="t('settings.maxFails')"><el-input-number v-model="form.proxy_max_fails" :min="1" /></el-form-item>
          <el-form-item :label="t('settings.failTimeout')"><el-input-number v-model="form.proxy_fail_timeout" :min="1" /></el-form-item>
          <el-form-item label="最大Keepalive连接"><el-input-number v-model="form.proxy_max_keepalive" :min="0" /></el-form-item>
          <el-form-item label="Keepalive过期时间(秒)"><el-input-number v-model="form.proxy_keepalive_expiry" :min="0" /></el-form-item>
        </el-form>
      </el-tab-pane>
      <el-tab-pane :label="t('settings.cache')" name="cache">
        <el-form :model="form" label-width="180px" style="max-width:600px">
          <el-form-item :label="t('settings.cacheEnabled')"><el-switch v-model="form.cache_enabled" /></el-form-item>
          <el-form-item :label="t('settings.cacheTtl')"><el-input-number v-model="form.cache_ttl" :min="0" /></el-form-item>
          <el-form-item :label="t('settings.cacheMaxEntries')"><el-input-number v-model="form.cache_max_entries" :min="0" /></el-form-item>
        </el-form>
      </el-tab-pane>
      <el-tab-pane :label="t('settings.log')" name="log">
        <el-form :model="form" label-width="180px" style="max-width:600px">
          <el-form-item :label="t('settings.batchSize')"><el-input-number v-model="form.log_batch_size" :min="1" /></el-form-item>
          <el-form-item :label="t('settings.batchInterval')"><el-input-number v-model="form.log_batch_interval" :min="1" /></el-form-item>
          <el-form-item :label="t('settings.retentionDays')"><el-input-number v-model="form.log_retention_days" :min="1" /></el-form-item>
          <el-form-item :label="t('settings.cleanupInterval')"><el-input-number v-model="form.log_cleanup_interval_hours" :min="1" /></el-form-item>
          <el-form-item label="日志清理批量大小"><el-input-number v-model="form.log_cleanup_batch_size" :min="1" /></el-form-item>
        </el-form>
      </el-tab-pane>
      <el-tab-pane :label="t('settings.registration')" name="registration">
        <el-form :model="form" label-width="180px" style="max-width:600px">
          <el-form-item :label="t('settings.allowRegister')"><el-switch v-model="form.allow_register" /></el-form-item>
          <el-form-item :label="t('settings.defaultMaxTokens')"><el-input-number v-model="form.default_max_tokens" :min="0" /></el-form-item>
          <el-form-item :label="t('settings.defaultTokenQuota')"><el-input-number v-model="form.default_token_quota" :min="-1" /></el-form-item>
          <el-form-item :label="t('settings.defaultGroup')"><el-input v-model="form.default_group" /></el-form-item>
        </el-form>
      </el-tab-pane>
      <el-tab-pane :label="t('settings.heartbeat')" name="heartbeat">
        <el-form :model="form" label-width="180px" style="max-width:600px">
          <el-form-item :label="t('settings.heartbeatEnabled')"><el-switch v-model="form.heartbeat_enabled" /></el-form-item>
          <el-form-item :label="t('settings.heartbeatInterval')"><el-input-number v-model="form.heartbeat_interval" :min="10" /></el-form-item>
          <el-form-item :label="t('settings.heartbeatTimeout')"><el-input-number v-model="form.heartbeat_timeout" :min="1" /></el-form-item>
        </el-form>
      </el-tab-pane>
      <el-tab-pane :label="t('settings.errorLog')" name="errorLog">
        <el-form :model="form" label-width="180px" style="max-width:600px;margin-bottom:16px">
          <el-form-item label="错误日志最大条数"><el-input-number v-model="form.error_log_max_entries" :min="100" /></el-form-item>
          <el-form-item label="错误日志最大保留天数"><el-input-number v-model="form.error_log_max_days" :min="1" /></el-form-item>
        </el-form>
        <el-button @click="loadErrorLog" style="margin-bottom:12px">{{ t('settings.viewErrorLog') }}</el-button>
        <el-button type="danger" @click="clearErrorLog" style="margin-bottom:12px">{{ t('settings.clearErrorLog') }}</el-button>
        <el-table :data="errorLogs" style="width:100%" v-if="errorLogs.length">
          <el-table-column prop="time" label="Time" width="180" />
          <el-table-column prop="type" label="Type" width="120" />
          <el-table-column prop="message" label="Message" min-width="300" show-overflow-tooltip />
        </el-table>
      </el-tab-pane>
    </el-tabs>
    <el-button type="primary" @click="save" :loading="saving" style="margin-top:16px">{{ t('common.save') }}</el-button>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { settingsApi } from '@/api/settings'
import type { Settings, ErrorLogEntry } from '@/api/types'

const { t } = useI18n()
const activeTab = ref('security')
const form = ref<Settings>({})
const saving = ref(false)
const errorLogs = ref<ErrorLogEntry[]>([])

onMounted(async () => { try { form.value = await settingsApi.get() } catch {} })

async function save() {
  saving.value = true
  try {
    const { server_host, server_port, database_url, db_pool_size, db_max_overflow, groups, all_models, ...writable } = form.value as any
    await settingsApi.update(writable); ElMessage.success(t('common.success'))
  } catch {}
  saving.value = false
}

async function loadErrorLog() {
  try { const d = await settingsApi.errorLog(); errorLogs.value = d.items || [] } catch {}
}

async function clearErrorLog() {
  try { await ElMessageBox.confirm('确定清空所有错误日志？', '', { type: 'warning' }); await settingsApi.clearErrorLog(); errorLogs.value = []; ElMessage.success(t('common.success')) } catch {}
}
</script>
