<template>
  <section class="graft-query-bar" data-testid="graft-query-bar">
    <div class="graft-query-bar__main">
      <t-input
        :model-value="draft.keyword"
        class="graft-query-bar__search"
        clearable
        :placeholder="config.placeholder ?? t('app.queryBar.searchPlaceholder')"
        @enter="apply"
        @update:model-value="updateKeyword"
      />
      <t-popup
        v-if="filters.length && !compact"
        v-model:visible="filtersVisible"
        attach="body"
        placement="bottom-left"
        trigger="click"
      >
        <template #content
          ><div class="graft-query-bar__panel"><filter-fields v-model="draft.filters" :filters="filters" /></div
        ></template>
        <t-button data-testid="graft-query-bar-more" variant="outline">{{ t('app.queryBar.moreFilters') }}</t-button>
      </t-popup>
      <t-button
        v-else-if="filters.length"
        data-testid="graft-query-bar-more"
        variant="outline"
        @click="filtersVisible = true"
      >
        {{ t('app.queryBar.moreFilters') }}
      </t-button>
      <t-button theme="primary" :loading="loading" @click="apply">{{ t('app.queryBar.search') }}</t-button>
      <t-button variant="outline" @click="reset">{{ t('app.queryBar.reset') }}</t-button>
    </div>

    <div v-if="quickFilters.length || $slots.quick" class="graft-query-bar__quick">
      <slot name="quick" :apply="applyQuickFilter" /><t-button
        v-for="quick in quickFilters"
        :key="quick.key"
        size="small"
        variant="text"
        @click="applyQuickFilter(quick.patch)"
        >{{ quick.label }}</t-button
      >
    </div>
    <div v-if="activeTags.length" class="graft-query-bar__tags" data-testid="graft-query-bar-tags">
      <span class="graft-query-bar__tags-label">{{ t('app.queryBar.applied') }}</span>
      <t-tag
        v-for="tag in activeTags"
        :key="tag.key"
        closable
        size="small"
        theme="primary"
        variant="light-outline"
        @close="removeFilter(tag.key)"
        >{{ tag.label }}</t-tag
      >
      <t-button size="small" variant="text" @click="reset">{{ t('app.queryBar.clearAll') }}</t-button>
    </div>

    <t-drawer
      v-if="compact && filters.length"
      v-model:visible="filtersVisible"
      :header="t('app.queryBar.moreFilters')"
      placement="bottom"
      size="auto"
    >
      <filter-fields v-model="draft.filters" :filters="filters" />
      <template #footer
        ><t-button variant="outline" @click="clearDraft">{{ t('app.queryBar.clear') }}</t-button
        ><t-button theme="primary" @click="applyFromDrawer">{{ t('app.queryBar.apply') }}</t-button></template
      >
    </t-drawer>
  </section>
</template>
<script setup lang="ts">
import { computed, defineComponent, h, type PropType, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import { useViewportResponsiveVariant } from '@/shared/composables/useViewportResponsiveVariant';

import type {
  GraftQueryBarConfig,
  GraftQueryFilterDefinition,
  GraftQueryFilterValue,
  GraftQueryState,
} from './graft-query-bar';

const FilterFields = defineComponent({
  name: 'GraftQueryFilterFields',
  props: {
    filters: { type: Array as PropType<GraftQueryFilterDefinition[]>, required: true },
    modelValue: { type: Object as PropType<Record<string, GraftQueryFilterValue>>, required: true },
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    const setValue = (key: string, value: GraftQueryFilterValue) =>
      emit('update:modelValue', { ...props.modelValue, [key]: value });
    return () =>
      h(
        'div',
        { class: 'graft-query-bar__fields' },
        props.filters.map((field) => {
          const value = props.modelValue[field.key];
          const common = {
            disabled: field.disabled,
            placeholder: field.placeholder,
            value,
            'onUpdate:value': (next: GraftQueryFilterValue) => setValue(field.key, next),
          };
          let control;
          if (field.type === 'select' || field.type === 'multi-select')
            control = h('t-select', {
              ...common,
              modelValue: value,
              multiple: field.type === 'multi-select',
              options: field.options ?? [],
              clearable: true,
              'onUpdate:modelValue': (next: GraftQueryFilterValue) => setValue(field.key, next),
            });
          else if (field.type === 'date-range')
            control = h('t-date-range-picker', {
              ...common,
              modelValue: value,
              clearable: true,
              'onUpdate:modelValue': (next: GraftQueryFilterValue) => setValue(field.key, next),
            });
          else if (field.type === 'number-range') {
            const range = Array.isArray(value) ? value : [];
            control = h('div', { class: 'graft-query-bar__number-range' }, [
              h('t-input-number', {
                ...common,
                modelValue: range[0],
                'onUpdate:modelValue': (next: number | undefined) => setValue(field.key, [next ?? '', range[1] ?? '']),
              }),
              h('span', { class: 'graft-query-bar__range-separator' }, '-'),
              h('t-input-number', {
                ...common,
                modelValue: range[1],
                'onUpdate:modelValue': (next: number | undefined) => setValue(field.key, [range[0] ?? '', next ?? '']),
              }),
            ]);
          } else if (field.type === 'boolean')
            control = h('t-switch', {
              ...common,
              value: Boolean(value),
              'onUpdate:value': (next: boolean) => setValue(field.key, next),
            });
          else
            control = h('t-input', {
              ...common,
              modelValue: typeof value === 'string' ? value : '',
              clearable: true,
              'onUpdate:modelValue': (next: string) => setValue(field.key, next),
            });
          return h('label', { class: 'graft-query-bar__field', key: field.key }, [h('span', field.label), control]);
        }),
      );
  },
});

const props = withDefaults(
  defineProps<{ modelValue: GraftQueryState; config: GraftQueryBarConfig; loading?: boolean }>(),
  { loading: false },
);
const emit = defineEmits<{
  (e: 'update:modelValue', value: GraftQueryState): void;
  (e: 'search', value: GraftQueryState): void;
  (e: 'reset', value: GraftQueryState): void;
}>();
const { t } = useI18n();
const viewport = useViewportResponsiveVariant();
const filtersVisible = ref(false);
const draft = ref<GraftQueryState>(cloneState(props.modelValue));
const filters = computed(() => props.config.filters ?? []);
const quickFilters = computed(() => props.config.quickFilters ?? []);
const compact = computed(() => viewport.value.density === 'compact');
const activeTags = computed(() =>
  filters.value.flatMap((field) => {
    const value = props.modelValue.filters[field.key];
    if (isEmpty(value)) return [];
    return [{ key: field.key, label: `${field.label}=${formatFilterValue(field, value)}` }];
  }),
);

watch(
  () => props.modelValue,
  (value) => {
    draft.value = cloneState(value);
  },
  { deep: true },
);

function cloneState(value: GraftQueryState): GraftQueryState {
  return { ...value, filters: { ...value.filters } };
}
function isEmpty(value: GraftQueryFilterValue) {
  return value === undefined || value === '' || (Array.isArray(value) && value.length === 0);
}
function formatFilterValue(field: GraftQueryFilterDefinition, value: GraftQueryFilterValue) {
  if (field.type === 'boolean') return value ? t('app.queryBar.yes') : t('app.queryBar.no');
  const formatOption = (candidate: string | number) =>
    field.options?.find((option) => option.value === candidate)?.label ?? String(candidate);
  if (Array.isArray(value)) return value.map(formatOption).join(' ~ ');
  return typeof value === 'string' || typeof value === 'number' ? formatOption(value) : '';
}
function updateKeyword(value: string | number) {
  draft.value.keyword = typeof value === 'string' ? value : '';
}
function apply() {
  const next = { ...cloneState(draft.value), page: 1 };
  emit('update:modelValue', next);
  emit('search', next);
  filtersVisible.value = false;
}
function applyFromDrawer() {
  apply();
}
function reset() {
  const next = { keyword: '', filters: {}, page: 1, pageSize: props.modelValue.pageSize };
  draft.value = cloneState(next);
  emit('update:modelValue', next);
  emit('reset', next);
}
function clearDraft() {
  draft.value.filters = {};
}
function removeFilter(key: string) {
  const next = cloneState(props.modelValue);
  delete next.filters[key];
  next.page = 1;
  draft.value = cloneState(next);
  emit('update:modelValue', next);
  emit('search', next);
}
function applyQuickFilter(patch: Record<string, GraftQueryFilterValue>) {
  draft.value.filters = { ...draft.value.filters, ...patch };
  apply();
}
</script>
<style scoped lang="less">
.graft-query-bar {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
  min-width: 0;
}

.graft-query-bar__main,
.graft-query-bar__quick,
.graft-query-bar__tags {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-8);
  min-height: 48px;
  min-width: 0;
}

.graft-query-bar__search {
  flex: 1 1 320px;
  max-width: 560px;
  min-width: 200px;
}

.graft-query-bar__panel {
  max-height: min(560px, calc(100vh - 160px));
  overflow: auto;
  padding: var(--graft-density-gap-16);
  width: min(560px, calc(100vw - 32px));
}

.graft-query-bar__fields {
  display: grid;
  gap: var(--graft-density-gap-14);
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.graft-query-bar__field {
  color: var(--td-text-color-secondary);
  display: flex;
  flex-direction: column;
  font: var(--td-font-body-small);
  gap: var(--graft-density-gap-8);
}

.graft-query-bar__number-range {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-8);
}

.graft-query-bar__number-range :deep(.t-input-number) {
  min-width: 0;
}

.graft-query-bar__range-separator {
  color: var(--td-text-color-secondary);
}

.graft-query-bar__tags {
  flex-wrap: nowrap;
  overflow-x: auto;
  padding-bottom: var(--graft-density-gap-2);
}

.graft-query-bar__tags-label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  white-space: nowrap;
}

@media (width <= 768px) {
  .graft-query-bar__main {
    align-items: stretch;
    flex-wrap: wrap;
  }

  .graft-query-bar__search {
    flex-basis: 100%;
    max-width: none;
  }

  .graft-query-bar__fields {
    grid-template-columns: 1fr;
  }
}
</style>
