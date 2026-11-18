<script setup>
import AppIcon from '../../components/AppIcon.vue'
import { money } from '../../utils/money'
import { t } from '../../i18n'
import { ref, onMounted, watch } from 'vue'
import api from '../../api/client'
import DataTable from '../../components/DataTable.vue'
import Modal from '../../components/Modal.vue'
import StatusBadge from '../../components/StatusBadge.vue'
import OrderStatusBar from '../../components/OrderStatusBar.vue'
import { useAppStore } from '../../stores/app'
import { NumberInput, AppSelect, FormField } from '../../components/ui'

const app = useAppStore()
const rows = ref([])
const loading = ref(false)
const error = ref('')

const suppliers = ref([])
const products = ref([])
const warehouses = ref([])
const modalOpen = ref(false)
const saving = ref(false)
const detailOpen = ref(false)
const selected = ref(null)

const form = ref({ supplier_id: '', product_id: '', quantity: 1, warehouse_id: '' })

const columns = [
  { key: 'id', label: t('Order') },
  { key: 'supplier_id', label: t('Supplier') },
  { key: 'warehouse_id', label: t('Warehouse') },
  { key: 'items', label: t('Items') },
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

async function loadSupport() {
  try {
    const [sups, prods, whs] = await Promise.all([
      api.get('/suppliers').catch(() => []),
      api.get('/products').catch(() => []),
      api.get('/warehouses').catch(() => [])
    ])
    suppliers.value = Array.isArray(sups) ? sups : (sups?.data || [])
    warehouses.value = Array.isArray(whs) ? whs : (whs?.data || [])
    products.value = Array.isArray(prods) ? prods : (prods?.data || [])
  } catch (err) {
    // support data is best-effort
  }
}

async function onSupplierChange() {
  if (!form.value.supplier_id) {
    const all = await api.get('/products').catch(() => [])
    products.value = Array.isArray(all) ? all : (all?.data || [])
    return
  }
  const data = await api.get('/products', { supplier_id: form.value.supplier_id }).catch(() => [])
  products.value = Array.isArray(data) ? data : (data?.data || [])
  form.value.product_id = ''
}

function openCreate() {
  form.value = { supplier_id: '', product_id: '', quantity: 1, warehouse_id: warehouses.value[0]?.id || '' }
  loadSupport()
  modalOpen.value = true
}

async function save() {
  const { supplier_id, product_id, quantity, warehouse_id } = form.value
  if (!supplier_id || !product_id || !quantity || !warehouse_id) {
    app.error(t('Please select a supplier, product, warehouse and quantity.'))
    return
  }
  saving.value = true
  try {
    await api.post('/orders', {
      supplier_id: Number(supplier_id),
      warehouse_id: Number(warehouse_id),
      items: [{ product_id: Number(product_id), quantity: Number(quantity) }]
    })
    app.success(t('Order created successfully'))
    modalOpen.value = false
    await load()
  } catch (err) {
    app.error(err?.message || 'Could not create order.')
  } finally {
    saving.value = false
  }
}

async function openDetail(order) {
  selected.value = order
  detailOpen.value = true
  try {
    selected.value = await api.get(`/orders/${order.id}`)
  } catch (err) {
    app.error(err?.message || 'Could not load order details.')
  }
}

function supplierName(id) {
  return suppliers.value.find((s) => s.id === id)?.company_name || (id ? `Supplier #${id}` : '—')
}
function warehouseName(id) {
  return warehouses.value.find((w) => w.id === id)?.name || (id ? `Warehouse #${id}` : '—')
}

watch(() => modalOpen.value, (open) => { if (open) loadSupport() })
onMounted(() => { load(); loadSupport() })
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t("Orders") }}</h1>
        <p class="page-desc">{{ t('Place new orders and track their progress.') }}</p>
      </div>
      <button class="btn" type="button" @click="openCreate">+ {{ t('New order') }}</button>
    </div>

    <div v-if="error" class="alert alert-error mb-3">{{ error }}</div>

    <div class="card">
      <DataTable :columns="columns" :rows="rows" :loading="loading" :empty-text="t('No orders yet')" empty-:desc="t('Create your first order to get started.')">
        <template #cell-id="{ row }"><span class="cell-strong">#{{ row.id }}</span></template>
        <template #cell-supplier_id="{ row }">{{ supplierName(row.supplier_id) }}</template>
        <template #cell-warehouse_id="{ row }">{{ warehouseName(row.warehouse_id) }}</template>
        <template #cell-items="{ row }">{{ (row.items || []).length }} item(s)</template>
        <template #cell-status="{ row }"><OrderStatusBar :status="row.status" compact /></template>
        <template #cell-total_amount="{ row }"><span class="cell-strong mono">{{ money(row.total_amount || 0) }}</span></template>
        <template #cell-created_at="{ row }"><span class="subtle">{{ row.created_at ? new Date(row.created_at).toLocaleDateString() : '—' }}</span></template>
        <template #actions="{ row }"><button class="icon-btn btn-sm icon-btn-md" type="button" :aria-label="t('View')" :title="t('View')" @click="openDetail(row)"><AppIcon name="eye" :size="18" /></button></template>
      </DataTable>
    </div>

    <Modal :open="modalOpen" :title="t('Create a new order')" @close="modalOpen = false">
      <FormField :label="t('Supplier')" required class="mb-2">
        <AppSelect
          v-model="form.supplier_id"
          :options="suppliers.map(s => ({ value: s.id, label: s.company_name || s.username || `#${s.id}` }))"
          :placeholder="t('Select a supplier')"
          clearable
          @change="onSupplierChange"
        />
      </FormField>
      <FormField :label="t('Product')" required class="mb-2">
        <AppSelect
          v-model="form.product_id"
          :options="products.map(p => ({ value: p.id, label: `${p.name} — ${money((p.effective_price ?? p.price) || 0)}` }))"
          :placeholder="t('Select a product')"
          clearable
        />
      </FormField>
      <div class="form-grid">
        <FormField :label="t('Quantity')" required>
          <NumberInput v-model="form.quantity" :min="1" />
        </FormField>
        <FormField :label="t('Deliver to warehouse')" required :hint="!warehouses.length ? t('Create a warehouse first') : ''">
          <AppSelect
            v-model="form.warehouse_id"
            :options="warehouses.map(w => ({ value: w.id, label: `${w.name} — ${w.city || w.address || ''}` }))"
            :placeholder="t('Select a warehouse')"
            clearable
          />
        </FormField>
      </div>
      <template #footer>
        <button class="btn btn-secondary" type="button" @click="modalOpen = false">{{ t('Cancel') }}</button>
        <button class="btn" type="button" :disabled="saving" @click="save">{{ saving ? t('Creating…') : t('Create order') }}</button>
      </template>
    </Modal>

    <Modal :open="detailOpen" :title="`Order #${selected?.id || ''}`" @close="detailOpen = false">
      <template v-if="selected">
        <div class="status-panel"><OrderStatusBar :status="selected.status" /></div>
        <div class="kv-list mb-3">
          <div class="kv-row"><span class="kv-key">{{ t('Status') }}</span><StatusBadge :status="selected.status" /></div>
          <div class="kv-row"><span class="kv-key">{{ t('Supplier') }}</span><span class="kv-val">{{ supplierName(selected.supplier_id) }}</span></div>
          <div class="kv-row"><span class="kv-key">{{ t('Warehouse') }}</span><span class="kv-val">{{ warehouseName(selected.warehouse_id) }}</span></div>
          <div class="kv-row"><span class="kv-key">{{ t('Total') }}</span><span class="kv-val mono">{{ money(selected.total_amount || 0) }}</span></div>
        </div>
        <div class="card-title mb-2">{{ t('Items') }}</div>
        <div v-if="(selected.items || []).length" class="kv-list">
          <div v-for="(item, i) in selected.items" :key="i" class="kv-row">
            <span class="kv-key">{{ item.name || `Product #${item.product_id}` }} <span class="subtle">× {{ item.quantity }}</span></span>
            <span class="kv-val mono">{{ money(item.line_total || 0) }}</span>
          </div>
        </div>
        <div v-else class="subtle">{{ t('No items in this order.') }}</div>
      </template>
      <template #footer><button class="btn btn-secondary" type="button" @click="detailOpen = false">{{ t('Cancel') }}</button></template>
    </Modal>
  </div>
</template>

<style scoped>
.status-panel { padding: 20px 16px 12px; margin-bottom: 16px; border: 1px solid var(--border); border-radius: var(--radius-sm); background: var(--surface-2); }
</style>
