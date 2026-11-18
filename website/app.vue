<script setup>
import { locale, setLocale } from '#imports'
import { onBeforeUnmount, onMounted } from 'vue'

// App-level delegation keeps the language switch reliable regardless of
// where the switcher is rendered.
function onLanguageClick(event) {
  const button = event.target?.closest?.('[data-lang]')
  if (button) setLocale(button.dataset.lang)
}
onMounted(() => {
  const saved = localStorage.getItem('bp_site_locale')
  if (saved && saved !== locale.value) setLocale(saved)
  document.addEventListener('click', onLanguageClick)
})
onBeforeUnmount(() => document.removeEventListener('click', onLanguageClick))
</script>

<template>
  <div class="site-shell">
    <SiteHeader />
    <main id="main-content" class="site-main">
      <NuxtPage />
    </main>
    <SiteFooter />
  </div>
</template>
