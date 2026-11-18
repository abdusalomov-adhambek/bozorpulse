<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useBasketStore } from '../stores/basket'
import api from '../api/client'
import { NAV_CONFIG } from '../router'
import CommandPalette from '../components/CommandPalette.vue'
import LanguageSwitcher from '../components/LanguageSwitcher.vue'
import Modal from '../components/Modal.vue'
import AppIcon from '../components/AppIcon.vue'
import AssistantChat from '../components/AssistantChat.vue'
import { t } from '../i18n'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const basket = useBasketStore()

const sidebarOpen = ref(false)
const expanded = ref({})
const paletteOpen = ref(false)
const assistantOpen = ref(false)
const collapsed = ref(localStorage.getItem('bp_sidebar_collapsed') === '1')

const nav = computed(() => NAV_CONFIG[auth.role] || [])
const commandItems = computed(() => {
  const out = []
  for (const item of nav.value) {
    if (item.children) {
      for (const child of item.children) {
        out.push({ label: t(child.label), to: child.to, group: t(item.label), icon: '↳' })
      }
    } else {
      out.push({ label: t(item.label), to: item.to, group: roleLabel.value, icon: '→' })
    }
  }
  return out
})
const roleLabel = computed(() => {
  const labels = { admin: 'Administrator', supplier: 'Supplier', customer: 'Customer' }
  return t(labels[auth.role] || (auth.role || '').replace(/^\w/, (c) => c.toUpperCase()))
})
const pageTitle = computed(() => route.meta.title || 'Dashboard')
const initials = computed(() => {
  const name = auth.user?.username || auth.user?.company_name || '?'
  return name.slice(0, 2).toUpperCase()
})

const icons = {
  dashboard: '<rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/>',
  orders: '<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 2v6h6M8 13h8M8 17h5"/>',
  suppliers: '<path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/>',
  customers: '<path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75"/>',
  payments: '<rect x="2" y="5" width="20" height="14" rx="2"/><path d="M2 10h20"/>',
  products: '<path d="m21 8-9-5-9 5 9 5 9-5z"/><path d="M3 8v8l9 5 9-5V8M12 13v8"/>',
  warehouses: '<path d="M3 21V8l9-5 9 5v13M3 21h18M9 21v-8h6v8"/>',
  cabinet: '<circle cx="12" cy="8" r="4"/><path d="M4 21a8 8 0 0 1 16 0"/>',
  settings: '<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .34 1.88l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.7 1.7 0 0 0-1.88-.34 1.7 1.7 0 0 0-1.03 1.56V21a2 2 0 1 1-4 0v-.09A1.7 1.7 0 0 0 8.94 19.4a1.7 1.7 0 0 0-1.88.34l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06A1.7 1.7 0 0 0 4.6 15a1.7 1.7 0 0 0-1.56-1.03H3a2 2 0 1 1 0-4h.09A1.7 1.7 0 0 0 4.6 8.94a1.7 1.7 0 0 0-.34-1.88l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06A1.7 1.7 0 0 0 9 4.6a1.7 1.7 0 0 0 1.03-1.56V3a2 2 0 1 1 4 0v.09A1.7 1.7 0 0 0 15.06 4.6a1.7 1.7 0 0 0 1.88-.34l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06A1.7 1.7 0 0 0 19.4 9v0a1.7 1.7 0 0 0 1.56 1.03H21a2 2 0 1 1 0 4h-.09A1.7 1.7 0 0 0 19.4 15z"/>'
}

function isActive(to) {
  return route.path === to
}
function groupActive(item) {
  if (!item.children) return isActive(item.to)
  return item.children.some((child) => route.path === child.to)
}
function toggleGroup(label) {
  expanded.value[label] = !expanded.value[label]
}
function closeSidebar() {
  sidebarOpen.value = false
}
const logoutOpen = ref(false)
function requestLogout() {
  logoutOpen.value = true
}
function confirmLogout() {
  logoutOpen.value = false
  auth.logout()
  router.push({ name: 'login' })
}
function runCommand(item) {
  router.push(item.to)
}
async function onAssistantAction(action) {
  // The floating assistant has no Market page to delegate to, so add the
  // selected offer directly using the same basket store.
  if (!action || action.kind !== 'add_to_basket') return
  try {
    const product = await api.get(`/products/${action.target}`)
    basket.add(product, action.meta?.supplier_name || 'Supplier', action.meta || null)
    router.push('/customer/basket')
  } catch { /* the assistant already shows a grounded error on failure */ }
}
function toggleCollapse() {
  collapsed.value = !collapsed.value
  localStorage.setItem('bp_sidebar_collapsed', collapsed.value ? '1' : '0')
}
function onShortcut(e) {
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    paletteOpen.value = !paletteOpen.value
  }
}
onMounted(() => window.addEventListener('keydown', onShortcut))
onBeforeUnmount(() => window.removeEventListener('keydown', onShortcut))

watch(() => route.fullPath, () => { sidebarOpen.value = false })
</script>

<template>
  <div class="layout" :class="{ collapsed }">
    <!-- Mobile backdrop -->
    <div class="backdrop" :class="{ show: sidebarOpen }" @click="closeSidebar" />

    <aside class="sidebar" :class="{ open: sidebarOpen }">
      <div class="sidebar-brand">
        <div class="brand-mark">BP</div>
        <span>BozorPulse</span>
      </div>


      <nav class="sidebar-nav">
        <template v-for="item in nav" :key="item.label">
          <!-- Plain nav item -->
          <RouterLink v-if="!item.children" :to="item.to" class="nav-item" :class="{ active: isActive(item.to) }">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" v-html="icons[item.icon] || icons.dashboard" />
            <span>{{ t(item.label) }}<span v-if="item.label === 'Basket' && basket.count" class="nav-count">{{ basket.count }}</span></span>
          </RouterLink>

          <!-- Nav group with sub-items -->
          <template v-else>
            <button type="button" class="nav-item" :class="{ active: groupActive(item) }" @click="toggleGroup(item.label)">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" v-html="icons[item.icon] || icons.settings" />
              <span>{{ t(item.label) }}</span>
              <svg class="chevron" :class="{ open: expanded[item.label] || groupActive(item) }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg>
            </button>
            <div v-show="expanded[item.label] || groupActive(item)" class="nav-sub">
              <RouterLink v-for="child in item.children" :key="child.to" :to="child.to" class="nav-sub-link" :class="{ active: isActive(child.to) }">
                {{ t(child.label) }}
              </RouterLink>
            </div>
          </template>
        </template>
      </nav>

      <div class="sidebar-foot">
        <div class="sidebar-user">
          <div class="avatar">{{ initials }}</div>
          <div class="who">
            <strong>{{ auth.displayName }}</strong>
            <small>{{ roleLabel }}</small>
          </div>
          <button class="sidebar-logout" type="button" :title="t('Log out')" @click="requestLogout">
            <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9"/></svg>
          </button>
        </div>
      </div>
    </aside>

    <div class="main">
      <header class="topbar">
        <button class="icon-btn menu-toggle" type="button" :aria-label="t('Open menu')" @click="sidebarOpen = true">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M3 6h18M3 12h18M3 18h18"/></svg>
        </button>

        <div class="breadcrumb">
          <span>{{ t('Home') }}</span>
          <span class="sep">/</span>
          <span>{{ roleLabel }}</span>
          <span class="sep">/</span>
          <strong>{{ t(pageTitle) }}</strong>
        </div>

        <div class="topbar-right">
          <LanguageSwitcher />
          <button class="cmd-trigger" type="button" @click="paletteOpen = true">
            <svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/></svg>
            <span>{{ t('Search or jump to…') }}</span>
            <kbd>⌘ K</kbd>
          </button>
          <button class="icon-btn collapse-btn" type="button" :title="collapsed ? t('Expand sidebar') : t('Collapse sidebar')" @click="toggleCollapse">
            <svg viewBox="0 0 24 24" width="17" height="17" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><rect x="3" y="4" width="18" height="16" rx="2"/><path d="M9 4v16"/></svg>
          </button>
          <span class="topbar-greeting">{{ t('Welcome back,') }} <strong>{{ auth.displayName }}</strong></span>
          <div class="avatar">{{ initials }}</div>
        </div>
      </header>

      <main class="content">
        <RouterView />
      </main>
    </div>

    <CommandPalette v-model:open="paletteOpen" :items="commandItems" @select="runCommand" />

    <!-- AI assistant drawer -->
    <button class="assistant-fab" type="button" :title="t('AI assistant')" :aria-label="t('AI assistant')" @click="assistantOpen = !assistantOpen">
      <svg viewBox="0 0 24 24" width="21" height="21" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 11.5a8.4 8.4 0 0 1-9 8.4 9.8 9.8 0 0 1-3.3-.6L3 21l1.7-5.3A8.4 8.4 0 0 1 12 3.1a8.4 8.4 0 0 1 9 8.4z"/></svg>
    </button>
    <div v-if="assistantOpen" class="assistant-overlay" @click="assistantOpen = false" />
    <transition name="assistant-slide">
      <aside v-if="assistantOpen" class="assistant-drawer">
        <AssistantChat :mode="auth.role === 'customer' ? 'shopping' : 'general'" @close="assistantOpen = false" @action="onAssistantAction" />
      </aside>
    </transition>

    <Modal :open="logoutOpen" :title="t('Log out?')" @close="logoutOpen = false">
      <div class="logout-confirm">
        <span class="logout-confirm-icon"><AppIcon name="logout" :size="22" /></span>
        <div>
          <strong>{{ t('Are you sure you want to log out?') }}</strong>
          <p>{{ t('You will need to sign in again to access your workspace.') }}</p>
        </div>
      </div>
      <template #footer>
        <button class="btn btn-secondary" type="button" @click="logoutOpen = false">{{ t('Cancel') }}</button>
        <button class="btn btn-danger" type="button" @click="confirmLogout">{{ t('Log out') }}</button>
      </template>
    </Modal>
  </div>
</template>

<style scoped>
.nav-count {
  display: inline-grid; place-items: center; min-width: 18px; height: 18px;
  margin-left: 8px; padding: 0 5px; border-radius: 999px;
  background: var(--primary); color: #fff; font-size: 10px; font-weight: 800;
}
.logout-confirm { display: flex; align-items: flex-start; gap: 14px; }
.logout-confirm-icon {
  width: 42px; height: 42px; flex-shrink: 0; display: grid; place-items: center;
  border-radius: 10px; background: var(--danger-50); color: var(--danger);
}
.logout-confirm strong { font-size: 14px; }
.logout-confirm p { margin: 5px 0 0; color: var(--text-2); font-size: 13px; line-height: 1.5; }
.assistant-fab {
  position: fixed; right: 22px; bottom: 22px; z-index: 60;
  width: 50px; height: 50px; border-radius: 50%; border: 0; cursor: pointer;
  background: var(--primary); color: #fff; display: grid; place-items: center;
  box-shadow: 0 6px 18px rgba(0,0,0,.2);
}
.assistant-fab:hover { filter: brightness(1.08); }
.assistant-overlay { position: fixed; inset: 0; z-index: 61; background: rgba(0,0,0,.28); }
.assistant-drawer {
  position: fixed; top: 0; right: 0; bottom: 0; z-index: 62;
  width: min(420px, 94vw); box-shadow: -8px 0 30px rgba(0,0,0,.18);
  border-left: 1px solid var(--border);
}
.assistant-slide-enter-active, .assistant-slide-leave-active { transition: transform .22s ease; }
.assistant-slide-enter-from, .assistant-slide-leave-to { transform: translateX(100%); }
</style>
