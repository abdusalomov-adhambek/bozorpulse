<script setup>
import { t } from '../i18n'
import { computed, useSlots } from 'vue'
import EmptyState from './EmptyState.vue'

const props = defineProps({
  columns: { type: Array, required: true }, // [{ key, label }]
  rows: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  emptyText: { type: String, default: 'Nothing here yet' },
  emptyDesc: { type: String, default: '' }
})

const slots = useSlots()
const hasActions = computed(() => Boolean(slots.actions))
const colspan = computed(() => props.columns.length + (hasActions.value ? 1 : 0))

function format(value) {
  if (value === null || value === undefined || value === '') return '—'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}
</script>

<template>
  <div class="table-wrap">
    <table class="data-table">
      <thead>
        <tr>
          <th v-for="col in columns" :key="col.key">{{ col.label }}</th>
          <th v-if="hasActions" class="right">{{ t("Actions") }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="loading">
          <td :colspan="colspan" class="table-state">
            <div class="loading-block"><span class="spinner"></span>{{ t('Loading data…') }}</div>
          </td>
        </tr>
        <tr v-else-if="!rows.length">
          <td :colspan="colspan" class="table-state">
            <EmptyState :title="emptyText" :description="emptyDesc" />
          </td>
        </tr>
        <template v-else>
          <tr v-for="(row, index) in rows" :key="row.id ?? index">
            <td v-for="col in columns" :key="col.key">
              <slot :name="`cell-${col.key}`" :row="row" :value="row[col.key]">
                {{ format(row[col.key]) }}
              </slot>
            </td>
            <td v-if="hasActions" class="actions-cell">
              <slot name="actions" :row="row" />
            </td>
          </tr>
        </template>
      </tbody>
    </table>
  </div>
</template>
