<script setup>
import { t } from '../../i18n'
defineProps({
  accept: { type: String, default: 'image/*' },
  label: { type: String, default: 'Choose file' },
})
const emit = defineEmits(['change'])
</script>

<template>
  <label class="file-input">
    <slot>{{ label }}</slot>
    <input type="file" :accept="accept" @change="emit('change', $event)" />
  </label>
</template>

<style scoped>
.file-input {
  position: relative; overflow: hidden; display: inline-flex;
  align-items: center; justify-content: center; cursor: pointer;
}
.file-input input { position: absolute; inset: 0; opacity: 0; cursor: pointer; }
</style>
