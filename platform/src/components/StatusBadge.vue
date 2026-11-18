<script setup>
import { t } from '../i18n'
import { computed } from 'vue'

const props = defineProps({
  status: { type: String, default: '' }
})

const map = {
  waiting: 'badge-yellow',
  pending: 'badge-yellow',
  confirmed: 'badge-green',
  paid: 'badge-green',
  delivered: 'badge-green',
  rejected: 'badge-red',
  cancelled: 'badge-red',
  failed: 'badge-red',
  new: 'badge-blue',
  processing: 'badge-purple',
  shipped: 'badge-indigo'
}

const cls = computed(() => map[props.status] || 'badge-gray')
const label = computed(() => {
  if (!props.status) return '—'
  return props.status.charAt(0).toUpperCase() + props.status.slice(1)
})
</script>

<template>
  <span class="badge" :class="cls">{{ label }}</span>
</template>
