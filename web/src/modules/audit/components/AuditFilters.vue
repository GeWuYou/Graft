<template>
  <management-toolbar>
    <template #filters>
      <div class="audit-filters">
        <div class="audit-filters__top-row">
          <t-input
            :model-value="modelValue.keyword"
            class="audit-filters__keyword"
            clearable
            :placeholder="t('audit.logList.filters.keywordPlaceholder')"
            @update:model-value="updateField('keyword', $event)"
          />
          <div class="audit-filters__actions">
            <t-button theme="primary" :loading="loading" @click="$emit('search')">
              {{ t('audit.logList.actions.search') }}
            </t-button>
            <t-button theme="default" variant="outline" @click="$emit('reset')">
              {{ t('audit.logList.actions.reset') }}
            </t-button>
          </div>
        </div>

        <div v-if="summaryText" class="audit-filters__summary">
          {{ summaryText }}
        </div>

        <section class="audit-filters__group">
          <div class="audit-filters__group-header">
            <span class="audit-filters__group-title">{{ t('audit.logList.builder.groups.time') }}</span>
          </div>
          <div class="audit-filters__group-body">
            <t-select
              :model-value="selectedTimePreset"
              class="audit-filters__time-preset"
              :options="timePresetOptions"
              @update:model-value="updateTimePreset($event)"
            />
            <t-date-range-picker
              v-if="selectedTimePreset === 'custom'"
              :model-value="modelValue.createdRange"
              allow-input
              clearable
              class="audit-filters__date"
              enable-time-picker
              format="YYYY-MM-DD HH:mm:ss"
              :placeholder="dateRangePlaceholder"
              @update:model-value="updateField('createdRange', $event)"
            />
          </div>
        </section>

        <section class="audit-filters__group">
          <div class="audit-filters__group-header">
            <span class="audit-filters__group-title">{{ t('audit.logList.builder.groups.filters') }}</span>
          </div>
          <div class="audit-filters__group-body audit-filters__group-body--builder">
            <t-popup
              v-model:visible="builderVisible"
              attach="body"
              destroy-on-close
              overlay-class-name="audit-filter-builder-popup"
              placement="bottom-left"
              trigger="click"
            >
              <template #content>
                <div class="audit-filter-builder">
                  <div class="audit-filter-builder__header">
                    <span class="audit-filter-builder__title">{{ t('audit.logList.builder.title') }}</span>
                    <span class="audit-filter-builder__hint">{{ t('audit.logList.builder.hint') }}</span>
                  </div>

                  <div class="audit-filter-builder__field-list">
                    <button
                      v-for="definition in availableDefinitions"
                      :key="definition.key"
                      class="audit-filter-builder__field-button"
                      type="button"
                      @click="selectDefinition(definition.key)"
                    >
                      {{ t(definition.fieldLabelKey) }}
                    </button>
                  </div>

                  <div v-if="selectedDefinition" class="audit-filter-builder__editor">
                    <div class="audit-filter-builder__editor-title">
                      {{ t(selectedDefinition.fieldLabelKey) }}
                    </div>
                    <t-select
                      v-if="selectedDefinition.kind === 'select'"
                      :model-value="selectValue(selectedDefinition.key)"
                      clearable
                      :options="selectedDefinition.options.value"
                      :placeholder="t(selectedDefinition.placeholderKey)"
                      @update:model-value="updateField(selectedDefinition.key, normalizeSelectValue($event))"
                    />
                    <t-select
                      v-else-if="selectedDefinition.kind === 'multi-select'"
                      :model-value="multiSelectValue(selectedDefinition.key)"
                      clearable
                      filterable
                      multiple
                      :min-collapsed-num="2"
                      :options="selectedDefinition.options.value"
                      :placeholder="t(selectedDefinition.placeholderKey)"
                      @update:model-value="updateField(selectedDefinition.key, normalizeMultiSelectValue($event))"
                    />
                    <t-tag-input
                      v-else-if="selectedDefinition.kind === 'tag-input'"
                      :model-value="tagInputValue(selectedDefinition.key)"
                      clearable
                      :input-props="{ placeholder: t(selectedDefinition.placeholderKey) }"
                      @update:model-value="updateField(selectedDefinition.key, normalizeTagInputValue($event))"
                    />
                    <t-input
                      v-else
                      :model-value="textValue(selectedDefinition.key)"
                      clearable
                      :placeholder="t(selectedDefinition.placeholderKey)"
                      @update:model-value="updateField(selectedDefinition.key, normalizeTextValue($event))"
                    />
                  </div>
                </div>
              </template>

              <t-button theme="default" variant="dashed"> + {{ t('audit.logList.actions.addFilter') }} </t-button>
            </t-popup>
          </div>
        </section>

        <section class="audit-filters__group">
          <div class="audit-filters__group-header">
            <span class="audit-filters__group-title">{{ t('audit.logList.builder.groups.sort') }}</span>
          </div>
          <div class="audit-filters__group-body audit-filters__group-body--sort">
            <t-select
              :model-value="sortFieldValue"
              clearable
              class="audit-filters__sort-select"
              :options="sortFieldOptions"
              :placeholder="t('audit.logList.sort.fieldPlaceholder')"
              @update:model-value="updateSortField($event)"
            />
            <t-select
              v-if="sortFieldValue"
              :model-value="sortDirectionValue"
              clearable
              class="audit-filters__sort-select"
              :options="sortDirectionOptions"
              :placeholder="t('audit.logList.sort.directionPlaceholder')"
              @update:model-value="updateSortDirection($event)"
            />
          </div>
        </section>

        <div class="audit-filters__preset-row">
          <span class="audit-filters__preset-label">{{ t('audit.logList.presets.label') }}</span>
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

        <div v-if="activeFilterTags.length" class="audit-filters__tag-row">
          <t-tag
            v-for="tag in activeFilterTags"
            :key="tag.key"
            :closable="!isLocked(tag.key)"
            max-width="240"
            theme="primary"
            variant="light-outline"
            @close="isLocked(tag.key) ? undefined : tag.key === 'sorter' ? clearSorter() : clearField(tag.key)"
          >
            {{ tag.label }}
          </t-tag>
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
  joinQuerySummary,
  normalizeSingleSorterDirection,
  normalizeSingleSorterField,
  prependSingleSorterTag,
  useSingleSorterSelection,
} from '@/shared/observability';

import type { AuditQuickPresetKey } from '../contract/presets';
import { AUDIT_TIME_PRESET, type AuditTimePreset } from '../contract/time-presets';
import type {
  AuditFilterDefinition,
  AuditFilterKey,
  AuditFilterOption,
  AuditMultiSelectFilterKey,
  AuditSingleSelectFilterKey,
  AuditTagInputFilterKey,
  AuditTextFilterKey,
} from '../shared/filter-definitions';
import type { AuditClientFilterState } from '../shared/presentation';
import type { AuditSortBy, AuditSortOrder } from '../types/audit';

type FilterTag = { key: AuditFilterKey; label: string };
type TimePresetSelection = AuditTimePreset | 'custom';

const props = defineProps<{
  activePreset: AuditQuickPresetKey;
  lockedFields?: AuditFilterKey[];
  loading?: boolean;
  modelValue: AuditClientFilterState;
  presets: { key: AuditQuickPresetKey; title: string }[];
}>();

const emit = defineEmits<{
  (e: 'apply-preset', preset: AuditQuickPresetKey): void;
  (e: 'reset'): void;
  (e: 'search'): void;
  (e: 'update:modelValue', value: AuditClientFilterState): void;
}>();

const { t } = useI18n();

const builderVisible = ref(false);
const selectedDefinitionKey = ref<AuditFilterKey>('actor');
const timePresetOptions = computed(() => [
  { label: t('audit.logList.time.last24h'), value: AUDIT_TIME_PRESET.LAST_24H },
  { label: t('audit.logList.time.last7d'), value: AUDIT_TIME_PRESET.LAST_7D },
  { label: t('audit.logList.time.last30d'), value: AUDIT_TIME_PRESET.LAST_30D },
  { label: t('audit.logList.time.custom'), value: 'custom' },
]);

const actionOptions = computed<AuditFilterOption[]>(() => [
  { label: t('audit.logList.filterOptions.auth'), value: 'auth' },
  { label: t('audit.logList.filterOptions.role'), value: 'role' },
  { label: t('audit.logList.filterOptions.permission'), value: 'permission' },
  { label: t('audit.logList.filterOptions.session'), value: 'session' },
]);

const actionPrefixOptions = computed<AuditFilterOption[]>(() => [
  { label: t('audit.logList.filterOptions.authPrefix'), value: 'auth.' },
  { label: t('audit.logList.filterOptions.rbacPrefix'), value: 'rbac.' },
  { label: t('audit.logList.filterOptions.rolePrefix'), value: 'role.' },
  { label: t('audit.logList.filterOptions.permissionPrefix'), value: 'permission.' },
]);

const sourceOptions = computed<AuditFilterOption[]>(() => [
  { label: t('audit.common.source.REQUEST'), value: 'REQUEST' },
  { label: t('audit.common.source.SECURITY_EVENT'), value: 'SECURITY_EVENT' },
  { label: t('audit.common.source.DOMAIN_EVENT'), value: 'DOMAIN_EVENT' },
]);

const businessCategoryOptions = computed<AuditFilterOption[]>(() => [
  { label: t('audit.logList.businessCategory.failedOperations'), value: 'failed_operations' },
  { label: t('audit.logList.businessCategory.highRiskOperations'), value: 'high_risk_operations' },
  { label: t('audit.logList.businessCategory.sensitiveOperations'), value: 'sensitive_operations' },
  { label: t('audit.logList.businessCategory.authFailures'), value: 'auth_failures' },
  { label: t('audit.logList.businessCategory.permissionDenials'), value: 'permission_denials' },
  { label: t('audit.logList.businessCategory.rbacChanges'), value: 'rbac_changes' },
  { label: t('audit.logList.businessCategory.criticalSecurity'), value: 'critical_security' },
]);

const resourceTypeOptions = computed<AuditFilterOption[]>(() => [
  { label: t('audit.logList.filterOptions.userResource'), value: 'user' },
  { label: t('audit.logList.filterOptions.roleResource'), value: 'role' },
  { label: t('audit.logList.filterOptions.permissionResource'), value: 'permission' },
  { label: t('audit.logList.filterOptions.authResource'), value: 'auth' },
]);

const resultOptions = computed<AuditFilterOption[]>(() => [
  { label: t('audit.logList.filterOptions.SUCCESS'), value: 'SUCCESS' },
  { label: t('audit.logList.filterOptions.FAILED'), value: 'FAILED' },
  { label: t('audit.logList.filterOptions.DENIED'), value: 'DENIED' },
  { label: t('audit.logList.filterOptions.ERROR'), value: 'ERROR' },
]);

const successOptions = computed<AuditFilterOption[]>(() => [
  { label: t('audit.logList.filterOptions.SUCCESS'), value: 'true' },
  { label: t('audit.logList.filterOptions.FAILED'), value: 'false' },
]);

const riskOptions = computed<AuditFilterOption[]>(() => [
  { label: t('audit.logList.filterOptions.LOW'), value: 'LOW' },
  { label: t('audit.logList.filterOptions.MEDIUM'), value: 'MEDIUM' },
  { label: t('audit.logList.filterOptions.HIGH'), value: 'HIGH' },
  { label: t('audit.logList.filterOptions.CRITICAL'), value: 'CRITICAL' },
]);
const sortFieldOptions = computed<AuditFilterOption[]>(() => [
  { label: t('audit.logList.sort.createdAt'), value: 'created_at' },
]);
const sortDirectionOptions = computed<AuditFilterOption[]>(() => [
  { label: t('audit.logList.sort.desc'), value: 'desc' },
  { label: t('audit.logList.sort.asc'), value: 'asc' },
]);

const definitions = computed<AuditFilterDefinition[]>(() => [
  {
    key: 'action',
    kind: 'select',
    fieldLabelKey: 'audit.logList.builder.fields.action',
    placeholderKey: 'audit.logList.filters.actionPlaceholder',
    options: actionOptions,
  },
  {
    key: 'actionPrefixes',
    kind: 'multi-select',
    fieldLabelKey: 'audit.logList.builder.fields.actionPrefixes',
    placeholderKey: 'audit.logList.filters.actionPrefixesPlaceholder',
    options: actionPrefixOptions,
  },
  {
    key: 'actionKeywords',
    kind: 'tag-input',
    fieldLabelKey: 'audit.logList.builder.fields.actionKeywords',
    placeholderKey: 'audit.logList.filters.actionKeywordsPlaceholder',
  },
  {
    key: 'result',
    kind: 'select',
    fieldLabelKey: 'audit.logList.builder.fields.result',
    placeholderKey: 'audit.logList.filters.resultPlaceholder',
    options: resultOptions,
  },
  {
    key: 'results',
    kind: 'multi-select',
    fieldLabelKey: 'audit.logList.builder.fields.results',
    placeholderKey: 'audit.logList.filters.resultsPlaceholder',
    options: resultOptions,
  },
  {
    key: 'riskLevel',
    kind: 'select',
    fieldLabelKey: 'audit.logList.builder.fields.riskLevel',
    placeholderKey: 'audit.logList.filters.riskPlaceholder',
    options: riskOptions,
  },
  {
    key: 'riskLevels',
    kind: 'multi-select',
    fieldLabelKey: 'audit.logList.builder.fields.riskLevels',
    placeholderKey: 'audit.logList.filters.riskLevelsPlaceholder',
    options: riskOptions,
  },
  {
    key: 'success',
    kind: 'select',
    fieldLabelKey: 'audit.logList.builder.fields.success',
    placeholderKey: 'audit.logList.filters.successPlaceholder',
    options: successOptions,
  },
  {
    key: 'source',
    kind: 'select',
    fieldLabelKey: 'audit.logList.builder.fields.source',
    placeholderKey: 'audit.logList.filters.sourcePlaceholder',
    options: sourceOptions,
  },
  {
    key: 'businessCategory',
    kind: 'select',
    fieldLabelKey: 'audit.logList.builder.fields.businessCategory',
    placeholderKey: 'audit.logList.filters.businessCategoryPlaceholder',
    options: businessCategoryOptions,
  },
  {
    key: 'actor',
    kind: 'text',
    fieldLabelKey: 'audit.logList.builder.fields.actor',
    placeholderKey: 'audit.logList.filters.actorPlaceholder',
  },
  {
    key: 'resourceName',
    kind: 'text',
    fieldLabelKey: 'audit.logList.builder.fields.resourceName',
    placeholderKey: 'audit.logList.filters.resourceNamePlaceholder',
  },
  {
    key: 'resourceType',
    kind: 'select',
    fieldLabelKey: 'audit.logList.builder.fields.resourceType',
    placeholderKey: 'audit.logList.filters.resourceTypePlaceholder',
    options: resourceTypeOptions,
  },
  {
    key: 'resourceTypes',
    kind: 'multi-select',
    fieldLabelKey: 'audit.logList.builder.fields.resourceTypes',
    placeholderKey: 'audit.logList.filters.resourceTypesPlaceholder',
    options: resourceTypeOptions,
  },
  {
    key: 'requestPathPrefixes',
    kind: 'tag-input',
    fieldLabelKey: 'audit.logList.builder.fields.requestPathPrefixes',
    placeholderKey: 'audit.logList.filters.requestPathPrefixesPlaceholder',
  },
  {
    key: 'requestId',
    kind: 'text',
    fieldLabelKey: 'audit.logList.builder.fields.requestId',
    placeholderKey: 'audit.logList.filters.requestIdPlaceholder',
  },
  {
    key: 'session',
    kind: 'text',
    fieldLabelKey: 'audit.logList.builder.fields.session',
    placeholderKey: 'audit.logList.filters.sessionPlaceholder',
  },
  {
    key: 'resourceId',
    kind: 'text',
    fieldLabelKey: 'audit.logList.builder.fields.resourceId',
    placeholderKey: 'audit.logList.filters.resourceIdPlaceholder',
  },
]);

const definitionMap = computed<Map<AuditFilterKey, AuditFilterDefinition>>(
  () => new Map(definitions.value.map((item) => [item.key, item])),
);
const selectedDefinition = computed(() => definitionMap.value.get(selectedDefinitionKey.value));
const activeSorterSelection = computed(() => useSingleSorterSelection(() => props.modelValue.sorters));
const activeSorter = computed(() => activeSorterSelection.value.sorter);
const sortFieldValue = computed(() => activeSorterSelection.value.field);
const sortDirectionValue = computed(() => activeSorterSelection.value.direction);
const selectedTimePreset = computed<TimePresetSelection>(() => {
  const [from, to] = props.modelValue.createdRange;
  if (!from || !to) {
    return 'custom';
  }

  const now = new Date();
  const presetRanges: AuditTimePreset[] = [
    AUDIT_TIME_PRESET.LAST_24H,
    AUDIT_TIME_PRESET.LAST_7D,
    AUDIT_TIME_PRESET.LAST_30D,
  ];

  for (const preset of presetRanges) {
    const hours = preset === AUDIT_TIME_PRESET.LAST_24H ? 24 : preset === AUDIT_TIME_PRESET.LAST_7D ? 24 * 7 : 24 * 30;
    const range = buildRecentHoursLocalRange(now, hours);
    if (range[0] === from && range[1] === to) {
      return preset;
    }
  }

  return 'custom';
});
const summaryText = computed(() => {
  const timeLabel =
    selectedTimePreset.value === 'custom'
      ? props.modelValue.createdRange.length
        ? t('audit.logList.builder.summary.customTime', { range: props.modelValue.createdRange.join(' ~ ') })
        : ''
      : (timePresetOptions.value.find((option) => option.value === selectedTimePreset.value)?.label ?? '');
  const resultLabel =
    props.modelValue.result !== 'all'
      ? t('audit.logList.builder.summary.result', {
          value:
            resultOptions.value.find((option) => option.value === props.modelValue.result)?.label ??
            props.modelValue.result,
        })
      : '';
  const sortLabel = activeSorter.value
    ? t('audit.logList.builder.summary.sortBy', {
        field:
          sortFieldOptions.value.find((option) => option.value === activeSorter.value?.field)?.label ??
          activeSorter.value.field,
        direction: activeSorter.value.direction === 'asc' ? t('audit.logList.sort.asc') : t('audit.logList.sort.desc'),
      })
    : '';

  return joinQuerySummary([timeLabel, resultLabel, sortLabel]);
});

const availableDefinitions = computed(() =>
  definitions.value.filter(
    (definition) =>
      (!isLocked(definition.key) && !isFieldActive(definition.key)) ||
      (!isLocked(definition.key) && definition.key === selectedDefinitionKey.value) ||
      activeFilterTags.value.some((tag) => tag.key === definition.key),
  ),
);

const activeFilterTags = computed(() => {
  const filterTags: FilterTag[] = [];

  definitions.value.forEach((definition) => {
    const label = buildTagLabel(definition);
    if (label) {
      filterTags.push({ key: definition.key, label });
    }
  });

  return prependSingleSorterTag(
    filterTags,
    activeSorter.value,
    sortFieldOptions.value,
    t('audit.logList.sort.tagPrefix'),
  );
});

const dateRangePlaceholder = computed(() => [
  t('audit.logList.filters.datePlaceholder'),
  t('audit.logList.filters.datePlaceholder'),
]);

function updateTimePreset(value: string | number | Array<string | number> | undefined) {
  if (typeof value !== 'string') {
    return;
  }

  if (value === 'custom') {
    return;
  }

  const hours = value === AUDIT_TIME_PRESET.LAST_24H ? 24 : value === AUDIT_TIME_PRESET.LAST_7D ? 24 * 7 : 24 * 30;
  updateField('createdRange', buildRecentHoursLocalRange(new Date(), hours));
}

function updateField<Key extends keyof AuditClientFilterState>(key: Key, value: AuditClientFilterState[Key]) {
  if (isLocked(key as AuditFilterKey)) {
    return;
  }
  emit('update:modelValue', {
    ...props.modelValue,
    [key]: value,
  });
}

function selectDefinition(key: AuditFilterKey) {
  if (isLocked(key)) {
    return;
  }
  selectedDefinitionKey.value = key;
}

function isLocked(key: AuditFilterKey | 'sorter') {
  if (key === 'sorter') {
    return false;
  }
  return props.lockedFields?.includes(key) ?? false;
}

function isFieldActive(key: AuditFilterKey) {
  const value = props.modelValue[key];
  if (key === 'result' || key === 'riskLevel') {
    return value !== 'all';
  }
  if (key === 'success') {
    return value !== 'all';
  }
  if (Array.isArray(value)) {
    return value.length > 0;
  }
  return typeof value === 'string' ? Boolean(value.trim()) : Boolean(value);
}

function buildTagLabel(definition: AuditFilterDefinition) {
  const value = props.modelValue[definition.key];
  if (definition.kind === 'multi-select') {
    if (!Array.isArray(value) || value.length === 0) {
      return '';
    }
    return `${t(definition.fieldLabelKey)}：${value.map((item) => optionLabel(definition.options.value, String(item))).join('、')}`;
  }

  if (definition.kind === 'tag-input') {
    if (!Array.isArray(value) || value.length === 0) {
      return '';
    }
    return `${t(definition.fieldLabelKey)}：${value.join('、')}`;
  }

  if (definition.key === 'result' || definition.key === 'riskLevel') {
    if (value === 'all') {
      return '';
    }
  } else if (definition.key === 'success') {
    if (value === 'all') {
      return '';
    }
  } else if (typeof value === 'string' && !value.trim()) {
    return '';
  }

  if (definition.kind !== 'select') {
    return '';
  }

  const display = optionLabel(definition.options.value, String(value));
  return `${t(definition.fieldLabelKey)}：${display}`;
}

function optionLabel(options: AuditFilterOption[], value: string) {
  return options.find((option) => option.value === value)?.label || value;
}

function clearField(key: AuditFilterKey) {
  if (isLocked(key)) {
    return;
  }
  if (key === 'result') {
    updateField(key, 'all' as AuditClientFilterState[typeof key]);
    return;
  }
  if (key === 'riskLevel') {
    updateField(key, 'all' as AuditClientFilterState[typeof key]);
    return;
  }
  if (key === 'success') {
    updateField(key, 'all' as AuditClientFilterState[typeof key]);
    return;
  }
  if (
    key === 'actionPrefixes' ||
    key === 'actionKeywords' ||
    key === 'requestPathPrefixes' ||
    key === 'resourceTypes' ||
    key === 'results' ||
    key === 'riskLevels'
  ) {
    updateField(key, [] as AuditClientFilterState[typeof key]);
    return;
  }
  updateField(key, '' as AuditClientFilterState[typeof key]);
}

function clearSorter() {
  emit('update:modelValue', {
    ...props.modelValue,
    sorters: [],
  });
}

function textValue(key: AuditTextFilterKey) {
  return props.modelValue[key];
}

function selectValue(key: AuditSingleSelectFilterKey) {
  const value = props.modelValue[key];
  return value === 'all' ? '' : value;
}

function multiSelectValue(key: AuditMultiSelectFilterKey) {
  return props.modelValue[key];
}

function tagInputValue(key: AuditTagInputFilterKey) {
  return props.modelValue[key];
}

function normalizeTextValue(value: string | number | undefined) {
  return typeof value === 'string' ? value : '';
}

function normalizeSelectValue(value: string | number | Array<string | number> | undefined) {
  return typeof value === 'string' ? value : '';
}

function normalizeStringArray(value: Array<string | number> | undefined) {
  if (!Array.isArray(value)) {
    return [];
  }

  return value.map((item) => String(item).trim()).filter(Boolean);
}

function normalizeMultiSelectValue(value: string | number | Array<string | number> | undefined) {
  return Array.isArray(value) ? normalizeStringArray(value) : [];
}

function normalizeTagInputValue(value: Array<string | number> | undefined) {
  return normalizeStringArray(value);
}

function updateSortField(value: string | number | Array<string | number> | undefined) {
  emit('update:modelValue', {
    ...props.modelValue,
    sorters: normalizeSingleSorterField(value, activeSorter.value?.direction, normalizeSortField),
  });
}

function updateSortDirection(value: string | number | Array<string | number> | undefined) {
  emit('update:modelValue', {
    ...props.modelValue,
    sorters: normalizeSingleSorterDirection(value, activeSorter.value?.field, normalizeSortDirection),
  });
}

function normalizeSortField(value: string): AuditSortBy {
  return value === 'created_at' ? 'created_at' : 'created_at';
}

function normalizeSortDirection(value: string): AuditSortOrder {
  return value === 'asc' ? 'asc' : 'desc';
}
</script>
<style scoped lang="less">
.audit-filters {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
}

.audit-filters__top-row,
.audit-filters__group-body,
.audit-filters__preset-row {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.audit-filters__keyword {
  flex: 1 1 340px;
  min-width: 240px;
}

.audit-filters__date {
  flex: 0 1 360px;
  min-width: 240px;
}

.audit-filters__actions {
  display: flex;
  gap: 12px;
  margin-left: auto;
}

.audit-filters__summary {
  color: var(--td-text-color-secondary);
  font-size: 13px;
}

.audit-filters__group {
  background-color: var(--td-bg-color-container);
  border-color: var(--td-component-border);
  border-radius: var(--td-radius-large);
  border-style: solid;
  border-width: 1px;
  padding: 12px;
  padding-inline: 14px;
}

.audit-filters__group-header {
  margin-bottom: 10px;
}

.audit-filters__group-title {
  color: var(--td-text-color-primary);
  font-size: 13px;
  font-weight: 600;
}

.audit-filters__group-body--builder {
  justify-content: flex-start;
}

.audit-filters__time-preset {
  min-width: 180px;
}

.audit-filters__tag-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.audit-filters__preset-row {
  flex: 1 1 auto;
  min-width: 0;
}

.audit-filters__sort-select {
  min-width: 160px;
}

.audit-filters__preset-label {
  color: var(--td-text-color-secondary);
  font-size: 12px;
  white-space: nowrap;
}

.audit-filter-builder {
  display: grid;
  gap: 16px;
  grid-template-columns: minmax(160px, 200px) minmax(220px, 320px);
  padding: 8px;
}

.audit-filter-builder__header {
  display: flex;
  flex-direction: column;
  gap: 4px;
  grid-column: 1 / -1;
}

.audit-filter-builder__title {
  color: var(--td-text-color-primary);
  font-size: 14px;
  font-weight: 600;
}

.audit-filter-builder__hint {
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.audit-filter-builder__field-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.audit-filter-builder__field-button {
  background: var(--td-bg-color-container-hover);
  border: 1px solid transparent;
  border-radius: var(--td-radius-default);
  color: var(--td-text-color-primary);
  cursor: pointer;
  font: inherit;
  padding: 8px 12px;
  text-align: left;
  transition:
    border-color 0.2s ease,
    background-color 0.2s ease;
}

.audit-filter-builder__field-button:hover {
  border-color: var(--td-brand-color);
}

.audit-filter-builder__editor {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.audit-filter-builder__editor-title {
  color: var(--td-text-color-primary);
  font-size: 13px;
  font-weight: 600;
}

@media (width <= 960px) {
  .audit-filters__actions {
    margin-left: 0;
  }

  .audit-filter-builder {
    grid-template-columns: 1fr;
  }
}

@media (width <= 768px) {
  .audit-filters__top-row,
  .audit-filters__group-body,
  .audit-filters__preset-row {
    align-items: stretch;
    flex-direction: column;
  }

  .audit-filters__keyword,
  .audit-filters__date,
  .audit-filters__actions,
  .audit-filters__sort-row,
  .audit-filters__preset-row {
    min-width: 0;
    width: 100%;
  }

  .audit-filters__actions {
    justify-content: flex-start;
  }
}
</style>
