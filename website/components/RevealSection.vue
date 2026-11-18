<script setup>
const { t } = useI18n()
import { onMounted, ref } from 'vue'

defineProps({
  as: { type: String, default: 'section' },
  delay: { type: Number, default: 0 },
})

const el = ref(null)
const visible = ref(false)

onMounted(() => {
  if (!el.value) return
  if (typeof IntersectionObserver === 'undefined') {
    visible.value = true
    return
  }
  const observer = new IntersectionObserver(
    ([entry]) => {
      if (entry.isIntersecting) {
        visible.value = true
        observer.disconnect()
      }
    },
    { threshold: 0.06, rootMargin: '0px 0px -40px 0px' },
  )
  observer.observe(el.value)
})
</script>

<template>
  <component
    :is="as"
    ref="el"
    class="reveal"
    :class="{ 'is-visible': visible }"
    :style="{ transitionDelay: `${delay}ms` }"
  >
    <slot />
  </component>
</template>

<style scoped>
.reveal {
  opacity: 0;
  transform: translateY(18px);
  transition: opacity 0.55s cubic-bezier(0.22, 1, 0.36, 1),
    transform 0.55s cubic-bezier(0.22, 1, 0.36, 1);
}
.reveal.is-visible { opacity: 1; transform: none; }
@media (prefers-reduced-motion: reduce) {
  .reveal { opacity: 1; transform: none; transition: none; }
}
</style>
