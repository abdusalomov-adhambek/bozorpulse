<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { t } from '../i18n'

const props = defineProps({
  // Numbers, or objects with effective_price / price / value and an optional
  // observed_at / updated_at / date / label for the x-axis.
  points: { type: Array, default: () => [] },
  height: { type: Number, default: 190 },
})

let seq = 0
const gradientId = `chartGradient-${++seq}`

const wrap = ref(null)
const width = ref(600)
let observer = null

onMounted(() => {
  if (!wrap.value) return
  const initial = Math.round(wrap.value.clientWidth)
  if (initial > 0) width.value = initial
  if (typeof ResizeObserver === 'undefined') return
  observer = new ResizeObserver(([entry]) => {
    const w = Math.round(entry.contentRect.width)
    if (w > 0) width.value = w
  })
  observer.observe(wrap.value)
})
onBeforeUnmount(() => observer?.disconnect())

// One SVG unit = one CSS pixel, so text, dots and strokes keep their real size.
const W = computed(() => Math.max(240, width.value))
const H = computed(() => props.height || 190)
const PAD_L = 46
const PAD_R = 12
const PAD_T = 14
const PAD_B = 26

const entries = computed(() => props.points.map((p, index) => {
  if (typeof p === 'number') return { value: Number(p), raw: null, index }
  const n = p?.effective_price ?? p?.price ?? p?.value
  return { value: n === undefined || n === null ? null : Number(n), raw: p, index }
}).filter((e) => e.value !== null && !Number.isNaN(e.value)))

const values = computed(() => entries.value.map((e) => e.value))
const min = computed(() => values.value.length ? Math.min(...values.value) : 0)
const max = computed(() => values.value.length ? Math.max(...values.value) : 0)

// Add a small margin so the line never sits on the chart border and flat
// series still render as a readable line instead of a single pixel row.
const bounds = computed(() => {
  const span = (max.value - min.value) || (Math.abs(max.value) * 0.1) || 1
  // Prices cannot be negative — never draw a negative y-axis label.
  const lo = min.value >= 0 ? Math.max(0, min.value - span * 0.08) : min.value - span * 0.12
  return { lo, hi: max.value + span * 0.12 }
})

const yTicks = computed(() => {
  const { lo, hi } = bounds.value
  const plotTop = PAD_T
  const plotH = H.value - PAD_T - PAD_B
  return [0, 1, 2, 3].map((i) => {
    const value = hi - (i / 3) * (hi - lo)
    return { value, y: plotTop + (i / 3) * plotH }
  })
})

const coords = computed(() => {
  const vals = values.value
  const { lo, hi } = bounds.value
  if (!vals.length) return []
  const plotW = W.value - PAD_L - PAD_R
  const plotH = H.value - PAD_T - PAD_B
  const step = vals.length > 1 ? plotW / (vals.length - 1) : 0
  return vals.map((v, i) => ({
    x: vals.length > 1 ? PAD_L + i * step : PAD_L + plotW / 2,
    y: PAD_T + plotH - ((v - lo) / (hi - lo || 1)) * plotH,
    value: v,
  }))
})

const linePoints = computed(() => coords.value.map((c) => `${c.x.toFixed(1)},${c.y.toFixed(1)}`).join(' '))
const baseline = computed(() => H.value - PAD_B)
const areaPoints = computed(() => {
  if (!coords.value.length) return ''
  const first = coords.value[0]
  const last = coords.value[coords.value.length - 1]
  return `${first.x.toFixed(1)},${baseline.value} ${linePoints.value} ${last.x.toFixed(1)},${baseline.value}`
})

function formatValue(v) {
  return Number(v).toLocaleString(undefined, { maximumFractionDigits: 2 })
}

const xLabels = computed(() => {
  const items = coords.value
  if (!items.length) return []
  const dateText = (raw) => {
    const value = raw?.observed_at || raw?.captured_at || raw?.updated_at || raw?.date
    if (!value) return ''
    const d = new Date(value)
    return Number.isNaN(d.getTime()) ? String(value).slice(0, 10) : d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
  }
  // A cross-sectional list (one latest price per product) has almost identical
  // dates — labelling it as a date axis would be misleading. Use product names
  // in that case, otherwise use real observation dates.
  const dates = entries.value.map((e) => dateText(e.raw))
  const distinctDates = new Set(dates.filter(Boolean)).size
  const useDates = distinctDates > Math.max(2, entries.value.length * 0.3)

  const maxLabels = Math.max(2, Math.floor((W.value - PAD_L - PAD_R) / 105))
  const count = Math.min(maxLabels, items.length)
  const result = []
  for (let i = 0; i < count; i++) {
    const idx = count === 1 ? 0 : Math.round((i * (items.length - 1)) / (count - 1))
    const raw = entries.value[idx]?.raw
    let text = useDates ? dateText(raw) : ''
    if (!useDates) {
      text = String(raw?.product_name || raw?.name || raw?.label || '').split(' ')[0].slice(0, 12)
    }
    if (!text) text = `#${idx + 1}`
    result.push({ x: items[idx].x, text })
  }
  return result
})
</script>

<template>
  <div ref="wrap" class="chart-wrap">
    <div v-if="!values.length" class="chart-empty">{{ t('No price data available yet.') }}</div>
    <svg v-else class="chart-svg" :viewBox="`0 0 ${W} ${H}`" role="img" :aria-label="t('Price trend chart')">
      <defs>
        <linearGradient :id="gradientId" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stop-color="#157a6e" stop-opacity="0.26" />
          <stop offset="100%" stop-color="#157a6e" stop-opacity="0" />
        </linearGradient>
      </defs>

      <line v-for="(tick, i) in yTicks" :key="`g${i}`" class="chart-grid" :x1="PAD_L" :x2="W - PAD_R" :y1="tick.y" :y2="tick.y" />
      <text v-for="(tick, i) in yTicks" :key="`l${i}`" class="chart-label" :x="PAD_L - 7" :y="tick.y + 3.5" text-anchor="end">{{ formatValue(tick.value) }}</text>

      <polygon class="chart-area" :points="areaPoints" :fill="`url(#${gradientId})`" />
      <polyline class="chart-line" :points="linePoints" />
      <circle v-for="(c, i) in coords" :key="`d${i}`" class="chart-dot" :cx="c.x" :cy="c.y" r="2.6">
        <title>{{ formatValue(c.value) }}</title>
      </circle>

      <text v-for="(label, i) in xLabels" :key="`x${i}`" class="chart-label chart-xlabel" :x="label.x" :y="H - 7" text-anchor="middle">{{ label.text }}</text>
    </svg>
  </div>
</template>
