<script setup>
import AppIcon from '../../components/AppIcon.vue'
import { money } from '../../utils/money'
import { t } from '../../i18n'
import { ref, onMounted } from 'vue'
import api from '../../api/client'
import DataTable from '../../components/DataTable.vue'
import Modal from '../../components/Modal.vue'
import { useAppStore } from '../../stores/app'
import { AppInput, AppSelect, AppTextarea, FormField, NumberInput, AppFileInput } from '../../components/ui'

const app = useAppStore()
const rows = ref([])
const categories = ref([])
const warehouses = ref([])
const companyId = ref('')
const loading = ref(false)
const error = ref('')
const modalOpen = ref(false)
const saving = ref(false)
const editId = ref(null)
const form = ref(emptyForm())
const photoUrl = ref('')

const pricingOpen = ref(false)
const pricingProduct = ref(null)
const rules = ref([])
const ruleForm = ref(emptyRule())
const savingRule = ref(false)

const columns = [
  { key: 'name', label: t('Product') },
  { key: 'sku', label: t('SKU') },
  { key: 'category', label: t('Category') },
  { key: 'price', label: t('Price') },
  { key: 'effective_price', label: t('Effective price') },
  { key: 'delivery', label: t('Delivery') },
  { key: 'stock', label: t('Stock') }
]

function emptyForm() {
  return {
    name: '', sku: '', description: '', unit: '', price: 0, discount_percent: 0, stock: 0,
    category_id: '', photos: [], min_order_qty: 1, max_order_qty: 0,
    delivery_min_days: 3, delivery_max_days: 7
  }
}

function emptyRule() {
  return {
    warehouse_id: '', min_qty: 1, max_qty: 0, unit_price: 0, discount_percent: 0,
    delivery_min_days: 3, delivery_max_days: 7, min_order: 1, max_order: 0
  }
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [company, products, cats, whs] = await Promise.all([
      api.get('/companies/me'),
      api.get('/products', { limit: 100 }),
      api.get('/categories').catch(() => ({ data: [] })),
      api.get('/warehouses').catch(() => ({ data: [] })),
    ])
    companyId.value = company?.id || ''
    const all = Array.isArray(products) ? products : (products?.data || [])
    rows.value = all.filter((p) => !companyId.value || p.supplier_id === companyId.value)
    categories.value = Array.isArray(cats) ? cats : (cats?.data || [])
    warehouses.value = Array.isArray(whs) ? whs : (whs?.data || [])
  } catch (err) {
    error.value = err?.message || 'Failed to load products.'
  } finally {
    loading.value = false
  }
}

function categoryName(id) {
  return categories.value.find((c) => c.id === id)?.name || (id ? `#${id}` : '—')
}

function deliveryLabel(row) {
  const min = row.delivery_min_days || 0
  const max = row.delivery_max_days || 0
  if (!min && !max) return '—'
  return min === max ? `${min} day(s)` : `${min}–${max} days`
}

function openCreate() {
  editId.value = null
  form.value = emptyForm()
  photoUrl.value = ''
  modalOpen.value = true
}

function openEdit(row) {
  editId.value = row.id
  form.value = {
    name: row.name, sku: row.sku || '', description: row.description || '', unit: row.unit || '',
    price: row.price, discount_percent: row.discount_percent || 0, stock: row.stock || 0,
    category_id: row.category_id || '', photos: [...(row.photos || [])],
    min_order_qty: row.min_order_qty || 1, max_order_qty: row.max_order_qty || 0,
    delivery_min_days: row.delivery_min_days || 3, delivery_max_days: row.delivery_max_days || 7
  }
  photoUrl.value = ''
  modalOpen.value = true
}

function addPhoto() {
  const url = photoUrl.value.trim()
  if (!url) return
  if (form.value.photos.length >= 6) { app.error(t('Maximum 6 photos per product.')); return }
  form.value.photos.push(url)
  photoUrl.value = ''
}

function onPhotoFile(event) {
  const file = event.target.files?.[0]
  if (!file) return
  if (file.size > 2 * 1024 * 1024) { app.error(t('Image must be smaller than 2 MB.')); return }
  const reader = new FileReader()
  reader.onload = () => {
    if (form.value.photos.length >= 6) { app.error(t('Maximum 6 photos per product.')); return }
    form.value.photos.push(String(reader.result))
    event.target.value = ''
  }
  reader.readAsDataURL(file)
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
      stock: Number(form.value.stock),
      category_id: form.value.category_id || undefined,
      photos: form.value.photos,
      min_order_qty: Number(form.value.min_order_qty),
      max_order_qty: Number(form.value.max_order_qty),
      delivery_min_days: Number(form.value.delivery_min_days),
      delivery_max_days: Number(form.value.delivery_max_days)
    }
    if (editId.value) await api.patch(`/products/${editId.value}`, payload)
    else await api.post('/products', payload)
    app.success(editId.value ? 'Product updated' : 'Product created')
    modalOpen.value = false
    await load()
  } catch (err) {
    app.error(err?.message || 'Could not save product.')
  } finally {
    saving.value = false
  }
}

async function updateDiscount(row, event) {
  const value = Number(event.target.value)
  if (Number.isNaN(value) || value < 0 || value > 90) { app.error(t('Discount must be between 0 and 90.')); return }
  try {
    const updated = await api.post(`/products/${row.id}/discount`, { discount_percent: value })
    Object.assign(row, updated)
    app.success(`Discount updated for ${row.name}`)
  } catch (err) {
    app.error(err?.message || 'Could not update discount.')
  }
}

async function openPricing(row) {
  pricingProduct.value = row
  ruleForm.value = emptyRule()
  try {
    const res = await api.get(`/products/${row.id}/pricing`)
    rules.value = Array.isArray(res) ? res : (res?.data || [])
  } catch {
    rules.value = []
  }
  pricingOpen.value = true
}

function warehouseName(id) {
  return id ? (warehouses.value.find((w) => w.id === id)?.name || `#${id}`) : 'All warehouses'
}

async function addRule() {
  if (Number(ruleForm.value.unit_price) <= 0) { app.error(t('Enter a price.')); return }
  savingRule.value = true
  try {
    const rule = await api.post(`/products/${pricingProduct.value.id}/pricing`, {
      ...ruleForm.value,
      unit_price: Number(ruleForm.value.unit_price),
      min_qty: Number(ruleForm.value.min_qty),
      max_qty: Number(ruleForm.value.max_qty),
      discount_percent: Number(ruleForm.value.discount_percent),
      delivery_min_days: Number(ruleForm.value.delivery_min_days),
      delivery_max_days: Number(ruleForm.value.delivery_max_days),
      min_order: Number(ruleForm.value.min_order),
      max_order: Number(ruleForm.value.max_order)
    })
    rules.value.push(rule)
    ruleForm.value = emptyRule()
    app.success(t('Pricing rule added'))
  } catch (err) {
    app.error(err?.message || 'Could not add pricing rule.')
  } finally {
    savingRule.value = false
  }
}

async function removeRule(rule) {
  try {
    await api.del(`/products/${pricingProduct.value.id}/pricing/${rule.id}`)
    rules.value = rules.value.filter((r) => r.id !== rule.id)
    app.success(t('Pricing rule removed'))
  } catch (err) {
    app.error(err?.message || 'Could not remove pricing rule.')
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
      <div><h1 class="page-title">{{ t("Products") }}</h1><p class="page-desc">{{ t('Manage your catalog, pricing, delivery windows and photos.') }}</p></div>
      <button class="btn" type="button" @click="openCreate">{{ t('+ Add product') }}</button>
    </div>

    <div v-if="error" class="alert alert-error mb-3">{{ error }}</div>

    <div class="card">
      <DataTable :columns="columns" :rows="rows" :loading="loading" :empty-text="t('No products yet')">
        <template #cell-name="{ row }">
          <div class="product-cell">
            <img v-if="row.photos?.length" :src="row.photos[0]" class="thumb" alt="" />
            <span class="cell-strong">{{ row.name }}</span>
          </div>
        </template>
        <template #cell-category="{ row }"><span class="subtle">{{ categoryName(row.category_id) }}</span></template>
        <template #cell-price="{ row }"><span class="mono">{{ money(row.price || 0) }}</span></template>
        <template #cell-effective_price="{ row }"><span class="cell-strong mono text-primary">{{ money((row.effective_price ?? row.price) || 0) }}</span></template>
        <template #cell-delivery="{ row }"><span class="subtle">{{ deliveryLabel(row) }}</span></template>
        <template #cell-stock="{ row }"><span class="mono">{{ row.stock }}</span></template>
        <template #actions="{ row }">
          <button class="btn btn-sm btn-secondary" type="button" @click="openPricing(row)">{{ t('Price rules') }}</button>
          <button class="icon-btn btn-sm" type="button" :aria-label="t('Edit')" :title="t('Edit')" @click="openEdit(row)"><AppIcon name="edit" :size="15" /></button>
          <button class="icon-btn btn-sm danger" type="button" :aria-label="t('Delete')" :title="t('Delete')" @click="remove(row)"><AppIcon name="trash" :size="15" /></button>
        </template>
      </DataTable>
    </div>

    <Modal :open="modalOpen" :title="editId ? t('Edit product') : t('New product')" @close="modalOpen = false" wide>
      <div class="form-grid">
        <FormField :label="t('Product name')" required><AppInput v-model="form.name" :placeholder="t('Product name')" /></FormField>
        <FormField :label="t('SKU')"><AppInput v-model="form.sku" placeholder="SKU-001" /></FormField>
        <FormField :label="t('Category')">
          <AppSelect v-model="form.category_id" :options="categories" value-key="id" label-key="name" :placeholder="t('None')" clearable />
        </FormField>
        <FormField :label="t('Unit')"><AppInput v-model="form.unit" placeholder="kg, pcs, L…" /></FormField>
        <FormField :label="t('Base price')" required><AppInput v-model.number="form.price" type="number" min="0" step="0.01" align="right" /></FormField>
        <FormField :label="t('Discount %')"><AppInput v-model.number="form.discount_percent" type="number" min="0" max="90" align="right" /></FormField>
        <FormField :label="t('Stock')"><NumberInput v-model="form.stock" :min="0" /></FormField>
        <FormField :label="t('Min order qty')"><NumberInput v-model="form.min_order_qty" :min="1" /></FormField>
        <FormField :label="t('Max order qty')" hint="0 = unlimited"><NumberInput v-model="form.max_order_qty" :min="0" /></FormField>
        <FormField :label="t('Delivery min (days)')"><NumberInput v-model="form.delivery_min_days" :min="0" /></FormField>
        <FormField :label="t('Delivery max (days)')"><NumberInput v-model="form.delivery_max_days" :min="0" /></FormField>
        <FormField :label="t('Description')">
          <AppTextarea v-model="form.description" :rows="2" :placeholder="t('Short description')" />
        </FormField>
      </div>

      <div class="photos-section">
        <label class="field-label">{{ t('Product photos (up to 6)') }}</label>
        <div class="photo-add">
          <AppInput v-model="photoUrl" :placeholder="t('Paste image URL')" @enter="addPhoto" />
          <button class="btn btn-secondary" type="button" @click="addPhoto">{{ t('Add URL') }}</button>
          <AppFileInput class="btn btn-secondary" :label="t('Upload image')" @change="onPhotoFile" />
        </div>
        <div v-if="form.photos.length" class="photo-list">
          <div v-for="(photo, i) in form.photos" :key="i" class="photo-item">
            <img :src="photo" alt="Product photo" />
            <button type="button" :aria-label="t('Remove photo')" @click="form.photos.splice(i, 1)">×</button>
          </div>
        </div>
        <p v-else class="subtle">{{ t('No photos yet. Add a URL or upload an image.') }}</p>
      </div>

      <template #footer>
        <button class="btn btn-secondary" type="button" @click="modalOpen = false">{{ t('Cancel') }}</button>
        <button class="btn" type="button" :disabled="saving" @click="save">{{ saving ? 'Saving…' : 'Save product' }}</button>
      </template>
    </Modal>

    <Modal :open="pricingOpen" :title="`Price rules — ${pricingProduct?.name || ''}`" @close="pricingOpen = false" wide>
      <div v-if="rules.length" class="card rule-list">
        <div v-for="rule in rules" :key="rule.id" class="rule-row">
          <div><strong>{{ warehouseName(rule.warehouse_id) }}</strong><small>{{ rule.min_qty }}–{{ rule.max_qty || '∞' }} pcs</small></div>
          <span class="mono">{{ money(rule.unit_price) }}</span>
          <span v-if="rule.discount_percent">−{{ rule.discount_percent }}%</span>
          <span class="text-primary mono">{{ money(rule.effective_price) }}</span>
          <span class="subtle">{{ rule.delivery_min_days }}–{{ rule.delivery_max_days }}d</span>
          <button class="icon-btn btn-sm danger" type="button" :aria-label="t('Delete')" :title="t('Delete')" @click="removeRule(rule)"><AppIcon name="trash" :size="15" /></button>
        </div>
      </div>
      <p v-else class="subtle mb-3">{{ t('No rules yet. The base product price applies everywhere.') }}</p>

      <div class="rule-form">
        <FormField :label="t('Warehouse')">
          <AppSelect v-model="ruleForm.warehouse_id" :options="[{ value: '', label: 'All warehouses' }, ...warehouses.map(w => ({ value: w.id, label: w.name }))]" :placeholder="t('All warehouses')" clearable />
        </FormField>
        <FormField :label="t('Min qty')"><NumberInput v-model="ruleForm.min_qty" :min="1" /></FormField>
        <FormField :label="t('Max qty')" hint="0 = unlimited"><NumberInput v-model="ruleForm.max_qty" :min="0" /></FormField>
        <FormField :label="t('Price')"><AppInput v-model.number="ruleForm.unit_price" type="number" min="0" step="0.01" align="right" /></FormField>
        <FormField :label="t('Discount %')"><AppInput v-model.number="ruleForm.discount_percent" type="number" min="0" max="90" align="right" /></FormField>
        <FormField :label="t('Min order')"><NumberInput v-model="ruleForm.min_order" :min="1" /></FormField>
        <FormField :label="t('Max order')"><NumberInput v-model="ruleForm.max_order" :min="0" /></FormField>
        <FormField :label="t('Delivery min')"><NumberInput v-model="ruleForm.delivery_min_days" :min="0" /></FormField>
        <FormField :label="t('Delivery max')"><NumberInput v-model="ruleForm.delivery_max_days" :min="0" /></FormField>
      </div>
      <template #footer>
        <button class="btn btn-secondary" type="button" @click="pricingOpen = false">{{ t('Cancel') }}</button>
        <button class="btn" type="button" :disabled="savingRule" @click="addRule">{{ savingRule ? 'Adding…' : 'Add rule' }}</button>
      </template>
    </Modal>
  </div>
</template>

<style scoped>
.product-cell { display: flex; align-items: center; gap: 9px; }
.thumb { width: 32px; height: 32px; border-radius: 6px; object-fit: cover; border: 1px solid var(--border); }
.photos-section { margin-top: 22px; padding-top: 18px; border-top: 1px solid var(--border); }
.photo-add { display: grid; grid-template-columns: 1fr auto auto; gap: 8px; margin: 8px 0 12px; }
.upload-btn { position: relative; overflow: hidden; display: inline-flex; align-items: center; }
.upload-btn input { position: absolute; inset: 0; opacity: 0; cursor: pointer; }
.photo-list { display: flex; flex-wrap: wrap; gap: 8px; }
.photo-item { position: relative; width: 64px; height: 64px; }
.photo-item img { width: 100%; height: 100%; object-fit: cover; border-radius: 8px; border: 1px solid var(--border); }
.photo-item button { position: absolute; top: -6px; right: -6px; width: 18px; height: 18px; border: 0; border-radius: 50%; background: var(--danger); color: #fff; font-size: 12px; line-height: 1; cursor: pointer; }
.rule-list { margin-bottom: 16px; }
.rule-row { display: grid; grid-template-columns: 1.6fr .8fr .6fr .8fr .8fr auto; align-items: center; gap: 10px; padding: 10px 14px; border-bottom: 1px solid var(--border); font-size: 12.5px; }
.rule-row:last-child { border-bottom: 0; }
.rule-row div { display: flex; flex-direction: column; gap: 2px; }
.rule-row small { color: var(--text-3); }
.rule-form { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; }
@media (max-width: 860px) {
  .photo-add { grid-template-columns: 1fr; }
  .rule-form { grid-template-columns: 1fr 1fr; }
  .rule-row { grid-template-columns: 1fr 1fr; }
}
</style>
