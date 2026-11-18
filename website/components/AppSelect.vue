<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import AppIcon from './AppIcon.vue'

const props = defineProps({
  modelValue: { type: [String, Number, Array, Boolean, Object], default: null },
  options: { type: Array, default: () => [] },
  placeholder: { type: String, default: 'Select…' },
  disabled: { type: Boolean, default: false },
  clearable: { type: Boolean, default: false },
  multiple: { type: Boolean, default: false },
  // null = auto: enable search when there are more than 8 options
  searchable: { type: Boolean, default: null },
  searchPlaceholder: { type: String, default: 'Search…' },
  emptyText: { type: String, default: 'No options found' },
  valueKey: { type: String, default: 'value' },
  labelKey: { type: String, default: 'label' },
  error: { type: Boolean, default: false },
  size: { type: String, default: 'md' },
})
const emit = defineEmits(['update:modelValue', 'change'])

const root = ref(null)
const searchInput = ref(null)
const open = ref(false)
const query = ref('')
const activeIndex = ref(-1)

function normalize(option) {
  if (option === null || option === undefined) return { value: option, label: '', disabled: false }
  if (typeof option !== 'object') return { value: option, label: String(option), disabled: false }
  const value = option[props.valueKey] ?? option.id ?? option.value
  const label = option[props.labelKey] ?? option.name ?? option.title ?? option.label ?? String(value)
  return { value, label, disabled: !!option.disabled, raw: option }
}

const items = computed(() => props.options.map(normalize))
const withSearch = computed(() =>
  props.searchable === null ? items.value.length > 8 : props.searchable,
)
const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return items.value
  return items.value.filter((i) => String(i.label).toLowerCase().includes(q))
})

const selectedItems = computed(() => {
  if (props.multiple) {
    const values = Array.isArray(props.modelValue) ? props.modelValue : []
    return values.map((v) => items.value.find((i) => i.value === v)).filter(Boolean)
  }
  if (props.modelValue === null || props.modelValue === undefined || props.modelValue === '') return []
  const found = items.value.find((i) => i.value === props.modelValue)
  return found ? [found] : []
})

const displayLabel = computed(() => {
  if (!selectedItems.value.length) return props.placeholder
  if (props.multiple) {
    if (selectedItems.value.length === 1) return selectedItems.value[0].label
    return `${selectedItems.value.length} selected`
  }
  return selectedItems.value[0].label
})

function toggleOpen() {
  if (props.disabled) return
  open.value = !open.value
  if (open.value) {
    query.value = ''
    activeIndex.value = filtered.value.findIndex((i) => !i.disabled)
    if (withSearch.value) nextTick(() => searchInput.value?.focus())
  }
}

function close() {
  open.value = false
  activeIndex.value = -1
}

function isSelected(item) {
  if (props.multiple) {
    return (Array.isArray(props.modelValue) ? props.modelValue : []).includes(item.value)
  }
  return props.modelValue === item.value
}

function choose(item) {
  if (item.disabled) return
  if (props.multiple) {
    const values = Array.isArray(props.modelValue) ? [...props.modelValue] : []
    const idx = values.indexOf(item.value)
    if (idx >= 0) values.splice(idx, 1)
    else values.push(item.value)
    emit('update:modelValue', values)
    emit('change', values)
  } else {
    emit('update:modelValue', item.value)
    emit('change', item.value)
    close()
  }
}

function clear(event) {
  event.stopPropagation()
  emit('update:modelValue', props.multiple ? [] : null)
  emit('change', props.multiple ? [] : null)
  close()
}

function removeChip(value, event) {
  event.stopPropagation()
  const values = (Array.isArray(props.modelValue) ? [...props.modelValue] : []).filter((v) => v !== value)
  emit('update:modelValue', values)
  emit('change', values)
}

function onKeydown(e) {
  if (props.disabled) return
  if (!open.value && ['ArrowDown', 'ArrowUp', 'Enter', ' '].includes(e.key)) {
    e.preventDefault()
    toggleOpen()
    return
  }
  if (!open.value) return
  if (e.key === 'Escape') { e.preventDefault(); close() }
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    let next = activeIndex.value + 1
    while (next < filtered.value.length && filtered.value[next].disabled) next++
    if (next < filtered.value.length) activeIndex.value = next
  }
  if (e.key === 'ArrowUp') {
    e.preventDefault()
    let prev = activeIndex.value - 1
    while (prev >= 0 && filtered.value[prev].disabled) prev--
    if (prev >= 0) activeIndex.value = prev
  }
  if (e.key === 'Enter') {
    e.preventDefault()
    const item = filtered.value[activeIndex.value]
    if (item) choose(item)
  }
}

function onDocumentClick(e) {
  if (root.value && !root.value.contains(e.target)) close()
}
onMounted(() => document.addEventListener('click', onDocumentClick))
onBeforeUnmount(() => document.removeEventListener('click', onDocumentClick))
watch(() => props.options.length, () => { activeIndex.value = -1 })
</script>

<template>
  <div ref="root" class="ui-select" :class="[`size-${size}`, { open, disabled, error }]">
    <button
      type="button"
      class="control"
      :disabled="disabled"
      :aria-expanded="open"
      @click="toggleOpen"
      @keydown="onKeydown"
    >
      <span v-if="selectedItems.length && multiple" class="chips">
        <span v-for="item in selectedItems" :key="item.value" class="chip">
          {{ item.label }}
          <span v-if="!disabled" class="chip-x" @click="removeChip(item.value, $event)">×</span>
        </span>
      </span>
      <span v-else class="value" :class="{ placeholder: !selectedItems.length }">{{ displayLabel }}</span>
      <svg class="caret" :class="{ flipped: open }" xmlns="http://www.w3.org/2000/svg" width="14" height="14" fill="currentColor" viewBox="0 0 256 256" aria-hidden="true">
        <path d="M213.66,101.66l-80,80a8,8,0,0,1-11.32,0l-80-80A8,8,0,0,1,53.66,90.34L128,164.69l74.34-74.35a8,8,0,0,1,11.32,11.32Z"></path>
      </svg>
    </button>

    <button
      v-if="clearable && selectedItems.length && !disabled"
      type="button"
      class="clear-btn"
      aria-label="Clear selection"
      @click="clear"
    >×</button>

    <transition name="pop">
      <div v-if="open" class="dropdown">
        <div v-if="withSearch" class="search-wrap">
          <span class="search-icon">⌕</span>
          <input
            ref="searchInput"
            v-model="query"
            type="text"
            class="search-input"
            :placeholder="searchPlaceholder"
            @keydown="onKeydown"
          />
        </div>
        <div class="option-list">
          <button
            v-for="(item, index) in filtered"
            :key="index"
            type="button"
            class="option"
            :class="{ selected: isSelected(item), active: index === activeIndex, disabled: item.disabled }"
            :disabled="item.disabled"
            @mouseenter="activeIndex = index"
            @click="choose(item)"
          >
            <span v-if="multiple" class="check" :class="{ on: isSelected(item) }"><AppIcon v-if="isSelected(item)" name="check" :size="11" /></span>
            <span class="option-label">{{ item.label }}</span>
            <AppIcon v-if="!multiple && isSelected(item)" name="check" :size="14" class="tick" />
          </button>
          <div v-if="!filtered.length" class="empty">{{ emptyText }}</div>
        </div>
        <div v-if="multiple && selectedItems.length" class="dropdown-foot">{{ selectedItems.length }} selected</div>
      </div>
    </transition>
  </div>
</template>

<style scoped>
.ui-select { position: relative; width: 100%; }
.control {
  width: 100%; display: flex; align-items: center; gap: 8px;
  min-height: 38px; padding: 7px 10px; cursor: pointer;
  background: var(--surface); border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm); color: var(--text);
  font: inherit; font-size: 13.5px; text-align: left; transition: border-color .15s, box-shadow .15s;
}
.ui-select.size-sm .control { min-height: 32px; font-size: 12.5px; padding: 4px 9px; }
.control:hover { border-color: var(--text-3); }
.ui-select.open .control, .control:focus-visible { border-color: var(--primary); box-shadow: 0 0 0 3px var(--primary-50); outline: 0; }
.ui-select.error .control { border-color: var(--danger); }
.ui-select.disabled .control { background: var(--surface-2); opacity: .65; cursor: not-allowed; }
.value { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.value.placeholder { color: var(--text-3); }
.caret { color: var(--text-3); flex-shrink: 0; transition: transform .15s; }
.caret.flipped { transform: rotate(180deg); }
.clear-btn {
  position: absolute; right: 28px; top: 50%; transform: translateY(-50%);
  border: 0; background: none; color: var(--text-3); cursor: pointer; font-size: 16px; padding: 0 2px;
}
.clear-btn:hover { color: var(--text); }
.chips { display: flex; flex-wrap: wrap; gap: 4px; flex: 1; min-width: 0; }
.chip {
  display: inline-flex; align-items: center; gap: 4px;
  background: var(--primary-50); color: var(--primary-700);
  border-radius: 6px; padding: 2px 6px; font-size: 12px;
}
.chip-x { cursor: pointer; font-size: 13px; line-height: 1; }
.dropdown {
  position: absolute; z-index: 60; top: calc(100% + 5px); left: 0; right: 0;
  background: var(--surface); border: 1px solid var(--border);
  border-radius: var(--radius-sm); box-shadow: var(--shadow);
  overflow: hidden;
}
.search-wrap {
  display: flex; align-items: center; gap: 7px; padding: 8px;
  border-bottom: 1px solid var(--border);
}
.search-icon { color: var(--text-3); font-size: 15px; }
.search-input {
  flex: 1; min-width: 0; border: 0; outline: 0; background: transparent;
  font: inherit; font-size: 13px; color: var(--text);
}
.search-input::placeholder { color: var(--text-3); }
.option-list { max-height: 240px; overflow-y: auto; padding: 4px; }
.option {
  width: 100%; display: flex; align-items: center; gap: 8px;
  border: 0; background: none; border-radius: 6px; padding: 7px 9px;
  font: inherit; font-size: 13px; color: var(--text); cursor: pointer; text-align: left;
}
.option.active, .option:hover { background: var(--surface-2); }
.option.selected { color: var(--primary); font-weight: 650; }
.option.disabled { opacity: .45; cursor: not-allowed; }
.option-label { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.check {
  width: 15px; height: 15px; border: 1px solid var(--border-strong); border-radius: 4px;
  display: inline-flex; align-items: center; justify-content: center; font-size: 10px; flex-shrink: 0;
}
.check.on { background: var(--primary); border-color: var(--primary); color: #fff; }
.empty { padding: 18px; text-align: center; color: var(--text-3); font-size: 12.5px; }
.dropdown-foot { padding: 7px 10px; border-top: 1px solid var(--border); color: var(--text-3); font-size: 11.5px; }
.pop-enter-active, .pop-leave-active { transition: opacity .12s, transform .12s; }
.pop-enter-from, .pop-leave-to { opacity: 0; transform: translateY(-4px); }
</style>
