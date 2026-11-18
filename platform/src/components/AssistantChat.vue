<script setup>
import { computed, nextTick, onMounted, ref } from 'vue'
import api from '../api/client'
import { locale, t } from '../i18n'
import AppIcon from './AppIcon.vue'

const props = defineProps({
  mode: { type: String, default: 'general' }, // general | shopping
  embedded: { type: Boolean, default: false },
  initialConversationId: { type: String, default: null },
})
const emit = defineEmits(['action', 'close', 'conversation-updated', 'conversation-cleared'])

const messages = ref([])
const input = ref('')
const busy = ref(false)
const error = ref('')
const conversationId = ref(null)
const scroller = ref(null)

const intro = computed(() => props.mode === 'shopping'
  ? t('What do you need? (product, quantity, region)')
  : t('Ask about prices, suppliers or orders…'))

function optionCards(message) {
  if (message?.intent !== 'buying_advice') return []
  for (const call of message.tool_calls || []) {
    if (call.name !== 'compare_offers' || !call.result) continue
    try {
      const parsed = typeof call.result === 'string' ? JSON.parse(call.result) : call.result
      return Array.isArray(parsed?.options) ? parsed.options : []
    } catch { /* ignore malformed evidence */ }
  }
  return []
}

// One selected offer per assistant message. Clicking a card marks it and
// retargets the "Add to basket" action at that supplier/offer.
const selectedOffers = ref({})

function isSelected(msgIndex, offerIndex) {
  return selectedOffers.value[`${msgIndex}:${offerIndex}`] === true
}

function selectOption(msgIndex, offerIndex) {
  const key = `${msgIndex}:${offerIndex}`
  selectedOffers.value[key] = !selectedOffers.value[key]
}

function actionsFor(message, msgIndex) {
  const actions = message.actions || []
  let chosen = null
  optionCards(message).forEach((offer, oi) => {
    if (isSelected(msgIndex, oi)) chosen = offer
  })
  if (!chosen) return actions
  return actions.map((action) => {
    if (action.kind !== 'add_to_basket') return action
    return {
      ...action,
      target: chosen.product_id || action.target,
      meta: {
        ...(action.meta || {}),
        supplier_id: chosen.supplier_id || action.meta?.supplier_id,
        supplier_name: chosen.supplier_name || action.meta?.supplier_name,
        price: String(chosen.effective_price ?? action.meta?.price ?? ''),
        unit: chosen.unit || action.meta?.unit,
      },
    }
  })
}

const reasonLabels = {
  'Lowest total for this quantity': { uz: 'Eng past jami narx', ru: 'Самая низкая итоговая цена' },
  'Fast delivery': { uz: 'Tez yetkazib berish', ru: 'Быстрая доставка' },
  'Enough stock available': { uz: 'Zaxira yetarli', ru: 'Достаточно на складе' },
  'Comparable offer in the catalog': { uz: 'Katalogdagi mos taklif', ru: 'Сопоставимое предложение в каталоге' },
}
function reasonLabel(reason, messageLocale) {
  const loc = String(messageLocale || locale.value || 'en').slice(0, 2)
  return reasonLabels[reason]?.[loc] || reason
}

function statusClass(status) {
  if (status === 'demo') return 'badge badge-warning'
  if (status === 'insufficient') return 'badge badge-muted'
  return 'badge badge-success'
}

// The model replies use light markdown. Render it as plain text without
// pulling in a markdown dependency: strip emphasis/backticks and turn
// "- " bullets into "• " so the bubbles stay readable in a narrow drawer.
function formatReply(text) {
  if (!text) return ''
  return String(text)
    .replace(/\*\*/g, '')
    .replace(/`/g, '')
    .split('\n')
    .map((line) => line.replace(/^\s*[-*]\s+/, '• ').replace(/^\s*>\s+/, ''))
    .join('\n')
}

async function scrollToBottom() {
  await nextTick()
  if (scroller.value) scroller.value.scrollTop = scroller.value.scrollHeight
}

const loadingHistory = ref(false)

// Restore an existing conversation so the user can continue it. Only the
// transcript is stored server-side; new replies regain full cards/actions.
async function loadHistory(id) {
  if (!id) return
  loadingHistory.value = true
  error.value = ''
  try {
    const detail = await api.getConversation(id)
    conversationId.value = detail.id || id
    messages.value = (detail.messages || []).map((m) => m.role === 'assistant'
      ? {
          role: 'assistant', reply: m.content, locale: m.locale || detail.locale,
          data_status: m.data_status, intent: m.intent,
          tool_calls: m.tool_calls || [], sources: m.sources || [],
        }
      : { role: m.role, content: m.content })
    await scrollToBottom()
  } catch (err) {
    error.value = err?.message || 'Could not load this conversation'
  } finally {
    loadingHistory.value = false
  }
}

onMounted(() => {
  if (props.initialConversationId) loadHistory(props.initialConversationId)
})

async function send(text) {
  const content = String(text ?? input.value).trim()
  if (!content || busy.value) return
  input.value = ''
  error.value = ''
  messages.value.push({ role: 'user', content })
  busy.value = true
  await scrollToBottom()
  try {
    const reply = await api.aiChat({
      message: content,
      conversation_id: conversationId.value || undefined,
      locale: locale.value,
      mode: props.mode,
    })
    conversationId.value = reply.conversation_id || null
    messages.value.push({ role: 'assistant', ...reply })
    if (conversationId.value) emit('conversation-updated', conversationId.value)
  } catch (err) {
    error.value = err?.message || 'Request failed'
    messages.value.push({
      role: 'assistant',
      reply: t('I couldn\'t find comparable offers right now.') || "I couldn't complete that request.",
      data_status: 'insufficient',
      tool_calls: [],
      sources: [],
    })
  } finally {
    busy.value = false
    await scrollToBottom()
  }
}

function newConversation() {
  messages.value = []
  conversationId.value = null
  error.value = ''
  emit('conversation-cleared')
}

function runAction(action) {
  emit('action', action)
  if (props.embedded) return
  if (action.kind === 'link' || action.kind === 'go_to_orders') {
    window.location.assign(action.target)
  }
}

function onKeydown(e) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    send()
  }
}
</script>

<template>
  <div class="assistant" :class="{ embedded }">
    <div class="assistant-head">
      <div class="assistant-title">
        <AppIcon name="chart" :size="16" />
        <strong>{{ t('AI assistant') }}</strong>
      </div>
      <div class="assistant-head-actions">
        <button type="button" class="assistant-link" @click="newConversation">{{ t('New conversation') }}</button>
        <button v-if="!embedded" type="button" class="icon-btn" :aria-label="t('Close')" @click="emit('close')">×</button>
      </div>
    </div>

    <div ref="scroller" class="assistant-messages">
      <div v-if="loadingHistory" class="assistant-loading"><span class="spinner" /> {{ t('Loading conversation…') }}</div>
      <div v-else-if="!messages.length" class="assistant-empty">
        <AppIcon name="chart" :size="26" />
        <p>{{ intro }}</p>
        <div class="assistant-chips">
          <button v-for="s in (mode === 'shopping' ? ['I need 100 bags of cement', 'What should I buy?'] : ['How much is cement?', 'Who sells cement?'])" :key="s" type="button" class="chip" @click="send(s)">{{ s }}</button>
        </div>
      </div>

      <template v-else>
        <div v-for="(msg, i) in messages" :key="i" class="assistant-row" :class="msg.role">
          <div class="bubble">
            <div class="bubble-text whitespace-pre-wrap">{{ formatReply(msg.reply || msg.content) }}</div>

            <div v-if="msg.role === 'assistant' && msg.data_status" class="bubble-badges">
              <span :class="statusClass(msg.data_status)">
                {{ msg.data_status === 'demo' ? t('Demo data — sample values, not live market quotes') : (msg.data_status === 'insufficient' ? t('Not enough data') : '') }}
              </span>
            </div>

            <div v-if="optionCards(msg).length" class="option-list">
              <div v-for="(option, oi) in optionCards(msg)" :key="oi" class="option-card" :class="{ selected: isSelected(i, oi) }">
                <div class="option-head">
                  <div class="option-name">{{ option.product_name }}</div>
                  <button type="button" class="select-offer" :class="{ on: isSelected(i, oi) }" @click.stop="selectOption(i, oi)">
                    {{ isSelected(i, oi) ? '✓ Selected' : 'Select' }}
                  </button>
                </div>
                <div class="option-sub">{{ option.supplier_name }}</div>
                <div class="option-price">
                  <strong>{{ Number(option.effective_price || 0).toLocaleString() }}</strong>
                  <span>/ {{ option.unit || 'pcs' }}</span>
                </div>
                <div class="option-meta">{{ option.stock }} in stock · {{ option.delivery_min_days }}–{{ option.delivery_max_days }} days</div>
                <div v-if="option.why?.length" class="option-why">{{ option.why.map(r => reasonLabel(r, msg.locale)).join(' · ') }}</div>
                <div v-if="option.forecast" class="option-forecast">
                  📈 ~{{ Number(option.forecast.predicted).toFixed(2) }} / {{ option.unit || 'pcs' }} in {{ option.forecast.horizon_days }}d
                  <span>({{ Number(option.forecast.low).toFixed(2) }}–{{ Number(option.forecast.high).toFixed(2) }})</span>
                  <span v-if="option.forecast.weather"> · {{ option.forecast.weather.temperature_c }}°C, {{ option.forecast.weather.precipitation_mm }}mm rain</span>
                </div>
              </div>
            </div>

            <div v-if="msg.role === 'assistant' && (msg.actions || []).length" class="assistant-actions">
              <button v-for="(action, ai) in actionsFor(msg, i)" :key="ai" type="button" class="btn btn-sm" :class="ai === 0 ? '' : 'btn-secondary'" @click="runAction(action)">
                {{ action.label }}
              </button>
            </div>

            <div v-if="msg.role === 'assistant' && (msg.suggestions || []).length" class="assistant-chips">
              <button v-for="s in msg.suggestions" :key="s" type="button" class="chip" @click="send(s)">{{ s }}</button>
            </div>

            <details v-if="msg.role === 'assistant' && (msg.tool_calls || []).length" class="assistant-evidence">
              <summary>{{ t('Tools used') }} · {{ t('Sources') }} ({{ msg.tool_calls.length }})</summary>
              <ul>
                <li v-for="(call, ci) in msg.tool_calls" :key="ci">
                  <code>{{ call.name }}</code> · {{ call.duration_ms }} ms
                  <span v-if="call.error" class="evidence-error">· {{ call.error }}</span>
                </li>
              </ul>
              <ul v-if="(msg.sources || []).length" class="evidence-sources">
                <li v-for="(src, si) in msg.sources" :key="si">
                  {{ src.type }} #{{ src.id }} <span v-if="src.label">— {{ src.label }}</span>
                  <span v-if="src.is_demo" class="evidence-demo">demo</span>
                </li>
              </ul>
            </details>
          </div>
        </div>
      </template>
    </div>

    <div v-if="error" class="assistant-error">{{ error }}</div>
    <div class="assistant-composer">
      <textarea v-model="input" rows="2" :placeholder="intro" :disabled="busy" @keydown="onKeydown" />
      <button class="btn" type="button" :disabled="busy || !input.trim()" @click="send()">
        {{ busy ? t('Thinking…') : t('Send') }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.assistant { display: flex; flex-direction: column; height: 100%; min-height: 0; background: var(--surface); }
.assistant.embedded { height: 520px; border: 1px solid var(--border); border-radius: var(--radius); overflow: hidden; }
.assistant-head { display: flex; align-items: center; justify-content: space-between; padding: 12px 14px; border-bottom: 1px solid var(--border); }
.assistant-title { display: flex; align-items: center; gap: 8px; font-size: 14px; }
.assistant-head-actions { display: flex; align-items: center; gap: 8px; }
.assistant-link { background: none; border: 0; color: var(--primary); font-size: 12px; cursor: pointer; padding: 0; }
.assistant-messages { flex: 1 1 auto; min-height: 0; overflow-y: auto; padding: 14px; display: flex; flex-direction: column; gap: 12px; }
.assistant-empty { margin: auto; text-align: center; color: var(--text-3); display: flex; flex-direction: column; align-items: center; gap: 10px; }
.assistant-loading { margin: auto; display: flex; align-items: center; gap: 8px; color: var(--text-3); font-size: 12px; }
/* Rows must not shrink below their content in a flex-column scroll area,
   otherwise long replies overflow the row and overlap the composer. */
.assistant-row { display: flex; flex: 0 0 auto; height: auto; min-height: 0; min-width: 0; }
.assistant-row.user { justify-content: flex-end; }
.assistant-row.assistant { justify-content: flex-start; }
.bubble { min-width: 0; max-width: 92%; padding: 10px 12px; border-radius: 12px; background: var(--surface-2); border: 1px solid var(--border); font-size: 13px; line-height: 1.5; overflow-wrap: anywhere; word-break: break-word; }
.assistant-row.user .bubble { background: var(--primary); color: #fff; border-color: var(--primary); }
.whitespace-pre-wrap { white-space: pre-wrap; }
.bubble-badges { margin-top: 8px; }
.badge { display: inline-block; font-size: 10.5px; padding: 2px 7px; border-radius: 999px; }
.badge-warning { background: var(--warning-50, #fff7e6); color: var(--warning, #b7791f); }
.badge-muted { background: var(--surface-3, #eee); color: var(--text-3); }
.badge-success { background: var(--success-50, #eafaf1); color: var(--success, #276749); }
.assistant-chips { display: flex; flex-wrap: wrap; gap: 6px; margin-top: 8px; }
.chip { border: 1px solid var(--border); background: var(--surface); color: var(--primary); border-radius: 999px; padding: 4px 10px; font-size: 11.5px; cursor: pointer; }
.assistant-actions { display: flex; flex-wrap: wrap; gap: 6px; margin-top: 10px; }
.option-list { display: grid; gap: 8px; margin-top: 10px; }
.option-card { border: 1px solid var(--border); border-radius: 8px; padding: 9px 10px; background: var(--surface); cursor: pointer; }
.option-card.selected { border-color: var(--primary); background: var(--primary-50, #eef7f4); box-shadow: inset 3px 0 0 var(--primary); }
.option-head { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
.select-offer { border: 1px solid var(--primary); background: var(--surface); color: var(--primary); border-radius: 999px; padding: 2px 9px; font-size: 10.5px; cursor: pointer; flex-shrink: 0; }
.select-offer.on { background: var(--primary); color: #fff; }
.option-forecast { margin-top: 5px; font-size: 11px; color: var(--text-2); }
.option-name { font-weight: 700; font-size: 12.5px; }
.option-sub, .option-meta, .option-why { color: var(--text-3); font-size: 11px; }
.option-price strong { font-size: 15px; }
.option-price span { color: var(--text-3); font-size: 11px; }
.assistant-evidence { margin-top: 9px; font-size: 11px; color: var(--text-3); }
.assistant-evidence summary { cursor: pointer; }
.assistant-evidence ul { margin: 6px 0 0; padding-left: 16px; }
.evidence-error { color: var(--danger, #c53030); }
.evidence-demo { margin-left: 6px; color: var(--warning, #b7791f); }
.assistant-composer { display: flex; gap: 8px; align-items: flex-end; padding: 10px 12px; border-top: 1px solid var(--border); }
.assistant-composer textarea { flex: 1; resize: vertical; min-height: 44px; font: inherit; font-size: 13px; padding: 8px; border: 1px solid var(--border); border-radius: 8px; background: var(--surface); color: var(--text); }
.assistant-error { padding: 6px 12px; color: var(--danger, #c53030); font-size: 12px; }
</style>
