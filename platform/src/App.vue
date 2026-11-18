<script setup>
import { t } from './i18n'
import { onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from './stores/auth'
import Toast from './components/Toast.vue'

const router = useRouter()
const auth = useAuthStore()

function handleUnauthorized() {
  auth.forceLogout()
  router.push({ name: 'login' })
}

onMounted(() => window.addEventListener('auth:unauthorized', handleUnauthorized))
onUnmounted(() => window.removeEventListener('auth:unauthorized', handleUnauthorized))
</script>

<template>
  <router-view />
  <Toast />
</template>
