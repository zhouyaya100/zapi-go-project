<template>
  <div>
    <div class="page-header">
      <h2 class="page-title">{{ t('notification.title') }}</h2>
      <div style="display: flex; gap: 8px">
        <el-button @click="markAllRead">{{ t('notification.markAllRead') }}</el-button>
      </div>
    </div>
    <el-table :data="items" style="width: 100%">
      <el-table-column prop="title" :label="t('notification.title_field')" min-width="200" />
      <el-table-column prop="category" :label="t('notification.category')" width="120" />
      <el-table-column prop="content" label="内容" min-width="300" />
      <el-table-column :label="t('common.status')" width="80">
        <template #default="{ row }"><el-tag :type="row.read ? 'info' : 'danger'">{{ row.read ? '已读' : '未读' }}</el-tag></template>
      </el-table-column>
      <el-table-column prop="created_at" :label="t('common.createdAt')" width="180" />
      <el-table-column :label="t('common.actions')" width="120">
        <template #default="{ row }">
          <el-button v-if="!row.read" link type="primary" @click="markRead(row.id)">{{ t('notification.markRead') }}</el-button>
          <el-button link type="danger" @click="del(row.id)">{{ t('common.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-pagination style="margin-top: 16px" v-model:current-page="page" :page-size="20" :total="total" layout="total, prev, pager, next" @current-change="load" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { notificationApi } from '@/api/notification'
import type { Notification } from '@/api/types'

const { t } = useI18n()
const items = ref<Notification[]>([])
const total = ref(0)
const page = ref(1)

onMounted(() => load())

async function load() {
  try { const d = await notificationApi.list({ page: page.value, page_size: 20 }); items.value = (d as any).items || d; total.value = (d as any).total || 0 } catch {}
}

async function markRead(id: number) {
  try { await notificationApi.markRead(id); load() } catch {}
}

async function markAllRead() {
  try { await notificationApi.markAllRead(); ElMessage.success(t('common.success')); load() } catch {}
}

async function del(id: number) {
  try { await ElMessageBox.confirm(t('common.confirmDelete'), '', { type: 'warning' }); await notificationApi.delete(id); load() } catch {}
}
</script>
