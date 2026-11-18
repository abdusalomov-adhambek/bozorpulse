<script setup>
import AppIcon from '../../components/AppIcon.vue'
import { money } from '../../utils/money'
import { t } from '../../i18n'
import { ref, onMounted } from 'vue'
import { useAuthStore } from '../../stores/auth'
import api from '../../api/client'
import DataTable from '../../components/DataTable.vue'
import Modal from '../../components/Modal.vue'
import StatusBadge from '../../components/StatusBadge.vue'
import { useAppStore } from '../../stores/app'

const app = useAppStore()
const auth = useAuthStore()
const rows = ref([])
const loading = ref(false)
const error = ref('')
const detailOpen = ref(false)
const selected = ref(null)

const columns = [
  { key: 'id', label: t('ID') },
  { key: 'order_id', label: t('Order') },
  { key: 'amount', label: t('Amount') },
  { key: 'method', label: t('Method') },
  { key: 'status', label: t('Status') },
  { key: 'created_at', label: t('Date') }
]

async function load() {
  loading.value = true
  error.value = ''
  try {
    const data = await api.get('/payments')
    rows.value = Array.isArray(data) ? data : (data?.data || [])
  } catch (err) {
    error.value = err?.message || 'Failed to load payments.'
  } finally {
    loading.value = false
  }
}

async function openDetail(row) {
  selected.value = row
  detailOpen.value = true
  try { selected.value = await api.get(`/payments/${row.id}`) }
  catch (err) { app.error(err?.message || 'Could not load payment details.') }
}

onMounted(load)
</script>

<template>
  <div>
    <div class="page-header">
      <div><h1 class="page-title">{{ t("Payments") }}</h1><p class="page-desc">{{ t('Payments related to orders for your products.') }}</p></div>
      <button class="btn btn-secondary" type="button" :disabled="loading" @click="load">{{ t("Refresh") }}</button>
    </div>

    <div v-if="error" class="alert alert-error mb-3">{{ error }}</div>

    <div class="card">
      <DataTable :columns="columns" :rows="rows" :loading="loading" :empty-text="t('No payments yet')" empty-:desc="t('Payments will appear here as orders are settled.')">
        <template #cell-id="{ row }"><span class="subtle mono">#{{ row.id }}</span></template>
        <template #cell-order_id="{ row }">Order #{{ row.order_id }}</template>
        <template #cell-amount="{ row }"><span class="cell-strong mono">{{ money(row.amount || 0) }}</span></template>
        <template #cell-status="{ row }"><StatusBadge :status="row.status" /></template>
        <template #cell-created_at="{ row }"><span class="subtle">{{ row.created_at ? new Date(row.created_at).toLocaleDateString() : '—' }}</span></template>
        <template #actions="{ row }"><button class="icon-btn btn-sm" type="button" :aria-label="t('View')" :title="t('View')" @click="openDetail(row)"><AppIcon name="eye" :size="15" /></button></template>
      </DataTable>
    </div>

    <Modal :open="detailOpen" :title="`Payment #${selected?.id || ''}`" @close="detailOpen = false">
      <div v-if="selected" class="kv-list">
        <div class="kv-row"><span class="kv-key">{{ t('Order') }}</span><span class="kv-val">#{{ selected.order_id }}</span></div>
        <div class="kv-row"><span class="kv-key">{{ t('Amount') }}</span><span class="kv-val mono">{{ money(selected.amount || 0) }}</span></div>
        <div class="kv-row"><span class="kv-key">{{ t('Method') }}</span><span class="kv-val">{{ selected.method || '—' }}</span></div>
        <div class="kv-row"><span class="kv-key">{{ t('Bank account') }}</span><span class="kv-val mono">{{ selected.bank_account || '—' }}</span></div>
        <div class="kv-row"><span class="kv-key">{{ t('Status') }}</span><StatusBadge :status="selected.status" /></div>
      </div>
      <template #footer><button class="btn btn-secondary" type="button" @click="detailOpen = false">{{ t('Cancel') }}</button></template>
    </Modal>
  </div>
</template>
