<template>
  <advanced-query-filter-builder
    :active-preset="frame.activePreset"
    :add-filter-label="`+ ${t(`${messagePrefix}.actions.addFilter`)}`"
    :add-sorter-label="t(`${messagePrefix}.actions.addSorter`)"
    :builder-hint="t(`${messagePrefix}.builder.hint`)"
    :builder-title="t(`${messagePrefix}.builder.title`)"
    :field-values="frame.fieldValues"
    :fields="frame.fields"
    :filters-group-label="t(`${messagePrefix}.builder.groups.filters`)"
    :keyword="frame.keyword"
    :keyword-placeholder="t(`${messagePrefix}.page.searchPlaceholder`)"
    :loading="frame.loading"
    :move-down-label="t(`${messagePrefix}.actions.moveSorterDown`)"
    :move-up-label="t(`${messagePrefix}.actions.moveSorterUp`)"
    :preset-label="t(`${messagePrefix}.presets.label`)"
    :presets="frame.presets"
    :remove-sorter-label="t(`${messagePrefix}.actions.removeSorter`)"
    :reset-label="t(`${messagePrefix}.actions.reset`)"
    :search-label="t(`${messagePrefix}.actions.search`)"
    :selected-field-key="frame.selectedFieldKey"
    :sort-add-disabled="frame.sortAddDisabled"
    :sort-direction-options="frame.sortDirectionOptions"
    :sort-direction-placeholder="t(`${messagePrefix}.sort.directionPlaceholder`)"
    :sort-field-key="frame.sortFieldKey"
    :sort-field-options-by-index="frame.sortFieldOptionsByIndex"
    :sort-field-placeholder="t(`${messagePrefix}.sort.fieldPlaceholder`)"
    :sort-move-down-disabled="frame.sortMoveDownDisabled"
    :sort-move-up-disabled="frame.sortMoveUpDisabled"
    :sorters="frame.sorters"
    :tags="frame.tags"
    :time-field-key="frame.timeFieldKey"
    :time-fields="frame.timeFields"
    v-on="frame.listeners"
  />
</template>
<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import type { QuerySorter, SortOption } from '@/shared/observability/sorters';

import AdvancedQueryFilterBuilder from './AdvancedQueryFilterBuilder.vue';
import type {
  AdvancedQueryFilterFieldDefinition,
  AdvancedQueryFilterPreset,
  AdvancedQueryFilterTag,
  AdvancedQueryTimeRangeField,
} from './query-filter-builder';

export type AdvancedQueryFilterBuilderFrameState = {
  activePreset: string;
  fieldValues: Record<string, string | string[]>;
  fields: AdvancedQueryFilterFieldDefinition[];
  keyword: string;
  listeners: Record<string, (...args: never[]) => void>;
  loading?: boolean;
  presets: AdvancedQueryFilterPreset[];
  selectedFieldKey: string;
  sortAddDisabled: boolean;
  sortDirectionOptions: Array<{ label: string; value: string }>;
  sortFieldKey: string;
  sortFieldOptionsByIndex: Array<Array<SortOption<string>>>;
  sortMoveDownDisabled: boolean[];
  sortMoveUpDisabled: boolean[];
  sorters: QuerySorter<string>[];
  tags: AdvancedQueryFilterTag[];
  timeFieldKey: string;
  timeFields: AdvancedQueryTimeRangeField[];
};

defineProps<{
  frame: AdvancedQueryFilterBuilderFrameState;
  messagePrefix: string;
}>();

const { t } = useI18n();
</script>
