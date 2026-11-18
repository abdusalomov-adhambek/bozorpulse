<script setup>
import { t } from '../../i18n'
import AppIcon from '../AppIcon.vue'

const props = defineProps({
  modelValue: { type: [String, Number, Boolean], default: null },
  options: { type: Array, default: () => [] },
  inline: { type: Boolean, default: false },
  disabled: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue', 'change'])

function normalize(o) {
  if (typeof o !== 'object' || o === null) return { value: o, label: String(o), description: '' }
  return {
    value: o.value ?? o.id,
    label: o.label ?? o.name ?? String(o.value),
    description: o.description || '',
    icon: o.icon || '',
    disabled: !!o.disabled,
  }
}

function choose(value) {
  emit('update:modelValue', value)
  emit('change', value)
}
</script>

<template>
  <div class="ui-radio" :class="{ inline }" role="radiogroup">
    <label
      v-for="(raw, i) in options"
      :key="i"
      class="radio-option"
      :class="{ active: normalize(raw).value === modelValue, disabled: disabled || normalize(raw).disabled }"
    >
      <input
        type="radio"
        :name="`ui-radio-${i}`"
        :checked="normalize(raw).value === modelValue"
        :disabled="disabled || normalize(raw).disabled"
        @change="choose(normalize(raw).value)"
      />
      <span class="dot"></span>
      <AppIcon v-if="normalize(raw).icon" :name="normalize(raw).icon" :size="17" class="radio-icon" />
      <span class="text">
        <span class="label">{{ normalize(raw).label }}</span>
        <span v-if="normalize(raw).description" class="description">{{ normalize(raw).description }}</span>
      </span>
    </label>
  </div>
</template>

<style scoped>
.ui-radio { display: flex; flex-direction: column; gap: 8px; }
.ui-radio.inline { flex-direction: row; flex-wrap: wrap; gap: 10px; }
.radio-option {
  display: inline-flex; align-items: flex-start; gap: 9px; cursor: pointer;
  font-size: 13px; color: var(--text);
  border: 1px solid var(--border); border-radius: var(--radius-sm);
  padding: 10px 12px; background: var(--surface); transition: border-color .15s, background .15s;
}
.inline .radio-option { flex: 1; min-width: 130px; }
.radio-icon { color: var(--primary); margin-top: 1px; flex-shrink: 0; }
.radio-option:hover { border-color: var(--text-3); }
.radio-option.active { border-color: var(--primary); background: var(--primary-50); }
.radio-option.disabled { opacity: .55; cursor: not-allowed; }
.radio-option input { position: absolute; opacity: 0; width: 1px; height: 1px; }
.dot {
  width: 16px; height: 16px; border: 1px solid var(--border-strong); border-radius: 50%;
  flex-shrink: 0; display: inline-flex; align-items: center; justify-content: center; margin-top: 1px;
}
.active .dot { border-color: var(--primary); }
.dot::after { content: ''; width: 8px; height: 8px; border-radius: 50%; background: var(--primary); transform: scale(0); transition: transform .15s; }
.active .dot::after { transform: scale(1); }
.text { display: flex; flex-direction: column; gap: 2px; }
.label { font-weight: 600; }
.description { font-size: 11.5px; color: var(--text-3); }
.radio-option:focus-within { box-shadow: 0 0 0 3px var(--primary-50); }
</style>
