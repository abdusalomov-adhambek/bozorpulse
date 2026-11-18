<script setup>
import { t } from '../i18n'
import { computed } from 'vue'
import AppIcon from './AppIcon.vue'

const props = defineProps({
  status: { type: String, default: 'new' },
  compact: { type: Boolean, default: false },
})

const steps = ['new', 'confirmed', 'processing', 'shipped', 'delivered']
const labels = {
  new: 'New',
  confirmed: 'Confirmed',
  processing: 'Processing',
  shipped: 'Shipped',
  delivered: 'Delivered',
  cancelled: 'Cancelled',
}
// Values are i18n keys; translated in the template so locale changes apply live.

const icons = {
  new: 'receipt',
  confirmed: 'check',
  processing: 'gear',
  shipped: 'truck',
  delivered: 'check-circle',
  cancelled: 'x-circle',
}

const cancelled = computed(() => props.status === 'cancelled')
const current = computed(() => steps.indexOf(props.status))
const percent = computed(() => {
  if (cancelled.value) return 0
  if (current.value <= 0) return 0
  return Math.round((current.value / (steps.length - 1)) * 100)
})
</script>

<template>
  <div class="order-status" :class="{ compact, cancelled }">
    <div v-if="cancelled" class="cancelled-bar">
      <AppIcon name="x-circle" :size="14" />{{ t('Order cancelled') }}
    </div>
    <template v-else>
      <div class="track">
        <div class="fill" :style="{ width: percent + '%' }"></div>
        <span
          v-for="(step, i) in steps"
          :key="step"
          class="node"
          :class="{ done: i <= current, current: i === current }"
          :style="{ left: (i / (steps.length - 1)) * 100 + '%' }"
        >
          <AppIcon :name="icons[step]" :size="16" />
        </span>
      </div>
      <div class="labels">
        <span
          v-for="(step, i) in steps"
          :key="step"
          :class="{ done: i <= current, current: i === current, first: i === 0, last: i === steps.length - 1 }"
          :style="{ left: (i / (steps.length - 1)) * 100 + '%' }"
        >
          {{ t(labels[step]) }}
        </span>
      </div>
    </template>
  </div>
</template>

<style scoped>
.order-status { width: 100%; min-width: 220px; padding: 8px 0; }
.track { position: relative; height: 4px; margin: 10px 0 20px; border-radius: 999px; background: var(--border); }
.fill { height: 100%; border-radius: 999px; background: var(--primary); transition: width .3s ease; }
.node {
  position: absolute; top: 50%; width: 32px; height: 32px;
  display: grid; place-items: center;
  border-radius: 50%; background: var(--surface); border: 1px solid var(--border-strong);
  color: var(--text-3); transform: translate(-50%, -50%);
  transition: border-color .2s, background .2s, color .2s;
}
.node.done { border-color: var(--primary); background: var(--primary-50); color: var(--primary); }
.node.current { border-color: var(--primary); background: var(--primary); color: #fff; box-shadow: 0 0 0 3px var(--primary-50); }
/* Labels sit directly under their status node, using the same percentage
   positions as the nodes on the track. */
.labels { position: relative; height: 15px; font-size: 10px; color: var(--text-3); }
.labels span { position: absolute; transform: translateX(-50%); white-space: nowrap; }
.labels span.first { transform: none; }
.labels span.last { transform: translateX(-100%); }
.labels span.done { color: var(--text-2); }
.labels span.current { color: var(--primary); font-weight: 750; }
.cancelled-bar {
  display: inline-flex; align-items: center; gap: 7px;
  padding: 7px 11px; border-radius: 999px;
  background: var(--danger-50); color: var(--danger); font-size: 11.5px; font-weight: 700;
}
.cancelled-bar .dot { width: 7px; height: 7px; border-radius: 50%; background: var(--danger); }
.order-status.compact { min-width: 180px; }
.order-status.compact .labels { font-size: 8.5px; }
</style>
