<script setup>
import { t } from '../i18n'
import { computed, nextTick, ref, watch } from 'vue'
import AppInput from './ui/AppInput.vue'

const props = defineProps({
  open: { type: Boolean, default: false },
  items: { type: Array, default: () => [] },
})
const emit = defineEmits(['update:open', 'select'])

const query = ref('')
const active = ref(0)
const input = ref(null)

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return props.items
  return props.items.filter((item) =>
    `${item.label} ${item.group || ''}`.toLowerCase().includes(q),
  )
})

function close() {
  emit('update:open', false)
}

function choose(item) {
  if (!item) return
  emit('select', item)
  close()
}

function onKeydown(e) {
  if (e.key === 'Escape') { e.preventDefault(); close() }
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    active.value = Math.min(active.value + 1, filtered.value.length - 1)
  }
  if (e.key === 'ArrowUp') {
    e.preventDefault()
    active.value = Math.max(active.value - 1, 0)
  }
  if (e.key === 'Enter') {
    e.preventDefault()
    choose(filtered.value[active.value])
  }
}

watch(() => props.open, async (isOpen) => {
  if (!isOpen) return
  query.value = ''
  active.value = 0
  await nextTick()
  input.value?.focus()
})
watch(query, () => { active.value = 0 })
</script>

<template>
  <transition name="cmd">
    <div v-if="open" class="cmd-overlay" @click.self="close">
      <div class="cmd-panel" role="dialog" aria-modal="true" :aria-label="t('Command palette')">
        <div class="cmd-search">
          <AppInput
            ref="input"
            v-model="query"
            :placeholder="t('Search pages and actions…')"
            clearable
            @keydown="onKeydown"
          >
            <template #prefix><span class="cmd-search-icon" aria-hidden="true">⌕</span></template>
          </AppInput>
          <kbd>{{ t('ESC') }}</kbd>
        </div>
        <div class="cmd-list">
          <template v-for="(item, i) in filtered" :key="`${item.group}-${item.label}-${i}`">
            <div v-if="i === 0 || filtered[i - 1]?.group !== item.group" class="cmd-group">
              {{ item.group || 'Pages' }}
            </div>
            <button
              type="button"
              class="cmd-item"
              :class="{ active: i === active }"
              @mouseenter="active = i"
              @click="choose(item)"
            >
              <span class="cmd-item-icon" aria-hidden="true">{{ item.icon || '→' }}</span>
              <span class="cmd-item-label">{{ item.label }}</span>
              <span v-if="item.hint" class="cmd-item-hint">{{ item.hint }}</span>
            </button>
          </template>
          <div v-if="!filtered.length" class="cmd-empty">No matches for “{{ query }}”</div>
        </div>
        <div class="cmd-foot">
          <span><kbd>↑</kbd><kbd>↓</kbd> {{ t('navigate') }}</span>
          <span><kbd>↵</kbd> {{ t('open') }}</span>
          <span><kbd>esc</kbd> {{ t('close') }}</span>
        </div>
      </div>
    </div>
  </transition>
</template>

<style scoped>
.cmd-overlay {
  position: fixed; inset: 0; z-index: 200;
  background: rgba(28, 28, 27, 0.42);
  backdrop-filter: blur(4px);
  display: flex; align-items: flex-start; justify-content: center;
  padding: 12vh 20px 20px;
}
.cmd-panel {
  width: 100%; max-width: 560px;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 14px;
  box-shadow: 0 24px 60px rgba(28, 28, 27, 0.22);
  overflow: hidden;
}
.cmd-search {
  display: flex; align-items: center; gap: 10px;
  padding: 13px 16px; border-bottom: 1px solid var(--border);
}
.cmd-search-icon { color: var(--text-3); font-size: 19px; }
.cmd-search input { flex: 1; border: 0; outline: 0; font: inherit; font-size: 14px; color: var(--text); background: transparent; }
.cmd-search input::placeholder { color: var(--text-3); }
kbd {
  font: inherit; font-size: 10px; font-weight: 700;
  border: 1px solid var(--border-strong); border-radius: 5px;
  padding: 2px 5px; color: var(--text-3); background: var(--surface-2);
}
.cmd-list { max-height: 390px; overflow-y: auto; padding: 8px; }
.cmd-group {
  padding: 10px 10px 5px; font-size: 10.5px; font-weight: 750;
  letter-spacing: .08em; text-transform: uppercase; color: var(--text-3);
}
.cmd-item {
  width: 100%; display: flex; align-items: center; gap: 11px;
  border: 0; background: none; border-radius: 8px; padding: 9px 10px;
  font: inherit; font-size: 13px; color: var(--text); cursor: pointer; text-align: left;
}
.cmd-item.active { background: var(--primary-50); color: var(--primary-700); }
.cmd-item-icon {
  width: 26px; height: 26px; display: grid; place-items: center;
  border: 1px solid var(--border); border-radius: 7px;
  color: var(--text-2); background: var(--surface); font-size: 13px; flex-shrink: 0;
}
.cmd-item.active .cmd-item-icon { border-color: var(--primary); color: var(--primary); }
.cmd-item-label { flex: 1; font-weight: 550; }
.cmd-item-hint { font-size: 11px; color: var(--text-3); }
.cmd-empty { padding: 28px; text-align: center; color: var(--text-3); font-size: 13px; }
.cmd-foot {
  display: flex; gap: 16px; padding: 10px 16px;
  border-top: 1px solid var(--border); color: var(--text-3); font-size: 11px;
}
.cmd-foot span { display: inline-flex; align-items: center; gap: 4px; }
.cmd-enter-active, .cmd-leave-active { transition: opacity .14s; }
.cmd-enter-from, .cmd-leave-to { opacity: 0; }
.cmd-enter-active .cmd-panel, .cmd-leave-active .cmd-panel { transition: transform .14s; }
.cmd-enter-from .cmd-panel, .cmd-leave-to .cmd-panel { transform: translateY(-8px); }
</style>
