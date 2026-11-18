<script setup>
import AppIcon from '../../components/AppIcon.vue'
import { computed, onMounted, ref } from 'vue'
import api from '../../api/client'
import { money } from '../../utils/money'
import { t } from '../../i18n'
import { useAppStore } from '../../stores/app'
import DataTable from '../../components/DataTable.vue'
import Modal from '../../components/Modal.vue'
import { AppInput, AppSelect, AppTextarea, AppCheckbox, NumberInput, FormField } from '../../components/ui'

const app = useAppStore()
const zones = ref([])
const regions = ref([])
const loading = ref(false)
const modalOpen = ref(false)
const saving = ref(false)
const editId = ref(null)
const form = ref(emptyForm())

const columns = [
  { key: 'region_name', label: t('Region') },
  { key: 'available', label: t('Available') },
  { key: 'price', label: t('Delivery price') },
  { key: 'period', label: t('Delivery period') },
  { key: 'notes', label: t('Notes') },
]

const regionOptions = computed(() => regions.value.map((r) => ({ value: r.id, label: r.name })))

function emptyForm() {
  return { region_id: '', available: true, price: 0, min_days: 3, max_days: 7, notes: '' }
}

async function load() {
  loading.value = true
  try {
    const [zoneRes, regionRes] = await Promise.all([
      api.get('/suppliers/me/delivery-zones'),
      api.get('/regions', {}, { auth: false }),
    ])
    zones.value = Array.isArray(zoneRes) ? zoneRes : (zoneRes?.data || [])
    regions.value = Array.isArray(regionRes) ? regionRes : (regionRes?.data || [])
  } catch (err) {
    app.error(err?.message || 'Could not load delivery zones.')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editId.value = null
  form.value = emptyForm()
  modalOpen.value = true
}
function openEdit(zone) {
  editId.value = zone.id
  form.value = { ...zone }
  modalOpen.value = true
}

async function save() {
  if (!form.value.region_id) { app.error(t('Select a region.')); return }
  saving.value = true
  try {
    const payload = {
      region_id: form.value.region_id,
      available: !!form.value.available,
      price: Number(form.value.price),
      min_days: Number(form.value.min_days),
      max_days: Number(form.value.max_days),
      notes: form.value.notes || '',
    }
    if (editId.value) await api.patch(`/suppliers/me/delivery-zones/${editId.value}`, payload)
    else await api.post('/suppliers/me/delivery-zones', payload)
    app.success(editId.value ? 'Delivery zone updated' : 'Delivery zone added')
    modalOpen.value = false
    await load()
  } catch (err) {
    app.error(err?.message || 'Could not save delivery zone.')
  } finally {
    saving.value = false
  }
}

async function remove(zone) {
  if (!window.confirm(`Remove delivery to ${zone.region_name || zone.region_id}?`)) return
  try {
    await api.del(`/suppliers/me/delivery-zones/${zone.id}`)
    app.success(t('Delivery zone removed'))
    await load()
  } catch (err) {
    app.error(err?.message || 'Could not remove delivery zone.')
  }
}

function period(zone) {
  if (!zone.min_days && !zone.max_days) return '—'
  return zone.min_days === zone.max_days ? `${zone.min_days} day(s)` : `${zone.min_days}–${zone.max_days} days`
}

onMounted(load)
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t('Delivery') }}</h1>
        <p class="page-desc">{{ t('Set delivery availability, price and period for each region.') }}</p>
      </div>
      <button class="btn" type="button" @click="openCreate">{{ t('+ Add region') }}</button>
    </div>

    <div class="card">
      <DataTable :columns="columns" :rows="zones" :loading="loading" :empty-text="t('No delivery zones yet')">
        <template #cell-region_name="{ row }"><span class="cell-strong">{{ row.region_name || row.region_id }}</span></template>
        <template #cell-available="{ row }">
          <span class="badge" :class="row.available ? 'badge-green' : 'badge-red'">{{ row.available ? 'Yes' : 'No' }}</span>
        </template>
        <template #cell-price="{ row }"><span class="mono">{{ row.available ? money(row.price) : '—' }}</span></template>
        <template #cell-period="{ row }"><span>{{ row.available ? period(row) : '—' }}</span></template>
        <template #cell-notes="{ row }"><span class="subtle">{{ row.notes || '—' }}</span></template>
        <template #actions="{ row }">
          <button class="icon-btn btn-sm" type="button" :aria-label="t('Edit')" :title="t('Edit')" @click="openEdit(row)"><AppIcon name="edit" :size="15" /></button>
          <button class="icon-btn btn-sm danger" type="button" :aria-label="t('Delete')" :title="t('Delete')" @click="remove(row)"><AppIcon name="trash" :size="15" /></button>
        </template>
      </DataTable>
    </div>

    <Modal :open="modalOpen" :title="editId ? t('Edit delivery zone') : t('Add delivery zone')" @close="modalOpen = false">
      <FormField :label="t('Region')" required>
        <AppSelect v-model="form.region_id" :options="regionOptions" :placeholder="t('Select a region')" />
      </FormField>
      <AppCheckbox v-model="form.available" :label="t('Delivery available')" class="mt-2" />
      <div class="form-grid mt-2">
        <FormField :label="t('Delivery price')">
          <AppInput v-model.number="form.price" type="number" min="0" step="0.01" align="right" />
        </FormField>
        <FormField :label="t('Min days')">
          <NumberInput v-model="form.min_days" :min="0" />
        </FormField>
        <FormField :label="t('Max days')">
          <NumberInput v-model="form.max_days" :min="0" />
        </FormField>
      </div>
      <FormField :label="t('Notes')" class="mt-2">
        <AppTextarea v-model="form.notes" :rows="2" :placeholder="t('Optional notes')" />
      </FormField>
      <template #footer>
        <button class="btn btn-secondary" type="button" @click="modalOpen = false">{{ t('Cancel') }}</button>
        <button class="btn" type="button" :disabled="saving" @click="save">{{ saving ? 'Saving…' : 'Save' }}</button>
      </template>
    </Modal>
  </div>
</template>
