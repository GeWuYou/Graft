<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <advanced-query-filter-builder-frame :frame="builderFrame" message-prefix="appLog" />
</template>
<script setup lang="ts">
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import type {
  AdvancedQueryFilterFieldDefinition,
  AdvancedQueryFilterTag,
  AdvancedQueryTimeRangeField,
} from '@/shared/components/query-list';
import { AdvancedQueryFilterBuilderFrame } from '@/shared/components/query-list';
import * as BuilderHelpers from '@/shared/components/query-list';

import type { AppLogFilterState, AppLogSeverity, AppLogSortBy } from '../types/app-log';

type AppLogPresetKey = 'all' | 'errors' | 'warnings' | 'lastHour';
type AppTimeRangeKey = 'occurredRange';
type FilterKey = Exclude<keyof AppLogFilterState, 'keyword' | 'sorters' | 'occurredRange'>;
type BuilderFieldKey = 'timeRange' | 'sorterBuilder' | FilterKey;
type TagKey = FilterKey | AppTimeRangeKey | `sorter:${number}`;

const props = defineProps<{
  activePreset: AppLogPresetKey;
  loading?: boolean;
  modelValue: AppLogFilterState;
  presets: { key: AppLogPresetKey; title: string }[];
}>();

const emit = defineEmits<{
  (e: 'apply-preset', preset: AppLogPresetKey): void;
  (e: 'reset'): void;
  (e: 'search'): void;
  (e: 'update:modelValue', value: AppLogFilterState): void;
}>();

const { t } = useI18n();
const selectedFieldKey = ref<BuilderFieldKey>('timeRange');

const severityOptions = computed(() =>
  (['debug', 'info', 'warn', 'error'] satisfies AppLogSeverity[]).map((value) => ({
    label: value.toUpperCase(),
    value,
  })),
);
const sortByOptions = computed(() =>
  [
    ['appLog.filters.sortOccurredAt', 'occurred_at'],
    ['appLog.filters.sortSeverity', 'severity'],
    ['appLog.filters.sortComponent', 'component'],
  ].map(([labelKey, value]) => ({ label: t(labelKey), value: value as AppLogSortBy })),
);
const sortOrderOptions = computed(() =>
  ['desc', 'asc'].map((value) => ({ label: t(`appLog.filters.sort${value === 'desc' ? 'Desc' : 'Asc'}`), value })),
);
const sorterControls = BuilderHelpers.useAdvancedQuerySorterControlsForModel<AppLogSortBy, AppLogFilterState>(
  currentFilters,
  emitFilterState,
  normalizeSortBy,
  () => sortByOptions.value,
);
const { mutators: sorterMutators, ui: sorterUi } = sorterControls;

const definitions = computed<AdvancedQueryFilterFieldDefinition[]>(() => [
  { key: 'timeRange', kind: 'special', label: t('appLog.builder.fields.timeRange') },
  { key: 'sorterBuilder', kind: 'special', label: t('appLog.builder.fields.sorterBuilder') },
  {
    key: 'severity',
    kind: 'select',
    label: t('appLog.builder.fields.severity'),
    placeholder: t('appLog.filters.allSeverity'),
    options: severityOptions.value,
  },
  {
    key: 'component',
    kind: 'text',
    label: t('appLog.builder.fields.component'),
    placeholder: t('appLog.filters.component'),
  },
  {
    key: 'operation',
    kind: 'text',
    label: t('appLog.builder.fields.operation'),
    placeholder: t('appLog.filters.operation'),
  },
  {
    key: 'requestId',
    kind: 'text',
    label: t('appLog.builder.fields.requestId'),
    placeholder: t('appLog.filters.requestId'),
  },
  {
    key: 'traceId',
    kind: 'text',
    label: t('appLog.builder.fields.traceId'),
    placeholder: t('appLog.filters.traceId'),
  },
  {
    key: 'message',
    kind: 'text',
    label: t('appLog.builder.fields.message'),
    placeholder: t('appLog.filters.message'),
  },
  {
    key: 'error',
    kind: 'text',
    label: t('appLog.builder.fields.error'),
    placeholder: t('appLog.filters.error'),
  },
]);

const fieldValues = computed<Record<string, string | string[]>>(() => ({
  severity: props.modelValue.severity,
  component: props.modelValue.component,
  operation: props.modelValue.operation,
  requestId: props.modelValue.requestId,
  traceId: props.modelValue.traceId,
  message: props.modelValue.message,
  error: props.modelValue.error,
}));

const timeFields = computed<AdvancedQueryTimeRangeField[]>(() => [
  {
    key: 'occurredRange',
    label: t('appLog.filters.occurredRange'),
    value: props.modelValue.occurredRange,
    placeholder: [t('appLog.filters.occurredRange'), t('appLog.filters.occurredRange')],
  },
]);

const builderListeners = createAppLogBuilderListeners();

const builderFrame = createAppLogBuilderFrame();

function currentFilters() {
  return props.modelValue;
}

function emitFilterState(value: AppLogFilterState) {
  emit('update:modelValue', value);
}

function createFrameSource() {
  return props;
}

function createAppLogBuilderFrame() {
  return BuilderHelpers.createAdvancedQueryFilterBuilderFrameStateFromSource({
    fieldValues: () => fieldValues.value,
    fields: () => definitions.value,
    keyword: () => props.modelValue.keyword,
    listeners: builderListeners,
    selectedFieldKey,
    sorterUi,
    sortDirectionOptions: () => sortOrderOptions.value,
    source: createFrameSource,
    tags: () => activeFilterTags.value,
    timeFields: () => timeFields.value,
  });
}

const activeFilterTags = computed<AdvancedQueryFilterTag[]>(() => {
  const label = buildTimeTag('occurredRange', t('appLog.filters.occurredRange'));

  return BuilderHelpers.buildAdvancedQueryActiveTags<AppLogFilterState, FilterKey, AppLogSortBy>({
    fields: definitions.value,
    filterState: props.modelValue,
    sorterPrefix: t('appLog.sort.tagPrefix'),
    sorters: sorterUi.normalizedSorters.value,
    sortOptions: sortByOptions.value,
    timeTags: label ? [{ key: 'occurredRange', label }] : [],
  });
});

function updateField<Key extends keyof AppLogFilterState>(key: Key, value: AppLogFilterState[Key]) {
  emit('update:modelValue', BuilderHelpers.updateAdvancedQueryFilterStateField(props.modelValue, key, value));
}

function handleFieldUpdate(payload: { key: string; value: string | string[] }) {
  updateField(payload.key as keyof AppLogFilterState, payload.value as never);
}

function createAppLogBuilderListeners() {
  return BuilderHelpers.createAdvancedQueryBuilderListeners<
    AppLogPresetKey,
    BuilderFieldKey,
    { key: string; value: string[] }
  >({
    addSorter: sorterMutators.addSorter,
    clearTag: (key) => clearTag(key as TagKey),
    emitApplyPreset: (preset) => emit('apply-preset', preset),
    emitReset: () => emit('reset'),
    emitSearch: () => emit('search'),
    handleFieldUpdate,
    moveSorterDown: sorterMutators.moveSorterDown,
    moveSorterUp: sorterMutators.moveSorterUp,
    removeSorter: sorterMutators.removeSorter,
    selectedFieldKey,
    updateKeyword: (value) => updateField('keyword', value),
    updateSortDirection: sorterMutators.updateSortDirection,
    updateSortField: sorterMutators.updateSortField,
    updateTimeField: ({ key, value }) => updateTimeField(key as AppTimeRangeKey, value),
  });
}

function clearTag(key: TagKey) {
  if (key === 'occurredRange') {
    updateTimeField(key, []);
    return;
  }
  if (key.startsWith('sorter:')) {
    sorterMutators.removeSorter(Number(key.split(':')[1] || 0));
    return;
  }
  updateField(key as FilterKey, '');
}

function normalizeSortBy(value: string): AppLogSortBy {
  return value === 'severity' || value === 'component' ? value : 'occurred_at';
}

function updateTimeField(key: AppTimeRangeKey, value: string[]) {
  emit('update:modelValue', {
    ...props.modelValue,
    [key]: value,
  });
}

function buildTimeTag(key: AppTimeRangeKey, label: string) {
  const range = props.modelValue[key];
  if (!range.length) {
    return '';
  }

  return BuilderHelpers.buildAdvancedQueryTimeTag(label, range);
}
</script>
