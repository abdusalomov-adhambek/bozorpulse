<script setup>
const { t } = useI18n()
const platformUrl = 'http://localhost:3000'
const year = new Date().getFullYear()
</script>

<template>
  <footer class="site-footer">
    <div class="container footer-grid">
      <div class="footer-brand">
        <NuxtLink to="/" class="brand" aria-label="BozorPulse home">
          <span class="brand-mark" aria-hidden="true">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M2 12h4l2.5-7 4 14 3-9 2 2h4.5" />
            </svg>
          </span>
          <span class="brand-name">Bozor<span>Pulse</span></span>
        </NuxtLink>
        <p class="footer-tagline">
          An AI-powered B2B marketplace where suppliers publish real offers and buyers compare
          real market prices.
        </p>
        <a class="btn btn-outline btn-sm" :href="platformUrl" target="_blank" rel="noopener">
          Open platform ↗
        </a>
      </div>

      <div class="footer-col">
        <h3>Explore</h3>
        <NuxtLink to="/prices">Market prices</NuxtLink>
        <NuxtLink to="/about">{{ t("About BozorPulse") }}</NuxtLink>
        <NuxtLink to="/contact">Contact &amp; FAQ</NuxtLink>
        <a :href="platformUrl" target="_blank" rel="noopener">Platform cabinets</a>
      </div>

      <div class="footer-col">
        <h3>For you</h3>
        <NuxtLink to="/about#for-suppliers">{{ t("For suppliers") }}</NuxtLink>
        <NuxtLink to="/about#for-customers">{{ t("For customers") }}</NuxtLink>
        <NuxtLink to="/about#approval">Approval workflow</NuxtLink>
        <NuxtLink to="/about#mvp">MVP scope</NuxtLink>
      </div>

      <div class="footer-col">
        <h3>Platform</h3>
        <span class="footer-note">{{ t("Dashboard") }}</span>
        <span class="footer-note">{{ t("Suppliers") }}</span>
        <span class="footer-note">{{ t("Orders") }}</span>
        <a :href="platformUrl" target="_blank" rel="noopener">Sign in ↗</a>
      </div>
    </div>

    <div class="container footer-bottom">
      <p>© {{ year }} BozorPulse. All rights reserved.</p>
      <p>Built to make B2B markets transparent for everyone.</p>
    </div>
  </footer>
</template>
