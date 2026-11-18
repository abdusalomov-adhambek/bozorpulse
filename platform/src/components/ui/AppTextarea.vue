<script setup>
import { t } from '../../i18n'
defineProps({
  modelValue: { type: String, default: '' },
  placeholder: { type: String, default: '' },
  rows: { type: Number, default: 3 },
  disabled: { type: Boolean, default: false },
  error: { type: Boolean, default: false },
  maxlength: { type: Number, default: undefined },
})
const emit = defineEmits(['update:modelValue'])
</script>

<template>
  <textarea
    class="ui-textarea"
    :class="{ error, disabled }"
    :value="modelValue"
    :placeholder="placeholder"
    :rows="rows"
    :disabled="disabled"
    :maxlength="maxlength"
    @input="emit('update:modelValue', $event.target.value)"
  ></textarea>
</template>

<style scoped>
.ui-textarea {
  width: 100%; resize: vertical; padding: 9px 11px;
  background: var(--surface); border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm); color: var(--text);
  font: inherit; font-size: 13.5px; line-height: 1.5; outline: 0; transition: border-color .15s, box-shadow .15s;
}
.ui-textarea::placeholder { color: var(--text-3); }
.ui-textarea:hover { border-color: var(--text-3); }
.ui-textarea:focus { border-color: var(--primary); box-shadow: 0 0 0 3px var(--primary-50); }
.ui-textarea.error { border-color: var(--danger); }
.ui-textarea.disabled { background: var(--surface-2); opacity: .65; }
</style>
