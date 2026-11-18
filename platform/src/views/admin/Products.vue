<script setup>
import AppIcon from '../../components/AppIcon.vue'
import { money } from '../../utils/money'
import { t } from '../../i18n'
import { ref, onMounted } from 'vue'
import api from '../../api/client'
import DataTable from '../../components/DataTable.vue'
import Modal from '../../components/Modal.vue'
import { useAppStore } from '../../stores/app'
import { AppInput, AppSelect, AppTextarea, FormField, NumberInput } from '../../components/ui'

const app = useAppStore()
const rows = ref([])
const categories = ref([])
const suppliers = ref([])
const loading = ref(false)
const error = ref('')
const modalOpen = ref(false)
const saving = ref(false)
const editing = ref(null)
const form = ref(emptyForm())

const columns = [
  { key: 'id', label: t('ID') },
  { key: 'name', label: t('Product') },
  { key: 'sku', label: t('SKU') },
  { key: 'category', label: t('Category') },
  { key: 'price', label: t('Price') },
  { key: 'discount_percent', label: t('Discount') },
  { key: 'effective_price', label: t('Effective') },
  { key: 'stock', label: t('Stock') }
]

function emptyForm() {
  return { name: '', sku: '', description: '', unit: '', price: 0, discount_percent: 0, stock: 0, category_id: '', supplier_id: '', region_id: '' }
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [products, cats, sups] = await Promise.all([
      api.get('/products'),
      api.get('/categories').catch(() => []),
      api.get('/suppliers').catch(() => [])
    ])
    rows.value = Array.isArray(products) ? products : (products?.data || [])
    categories.value = Array.isArray(cats) ? cats : (cats?.data || [])
    suppliers.value = Array.isArray(sups) ? sups : (sups?.data || [])
  } catch (err) {
    error.value = err?.message || 'Failed to load products.'
  } finally {
    loading.value = false
  }
}

function categoryName(id) {
  return categories.value.find((c) => c.id === id)?.name || (id ? `#${id}` : '—')
}

function openCreate() {
  editing.value = null
  form.value = emptyForm()
  modalOpen.value = true
}

function openEdit(row) {
  editing.value = row
  form.value = {
    name: row.name, sku: row.sku || '', description: row.description || '', unit: row.unit || '',
    price: row.price, discount_percent: row.discount_percent || 0, stock: row.stock || 0,
    category_id: row.category_id || '', supplier_id: row.supplier_id || '', region_id: row.region_id || ''
  }
  modalOpen.value = true
}

async function save() {
  if (!form.value.name.trim()) { app.error(t('Product name is required.')); return }
  saving.value = true
  try {
    const payload = {
      name: form.value.name.trim(),
      sku: form.value.sku,
      description: form.value.description,
      unit: form.value.unit,
      price: Number(form.value.price),
      discount_percent: Number(form.value.discount_percent),
      stock: Number(form.value.stock)
    }
    if (form.value.category_id) payload.category_id = Number(form.value.category_id)
    if (form.value.supplier_id) payload.supplier_id = Number(form.value.supplier_id)
    if (form.value.region_id) payload.region_id = Number(form.value.region_id)
    if (editing.value) {
      await api.patch(`/products/${editing.value.id}`, payload)
      app.success(t('Product updated'))
    } else {
      await api.post('/products', payload)
      app.success(t('Product created'))
    }
    modalOpen.value = false
    await load()
  } catch (err) {
    app.error(err?.message || 'Could not save product.')
  } finally {
    saving.value = false
  }
}

async function updateDiscount(row, value) {
  value = Number(value)
  if (Number.isNaN(value) || value < 0 || value > 100) { app.error(t('Discount must be between 0 and 100.')); return }
  try {
    const updated = await api.post(`/products/${row.id}/discount`, { discount_percent: value })
    Object.assign(row, updated)
    app.success(`Discount updated for ${row.name}`)
  } catch (err) {
    app.error(err?.message || 'Could not update discount.')
  }
}

async function remove(row) {
  if (!window.confirm(`Delete product "${row.name}"?`)) return
  try {
    await api.del(`/products/${row.id}`)
    app.success(t('Product deleted'))
    await load()
  } catch (err) {
    app.error(err?.message || 'Could not delete product.')
  }
}

onMounted(load)
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t("Products") }}</h1>
        <p class="page-desc">{{ t('Manage the product catalog across all suppliers.') }}</p>
      </div>
      <button class="btn" type="button" @click="openCreate">{{ t('+ Add product') }}</button>
    </div>

    <div v-if="error" class="alert alert-error mb-3">{{ error }}</div>

    <div class="card">
      <DataTable :columns="columns" :rows="rows" :loading="loading" :empty-text="t('No products yet')" empty-:desc="t('Add a product to populate the catalog.')">
        <template #cell-id="{ row }"><span class="subtle mono">#{{ row.id }}</span></template>
        <template #cell-name="{ row }"><span class="cell-strong">{{ row.name }}</span></template>
        <template #cell-category="{ row }">{{ categoryName(row.category_id) }}</template>
        <template #cell-price="{ row }"><span class="mono">{{ money(row.price || 0) }}</span></template>
        <template #cell-discount_percent="{ row }">
          <NumberInput class="inline-discount" :model-value="Number(row.discount_percent || 0)" :min="0" :max="90" size="sm" @update:model-value="updateDiscount(row, $event)" />
        </template>
        <template #cell-effective_price="{ row }"><span class="cell-strong mono text-primary">{{ money((row.effective_price ?? row.price) || 0) }}</span></template>
        <template #cell-stock="{ row }"><span class="mono">{{ row.stock ?? '—' }}</span></template>
        <template #actions="{ row }">
          <button class="icon-btn btn-sm" type="button" :aria-label="t('Edit')" :title="t('Edit')" @click="openEdit(row)"><AppIcon name="edit" :size="15" /></button>
          <button class="icon-btn btn-sm danger" type="button" :aria-label="t('Delete')" :title="t('Delete')" @click="remove(row)"><AppIcon name="trash" :size="15" /></button>
        </template>
      </DataTable>
    </div>

    <Modal :open="modalOpen" :title="editing ? t('Edit product') : t('New product')" @close="modalOpen = false" wide>
      <div class="form-grid">
        <FormField :label="t('Product name')" required>
          <AppInput v-model="form.name" :placeholder="t('Product name')" />
        </FormField>
        <FormField :label="t('SKU')">
          <AppInput v-model="form.sku" placeholder="SKU-001" />
        </FormField>
        <FormField :label="t('Category')">
          <AppSelect v-model="form.category_id" :options="categories" value-key="id" label-key="name" :placeholder="t('None')" clearable />
        </FormField>
        <FormField :label="t('Supplier')">
          <AppSelect v-model="form.supplier_id" :options="suppliers" value-key="id" label-key="company_name" :placeholder="t('None')" clearable />
        </FormField>
        <FormField :label="t('Price')" required>
          <AppInput v-model.number="form.price" type="number" min="0" step="0.01" align="right" />
        </FormField>
        <FormField :label="t('Discount %')">
          <AppInput v-model.number="form.discount_percent" type="number" min="0" max="90" align="right" />
        </FormField>
        <FormField :label="t('Stock')">
          <AppInput v-model.number="form.stock" type="number" min="0" align="right" />
        </FormField>
        <FormField :label="t('Unit')">
          <AppInput v-model="form.unit" placeholder="kg, pcs, L…" />
        </FormField>
        <FormField :label="t('Description')">
          <AppTextarea v-model="form.description" :rows="2" :placeholder="t('Short description')" />
        </FormField>
      </div>
      <template #footer>
        <button class="btn btn-secondary" type="button" @click="modalOpen = false">{{ t('Cancel') }}</button>
        <button class="btn" type="button" :disabled="saving" @click="save">{{ saving ? 'Saving…' : 'Save product' }}</button>
      </template>
    </Modal>
  </div>
</template>

<style scoped>
.inline-discount { width: 96px; }
</style>
