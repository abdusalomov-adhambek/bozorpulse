<script setup>
import { t } from '../../i18n'
import { computed, ref, onMounted, watch } from 'vue'
import api from '../../api/client'
import DataTable from '../../components/DataTable.vue'
import EmptyState from '../../components/EmptyState.vue'
import { useAppStore } from '../../stores/app'
import { AppInput, AppSelect, FormField } from '../../components/ui'

const app = useAppStore()
const warehouses = ref([])
const warehouseOptions = computed(() => warehouses.value.map((w) => ({ value: w.id, label: `${w.name} — ${w.city || w.address || ''}` })))
const selected = ref('')
const rows = ref([])
const loading = ref(false)
const error = ref('')

const tabs = [
  { label: t('Overview'), to: '/customer/warehouses' },
  { label: t('Stock'), to: '/customer/warehouses/stock' },
  { label: t('Transfers'), to: '/customer/warehouses/transfers' },
  { label: t('Supplies'), to: '/customer/warehouses/supplies' },
  { label: t('Write-off'), to: '/customer/warehouses/writeoffs' },
  { label: t('Returns'), to: '/customer/warehouses/returns' }
]
const columns = [
  { key: 'product_id', label: t('Product ID') },
  { key: 'product_name', label: t('Product') },
  { key: 'quantity', label: t('Quantity') },
  { key: 'updated_at', label: t('Updated') }
]

async function loadWarehouses() {
  try {
    const data = await api.get('/warehouses')
    warehouses.value = Array.isArray(data) ? data : (data?.data || [])
    if (!selected.value && warehouses.value.length) selected.value = warehouses.value[0].id
  } catch (err) {
    app.error(err?.message || 'Could not load warehouses.')
  }
}

async function loadStock() {
  if (!selected.value) { rows.value = []; return }
  loading.value = true
  error.value = ''
  try {
    const data = await api.get('/inventory/stock', { warehouse_id: selected.value })
    rows.value = Array.isArray(data) ? data : (data?.data || [])
  } catch (err) {
    error.value = err?.message || 'Failed to load stock.'
  } finally {
    loading.value = false
  }
}

watch(selected, loadStock)
onMounted(async () => { await loadWarehouses(); loadStock() })
</script>

<template>
  <div>
    <div class="page-header">
      <div><h1 class="page-title">{{ t("Warehouses") }}</h1><p class="page-desc">{{ t('Current inventory levels per warehouse.') }}</p></div>
    </div>

    <div class="tabs mb-3">
      <RouterLink v-for="tab in tabs" :key="tab.to" :to="tab.to" class="tab tab-sub" :class="{ active: $route.path === tab.to }">{{ tab.label }}</RouterLink>
    </div>

    <div class="card">
      <div class="card-head">
        <div class="card-title">{{ t('Warehouse') }}</div>
        <AppSelect v-model="selected" :options="warehouseOptions" :placeholder="t('Select a warehouse')" class="warehouse-select" />
      </div>
      <div v-if="error" class="alert alert-error mb-3" style="margin:16px">{{ error }}</div>
      <DataTable :columns="columns" :rows="rows" :loading="loading" :empty-text="t('No stock recorded')" empty-:desc="t('Add a supply to this warehouse to see stock here.')">
        <template #cell-product_id="{ row }"><span class="subtle mono">#{{ row.product_id }}</span></template>
        <template #cell-product_name="{ row }"><span class="cell-strong">{{ row.product_name || `Product #${row.product_id}` }}</span></template>
        <template #cell-quantity="{ row }"><span class="cell-strong mono">{{ row.quantity }}</span></template>
        <template #cell-updated_at="{ row }"><span class="subtle">{{ row.updated_at ? new Date(row.updated_at).toLocaleString() : '—' }}</span></template>
      </DataTable>
      <div v-if="!warehouses.length" style="padding:0 22px 22px">
        <EmptyState :title="t('No warehouses')" description="Create a warehouse before viewing stock." icon="warehouse">
          <RouterLink to="/customer/warehouses" class="btn btn-sm">{{ t('Go to warehouses') }}</RouterLink>
        </EmptyState>
      </div>
    </div>
  </div>
</template>

<style scoped>
.warehouse-select { width: 280px; max-width: 100%; }
</style>
