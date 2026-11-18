<script setup>
import { t } from '../../i18n'
import { ref, onMounted, watch } from 'vue'
import api from '../../api/client'
import DataTable from '../../components/DataTable.vue'
import { useAppStore } from '../../stores/app'
import { AppInput, AppSelect, FormField } from '../../components/ui'

const app = useAppStore()
const warehouses = ref([])
const products = ref([])
const selected = ref('')
const movements = ref([])
const loading = ref(false)
const error = ref('')
const saving = ref(false)
const form = ref({ product_id: '', quantity: 1, reason: '' })

const tabs = [
  { label: t('Overview'), to: '/customer/warehouses' },
  { label: t('Stock'), to: '/customer/warehouses/stock' },
  { label: t('Transfers'), to: '/customer/warehouses/transfers' },
  { label: t('Supplies'), to: '/customer/warehouses/supplies' },
  { label: t('Write-off'), to: '/customer/warehouses/writeoffs' },
  { label: t('Returns'), to: '/customer/warehouses/returns' }
]
const columns = [
  { key: 'id', label: t('ID') },
  { key: 'product_id', label: t('Product') },
  { key: 'quantity', label: t('Quantity') },
  { key: 'type', label: t('Type') },
  { key: 'note', label: t('Note') },
  { key: 'created_at', label: t('Date') }
]

async function loadSupport() {
  try {
    const [whs, prods] = await Promise.all([api.get('/warehouses'), api.get('/products').catch(() => [])])
    warehouses.value = Array.isArray(whs) ? whs : (whs?.data || [])
    products.value = Array.isArray(prods) ? prods : (prods?.data || [])
    if (!selected.value && warehouses.value.length) selected.value = warehouses.value[0].id
  } catch (err) {
    app.error(err?.message || 'Could not load warehouses.')
  }
}

async function loadMovements() {
  if (!selected.value) { movements.value = []; return }
  loading.value = true
  error.value = ''
  try {
    const data = await api.get('/inventory/movements', { warehouse_id: selected.value })
    movements.value = Array.isArray(data) ? data : (data?.data || [])
  } catch (err) {
    error.value = err?.message || 'Failed to load movements.'
  } finally {
    loading.value = false
  }
}

async function submit() {
  if (!selected.value || !form.value.product_id || !form.value.quantity || !form.value.reason) {
    app.error(t('Warehouse, product, quantity and reason are required.')); return
  }
  saving.value = true
  try {
    await api.post('/inventory/write-offs', {
      warehouse_id: Number(selected.value),
      product_id: Number(form.value.product_id),
      quantity: Number(form.value.quantity),
      reason: form.value.reason
    })
    app.success(t('Write-off recorded'))
    form.value = { product_id: '', quantity: 1, reason: '' }
    await loadMovements()
  } catch (err) {
    app.error(err?.message || 'Could not record write-off.')
  } finally {
    saving.value = false
  }
}

watch(selected, loadMovements)
onMounted(async () => { await loadSupport(); loadMovements() })
</script>

<template>
  <div>
    <div class="page-header">
      <div><h1 class="page-title">{{ t('Write-off') }}</h1><p class="page-desc">{{ t('Record damaged or expired stock leaving a warehouse.') }}</p></div>
    </div>

    <div class="tabs mb-3">
      <RouterLink v-for="tab in tabs" :key="tab.to" :to="tab.to" class="tab tab-sub" :class="{ active: $route.path === tab.to }">{{ tab.label }}</RouterLink>
    </div>

    <div class="grid grid-2 mb-3">
      <div class="card card-pad">
        <div class="card-title mb-3">{{ t('New write-off') }}</div>
        <div class="field mb-2">
          <label class="field-label">{{ t('Warehouse *') }}</label>
          <AppSelect v-model="selected" :options="warehouses.map(w => ({ value: w.id, label: w.name }))" :placeholder="t('Select a warehouse')" />
        </div>
        <div class="field mb-2">
          <label class="field-label">{{ t('Product *') }}</label>
          <AppSelect v-model="form.product_id" :options="products.map(p => ({ value: p.id, label: p.name }))" :placeholder="t('Select a product')" clearable />
        </div>
        <div class="form-grid">
          <FormField :label="t('Quantity')" required><AppInput v-model.number="form.quantity" type="number" min="1" align="right" /></FormField>
          <FormField :label="t('Reason')" required><AppInput v-model="form.reason" placeholder="e.g. Damaged" /></FormField>
        </div>
        <div class="mt-3"><button class="btn btn-block" type="button" :disabled="saving" @click="submit">{{ saving ? 'Saving…' : 'Record write-off' }}</button></div>
      </div>

      <div class="card">
        <div class="card-head"><div class="card-title">{{ t('Movement history') }}</div></div>
        <div v-if="error" class="alert alert-error mb-3" style="margin:16px">{{ error }}</div>
        <DataTable :columns="columns" :rows="movements" :loading="loading" :empty-text="t('No movements yet')">
          <template #cell-id="{ row }"><span class="subtle mono">#{{ row.id }}</span></template>
          <template #cell-product_id="{ row }">Product #{{ row.product_id }}</template>
          <template #cell-created_at="{ row }"><span class="subtle">{{ row.created_at ? new Date(row.created_at).toLocaleDateString() : '—' }}</span></template>
        </DataTable>
      </div>
    </div>
  </div>
</template>
