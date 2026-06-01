<template>
  <management-toolbar>
    <template #filters>
      <div class="access-log-filters">
        <div class="access-log-filters__top-row">
          <t-input
            :model-value="modelValue.keyword"
            class="access-log-filters__keyword"
            clearable
            :placeholder="t('accessLog.page.searchPlaceholder')"
            @update:model-value="updateField('keyword', normalizeTextValue($event))"
          />
          <div class="access-log-filters__actions">
            <t-button theme="primary" :loading="loading" @click="$emit('search')">
              {{ t('accessLog.actions.search') }}
            </t-button>
            <t-button theme="default" variant="outline" @click="$emit('reset')">
              {{ t('accessLog.actions.reset') }}
            </t-button>
          </div>
        </div>

        <div v-if="summaryText" class="access-log-filters__summary">
          {{ summaryText }}
        </div>

        <section class="access-log-filters__group">
          <div class="access-log-filters__group-header">
            <span class="access-log-filters__group-title">{{ t('accessLog.builder.groups.time') }}</span>
          </div>
          <div class="access-log-filters__group-body">
            <t-select
              :model-value="selectedTimePreset"
              class="access-log-filters__time-preset"
              :options="timePresetOptions"
              @update:model-value="updateTimePreset"
            />
            <t-date-range-picker
              v-if="selectedTimePreset === 'custom'"
              :model-value="modelValue.startedRange"
              allow-input
              clearable
              enable-time-picker
              format="YYYY-MM-DD HH:mm:ss"
              :placeholder="[t('accessLog.filters.startedRange'), t('accessLog.filters.startedRange')]"
              @update:model-value="updateStartedRange"
            />
          </div>
        </section>

        <section class="access-log-filters__group">
          <div class="access-log-filters__group-header">
            <span class="access-log-filters__group-title">{{ t('accessLog.builder.groups.filters') }}</span>
          </div>
          <div class="access-log-filters__group-body access-log-filters__group-body--builder">
            <t-popup
              v-model:visible="builderVisible"
              attach="body"
              destroy-on-close
              placement="bottom-left"
              trigger="click"
            >
              <template #content>
                <div class="access-log-filter-builder">
                  <div class="access-log-filter-builder__header">
                    <span class="access-log-filter-builder__title">{{ t('accessLog.builder.title') }}</span>
                    <span class="access-log-filter-builder__hint">{{ t('accessLog.builder.hint') }}</span>
                  </div>

                  <div class="access-log-filter-builder__field-list">
                    <button
                      v-for="definition in definitions"
                      :key="definition.key"
                      :class="[
                        'access-log-filter-builder__field-button',
                        { 'access-log-filter-builder__field-button--active': selectedDefinitionKey === definition.key },
                      ]"
                      type="button"
                      @click="selectDefinition(definition.key)"
                    >
                      {{ definition.fieldLabel }}
                    </button>
                  </div>

                  <div v-if="selectedDefinition" class="access-log-filter-builder__editor">
                    <div class="access-log-filter-builder__editor-title">
                      {{ selectedDefinition.fieldLabel }}
                    </div>
                    <t-select
                      v-if="selectedDefinition.kind === 'select'"
                      :model-value="props.modelValue[selectedDefinition.key]"
                      clearable
                      :options="selectedDefinition.options"
                      :placeholder="selectedDefinition.placeholder"
                      @update:model-value="updateSelectField(selectedDefinition.key as SelectFilterKey, $event)"
                    />
                    <t-input
                      v-else
                      :model-value="String(props.modelValue[selectedDefinition.key] ?? '')"
                      clearable
                      :placeholder="selectedDefinition.placeholder"
                      @update:model-value="updateField(selectedDefinition.key, normalizeTextValue($event))"
                    />
                  </div>
                </div>
              </template>

              <t-button theme="default" variant="dashed">+ {{ t('accessLog.actions.addFilter') }}</t-button>
            </t-popup>
          </div>
        </section>

        <section class="access-log-filters__group">
          <div class="access-log-filters__group-header">
            <span class="access-log-filters__group-title">{{ t('accessLog.builder.groups.sort') }}</span>
          </div>
          <div class="access-log-filters__group-body access-log-filters__group-body--sort">
            <t-select
              :model-value="sortFieldValue"
              clearable
              class="access-log-filters__sort-select"
              :options="sortByOptions"
              :placeholder="t('accessLog.sort.fieldPlaceholder')"
              @update:model-value="updateSortField($event)"
            />
            <t-select
              v-if="sortFieldValue"
              :model-value="sortDirectionValue"
              clearable
              class="access-log-filters__sort-select"
              :options="sortOrderOptions"
              :placeholder="t('accessLog.sort.directionPlaceholder')"
              @update:model-value="updateSortDirection($event)"
            />
          </div>
        </section>

        <div v-if="presets.length" class="access-log-filters__preset-row">
          <span class="access-log-filters__preset-label">{{ t('accessLog.presets.label') }}</span>
          <t-button
            v-for="preset in presets"
            :key="preset.key"
            size="small"
            :theme="activePreset === preset.key ? 'primary' : 'default'"
            :variant="activePreset === preset.key ? 'base' : 'outline'"
            @click="$emit('apply-preset', preset.key)"
          >
            {{ preset.title }}
          </t-button>
        </div>

        <div v-if="activeFilterTags.length" class="access-log-filters__tag-row">
          <template v-for="tag in activeFilterTags" :key="tag.key">
            <t-tag
              closable
              max-width="280"
              size="small"
              theme="primary"
              :title="tag.label"
              variant="light-outline"
              @close="clearField(tag.key)"
            >
              {{ tag.label }}
            </t-tag>
          </template>
        </div>
      </div>
    </template>
  </management-toolbar>
</template>
<script setup lang="ts">
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { ManagementToolbar } from '@/shared/components/management';
import {
  buildRecentHoursLocalRange,
  getSingleSorter,
  joinQuerySummary,
  normalizeSingleSorterDirection,
  normalizeSingleSorterField,
  prependSingleSorterTag,
} from '@/shared/observability';

import type {
  AccessLogFilterState,
  AccessLogPathMatch,
  AccessLogSortBy,
  AccessLogSortOrder,
} from '../types/access-log';

type AccessLogPresetKey =
  | 'all'
  | 'todayErrors'
  | 'status4xx'
  | 'status5xx'
  | 'slowRequests'
  | 'currentUser'
  | 'lastHour';
type TimePresetKey = 'last24h' | 'last7d' | 'last30d' | 'custom';
type FilterKey = Exclude<keyof AccessLogFilterState, 'keyword' | 'pathMatch' | 'route' | 'sorters' | 'startedRange'>;
type SelectFilterKey = 'method';
type BaseFilterDefinition<Key extends FilterKey> = {
  key: Key;
  fieldLabel: string;
};
type TextFilterDefinition = BaseFilterDefinition<Exclude<FilterKey, 'method'>> & {
  kind: 'text';
  placeholder: string;
};
type SelectFilterDefinition = BaseFilterDefinition<'method'> & {
  kind: 'select';
  placeholder: string;
  options: Array<{ label: string; value: string }>;
};
type FilterDefinition = TextFilterDefinition | SelectFilterDefinition;

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

const builderVisible = ref(false);
const selectedDefinitionKey = ref<FilterKey>('requestId');

const methodOptions = computed(() =>
  ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'].map((value) => ({ label: value, value })),
);
const timePresetOptions = computed(() => [
  { label: t('accessLog.time.last24h'), value: 'last24h' },
  { label: t('accessLog.time.last7d'), value: 'last7d' },
  { label: t('accessLog.time.last30d'), value: 'last30d' },
  { label: t('accessLog.time.custom'), value: 'custom' },
]);
const sortByOptions = computed(() => [
  { label: t('accessLog.filters.sortStartedAt'), value: 'started_at' },
  { label: t('accessLog.filters.sortOccurredAt'), value: 'occurred_at' },
  { label: t('accessLog.filters.sortDuration'), value: 'duration_ms' },
  { label: t('accessLog.filters.sortStatusCode'), value: 'status_code' },
]);
const sortOrderOptions = computed(() => [
  { label: t('accessLog.filters.sortDesc'), value: 'desc' },
  { label: t('accessLog.filters.sortAsc'), value: 'asc' },
]);

const definitions = computed<FilterDefinition[]>(() => [
  {
    key: 'requestId',
    kind: 'text',
    fieldLabel: t('accessLog.builder.fields.requestId'),
    placeholder: t('accessLog.filters.requestId'),
  },
  {
    key: 'userId',
    kind: 'text',
    fieldLabel: t('accessLog.builder.fields.userId'),
    placeholder: t('accessLog.filters.userId'),
  },
  {
    key: 'username',
    kind: 'text',
    fieldLabel: t('accessLog.builder.fields.username'),
    placeholder: t('accessLog.filters.username'),
  },
  {
    key: 'method',
    kind: 'select',
    fieldLabel: t('accessLog.builder.fields.method'),
    placeholder: t('accessLog.filters.method'),
    options: methodOptions.value,
  },
  {
    key: 'path',
    kind: 'text',
    fieldLabel: t('accessLog.builder.fields.path'),
    placeholder: t('accessLog.filters.path'),
  },
  {
    key: 'statusCode',
    kind: 'text',
    fieldLabel: t('accessLog.builder.fields.statusCode'),
    placeholder: t('accessLog.filters.statusCode'),
  },
  {
    key: 'durationMinMs',
    kind: 'text',
    fieldLabel: t('accessLog.builder.fields.durationMinMs'),
    placeholder: t('accessLog.filters.durationMin'),
  },
  {
    key: 'durationMaxMs',
    kind: 'text',
    fieldLabel: t('accessLog.builder.fields.durationMaxMs'),
    placeholder: t('accessLog.filters.durationMax'),
  },
]);

const selectedDefinition = computed(() =>
  definitions.value.find((definition) => definition.key === selectedDefinitionKey.value),
);
const activeSorter = computed(() => getSingleSorter(props.modelValue.sorters));
const sortFieldValue = computed(() => activeSorter.value?.field ?? '');
const sortDirectionValue = computed(() => activeSorter.value?.direction ?? '');
const selectedTimePreset = computed<TimePresetKey>(() => {
  const [startedFrom, startedTo] = props.modelValue.startedRange;
  if (!startedFrom || !startedTo) {
    return 'custom';
  }

  const now = new Date();
  const candidateRanges: Record<Exclude<TimePresetKey, 'custom'>, string[]> = {
    last24h: buildRecentHoursLocalRange(now, 24),
    last7d: buildRecentHoursLocalRange(now, 24 * 7),
    last30d: buildRecentHoursLocalRange(now, 24 * 30),
  };

  for (const [preset, range] of Object.entries(candidateRanges) as Array<
    [Exclude<TimePresetKey, 'custom'>, string[]]
  >) {
    if (range[0] === startedFrom && range[1] === startedTo) {
      return preset;
    }
  }

  return 'custom';
});
const summaryText = computed(() => {
  const sorterLabel =
    activeSorter.value && sortByOptions.value.find((option) => option.value === activeSorter.value?.field)?.label
      ? `${t('accessLog.builder.summary.sortBy', {
          field: sortByOptions.value.find((option) => option.value === activeSorter.value?.field)?.label,
          direction:
            activeSorter.value?.direction === 'asc' ? t('accessLog.filters.sortAsc') : t('accessLog.filters.sortDesc'),
        })}`
      : '';
  const timeLabel =
    selectedTimePreset.value === 'custom'
      ? props.modelValue.startedRange.length
        ? `${t('accessLog.builder.summary.customTime', {
            range: props.modelValue.startedRange.filter(Boolean).join(' ~ '),
          })}`
        : ''
      : t(`accessLog.time.${selectedTimePreset.value}`);
  const statusLabel = props.modelValue.statusCode
    ? `${t('accessLog.builder.fields.statusCode')} ${props.modelValue.statusCode}`
    : '';

  return joinQuerySummary([timeLabel, statusLabel, sorterLabel]);
});

const activeFilterTags = computed(() => {
  const filterTags = definitions.value
    .map((definition) => {
      const rawValue = props.modelValue[definition.key];
      const value = typeof rawValue === 'string' ? rawValue.trim() : rawValue;
      if (!value) {
        return null;
      }
      const label =
        definition.kind === 'select'
          ? definition.options.find((option) => option.value === value)?.label || String(value)
          : String(value);
      return { key: definition.key, label: `${definition.fieldLabel}：${label}` };
    })
    .flat()
    .filter((item): item is { key: FilterKey; label: string } => Boolean(item));

  return prependSingleSorterTag(filterTags, activeSorter.value, sortByOptions.value, t('accessLog.sort.tagPrefix'));
});

function updateField<Key extends keyof AccessLogFilterState>(key: Key, value: AccessLogFilterState[Key]) {
  emit('update:modelValue', {
    ...props.modelValue,
    [key]: typeof value === 'string' ? value.trim() : value,
  });
}

function updateSelectField(key: SelectFilterKey, value: string | number | Array<string | number> | undefined) {
  updateField(key, normalizeSelect(value));
}

function updateStartedRange(value: string[] | undefined) {
  emit('update:modelValue', {
    ...props.modelValue,
    startedRange: Array.isArray(value) ? value : [],
  });
}

function updateTimePreset(value: string | number | Array<string | number> | undefined) {
  if (typeof value !== 'string') {
    return;
  }

  const now = new Date();
  const presetRanges: Record<Exclude<TimePresetKey, 'custom'>, string[]> = {
    last24h: buildRecentHoursLocalRange(now, 24),
    last7d: buildRecentHoursLocalRange(now, 24 * 7),
    last30d: buildRecentHoursLocalRange(now, 24 * 30),
  };

  emit('update:modelValue', {
    ...props.modelValue,
    startedRange:
      value === 'custom' ? props.modelValue.startedRange : presetRanges[value as Exclude<TimePresetKey, 'custom'>],
  });
}

function clearField(key: FilterKey | 'sorter') {
  if (key === 'sorter') {
    emit('update:modelValue', {
      ...props.modelValue,
      sorters: [],
    });
    return;
  }
  if (key === 'method') {
    updateField('method', '');
    return;
  }
  updateField(key, '');
}

function selectDefinition(key: FilterKey) {
  selectedDefinitionKey.value = key;
}

function normalizeTextValue(value: string | number | undefined) {
  return typeof value === 'string' ? value : '';
}

function normalizeSelect(value: string | number | Array<string | number> | undefined) {
  return typeof value === 'string' ? value : '';
}

function normalizeSortBy(value: string): AccessLogSortBy {
  return value === 'occurred_at' || value === 'duration_ms' || value === 'status_code' ? value : 'started_at';
}

function normalizeSortOrder(value: string): AccessLogSortOrder {
  return value === 'asc' ? 'asc' : 'desc';
}

function updateSortField(value: string | number | Array<string | number> | undefined) {
  emit('update:modelValue', {
    ...props.modelValue,
    sorters: normalizeSingleSorterField(value, activeSorter.value?.direction, normalizeSortBy),
  });
}

function updateSortDirection(value: string | number | Array<string | number> | undefined) {
  emit('update:modelValue', {
    ...props.modelValue,
    sorters: normalizeSingleSorterDirection(value, activeSorter.value?.field, normalizeSortOrder),
  });
}

void (null as unknown as AccessLogPathMatch);
</script>
<style scoped lang="less">
.access-log-filters {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
}

.access-log-filters__top-row,
.access-log-filters__group-body,
.access-log-filters__preset-row {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.access-log-filters__keyword {
  flex: 1 1 340px;
  min-width: 240px;
}

.access-log-filters__actions {
  display: flex;
  gap: 12px;
  margin-left: auto;
}

.access-log-filters__summary {
  color: var(--td-text-color-secondary);
  font-size: 13px;
}

.access-log-filters__group {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-border);
  border-radius: var(--td-radius-large);
  padding: 12px 14px;
}

.access-log-filters__group-header {
  margin-bottom: 10px;
}

.access-log-filters__group-title {
  color: var(--td-text-color-primary);
  font-size: 13px;
  font-weight: 600;
}

.access-log-filters__group-body--builder {
  justify-content: flex-start;
}

.access-log-filters__group-body--sort {
  align-items: center;
}

.access-log-filters__time-preset {
  min-width: 180px;
}

.access-log-filters__tag-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.access-log-filters__preset-row {
  flex: 1 1 auto;
  min-width: 0;
}

.access-log-filters__sort-select {
  min-width: 160px;
}

.access-log-filters__preset-label {
  color: var(--td-text-color-secondary);
  font-size: 12px;
  white-space: nowrap;
}

.access-log-filter-builder {
  display: grid;
  gap: 16px;
  grid-template-columns: minmax(160px, 200px) minmax(280px, 360px);
  padding: 8px;
}

.access-log-filter-builder__header {
  display: flex;
  flex-direction: column;
  gap: 4px;
  grid-column: 1 / -1;
}

.access-log-filter-builder__title {
  color: var(--td-text-color-primary);
  font-size: 14px;
  font-weight: 600;
}

.access-log-filter-builder__hint {
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.access-log-filter-builder__field-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.access-log-filter-builder__field-button {
  background: var(--td-bg-color-container-hover);
  border: 1px solid transparent;
  border-radius: var(--td-radius-default);
  box-shadow: inset 0 0 0 1px transparent;
  color: var(--td-text-color-primary);
  cursor: pointer;
  font: inherit;
  padding: 8px 12px;
  text-align: left;
  transition:
    border-color 0.2s ease,
    box-shadow 0.2s ease,
    transform 0.2s ease;
}

.access-log-filter-builder__field-button:hover {
  border-color: var(--td-brand-color);
  transform: translateX(2px);
}

.access-log-filter-builder__field-button--active {
  background: var(--td-brand-color-light);
  border-color: var(--td-brand-color);
  box-shadow: inset 0 0 0 1px var(--td-brand-color-focus);
}

.access-log-filter-builder__time-group {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.access-log-filter-builder__time-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.access-log-filter-builder__time-label {
  color: var(--td-text-color-primary);
  font-size: 12px;
  font-weight: 600;
}

.access-log-filter-builder__editor {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.access-log-filter-builder__editor-title {
  color: var(--td-text-color-primary);
  font-size: 13px;
  font-weight: 600;
}

@media (width <= 960px) {
  .access-log-filters__actions {
    margin-left: 0;
  }

  .access-log-filter-builder {
    grid-template-columns: 1fr;
  }
}

@media (width <= 768px) {
  .access-log-filters__top-row,
  .access-log-filters__group-body,
  .access-log-filters__preset-row {
    align-items: stretch;
    flex-direction: column;
  }

  .access-log-filters__keyword,
  .access-log-filters__actions,
  .access-log-filters__sort-row,
  .access-log-filters__preset-row {
    min-width: 0;
    width: 100%;
  }

  .access-log-filters__actions {
    justify-content: flex-start;
  }
}
</style>
