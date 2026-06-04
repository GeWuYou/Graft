<template>
  <log-filter-builder-frame :frame="builderFrame" message-prefix="accessLog" />
</template>
<script setup lang="ts">
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import type {
  LogFilterFieldDefinition,
  LogFilterTag,
  LogTimeRangeField,
} from '@/shared/observability/log-filter-builder';
import {
  buildLogActiveTags,
  buildLogTimeTag,
  createLogBuilderListeners,
  createLogFilterBuilderFrameStateFromSource,
  updateLogFilterStateField,
  useLogSorterControlsForModel,
} from '@/shared/observability/log-filter-builder-helpers';
import LogFilterBuilderFrame from '@/shared/observability/LogFilterBuilderFrame.vue';

import { buildAccessLogSortOptions } from '../shared/presentation';
import type { AccessLogFilterState, AccessLogPathMatch, AccessLogSortBy } from '../types/access-log';

type AccessLogPresetKey =
  | 'all'
  | 'todayErrors'
  | 'status4xx'
  | 'status5xx'
  | 'slowRequests'
  | 'currentUser'
  | 'lastHour';
type AccessTimeRangeKey = 'startedRange' | 'occurredRange';
type FilterKey = Exclude<
  keyof AccessLogFilterState,
  'keyword' | 'pathMatch' | 'route' | 'sorters' | 'startedRange' | 'occurredRange'
>;
type BuilderFieldKey = 'timeRange' | 'sorterBuilder' | FilterKey;
type TagKey = FilterKey | AccessTimeRangeKey | `sorter:${number}`;

const props = defineProps<{
  activePreset: AccessLogPresetKey;
  loading?: boolean;
  modelValue: AccessLogFilterState;
  presets: { key: AccessLogPresetKey; title: string }[];
}>();

const emit = defineEmits<{
  (e: 'apply-preset', preset: AccessLogPresetKey): void;
  (e: 'reset'): void;
  (e: 'search'): void;
  (e: 'update:modelValue', value: AccessLogFilterState): void;
}>();

const { t } = useI18n();

const selectedFieldKey = ref<BuilderFieldKey>('timeRange');

const methodOptions = computed(() =>
  ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'].map((value) => ({ label: value, value })),
);
const sortByOptions = computed(() => buildAccessLogSortOptions(t));
const sortOrderOptions = computed(() => [
  { label: t('accessLog.filters.sortDesc'), value: 'desc' },
  { label: t('accessLog.filters.sortAsc'), value: 'asc' },
]);
const sorterControlInput = {
  emitValue: (value: AccessLogFilterState) => emit('update:modelValue', value),
  model: () => props.modelValue,
  options: () => sortByOptions.value,
};
const sorterControls = useLogSorterControlsForModel<AccessLogSortBy, AccessLogFilterState>(
  sorterControlInput.model,
  sorterControlInput.emitValue,
  normalizeSortBy,
  sorterControlInput.options,
);
const { mutators: sorterMutators, ui: sorterUi } = sorterControls;

const definitions = computed<LogFilterFieldDefinition[]>(() => [
  { key: 'timeRange', kind: 'special', label: t('accessLog.builder.fields.timeRange') },
  { key: 'sorterBuilder', kind: 'special', label: t('accessLog.builder.fields.sorterBuilder') },
  {
    key: 'requestId',
    kind: 'text',
    label: t('accessLog.builder.fields.requestId'),
    placeholder: t('accessLog.filters.requestId'),
  },
  {
    key: 'userId',
    kind: 'text',
    label: t('accessLog.builder.fields.userId'),
    placeholder: t('accessLog.filters.userId'),
  },
  {
    key: 'username',
    kind: 'text',
    label: t('accessLog.builder.fields.username'),
    placeholder: t('accessLog.filters.username'),
  },
  {
    key: 'method',
    kind: 'select',
    label: t('accessLog.builder.fields.method'),
    placeholder: t('accessLog.filters.method'),
    options: methodOptions.value,
  },
  {
    key: 'path',
    kind: 'text',
    label: t('accessLog.builder.fields.path'),
    placeholder: t('accessLog.filters.path'),
  },
  {
    key: 'statusCode',
    kind: 'text',
    label: t('accessLog.builder.fields.statusCode'),
    placeholder: t('accessLog.filters.statusCode'),
  },
  {
    key: 'durationMinMs',
    kind: 'text',
    label: t('accessLog.builder.fields.durationMinMs'),
    placeholder: t('accessLog.filters.durationMin'),
  },
  {
    key: 'durationMaxMs',
    kind: 'text',
    label: t('accessLog.builder.fields.durationMaxMs'),
    placeholder: t('accessLog.filters.durationMax'),
  },
]);

const fieldValues = computed<Record<string, string | string[]>>(() => ({
  requestId: props.modelValue.requestId,
  userId: props.modelValue.userId,
  username: props.modelValue.username,
  method: props.modelValue.method,
  path: props.modelValue.path,
  statusCode: props.modelValue.statusCode,
  durationMinMs: props.modelValue.durationMinMs,
  durationMaxMs: props.modelValue.durationMaxMs,
}));

const timeFields = computed<LogTimeRangeField[]>(() => [
  {
    key: 'startedRange',
    label: t('accessLog.filters.startedRange'),
    value: props.modelValue.startedRange,
    placeholder: [t('accessLog.filters.startedRange'), t('accessLog.filters.startedRange')],
  },
  {
    key: 'occurredRange',
    label: t('accessLog.filters.occurredRange'),
    value: props.modelValue.occurredRange,
    placeholder: [t('accessLog.filters.occurredRange'), t('accessLog.filters.occurredRange')],
  },
]);

const builderListeners = createAccessLogBuilderListeners();
const frameAccessors = {
  fieldValues,
  filterDefinitions: definitions,
  timeRangeFields: timeFields,
};

const builderFrame = createAccessLogBuilderFrame();

function createAccessLogBuilderFrame() {
  return createLogFilterBuilderFrameStateFromSource({
    fieldValues: () => frameAccessors.fieldValues.value,
    fields: () => frameAccessors.filterDefinitions.value,
    keyword: () => props.modelValue.keyword,
    listeners: builderListeners,
    selectedFieldKey,
    sorterUi,
    sortDirectionOptions: () => sortOrderOptions.value,
    source: () => props,
    tags: () => activeFilterTags.value,
    timeFields: () => frameAccessors.timeRangeFields.value,
  });
}

const activeFilterTags = computed<LogFilterTag[]>(() => {
  const timeTags: LogFilterTag[] = [];
  const startedTimeTag = buildTimeTag('startedRange', t('accessLog.filters.startedRange'));
  if (startedTimeTag) {
    timeTags.push({ key: 'startedRange', label: startedTimeTag });
  }
  const occurredTimeTag = buildTimeTag('occurredRange', t('accessLog.filters.occurredRange'));
  if (occurredTimeTag) {
    timeTags.push({ key: 'occurredRange', label: occurredTimeTag });
  }

  return buildLogActiveTags<AccessLogFilterState, FilterKey, AccessLogSortBy>({
    fieldSeparator: '：',
    fields: definitions.value,
    filterState: props.modelValue,
    sorterPrefix: t('accessLog.sort.tagPrefix'),
    sorters: sorterUi.normalizedSorters.value,
    sortOptions: sortByOptions.value,
    timeTags,
  });
});

function updateField<Key extends keyof AccessLogFilterState>(key: Key, value: AccessLogFilterState[Key]) {
  emit('update:modelValue', updateLogFilterStateField(props.modelValue, key, value));
}

function handleFieldUpdate(payload: { key: string; value: string | string[] }) {
  updateField(payload.key as keyof AccessLogFilterState, payload.value as never);
}

function createAccessLogBuilderListeners() {
  return createLogBuilderListeners<AccessLogPresetKey, BuilderFieldKey, { key: string; value: string[] }>({
    handleFieldUpdate,
    selectedFieldKey,
    updateKeyword: (value) => updateField('keyword', value),
    emitSearch: () => emit('search'),
    addSorter: sorterMutators.addSorter,
    updateSortField: sorterMutators.updateSortField,
    clearTag: (key) => clearTag(key as TagKey),
    moveSorterUp: sorterMutators.moveSorterUp,
    emitReset: () => emit('reset'),
    updateTimeField: ({ key, value }) => updateTimeField(key as AccessTimeRangeKey, value),
    removeSorter: sorterMutators.removeSorter,
    updateSortDirection: sorterMutators.updateSortDirection,
    moveSorterDown: sorterMutators.moveSorterDown,
    emitApplyPreset: (preset) => emit('apply-preset', preset),
  });
}

function clearField(key: FilterKey) {
  if (key === 'method') {
    updateField('method', '');
    return;
  }
  updateField(key, '');
}

function clearTag(key: TagKey) {
  if (key === 'startedRange' || key === 'occurredRange') {
    updateTimeField(key, []);
    return;
  }

  if (key.startsWith('sorter:')) {
    sorterMutators.removeSorter(Number(key.split(':')[1] || 0));
    return;
  }

  clearField(key as FilterKey);
}

function normalizeSortBy(value: string): AccessLogSortBy {
  return value === 'occurred_at' || value === 'duration_ms' || value === 'status_code' ? value : 'started_at';
}

function updateTimeField(key: AccessTimeRangeKey, value: string[]) {
  emit('update:modelValue', {
    ...props.modelValue,
    [key]: value,
  });
}

function buildTimeTag(key: AccessTimeRangeKey, label: string) {
  const range = props.modelValue[key];
  if (!range.length) {
    return '';
  }

  return buildLogTimeTag(label, range, '：');
}

void (null as unknown as AccessLogPathMatch);
</script>
