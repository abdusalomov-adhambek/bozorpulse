<script setup>
import { t } from '../../i18n'
import { computed } from 'vue'
import AppIcon from '../AppIcon.vue'

const props = defineProps({
  modelValue: { type: [Boolean, Array], default: false },
  value: { type: [String, Number, Boolean], default: undefined },
  label: { type: String, default: '' },
  description: { type: String, default: '' },
  disabled: { type: Boolean, default: false },
  indeterminate: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue', 'change'])

const checked = computed(() => {
  if (Array.isArray(props.modelValue) && props.value !== undefined) return props.modelValue.includes(props.value)
  return !!props.modelValue
})

function change(e) {
  const isChecked = e.target.checked
  if (Array.isArray(props.modelValue) && props.value !== undefined) {
    const next = isChecked
      ? [...props.modelValue, props.value]
      : props.modelValue.filter((v) => v !== props.value)
    emit('update:modelValue', next)
    emit('change', next)
  } else {
    emit('update:modelValue', isChecked)
    emit('change', isChecked)
  }
}
</script>

<template>
  <label class="ui-check" :class="{ disabled, checked }">
    <input
      type="checkbox"
      :checked="checked"
      :disabled="disabled"
      :indeterminate.prop="indeterminate"
      @change="change"
    />
    <span class="box"><AppIcon v-if="indeterminate" name="minus" :size="12" class="mark" /><AppIcon v-else-if="checked" name="check" :size="12" class="mark" /></span>
    <span v-if="label || description" class="text">
      <span v-if="label" class="label">{{ label }}</span>
      <span v-if="description" class="description">{{ description }}</span>
    </span>
    <slot />
  </label>
</template>

<style scoped>
.ui-check { display: inline-flex; align-items: flex-start; gap: 9px; cursor: pointer; font-size: 13px; color: var(--text); }
.ui-check.disabled { opacity: .55; cursor: not-allowed; }
.ui-check input { position: absolute; opacity: 0; width: 1px; height: 1px; }
.box {
  width: 17px; height: 17px; border: 1px solid var(--border-strong); border-radius: 5px;
  background: var(--surface); display: inline-flex; align-items: center; justify-content: center;
  flex-shrink: 0; margin-top: 1px; transition: background .15s, border-color .15s;
}
.checked .box { background: var(--primary); border-color: var(--primary); }
.mark { color: #fff; font-size: 11px; line-height: 1; opacity: 0; }
.checked .mark, .ui-check:has(input:indeterminate) .mark { opacity: 1; }
.text { display: flex; flex-direction: column; gap: 2px; }
.label { font-weight: 550; }
.description { font-size: 11.5px; color: var(--text-3); line-height: 1.4; }
.ui-check:focus-within .box { box-shadow: 0 0 0 3px var(--primary-50); border-color: var(--primary); }
</style>
