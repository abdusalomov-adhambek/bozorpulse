<script setup>
import AppIcon from '../../components/AppIcon.vue'
import { money } from '../../utils/money'
import { t } from '../../i18n'
import { ref, onMounted } from 'vue'
import api from '../../api/client'
import DataTable from '../../components/DataTable.vue'
import Modal from '../../components/Modal.vue'
import StatusBadge from '../../components/StatusBadge.vue'
import OrderStatusBar from '../../components/OrderStatusBar.vue'
import { useAppStore } from '../../stores/app'

const app = useAppStore()
const rows = ref([])
const loading = ref(false)
const error = ref('')
const detailOpen = ref(false)
const selected = ref(null)

const columns = [
  { key: 'id', label: t('Order') },
  { key: 'customer_id', label: t('Customer') },
  { key: 'supplier_id', label: t('Supplier') },
  { key: 'status', label: t('Status') },
  { key: 'total_amount', label: t('Total') },
  { key: 'created_at', label: t('Date') }
]

async function load() {
  loading.value = true
  error.value = ''
  try {
    const data = await api.get('/orders')
    rows.value = Array.isArray(data) ? data : (data?.data || [])
  } catch (err) {
    error.value = err?.message || 'Failed to load orders.'
  } finally {
    loading.value = false
  }
}

async function openDetail(order) {
  selected.value = order
  detailOpen.value = true
  try {
    const fresh = await api.get(`/orders/${order.id}`)
    selected.value = fresh
  } catch (err) {
    app.error(err?.message || 'Could not load order details.')
  }
}

onMounted(load)
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t("Orders") }}</h1>
        <p class="page-desc">{{ t('All orders placed across the marketplace. Only suppliers can update order status.') }}</p>
      </div>
      <button class="btn btn-secondary" type="button" :disabled="loading" @click="load">{{ t("Refresh") }}</button>
    </div>

    <div v-if="error" class="alert alert-error mb-3">{{ error }}</div>

    <div class="card">
      <DataTable :columns="columns" :rows="rows" :loading="loading" :empty-text="t('No orders yet')" empty-:desc="t('Orders will appear here once customers start buying.')">
        <template #cell-id="{ row }"><span class="cell-strong">#{{ row.id }}</span></template>
        <template #cell-customer_id="{ row }">Customer #{{ row.customer_id }}</template>
        <template #cell-supplier_id="{ row }">Supplier #{{ row.supplier_id }}</template>
        <template #cell-status="{ row }"><OrderStatusBar :status="row.status" compact /></template>
        <template #cell-total_amount="{ row }"><span class="cell-strong mono">{{ money(row.total_amount || 0) }}</span></template>
        <template #cell-created_at="{ row }"><span class="subtle">{{ row.created_at ? new Date(row.created_at).toLocaleDateString() : '—' }}</span></template>
        <template #actions="{ row }">
          <button class="icon-btn btn-sm icon-btn-md" type="button" :aria-label="t('View')" :title="t('View')" @click="openDetail(row)"><AppIcon name="eye" :size="18" /></button>
        </template>
      </DataTable>
    </div>

    <Modal :open="detailOpen" :title="`Order #${selected?.id || ''}`" @close="detailOpen = false">
      <template v-if="selected">
        <div class="status-panel"><OrderStatusBar :status="selected.status" /></div>
        <div class="kv-list mb-3">
          <div class="kv-row"><span class="kv-key">{{ t('Status') }}</span><StatusBadge :status="selected.status" /></div>
          <div class="kv-row"><span class="kv-key">{{ t('Customer') }}</span><span class="kv-val">#{{ selected.customer_id }}</span></div>
          <div class="kv-row"><span class="kv-key">{{ t('Supplier') }}</span><span class="kv-val">#{{ selected.supplier_id }}</span></div>
          <div class="kv-row"><span class="kv-key">{{ t('Warehouse') }}</span><span class="kv-val">#{{ selected.warehouse_id || '—' }}</span></div>
          <div class="kv-row"><span class="kv-key">{{ t('Total') }}</span><span class="kv-val mono">{{ money(selected.total_amount || 0) }}</span></div>
        </div>
        <div class="card-title mb-2">{{ t('Items') }}</div>
        <div v-if="(selected.items || []).length" class="kv-list">
          <div v-for="(item, i) in selected.items" :key="i" class="kv-row">
            <span class="kv-key">{{ item.name }} <span class="subtle">× {{ item.quantity }}</span></span>
            <span class="kv-val mono">{{ money(item.line_total || 0) }}</span>
          </div>
        </div>
        <div v-else class="subtle">{{ t('No items in this order.') }}</div>
      </template>
      <template #footer>
        <button class="btn btn-secondary" type="button" @click="detailOpen = false">{{ t('Cancel') }}</button>
      </template>
    </Modal>
  </div>
</template>

<style scoped>
.status-panel { padding: 20px 16px 12px; margin-bottom: 16px; border: 1px solid var(--border); border-radius: var(--radius-sm); background: var(--surface-2); }
</style>
