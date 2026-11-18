<script setup>
import { t } from '../../i18n'
defineProps({
  modelValue: { type: Boolean, default: false },
  label: { type: String, default: '' },
  description: { type: String, default: '' },
  disabled: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue', 'change'])
</script>

<template>
  <button
    type="button"
    class="ui-switch"
    :class="{ on: modelValue, disabled }"
    role="switch"
    :aria-checked="modelValue"
    :disabled="disabled"
    @click="emit('update:modelValue', !modelValue); emit('change', !modelValue)"
  >
    <span v-if="label || description" class="text">
      <span class="label">{{ label }}</span>
      <span v-if="description" class="description">{{ description }}</span>
    </span>
    <span class="track"><span class="knob"></span></span>
  </button>
</template>

<style scoped>
.ui-switch {
  display: inline-flex; align-items: center; gap: 12px;
  border: 0; background: none; padding: 0; cursor: pointer;
  font: inherit; color: var(--text); text-align: left;
}
.ui-switch.disabled { opacity: .55; cursor: not-allowed; }
.text { display: flex; flex-direction: column; gap: 2px; }
.label { font-size: 13px; font-weight: 600; }
.description { font-size: 11.5px; color: var(--text-3); }
.track {
  position: relative; width: 36px; height: 20px; border-radius: 999px;
  background: var(--border-strong); transition: background .18s; flex-shrink: 0;
}
.on .track { background: var(--primary); }
.knob {
  position: absolute; top: 2px; left: 2px; width: 16px; height: 16px;
  background: #fff; border-radius: 50%; box-shadow: 0 1px 2px rgba(0,0,0,.15);
  transition: transform .18s;
}
.on .knob { transform: translateX(16px); }
.ui-switch:focus-visible .track { box-shadow: 0 0 0 3px var(--primary-50); }
</style>
