<template>
  <section class="resource-query-panel" :data-resource="config.resource" data-testid="resource-query-panel">
    <advanced-query-filter-builder-frame v-if="frame" :frame="frame" v-bind="{ messagePrefix }">
      <template #saved-query-views>
        <saved-query-view-control v-if="config.savedView && savedViewController" :controller="savedViewController" />
        <slot name="saved-query-views" />
      </template>
      <template #toolbar-after-search><slot name="toolbar-after-search" /></template>
    </advanced-query-filter-builder-frame>
    <management-toolbar v-else class="resource-query-panel__toolbar" data-testid="resource-query-toolbar">
      <template #filters>
        <div class="resource-query-panel__content">
          <div class="resource-query-panel__main">
            <t-input
              v-if="searchEnabled"
              :model-value="draft.keyword"
              class="management-query-search"
              clearable
              :placeholder="config.placeholder ?? t('app.queryBar.searchPlaceholder')"
              @enter="apply"
              @update:model-value="updateKeyword"
            />
            <slot name="toolbar-after-search" />
            <t-button
              v-if="filters.length && !compact"
              data-testid="resource-query-builder-trigger"
              :aria-expanded="filtersVisible"
              :theme="filtersVisible ? 'primary' : 'default'"
              :variant="filtersVisible ? 'base' : 'outline'"
              @click="filtersVisible = !filtersVisible"
            >
              {{ t('app.queryBar.moreFilters') }}
            </t-button>
            <div class="resource-query-panel__commands">
              <t-button data-testid="resource-query-search" theme="primary" :loading="loading" @click="apply">
                {{ t('app.queryBar.search') }}
              </t-button>
              <t-button data-testid="resource-query-reset" variant="outline" @click="reset">
                {{ t('app.queryBar.reset') }}
              </t-button>
            </div>
          </div>

          <div v-if="filtersVisible && !compact && filters.length" class="resource-query-panel__expanded-filters">
            <filter-fields v-model="draft.filters" :filters="filters" />
          </div>

          <div v-if="simpleFiltersVisible && $slots['simple-filters']" class="resource-query-panel__simple-filters">
            <slot name="simple-filters" />
          </div>
          <div v-if="quickFilters.length || $slots.quick" class="resource-query-panel__quick-filters">
            <slot name="quick" :apply="applyQuickFilter" />
            <t-button
              v-for="quick in quickFilters"
              :key="quick.key"
              size="small"
              variant="text"
              @click="applyQuickFilter(quick.patch)"
            >
              {{ quick.label }}
            </t-button>
          </div>
          <div v-if="activeTags.length" class="resource-query-panel__tags" data-testid="resource-query-tags">
            <span class="resource-query-panel__tags-label">{{ t('app.queryBar.applied') }}</span>
            <t-tag
              v-for="tag in activeTags"
              :key="tag.key"
              closable
              size="small"
              theme="primary"
              variant="light-outline"
              @close="removeFilter(tag.key)"
            >
              {{ tag.label }}
            </t-tag>
            <t-button size="small" variant="text" @click="reset">{{ t('app.queryBar.clearAll') }}</t-button>
          </div>
        </div>
        <t-drawer
          v-if="compact && filters.length"
          v-model:visible="filtersVisible"
          :header="t('app.queryBar.moreFilters')"
          placement="bottom"
          size="auto"
        >
          <filter-fields v-model="draft.filters" :filters="filters" />
          <template #footer>
            <t-button variant="outline" @click="clearDraft">{{ t('app.queryBar.clear') }}</t-button>
            <t-button theme="primary" @click="apply">{{ t('app.queryBar.apply') }}</t-button>
          </template>
        </t-drawer>
      </template>
      <template v-if="(config.savedView && savedViewController) || $slots['saved-query-views']" #actions>
        <saved-query-view-control v-if="config.savedView && savedViewController" :controller="savedViewController" />
        <slot name="saved-query-views" />
      </template>
    </management-toolbar>
  </section>
</template>
<script setup lang="ts">
import { computed, defineComponent, h, type PropType, ref, resolveComponent, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import { ManagementToolbar } from '@/shared/components/management';
import { useViewportResponsiveVariant } from '@/shared/composables/useViewportResponsiveVariant';

import AdvancedQueryFilterBuilderFrame, {
  type AdvancedQueryFilterBuilderFrameState,
} from '../AdvancedQueryFilterBuilderFrame.vue';
import type { SavedQueryViewController, SavedQueryViewId } from '../saved-query-views';
import SavedQueryViewControl from '../SavedQueryViewControl.vue';
import type {
  ResourceQueryConfig,
  ResourceQueryFilterDefinition,
  ResourceQueryFilterValue,
  ResourceQueryState,
} from './types';

const FilterFields = defineComponent({
  name: 'ResourceQueryFilterFields',
  props: {
    filters: { type: Array as PropType<ResourceQueryFilterDefinition[]>, required: true },
    modelValue: { type: Object as PropType<Record<string, ResourceQueryFilterValue>>, required: true },
  },
  emits: ['update:modelValue'],
  setup(fieldProps, { emit: emitField }) {
    const inputComponent = resolveComponent('t-input');
    const selectComponent = resolveComponent('t-select');
    const dateRangePickerComponent = resolveComponent('t-date-range-picker');
    const inputNumberComponent = resolveComponent('t-input-number');
    const switchComponent = resolveComponent('t-switch');
    const setValue = (key: string, value: ResourceQueryFilterValue) =>
      emitField('update:modelValue', { ...fieldProps.modelValue, [key]: value });
    return () =>
      h(
        'div',
        { class: 'resource-query-panel__fields' },
        fieldProps.filters.map((field) => {
          const value = fieldProps.modelValue[field.key];
          const common = { disabled: field.disabled, placeholder: field.placeholder, value };
          let control;
          if (field.type === 'select' || field.type === 'multi-select') {
            control = h(selectComponent, {
              ...common,
              modelValue: value,
              multiple: field.type === 'multi-select',
              options: field.options ?? [],
              clearable: true,
              'onUpdate:modelValue': (next: ResourceQueryFilterValue) => setValue(field.key, next),
            });
          } else if (field.type === 'date-range') {
            control = h(dateRangePickerComponent, {
              ...common,
              modelValue: value,
              clearable: true,
              'onUpdate:modelValue': (next: ResourceQueryFilterValue) => setValue(field.key, next),
            });
          } else if (field.type === 'number-range') {
            const range = Array.isArray(value) ? value : [];
            control = h('div', { class: 'resource-query-panel__number-range' }, [
              h(inputNumberComponent, {
                ...common,
                modelValue: range[0],
                'onUpdate:modelValue': (next: number | undefined) => setValue(field.key, [next ?? '', range[1] ?? '']),
              }),
              h('span', { class: 'resource-query-panel__range-separator' }, '-'),
              h(inputNumberComponent, {
                ...common,
                modelValue: range[1],
                'onUpdate:modelValue': (next: number | undefined) => setValue(field.key, [range[0] ?? '', next ?? '']),
              }),
            ]);
          } else if (field.type === 'boolean') {
            control = h(switchComponent, {
              disabled: field.disabled,
              modelValue: Boolean(value),
              'onUpdate:modelValue': (next: boolean) => setValue(field.key, next),
            });
          } else {
            control = h(inputComponent, {
              ...common,
              modelValue: typeof value === 'string' ? value : '',
              clearable: true,
              'onUpdate:modelValue': (next: string) => setValue(field.key, next),
            });
          }
          return h('label', { class: 'resource-query-panel__field', key: field.key }, [
            h('span', field.label),
            control,
          ]);
        }),
      );
  },
});

// 统一入口承载资源查询状态；访问日志的成熟构建器仅作为内部可选能力保留既有契约。
const props = withDefaults(
  defineProps<{
    config: ResourceQueryConfig;
    frame?: AdvancedQueryFilterBuilderFrameState;
    loading?: boolean;
    messagePrefix?: string;
    modelValue?: ResourceQueryState;
    simpleFiltersVisible?: boolean;
    savedViewController?: SavedQueryViewController<unknown, SavedQueryViewId>;
  }>(),
  {
    frame: undefined,
    loading: false,
    messagePrefix: '',
    modelValue: undefined,
    simpleFiltersVisible: false,
    savedViewController: undefined,
  },
);

const emit = defineEmits<{
  (e: 'reset', value: ResourceQueryState): void;
  (e: 'search', value: ResourceQueryState): void;
  (e: 'update:modelValue', value: ResourceQueryState): void;
}>();

const { t } = useI18n();
const viewport = useViewportResponsiveVariant();
const emptyQueryState: ResourceQueryState = { keyword: '', filters: {}, page: 1, pageSize: 20 };
const draft = ref<ResourceQueryState>(cloneState(props.modelValue ?? emptyQueryState));
const filtersVisible = ref(false);
const filters = computed(() => (props.config.filterBuilder?.enabled === false ? [] : (props.config.filters ?? [])));
const quickFilters = computed(() => props.config.quickFilters ?? []);
const searchEnabled = computed(() => props.config.search !== false);
const compact = computed(() => viewport.value.density === 'compact');
const activeTags = computed(() =>
  filters.value.flatMap((field) => {
    const value = (props.modelValue ?? emptyQueryState).filters[field.key];
    if (isEmpty(value)) return [];
    return [{ key: field.key, label: `${field.label}=${formatFilterValue(field, value)}` }];
  }),
);

watch(
  () => props.modelValue,
  (value) => {
    draft.value = cloneState(value ?? emptyQueryState);
  },
  { deep: true },
);

function cloneState(value: ResourceQueryState): ResourceQueryState {
  return { ...value, filters: { ...value.filters } };
}

function isEmpty(value: ResourceQueryFilterValue) {
  return value === undefined || value === '' || (Array.isArray(value) && value.length === 0);
}

function formatFilterValue(field: ResourceQueryFilterDefinition, value: ResourceQueryFilterValue) {
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
  if (compact.value) {
    filtersVisible.value = false;
  }
}

function reset() {
  const next = { keyword: '', filters: {}, page: 1, pageSize: (props.modelValue ?? emptyQueryState).pageSize };
  draft.value = cloneState(next);
  emit('update:modelValue', next);
  emit('reset', next);
}

function clearDraft() {
  draft.value.filters = {};
}

function removeFilter(key: string) {
  const next = cloneState(props.modelValue ?? emptyQueryState);
  delete next.filters[key];
  next.page = 1;
  draft.value = cloneState(next);
  emit('update:modelValue', next);
  emit('search', next);
}

function applyQuickFilter(patch: Record<string, ResourceQueryFilterValue>) {
  draft.value.filters = { ...draft.value.filters, ...patch };
  apply();
}
</script>
<style scoped lang="less">
.resource-query-panel {
  min-width: 0;
}

.resource-query-panel :deep(.resource-query-panel__toolbar) {
  align-items: flex-start;
}

.resource-query-panel__content {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
  min-width: 0;
}

.resource-query-panel__main,
.resource-query-panel__simple-filters,
.resource-query-panel__quick-filters,
.resource-query-panel__tags,
.resource-query-panel__commands {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-8);
  min-height: 32px;
  min-width: 0;
}

.resource-query-panel__simple-filters,
.resource-query-panel__quick-filters {
  flex-wrap: wrap;
}

.resource-query-panel__main {
  width: 100%;
}

.resource-query-panel__main :deep(.management-query-search) {
  flex: 0 1 clamp(280px, 24vw, 360px);
  min-width: 240px;
  width: clamp(280px, 24vw, 360px);
}

.resource-query-panel__commands {
  flex: 0 0 auto;
  margin-left: auto;
}

.resource-query-panel__tags {
  flex-wrap: nowrap;
  overflow-x: auto;
  padding-bottom: var(--graft-density-gap-2);
}

.resource-query-panel__expanded-filters {
  width: 100%;
}

.resource-query-panel__fields {
  align-items: flex-end;
  display: flex;
  flex-wrap: nowrap;
  gap: var(--graft-density-gap-14);
  width: 100%;
}

.resource-query-panel :deep(.resource-query-panel__fields) {
  align-items: flex-end;
  display: flex;
  flex-wrap: nowrap;
  gap: var(--graft-density-gap-14);
  width: 100%;
}

.resource-query-panel__field {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: flex;
  flex: 1 1 0;
  font: var(--td-font-body-small);
  gap: var(--graft-density-gap-8);
  min-width: 0;
}

.resource-query-panel__field > span {
  flex: 0 0 auto;
  white-space: nowrap;
}

.resource-query-panel__field > :deep(.t-input),
.resource-query-panel__field > :deep(.t-select),
.resource-query-panel__field > :deep(.t-date-range-picker),
.resource-query-panel__field > :deep(.t-input-number),
.resource-query-panel__field > :deep(.resource-query-panel__number-range) {
  flex: 1 1 auto;
  min-width: 0;
}

.resource-query-panel :deep(.resource-query-panel__field) {
  align-items: center;
  display: flex;
  flex: 1 1 0;
  gap: var(--graft-density-gap-8);
  min-width: 0;
}

.resource-query-panel :deep(.resource-query-panel__field > span) {
  flex: 0 0 auto;
  white-space: nowrap;
}

.resource-query-panel :deep(.resource-query-panel__field > .t-input),
.resource-query-panel :deep(.resource-query-panel__field > .t-select),
.resource-query-panel :deep(.resource-query-panel__field > .t-date-range-picker),
.resource-query-panel :deep(.resource-query-panel__field > .t-input-number),
.resource-query-panel :deep(.resource-query-panel__field > .resource-query-panel__number-range) {
  flex: 1 1 auto;
  min-width: 0;
}

.resource-query-panel__number-range {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-8);
}

.resource-query-panel__number-range :deep(.t-input-number) {
  min-width: 0;
}

.resource-query-panel__range-separator,
.resource-query-panel__tags-label {
  color: var(--td-text-color-secondary);
}

.resource-query-panel__tags-label {
  font: var(--td-font-body-small);
  white-space: nowrap;
}

@media (width <= 768px) {
  .resource-query-panel__main {
    align-items: stretch;
    flex-wrap: wrap;
  }

  .resource-query-panel__main :deep(.management-query-search) {
    flex-basis: 100%;
    max-width: none;
    width: 100%;
  }

  .resource-query-panel__commands {
    margin-left: 0;
  }

  .resource-query-panel__fields {
    align-items: stretch;
    flex-wrap: wrap;
  }

  .resource-query-panel__field {
    flex-basis: 100%;
  }

  .resource-query-panel :deep(.resource-query-panel__fields) {
    align-items: stretch;
    flex-wrap: wrap;
  }

  .resource-query-panel :deep(.resource-query-panel__field) {
    flex-basis: 100%;
  }
}
</style>
