<script setup>
const { t } = useI18n()
const platformUrl = 'http://localhost:3000'

const problemSolution = [
  {
    tag: 'The problem',
    title: 'Markets run on guesswork',
    text: 'Customers rarely know the real market price, so they overpay or hesitate. Suppliers, meanwhile, struggle to reach enough buyers to grow.',
    icon: 'search',
    tone: 'problem'
  },
  {
    tag: 'Our solution',
    title: 'One transparent marketplace',
    text: 'BozorPulse brings verified suppliers and customers together. Suppliers publish offers with their own prices and discounts; customers compare those offers against real market statistics.',
    icon: 'check-circle',
    tone: 'solution'
  }
]

const whyCards = [
  {
    title: 'INN',
    text: 'A company tax identifier confirms that a business is legitimate and makes suppliers and customers traceable.'
  },
  {
    title: 'Bank account',
    text: 'Bank details confirm the organisation behind the account and prepare it for real B2B settlement as the platform grows.'
  },
  {
    title: 'Admin approval',
    text: 'An administrator reviews every new registration before access is granted, keeping the marketplace trustworthy for all sides.'
  }
]
</script>

<template>
  <div>
    <section class="page-hero">
      <div class="hero-glow hero-glow--two" aria-hidden="true"></div>
      <div class="container">
        <span class="eyebrow eyebrow-light">{{ t("AI-powered B2B marketplace") }}</span>
        <h1 class="page-title">Making B2B markets transparent</h1>
        <p class="page-sub">
          We are building a place where real prices are visible to buyers and real opportunities
          are reachable for suppliers.
        </p>
      </div>
    </section>

    <!-- MISSION -->
    <section class="section">
      <div class="container narrow">
        <div class="section-head center">
          <span class="eyebrow">Our mission</span>
          <h2 class="section-title">Confident decisions for both sides of the market</h2>
          <p class="section-lead">
            BozorPulse is an AI-powered B2B marketplace connecting suppliers and customers. We
            believe fair trade starts with shared, reliable information — not private guesses.
          </p>
        </div>
      </div>
    </section>

    <!-- PROBLEM / SOLUTION -->
    <section class="section section-soft">
      <div class="container">
        <div class="problem-grid">
          <article v-for="item in problemSolution" :key="item.tag" class="problem-card" :class="`tone-${item.tone}`">
            <span class="problem-icon" aria-hidden="true"><AppIcon :name="item.icon" :size="24" /></span>
            <span class="eyebrow">{{ item.tag }}</span>
            <h3>{{ item.title }}</h3>
            <p>{{ item.text }}</p>
          </article>
        </div>
      </div>
    </section>

    <!-- WHY REQUIREMENTS -->
    <section class="section" id="approval">
      <div class="container">
        <div class="section-head center">
          <span class="eyebrow">Verification</span>
          <h2 class="section-title">Why INN, bank details and approval?</h2>
          <p class="section-lead">
            Registration asks for more than a username and password because B2B trade involves real
            organisations and real money.
          </p>
        </div>
        <div class="role-grid">
          <article v-for="card in whyCards" :key="card.title" class="role-card">
            <h3>{{ card.title }}</h3>
            <p>{{ card.text }}</p>
          </article>
        </div>
        <p class="workflow-note">
          Registration requires a username and password. Suppliers and customers additionally
          provide an INN and bank account. An admin approves every account before it can be used on
          the platform.
        </p>
      </div>
    </section>

    <!-- AUDIENCES -->
    <section class="section section-soft">
      <div class="container split">
        <div class="split-copy" id="for-suppliers">
          <span class="eyebrow">{{ t("For suppliers") }}</span>
          <h2 class="section-title">Reach more customers</h2>
          <p class="section-lead">
            Publish products, set your own prices and discounts, manage warehouses and stock, and
            fulfil orders from your supplier cabinet.
          </p>
        </div>
        <div class="split-copy" id="for-customers">
          <span class="eyebrow">{{ t("For customers") }}</span>
          <h2 class="section-title">Buy with market context</h2>
          <p class="section-lead">
            Browse offers, compare price statistics, find verified suppliers and create purchase
            orders from your customer cabinet.
          </p>
        </div>
      </div>
    </section>

    <!-- MVP SCOPE -->
    <section class="section" id="mvp">
      <div class="container narrow">
        <div class="scope-card">
          <span class="eyebrow">What the MVP includes</span>
          <h2 class="section-title">Focused on verified offers and order creation</h2>
          <p>
            The current version of BozorPulse covers supplier product publishing, customer browsing,
            market price statistics, supplier discovery, account approval, and order creation.
          </p>
          <p class="scope-limit">
            Out of scope for the MVP: payment processing, delivery tracking, and automated
            purchasing. These may be added in future releases.
          </p>
          <a class="btn btn-primary" :href="platformUrl" target="_blank" rel="noopener">Open platform ↗</a>
        </div>
      </div>
    </section>
  </div>
</template>
