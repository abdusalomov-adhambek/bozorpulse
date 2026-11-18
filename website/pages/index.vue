<script setup>
const { t } = useI18n()
const platformUrl = 'http://localhost:3000'
const config = useRuntimeConfig()
const apiBase = config.public.apiBase

const selectedCategory = ref('')

// Categories for the filter — never let a failed request break the page.
const { data: categoriesData } = useAsyncData('home-categories', () =>
  $fetch(`${apiBase}/categories`).catch(() => ({ data: [] }))
)
const categories = computed(() => categoriesData.value?.data || [])

// Live market prices. Refetches when the category filter changes.
const { data: priceData, pending: pricePending } = useAsyncData(
  'home-prices',
  () =>
    $fetch(`${apiBase}/market/prices`, {
      query: selectedCategory.value ? { category_id: selectedCategory.value } : {}
    }).catch(() => null),
  { watch: [selectedCategory] }
)

const steps = [
  {
    n: '01',
    title: 'Publish supply',
    text: 'Suppliers list products with their own prices, discounts and stock levels.'
  },
  {
    n: '02',
    title: 'Compare the market',
    text: 'Customers browse offers side by side with real price statistics for each category.'
  },
  {
    n: '03',
    title: 'Get verified',
    text: 'Registration adds INN and bank details; an admin approves every new account.'
  },
  {
    n: '04',
    title: 'Create an order',
    text: 'Approved customers create orders directly with the supplier of their choice.'
  }
]

const capabilities = [
  {
    icon: 'pulse',
    title: 'Price intelligence',
    text: 'Min, median and average prices from live supplier offers — with source and sample size always visible.',
    tag: 'Market Pulse',
    size: 'wide',
  },
  {
    icon: 'search',
    title: 'Supplier discovery',
    text: 'Find the right supplier by product, region and price in seconds.',
    tag: 'Matching',
    size: '',
  },
  {
    icon: 'sparkle',
    title: 'AI assistant',
    text: 'Ask in plain language and get answers grounded in platform data — never invented.',
    tag: 'Assistant',
    size: '',
  },
]

const roles = [
  {
    name: 'Supplier',
    icon: 'box',
    text: 'Publish products, set prices and discounts, manage warehouses and receive orders.',
    tone: 'supplier'
  },
  {
    name: 'Customer',
    icon: 'cart',
    text: 'Browse offers, compare market prices, find suppliers and create purchase orders.',
    tone: 'customer'
  },
  {
    name: 'Admin',
    icon: 'shield',
    text: 'Review registrations, approve accounts, and keep categories and the marketplace healthy.',
    tone: 'admin'
  }
]
</script>

<template>
  <div>
    <!-- HERO -->
    <section class="hero">
      <div class="hero-grid" aria-hidden="true"></div>
      <div class="hero-glow hero-glow--one" aria-hidden="true"></div>
      <div class="hero-glow hero-glow--two" aria-hidden="true"></div>
      <div class="container hero-inner">
        <div class="hero-copy">
          <span class="eyebrow eyebrow-light"><span class="pulse-dot"></span>{{ t("AI-powered B2B marketplace") }}</span>
          <h1 class="hero-title">{{ t("See real market prices.") }}<br />
            <span class="gradient-text">{{ t("Sell to more buyers.") }}</span>
          </h1>
          <p class="hero-sub">{{ t("BozorPulse connects suppliers and customers in one transparent marketplace. Publish your offers or compare live prices \u2014 and turn market uncertainty into confident decisions.") }}</p>
          <div class="hero-actions">
            <a class="btn btn-primary btn-lg" :href="platformUrl" target="_blank" rel="noopener">{{ t("Open platform") }}<span aria-hidden="true">↗</span>
            </a>
            <NuxtLink class="btn btn-ghost-light btn-lg" to="/prices">{{ t("Explore market prices") }}</NuxtLink>
          </div>
          <div class="hero-chips">
            <span><AppIcon name="check" :size="13" />{{ t("Admin-approved accounts") }}</span>
            <span><AppIcon name="check" :size="13" />{{ t("Supplier-owned prices") }}</span>
            <span><AppIcon name="check" :size="13" />{{ t("Orders only in MVP") }}</span>
          </div>
          <a class="scroll-cue" href="#market-prices" aria-label="Scroll to market prices">
            <span class="scroll-line"></span>{{ t("Explore live prices") }}</a>
        </div>

        <div class="hero-visual" aria-hidden="true">
          <div class="mock-card">
            <div class="mock-head">
              <span class="mock-dot"></span><span class="mock-dot"></span><span class="mock-dot"></span>
              <span class="mock-head-label">live market feed</span>
            </div>
            <div class="mock-title">Market overview <span class="mock-live">● LIVE</span></div>
            <div class="mock-stats">
              <div class="mock-stat"><span>Median</span><strong>1,240</strong></div>
              <div class="mock-stat"><span>Average</span><strong>1,318</strong></div>
              <div class="mock-stat"><span>Offers</span><strong>128</strong></div>
            </div>
            <div class="mock-rows">
              <div class="mock-row"><span>Cement M500</span><b>1,180</b></div>
              <div class="mock-row"><span>Steel rebar</span><b>2,450</b></div>
              <div class="mock-row"><span>Brick M150</span><b>860</b></div>
              <div class="mock-row"><span>Sand, washed</span><b>540</b></div>
            </div>
            <div class="mock-bar"><span></span></div>
          </div>
        </div>
      </div>
    </section>

    <!-- TRUST / STATS BAND -->
    <section class="trust-band">
      <div class="container trust-grid">
        <div class="trust-item"><strong>2</strong><span>{{ t("sides of one marketplace") }}</span></div>
        <div class="trust-item"><strong>3</strong><span>{{ t("role-based cabinets") }}</span></div>
        <div class="trust-item"><strong>100%</strong><span>{{ t("admin-approved accounts") }}</span></div>
        <div class="trust-item"><strong>Live</strong><span>{{ t("supplier price data") }}</span></div>
      </div>
    </section>

    <!-- HOW IT WORKS -->
    <section class="section">
      <div class="container">
        <div class="section-head center">
          <span class="eyebrow">{{ t("How it works") }}</span>
          <h2 class="section-title">{{ t("From listing to order in four steps") }}</h2>
          <p class="section-lead">{{ t("A simple, transparent flow built around real prices and verified participants.") }}</p>
        </div>
        <div class="steps-grid">
          <article v-for="step in steps" :key="step.n" class="step-card">
            <span class="step-number">{{ step.n }}</span>
            <h3>{{ t(step.title) }}</h3>
            <p>{{ t(step.text) }}</p>
          </article>
        </div>
      </div>
    </section>

    <!-- CAPABILITIES BENTO -->
    <section class="section section-soft" id="capabilities">
      <div class="container">
        <div class="section-head center">
          <span class="eyebrow">{{ t("Built for better decisions") }}</span>
          <h2 class="section-title">{{ t("Intelligence, not just a catalogue") }}</h2>
          <p class="section-lead">{{ t("Every number is traceable to a supplier offer. Every suggestion explains itself.") }}</p>
        </div>
        <div class="bento-grid">
          <RevealSection
            v-for="(cap, i) in capabilities"
            :key="cap.title"
            as="article"
            class="bento-card"
            :class="{ 'bento-wide': cap.size === 'wide' }"
            :delay="i * 70"
          >
            <span class="bento-tag">{{ cap.tag }}</span>
            <span class="bento-icon" aria-hidden="true"><AppIcon :name="cap.icon" :size="24" /></span>
            <h3>{{ t(cap.title) }}</h3>
            <p>{{ t(cap.text) }}</p>
            <span class="bento-arrow" aria-hidden="true">↗</span>
          </RevealSection>
        </div>
      </div>
    </section>

    <!-- LIVE MARKET PRICES -->
    <section class="section section-soft" id="market-prices">
      <div class="container">
        <div class="section-head row-head">
          <div>
            <span class="eyebrow"><span class="pulse-dot"></span>{{ t("Live from the API") }}</span>
            <h2 class="section-title">{{ t("Live market prices") }}</h2>
            <p class="section-lead">{{ t("Real product prices submitted by suppliers. Filter by category to compare the market.") }}</p>
          </div>
          <NuxtLink class="btn btn-outline btn-sm" to="/prices">{{ t("Open full explorer \u2192") }}</NuxtLink>
        </div>

        <div class="chip-filter" role="group" aria-label="Filter prices by category">
          <button
            type="button"
            class="filter-chip"
            :class="{ active: !selectedCategory }"
            @click="selectedCategory = ''"
          >{{ t("All categories") }}</button>
          <button
            v-for="cat in categories"
            :key="cat.id"
            type="button"
            class="filter-chip"
            :class="{ active: selectedCategory === cat.id }"
            @click="selectedCategory = cat.id"
          >{{ cat.name }}</button>
        </div>

        <PriceTable
          :samples="priceData?.samples || []"
          :stats="priceData?.stats || {}"
          :data_status="priceData?.data_status || ''"
          :loading="pricePending"
          :unavailable="!pricePending && !priceData"
        />
      </div>
    </section>

    <!-- FOR SUPPLIERS -->
    <section class="section" id="for-suppliers">
      <div class="container split">
        <div class="split-copy">
          <span class="eyebrow">{{ t("For suppliers") }}</span>
          <h2 class="section-title">{{ t("Put your products in front of more buyers") }}</h2>
          <p class="section-lead">{{ t("Stop relying on word of mouth. Publish your catalogue once and let the marketplace bring qualified customers to you.") }}</p>
          <ul class="feature-list">
            <li><span class="check"><AppIcon name="check" :size="13" /></span>{{ t("Publish products with your own prices and discounts") }}</li>
            <li><span class="check"><AppIcon name="check" :size="13" /></span>{{ t("Show stock and regions so buyers know what is available") }}</li>
            <li><span class="check"><AppIcon name="check" :size="13" /></span>{{ t("Appear in supplier search and market price comparisons") }}</li>
            <li><span class="check"><AppIcon name="check" :size="13" /></span>{{ t("Manage orders from your dedicated supplier cabinet") }}</li>
          </ul>
          <a class="btn btn-primary" :href="platformUrl" target="_blank" rel="noopener">{{ t("Open supplier cabinet \u2197") }}</a>
        </div>
        <div class="split-visual visual-supplier">
          <div class="visual-card">
            <span class="visual-icon">▣</span>
            <h3>{{ t("Your catalogue, your rules") }}</h3>
            <p>{{ t("Prices, discounts, warehouses and stock \u2014 all controlled from your cabinet.") }}</p>
            <div class="mini-chips"><span>{{ t("Products") }}</span><span>{{ t("Warehouses") }}</span><span>{{ t("Orders") }}</span></div>
          </div>
        </div>
      </div>
    </section>

    <!-- FOR CUSTOMERS -->
    <section class="section section-soft" id="for-customers">
      <div class="container split split-reverse">
        <div class="split-copy">
          <span class="eyebrow">{{ t("For customers") }}</span>
          <h2 class="section-title">{{ t("Buy with a clear view of the market") }}</h2>
          <p class="section-lead">{{ t("Know what things really cost. Compare supplier offers against market statistics and choose with confidence.") }}</p>
          <ul class="feature-list">
            <li><span class="check"><AppIcon name="check" :size="13" /></span>{{ t("Browse offers and filter by category and search") }}</li>
            <li><span class="check"><AppIcon name="check" :size="13" /></span>{{ t("See min, median and average market prices") }}</li>
            <li><span class="check"><AppIcon name="check" :size="13" /></span>{{ t("Discover verified suppliers by region and rating") }}</li>
            <li><span class="check"><AppIcon name="check" :size="13" /></span>{{ t("Create orders directly from your customer cabinet") }}</li>
          </ul>
          <a class="btn btn-primary" :href="platformUrl" target="_blank" rel="noopener">{{ t("Open customer cabinet \u2197") }}</a>
        </div>
        <div class="split-visual visual-customer">
          <div class="visual-card">
            <span class="visual-icon">◎</span>
            <h3>{{ t("Compare before you commit") }}</h3>
            <p>{{ t("Live pricing and supplier details in one place, so every order is informed.") }}</p>
            <div class="mini-chips"><span>Market prices</span><span>{{ t("Suppliers") }}</span><span>{{ t("Orders") }}</span></div>
          </div>
        </div>
      </div>
    </section>

    <!-- ROLES & APPROVAL -->
    <section class="section" id="approval">
      <div class="container">
        <div class="section-head center">
          <span class="eyebrow">{{ t("Three roles, one workflow") }}</span>
          <h2 class="section-title">{{ t("A marketplace built on trust") }}</h2>
          <p class="section-lead">{{ t("Every participant has a role and a cabinet. New accounts are reviewed before they can trade, keeping offers and buyers verified.") }}</p>
        </div>
        <div class="role-grid">
          <article v-for="role in roles" :key="role.name" class="role-card" :class="`role-${role.tone}`">
            <span class="role-icon" aria-hidden="true"><AppIcon :name="role.icon" :size="22" /></span>
            <h3>{{ t(role.name) }}</h3>
            <p>{{ t(role.text) }}</p>
          </article>
        </div>

        <div class="workflow">
          <h3 class="workflow-title">{{ t("The approval workflow") }}</h3>
          <ol class="workflow-list">
            <li><strong>{{ t("Sign up") }}</strong>{{ t("with a username and password.") }}</li>
            <li><strong>{{ t("Add your details") }}</strong>{{ t("\u2014 suppliers and customers provide INN and bank account.") }}</li>
            <li><strong>{{ t("Admin review") }}</strong>{{ t("\u2014 an administrator approves the account.") }}</li>
            <li><strong>{{ t("Start trading") }}</strong>{{ t("inside your role cabinet once approved.") }}</li>
          </ol>
        </div>
      </div>
    </section>

    <!-- FINAL CTA -->
    <section class="cta-band">
      <div class="container cta-inner">
        <div>
          <h2>{{ t("Ready to see what the market really looks like?") }}</h2>
          <p>{{ t("Explore live prices or open your cabinet and get started today.") }}</p>
        </div>
        <div class="cta-actions">
          <a class="btn btn-light btn-lg" :href="platformUrl" target="_blank" rel="noopener">Open platform ↗</a>
          <NuxtLink class="btn btn-outline-light btn-lg" to="/contact">{{ t("Contact us") }}</NuxtLink>
        </div>
      </div>
    </section>
  </div>
</template>
