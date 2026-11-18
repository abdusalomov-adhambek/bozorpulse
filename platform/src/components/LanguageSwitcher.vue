<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { LOCALES, locale, setLocale, t } from '../i18n'
import AppIcon from './AppIcon.vue'

const open = ref(false)
const root = ref(null)

function choose(code) {
  setLocale(code)
  open.value = false
}
function onDocClick(e) {
  if (root.value && !root.value.contains(e.target)) open.value = false
}
onMounted(() => document.addEventListener('click', onDocClick))
onBeforeUnmount(() => document.removeEventListener('click', onDocClick))
</script>

<template>
  <div ref="root" class="lang-switch">
    <button type="button" class="lang-btn" :aria-label="t('Language') + ': ' + locale" @click="open = !open">
      <svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3a15 15 0 0 1 0 18M12 3a15 15 0 0 0 0 18"/></svg>
      <span>{{ locale.toUpperCase() }}</span>
    </button>
    <transition name="lang">
      <div v-if="open" class="lang-menu">
        <button
          v-for="l in LOCALES"
          :key="l.code"
          type="button"
          :class="{ active: l.code === locale }"
          @click="choose(l.code)"
        >
          <span class="lang-code">{{ l.short }}</span>
          <span>{{ l.label }}</span>
          <AppIcon v-if="l.code === locale" name="check" :size="13" class="lang-check" />
        </button>
      </div>
    </transition>
  </div>
</template>

<style scoped>
.lang-switch { position: relative; }
.lang-btn {
  display: inline-flex; align-items: center; gap: 6px; height: 36px;
  padding: 0 9px; border: 1px solid var(--border); border-radius: var(--radius-sm);
  background: var(--surface); color: var(--text-2); font: inherit; font-size: 12px;
  font-weight: 700; cursor: pointer; transition: border-color .15s, color .15s;
}
.lang-btn:hover { border-color: var(--primary); color: var(--primary); }
.lang-menu {
  position: absolute; right: 0; top: calc(100% + 6px); z-index: 70;
  min-width: 158px; padding: 5px; background: var(--surface);
  border: 1px solid var(--border); border-radius: var(--radius-sm); box-shadow: var(--shadow);
}
.lang-menu button {
  width: 100%; display: flex; align-items: center; gap: 9px;
  border: 0; background: none; border-radius: 7px; padding: 8px 9px;
  font: inherit; font-size: 12.5px; color: var(--text); cursor: pointer; text-align: left;
}
.lang-menu button:hover { background: var(--surface-2); }
.lang-menu button.active { color: var(--primary); background: var(--primary-50); }
.lang-code { font-size: 10px; font-weight: 800; color: var(--text-3); width: 20px; }
.lang-check { margin-left: auto; }
.lang-enter-active, .lang-leave-active { transition: opacity .12s, transform .12s; }
.lang-enter-from, .lang-leave-to { opacity: 0; transform: translateY(-4px); }
</style>
