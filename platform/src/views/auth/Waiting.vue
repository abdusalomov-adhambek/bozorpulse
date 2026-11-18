<script setup>
import { t } from '../../i18n'
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../../stores/auth'
import { useAppStore } from '../../stores/app'
import LanguageSwitcher from '../../components/LanguageSwitcher.vue'
import Modal from '../../components/Modal.vue'
import AppIcon from '../../components/AppIcon.vue'

const router = useRouter()
const auth = useAuthStore()
const app = useAppStore()

const checking = ref(false)
const logoutOpen = ref(false)

async function checkStatus(silent = false) {
  checking.value = true
  try {
    await auth.me()
    if (auth.isConfirmed || auth.isAdmin) {
      if (!silent) app.success(t('Your account has been approved!'))
      router.push(`/${auth.role}`)
    } else if (!silent) {
      app.info(t('Your account is still awaiting approval.'))
    }
  } catch (err) {
    if (!silent) app.error(err?.message || 'Could not check status.')
  } finally {
    checking.value = false
  }
}

function confirmLogout() {
  logoutOpen.value = false
  auth.logout()
  router.push({ name: 'login' })
}

onMounted(() => checkStatus(true))
</script>

<template>
  <div class="waiting-shell">
    <div class="card waiting-card">
      <div class="waiting-icon"><AppIcon name="clock" :size="34" /></div>
      <h1>{{ t("Waiting for admin approval") }}</h1>
      <p>{{ t('Thanks for registering,') }} <strong>{{ auth.displayName }}</strong>. Your {{ auth.role }} account is currently
        <strong>{{ auth.status || 'waiting' }}</strong>{{ t('and needs to be reviewed by an administrator before you can use your cabinet.') }}
      </p>

      <div class="waiting-steps">
        <div class="waiting-step"><span class="n">1</span>{{ t('Registration submitted successfully') }}</div>
        <div class="waiting-step"><span class="n">2</span>{{ t('Administrator reviews your company details') }}</div>
        <div class="waiting-step"><span class="n">3</span>{{ t('Account confirmed — full access unlocked') }}</div>
      </div>

      <div class="row" style="justify-content:center">
        <button class="btn" type="button" :disabled="checking" @click="checkStatus()">
          <span v-if="checking" class="spinner" style="border-color:rgba(255,255,255,.4);border-top-color:#fff" />
          {{ checking ? 'Checking…' : 'Check status' }}
        </button>
        <button class="btn btn-secondary" type="button" @click="logoutOpen = true">{{ t("Log out") }}</button>
      </div>
    </div>

    <Modal :open="logoutOpen" :title="t('Log out?')" @close="logoutOpen = false">
      <div class="logout-confirm">
        <span class="logout-confirm-icon"><AppIcon name="logout" :size="22" /></span>
        <div>
          <strong>{{ t('Are you sure you want to log out?') }}</strong>
          <p>{{ t('You will need to sign in again to access your workspace.') }}</p>
        </div>
      </div>
      <template #footer>
        <button class="btn btn-secondary" type="button" @click="logoutOpen = false">{{ t('Cancel') }}</button>
        <button class="btn btn-danger" type="button" @click="confirmLogout">{{ t('Log out') }}</button>
      </template>
    </Modal>
  </div>
</template>

<style scoped>
.logout-confirm { display: flex; align-items: flex-start; gap: 14px; text-align: left; }
.logout-confirm-icon {
  width: 42px; height: 42px; flex-shrink: 0; display: grid; place-items: center;
  border-radius: 10px; background: var(--danger-50); color: var(--danger);
}
.logout-confirm strong { font-size: 14px; }
.logout-confirm p { margin: 5px 0 0; color: var(--text-2); font-size: 13px; line-height: 1.5; }
</style>
