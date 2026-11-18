<script setup>
import { t } from '../../i18n'
import { computed, ref } from 'vue'

let uid = 0

const props = defineProps({
  modelValue: { type: [String, Number], default: '' },
  type: { type: String, default: 'text' },
  placeholder: { type: String, default: '' },
  disabled: { type: Boolean, default: false },
  readonly: { type: Boolean, default: false },
  clearable: { type: Boolean, default: false },
  error: { type: Boolean, default: false },
  size: { type: String, default: 'md' }, // sm | md
  autocomplete: { type: String, default: undefined },
  min: { type: [String, Number], default: undefined },
  max: { type: [String, Number], default: undefined },
  step: { type: [String, Number], default: undefined },
  align: { type: String, default: 'left' }, // left | right
})
const emit = defineEmits(['update:modelValue', 'blur', 'focus', 'enter', 'clear'])
const id = `ui-input-${++uid}`

const filled = computed(() => props.modelValue !== '' && props.modelValue !== null && props.modelValue !== undefined)

function onInput(e) {
  emit('update:modelValue', props.type === 'number' && e.target.value !== '' ? Number(e.target.value) : e.target.value)
}
const inputEl = ref(null)
function focus() { inputEl.value?.focus() }

function clear() {
  emit('update:modelValue', '')
  emit('clear')
}

defineExpose({ focus })
</script>

<template>
  <div class="ui-input" :class="[`size-${size}`, { error, disabled, filled }]">
    <span v-if="$slots.prefix" class="prefix"><slot name="prefix" /></span>
    <input
      ref="inputEl"
      :id="id"
      :value="modelValue"
      :type="type"
      :placeholder="placeholder"
      :disabled="disabled"
      :readonly="readonly"
      :autocomplete="autocomplete"
      :min="min"
      :max="max"
      :step="step"
      :class="{ 'text-right': align === 'right' }"
      @input="onInput"
      @blur="emit('blur', $event)"
      @focus="emit('focus', $event)"
      @keyup.enter="emit('enter', $event)"
    />
    <button v-if="clearable && filled && !disabled && !readonly" type="button" class="clear" :aria-label="t('Clear')" @click="clear">×</button>
    <span v-else-if="$slots.suffix" class="suffix"><slot name="suffix" /></span>
  </div>
</template>

<style scoped>
.ui-input {
  display: flex; align-items: center; gap: 8px;
  width: 100%; min-height: 38px; background: var(--surface);
  border: 1px solid var(--border-strong); border-radius: var(--radius-sm);
  padding: 0 11px; font-size: 13.5px;
  transition: border-color .15s, box-shadow .15s;
}
.ui-input:hover { border-color: var(--text-3); }
.ui-input:focus-within { border-color: var(--primary); box-shadow: 0 0 0 3px var(--primary-50); }
.ui-input.error { border-color: var(--danger); }
.ui-input.disabled { background: var(--surface-2); opacity: .65; }
.ui-input input {
  flex: 1; min-width: 0; border: 0; outline: 0; background: transparent;
  color: var(--text); font: inherit; font-size: 13.5px; line-height: 1.4; padding: 0;
}
.ui-input input::placeholder { color: var(--text-3); }
.ui-input input.text-right { text-align: right; }
.ui-input.size-sm { min-height: 32px; padding: 0 9px; font-size: 12.5px; }
.ui-input.size-sm input { font-size: 12.5px; }
.prefix, .suffix { display: flex; align-items: center; color: var(--text-3); font-size: 13px; flex-shrink: 0; }
.clear {
  border: 0; background: none; color: var(--text-3); cursor: pointer;
  font-size: 18px; line-height: 1; padding: 0 2px;
}
.clear:hover { color: var(--text); }
</style>
