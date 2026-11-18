<script setup>
import { money } from '../../utils/money'
import AppIcon from '../../components/AppIcon.vue'
import { t } from '../../i18n'
import { computed, ref, onMounted } from 'vue'
import api from '../../api/client'
import { useAuthStore } from '../../stores/auth'
import StatCard from '../../components/StatCard.vue'
import DataTable from '../../components/DataTable.vue'
import PriceChart from '../../components/PriceChart.vue'
import StatusBadge from '../../components/StatusBadge.vue'
import EmptyState from '../../components/EmptyState.vue'
import { useAppStore } from '../../stores/app'

const app = useAppStore()
const auth = useAuthStore()
const loading = ref(false)
const error = ref('')
const analytics = ref(null)
const samples = ref([])
const tones = ['primary', 'success', 'info', 'warning', 'danger']
const icons = ['chart', 'orders', 'users', 'box', 'wallet']

const quickActions = computed(() => {
  if (auth.role === 'supplier') {
    return [
      { icon: 'box', title: t('Add a product'), desc: 'Publish a new offer with price and discount', to: '/supplier/products' },
      { icon: 'warehouse', title: t('Warehouses'), desc: 'Manage stock locations and availability', to: '/supplier/warehouses' },
      { icon: 'receipt', title: t('View orders'), desc: 'Review incoming purchase orders', to: '/supplier/orders' },
    ]
  }
  if (auth.role === 'customer') {
    return [
      { icon: 'cart', title: t('New order'), desc: 'Pick a supplier, product and warehouse', to: '/customer/orders' },
      { icon: 'users', title: t('Find suppliers'), desc: 'Browse approved sellers', to: '/customer/suppliers' },
      { icon: 'warehouse', title: t('Warehouses'), desc: 'Manage delivery destinations', to: '/customer/warehouses' },
    ]
  }
  return [
    { icon: 'check-circle', title: t('Review accounts'), desc: 'Approve pending suppliers and customers', to: '/admin/suppliers' },
    { icon: 'box', title: t('Manage catalogue'), desc: 'Products, categories and regions', to: '/admin/settings/products' },
    { icon: 'receipt', title: t('All orders'), desc: 'Track marketplace order activity', to: '/admin/orders' },
  ]
})

const orderColumns = [
  { key: 'id', label: t('Order') },
  { key: 'customer_id', label: t('Customer') },
  { key: 'status', label: t('Status') },
  { key: 'total_amount', label: t('Total') },
  { key: 'created_at', label: t('Date') }
]

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [a, market] = await Promise.all([
      api.get('/analytics/dashboard'),
      api.get('/market/prices').catch(() => null)
    ])
    analytics.value = a
    samples.value = market?.samples || []
  } catch (err) {
    error.value = err?.message || 'Failed to load dashboard data.'
  } finally {
    loading.value = false
  }
}

function marketEntries() {
  const summary = analytics.value?.market_summary
  if (!summary || typeof summary !== 'object') return []
  return Object.entries(summary).map(([key, value]) => ({
    key: key.replace(/_/g, ' '),
    value: typeof value === 'number' ? value.toLocaleString() : value
  }))
}

onMounted(load)
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t("Dashboard") }}</h1>
        <p class="page-desc">{{ t("Overview of marketplace activity and performance.") }}</p>
      </div>
      <button class="btn btn-secondary" type="button" :disabled="loading" @click="load">{{ t("Refresh") }}</button>
    </div>

    <div class="quick-grid mb-3">
      <RouterLink v-for="action in quickActions" :key="action.title" :to="action.to" class="quick-card">
        <span class="quick-icon"><AppIcon :name="action.icon" :size="18" aria-hidden="true" /></span>
        <span class="quick-copy">
          <strong>{{ action.title }}</strong>
          <small>{{ action.desc }}</small>
        </span>
        <span class="quick-arrow" aria-hidden="true">↗</span>
      </RouterLink>
    </div>

    <div v-if="error" class="alert alert-error mb-3">
      {{ error }}
      <button class="btn btn-sm btn-secondary" style="margin-left:10px" @click="load">{{ t("Retry") }}</button>
    </div>

    <template v-if="loading">
      <div class="grid grid-stats">
        <div v-for="n in 4" :key="n" class="card card-pad"><div class="loading-block"><span class="spinner" /></div></div>
      </div>
    </template>

    <template v-else-if="analytics">
      <div class="grid grid-stats mb-3">
        <StatCard
          v-for="(metric, i) in (analytics.metrics || [])"
          :key="metric.label"
          :label="metric.label"
          :value="metric.value"
          :delta="metric.delta"
          :icon="icons[i % icons.length]"
          :tone="tones[i % tones.length]"
        />
      </div>
      <EmptyState v-if="!(analytics.metrics || []).length" :title="t('No metrics yet')" description="Metrics will appear as activity is recorded." />

      <div class="grid grid-2 mb-3">
        <div class="card">
          <div class="card-head">
            <div><div class="card-title">{{ t('Recent orders') }}</div><div class="card-sub">{{ t('Latest activity across the platform') }}</div></div>
          </div>
          <DataTable :columns="orderColumns" :rows="analytics.recent_orders || []" :empty-text="t('No recent orders')">
            <template #cell-id="{ row }"><span class="cell-strong">#{{ row.id }}</span></template>
            <template #cell-customer_id="{ row }">Customer #{{ row.customer_id }}</template>
            <template #cell-status="{ row }"><StatusBadge :status="row.status" /></template>
            <template #cell-total_amount="{ row }"><span class="cell-strong mono">{{ money(row.total_amount || 0) }}</span></template>
            <template #cell-created_at="{ row }"><span class="subtle">{{ row.created_at ? new Date(row.created_at).toLocaleDateString() : '—' }}</span></template>
          </DataTable>
        </div>

        <div class="card">
          <div class="card-head"><div class="card-title">{{ t('Market summary') }}</div></div>
          <div class="card-body">
            <div v-if="marketEntries().length" class="kv-list">
              <div v-for="entry in marketEntries()" :key="entry.key" class="kv-row">
                <span class="kv-key">{{ entry.key }}</span>
                <span class="kv-val mono">{{ entry.value }}</span>
              </div>
            </div>
            <EmptyState v-else :title="t('No market summary')" description="Market statistics will show here once data is available." icon="trend" />
          </div>
        </div>
      </div>

      <div class="card">
        <div class="card-head"><div><div class="card-title">{{ t('Market price trend') }}</div><div class="card-sub">{{ t('Effective prices across listings') }}</div></div></div>
        <div class="card-body"><PriceChart :points="samples" /></div>
      </div>
    </template>

    <EmptyState v-else :title="t('No analytics data')" description="Analytics will appear once the platform has activity." icon="chart" />
  </div>
</template>
