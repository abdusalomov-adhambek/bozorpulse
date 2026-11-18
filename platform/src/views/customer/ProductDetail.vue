<script setup>
import { money } from '../../utils/money'
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import api from '../../api/client'
import { t } from '../../i18n'
import { useBasketStore } from '../../stores/basket'
import { useAppStore } from '../../stores/app'
import NumberInput from '../../components/ui/NumberInput.vue'
import AppIcon from '../../components/AppIcon.vue'
import PriceChart from '../../components/PriceChart.vue'

const route = useRoute()
const router = useRouter()
const basket = useBasketStore()
const app = useAppStore()

const product = ref(null)
const supplier = ref(null)
const rules = ref([])
const related = ref([])
const deliveryZones = ref([])
const priceHistory = ref(null)
const loading = ref(true)
const activePhoto = ref(0)
const quantity = ref(1)

const photos = computed(() => product.value?.photos?.length ? product.value.photos : [])
const bestPrice = computed(() => {
  if (!product.value) return 0
  const base = Number((product.value.effective_price ?? product.value.price) || 0)
  const best = rules.value.reduce((min, rule) => Math.min(min, Number(rule.effective_price || Infinity)), Infinity)
  return Number.isFinite(best) ? Math.min(base, best) : base
})
const minOrder = computed(() => {
  const fromRules = rules.value.reduce((min, r) => Math.min(min, Number(r.min_order || Infinity)), Infinity)
  const min = Number.isFinite(fromRules) ? Math.min(Number(product.value?.min_order_qty || 1), fromRules) : Number(product.value?.min_order_qty || 1)
  return min || 1
})
const maxOrder = computed(() => {
  const limits = rules.value.map((r) => Number(r.max_order || 0)).filter((v) => v > 0)
  const productMax = Number(product.value?.max_order_qty || 0)
  const candidates = [...limits, productMax].filter((v) => v > 0)
  return candidates.length ? Math.min(...candidates) : 0
})
const stock = computed(() => {
  const n = Number(product.value?.stock)
  return Number.isFinite(n) && n >= 0 ? n : 0
})
const maxAllowed = computed(() => {
  const caps = []
  if (stock.value > 0) caps.push(stock.value)
  if (maxOrder.value > 0) caps.push(maxOrder.value)
  return caps.length ? Math.min(...caps) : 0
})
const invalidQty = computed(() => Number(quantity.value) < minOrder.value || (maxAllowed.value > 0 && Number(quantity.value) > maxAllowed.value))
const qtyError = computed(() => {
  const q = Number(quantity.value)
  if (q < minOrder.value) return `Quantity must be at least ${minOrder.value}.`
  if (maxAllowed.value > 0 && q > maxAllowed.value) return `Only ${maxAllowed.value} available.`
  return ''
})
const historySamples = computed(() => priceHistory.value?.samples || [])
const chartPoints = computed(() => historySamples.value.map((s) => ({ price: Number(s.price), observed_at: s.observed_at })))

function formatDate(value) {
  if (!value) return '—'
  const d = new Date(value)
  return Number.isNaN(d.getTime()) ? value : d.toLocaleDateString()
}

function deliveryLabel(item = product.value) {
  const min = Number(item?.delivery_min_days || 0)
  const max = Number(item?.delivery_max_days || 0)
  if (!min && !max) return 'Delivery on request'
  return min === max ? `${min} day(s)` : `${min}–${max} days`
}
function band(rule) {
  return `${rule.min_qty}–${rule.max_qty || '∞'} ${product.value?.unit || 'pcs'}`
}
function warehouseName(rule) {
  return rule.warehouse_name || 'All warehouses'
}

function addToBasket() {
  if (!product.value || invalidQty.value) return
  const result = basket.add(product.value, supplier.value?.company_name || 'Supplier', {
    quantity: Number(quantity.value),
    price: bestPrice.value,
    unit: product.value.unit,
    stock: stock.value,
    min_order: minOrder.value,
    max_order: maxOrder.value,
  })
  quantity.value = result.quantity
  if (!result.ok && result.reason === 'stock') {
    app.error(`Only ${result.quantity} available — quantity adjusted.`)
  } else {
    app.success(`${product.value.name} added to basket`)
  }
}

onMounted(async () => {
  loading.value = true
  const id = route.params.id
  try {
    const productRes = await api.get(`/products/${id}`, {}, { auth: false })
    product.value = Array.isArray(productRes) ? productRes[0] : productRes
    const [supplierRes, pricingRes, relatedRes, deliveryRes, historyRes] = await Promise.all([
      product.value?.supplier_id ? api.get(`/suppliers/${product.value.supplier_id}`, {}, { auth: false }).catch(() => null) : null,
      api.get(`/products/${id}/pricing`, {}, { auth: false }).catch(() => ({ data: [] })),
      product.value?.supplier_id ? api.get('/products', { supplier_id: product.value.supplier_id, limit: 5 }, { auth: false }).catch(() => ({ data: [] })) : null,
      product.value?.supplier_id ? api.get(`/suppliers/${product.value.supplier_id}/delivery-zones`, {}, { auth: false }).catch(() => ({ data: [] })) : null,
      api.get(`/products/${id}/price-history`, {}, { auth: false }).catch(() => null),
    ])
    supplier.value = supplierRes
    rules.value = Array.isArray(pricingRes) ? pricingRes : (pricingRes?.data || [])
    related.value = (Array.isArray(relatedRes) ? relatedRes : (relatedRes?.data || [])).filter((p) => p.id !== id)
    deliveryZones.value = Array.isArray(deliveryRes) ? deliveryRes : (deliveryRes?.data || [])
    priceHistory.value = historyRes
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div>
    <button class="btn btn-secondary back-btn" type="button" @click="router.push('/customer/market')">
      <AppIcon name="chevron-left" :size="15" />{{ t('Back to market') }}
    </button>

    <div v-if="loading" class="loading-block"><span class="spinner" /></div>
    <div v-else-if="product">
      <div class="detail-grid">
        <section class="card gallery">
          <div class="main-photo">
            <img v-if="photos.length" :src="photos[activePhoto]" :alt="product.name" />
            <div v-else class="photo-placeholder"><AppIcon name="box" :size="54" /></div>
          </div>
          <div v-if="photos.length > 1" class="thumbs">
            <button
              v-for="(photo, i) in photos"
              :key="i"
              type="button"
              :class="{ active: i === activePhoto }"
              @click="activePhoto = i"
            ><img :src="photo" alt="" /></button>
          </div>
        </section>

        <section class="detail-info">
          <div class="detail-head">
            <div>
              <h1 class="page-title">{{ product.name }}</h1>
              <p class="subtle">SKU {{ product.sku || '—' }}</p>
            </div>
            <span v-if="Number(product.discount_percent) > 0" class="discount-pill">−{{ product.discount_percent }}%</span>
          </div>

          <div class="price-box">
            <strong>{{ money(bestPrice) }} <small>/ {{ product.unit || 'pcs' }}</small></strong>
            <span v-if="Number(product.discount_percent) > 0" class="old-price">{{ money(product.price || 0) }}</span>
          </div>

          <p class="description">{{ product.description || 'No description provided by the supplier.' }}</p>

          <div class="spec-grid">
            <div><span>{{ t('Supplier') }}</span><strong>{{ supplier?.company_name || 'Supplier' }}</strong></div>
            <div><span>{{ t('Region') }}</span><strong>{{ supplier?.region_name || '—' }}</strong></div>
            <div><span>{{ t('Available stock') }}</span><strong>{{ product.stock }}</strong></div>
            <div><span>{{ t('Delivery') }}</span><strong>{{ deliveryLabel() }}</strong></div>
            <div><span>{{ t('Order quantity') }}</span><strong>{{ minOrder }}<template v-if="maxOrder">–{{ maxOrder }}</template> {{ product.unit || 'pcs' }}</strong></div>
            <div><span>{{ t('SKU') }}</span><strong>{{ product.sku || '—' }}</strong></div>
          </div>

          <div class="order-box">
            <label class="field-label">{{ t('Quantity') }}</label>
            <div class="order-row">
              <NumberInput v-model="quantity" :min="minOrder" :max="maxAllowed || undefined" />
              <button class="btn" type="button" :disabled="invalidQty" @click="addToBasket">
                <AppIcon name="basket" :size="16" />{{ t('Add to basket') }}
              </button>
            </div>
            <p v-if="invalidQty" class="field-error">{{ qtyError }}</p>
          </div>
        </section>
      </div>

      <div class="detail-lower">
        <section class="card">
          <div class="card-head"><div class="card-title">{{ t('Supplier information') }}</div></div>
          <div class="card-body">
            <div class="kv-list">
              <div class="kv-row"><span class="kv-key">{{ t('Company') }}</span><span class="kv-val">{{ supplier?.company_name || '—' }}</span></div>
              <div class="kv-row"><span class="kv-key">{{ t('Region') }}</span><span class="kv-val">{{ supplier?.region_name || '—' }}</span></div>
              <div class="kv-row"><span class="kv-key">{{ t('INN') }}</span><span class="kv-val mono">{{ supplier?.inn || '—' }}</span></div>
              <div class="kv-row"><span class="kv-key">{{ t('Listed products') }}</span><span class="kv-val">{{ related.length + 1 }}</span></div>
            </div>
          </div>
        </section>

        <section class="card">
          <div class="card-head"><div><div class="card-title">{{ t('Price rates') }}</div><div class="card-sub">{{ t('Warehouse and quantity-based pricing') }}</div></div></div>
          <div class="card-body">
            <div v-if="rules.length" class="rate-table">
              <div class="rate-head"><span>{{ t('Warehouse') }}</span><span>{{ t('Qty band') }}</span><span>{{ t('Price') }}</span><span>{{ t('Effective') }}</span><span>{{ t('Delivery') }}</span></div>
              <div v-for="rule in rules" :key="rule.id" class="rate-row">
                <span>{{ warehouseName(rule) }}</span>
                <span>{{ band(rule) }}</span>
                <span class="mono">{{ money(rule.unit_price) }}</span>
                <span class="mono text-primary">{{ money(rule.effective_price) }}</span>
                <span>{{ rule.delivery_min_days }}–{{ rule.delivery_max_days }}d</span>
              </div>
            </div>
            <p v-else class="subtle">{{ t("No special rates. The supplier's base price applies to every order.") }}</p>
          </div>
        </section>
      </div>

      <section class="card section-card">
        <div class="card-head">
          <div><div class="card-title">{{ t('Price history') }}</div><div class="card-sub">{{ t('Observed prices over time') }}</div></div>
        </div>
        <div class="card-body">
          <template v-if="historySamples.length">
            <div class="history-stats">
              <div><span>{{ t('Min') }}</span><strong class="mono">{{ money(priceHistory.min || 0) }}</strong></div>
              <div><span>{{ t('Average') }}</span><strong class="mono">{{ money(priceHistory.avg || 0) }}</strong></div>
              <div><span>{{ t('Max') }}</span><strong class="mono">{{ money(priceHistory.max || 0) }}</strong></div>
              <div><span>{{ t('Observations') }}</span><strong class="mono">{{ priceHistory.count }}</strong></div>
            </div>
            <PriceChart :points="chartPoints" :height="170" />
            <div v-if="priceHistory.data_status === 'demo'" class="history-badge">
              {{ t('Demo data — sample values, not live market quotes') }}
            </div>
            <div class="rate-table history-table">
              <div class="rate-head history-head"><span>{{ t('Date') }}</span><span>{{ t('Price') }}</span><span>{{ t('Region') }}</span></div>
              <div v-for="sample in historySamples" :key="sample.id" class="rate-row history-row">
                <span>{{ formatDate(sample.observed_at) }}</span>
                <span class="mono">{{ money(sample.price) }}</span>
                <span>{{ sample.region_id || '—' }}<small v-if="sample.is_demo"> · {{ t('Demo') }}</small></span>
              </div>
            </div>
            <p v-if="priceHistory.disclaimer" class="subtle history-note">{{ priceHistory.disclaimer }}</p>
          </template>
          <p v-else class="subtle">{{ t('No price history available yet.') }}</p>
        </div>
      </section>

      <section class="card section-card">
        <div class="card-head">
          <div><div class="card-title">{{ t('Delivery by region') }}</div><div class="card-sub">{{ t('Availability, price and period set by the supplier') }}</div></div>
        </div>
        <div class="card-body">
          <div v-if="deliveryZones.length" class="rate-table">
            <div class="rate-head four"><span>{{ t('Region') }}</span><span>{{ t('Available') }}</span><span>{{ t('Price') }}</span><span>{{ t('Period') }}</span></div>
            <div v-for="zone in deliveryZones" :key="zone.id" class="rate-row four">
              <span>{{ zone.region_name || zone.region_id }}</span>
              <span :class="zone.available ? 'text-primary' : 'text-danger'">{{ zone.available ? 'Yes' : 'No' }}</span>
              <span class="mono">{{ zone.available ? money(zone.price) : '—' }}</span>
              <span>{{ zone.available ? (zone.min_days === zone.max_days ? `${zone.min_days}d` : `${zone.min_days}–${zone.max_days}d`) : '—' }}</span>
            </div>
          </div>
          <p v-else class="subtle">{{ t('This supplier has not configured delivery regions yet.') }}</p>
        </div>
      </section>

      <section v-if="related.length" class="card related">
        <div class="card-head"><div class="card-title">{{ t('More from this supplier') }}</div></div>
        <div class="card-body related-grid">
          <RouterLink v-for="item in related" :key="item.id" :to="`/customer/market/${item.id}`" class="related-item">
            <img v-if="item.photos?.length" :src="item.photos[0]" alt="" />
            <div v-else class="related-placeholder"><AppIcon name="box" :size="20" /></div>
            <strong>{{ item.name }}</strong>
            <span class="mono">{{ money((item.effective_price ?? item.price) || 0) }}</span>
          </RouterLink>
        </div>
      </section>
    </div>

    <div v-else class="card card-pad">{{ t('Product not found.') }}</div>
  </div>
</template>

<style scoped>
.back-btn { margin-bottom: 18px; color: #1c1c1b; }
.back-btn :deep(.app-icon) { color: #1c1c1b; stroke: #1c1c1b; }
.detail-grid { display: grid; grid-template-columns: minmax(280px, 1fr) minmax(320px, 1.1fr); gap: 18px; }
.gallery { padding: 16px; }
.main-photo { height: 320px; border-radius: var(--radius-sm); overflow: hidden; background: var(--surface-2); }
.main-photo img { width: 100%; height: 100%; object-fit: cover; }
.photo-placeholder { width: 100%; height: 100%; display: grid; place-items: center; color: var(--text-3); background: linear-gradient(135deg, var(--primary-50), var(--surface-2)); }
.thumbs { display: flex; gap: 8px; margin-top: 10px; }
.thumbs button { width: 56px; height: 56px; padding: 0; border: 2px solid transparent; border-radius: 8px; overflow: hidden; cursor: pointer; background: none; }
.thumbs button.active { border-color: var(--primary); }
.thumbs img { width: 100%; height: 100%; object-fit: cover; }
.detail-info { padding: 4px 0; }
.detail-head { display: flex; justify-content: space-between; gap: 12px; align-items: flex-start; }
.discount-pill { background: var(--success-50); color: var(--success); border-radius: 999px; padding: 4px 10px; font-size: 12px; font-weight: 750; }
.price-box { display: flex; align-items: baseline; gap: 12px; margin: 16px 0 10px; }
.price-box strong { font-size: 30px; letter-spacing: -.03em; }
.price-box small { font-size: 13px; color: var(--text-3); font-weight: 500; }
.old-price { color: var(--text-3); text-decoration: line-through; }
.description { color: var(--text-2); line-height: 1.6; margin: 12px 0 20px; }
.spec-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; margin-bottom: 20px; }
.spec-grid div { display: flex; flex-direction: column; gap: 3px; padding: 12px; border: 1px solid var(--border); border-radius: var(--radius-sm); }
.spec-grid span { font-size: 11px; color: var(--text-3); }
.spec-grid strong { font-size: 13px; }
.order-box { padding: 16px; border: 1px solid var(--border); border-radius: var(--radius); background: var(--surface-2); }
.order-row { display: flex; gap: 10px; margin-top: 8px; }
.order-row .number-input { width: 130px; }
.order-row .btn { flex: 1; justify-content: center; }
.detail-lower { display: grid; grid-template-columns: 1fr 1.3fr; gap: 18px; margin-top: 18px; }
.section-card { margin-top: 18px; }
.rate-table { width: 100%; }
.rate-head, .rate-row { display: grid; grid-template-columns: 1.4fr .9fr .7fr .7fr .6fr; gap: 8px; padding: 9px 0; border-bottom: 1px solid var(--border); font-size: 12px; }
.rate-head.four, .rate-row.four { grid-template-columns: 1.4fr .6fr .8fr .8fr; }
.rate-head { color: var(--text-3); font-weight: 700; }
.rate-row:last-child { border-bottom: 0; }
.history-stats { display: grid; grid-template-columns: repeat(4, 1fr); gap: 10px; margin-bottom: 14px; }
.history-stats div { display: flex; flex-direction: column; gap: 3px; padding: 10px 12px; border: 1px solid var(--border); border-radius: var(--radius-sm); }
.history-stats span { font-size: 11px; color: var(--text-3); }
.history-stats strong { font-size: 15px; }
.history-badge { display: inline-block; margin: 10px 0; padding: 4px 10px; border-radius: 999px; background: var(--warning-50, #fff7e6); color: var(--warning, #b7791f); font-size: 11px; font-weight: 700; }
.history-head, .history-row { grid-template-columns: 1fr 1fr 1fr; }
.history-row small { color: var(--text-3); }
.history-note { margin-top: 10px; font-size: 11.5px; }
.related { margin-top: 18px; }
.related-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(130px, 1fr)); gap: 12px; }
.related-item { display: flex; flex-direction: column; gap: 5px; text-decoration: none; color: var(--text); font-size: 12px; }
.related-item img, .related-placeholder { width: 100%; height: 80px; object-fit: cover; border-radius: 8px; background: var(--surface-2); display: grid; place-items: center; color: var(--text-3); }
@media (max-width: 900px) { .detail-grid, .detail-lower { grid-template-columns: 1fr; } .spec-grid { grid-template-columns: 1fr; } }
</style>
