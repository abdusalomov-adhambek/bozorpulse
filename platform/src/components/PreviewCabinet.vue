<script setup>
import { money } from '../utils/money'
import { computed, onMounted, ref } from 'vue'
import api from '../api/client'
import { t } from '../i18n'
import { NAV_CONFIG } from '../router'
import AppIcon from './AppIcon.vue'
import StatusBadge from './StatusBadge.vue'

const props = defineProps({
  role: { type: String, required: true }, // supplier | customer
})

const products = ref([])
const suppliers = ref([])
const orders = ref([])
const loading = ref(true)

const nav = computed(() => NAV_CONFIG[props.role] || [])
const roleLabel = computed(() => props.role === 'supplier' ? t('Supplier') : t('Customer'))

const metrics = computed(() => {
  if (props.role === 'supplier') {
    return [
      { label: t('Products'), value: products.value.length, icon: 'box' },
      { label: t('Orders'), value: orders.value.length, icon: 'receipt' },
      { label: t('Warehouses'), value: 2, icon: 'warehouse' },
    ]
  }
  return [
    { label: t('Suppliers'), value: suppliers.value.length, icon: 'users' },
    { label: t('Products'), value: products.value.length, icon: 'box' },
    { label: t('Orders'), value: orders.value.length, icon: 'receipt' },
  ]
})

onMounted(async () => {
  loading.value = true
  try {
    const [p, s, o] = await Promise.all([
      api.get('/products', { limit: 8 }, { auth: false }),
      api.get('/suppliers', { limit: 8 }, { auth: false }),
      api.get('/orders', { limit: 5 }),
    ])
    products.value = p?.data || []
    suppliers.value = s?.data || []
    orders.value = o?.data || []
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ roleLabel }} cabinet preview</h1>
        <p class="page-desc">Read-only view of what the {{ role }} workspace looks like to its users.</p>
      </div>
      <span class="preview-badge"><AppIcon name="shield" :size="13" />{{ t('Preview mode') }}</span>
    </div>

    <div class="preview-shell">
      <aside class="preview-side">
        <div class="preview-brand"><span class="preview-mark">BP</span> BozorPulse</div>
        <div class="preview-role">{{ roleLabel }} workspace</div>
        <div v-for="item in nav" :key="item.label" class="preview-nav">
          <AppIcon :name="item.icon === 'dashboard' ? 'chart' : item.icon === 'orders' ? 'receipt' : item.icon === 'warehouses' ? 'warehouse' : item.icon === 'settings' ? 'box' : 'box'" :size="15" />
          <span>{{ t(item.label) }}</span>
        </div>
      </aside>

      <div class="preview-main">
        <div v-if="loading" class="loading-block"><span class="spinner" /></div>
        <template v-else>
          <div class="preview-metrics">
            <div v-for="m in metrics" :key="m.label" class="preview-metric">
              <AppIcon :name="m.icon" :size="18" />
              <strong>{{ m.value }}</strong>
              <span>{{ m.label }}</span>
            </div>
          </div>

          <div class="preview-grid">
            <div class="preview-card">
              <h3>{{ role === 'supplier' ? 'Your products' : 'Suppliers to explore' }}</h3>
              <div v-if="role === 'supplier'">
                <div v-for="p in products.slice(0, 5)" :key="p.id" class="preview-row">
                  <span>{{ p.name }}</span>
                  <strong>{{ money(p.effective_price || p.price || 0) }}</strong>
                </div>
                <p v-if="!products.length" class="subtle">{{ t('No products available.') }}</p>
              </div>
              <div v-else>
                <div v-for="s in suppliers.slice(0, 5)" :key="s.id" class="preview-row">
                  <span>{{ s.company_name }}</span>
                  <strong>{{ s.region_name || '—' }}</strong>
                </div>
                <p v-if="!suppliers.length" class="subtle">{{ t('No suppliers available.') }}</p>
              </div>
            </div>

            <div class="preview-card">
              <h3>{{ t('Recent orders') }}</h3>
              <div v-for="o in orders.slice(0, 4)" :key="o.id" class="preview-row">
                <span>#{{ o.id }}</span>
                <StatusBadge :status="o.status" />
              </div>
              <p v-if="!orders.length" class="subtle">{{ t('No orders yet.') }}</p>
            </div>
          </div>

          <p class="preview-note">
            This preview uses live marketplace data but does not grant access to {{ role }} actions.
          </p>
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped>
.preview-badge {
  display: inline-flex; align-items: center; gap: 6px;
  border: 1px solid var(--primary); color: var(--primary);
  background: var(--primary-50); border-radius: 999px;
  padding: 6px 11px; font-size: 11.5px; font-weight: 700;
}
.preview-shell {
  display: grid; grid-template-columns: 190px 1fr; gap: 0;
  border: 1px solid var(--border); border-radius: var(--radius);
  overflow: hidden; background: var(--surface);
}
.preview-side {
  background: var(--text); color: #fff; padding: 16px 12px;
  display: flex; flex-direction: column; gap: 3px;
}
.preview-brand { display: flex; align-items: center; gap: 8px; font-weight: 800; padding: 2px 6px 12px; font-size: 13px; }
.preview-mark {
  width: 24px; height: 24px; border-radius: 6px; background: var(--primary);
  display: grid; place-items: center; font-size: 10px;
}
.preview-role {
  font-size: 9.5px; letter-spacing: .08em; text-transform: uppercase;
  color: rgba(255,255,255,.5); padding: 8px 6px 6px;
}
.preview-nav { display: flex; align-items: center; gap: 9px; padding: 8px; border-radius: 7px; font-size: 12px; color: rgba(255,255,255,.75); }
.preview-main { padding: 20px; min-width: 0; }
.preview-metrics { display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px; margin-bottom: 16px; }
.preview-metric {
  display: flex; flex-direction: column; gap: 3px; padding: 14px;
  border: 1px solid var(--border); border-radius: var(--radius-sm); color: var(--text-2);
}
.preview-metric strong { font-size: 22px; color: var(--text); }
.preview-metric span { font-size: 11.5px; }
.preview-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
.preview-card { border: 1px solid var(--border); border-radius: var(--radius-sm); padding: 14px; }
.preview-card h3 { margin: 0 0 10px; font-size: 13px; }
.preview-row { display: flex; align-items: center; justify-content: space-between; gap: 10px; padding: 8px 0; border-bottom: 1px solid var(--border); font-size: 12px; }
.preview-row:last-child { border-bottom: 0; }
.preview-note { margin-top: 14px; font-size: 11.5px; color: var(--text-3); }
@media (max-width: 860px) {
  .preview-shell { grid-template-columns: 1fr; }
  .preview-side { flex-direction: row; flex-wrap: wrap; }
  .preview-metrics, .preview-grid { grid-template-columns: 1fr; }
}
</style>
