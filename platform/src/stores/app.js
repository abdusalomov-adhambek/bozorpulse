import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useAppStore = defineStore('app', () => {
  const toasts = ref([])
  let sequence = 0

  function dismiss(id) {
    toasts.value = toasts.value.filter((toast) => toast.id !== id)
  }

  function toast(message, type = 'success', timeout = 3800) {
    const id = ++sequence
    toasts.value.push({ id, message, type })
    if (timeout) {
      setTimeout(() => dismiss(id), timeout)
    }
    return id
  }

  const success = (message) => toast(message, 'success')
  const error = (message) => toast(message, 'error')
  const info = (message) => toast(message, 'info')

  // Small helper used by pages that want a standard { data, loading, error } trio.
  function createResource() {
    const data = ref(null)
    const loading = ref(false)
    const error = ref('')

    async function run(fn) {
      loading.value = true
      error.value = ''
      try {
        data.value = await fn()
        return data.value
      } catch (err) {
        error.value = err?.message || 'Something went wrong'
        throw err
      } finally {
        loading.value = false
      }
    }

    return { data, loading, error, run }
  }

  return { toasts, toast, dismiss, success, error, info, createResource }
})
