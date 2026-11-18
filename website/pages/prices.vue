<script setup>
const { t } = useI18n()
const config = useRuntimeConfig()
const apiBase = config.public.apiBase

const q = ref('')
const selectedCategory = ref('')

const { data: categoriesData } = useAsyncData('prices-categories', () =>
  $fetch(`${apiBase}/categories`).catch(() => ({ data: [] }))
)
const categories = computed(() => categoriesData.value?.data || [])
const categoryOptions = computed(() => categories.value.map((c) => ({ value: c.id, label: c.name })))

const {
  data: priceData,
  pending: pricePending,
  refresh: refreshPrices
} = useAsyncData(
  'prices-data',
  () =>
    $fetch(`${apiBase}/market/prices`, {
      query: {
        ...(q.value ? { q: q.value } : {}),
        ...(selectedCategory.value ? { category_id: selectedCategory.value } : {})
      }
    }).catch(() => null),
  { watch: [q, selectedCategory] }
)

const samples = computed(() => priceData.value?.samples || [])
const stats = computed(() => priceData.value?.stats || {})
const dataStatus = computed(() => priceData.value?.data_status || '')

const statCards = computed(() => [
  { label: 'Minimum price', value: fmt(stats.value.min), hint: 'Lowest observed offer' },
  { label: 'Median price', value: fmt(stats.value.median), hint: 'Midpoint of the market' },
  { label: 'Average price', value: fmt(stats.value.avg), hint: 'Mean of all offers' },
  { label: 'Offers counted', value: fmt(stats.value.count), hint: 'Samples in this view' }
])

function fmt(value) {
  if (value === null || value === undefined || value === '') return '—'
  const n = Number(value)
  if (Number.isNaN(n)) return '—'
  return `${new Intl.NumberFormat('en-US', { maximumFractionDigits: 2 }).format(n)} sum`
}

function clearFilters() {
  q.value = ''
  selectedCategory.value = ''
}
</script>

<template>
  <div>
    <section class="page-hero">
      <div class="hero-glow hero-glow--one" aria-hidden="true"></div>
      <div class="container">
        <span class="eyebrow eyebrow-light"><span class="pulse-dot"></span>{{ t("Live from the API") }}</span>
        <h1 class="page-title">Market price explorer</h1>
        <p class="page-sub">
          Search supplier offers, filter by category, and see real market statistics in one place.
        </p>
      </div>
    </section>

    <section class="section">
      <div class="container">
        <!-- Stats -->
        <div class="stats-grid">
          <div v-for="card in statCards" :key="card.label" class="stat-card">
            <span class="stat-label">{{ card.label }}</span>
            <strong class="stat-value">{{ card.value }}</strong>
            <span class="stat-hint">{{ card.hint }}</span>
          </div>
        </div>

        <!-- Filters -->
        <div class="explorer-controls">
          <div class="field field-grow">
            <span class="field-label">Search products</span>
            <AppInput v-model="q" placeholder="e.g. cement, rebar, brick…" clearable />
          </div>
          <div class="field">
            <span class="field-label">Category</span>
            <AppSelect
              v-model="selectedCategory"
              :options="categoryOptions"
              :placeholder="t('All categories')"
              clearable
            />
          </div>
          <button class="btn btn-outline" type="button" @click="clearFilters">Clear</button>
          <button class="btn btn-soft" type="button" @click="refreshPrices()">{{ t("Refresh") }}</button>
        </div>

        <div class="result-head">
          <span v-if="dataStatus === 'demo'" class="badge badge-demo">Sample/demo data</span>
          <span class="muted small">
            {{ pricePending ? 'Updating…' : `${samples.length} offer${samples.length === 1 ? '' : 's'} shown` }}
          </span>
        </div>

        <PriceTable
          :samples="samples"
          :stats="stats"
          :data_status="dataStatus"
          :loading="pricePending"
          :unavailable="!pricePending && !priceData"
        />

        <aside class="provenance">
          <h3>Where these prices come from</h3>
          <p>
            Market prices are aggregated from product offers published by approved suppliers on
            BozorPulse. The minimum, median and average are calculated from the samples currently
            visible in your filter. When a <span class="badge badge-demo">Sample/demo data</span>
            badge appears, the figures are illustrative rather than live supplier quotes.
          </p>
          <p class="muted small">
            If the marketplace API is offline, this page keeps working and shows a clear
            “prices unavailable” state instead of empty or broken content.
          </p>
        </aside>
      </div>
    </section>
  </div>
</template>
