<script setup>
import { money } from '../../utils/money'
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import api from '../../api/client'
import { t } from '../../i18n'
import { useBasketStore } from '../../stores/basket'
import { useAppStore } from '../../stores/app'
import AppInput from '../../components/ui/AppInput.vue'
import AppSelect from '../../components/ui/AppSelect.vue'
import AppCheckbox from '../../components/ui/AppCheckbox.vue'
import AppIcon from '../../components/AppIcon.vue'
import AssistantChat from '../../components/AssistantChat.vue'

const basket = useBasketStore()
const app = useAppStore()
const router = useRouter()

const products = ref([])
const categories = ref([])
const suppliers = ref([])
const query = ref('')
const categoryId = ref('')
const supplierId = ref('')
const sort = ref('featured')
const inStockOnly = ref(false)
const loading = ref(false)
const priceHints = ref({})
const previewOpen = ref(false)
const advisorOpen = ref(false)
let searchTimer = null

const supplierNames = computed(() => {
  const map = {}
  for (const s of suppliers.value) map[s.id] = s.company_name
  return map
})

const categoryOptions = computed(() => categories.value.map((c) => ({ value: c.id, label: c.name })))
const supplierOptions = computed(() => suppliers.value.map((s) => ({ value: s.id, label: s.company_name })))
const sortOptions = computed(() => [
  { value: 'featured', label: t('Featured') },
  { value: 'price-asc', label: t('Price: low to high') },
  { value: 'price-desc', label: t('Price: high to low') },
  { value: 'newest', label: t('Newest first') },
])

const visibleProducts = computed(() => {
  let list = [...products.value]
  if (inStockOnly.value) list = list.filter((p) => Number(p.stock || 0) > 0)
  if (sort.value === 'price-asc') list.sort((a, b) => displayPrice(a) - displayPrice(b))
  if (sort.value === 'price-desc') list.sort((a, b) => displayPrice(b) - displayPrice(a))
  if (sort.value === 'newest') list.sort((a, b) => String(b.created_at || '').localeCompare(String(a.created_at || '')))
  return list
})

const hasFilters = computed(() => !!query.value || !!categoryId.value || !!supplierId.value || inStockOnly.value || sort.value !== 'featured')

async function loadProducts() {
  loading.value = true
  try {
    const res = await api.get('/products', {
      q: query.value || undefined,
      category_id: categoryId.value || undefined,
      supplier_id: supplierId.value || undefined,
      limit: 100,
    })
    products.value = Array.isArray(res) ? res : (res?.data || [])
    await loadPriceHints()
  } finally {
    loading.value = false
  }
}

async function loadPriceHints() {
  const entries = await Promise.all(products.value.map(async (product) => {
    try {
      const res = await api.get(`/products/${product.id}/pricing`, {}, { auth: false })
      const rules = Array.isArray(res) ? res : (res?.data || [])
      const best = rules.reduce((min, rule) => Math.min(min, Number(rule.effective_price || rule.unit_price || Infinity)), Infinity)
      return [product.id, Number.isFinite(best) ? best : null]
    } catch {
      return [product.id, null]
    }
  }))
  priceHints.value = Object.fromEntries(entries.filter(([, value]) => value !== null))
}

watch([query, categoryId, supplierId], () => {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => loadProducts(), 250)
})

onMounted(async () => {
  try {
    const [cats, sups] = await Promise.all([
      api.get('/categories', {}, { auth: false }),
      api.get('/suppliers', { limit: 100 }, { auth: false }),
    ])
    categories.value = Array.isArray(cats) ? cats : (cats?.data || [])
    suppliers.value = Array.isArray(sups) ? sups : (sups?.data || [])
  } catch { /* filters simply stay empty */ }
  await loadProducts()
})

function displayPrice(product) {
  const base = Number((product.effective_price ?? product.price) || 0)
  const hint = priceHints.value[product.id]
  return hint !== undefined && hint < base ? hint : base
}

function priceLabel(product) {
  const base = Number((product.effective_price ?? product.price) || 0)
  const hint = priceHints.value[product.id]
  return hint !== undefined && hint < base ? `from ${money(hint)}` : money(base)
}

function deliveryLabel(product) {
  const min = Number(product.delivery_min_days || 0)
  const max = Number(product.delivery_max_days || 0)
  if (!min && !max) return 'Delivery on request'
  return min === max ? `${min} day(s)` : `${min}–${max} days`
}

function orderLabel(product) {
  const min = Number(product.min_order_qty || 1)
  const max = Number(product.max_order_qty || 0)
  return max > 0 ? `${min}–${max} pcs` : `Min ${min} · no max`
}

function categoryLabel(id) {
  return categories.value.find((c) => c.id === id)?.name || 'Product'
}

function resetFilters() {
  query.value = ''
  categoryId.value = ''
  supplierId.value = ''
  sort.value = 'featured'
  inStockOnly.value = false
  loadProducts()
}

function addToBasket(product) {
  const result = basket.add(product, supplierNames.value[product.supplier_id] || 'Supplier')
  if (!result.ok && result.reason === 'stock') {
    app.error(`Only ${result.quantity} available in stock.`)
  } else {
    app.success(`${product.name} → basket`)
  }
}

async function onAssistantAction(action) {
  if (!action) return
  if (action.kind === 'link' || action.kind === 'go_to_orders') {
    router.push(action.target)
    return
  }
  if (action.kind === 'add_to_basket') {
    let product = products.value.find((p) => p.id === action.target)
    if (!product) {
      try { product = await api.get(`/products/${action.target}`) } catch { product = null }
    }
    if (!product) {
      app.error?.("Couldn't find that product in the catalog.")
      return
    }
    // Use the selected offer's supplier, price, unit and quantity, not the
    // catalog default — the chat shortlist may contain several suppliers.
    basket.add(product, supplierNames.value[product.supplier_id] || 'Supplier', action.meta || null)
    app.success?.(`${product.name} → basket`)
  }
}
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t('Market') }}</h1>
        <p class="page-desc">{{ t('Browse live supplier offers, compare prices and find products you need.') }}</p>
      </div>
      <div class="header-actions">
        <button class="btn btn-secondary" type="button" @click="previewOpen = !previewOpen">
          <AppIcon name="basket" :size="16" /> {{ t('Basket') }}
          <span v-if="basket.count" class="basket-count">{{ basket.count }}</span>
        </button>
        <RouterLink to="/customer/basket" class="btn">{{ t('Open basket') }}</RouterLink>
      </div>
    </div>

    <div v-if="previewOpen && basket.count" class="mini-basket card">
      <div class="mini-basket-head">
        <strong>{{ t('Your basket') }}</strong>
        <button class="icon-btn" type="button" :aria-label="t('Close')" @click="previewOpen = false">×</button>
      </div>
      <div v-for="group in basket.grouped" :key="group.supplier_id" class="mini-basket-row">
        <span>{{ group.supplier_name }}</span>
        <strong>{{ money(group.total) }}</strong>
      </div>
      <RouterLink to="/customer/basket" class="btn btn-block">{{ t('Review basket') }}</RouterLink>
    </div>

    <div class="card advisor-card">
      <button class="advisor-toggle" type="button" @click="advisorOpen = !advisorOpen">
        <span class="advisor-toggle-icon"><AppIcon name="chart" :size="17" /></span>
        <span>
          <strong>{{ t('Ask AI what to buy') }}</strong>
          <small>{{ t('What do you need? (product, quantity, region)') }}</small>
        </span>
        <span class="advisor-chevron">{{ advisorOpen ? '▴' : '▾' }}</span>
      </button>
      <AssistantChat v-if="advisorOpen" mode="shopping" embedded @action="onAssistantAction" />
    </div>

    <div class="card market-filters">
      <AppInput v-model="query" :placeholder="t('Search products')" clearable>
        <template #prefix><AppIcon name="search" :size="16" /></template>
      </AppInput>
      <AppSelect v-model="categoryId" :options="categoryOptions" :placeholder="t('All categories')" clearable />
      <AppSelect v-model="supplierId" :options="supplierOptions" :placeholder="t('All suppliers')" clearable />
      <AppSelect v-model="sort" :options="sortOptions" :placeholder="t('Sort')" />
      <AppCheckbox v-model="inStockOnly" :label="t('In stock only')" />
      <button v-if="hasFilters" class="btn btn-secondary" type="button" @click="resetFilters">{{ t('Reset') }}</button>
    </div>

    <div v-if="loading" class="loading-block"><span class="spinner" /></div>

    <div v-else-if="visibleProducts.length" class="market-grid">
      <article v-for="product in visibleProducts" :key="product.id" class="market-card">
        <div class="market-card-top">
          <div class="photo-wrap">
            <img v-if="product.photos?.length" :src="product.photos[0]" class="market-photo" alt="" />
            <div v-else class="market-placeholder" :aria-label="categoryLabel(product.category_id)">
              <span>{{ categoryLabel(product.category_id).slice(0, 1) }}</span>
            </div>
          </div>
          <span v-if="Number(product.discount_percent) > 0" class="discount-pill">−{{ product.discount_percent }}%</span>
        </div>
        <RouterLink :to="`/customer/market/${product.id}`" class="market-title">{{ product.name }}</RouterLink>
        <p class="market-supplier">{{ supplierNames[product.supplier_id] || t('Supplier') }}</p>
        <div class="market-price">{{ priceLabel(product) }} <small>/ {{ product.unit || 'pcs' }}</small></div>
        <div class="market-meta">
          <span><AppIcon name="clock" :size="13" /> {{ deliveryLabel(product) }}</span>
          <span>{{ orderLabel(product) }}</span>
        </div>
        <p v-if="product.description" class="market-desc">{{ product.description }}</p>
        <div class="card-actions">
          <button class="btn" type="button" @click="addToBasket(product)">
            <AppIcon name="basket" :size="15" /> {{ t('Add to basket') }}
          </button>
          <RouterLink :to="`/customer/market/${product.id}`" class="btn btn-secondary">{{ t('Details') }}</RouterLink>
        </div>
      </article>
    </div>

    <div v-else class="card card-pad empty-market">
      <AppIcon name="search" :size="28" />
      <h3>{{ t('No products found') }}</h3>
      <p>{{ t('Try different filters or clear them to see all offers.') }}</p>
      <button class="btn btn-secondary" type="button" @click="resetFilters">{{ t('Clear filters') }}</button>
    </div>
  </div>
</template>

<style scoped>
.header-actions { display: flex; gap: 8px; align-items: center; }
.basket-count {
  display: inline-grid; place-items: center; min-width: 18px; height: 18px;
  padding: 0 5px; border-radius: 999px; background: var(--primary); color: #fff; font-size: 10px; font-weight: 800;
}
.mini-basket { max-width: 420px; margin: 0 0 16px auto; padding: 14px; }
.mini-basket-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; }
.mini-basket-row { display: flex; justify-content: space-between; padding: 7px 0; border-bottom: 1px solid var(--border); font-size: 12.5px; }
.advisor-card { padding: 14px; margin-bottom: 18px; }
.advisor-toggle { display: flex; align-items: center; gap: 12px; width: 100%; text-align: left; border: 0; background: none; cursor: pointer; padding: 0; color: var(--text); }
.advisor-toggle-icon { width: 34px; height: 34px; display: grid; place-items: center; border-radius: 9px; background: var(--primary-50); color: var(--primary); }
.advisor-toggle strong { display: block; font-size: 14px; }
.advisor-toggle small { display: block; color: var(--text-3); font-size: 12px; margin-top: 2px; }
.advisor-chevron { margin-left: auto; color: var(--text-3); }
.advisor-card :deep(.assistant) { margin-top: 12px; }
.market-filters {
  display: grid; grid-template-columns: 1.3fr 1fr 1fr .9fr auto auto; gap: 10px;
  align-items: end; padding: 16px; margin-bottom: 18px;
}
.market-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(230px, 1fr)); gap: 14px; }
.market-card {
  display: flex; flex-direction: column; gap: 8px;
  border: 1px solid var(--border); border-radius: var(--radius);
  background: var(--surface); padding: 16px;
  transition: transform .18s, box-shadow .18s, border-color .18s;
}
.market-card:hover { transform: translateY(-2px); box-shadow: var(--shadow-sm); border-color: var(--primary); }
.market-card-top { position: relative; }
.photo-wrap { width: 100%; height: 110px; min-width: 0; }
.market-photo { width: 100%; height: 100%; object-fit: cover; border-radius: 8px; border: 1px solid var(--border); }
.market-placeholder {
  width: 100%; height: 100%; display: grid; place-items: center;
  border-radius: 8px; border: 1px dashed var(--border-strong);
  background: linear-gradient(135deg, var(--primary-50), var(--surface-2));
  color: var(--primary); font-size: 30px; font-weight: 800; opacity: .8;
}
.discount-pill { position: absolute; top: 8px; right: 8px; z-index: 1; background: var(--success-50); color: var(--success); border-radius: 999px; padding: 3px 8px; font-size: 11px; font-weight: 750; }
.market-title { font-weight: 700; font-size: 14px; color: var(--text); text-decoration: none; }
.market-title:hover { color: var(--primary); }
.market-supplier { color: var(--text-3); font-size: 11.5px; }
.market-price { font-size: 19px; font-weight: 800; letter-spacing: -.02em; margin-top: 2px; }
.market-price small { font-size: 11px; font-weight: 500; color: var(--text-3); }
.market-meta { display: flex; flex-wrap: wrap; gap: 10px; color: var(--text-3); font-size: 11.5px; }
.market-meta span { display: inline-flex; align-items: center; gap: 4px; }
.market-desc { color: var(--text-2); font-size: 12px; flex: 1; line-height: 1.4; }
.card-actions { display: flex; flex-direction: column; gap: 8px; }
.card-actions .btn { width: 100%; min-width: 0; justify-content: center; }
.empty-market { text-align: center; display: flex; flex-direction: column; align-items: center; gap: 8px; color: var(--text-3); }
.empty-market h3 { color: var(--text); margin: 4px 0 0; }
@media (max-width: 1100px) { .market-filters { grid-template-columns: 1fr 1fr 1fr; } }
@media (max-width: 720px) { .market-filters { grid-template-columns: 1fr; } }
</style>
