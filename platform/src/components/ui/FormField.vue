<script setup>
import { t } from '../../i18n'
defineProps({
  label: { type: String, default: '' },
  required: { type: Boolean, default: false },
  hint: { type: String, default: '' },
  error: { type: String, default: '' },
  for: { type: String, default: undefined },
})
</script>

<template>
  <div class="form-field" :class="{ 'has-error': !!error }">
    <label v-if="label" class="field-label" :for="for">
      {{ label }}<span v-if="required" class="required">*</span>
    </label>
    <slot />
    <p v-if="error" class="field-error">{{ error }}</p>
    <p v-else-if="hint" class="field-hint">{{ hint }}</p>
  </div>
</template>

<style scoped>
.form-field { display: flex; flex-direction: column; gap: 6px; min-width: 0; }
.field-label { font-size: 12.5px; font-weight: 650; color: var(--text-2); letter-spacing: .01em; }
.required { color: var(--danger); margin-left: 3px; }
.field-hint { font-size: 11.5px; color: var(--text-3); line-height: 1.4; }
.field-error { font-size: 11.5px; color: var(--danger); line-height: 1.4; }
</style>
