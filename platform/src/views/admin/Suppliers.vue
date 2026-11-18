<script setup>
import AppIcon from '../../components/AppIcon.vue'
import { t } from '../../i18n'
import { ref, onMounted } from 'vue'
import api from '../../api/client'
import DataTable from '../../components/DataTable.vue'
import StatusBadge from '../../components/StatusBadge.vue'
import { useAppStore } from '../../stores/app'

const app = useAppStore()
const rows = ref([])
const loading = ref(false)
const error = ref('')
const activeTab = ref('all')
const actionId = ref(null)

const tabs = [
  { value: 'all', label: t('All') },
  { value: 'waiting', label: t('Waiting') },
  { value: 'confirmed', label: t('Confirmed') },
  { value: 'rejected', label: t('Rejected') }
]

const columns = [
  { key: 'id', label: t('ID') },
  { key: 'username', label: t('Username') },
  { key: 'company_name', label: t('Company') },
  { key: 'inn', label: t('INN') },
  { key: 'bank_account', label: t('Bank account') },
  { key: 'status', label: t('Status') },
  { key: 'created_at', label: t('Registered') }
]

async function load() {
  loading.value = true
  error.value = ''
  try {
    const params = { role: 'supplier' }
    if (activeTab.value !== 'all') params.status = activeTab.value
    const data = await api.get('/admin/users', params)
    rows.value = Array.isArray(data) ? data : (data?.data || [])
  } catch (err) {
    error.value = err?.message || 'Failed to load suppliers.'
  } finally {
    loading.value = false
  }
}

async function act(user, action) {
  actionId.value = user.id
  try {
    await api.post(`/admin/users/${user.id}/${action}`)
    app.success(`Supplier ${user.username} ${action === 'approve' ? 'approved' : 'rejected'}`)
    await load()
  } catch (err) {
    app.error(err?.message || `Could not ${action} supplier.`)
  } finally {
    actionId.value = null
  }
}

function setTab(tab) {
  activeTab.value = tab
  load()
}

onMounted(load)
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t("Suppliers") }}</h1>
        <p class="page-desc">{{ t('Review and manage supplier registrations.') }}</p>
      </div>
      <button class="btn btn-secondary" type="button" :disabled="loading" @click="load">{{ t("Refresh") }}</button>
    </div>

    <div class="tabs mb-3">
      <button v-for="tab in tabs" :key="tab.value" class="tab" :class="{ active: activeTab === tab.value }" type="button" @click="setTab(tab.value)">
        {{ tab.label }}
      </button>
    </div>

    <div v-if="error" class="alert alert-error mb-3">{{ error }}</div>

    <div class="card">
      <DataTable :columns="columns" :rows="rows" :loading="loading" :empty-text="t('No suppliers found')" :empty-desc="activeTab === 'all' ? 'Supplier registrations will appear here.' : `No ${activeTab} suppliers.`">
        <template #cell-id="{ row }"><span class="subtle mono">#{{ row.id }}</span></template>
        <template #cell-username="{ row }"><span class="cell-strong">{{ row.username }}</span></template>
        <template #cell-status="{ row }"><StatusBadge :status="row.status" /></template>
        <template #cell-created_at="{ row }"><span class="subtle">{{ row.created_at ? new Date(row.created_at).toLocaleDateString() : '—' }}</span></template>
        <template #actions="{ row }">
          <template v-if="row.status === 'waiting'">
            <button class="icon-btn btn-sm success" type="button" :aria-label="t('Approve')" :title="t('Approve')" :disabled="actionId === row.id" @click="act(row, 'approve')">
              <template v-if="actionId === row.id">…</template>
              <template v-else><AppIcon name="check" :size="15" /></template>
            </button>
            <button class="icon-btn btn-sm danger" type="button" :aria-label="t('Reject')" :title="t('Reject')" :disabled="actionId === row.id" @click="act(row, 'reject')"><AppIcon name="x" :size="15" /></button>
          </template>
          <span v-else class="subtle">—</span>
        </template>
      </DataTable>
    </div>
  </div>
</template>
