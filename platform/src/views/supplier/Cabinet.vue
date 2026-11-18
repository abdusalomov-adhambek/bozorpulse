<script setup>
import { t } from '../../i18n'
import { computed, ref, onMounted } from 'vue'
import api from '../../api/client'
import { useAuthStore } from '../../stores/auth'
import { useAppStore } from '../../stores/app'
import { AppInput, AppSelect, FormField } from '../../components/ui'

const auth = useAuthStore()
const app = useAppStore()
const loading = ref(false)
const saving = ref(false)
const error = ref('')
const company = ref(null)
const regions = ref([])
const form = ref({})
const detailsLocked = computed(() => auth.isConfirmed || auth.isAdmin)
const regionOptions = computed(() => regions.value.map((r) => ({ value: r.id, label: r.name })))

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [companyData, regs] = await Promise.all([api.get('/companies/me'), api.get('/regions').catch(() => [])])
    company.value = companyData
    regions.value = Array.isArray(regs) ? regs : (regs?.data || [])
    form.value = {
      name: companyData?.name || '',
      inn: companyData?.inn || auth.user?.inn || '',
      bank_account: companyData?.bank_account || auth.user?.bank_account || '',
      phone: companyData?.phone || auth.user?.phone || '',
      region_id: companyData?.region_id || auth.user?.region_id || '',
      address: companyData?.address || '',
      city: companyData?.city || ''
    }
  } catch (err) {
    error.value = err?.message || 'Failed to load company profile.'
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    const payload = { ...form.value }
    if (detailsLocked.value) { delete payload.inn; delete payload.bank_account }
    if (payload.region_id) payload.region_id = Number(payload.region_id)
    const updated = await api.patch('/companies/me', payload)
    company.value = updated
    app.success(t('Company profile updated'))
  } catch (err) {
    app.error(err?.message || 'Could not update company profile.')
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div>
    <div class="page-header">
      <div><h1 class="page-title">{{ t("Cabinet") }}</h1><p class="page-desc">{{ t('Your account and company information.') }}</p></div>
    </div>

    <div class="grid grid-2">
      <div class="card">
        <div class="card-head"><div class="card-title">{{ t('Account') }}</div></div>
        <div class="card-body">
          <div class="kv-list">
            <div class="kv-row"><span class="kv-key">{{ t("Username") }}</span><span class="kv-val">{{ auth.user?.username || '—' }}</span></div>
            <div class="kv-row"><span class="kv-key">{{ t('Role') }}</span><span class="kv-val">{{ auth.role }}</span></div>
            <div class="kv-row"><span class="kv-key">{{ t("Status") }}</span><span class="kv-val text-success">{{ auth.status }}</span></div>
            <div class="kv-row"><span class="kv-key">{{ t('Registered') }}</span><span class="kv-val">{{ auth.user?.created_at ? new Date(auth.user.created_at).toLocaleDateString() : '—' }}</span></div>
          </div>
        </div>
      </div>

      <div class="card">
        <div class="card-head"><div><div class="card-title">{{ t('Company profile') }}</div><div class="card-sub">{{ t('Keep your details up to date') }}</div></div></div>
        <div class="card-body">
          <div v-if="error" class="alert alert-error mb-3">{{ error }}</div>
          <div v-if="loading" class="loading-block"><span class="spinner" />{{ t('Loading profile…') }}</div>
          <template v-else>
            <div class="form-grid">
              <FormField :label="t('Company name')">
                <AppInput v-model="form.name" />
              </FormField>
              <FormField :label="t('INN')">
                <AppInput v-model="form.inn" :disabled="detailsLocked" />
              </FormField>
              <FormField :label="t('Bank account')">
                <AppInput v-model="form.bank_account" :disabled="detailsLocked" />
                <p v-if="detailsLocked" class="field-hint">{{ t('INN and bank account are locked after approval.') }}</p>
              </FormField>
              <FormField :label="t('Phone')">
                <AppInput v-model="form.phone" />
              </FormField>
              <FormField :label="t('Region')">
                <AppSelect v-model="form.region_id" :options="regionOptions" :placeholder="t('None')" clearable />
              </FormField>
              <FormField :label="t('City')">
                <AppInput v-model="form.city" />
              </FormField>
              <FormField :label="t('Address')" class="span-2">
                <AppInput v-model="form.address" />
              </FormField>
            </div>
            <div class="form-actions">
              <button class="btn btn-secondary" type="button" @click="load">{{ t('Reset') }}</button>
              <button class="btn" type="button" :disabled="saving" @click="save">{{ saving ? 'Saving…' : 'Save changes' }}</button>
            </div>
          </template>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.span-2 { grid-column: 1 / -1; }
</style>
