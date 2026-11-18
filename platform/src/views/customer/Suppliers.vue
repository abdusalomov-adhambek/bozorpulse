<script setup>
import AppIcon from '../../components/AppIcon.vue'
import { t } from '../../i18n'
import { ref, onMounted } from 'vue'
import api from '../../api/client'
import DataTable from '../../components/DataTable.vue'
import Modal from '../../components/Modal.vue'
import { useAppStore } from '../../stores/app'

const app = useAppStore()
const rows = ref([])
const loading = ref(false)
const error = ref('')
const detailOpen = ref(false)
const selected = ref(null)
const detailLoading = ref(false)

const columns = [
  { key: 'id', label: t('ID') },
  { key: 'username', label: t('Supplier') },
  { key: 'company_name', label: t('Company') },
  { key: 'region_id', label: t('Region') },
  { key: 'phone', label: t('Phone') }
]

async function load() {
  loading.value = true
  error.value = ''
  try {
    const data = await api.get('/suppliers')
    rows.value = Array.isArray(data) ? data : (data?.data || [])
  } catch (err) {
    error.value = err?.message || 'Failed to load suppliers.'
  } finally {
    loading.value = false
  }
}

async function openDetail(supplier) {
  selected.value = supplier
  detailOpen.value = true
  detailLoading.value = true
  try {
    selected.value = await api.get(`/suppliers/${supplier.id}`)
  } catch (err) {
    app.error(err?.message || 'Could not load supplier details.')
  } finally {
    detailLoading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t("Suppliers") }}</h1>
        <p class="page-desc">{{ t('Browse verified suppliers and their companies.') }}</p>
      </div>
      <button class="btn btn-secondary" type="button" :disabled="loading" @click="load">{{ t("Refresh") }}</button>
    </div>

    <div v-if="error" class="alert alert-error mb-3">{{ error }}</div>

    <div class="card">
      <DataTable :columns="columns" :rows="rows" :loading="loading" :empty-text="t('No suppliers yet')" empty-:desc="t('Approved suppliers will appear here.')">
        <template #cell-id="{ row }"><span class="subtle mono">#{{ row.id }}</span></template>
        <template #cell-username="{ row }"><span class="cell-strong">{{ row.username }}</span></template>
        <template #cell-region_id="{ row }">{{ row.region_id || '—' }}</template>
        <template #cell-phone="{ row }">{{ row.phone || '—' }}</template>
        <template #actions="{ row }"><button class="icon-btn btn-sm" type="button" :aria-label="t('View')" :title="t('View')" @click="openDetail(row)"><AppIcon name="eye" :size="15" /></button></template>
      </DataTable>
    </div>

    <Modal :open="detailOpen" :title="selected?.company_name || t('Supplier details')" @close="detailOpen = false">
      <div v-if="detailLoading" class="loading-block"><span class="spinner" />{{ t('Loading details…') }}</div>
      <div v-else-if="selected" class="kv-list">
        <div class="kv-row"><span class="kv-key">{{ t('Username') }}</span><span class="kv-val">{{ selected.username }}</span></div>
        <div class="kv-row"><span class="kv-key">{{ t('Company') }}</span><span class="kv-val">{{ selected.company_name || '—' }}</span></div>
        <div class="kv-row"><span class="kv-key">{{ t('INN') }}</span><span class="kv-val mono">{{ selected.inn || '—' }}</span></div>
        <div class="kv-row"><span class="kv-key">{{ t('Bank account') }}</span><span class="kv-val mono">{{ selected.bank_account || '—' }}</span></div>
        <div class="kv-row"><span class="kv-key">{{ t('Region') }}</span><span class="kv-val">{{ selected.region_id || '—' }}</span></div>
        <div class="kv-row"><span class="kv-key">{{ t('Phone') }}</span><span class="kv-val">{{ selected.phone || '—' }}</span></div>
      </div>
      <template #footer><button class="btn btn-secondary" type="button" @click="detailOpen = false">{{ t('Cancel') }}</button></template>
    </Modal>
  </div>
</template>
