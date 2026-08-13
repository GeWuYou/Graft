<template>
  <advanced-query-filter-builder-frame :frame="builderFrame" message-prefix="scheduledTask.list">
    <template #saved-query-views>
      <saved-query-view-control v-if="savedViewController" :controller="savedViewController" />
    </template>
  </advanced-query-filter-builder-frame>
</template>
<script setup lang="ts">
// 定时任务筛选器只维护列表查询草稿，列表页负责请求、保存视图和分页生命周期。
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import {
  AdvancedQueryFilterBuilderFrame,
  type AdvancedQueryFilterFieldDefinition,
  type AdvancedQueryFilterTag,
  SavedQueryViewControl,
  type SavedQueryViewController,
  type SavedQueryViewId,
} from '@/shared/components/query-list';
import * as BuilderHelpers from '@/shared/components/query-list';

import type { ScheduledTaskJobKey } from '../types/scheduled-task';

type FilterState = {
  keyword: string;
  jobKey: ScheduledTaskJobKey | 'all';
  status: 'enabled' | 'disabled' | 'all';
};
type FilterKey = Exclude<keyof FilterState, 'keyword'>;

const props = defineProps<{
  loading?: boolean;
  modelValue: FilterState;
  jobDefinitions: Array<{ job_key: ScheduledTaskJobKey; title: string; module_key: string }>;
  savedViewController?: SavedQueryViewController<unknown, SavedQueryViewId>;
}>();

const emit = defineEmits<{
  (event: 'reset'): void;
  (event: 'search'): void;
  (event: 'update:modelValue', value: FilterState): void;
}>();

const { t } = useI18n();
const selectedFieldKey = ref<FilterKey>('status');

const fields = computed<AdvancedQueryFilterFieldDefinition[]>(() => [
  {
    key: 'jobKey',
    kind: 'select',
    label: t('scheduledTask.list.filters.job'),
    options: [
      { label: t('scheduledTask.list.filters.allJobs'), value: 'all' },
      ...props.jobDefinitions.map((job) => ({ label: job.title || job.job_key, value: job.job_key })),
    ],
    placeholder: t('scheduledTask.list.filters.job'),
  },
  {
    key: 'status',
    kind: 'select',
    label: t('scheduledTask.list.filters.status'),
    options: [
      { label: t('scheduledTask.list.filters.allStatuses'), value: 'all' },
      ...(['enabled', 'disabled'] as const).map((value) => ({
        label: t(`scheduledTask.list.statusLabels.${value}`),
        value,
      })),
    ],
    placeholder: t('scheduledTask.list.filters.status'),
  },
]);

const fieldValues = computed<Record<string, string>>(() => ({
  jobKey: props.modelValue.jobKey,
  status: props.modelValue.status,
}));

const tags = computed<AdvancedQueryFilterTag[]>(() =>
  BuilderHelpers.buildAdvancedQueryActiveTags<FilterState, FilterKey, string>({
    fields: fields.value,
    filterState: props.modelValue,
    sortOptions: [],
    sorterPrefix: '',
    sorters: [],
  }),
);

/* jscpd:ignore-start */
const builderFrame = BuilderHelpers.createAdvancedQueryFilterBuilderFrameStateFromSource({
  fieldValues: () => fieldValues.value,
  fields: () => fields.value,
  keyword: () => props.modelValue.keyword,
  listeners: BuilderHelpers.createAdvancedQueryBuilderListeners({
    addSorter: () => undefined,
    clearTag,
    emitApplyPreset: () => undefined,
    emitReset: () => emit('reset'),
    emitSearch: () => emit('search'),
    handleFieldUpdate: ({ key, value }) => updateField(key as FilterKey, value as FilterState[FilterKey]),
    moveSorterDown: () => undefined,
    moveSorterUp: () => undefined,
    removeSorter: () => undefined,
    selectedFieldKey,
    updateKeyword: (value) => emit('update:modelValue', { ...props.modelValue, keyword: value }),
    updateSortDirection: () => undefined,
    updateSortField: () => undefined,
    updateTimeField: () => undefined,
  }),
  selectedFieldKey,
  sorterUi: BuilderHelpers.useAdvancedQuerySorterUiState(
    () => [],
    () => [],
  ),
  sortDirectionOptions: () => [],
  source: () => ({ activePreset: '', loading: props.loading, presets: [], showSorterBuilder: false }),
  tags: () => tags.value,
  timeFields: () => [],
});
/* jscpd:ignore-end */

function updateField(key: FilterKey, value: FilterState[FilterKey]) {
  emit('update:modelValue', BuilderHelpers.updateAdvancedQueryFilterStateField(props.modelValue, key, value));
}

function clearTag(key: string) {
  updateField(key as FilterKey, 'all');
}
</script>
