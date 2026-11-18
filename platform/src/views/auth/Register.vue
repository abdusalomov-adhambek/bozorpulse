<script setup>
import { t } from '../../i18n'
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../../stores/auth'
import { useAppStore } from '../../stores/app'
import LanguageSwitcher from '../../components/LanguageSwitcher.vue'
import AppIcon from '../../components/AppIcon.vue'
import { get } from '../../api/client'
import { AppInput, AppRadioGroup, AppSelect, FormField } from '../../components/ui'

const router = useRouter()
const auth = useAuthStore()
const app = useAppStore()

const form = ref({
  username: '',
  password: '',
  role: 'supplier',
  inn: '',
  bank_account: '',
  company_name: '',
  region_id: '',
  phone: ''
})
const loading = ref(false)
const error = ref('')
const submitted = ref(false)

const roles = [
  { value: 'supplier', label: t('Supplier'), description: t('Sell your products'), icon: 'box' },
  { value: 'customer', label: t('Customer'), description: t('Buy & source goods'), icon: 'cart' },
]
const regions = ref([])
onMounted(async () => {
  try {
    const res = await get('/regions', {}, { auth: false })
    regions.value = res?.data || []
  } catch { regions.value = [] }
})

async function submit() {
  error.value = ''
  if (!form.value.username || !form.value.password || !form.value.inn || !form.value.bank_account || !form.value.company_name) {
    error.value = 'Please fill in all required fields.'
    return
  }
  const payload = {
    username: form.value.username.trim(),
    password: form.value.password,
    role: form.value.role,
    inn: form.value.inn.trim(),
    bank_account: form.value.bank_account.trim(),
    company_name: form.value.company_name.trim()
  }
  if (form.value.region_id) payload.region_id = form.value.region_id
  if (form.value.phone) payload.phone = form.value.phone.trim()

  loading.value = true
  try {
    await auth.register(payload)
    submitted.value = true
    app.success(t('Registration submitted for approval'))
  } catch (err) {
    error.value = err?.message || 'Registration failed. Please try again.'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="auth-shell">
    <aside class="auth-visual">
      <div class="auth-brand">
        <div class="brand-mark">BP</div>
        <span>BozorPulse</span>
      </div>
      <div class="auth-hero">
        <h1>{{ t('Grow your business with trusted partners.') }}</h1>
        <p>{{ t('Join a verified network of suppliers and buyers. Get approved by an administrator and unlock your dedicated workspace.') }}</p>
      </div>
      <div class="auth-features">
        <div class="auth-feature"><AppIcon name="shield-check" :size="17" />{{ t('Verified, trusted business accounts') }}</div>
        <div class="auth-feature"><AppIcon name="chart" :size="17" />{{ t('Competitive, transparent pricing') }}</div>
        <div class="auth-feature"><AppIcon name="handshake" :size="17" />{{ t('Smart partner matching') }}</div>
      </div>
    </aside>

    <section class="auth-panel">
      <div class="auth-form">
          <div class="auth-lang"><LanguageSwitcher /></div>
        <template v-if="!submitted">
          <h2>{{ t('Create your account') }}</h2>
          <p class="lead">{{ t('Register as a supplier or customer. Approval is required before use.') }}</p>

          <div v-if="error" class="alert alert-error mb-3">{{ error }}</div>

          <form @submit.prevent="submit">
            <FormField :label="t('I am a')" class="mb-3">
              <AppRadioGroup v-model="form.role" :options="roles" inline />
            </FormField>

            <div class="form-grid">
              <FormField :label="t('Company name')" required>
                <AppInput v-model="form.company_name" :placeholder="t('LLC Your Company')" />
              </FormField>
              <FormField :label="t('Username')" required>
                <AppInput v-model="form.username" autocomplete="username" :placeholder="t('Choose a username')" />
              </FormField>
              <FormField :label="t('INN (Tax ID)')" required>
                <AppInput v-model="form.inn" placeholder="e.g. 123456789" />
              </FormField>
              <FormField :label="t('Bank account')" required>
                <AppInput v-model="form.bank_account" :placeholder="t('Account number')" />
              </FormField>
              <FormField :label="t('Password')" required>
                <AppInput v-model="form.password" type="password" autocomplete="new-password" :placeholder="t('Create a password')" />
              </FormField>
              <FormField :label="t('Phone')" :hint="t('Optional')">
                <AppInput v-model="form.phone" type="tel" placeholder="+998 90 000 00 00" />
              </FormField>
              <FormField :label="t('Region')" :hint="t('Optional')">
                <AppSelect
                  v-model="form.region_id"
                  :options="regions"
                  value-key="id"
                  label-key="name"
                  :placeholder="t('Select a region')"
                  clearable
                />
              </FormField>
            </div>

            <div class="mt-3">
              <button class="btn btn-lg btn-block" type="submit" :disabled="loading">
                <span v-if="loading" class="spinner" style="border-color:rgba(255,255,255,.4);border-top-color:#fff" />
                {{ loading ? t('Submitting…') : t('Submit registration') }}
              </button>
            </div>
          </form>

          <div class="auth-alt">{{ t('Already registered?') }} <RouterLink to="/login">{{ t('Sign in') }}</RouterLink></div>
        </template>

        <template v-else>
          <div class="waiting-icon" style="margin-left:0">⏳</div>
          <h2>{{ t('Waiting for admin approval') }}</h2>
          <p class="lead">{{ t('Your registration has been submitted. An administrator will review your account shortly.') }}</p>
          <div class="alert alert-warning mb-3">{{ t('Your account status is') }} <strong>{{ t('waiting') }}</strong>. You'll be able to access your {{ form.role }} cabinet once it is confirmed.
          </div>
          <RouterLink to="/login" class="btn btn-block"><AppIcon name="chevron-left" :size="15" />{{ t('Back to sign in') }}</RouterLink>
        </template>
      </div>
    </section>
  </div>
</template>
