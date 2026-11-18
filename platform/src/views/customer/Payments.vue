<script setup>
import AppIcon from '../../components/AppIcon.vue'
import { money } from '../../utils/money'
import { t } from '../../i18n'
import { computed, ref, onMounted } from 'vue'
import { useAuthStore } from '../../stores/auth'
import api from '../../api/client'
import DataTable from '../../components/DataTable.vue'
import Modal from '../../components/Modal.vue'
import StatusBadge from '../../components/StatusBadge.vue'
import { useAppStore } from '../../stores/app'
import { AppInput, AppSelect, FormField } from '../../components/ui'

const app = useAppStore()
const auth = useAuthStore()
const rows = ref([])
const orders = ref([])
const loading = ref(false)
const error = ref('')
const modalOpen = ref(false)
const saving = ref(false)
const detailOpen = ref(false)
const selected = ref(null)
const form = ref({ order_id: '', amount: 0, method: '', bank_account: auth.user?.bank_account || '' })
const orderOptions = computed(() => orders.value.map((o) => ({ value: o.id, label: `Order #${o.id} — ${money(o.total_amount || 0)}` })))
const methodOptions = [
  { value: 'bank_transfer', label: t('Bank transfer') },
  { value: 'card', label: t('Card') },
  { value: 'cash', label: t('Cash') },
]

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
    const [payments, ords] = await Promise.all([
      api.get('/payments'),
      api.get('/orders').catch(() => [])
    ])
    rows.value = Array.isArray(payments) ? payments : (payments?.data || [])
    orders.value = Array.isArray(ords) ? ords : (ords?.data || [])
  } catch (err) {
    error.value = err?.message || 'Failed to load payments.'
  } finally {
    loading.value = false
  }
}

function openCreate() {
  form.value = { order_id: orders.value[0]?.id || '', amount: 0, method: 'bank_transfer', bank_account: auth.user?.bank_account || '' }
  modalOpen.value = true
}

async function save() {
  if (!form.value.order_id || !form.value.amount) { app.error(t('Order and amount are required.')); return }
  saving.value = true
  try {
    await api.post('/payments', {
      order_id: Number(form.value.order_id),
      amount: Number(form.value.amount),
      method: form.value.method,
      bank_account: form.value.bank_account
    })
    app.success(t('Payment recorded'))
    modalOpen.value = false
    await load()
  } catch (err) {
    app.error(err?.message || 'Could not record payment.')
  } finally {
    saving.value = false
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
      <div>
        <h1 class="page-title">{{ t("Payments") }}</h1>
        <p class="page-desc">{{ t('Record and track payments for your orders.') }}</p>
      </div>
      <button class="btn" type="button" @click="openCreate">+ {{ t('Record payment') }}</button>
    </div>

    <div v-if="error" class="alert alert-error mb-3">{{ error }}</div>

    <div class="card">
      <DataTable :columns="columns" :rows="rows" :loading="loading" :empty-text="t('No payments yet')" empty-:desc="t('Payments recorded for your orders will appear here.')">
        <template #cell-id="{ row }"><span class="subtle mono">#{{ row.id }}</span></template>
        <template #cell-order_id="{ row }">Order #{{ row.order_id }}</template>
        <template #cell-amount="{ row }"><span class="cell-strong mono">{{ money(row.amount || 0) }}</span></template>
        <template #cell-status="{ row }"><StatusBadge :status="row.status" /></template>
        <template #cell-created_at="{ row }"><span class="subtle">{{ row.created_at ? new Date(row.created_at).toLocaleDateString() : '—' }}</span></template>
        <template #actions="{ row }"><button class="icon-btn btn-sm" type="button" :aria-label="t('View')" :title="t('View')" @click="openDetail(row)"><AppIcon name="eye" :size="15" /></button></template>
      </DataTable>
    </div>

    <Modal :open="modalOpen" :title="t('Record a payment')" @close="modalOpen = false">
      <FormField :label="t('Order')" required class="mb-2">
        <AppSelect v-model="form.order_id" :options="orderOptions" :placeholder="t('Select an order')" />
      </FormField>
      <div class="form-grid">
        <FormField :label="t('Amount')" required>
          <AppInput v-model.number="form.amount" type="number" min="0" step="0.01" align="right" />
        </FormField>
        <FormField :label="t('Method')">
          <AppSelect v-model="form.method" :options="methodOptions" />
        </FormField>
      </div>
      <FormField :label="t('Bank account')" class="mt-2">
        <AppInput v-model="form.bank_account" />
      </FormField>
      <template #footer>
        <button class="btn btn-secondary" type="button" @click="modalOpen = false">{{ t('Cancel') }}</button>
        <button class="btn" type="button" :disabled="saving" @click="save">{{ saving ? 'Saving…' : 'Record payment' }}</button>
      </template>
    </Modal>

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
