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
const loading = ref(false)
const error = ref('')
const modalOpen = ref(false)
const saving = ref(false)
const editing = ref(null)
const form = ref({ name: '', parent_id: '' })

const columns = [
  { key: 'id', label: t('ID') },
  { key: 'name', label: t('Name') },
  { key: 'parent', label: t('Parent category') },
  { key: 'created_at', label: t('Created') }
]

async function load() {
  loading.value = true
  error.value = ''
  try {
    const data = await api.get('/categories')
    rows.value = Array.isArray(data) ? data : (data?.data || [])
  } catch (err) {
    error.value = err?.message || 'Failed to load categories.'
  } finally {
    loading.value = false
  }
}

function parentName(row) {
  if (!row.parent_id) return '—'
  return rows.value.find((c) => c.id === row.parent_id)?.name || `#${row.parent_id}`
}

function openCreate() {
  editing.value = null
  form.value = { name: '', parent_id: '' }
  modalOpen.value = true
}

function openEdit(row) {
  editing.value = row
  form.value = { name: row.name, parent_id: row.parent_id || '' }
  modalOpen.value = true
}

async function save() {
  if (!form.value.name.trim()) {
    app.error(t('Category name is required.'))
    return
  }
  saving.value = true
  try {
    const payload = { name: form.value.name.trim() }
    if (form.value.parent_id) payload.parent_id = Number(form.value.parent_id)
    else payload.parent_id = null
    if (editing.value) {
      await api.patch(`/categories/${editing.value.id}`, payload)
      app.success(t('Category updated'))
    } else {
      await api.post('/categories', payload)
      app.success(t('Category created'))
    }
    modalOpen.value = false
    await load()
  } catch (err) {
    app.error(err?.message || 'Could not save category.')
  } finally {
    saving.value = false
  }
}

async function remove(row) {
  if (!window.confirm(`Delete category "${row.name}"?`)) return
  try {
    await api.del(`/categories/${row.id}`)
    app.success(t('Category deleted'))
    await load()
  } catch (err) {
    app.error(err?.message || 'Could not delete category.')
  }
}

onMounted(load)
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t("Product categories") }}</h1>
        <p class="page-desc">{{ t('Organize products into categories and subcategories.') }}</p>
      </div>
      <button class="btn" type="button" @click="openCreate">{{ t('+ Add category') }}</button>
    </div>

    <div v-if="error" class="alert alert-error mb-3">{{ error }}</div>

    <div class="card">
      <DataTable :columns="columns" :rows="rows" :loading="loading" :empty-text="t('No categories yet')" empty-:desc="t('Create your first product category to get started.')">
        <template #cell-id="{ row }"><span class="subtle mono">#{{ row.id }}</span></template>
        <template #cell-name="{ row }"><span class="cell-strong">{{ row.name }}</span></template>
        <template #cell-parent="{ row }">{{ parentName(row) }}</template>
        <template #cell-created_at="{ row }"><span class="subtle">{{ row.created_at ? new Date(row.created_at).toLocaleDateString() : '—' }}</span></template>
        <template #actions="{ row }">
          <button class="icon-btn btn-sm" type="button" :aria-label="t('Edit')" :title="t('Edit')" @click="openEdit(row)"><AppIcon name="edit" :size="15" /></button>
          <button class="icon-btn btn-sm danger" type="button" :aria-label="t('Delete')" :title="t('Delete')" @click="remove(row)"><AppIcon name="trash" :size="15" /></button>
        </template>
      </DataTable>
    </div>

    <Modal :open="modalOpen" :title="editing ? t('Edit category') : t('New category')" @close="modalOpen = false">
      <FormField :label="t('Category name')" required>
        <AppInput v-model="form.name" placeholder="e.g. Fresh Produce" />
      </FormField>
      <FormField :label="t('Parent category')" :hint="t('Optional')" class="mt-2">
        <AppSelect
          v-model="form.parent_id"
          :options="[{ value: '', label: 'None (top level)' }, ...rows.filter(c => !editing || c.id !== editing.id).map(c => ({ value: c.id, label: c.name }))]"
          clearable
        />
      </FormField>
      <template #footer>
        <button class="btn btn-secondary" type="button" @click="modalOpen = false">{{ t('Cancel') }}</button>
        <button class="btn" type="button" :disabled="saving" @click="save">{{ saving ? 'Saving…' : 'Save category' }}</button>
      </template>
    </Modal>
  </div>
</template>
