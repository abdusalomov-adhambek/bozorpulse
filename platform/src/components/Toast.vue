<script setup>
import { t } from '../i18n'
import { useAppStore } from '../stores/app'
import AppIcon from './AppIcon.vue'

const app = useAppStore()

const icons = {
  success: '✓',
  error: '!',
  info: 'i'
}
</script>

<template>
  <div class="toast-stack" aria-live="polite">
    <TransitionGroup name="toast">
      <div v-for="toast in app.toasts" :key="toast.id" class="toast" :class="`toast-${toast.type}`">
        <span class="toast-icon"><AppIcon v-if="toast.type === 'success'" name="check" :size="15" /><template v-else>{{ icons[toast.type] || '•' }}</template></span>
        <span class="grow">{{ toast.message }}</span>
        <button class="toast-close" type="button" :aria-label="t('Dismiss')" @click="app.dismiss(toast.id)">×</button>
      </div>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.toast-enter-active, .toast-leave-active { transition: all .22s ease; }
.toast-enter-from { opacity: 0; transform: translateX(20px); }
.toast-leave-to { opacity: 0; transform: translateX(20px); }
.toast-leave-active { position: absolute; right: 0; }
</style>
