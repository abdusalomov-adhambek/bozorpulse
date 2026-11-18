import { defineStore } from 'pinia'
import { computed, ref, watch } from 'vue'

const STORAGE_KEY = 'bp_basket'

function load() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    const parsed = raw ? JSON.parse(raw) : []
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

function toPositiveNumber(value) {
  const n = Number(value)
  return Number.isFinite(n) && n >= 0 ? n : 0
}

export const useBasketStore = defineStore('basket', () => {
  const items = ref(load())

  watch(items, (value) => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(value))
  }, { deep: true })

  const count = computed(() => items.value.reduce((sum, item) => sum + Number(item.quantity || 1), 0))

  const grouped = computed(() => {
    const groups = new Map()
    for (const item of items.value) {
      const key = item.supplier_id || 'unknown'
      if (!groups.has(key)) {
        groups.set(key, {
          supplier_id: key,
          supplier_name: item.supplier_name || 'Supplier',
          items: [],
          total: 0,
        })
      }
      const group = groups.get(key)
      group.items.push(item)
      group.total += Number(item.price || 0) * Number(item.quantity || 1)
    }
    return [...groups.values()]
  })

  const total = computed(() => grouped.value.reduce((sum, group) => sum + group.total, 0))

  // Maximum quantity a customer may request for an item. Stock is authoritative;
  // the supplier's max order is an additional cap when present.
  function maxFor(item) {
    const caps = []
    const stock = toPositiveNumber(item?.stock)
    if (stock > 0) caps.push(stock)
    const maxOrder = toPositiveNumber(item?.max_order)
    if (maxOrder > 0) caps.push(maxOrder)
    return caps.length ? Math.min(...caps) : 0 // 0 means "no known limit"
  }

  function minFor(item) {
    const n = toPositiveNumber(item?.min_order)
    return n > 0 ? n : 1
  }

  function setStock(productId, stock) {
    const item = items.value.find((i) => i.product_id === productId)
    if (!item) return
    item.stock = toPositiveNumber(stock)
  }

  // `offer` carries the supplier/price/unit/quantity chosen in the AI chat,
  // so adding a shortlist item keeps the exact offer the user selected.
  // Returns { ok, quantity, reason } so the caller can inform the user when
  // the requested quantity was capped by available stock.
  function add(product, supplierName = 'Supplier', offer = null) {
    const requested = Number(offer?.quantity) > 0 ? Math.floor(Number(offer.quantity)) : 1
    const price = Number.isFinite(Number(offer?.price))
      ? Number(offer.price)
      : Number(product.effective_price ?? product.price ?? 0)
    const unit = offer?.unit || product.unit || 'pcs'
    const supplierId = offer?.supplier_id || product.supplier_id || 'unknown'
    const stock = toPositiveNumber(offer?.stock ?? product?.stock)
    const minOrder = toPositiveNumber(offer?.min_order ?? product?.min_order_qty)
    const maxOrder = toPositiveNumber(offer?.max_order ?? product?.max_order_qty)

    const existing = items.value.find((item) => item.product_id === product.id && item.supplier_id === supplierId)
    const already = existing ? Number(existing.quantity || 1) : 0
    const limits = { stock, min_order: minOrder, max_order: maxOrder }
    const cap = maxFor(limits)
    let quantity = requested + already
    let reason = ''
    if (cap > 0 && quantity > cap) {
      quantity = cap
      reason = 'stock'
    }
    if (quantity < minFor(limits)) quantity = minFor(limits)

    if (existing) {
      existing.quantity = quantity
      existing.price = price
      existing.unit = unit
      existing.stock = stock || existing.stock
      existing.min_order = minOrder || existing.min_order
      existing.max_order = maxOrder || existing.max_order
    } else {
      items.value.push({
        product_id: product.id,
        supplier_id: supplierId,
        supplier_name: offer?.supplier_name || supplierName,
        name: product.name,
        sku: product.sku || '',
        price,
        unit,
        quantity,
        stock,
        min_order: minOrder,
        max_order: maxOrder,
      })
    }
    return { ok: !reason, quantity, reason }
  }

  function setQuantity(productId, quantity) {
    const item = items.value.find((i) => i.product_id === productId)
    if (!item) return { ok: false, quantity: 0, reason: 'missing' }
    const next = Number(quantity)
    if (!Number.isFinite(next) || next <= 0) {
      remove(productId)
      return { ok: true, quantity: 0, reason: '' }
    }
    const cap = maxFor(item)
    let value = Math.floor(next)
    let reason = ''
    if (cap > 0 && value > cap) {
      value = cap
      reason = 'stock'
    }
    if (value < minFor(item)) value = minFor(item)
    item.quantity = value
    return { ok: !reason, quantity: value, reason }
  }

  function remove(productId) {
    items.value = items.value.filter((item) => item.product_id !== productId)
  }

  function removeSupplier(supplierId) {
    items.value = items.value.filter((item) => item.supplier_id !== supplierId)
  }

  function clear() {
    items.value = []
  }

  return { items, count, grouped, total, add, setQuantity, setStock, maxFor, minFor, remove, removeSupplier, clear }
})
