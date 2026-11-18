<script setup>
import AppIcon from '../../components/AppIcon.vue'
import { t } from '../../i18n'
import { ref, onMounted } from 'vue'
import api from '../../api/client'
import DataTable from '../../components/DataTable.vue'
import Modal from '../../components/Modal.vue'
import { useAppStore } from '../../stores/app'
import { AppInput, AppSelect, FormField } from '../../components/ui'

const app = useAppStore()
const rows = ref([])
const regions = ref([])
const loading = ref(false)
const error = ref('')
const modalOpen = ref(false)
const saving = ref(false)
const editId = ref(null)
const form = ref(emptyForm())

const columns = [
  { key: 'id', label: t('ID') },
  { key: 'name', label: t('Name') },
  { key: 'city', label: t('City') },
  { key: 'address', label: t('Address') },
  { key: 'region_id', label: t('Region') },
  { key: 'coordinates', label: t('Coordinates') }
]

function emptyForm() {
  return { name: '', address: '', city: '', region_id: '', latitude: '', longitude: '' }
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [whs, regs] = await Promise.all([api.get('/warehouses'), api.get('/regions').catch(() => [])])
    rows.value = Array.isArray(whs) ? whs : (whs?.data || [])
    regions.value = Array.isArray(regs) ? regs : (regs?.data || [])
  } catch (err) {
    error.value = err?.message || 'Failed to load warehouses.'
  } finally {
    loading.value = false
  }
}

function regionName(id) {
  return regions.value.find((r) => r.id === id)?.name || (id ? `#${id}` : '—')
}

function openCreate() {
  editId.value = null
  form.value = emptyForm()
  modalOpen.value = true
}

function openEdit(row) {
  editId.value = row.id
  form.value = {
    name: row.name, address: row.address || '', city: row.city || '',
    region_id: row.region_id || '', latitude: row.latitude ?? '', longitude: row.longitude ?? ''
  }
  modalOpen.value = true
}

async function save() {
  if (!form.value.name.trim()) { app.error(t('Warehouse name is required.')); return }
  saving.value = true
  try {
    const payload = {
      name: form.value.name.trim(),
      address: form.value.address,
      city: form.value.city,
      region_id: form.value.region_id ? Number(form.value.region_id) : null,
      latitude: form.value.latitude === '' ? null : Number(form.value.latitude),
      longitude: form.value.longitude === '' ? null : Number(form.value.longitude)
    }
    if (editId.value) {
      await api.patch(`/warehouses/${editId.value}`, payload)
      app.success(t('Warehouse updated'))
    } else {
      await api.post('/warehouses', payload)
      app.success(t('Warehouse created'))
    }
    modalOpen.value = false
    await load()
  } catch (err) {
    app.error(err?.message || 'Could not save warehouse.')
  } finally {
    saving.value = false
  }
}

async function remove(row) {
  if (!window.confirm(`Delete warehouse "${row.name}"?`)) return
  try {
    await api.del(`/warehouses/${row.id}`)
    app.success(t('Warehouse deleted'))
    await load()
  } catch (err) {
    app.error(err?.message || 'Could not delete warehouse.')
  }
}

onMounted(load)
</script>

<template>
  <div>
    <div class="page-header">
      <div><h1 class="page-title">{{ t("Warehouses") }}</h1><p class="page-desc">{{ t('Manage the locations where your goods are stored.') }}</p></div>
      <button class="btn" type="button" @click="openCreate">+ {{ t('New warehouse') }}</button>
    </div>

    <div v-if="error" class="alert alert-error mb-3">{{ error }}</div>

    <div class="card">
      <DataTable :columns="columns" :rows="rows" :loading="loading" :empty-text="t('No warehouses yet')" empty-:desc="t('Create a warehouse to organize your inventory.')">
        <template #cell-id="{ row }"><span class="subtle mono">#{{ row.id }}</span></template>
        <template #cell-name="{ row }"><span class="cell-strong">{{ row.name }}</span></template>
        <template #cell-region_id="{ row }">{{ regionName(row.region_id) }}</template>
        <template #cell-coordinates="{ row }">
          <span class="mono subtle">{{ row.latitude != null && row.longitude != null ? `${row.latitude}, ${row.longitude}` : '—' }}</span>
        </template>
        <template #actions="{ row }">
          <button class="icon-btn btn-sm" type="button" :aria-label="t('Edit')" :title="t('Edit')" @click="openEdit(row)"><AppIcon name="edit" :size="15" /></button>
          <button class="icon-btn btn-sm danger" type="button" :aria-label="t('Delete')" :title="t('Delete')" @click="remove(row)"><AppIcon name="trash" :size="15" /></button>
        </template>
      </DataTable>
    </div>

    <Modal :open="modalOpen" :title="editId ? t('Edit warehouse') : t('New warehouse')" @close="modalOpen = false" wide>
      <div class="form-grid">
        <FormField :label="t('Warehouse name')" required>
          <AppInput v-model="form.name" placeholder="e.g. Central Depot" />
        </FormField>
        <FormField :label="t('City')">
          <AppInput v-model="form.city" placeholder="e.g. Tashkent" />
        </FormField>
        <FormField :label="t('Region')">
          <AppSelect v-model="form.region_id" :options="regions" value-key="id" label-key="name" :placeholder="t('None')" clearable />
        </FormField>
        <FormField :label="t('Address')">
          <AppInput v-model="form.address" :placeholder="t('Street and building')" />
        </FormField>
        <FormField :label="t('Latitude')">
          <AppInput v-model.number="form.latitude" type="number" step="any" placeholder="41.2995" align="right" />
        </FormField>
        <FormField :label="t('Longitude')">
          <AppInput v-model.number="form.longitude" type="number" step="any" placeholder="69.2401" align="right" />
        </FormField>
      </div>
      <template #footer>
        <button class="btn btn-secondary" type="button" @click="modalOpen = false">{{ t('Cancel') }}</button>
        <button class="btn" type="button" :disabled="saving" @click="save">{{ saving ? 'Saving…' : 'Save warehouse' }}</button>
      </template>
    </Modal>
  </div>
</template>
