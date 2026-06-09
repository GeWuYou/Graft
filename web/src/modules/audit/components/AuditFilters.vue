<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <advanced-query-filter-builder
    :active-preset="activePreset"
    :add-filter-label="`+ ${t('audit.logList.actions.addFilter')}`"
    :add-sorter-label="t('audit.logList.actions.addSorter')"
    :builder-hint="t('audit.logList.builder.hint')"
    :builder-title="t('audit.logList.builder.title')"
    :field-values="fieldValues"
    :fields="definitions"
    :filters-group-label="t('audit.logList.builder.groups.filters')"
    :keyword="modelValue.keyword"
    :keyword-placeholder="t('audit.logList.filters.keywordPlaceholder')"
    :loading="loading"
    :move-down-label="t('audit.logList.actions.moveSorterDown')"
    :move-up-label="t('audit.logList.actions.moveSorterUp')"
    :preset-label="t('audit.logList.presets.label')"
    :presets="presets"
    :remove-sorter-label="t('audit.logList.actions.removeSorter')"
    :reset-label="t('audit.logList.actions.reset')"
    :search-label="t('audit.logList.actions.search')"
    :selected-field-key="selectedFieldKey"
    :sort-add-disabled="sortAddDisabled"
    :sort-direction-options="sortDirectionOptions"
    :sort-direction-placeholder="t('audit.logList.sort.directionPlaceholder')"
    :sort-field-key="'sorterBuilder'"
    :sort-field-options-by-index="sortFieldOptionsByIndex"
    :sort-field-placeholder="t('audit.logList.sort.fieldPlaceholder')"
    :sort-move-down-disabled="sortMoveDownDisabled"
    :sort-move-up-disabled="sortMoveUpDisabled"
    :sorters="normalizedSorters"
    :tags="activeFilterTags"
    :time-field-key="'timeRange'"
    :time-fields="timeFields"
    v-on="builderListeners"
  />
</template>
<script setup lang="ts">
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import {
  AdvancedQueryFilterBuilder,
  type AdvancedQueryFilterFieldDefinition,
  type AdvancedQueryFilterOption,
  type AdvancedQueryFilterTag,
  type AdvancedQueryTimeRangeField,
  createAdvancedQueryBuilderListeners,
  createSortDirection,
  useAdvancedQuerySorterUiState,
} from '@/shared/components/query-list';
import {
  appendSorterToState,
  buildRecentHoursLocalRange,
  moveSorterInState,
  prependSorterTags,
  removeSorterFromState,
  withSorterDirectionFromInput,
  withSorterFieldFromInput,
} from '@/shared/observability';

import type { AuditQuickPresetKey } from '../contract/presets';
import { AUDIT_TIME_PRESET } from '../contract/time-presets';
import type {
  AuditFilterKey,
  AuditFilterOption as ModuleAuditFilterOption,
  AuditMultiSelectFilterKey,
  AuditSingleSelectFilterKey,
  AuditTagInputFilterKey,
  AuditTextFilterKey,
} from '../shared/filter-definitions';
import type { AuditClientFilterState } from '../shared/presentation';
import type { AuditSortBy, AuditSortOrder } from '../types/audit';

type FilterTagKey = AuditFilterKey | 'createdRange' | `sorter:${number}`;
type BuilderFieldKey = 'timeRange' | 'sorterBuilder' | AuditFilterKey;

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

const selectedFieldKey = ref<BuilderFieldKey>('timeRange');

const actionOptions = computed<ModuleAuditFilterOption[]>(() => [
  { label: t('audit.logList.filterOptions.auth'), value: 'auth' },
  { label: t('audit.logList.filterOptions.role'), value: 'role' },
  { label: t('audit.logList.filterOptions.permission'), value: 'permission' },
  { label: t('audit.logList.filterOptions.session'), value: 'session' },
]);
const actionPrefixOptions = computed<ModuleAuditFilterOption[]>(() => [
  { label: t('audit.logList.filterOptions.authPrefix'), value: 'auth.' },
  { label: t('audit.logList.filterOptions.rbacPrefix'), value: 'rbac.' },
  { label: t('audit.logList.filterOptions.rolePrefix'), value: 'role.' },
  { label: t('audit.logList.filterOptions.permissionPrefix'), value: 'permission.' },
]);
const sourceOptions = computed<ModuleAuditFilterOption[]>(() => [
  { label: t('audit.common.source.REQUEST'), value: 'REQUEST' },
  { label: t('audit.common.source.SECURITY_EVENT'), value: 'SECURITY_EVENT' },
  { label: t('audit.common.source.DOMAIN_EVENT'), value: 'DOMAIN_EVENT' },
]);
const businessCategoryOptions = computed<ModuleAuditFilterOption[]>(() => [
  { label: t('audit.logList.businessCategory.failedOperations'), value: 'failed_operations' },
  { label: t('audit.logList.businessCategory.highRiskOperations'), value: 'high_risk_operations' },
  { label: t('audit.logList.businessCategory.sensitiveOperations'), value: 'sensitive_operations' },
  { label: t('audit.logList.businessCategory.authFailures'), value: 'auth_failures' },
  { label: t('audit.logList.businessCategory.permissionDenials'), value: 'permission_denials' },
  { label: t('audit.logList.businessCategory.rbacChanges'), value: 'rbac_changes' },
  { label: t('audit.logList.businessCategory.criticalSecurity'), value: 'critical_security' },
]);
const resourceTypeOptions = computed<ModuleAuditFilterOption[]>(() => [
  { label: t('audit.logList.filterOptions.userResource'), value: 'user' },
  { label: t('audit.logList.filterOptions.roleResource'), value: 'role' },
  { label: t('audit.logList.filterOptions.permissionResource'), value: 'permission' },
  { label: t('audit.logList.filterOptions.authResource'), value: 'auth' },
]);
const resultOptions = computed<ModuleAuditFilterOption[]>(() => [
  { label: t('audit.logList.filterOptions.SUCCESS'), value: 'SUCCESS' },
  { label: t('audit.logList.filterOptions.FAILED'), value: 'FAILED' },
  { label: t('audit.logList.filterOptions.DENIED'), value: 'DENIED' },
  { label: t('audit.logList.filterOptions.ERROR'), value: 'ERROR' },
]);
const successOptions = computed<ModuleAuditFilterOption[]>(() => [
  { label: t('audit.logList.filterOptions.SUCCESS'), value: 'true' },
  { label: t('audit.logList.filterOptions.FAILED'), value: 'false' },
]);
const riskOptions = computed<ModuleAuditFilterOption[]>(() => [
  { label: t('audit.logList.filterOptions.LOW'), value: 'LOW' },
  { label: t('audit.logList.filterOptions.MEDIUM'), value: 'MEDIUM' },
  { label: t('audit.logList.filterOptions.HIGH'), value: 'HIGH' },
  { label: t('audit.logList.filterOptions.CRITICAL'), value: 'CRITICAL' },
]);
const sortFieldOptions = computed<AdvancedQueryFilterOption[]>(() => [
  { label: t('audit.logList.sort.createdAt'), value: 'created_at' },
]);
const sortDirectionOptions = computed<AdvancedQueryFilterOption[]>(() => [
  { label: t('audit.logList.sort.desc'), value: 'desc' },
  { label: t('audit.logList.sort.asc'), value: 'asc' },
]);
const { normalizedSorters, sortFieldOptionsByIndex, sortAddDisabled, sortMoveUpDisabled, sortMoveDownDisabled } =
  useAdvancedQuerySorterUiState(
    () => props.modelValue.sorters,
    () => sortFieldOptions.value,
  );

const definitions = computed<AdvancedQueryFilterFieldDefinition[]>(() => [
  { key: 'timeRange', kind: 'special', label: t('audit.logList.builder.fields.timeRange') },
  { key: 'sorterBuilder', kind: 'special', label: t('audit.logList.builder.fields.sorterBuilder') },
  {
    key: 'action',
    kind: 'select',
    label: t('audit.logList.builder.fields.action'),
    placeholder: t('audit.logList.filters.actionPlaceholder'),
    options: actionOptions.value,
  },
  {
    key: 'actionPrefixes',
    kind: 'multi-select',
    label: t('audit.logList.builder.fields.actionPrefixes'),
    placeholder: t('audit.logList.filters.actionPrefixesPlaceholder'),
    options: actionPrefixOptions.value,
  },
  {
    key: 'actionKeywords',
    kind: 'tag-input',
    label: t('audit.logList.builder.fields.actionKeywords'),
    placeholder: t('audit.logList.filters.actionKeywordsPlaceholder'),
  },
  {
    key: 'result',
    kind: 'select',
    label: t('audit.logList.builder.fields.result'),
    placeholder: t('audit.logList.filters.resultPlaceholder'),
    options: resultOptions.value,
  },
  {
    key: 'results',
    kind: 'multi-select',
    label: t('audit.logList.builder.fields.results'),
    placeholder: t('audit.logList.filters.resultsPlaceholder'),
    options: resultOptions.value,
  },
  {
    key: 'riskLevel',
    kind: 'select',
    label: t('audit.logList.builder.fields.riskLevel'),
    placeholder: t('audit.logList.filters.riskPlaceholder'),
    options: riskOptions.value,
  },
  {
    key: 'riskLevels',
    kind: 'multi-select',
    label: t('audit.logList.builder.fields.riskLevels'),
    placeholder: t('audit.logList.filters.riskLevelsPlaceholder'),
    options: riskOptions.value,
  },
  {
    key: 'success',
    kind: 'select',
    label: t('audit.logList.builder.fields.success'),
    placeholder: t('audit.logList.filters.successPlaceholder'),
    options: successOptions.value,
  },
  {
    key: 'source',
    kind: 'select',
    label: t('audit.logList.builder.fields.source'),
    placeholder: t('audit.logList.filters.sourcePlaceholder'),
    options: sourceOptions.value,
  },
  {
    key: 'businessCategory',
    kind: 'select',
    label: t('audit.logList.builder.fields.businessCategory'),
    placeholder: t('audit.logList.filters.businessCategoryPlaceholder'),
    options: businessCategoryOptions.value,
  },
  {
    key: 'actor',
    kind: 'text',
    label: t('audit.logList.builder.fields.actor'),
    placeholder: t('audit.logList.filters.actorPlaceholder'),
  },
  {
    key: 'resourceName',
    kind: 'text',
    label: t('audit.logList.builder.fields.resourceName'),
    placeholder: t('audit.logList.filters.resourceNamePlaceholder'),
  },
  {
    key: 'resourceType',
    kind: 'select',
    label: t('audit.logList.builder.fields.resourceType'),
    placeholder: t('audit.logList.filters.resourceTypePlaceholder'),
    options: resourceTypeOptions.value,
  },
  {
    key: 'resourceTypes',
    kind: 'multi-select',
    label: t('audit.logList.builder.fields.resourceTypes'),
    placeholder: t('audit.logList.filters.resourceTypesPlaceholder'),
    options: resourceTypeOptions.value,
  },
  {
    key: 'requestPathPrefixes',
    kind: 'tag-input',
    label: t('audit.logList.builder.fields.requestPathPrefixes'),
    placeholder: t('audit.logList.filters.requestPathPrefixesPlaceholder'),
  },
  {
    key: 'requestId',
    kind: 'text',
    label: t('audit.logList.builder.fields.requestId'),
    placeholder: t('audit.logList.filters.requestIdPlaceholder'),
  },
  {
    key: 'session',
    kind: 'text',
    label: t('audit.logList.builder.fields.session'),
    placeholder: t('audit.logList.filters.sessionPlaceholder'),
  },
  {
    key: 'resourceId',
    kind: 'text',
    label: t('audit.logList.builder.fields.resourceId'),
    placeholder: t('audit.logList.filters.resourceIdPlaceholder'),
  },
]);

const fieldValues = computed<Record<string, string | string[]>>(() => ({
  action: props.modelValue.action,
  actionPrefixes: props.modelValue.actionPrefixes,
  actionKeywords: props.modelValue.actionKeywords,
  result: props.modelValue.result === 'all' ? '' : props.modelValue.result,
  results: props.modelValue.results,
  riskLevel: props.modelValue.riskLevel === 'all' ? '' : String(props.modelValue.riskLevel),
  riskLevels: props.modelValue.riskLevels,
  success: props.modelValue.success === 'all' ? '' : props.modelValue.success,
  source: props.modelValue.source,
  businessCategory: props.modelValue.businessCategory,
  actor: props.modelValue.actor,
  resourceName: props.modelValue.resourceName,
  resourceType: props.modelValue.resourceType,
  resourceTypes: props.modelValue.resourceTypes,
  requestPathPrefixes: props.modelValue.requestPathPrefixes,
  requestId: props.modelValue.requestId,
  session: props.modelValue.session,
  resourceId: props.modelValue.resourceId,
}));

const timeFields = computed<AdvancedQueryTimeRangeField[]>(() => [
  {
    key: 'createdRange',
    label: t('audit.logList.sort.createdAt'),
    value: props.modelValue.createdRange,
    placeholder: [t('audit.logList.filters.datePlaceholder'), t('audit.logList.filters.datePlaceholder')],
  },
]);

const builderListeners = createAuditBuilderListeners();

const activeFilterTags = computed<AdvancedQueryFilterTag[]>(() => {
  const filterTags: AdvancedQueryFilterTag[] = [];

  definitions.value
    .filter((definition) => definition.kind !== 'special')
    .forEach((definition) => {
      const label = buildTagLabel(definition);
      if (label) {
        filterTags.push({
          key: definition.key,
          label,
          closable: !isLocked(definition.key as AuditFilterKey),
        });
      }
    });

  if (props.modelValue.createdRange.length) {
    filterTags.unshift({
      key: 'createdRange',
      label: `${t('audit.logList.sort.createdAt')}：${props.modelValue.createdRange.join(' ~ ')}`,
    });
  }

  return prependSorterTags(
    filterTags,
    normalizedSorters.value,
    sortFieldOptions.value,
    t('audit.logList.sort.tagPrefix'),
  ).map(
    (tag): AdvancedQueryFilterTag => ({
      ...tag,
      closable:
        tag.key === 'createdRange' || tag.key.startsWith('sorter:') ? true : !isLocked(tag.key as AuditFilterKey),
    }),
  );
});

function updateField<Key extends keyof AuditClientFilterState>(key: Key, value: AuditClientFilterState[Key]) {
  if (isLocked(key as AuditFilterKey)) {
    return;
  }
  emit('update:modelValue', {
    ...props.modelValue,
    [key]: value,
  });
}

function handleFieldUpdate(payload: { key: string; value: string | string[] }) {
  const key = payload.key as keyof AuditClientFilterState;
  if (key === 'result' || key === 'riskLevel' || key === 'success') {
    updateField(key, ((payload.value as string) || 'all') as never);
    return;
  }
  updateField(key, payload.value as never);
}

function createAuditBuilderListeners() {
  return createAdvancedQueryBuilderListeners<AuditQuickPresetKey, BuilderFieldKey, { key: string; value: string[] }>({
    selectedFieldKey,
    updateSortDirection,
    handleFieldUpdate,
    emitApplyPreset: (preset) => emit('apply-preset', preset),
    updateKeyword: (value) => updateField('keyword', value),
    updateSortField,
    emitSearch: () => emit('search'),
    moveSorterUp,
    clearTag: (key) => closeTag(key as FilterTagKey),
    addSorter,
    updateTimeField: ({ value }) => updateField('createdRange', value),
    removeSorter,
    emitReset: () => emit('reset'),
    moveSorterDown,
  });
}

function isLocked(key: AuditFilterKey) {
  return props.lockedFields?.includes(key) ?? false;
}

function buildTagLabel(definition: AdvancedQueryFilterFieldDefinition) {
  const value = fieldValues.value[definition.key];
  if (Array.isArray(value)) {
    if (!value.length) {
      return '';
    }
    return `${definition.label}：${value.map((item) => optionLabel(definition.options, String(item))).join('、')}`;
  }

  if (!value) {
    return '';
  }

  return `${definition.label}：${optionLabel(definition.options, String(value))}`;
}

function optionLabel(options: AdvancedQueryFilterOption[] | undefined, value: string) {
  return options?.find((option) => option.value === value)?.label || value;
}

function clearField(key: AuditFilterKey) {
  if (isLocked(key)) {
    return;
  }
  if (key === 'result' || key === 'riskLevel' || key === 'success') {
    updateField(key, 'all' as never);
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
    updateField(key, [] as never);
    return;
  }
  updateField(key, '' as never);
}

function closeTag(key: FilterTagKey) {
  if (key === 'createdRange') {
    emit('update:modelValue', { ...props.modelValue, createdRange: [] });
    return;
  }

  if (key.startsWith('sorter:')) {
    removeSorter(Number(key.split(':')[1] || 0));
    return;
  }

  clearField(key as AuditFilterKey);
}

function addSorter() {
  emit('update:modelValue', appendSorterToState(props.modelValue, sortFieldOptions.value));
}

function removeSorter(index: number) {
  emit('update:modelValue', removeSorterFromState(props.modelValue, index, sortFieldOptions.value));
}

function moveSorterUp(index: number) {
  emit('update:modelValue', moveSorterInState(props.modelValue, index, -1, sortFieldOptions.value));
}

function moveSorterDown(index: number) {
  emit('update:modelValue', moveSorterInState(props.modelValue, index, 1, sortFieldOptions.value));
}

function normalizeSortField(value: string): AuditSortBy | '' {
  return value === 'created_at' ? 'created_at' : '';
}

function normalizeSortDirection(value: string): AuditSortOrder {
  return createSortDirection(value);
}

function updateSortField(index: number, value: string | number | Array<string | number> | undefined) {
  emit(
    'update:modelValue',
    withSorterFieldFromInput(props.modelValue, index, value, normalizeSortField, sortFieldOptions.value, 'desc'),
  );
}

function updateSortDirection(index: number, value: string | number | Array<string | number> | undefined) {
  emit(
    'update:modelValue',
    withSorterDirectionFromInput(props.modelValue, index, value, normalizeSortDirection, sortFieldOptions.value),
  );
}

void AUDIT_TIME_PRESET;
void (null as unknown as AuditTextFilterKey);
void (null as unknown as AuditSingleSelectFilterKey);
void (null as unknown as AuditMultiSelectFilterKey);
void (null as unknown as AuditTagInputFilterKey);
void buildRecentHoursLocalRange;
</script>
