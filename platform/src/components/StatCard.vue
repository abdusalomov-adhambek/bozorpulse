<script setup>
import { t } from '../i18n'
import { computed } from 'vue'

const props = defineProps({
  label: { type: String, required: true },
  value: { type: [String, Number], default: '—' },
  delta: { type: [String, Number], default: null },
  icon: { type: String, default: 'chart' },
  tone: { type: String, default: 'primary' }
})

const icons = {
  chart: '<path d="M3 3v18h18"/><path d="m7 14 4-4 3 3 5-6"/>',
  orders: '<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 2v6h6"/><path d="M8 13h8M8 17h5"/>',
  users: '<path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75"/>',
  box: '<path d="m21 8-9-5-9 5 9 5 9-5z"/><path d="M3 8v8l9 5 9-5V8"/><path d="M12 13v8"/>',
  wallet: '<rect x="2" y="5" width="20" height="14" rx="2"/><path d="M2 10h20"/>',
  warehouse: '<path d="M3 21V8l9-5 9 5v13"/><path d="M3 21h18"/><path d="M9 21v-8h6v8"/>',
  trend: '<path d="m3 17 6-6 4 4 8-8"/><path d="M15 7h6v6"/>',
  clock: '<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>'
}

const iconPath = computed(() => icons[props.icon] || icons.chart)
const deltaText = computed(() => {
  if (props.delta === null || props.delta === undefined || props.delta === '') return ''
  const str = String(props.delta)
  return str.startsWith('+') || str.startsWith('-') ? str : `+${str}`
})
const deltaUp = computed(() => !String(props.delta).trim().startsWith('-'))
</script>

<template>
  <div class="stat-card" :class="`stat-tone-${tone}`">
    <div class="stat-top">
      <div class="stat-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" v-html="iconPath" />
      </div>
      <span v-if="deltaText" class="stat-delta" :class="deltaUp ? 'up' : 'down'">
        <svg v-if="deltaUp" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><path d="m5 15 7-7 7 7"/></svg>
        <svg v-else width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><path d="m5 9 7 7 7-7"/></svg>
        {{ deltaText }}
      </span>
    </div>
    <div class="col">
      <span class="stat-value mono">{{ value }}</span>
      <span class="stat-label">{{ label }}</span>
    </div>
  </div>
</template>
