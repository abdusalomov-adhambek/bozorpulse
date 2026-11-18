<script setup>
const { t } = useI18n()
const props = defineProps({
  samples: { type: Array, default: () => [] },
  stats: { type: Object, default: () => ({}) },
  data_status: { type: String, default: '' },
  loading: { type: Boolean, default: false },
  unavailable: { type: Boolean, default: false }
})

function fmt(value) {
  if (value === null || value === undefined || value === '') return '—'
  const n = Number(value)
  if (Number.isNaN(n)) return '—'
  return `${new Intl.NumberFormat('en-US', { maximumFractionDigits: 2 }).format(n)} sum`
}

function fmtDate(value) {
  if (!value) return '—'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return '—'
  return d.toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric'
  })
}

function effective(row) {
  return row.effective_price !== null && row.effective_price !== undefined
    ? row.effective_price
    : row.price
}
</script>

<template>
  <div class="price-table-wrap">
    <!-- Loading -->
    <div v-if="loading" class="table-state" role="status" aria-live="polite">
      <span class="spinner" aria-hidden="true"></span>
      <p>Loading live prices…</p>
    </div>

    <!-- Backend offline / fetch failed -->
    <div v-else-if="unavailable" class="table-state table-state--error">
      <span class="state-icon" aria-hidden="true">!</span>
      <h3>Prices unavailable</h3>
      <p>The market price service could not be reached. Please try again shortly.</p>
    </div>

    <!-- Empty -->
    <div v-else-if="!samples || samples.length === 0" class="table-state">
      <span class="state-icon state-icon--muted" aria-hidden="true">⌕</span>
      <h3>No offers found</h3>
      <p>Try a different search term or choose another category.</p>
    </div>

    <template v-else>
      <div v-if="data_status === 'demo'" class="demo-banner">
        <span class="badge badge-demo">Sample/demo data</span>
        <span>These figures are illustrative and are not live supplier quotes.</span>
      </div>

      <div class="table-scroll">
        <table class="price-table">
          <thead>
            <tr>
              <th scope="col">{{ t("Product") }}</th>
              <th scope="col">{{ t("Supplier") }}</th>
              <th scope="col">{{ t("Region") }}</th>
              <th scope="col" class="num">{{ t("Price") }}</th>
              <th scope="col">{{ t("Discount") }}</th>
              <th scope="col" class="num">{{ t("Effective price") }}</th>
              <th scope="col">{{ t("Updated") }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, index) in samples" :key="row.product_id || `row-${index}`">
              <td>
                <span class="product-name">{{ row.product_name || '—' }}</span>
              </td>
              <td>{{ row.supplier_name || '—' }}</td>
              <td>{{ row.region_name || '—' }}</td>
              <td class="num">{{ fmt(row.price) }}</td>
              <td>
                <span v-if="row.discount_percent" class="badge badge-discount">
                  −{{ fmt(row.discount_percent) }}%
                </span>
                <span v-else class="muted">—</span>
              </td>
              <td class="num strong">{{ fmt(effective(row)) }}</td>
              <td class="muted small">{{ fmtDate(row.updated_at) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>
  </div>
</template>
