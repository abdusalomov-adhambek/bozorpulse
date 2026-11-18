<script setup>
import { t } from '../../i18n'
import { ref, onMounted } from 'vue'
import api from '../../api/client'
import DataTable from '../../components/DataTable.vue'
import Modal from '../../components/Modal.vue'
import { useAppStore } from '../../stores/app'
import { AppInput, FormField } from '../../components/ui'

const app = useAppStore()
const rows = ref([])
const loading = ref(false)
const error = ref('')
const modalOpen = ref(false)
const saving = ref(false)
const name = ref('')

const columns = [
  { key: 'id', label: t('ID') },
  { key: 'name', label: t('Region name') },
  { key: 'created_at', label: t('Created') }
]

async function load() {
  loading.value = true
  error.value = ''
  try {
    const data = await api.get('/regions')
    rows.value = Array.isArray(data) ? data : (data?.data || [])
  } catch (err) {
    error.value = err?.message || 'Failed to load regions.'
  } finally {
    loading.value = false
  }
}

function openCreate() {
  name.value = ''
  modalOpen.value = true
}

async function save() {
  if (!name.value.trim()) { app.error(t('Region name is required.')); return }
  saving.value = true
  try {
    await api.post('/regions', { name: name.value.trim() })
    app.success(t('Region created'))
    modalOpen.value = false
    await load()
  } catch (err) {
    app.error(err?.message || 'Could not create region.')
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t("Regions") }}</h1>
        <p class="page-desc">{{ t('Define the regions used across warehouses and listings.') }}</p>
      </div>
      <button class="btn" type="button" @click="openCreate">{{ t('+ Add region') }}</button>
    </div>

    <div v-if="error" class="alert alert-error mb-3">{{ error }}</div>

    <div class="card">
      <DataTable :columns="columns" :rows="rows" :loading="loading" :empty-text="t('No regions yet')" empty-:desc="t('Create regions to organize your operations.')">
        <template #cell-id="{ row }"><span class="subtle mono">#{{ row.id }}</span></template>
        <template #cell-name="{ row }"><span class="cell-strong">{{ row.name }}</span></template>
        <template #cell-created_at="{ row }"><span class="subtle">{{ row.created_at ? new Date(row.created_at).toLocaleDateString() : '—' }}</span></template>
      </DataTable>
    </div>

    <Modal :open="modalOpen" :title="t('New region')" @close="modalOpen = false">
      <FormField :label="t('Region name')" required>
        <AppInput v-model="name" placeholder="e.g. Tashkent" @enter="save" />
      </FormField>
      <template #footer>
        <button class="btn btn-secondary" type="button" @click="modalOpen = false">{{ t('Cancel') }}</button>
        <button class="btn" type="button" :disabled="saving" @click="save">{{ saving ? 'Saving…' : 'Create region' }}</button>
      </template>
    </Modal>
  </div>
</template>
