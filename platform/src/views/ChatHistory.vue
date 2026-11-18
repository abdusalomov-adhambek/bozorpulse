<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import api from '../api/client'
import { t } from '../i18n'
import { useAuthStore } from '../stores/auth'
import { useBasketStore } from '../stores/basket'
import AssistantChat from '../components/AssistantChat.vue'
import AppIcon from '../components/AppIcon.vue'

const auth = useAuthStore()
const basket = useBasketStore()
const router = useRouter()

const conversations = ref([])
const loading = ref(true)
const error = ref('')
const selectedId = ref(null)
const chatKey = ref(0)

const chatMode = computed(() => auth.role === 'customer' ? 'shopping' : 'general')

function normalize(res) {
  return Array.isArray(res) ? res : (res?.data || [])
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    conversations.value = normalize(await api.listConversations())
  } catch (err) {
    error.value = err?.message || 'Could not load chat history'
  } finally {
    loading.value = false
  }
}

function open(conversation) {
  selectedId.value = conversation.id
  chatKey.value += 1
}

function startNew() {
  selectedId.value = null
  chatKey.value += 1
}

async function remove(conversation) {
  if (!window.confirm(t('Delete this conversation?'))) return
  try {
    await api.deleteConversation(conversation.id)
    if (selectedId.value === conversation.id) startNew()
    await load()
  } catch (err) {
    error.value = err?.message || 'Could not delete the conversation'
  }
}

async function onConversationUpdated() {
  // Keep the list ordered by most recent activity without disturbing the chat.
  try { conversations.value = normalize(await api.listConversations()) } catch { /* keep current list */ }
}

async function onAssistantAction(action) {
  if (!action || action.kind !== 'add_to_basket') return
  try {
    const product = await api.get(`/products/${action.target}`)
    basket.add(product, action.meta?.supplier_name || 'Supplier', action.meta || null)
    router.push('/customer/basket')
  } catch { /* the assistant already reported a grounded error */ }
}

function formatDate(value) {
  if (!value) return ''
  const d = new Date(value)
  return Number.isNaN(d.getTime()) ? value : d.toLocaleString()
}

function statusClass(status) {
  if (status === 'demo') return 'badge badge-warning'
  if (status === 'insufficient') return 'badge badge-muted'
  return 'badge badge-success'
}

onMounted(load)
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t('Chat history') }}</h1>
        <p class="page-desc">{{ t('Reopen any conversation and continue where you left off.') }}</p>
      </div>
      <button class="btn" type="button" @click="startNew">
        <AppIcon name="sparkle" :size="15" /> {{ t('New conversation') }}
      </button>
    </div>

    <p v-if="error" class="alert alert-error">{{ error }}</p>

    <div class="history-layout">
      <section class="card history-list">
        <div class="history-list-head">
          <strong>{{ t('Conversations') }}</strong>
          <span class="history-count">{{ conversations.length }}</span>
        </div>

        <div v-if="loading" class="loading-block"><span class="spinner" /></div>
        <div v-else-if="!conversations.length" class="history-empty">
          <AppIcon name="chart" :size="26" />
          <p>{{ t('No conversations yet. Ask the assistant something to get started.') }}</p>
        </div>

        <article v-for="c in conversations" :key="c.id" class="history-item"
                 :class="{ active: c.id === selectedId }" @click="open(c)">
          <div class="history-item-top">
            <span class="history-intent">{{ c.intent || 'chat' }}</span>
            <span :class="statusClass(c.data_status)">{{ c.data_status || '—' }}</span>
          </div>
          <p class="history-preview">{{ c.preview || t('Conversation') }}</p>
          <div class="history-item-foot">
            <span>{{ c.locale?.toUpperCase() }}</span>
            <span>{{ formatDate(c.updated_at || c.created_at) }}</span>
            <button class="icon-btn history-delete" type="button" :aria-label="t('Delete')"
                    @click.stop="remove(c)">×</button>
          </div>
        </article>
      </section>

      <section class="card history-chat">
        <AssistantChat
          :key="chatKey"
          :mode="chatMode"
          :initial-conversation-id="selectedId"
          embedded
          @action="onAssistantAction"
          @conversation-updated="onConversationUpdated"
          @conversation-cleared="startNew"
        />
      </section>
    </div>
  </div>
</template>

<style scoped>
.history-layout { display: grid; grid-template-columns: minmax(240px, 340px) 1fr; gap: 16px; align-items: start; }
.history-list { padding: 14px; max-height: 760px; overflow-y: auto; }
.history-list-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 10px; }
.history-count { font-size: 11px; color: var(--text-3); background: var(--surface-2); border-radius: 999px; padding: 2px 8px; }
.history-empty { text-align: center; color: var(--text-3); font-size: 12.5px; padding: 28px 8px; }
.history-item { border: 1px solid var(--border); border-radius: 9px; padding: 10px; margin-bottom: 8px; cursor: pointer; background: var(--surface); }
.history-item:hover { border-color: var(--primary); }
.history-item.active { border-color: var(--primary); background: var(--primary-50, #eef7f4); box-shadow: inset 3px 0 0 var(--primary); }
.history-item-top { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
.history-intent { font-size: 11.5px; font-weight: 700; }
.history-preview { font-size: 12px; color: var(--text-2); margin: 6px 0; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
.history-item-foot { display: flex; align-items: center; gap: 8px; font-size: 10.5px; color: var(--text-3); }
.history-delete { margin-left: auto; font-size: 15px; line-height: 1; padding: 2px 6px; }
.history-chat { padding: 0; overflow: hidden; }
.history-chat :deep(.assistant.embedded) { height: 720px; border: 0; border-radius: 0; }
@media (max-width: 900px) { .history-layout { grid-template-columns: 1fr; } }
</style>
