<script setup>
import { useI18n } from '#imports'
const { t } = useI18n()
const menuOpen = ref(false)
const platformUrl = 'http://localhost:3000'

const links = [
  { to: '/', label: 'Home' },
  { to: '/prices', label: 'Prices' },
  { to: '/about', label: 'About' },
  { to: '/contact', label: 'Contact' }
]
</script>

<template>
  <header class="site-header">
    <div class="container header-inner">
      <NuxtLink to="/" class="brand" aria-label="BozorPulse home" @click="menuOpen = false">
        <span class="brand-mark" aria-hidden="true">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M2 12h4l2.5-7 4 14 3-9 2 2h4.5" />
          </svg>
        </span>
        <span class="brand-name">Bozor<span>Pulse</span></span>
      </NuxtLink>

      <nav class="nav" :class="{ 'nav-open': menuOpen }" aria-label="Main navigation">
        <NuxtLink
          v-for="link in links"
          :key="link.to"
          :to="link.to"
          class="nav-link"
          @click="menuOpen = false"
        >
          {{ t(link.label) }}
        </NuxtLink>
        <ClientOnly><LanguageSwitcher /></ClientOnly>
        <a class="btn btn-primary nav-cta" :href="platformUrl" target="_blank" rel="noopener">
          {{ t('Open platform') }} <span aria-hidden="true">↗</span>
        </a>
      </nav>

      <button
        class="nav-toggle"
        type="button"
        :aria-expanded="menuOpen"
        aria-label="Toggle navigation"
        @click="menuOpen = !menuOpen"
      >
        <span class="nav-toggle-bar" :class="{ 'bar-top-open': menuOpen }"></span>
        <span class="nav-toggle-bar" :class="{ 'bar-mid-open': menuOpen }"></span>
        <span class="nav-toggle-bar" :class="{ 'bar-bottom-open': menuOpen }"></span>
      </button>
    </div>
  </header>
</template>
