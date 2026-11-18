<script setup>
import { t } from '../../i18n'
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../../stores/auth'
import { useAppStore } from '../../stores/app'
import LanguageSwitcher from '../../components/LanguageSwitcher.vue'
import AppIcon from '../../components/AppIcon.vue'
import { AppInput, FormField } from '../../components/ui'

const router = useRouter()
const auth = useAuthStore()
const app = useAppStore()

const username = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')
const demoLoading = ref('')

function afterAuth() {
  if (!auth.isAdmin && !auth.isConfirmed) {
    router.push({ name: 'waiting' })
  } else {
    router.push(`/${auth.role}`)
  }
}

async function submit() {
  if (!username.value || !password.value) {
    error.value = 'Please enter your username and password.'
    return
  }
  loading.value = true
  error.value = ''
  try {
    await auth.login(username.value.trim(), password.value)
    app.success(`Welcome back, ${auth.displayName}`)
    afterAuth()
  } catch (err) {
    error.value = err?.message || 'Invalid username or password.'
  } finally {
    loading.value = false
  }
}

async function demo(role) {
  demoLoading.value = role
  error.value = ''
  try {
    await auth.demoLogin(role)
    app.success(`Signed in as ${auth.displayName}`)
    afterAuth()
  } catch (err) {
    error.value = err?.message || 'Could not sign in to the demo account.'
  } finally {
    demoLoading.value = ''
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
        <h1>{{ t('The intelligent B2B marketplace for modern trade.') }}</h1>
        <p>{{ t('Discover suppliers, compare live market prices, manage warehouses and make smarter purchasing decisions — all powered by AI.') }}</p>
      </div>
      <div class="auth-features">
        <div class="auth-feature"><AppIcon name="chart" :size="17" />{{ t('Real-time market price intelligence') }}</div>
        <div class="auth-feature"><AppIcon name="bot" :size="17" />{{ t('AI supplier recommendations') }}</div>
        <div class="auth-feature"><AppIcon name="warehouse" :size="17" />{{ t('Multi-warehouse inventory control') }}</div>
      </div>
    </aside>

    <section class="auth-panel">
      <div class="auth-form">
          <div class="auth-lang"><LanguageSwitcher /></div>
        <h2>{{ t('Welcome back') }}</h2>
        <p class="lead">{{ t('Sign in to your BozorPulse account to continue.') }}</p>

        <div v-if="error" class="alert alert-error mb-3">{{ error }}</div>

        <form @submit.prevent="submit">
          <FormField :label="t('Username')" for="username">
            <AppInput id="username" v-model="username" autocomplete="username" :placeholder="t('Enter your username')" />
          </FormField>
          <FormField :label="t('Password')" for="password">
            <AppInput id="password" v-model="password" type="password" autocomplete="current-password" :placeholder="t('Enter your password')" @enter="submit" />
          </FormField>
          <div class="mt-3">
            <button class="btn btn-lg btn-block" type="submit" :disabled="loading">
              <span v-if="loading" class="spinner" style="border-top-color:#fff;border-color:rgba(255,255,255,.4);border-top-color:#fff" />
              {{ loading ? t('Signing in…') : t('Sign in') }}
            </button>
          </div>
        </form>

        <div class="demo-box">
          <div class="demo-title">{{ t('Try a demo account — one click') }}</div>
          <div class="demo-buttons">
            <button class="demo-btn" type="button" :disabled="!!demoLoading" @click="demo('admin')">
              <AppIcon name="shield" :size="19" class="demo-icon" />
              <span>{{ demoLoading === 'admin' ? '…' : 'Admin' }}</span>
            </button>
            <button class="demo-btn" type="button" :disabled="!!demoLoading" @click="demo('supplier')">
              <AppIcon name="box" :size="19" class="demo-icon" />
              <span>{{ demoLoading === 'supplier' ? '…' : 'Supplier' }}</span>
            </button>
            <button class="demo-btn" type="button" :disabled="!!demoLoading" @click="demo('customer')">
              <AppIcon name="cart" :size="19" class="demo-icon" />
              <span>{{ demoLoading === 'customer' ? '…' : 'Customer' }}</span>
            </button>
          </div>
        </div>

        <div class="auth-alt">{{ t("Don't have an account?") }} <RouterLink to="/register">{{ t('Create one') }}</RouterLink></div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.auth-form .form-field + .form-field { margin-top: 16px; }
</style>
