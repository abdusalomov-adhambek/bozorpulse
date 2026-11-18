<script setup>
import { money } from '../../utils/money'
import { onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import api from '../../api/client'
import { t } from '../../i18n'
import { useBasketStore } from '../../stores/basket'
import { useAppStore } from '../../stores/app'
import AppIcon from '../../components/AppIcon.vue'
import AppSelect from '../../components/ui/AppSelect.vue'
import NumberInput from '../../components/ui/NumberInput.vue'
import EmptyState from '../../components/EmptyState.vue'

const basket = useBasketStore()
const app = useAppStore()
const router = useRouter()

const warehouses = ref([])
const warehousesBySupplier = reactive({})
const busy = ref('')
const resolved = ref({})

function priceFor(item) {
  return Number(resolved.value[item.product_id]?.effective_price ?? item.price ?? 0)
}
function limitsFor(item) {
  return resolved.value[item.product_id] || null
}
function deliveryFor(item) {
  const r = resolved.value[item.product_id]
  if (!r) return ''
  if (!r.delivery_min_days && !r.delivery_max_days) return ''
  return r.delivery_min_days === r.delivery_max_days ? `${r.delivery_min_days}d` : `${r.delivery_min_days}–${r.delivery_max_days}d`
}

async function resolvePrices() {
  const warehouseId = warehouses.value[0]?.id
  if (!warehouseId || !basket.items.length) return
  const entries = await Promise.all(basket.items.map(async (item) => {
    try {
      const [price, product] = await Promise.all([
        api.get(`/products/${item.product_id}/price`, {
          warehouse_id: warehouseId, quantity: Number(item.quantity)
        }, { auth: false }),
        api.get(`/products/${item.product_id}`, {}, { auth: false }).catch(() => null),
      ])
      if (product && product.stock !== undefined) basket.setStock(item.product_id, product.stock)
      return [item.product_id, price]
    } catch {
      return [item.product_id, null]
    }
  }))
  resolved.value = Object.fromEntries(entries.filter(([, v]) => v))
}

watch(
  () => basket.items.map((item) => `${item.product_id}:${item.quantity}`).join('|') + '|' + (warehouses.value[0]?.id || ''),
  () => { resolvePrices() },
)

onMounted(async () => {
  try {
    const res = await api.get('/warehouses')
    warehouses.value = res?.data || []
    for (const group of basket.grouped) {
      warehousesBySupplier[group.supplier_id] = warehouses.value[0]?.id || ''
    }
  } catch {
    warehouses.value = []
  }
  await resolvePrices()
})

function optionsFor(group) {
  // Customers choose their own destination warehouse; the supplier group only
  // determines which seller receives the resulting order.
  return warehouses.value.map((w) => ({ value: w.id, label: `${w.name} — ${w.city || w.address || ''}` }))
}

function groupTotal(group) {
  return group.items.reduce((sum, item) => sum + priceFor(item) * Number(item.quantity || 1), 0)
}
function invalidQuantity(item) {
  const q = Number(item.quantity)
  const cap = basket.maxFor(item)
  return q < basket.minFor(item) || (cap > 0 && q > cap)
}
const hasInvalid = () => basket.items.some(invalidQuantity)

async function createOrder(group) {
  const warehouseId = warehousesBySupplier[group.supplier_id] || warehouses.value[0]?.id
  if (!warehouseId) {
    app.error(t('Create a warehouse before placing an order.'))
    return
  }
  busy.value = group.supplier_id
  try {
    const order = await api.post('/orders', {
      supplier_id: group.supplier_id,
      warehouse_id: warehouseId,
      items: group.items.map((item) => ({ product_id: item.product_id, quantity: Number(item.quantity) })),
    })
    basket.removeSupplier(group.supplier_id)
    app.success(`Order #${order.id} sent to ${group.supplier_name}`)
  } catch (err) {
    app.error(err?.message || 'Could not create order.')
  } finally {
    busy.value = ''
  }
}

async function createAll() {
  for (const group of [...basket.grouped]) {
    await createOrder(group)
  }
  if (!basket.grouped.length) router.push('/customer/orders')
}
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t('Basket') }}</h1>
        <p class="page-desc">{{ t('Products are grouped by supplier. Each supplier receives a separate order.') }}</p>
      </div>
      <RouterLink to="/customer/market" class="btn btn-secondary">
        <AppIcon name="market" :size="16" /> {{ t('Market') }}
      </RouterLink>
    </div>

    <EmptyState v-if="!basket.grouped.length" :title="t('Your basket is empty')" description="Browse the market and add products you want to order.">
      <RouterLink to="/customer/market" class="btn">{{ t('Market') }}</RouterLink>
    </EmptyState>

    <template v-else>
      <div class="supplier-orders">
        <section v-for="group in basket.grouped" :key="group.supplier_id" class="card supplier-order">
          <div class="card-head">
            <div>
              <div class="card-title"><AppIcon name="users" :size="16" /> {{ group.supplier_name }}</div>
              <div class="card-sub">{{ group.items.length }} product(s) · {{ money(groupTotal(group)) }}</div>
            </div>
            <button class="btn btn-sm btn-danger-ghost" type="button" @click="basket.removeSupplier(group.supplier_id)">
              {{ t('Remove') }}
            </button>
          </div>

          <div class="supplier-order-body">
            <div v-for="item in group.items" :key="item.product_id" class="basket-item">
              <div class="basket-item-name">
                <strong>{{ item.name }}</strong>
                <small>{{ item.sku }} · {{ money(priceFor(item)) }} / {{ item.unit }}<template v-if="deliveryFor(item)"> · {{ deliveryFor(item) }}</template></small>
                <small v-if="invalidQuantity(item)" class="field-error">{{ t('Quantity outside the allowed order range') }}</small>
              </div>
              <NumberInput
                class="basket-qty"
                size="sm"
                :model-value="Number(item.quantity)"
                :min="1"
                :max="basket.maxFor(item) || undefined"
                @update:model-value="basket.setQuantity(item.product_id, $event)"
              />
              <strong class="line-total">{{ money(priceFor(item) * Number(item.quantity)) }}</strong>
              <button type="button" class="remove-item" :aria-label="t('Remove item')" @click="basket.remove(item.product_id)">×</button>
            </div>

            <div class="order-actions">
              <label class="warehouse-field">
                <span class="field-label">{{ t('Deliver to warehouse') }}</span>
                <AppSelect
                  v-model="warehousesBySupplier[group.supplier_id]"
                  :options="optionsFor(group)"
                  :placeholder="t('Select a warehouse')"
                />
              </label>
              <button class="btn" type="button" :disabled="busy === group.supplier_id || !warehouses.length || hasInvalid()" @click="createOrder(group)">
                {{ busy === group.supplier_id ? t('Creating…') : t(t('Create order')) }}
              </button>
            </div>
            <p v-if="!warehouses.length" class="field-error">{{ t('No warehouses yet.') }} <RouterLink to="/customer/warehouses">{{ t('Create one first') }}</RouterLink>.</p>
          </div>
        </section>
      </div>

      <div class="basket-summary">
        <div><span>{{ basket.count }} item(s) · {{ basket.grouped.length }} supplier(s)</span><strong>{{ money(basket.grouped.reduce((sum, g) => sum + groupTotal(g), 0)) }}</strong></div>
        <button class="btn" type="button" :disabled="!!busy || hasInvalid()" @click="createAll">{{ t('Create all orders') }}</button>
      </div>
    </template>
  </div>
</template>

<style scoped>
.supplier-orders { display: grid; gap: 14px; }
.card-title { display: flex; align-items: center; gap: 8px; }
.supplier-order-body { padding: 16px 22px 20px; }
.basket-item {
  display: grid; grid-template-columns: 1fr auto auto auto; align-items: center; gap: 14px;
  padding: 10px 0; border-bottom: 1px solid var(--border);
}
.basket-item-name { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.basket-item-name small { color: var(--text-3); font-size: 11px; }
.basket-qty { width: 96px; }
.qty { display: inline-flex; align-items: center; border: 1px solid var(--border-strong); border-radius: 8px; overflow: hidden; }
.qty button { border: 0; background: var(--surface-2); width: 28px; height: 28px; cursor: pointer; font-size: 14px; color: var(--text-2); }
.qty button:hover { background: var(--primary-50); color: var(--primary); }
.qty span { min-width: 30px; text-align: center; font-size: 12.5px; font-weight: 700; }
.line-total { font-size: 13px; min-width: 70px; text-align: right; }
.remove-item { border: 0; background: none; color: var(--text-3); font-size: 18px; cursor: pointer; }
.remove-item:hover { color: var(--danger); }
.order-actions { display: flex; align-items: flex-end; gap: 12px; margin-top: 16px; }
.warehouse-field { flex: 1; display: flex; flex-direction: column; gap: 6px; }
.basket-summary {
  display: flex; align-items: center; justify-content: space-between; gap: 16px;
  margin-top: 18px; padding: 16px 20px; border: 1px solid var(--border);
  border-radius: var(--radius); background: var(--surface);
}
.basket-summary div { display: flex; flex-direction: column; gap: 2px; font-size: 12px; color: var(--text-2); }
.basket-summary strong { font-size: 20px; color: var(--text); }
@media (max-width: 720px) {
  .basket-item { grid-template-columns: 1fr auto; }
  .order-actions { flex-direction: column; align-items: stretch; }
}
</style>
