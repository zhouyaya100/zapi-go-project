<template>
  <div>
    <div class="page-header">
      <h2 class="page-title">{{ t('notification.title') }}</h2>
      <div style="display:flex;gap:8px">
        <el-button type="primary" @click="openSend">{{ t('notification.sendNotification') }}</el-button>
        <el-button @click="openBatch">{{ t('notification.batchSend') }}</el-button>
      </div>
    </div>
    <el-tabs v-model="activeTab" @tab-change="handleTabChange">
      <!-- 收件箱 Tab -->
      <el-tab-pane :label="t('notification.inbox')" name="inbox">
        <div style="margin-bottom:12px;display:flex;gap:8px">
          <el-button @click="markAllRead" :disabled="!inboxItems.length">{{ t('notification.markAllRead') }}</el-button>
        </div>
        <el-table :data="inboxItems" style="width:100%" v-loading="inboxLoading">
          <el-table-column :label="t('common.status')" width="60">
            <template #default="{row}">
              <el-tag :type="row.read?'info':'danger'" size="small">{{ row.read ? t('notification.read') : t('notification.unread') }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="title" :label="t('notification.title_field')" min-width="150" />
          <el-table-column prop="category" :label="t('notification.category')" width="100" />
          <el-table-column prop="content" :label="t('notification.content')" min-width="250" show-overflow-tooltip />
          <el-table-column prop="sender_name" :label="t('notification.sender')" width="100" />
          <el-table-column prop="created_at" :label="t('common.createdAt')" width="170" />
          <el-table-column :label="t('common.actions')" width="160">
            <template #default="{row}">
              <el-button v-if="!row.read" link type="primary" @click="markRead(row)">{{ t('notification.markRead') }}</el-button>
              <el-button link type="danger" @click="delInbox(row)">{{ t('common.delete') }}</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-pagination style="margin-top:16px" v-model:current-page="inboxPage" :page-size="20" :total="inboxTotal" layout="total, prev, pager, next" @current-change="loadInbox" />
      </el-tab-pane>
      <!-- 已发送 Tab -->
      <el-tab-pane :label="t('notification.sent')" name="sent">
        <div style="margin-bottom:12px;display:flex;gap:8px">
          <el-button type="danger" @click="deleteOld">{{ t('notification.deleteOld') }}</el-button>
        </div>
        <el-table :data="sentItems" style="width:100%" v-loading="sentLoading">
          <el-table-column prop="title" :label="t('notification.title_field')" min-width="150" />
          <el-table-column prop="category" :label="t('notification.category')" width="100" />
          <el-table-column prop="content" :label="t('notification.content')" min-width="250" show-overflow-tooltip />
          <el-table-column :label="t('notification.receiver')" width="100"><template #default="{row}">{{row.receiver_id||t('notification.broadcast')}}</template></el-table-column>
          <el-table-column prop="recipient_count" label="接收人数" width="80" />
          <el-table-column prop="created_at" :label="t('common.createdAt')" width="170" />
          <el-table-column :label="t('common.actions')" width="80"><template #default="{row}"><el-button link type="danger" @click="delSent(row)">{{ t('common.delete') }}</el-button></template></el-table-column>
        </el-table>
        <el-pagination style="margin-top:16px" v-model:current-page="sentPage" :page-size="20" :total="sentTotal" layout="total, prev, pager, next" @current-change="loadSent" />
      </el-tab-pane>
    </el-tabs>

    <!-- 发送通知弹窗 -->
    <el-dialog v-model="sendVisible" :title="t('notification.sendNotification')" width="500">
      <el-form :model="sendForm" label-width="80px">
        <el-form-item :label="t('notification.title_field')"><el-input v-model="sendForm.title" /></el-form-item>
        <el-form-item :label="t('notification.category')"><el-input v-model="sendForm.category" placeholder="system" /></el-form-item>
        <el-form-item :label="t('notification.content')"><el-input v-model="sendForm.content" type="textarea" :rows="4" /></el-form-item>
        <el-form-item :label="t('notification.receiver')">
          <el-select v-model="sendForm.receiver_id" clearable filterable placeholder="留空=广播所有用户" style="width:100%">
            <el-option label="广播所有用户" :value="null" />
            <el-option v-for="u in users" :key="u.id" :label="`${u.username} (ID:${u.id})`" :value="u.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="sendVisible=false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="doSend" :loading="sending">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <!-- 批量发送弹窗 -->
    <el-dialog v-model="batchVisible" :title="t('notification.batchSend')" width="500">
      <el-form :model="batchForm" label-width="80px">
        <el-form-item :label="t('notification.title_field')"><el-input v-model="batchForm.title" /></el-form-item>
        <el-form-item :label="t('notification.category')"><el-input v-model="batchForm.category" /></el-form-item>
        <el-form-item :label="t('notification.content')"><el-input v-model="batchForm.content" type="textarea" :rows="4" /></el-form-item>
        <el-form-item label="选择用户">
          <el-select v-model="batchForm.receiver_ids" multiple filterable placeholder="选择用户" style="width:100%">
            <el-option v-for="u in users" :key="u.id" :label="`${u.username} (ID:${u.id})`" :value="u.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="batchVisible=false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="doBatch" :loading="sending">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { notificationApi } from '@/api/notification'
import { userApi } from '@/api/user'
import type { Notification, User } from '@/api/types'

const { t } = useI18n()

// Tabs
const activeTab = ref('inbox')

// Users for notification targeting
const users = ref<User[]>([])

// Inbox state
const inboxItems = ref<Notification[]>([])
const inboxTotal = ref(0)
const inboxPage = ref(1)
const inboxLoading = ref(false)

// Sent state
const sentItems = ref<Notification[]>([])
const sentTotal = ref(0)
const sentPage = ref(1)
const sentLoading = ref(false)

// Dialogs
const sendVisible = ref(false)
const batchVisible = ref(false)
const sending = ref(false)
const sendForm = ref({ title: '', category: 'system', content: '', receiver_id: null as number | null })
const batchForm = ref({ title: '', category: 'system', content: '', receiver_ids: [] as number[] })

onMounted(() => {
  loadInbox()
  loadSent()
  loadUsers()
})

async function loadUsers() { try { const d = await userApi.list(); users.value = Array.isArray(d) ? d : (d as any).items || [] } catch {} }

function handleTabChange(tab: string | number) {
  if (tab === 'inbox') loadInbox()
  else if (tab === 'sent') loadSent()
}

// ---- Inbox ----
async function loadInbox() {
  inboxLoading.value = true
  try {
    const d = await notificationApi.list({ page: inboxPage.value, page_size: 20 })
    inboxItems.value = (d as any).items || []
    inboxTotal.value = (d as any).total || 0
  } catch {}
  inboxLoading.value = false
}

async function markRead(row: Notification) {
  try { await notificationApi.markRead(row.id); row.read = true } catch {}
}

async function markAllRead() {
  try { await notificationApi.markAllRead(); ElMessage.success(t('common.success')); loadInbox() } catch {}
}

async function delInbox(row: Notification) {
  try {
    await ElMessageBox.confirm(t('common.confirmDelete'), '', { type: 'warning' })
    await notificationApi.delete(row.id)
    loadInbox()
  } catch {}
}

// ---- Sent ----
async function loadSent() {
  sentLoading.value = true
  try {
    const d = await notificationApi.sent({ page: sentPage.value, page_size: 20 })
    sentItems.value = (d as any).items || []
    sentTotal.value = (d as any).total || 0
  } catch {}
  sentLoading.value = false
}

async function delSent(row: Notification) {
  try {
    await ElMessageBox.confirm(t('common.confirmDelete'), '', { type: 'warning' })
    await notificationApi.deleteSent(row.id)
    loadSent()
  } catch {}
}

async function deleteOld() {
  try {
    await ElMessageBox.confirm('确定清理30天前的通知？', '', { type: 'warning' })
    await notificationApi.deleteOldSent()
    ElMessage.success(t('common.success'))
    loadSent()
  } catch {}
}

// ---- Send ----
function openSend() { sendForm.value = { title: '', category: 'system', content: '', receiver_id: null }; sendVisible.value = true }
function openBatch() { batchForm.value = { title: '', category: 'system', content: '', receiver_ids: [] }; batchVisible.value = true }

async function doSend() {
  sending.value = true
  try { await notificationApi.send(sendForm.value as any); ElMessage.success(t('common.success')); sendVisible.value = false; loadSent() } catch {}
  sending.value = false
}

async function doBatch() {
  if (!batchForm.value.receiver_ids.length) { ElMessage.warning('请选择至少一个用户'); return }
  sending.value = true
  try {
    await notificationApi.batchSend({ title: batchForm.value.title, category: batchForm.value.category, content: batchForm.value.content, receiver_ids: batchForm.value.receiver_ids } as any)
    ElMessage.success(t('common.success')); batchVisible.value = false; loadSent()
  } catch {}
  sending.value = false
}
</script>
