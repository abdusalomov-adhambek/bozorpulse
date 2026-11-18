<script setup>
import { t } from '../../i18n'
import { computed } from 'vue'

const props = defineProps({
  modelValue: { type: [Number, String], default: 0 },
  min: { type: Number, default: 0 },
  max: { type: Number, default: undefined },
  step: { type: Number, default: 1 },
  disabled: { type: Boolean, default: false },
  size: { type: String, default: 'md' }, // sm | md
})
const emit = defineEmits(['update:modelValue', 'change'])

const current = computed(() => {
  const n = Number(props.modelValue)
  return Number.isFinite(n) ? n : props.min
})

function commit(value) {
  let next = Number(value)
  if (!Number.isFinite(next)) next = props.min
  if (next < props.min) next = props.min
  if (props.max !== undefined && next > props.max) next = props.max
  emit('update:modelValue', next)
  emit('change', next)
}

function stepBy(delta) {
  if (props.disabled) return
  commit(current.value + props.step * delta)
}
</script>

<template>
  <div class="number-input" :class="[`size-${size}`, { disabled }]">
    <button type="button" class="step-btn" :disabled="disabled || current <= min" :aria-label="t('Decrease')" @click="stepBy(-1)">−</button>
    <input
      type="number"
      :value="modelValue"
      :min="min"
      :max="max"
      :step="step"
      :disabled="disabled"
      @input="commit($event.target.value)"
    />
    <button type="button" class="step-btn" :disabled="disabled || (max !== undefined && current >= max)" :aria-label="t('Increase')" @click="stepBy(1)">+</button>
  </div>
</template>

<style scoped>
.number-input {
  display: inline-flex; align-items: stretch; width: 100%;
  min-height: 38px;
  border: 1px solid var(--border-strong); border-radius: var(--radius-sm);
  background: var(--surface); overflow: hidden;
}
.number-input.size-sm { min-height: 32px; }
.number-input:focus-within { border-color: var(--primary); box-shadow: 0 0 0 3px var(--primary-50); }
.number-input.disabled { background: var(--surface-2); opacity: .65; }
.number-input input {
  flex: 1; min-width: 0; width: 100%; border: 0; outline: 0; background: transparent;
  text-align: center; font: inherit; font-size: 13px; color: var(--text);
  -moz-appearance: textfield;
}
.number-input input::-webkit-inner-spin-button { -webkit-appearance: none; margin: 0; }
.step-btn {
  width: 30px; flex-shrink: 0; border: 0; background: var(--surface-2);
  color: var(--text-2); font-size: 15px; font-weight: 700; cursor: pointer;
  transition: background .15s, color .15s;
}
.step-btn:hover:not(:disabled) { background: var(--primary-50); color: var(--primary); }
.step-btn:disabled { opacity: .35; cursor: not-allowed; }
.number-input.size-sm .step-btn { width: 24px; font-size: 13px; }
.number-input.size-sm input { font-size: 12px; }
</style>
