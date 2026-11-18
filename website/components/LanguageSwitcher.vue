<script setup>
import { computed } from 'vue'
import { LOCALES, locale, setLocale } from '#imports'
function isActive(code) { return locale.value === code }
</script>

<template>
  <div class="site-lang" role="group" aria-label="Language">
    <button
      v-for="l in LOCALES"
      :key="l.code"
      type="button"
      class="site-lang-btn"
      :class="{ active: isActive(l.code) }"
      :title="l.label"
      :data-lang="l.code"
    >{{ l.short }}</button>
  </div>
</template>

<style scoped>
.site-lang {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 3px;
  border: 1px solid var(--line-strong);
  border-radius: var(--radius-pill);
  background: rgba(255, 255, 255, 0.7);
}
.site-lang-btn {
  min-width: 28px;
  height: 26px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--ink-faint);
  font: inherit;
  font-size: 10.5px;
  font-weight: 800;
  cursor: pointer;
  transition: background 0.16s, color 0.16s;
}
.site-lang-btn:hover { color: var(--brand); }
.site-lang-btn.active { background: var(--brand); color: #fff; }
</style>
