<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { t } from '../../i18n'
import { money } from '../../utils/money'
import api from '../../api/client'
import { useAppStore } from '../../stores/app'
import { useBasketStore } from '../../stores/basket'
import AppIcon from '../../components/AppIcon.vue'
import AppInput from '../../components/ui/AppInput.vue'
import AppSelect from '../../components/ui/AppSelect.vue'
import StatusBadge from '../../components/StatusBadge.vue'

const app = useAppStore()
const basket = useBasketStore()
const router = useRouter()

// Forecast reasons arrive from the backend as English sentences. Map the
// known set to i18n keys so supported locales see translated text.
function reasonText(text) {
  if (!text) return ''
  const value = String(text)
  const count = value.match(/^(\d+) comparable observations$/)
  if (count) return t('{count} comparable observations', { count: count[1] })
  if (value === 'long, product-specific history') return t('long product-specific history')
  if (value === 'usable history') return t('usable history')
  if (value === 'limited or cold-start history') return t('limited or cold-start history')
  if (value.startsWith('fallback level: ')) return t('fallback level: {level}', { level: value.slice('fallback level: '.length) })
  return value
}

const plans = ref([])
const products = ref([])
const regions = ref([])
const suppliers = ref([])
const selected = ref(null)
const preview = ref(null)
const loading = ref(false)
const saving = ref(false)
const editorOpen = ref(false)
const error = ref('')

const maxDate = new Date(Date.now() + 30 * 86400000).toISOString().slice(0, 10)
const blankItem = () => ({ product_id: '', quantity: 10, target_date: '', note: '' })
const form = ref({ title: '', region_id: '', items: [blankItem()] })

const productOptions = computed(() => products.value.map((p) => ({
  value: p.id,
  label: `${p.name}${p.effective_price != null ? ` — ${money(p.effective_price)}/${p.unit || 'pcs'}` : ''}`,
})))
const regionOptions = computed(() => [
  { value: '', label: t('All regions') },
  ...regions.value.map((r) => ({ value: r.id, label: r.name })),
])
const supplierNames = computed(() => Object.fromEntries(suppliers.value.map((s) => [s.id, s.company_name])))

const recommendations = {
  buy_now: { label: t('Buy sooner'), cls: 'rec-buy' },
  wait: { label: t('Consider waiting'), cls: 'rec-wait' },
  monitor: { label: t('No clear signal'), cls: 'rec-monitor' },
  insufficient: { label: t('Not enough data'), cls: 'rec-none' },
}

function recommendation(item) {
  return recommendations[item?.projection?.recommendation] || recommendations.insufficient
}
function hasDemo(items = []) {
  return items.some((i) => i?.projection?.data_status === 'demo')
}
function display(value) {
  return value === null || value === undefined ? '—' : money(value)
}
function signed(value) {
  return value === null || value === undefined ? '—' : `${value > 0 ? '+' : ''}${money(value)}`
}
function productFor(id) {
  return products.value.find((p) => p.id === id) || null
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.listPlans()
    plans.value = Array.isArray(res) ? res : (res?.data || [])
  } catch (err) {
    error.value = err?.message || t('Could not load plans.')
  } finally {
    loading.value = false
  }
}

async function loadSupport() {
  try {
    const [prods, regs, sups] = await Promise.all([
      api.get('/products', { limit: 100 }).catch(() => []),
      api.get('/regions', {}, { auth: false }).catch(() => []),
      api.get('/suppliers', { limit: 100 }, { auth: false }).catch(() => []),
    ])
    products.value = Array.isArray(prods) ? prods : (prods?.data || [])
    regions.value = Array.isArray(regs) ? regs : (regs?.data || [])
    suppliers.value = Array.isArray(sups) ? sups : (sups?.data || [])
  } catch { /* best effort */ }
}

function openEditor() {
  form.value = { title: '', region_id: '', items: [blankItem()] }
  preview.value = null
  selected.value = null
  editorOpen.value = true
}

function addItem() { form.value.items.push(blankItem()) }
function removeItem(index) { form.value.items.splice(index, 1) }

function payload() {
  return {
    title: form.value.title,
    region_id: form.value.region_id || undefined,
    items: form.value.items.map((i) => ({
      product_id: i.product_id,
      quantity: Number(i.quantity),
      target_date: i.target_date || undefined,
      note: i.note || undefined,
    })),
  }
}

async function save() {
  saving.value = true
  try {
    const created = await api.createPlan(payload())
    editorOpen.value = false
    preview.value = null
    app.success(t('Plan created'))
    await load()
    selected.value = created
  } catch (err) {
    app.error(err?.message || t('Could not save plan.'))
  } finally {
    saving.value = false
  }
}

async function doPreview() {
  saving.value = true
  try {
    preview.value = await api.previewPlan(payload())
  } catch (err) {
    app.error(err?.message || t('Could not build preview.'))
  } finally {
    saving.value = false
  }
}

async function openPlan(id) {
  try {
    selected.value = await api.getPlan(id)
    editorOpen.value = false
    preview.value = null
  } catch (err) {
    app.error(err?.message || t('Could not open plan.'))
  }
}

async function refresh(id) {
  try {
    const updated = await api.refreshPlan(id)
    if (selected.value?.id === id) selected.value = updated
    await load()
    app.success(t('Forecasts refreshed'))
  } catch (err) {
    app.error(err?.message || t('Could not refresh forecasts.'))
  }
}

async function remove(id) {
  try {
    await api.deletePlan(id)
    if (selected.value?.id === id) selected.value = null
    await load()
    app.success(t('Plan deleted'))
  } catch (err) {
    app.error(err?.message || t('Could not delete plan.'))
  }
}

function addToBasket(item) {
  const product = productFor(item.product_id)
  if (!product) return
  basket.add(product, supplierNames.value[product.supplier_id] || t('Supplier'))
  app.success(`${product.name} → ${t('Basket')}`)
}

function viewOffer(item) {
  router.push(`/customer/market/${item.product_id}`)
}

onMounted(async () => {
  await loadSupport()
  await load()
})
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t('Planning') }}</h1>
        <p class="page-desc">{{ t('Plan upcoming purchases using conservative price estimates.') }}</p>
      </div>
      <button class="btn" type="button" @click="openEditor">
        <AppIcon name="sparkle" :size="15" /> {{ t('New plan') }}
      </button>
    </div>

    <div class="card note-card">
      <AppIcon name="chart" :size="16" />
      <span>{{ t('Estimates are based on past price samples, not a supplier quote. Always compare live offers before buying.') }}</span>
    </div>

    <p v-if="error" class="error-text">{{ error }}</p>

    <!-- Editor -->
    <section v-if="editorOpen" class="card editor-card">
      <h2 class="section-title">{{ t('New purchase plan') }}</h2>
      <div class="editor-head">
        <AppInput v-model="form.title" :placeholder="t('Plan title')" />
        <AppSelect v-model="form.region_id" :options="regionOptions" :placeholder="t('Region (optional)')" clearable />
      </div>

      <div v-for="(item, index) in form.items" :key="index" class="item-row">
        <AppSelect
          v-model="item.product_id"
          :options="productOptions"
          :placeholder="t('Select product')"
          searchable
        />
        <input v-model.number="item.quantity" class="plain-input" type="number" min="1" max="10000" :aria-label="t('Quantity')" />
        <input v-model="item.target_date" class="plain-input" type="date" min="" :max="maxDate" :aria-label="t('Target date')" />
        <input v-model="item.note" class="plain-input" type="text" :placeholder="t('Note (optional)')" />
        <button class="icon-btn" type="button" :aria-label="t('Remove item')" @click="removeItem(index)">×</button>
      </div>
      <button class="btn btn-secondary btn-sm" type="button" @click="addItem">+ {{ t('Add item') }}</button>

      <div class="editor-actions">
        <button class="btn btn-secondary" type="button" :disabled="saving" @click="doPreview">
          {{ t('Preview estimate') }}
        </button>
        <button class="btn" type="button" :disabled="saving" @click="save">{{ t('Save plan') }}</button>
        <button class="btn btn-ghost" type="button" @click="editorOpen = false">{{ t('Cancel') }}</button>
      </div>

      <div v-if="preview" class="preview-block">
        <h3 class="section-title">{{ t('Preview (not saved)') }}</h3>
        <div v-if="hasDemo(preview.items)" class="demo-badge">{{ t('Demo data — sample values, not live market quotes') }}</div>
        <div v-for="item in preview.items" :key="item.product_id" class="preview-row">
          <strong>{{ item.product_name }}</strong>
          <span>{{ t('Price now') }}: {{ display(item.projection.current_unit_price) }}</span>
          <span>{{ t('Estimate') }}: {{ display(item.projection.predicted) }}</span>
          <span>{{ t('Budget') }}: {{ display(item.projection.estimated_budget) }}</span>
          <span :class="recommendation(item).cls">{{ recommendation(item).label }}</span>
        </div>
      </div>
    </section>

    <!-- Plans list -->
    <section v-if="!editorOpen" class="plans-grid">
      <article v-for="plan in plans" :key="plan.id" class="card plan-card" @click="openPlan(plan.id)">
        <div class="plan-card-top">
          <strong class="plan-title">{{ plan.title }}</strong>
          <StatusBadge :status="plan.status" />
        </div>
        <p class="plan-meta">{{ plan.items?.length || 0 }} {{ t('items') }} · {{ plan.updated_at }}</p>
        <div v-if="hasDemo(plan.items)" class="demo-badge">{{ t('Demo data') }}</div>
        <div class="plan-actions" @click.stop>
          <button class="btn btn-secondary btn-sm" type="button" @click="openPlan(plan.id)">{{ t('Open') }}</button>
          <button class="btn btn-secondary btn-sm" type="button" @click="refresh(plan.id)">{{ t('Refresh forecast') }}</button>
          <button class="btn btn-ghost btn-sm danger" type="button" @click="remove(plan.id)">{{ t('Delete') }}</button>
        </div>
      </article>
      <div v-if="!loading && !plans.length" class="card card-pad empty-state">
        <AppIcon name="chart" :size="26" />
        <h3>{{ t('No plans yet') }}</h3>
        <p>{{ t('Create a plan to see predicted prices for the products you need.') }}</p>
      </div>
    </section>

    <!-- Selected plan detail -->
    <section v-if="selected && !editorOpen" class="card detail-card">
      <div class="detail-head">
        <div>
          <h2 class="section-title">{{ selected.title }}</h2>
          <p class="plan-meta">{{ t('Updated') }}: {{ selected.updated_at }}</p>
        </div>
        <button class="btn btn-ghost btn-sm" type="button" @click="selected = null">{{ t('Close') }}</button>
      </div>

      <div v-if="hasDemo(selected.items)" class="demo-badge">
        {{ t('Demo data — sample values, not live market quotes') }}
      </div>

      <div class="table-wrap">
        <table class="plan-table">
          <thead>
            <tr>
              <th>{{ t('Product') }}</th>
              <th>{{ t('Qty') }}</th>
              <th>{{ t('Target date') }}</th>
              <th>{{ t('Price now') }}</th>
              <th>{{ t('Predicted (range)') }}</th>
              <th>{{ t('Est. budget') }}</th>
              <th>{{ t('Signal') }}</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in selected.items" :key="item.id">
              <td><strong>{{ item.product_name }}</strong><small v-if="item.note">{{ item.note }}</small></td>
              <td>{{ item.quantity }} {{ item.unit }}</td>
              <td>{{ item.target_date || '—' }}</td>
              <td>{{ display(item.projection.current_unit_price) }}</td>
              <td>
                <template v-if="item.projection.predicted !== null">
                  {{ display(item.projection.predicted) }}
                  <small>({{ display(item.projection.low) }} – {{ display(item.projection.high) }})</small>
                </template>
                <template v-else>—</template>
              </td>
              <td>{{ display(item.projection.estimated_budget) }}</td>
              <td>
                <span class="rec-badge" :class="recommendation(item).cls">{{ recommendation(item).label }}</span>
                <small class="reason" :title="reasonText(item.projection.reason)">{{ reasonText(item.projection.reason) }}</small>
              </td>
              <td class="row-actions">
                <button class="btn btn-secondary btn-xs" type="button" @click="viewOffer(item)">{{ t('View offer') }}</button>
                <button class="btn btn-xs" type="button" @click="addToBasket(item)">{{ t('Add to basket') }}</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <p class="basis-note">{{ t('Forecast basis') }}: {{ selected.items?.[0]?.projection?.basis || '—' }}</p>
    </section>
  </div>
</template>

<style scoped>
.note-card { display: flex; gap: 10px; align-items: center; padding: 12px 14px; margin-bottom: 16px; color: var(--text-2); font-size: 12.5px; }
.note-card :deep(.app-icon) { color: var(--primary); flex: none; }
.section-title { font-size: 15px; margin: 0 0 12px; }
.editor-card { padding: 18px; margin-bottom: 18px; }
.editor-head { display: grid; grid-template-columns: 2fr 1fr; gap: 10px; margin-bottom: 14px; }
.item-row { display: grid; grid-template-columns: 2.2fr .7fr 1fr 1.2fr auto; gap: 8px; align-items: center; margin-bottom: 8px; }
.plain-input { width: 100%; min-width: 0; height: 38px; padding: 0 10px; border: 1px solid var(--border); border-radius: var(--radius-sm, 8px); background: var(--surface); color: var(--text); font-size: 12.5px; }
.icon-btn { border: 0; background: none; color: var(--text-3); font-size: 20px; cursor: pointer; }
.editor-actions { display: flex; gap: 8px; margin-top: 14px; flex-wrap: wrap; }
.preview-block { margin-top: 18px; padding-top: 14px; border-top: 1px solid var(--border); }
.preview-row { display: flex; flex-wrap: wrap; gap: 12px; padding: 8px 0; border-bottom: 1px solid var(--border); font-size: 12.5px; }
.demo-badge { display: inline-block; margin: 8px 0; padding: 4px 9px; border-radius: 999px; background: var(--warning-50, #fff7ed); color: var(--warning, #b45309); font-size: 11px; font-weight: 700; }
.plans-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 12px; margin-bottom: 18px; }
.plan-card { padding: 14px; cursor: pointer; }
.plan-card-top { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
.plan-title { font-size: 14px; }
.plan-meta { color: var(--text-3); font-size: 11.5px; margin: 6px 0; }
.plan-actions, .row-actions { display: flex; gap: 6px; flex-wrap: wrap; }
.detail-card { padding: 18px; }
.detail-head { display: flex; justify-content: space-between; align-items: flex-start; }
.table-wrap { overflow-x: auto; }
.plan-table { width: 100%; border-collapse: collapse; font-size: 12.5px; }
.plan-table th { text-align: left; color: var(--text-3); font-weight: 650; padding: 8px 10px; border-bottom: 1px solid var(--border); white-space: nowrap; }
.plan-table td { padding: 10px; border-bottom: 1px solid var(--border); vertical-align: top; }
.plan-table small { display: block; color: var(--text-3); font-size: 11px; margin-top: 2px; }
.rec-badge, .rec-buy, .rec-wait, .rec-monitor, .rec-none { display: inline-block; padding: 3px 8px; border-radius: 999px; font-size: 11px; font-weight: 700; white-space: nowrap; }
.rec-buy { background: var(--danger-50, #fef2f2); color: var(--danger, #dc2626); }
.rec-wait { background: var(--success-50, #ecfdf5); color: var(--success, #059669); }
.rec-monitor { background: var(--primary-50); color: var(--primary); }
.rec-none { background: var(--surface-2); color: var(--text-3); }
.reason { max-width: 220px; }
.basis-note { color: var(--text-3); font-size: 11.5px; margin-top: 12px; }
.error-text { color: var(--danger); font-size: 12.5px; }
.empty-state { text-align: center; display: flex; flex-direction: column; align-items: center; gap: 6px; color: var(--text-3); }
.empty-state h3 { color: var(--text); margin: 4px 0 0; }
.danger { color: var(--danger); }
@media (max-width: 1000px) { .item-row { grid-template-columns: 1fr 1fr; } .editor-head { grid-template-columns: 1fr; } }
</style>
