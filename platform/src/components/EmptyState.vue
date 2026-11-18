<script setup>
import { t } from '../i18n'
import AppIcon from './AppIcon.vue'
defineProps({
  title: { type: String, default: 'Nothing here yet' },
  description: { type: String, default: '' },
  icon: { type: String, default: 'inbox' }
})
</script>

<template>
  <div class="empty">
    <div class="empty-icon"><AppIcon :name="icon" :size="30" /></div>
    <div class="empty-title">{{ title }}</div>
    <div v-if="description" class="empty-desc">{{ description }}</div>
    <slot />
  </div>
</template>
