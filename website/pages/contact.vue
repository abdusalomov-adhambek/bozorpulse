<script setup>
const { t } = useI18n()
const platformUrl = 'http://localhost:3000'

const faqs = [
  {
    q: 'What is BozorPulse?',
    a: 'An AI-powered B2B marketplace that connects suppliers and customers. Suppliers publish products with their own prices and discounts; customers browse offers, see market price statistics, find suppliers and create orders.'
  },
  {
    q: 'Who can use the platform?',
    a: 'There are three roles: supplier, customer and admin. Each role has its own cabinet with the tools relevant to it.'
  },
  {
    q: 'Why do I need an INN and bank account?',
    a: 'Suppliers and customers provide an INN and bank account during registration so the business behind the account is identifiable. An admin reviews and approves every new account before it can be used.'
  },
  {
    q: 'Does BozorPulse process payments or delivery?',
    a: 'Not in the MVP. The platform currently stops at order creation — there is no payment processing, delivery tracking or automated purchasing yet.'
  },
  {
    q: 'Where do the market prices come from?',
    a: 'They are aggregated from product offers published by approved suppliers. When the figures are illustrative, the price table is marked with a “Sample/demo data” badge.'
  }
]
</script>

<template>
  <div>
    <section class="page-hero">
      <div class="hero-glow hero-glow--one" aria-hidden="true"></div>
      <div class="container">
        <span class="eyebrow eyebrow-light">Contact &amp; FAQ</span>
        <h1 class="page-title">We’re here to help</h1>
        <p class="page-sub">Find quick answers below or reach out to the BozorPulse team.</p>
      </div>
    </section>

    <section class="section">
      <div class="container contact-grid">
        <div class="contact-main">
          <div class="section-head">
            <span class="eyebrow">Frequently asked questions</span>
            <h2 class="section-title">Quick answers</h2>
          </div>
          <div class="faq-list">
            <details v-for="faq in faqs" :key="faq.q" class="faq-item">
              <summary>{{ faq.q }}</summary>
              <p>{{ faq.a }}</p>
            </details>
          </div>
        </div>

        <aside class="contact-side">
          <div class="contact-card">
            <h3>Get in touch</h3>
            <p>For questions about registration, approval or using the marketplace, contact the team.</p>
            <div class="contact-row">
              <span class="contact-label">Email</span>
              <span>support@bozorpulse.local</span>
            </div>
            <div class="contact-row">
              <span class="contact-label">Platform</span>
              <a :href="platformUrl" target="_blank" rel="noopener">localhost:3000 ↗</a>
            </div>
            <div class="contact-row">
              <span class="contact-label">API</span>
              <span>localhost:8080/api/v1</span>
            </div>
            <a class="btn btn-primary btn-block" :href="platformUrl" target="_blank" rel="noopener">
              Open platform ↗
            </a>
          </div>
          <p class="contact-note">
            This website is the public face of BozorPulse. Account management, approvals and orders
            happen inside the platform application.
          </p>
        </aside>
      </div>
    </section>
  </div>
</template>
