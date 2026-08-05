<template>
  <advanced-query-filter-builder-frame :frame="builderFrame" message-prefix="announcement.management">
    <template #saved-query-views>
      <saved-query-view-control v-if="savedViewController" :controller="savedViewController" />
    </template>
  </advanced-query-filter-builder-frame>
</template>
<script setup lang="ts">
// 公告筛选器只维护管理列表的查询草稿，列表页负责请求、保存视图和生命周期操作。
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

import type {
  AnnouncementFilterState,
  AnnouncementLevel,
  AnnouncementPinnedFilter,
  AnnouncementSort,
  AnnouncementStatus,
} from '../types/announcement';

type FilterKey = Exclude<keyof AnnouncementFilterState, 'keyword'>;

const props = defineProps<{
  loading?: boolean;
  modelValue: AnnouncementFilterState;
  savedViewController?: SavedQueryViewController<unknown, SavedQueryViewId>;
}>();

const emit = defineEmits<{
  (event: 'reset'): void;
  (event: 'search'): void;
  (event: 'update:modelValue', value: AnnouncementFilterState): void;
}>();

const { t } = useI18n();
const selectedFieldKey = ref<FilterKey>('status');

const fields = computed<AdvancedQueryFilterFieldDefinition[]>(() => [
  {
    key: 'status',
    kind: 'select',
    label: t('announcement.management.filters.status'),
    options: (['draft', 'published', 'archived'] satisfies AnnouncementStatus[]).map((value) => ({
      label: t(`announcement.status.${value}`),
      value,
    })),
    placeholder: t('announcement.management.filters.status'),
  },
  {
    key: 'level',
    kind: 'select',
    label: t('announcement.management.filters.level'),
    options: (['info', 'warning', 'success', 'error'] satisfies AnnouncementLevel[]).map((value) => ({
      label: t(`announcement.level.${value}`),
      value,
    })),
    placeholder: t('announcement.management.filters.level'),
  },
  {
    key: 'pinned',
    kind: 'select',
    label: t('announcement.management.filters.pinned'),
    options: [
      { label: t('announcement.pinned.yes'), value: 'true' },
      { label: t('announcement.pinned.no'), value: 'false' },
    ],
    placeholder: t('announcement.management.filters.pinned'),
  },
  {
    key: 'sort',
    kind: 'select',
    label: t('announcement.management.filters.sort'),
    options: [
      { label: t('announcement.management.sort.updatedDesc'), value: 'updated_desc' },
      { label: t('announcement.management.sort.publishDesc'), value: 'publish_desc' },
      { label: t('announcement.management.sort.pinnedPublishDesc'), value: 'pinned_publish_desc' },
    ],
    placeholder: t('announcement.management.filters.sort'),
  },
]);

const fieldValues = computed<Record<string, string>>(() => ({
  status: props.modelValue.status,
  level: props.modelValue.level,
  pinned: props.modelValue.pinned,
  sort: props.modelValue.sort,
}));

const tags = computed<AdvancedQueryFilterTag[]>(() =>
  BuilderHelpers.buildAdvancedQueryActiveTags<AnnouncementFilterState, FilterKey, string>({
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
    handleFieldUpdate: ({ key, value }) => updateField(key as FilterKey, value as AnnouncementFilterState[FilterKey]),
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

function updateField(key: FilterKey, value: AnnouncementFilterState[FilterKey]) {
  emit('update:modelValue', BuilderHelpers.updateAdvancedQueryFilterStateField(props.modelValue, key, value));
}

function clearTag(key: string) {
  if (key === 'sort') {
    updateField('sort', 'updated_desc' as AnnouncementSort);
    return;
  }
  updateField(
    key as Exclude<FilterKey, 'sort'>,
    '' as AnnouncementStatus | AnnouncementLevel | AnnouncementPinnedFilter,
  );
}
</script>
