<template>
  <t-dialog
    :cancel-btn="null"
    :close-on-esc-keydown="!confirmLoading"
    :close-on-overlay-click="!confirmLoading"
    :confirm-btn="null"
    :confirm-loading="confirmLoading"
    :footer="false"
    :header="title"
    placement="center"
    :visible="visible"
    width="min(920px, calc(100vw - 32px))"
    @close="emit('cancel')"
  >
    <section class="paged-multi-select">
      <p v-if="description" class="paged-multi-select__description">{{ description }}</p>

      <div v-if="search" class="paged-multi-select__toolbar">
        <t-input
          v-model="keyword"
          clearable
          :disabled="loading"
          :placeholder="search.placeholder"
          type="search"
          @clear="clearSearch"
          @enter="searchImmediately"
        >
          <template #prefix-icon><t-icon name="search" /></template>
        </t-input>
        <slot name="toolbar" />
      </div>

      <t-alert v-if="errorMessage" class="paged-multi-select__feedback" theme="error" :message="errorMessage" />

      <management-paged-table
        v-model:current="current"
        v-model:page-size="pageSize"
        :cell-slot-names="cellSlotNames"
        :columns="columns"
        :empty-description="resolvedEmptyDescription"
        :empty-title="resolvedEmptyTitle"
        footer-summary=""
        :loading="loading"
        :pagination-props="{ totalContent: false }"
        :row-class-name="rowClassName"
        :row-key="rowKey"
        :rows="rows"
        :selected-row-keys="selectedIds"
        :total="total"
        @page-change="emitPageChange"
        @select-change="updateSelection"
      >
        <template v-for="slotName in cellSlotNames" #[slotName]="slotProps" :key="slotName">
          <slot :name="slotName" v-bind="slotProps" />
        </template>
        <template #feedback><slot name="feedback" /></template>
        <template #empty>
          <slot name="empty">
            <t-empty :description="resolvedEmptyDescription" :title="resolvedEmptyTitle">
              <template v-if="keyword && search?.clearLabel" #action>
                <t-button variant="text" @click="clearSearch">{{ search.clearLabel }}</t-button>
              </template>
            </t-empty>
          </slot>
        </template>
        <template #footer>
          <div class="paged-multi-select__data-footer">
            <div class="paged-multi-select__summary">
              <span>{{ selectionSummary }}</span>
              <span v-if="showTotal && totalLabel" class="paged-multi-select__total">{{ totalLabel(total) }}</span>
            </div>
            <t-pagination
              v-model:current="current"
              v-model:page-size="pageSize"
              :disabled="loading"
              :page-size-options="[10, 20, 50, 100]"
              :total="total"
              :total-content="false"
              @change="emitPageChange"
            />
          </div>
        </template>
      </management-paged-table>

      <footer class="paged-multi-select__actions">
        <t-space>
          <t-button :disabled="loading || confirmLoading" variant="outline" @click="emit('cancel')">
            {{ cancelLabel }}
          </t-button>
          <t-button
            :disabled="!confirmWithoutSelection && selectedIds.length === 0"
            :loading="confirmLoading"
            theme="primary"
            @click="emit('confirm')"
          >
            {{ confirmLabel }}
          </t-button>
        </t-space>
      </footer>
    </section>
  </t-dialog>
</template>
<script setup lang="ts">
import type { PageInfo, TableRowData, TdBaseTableProps } from 'tdesign-vue-next';
import { computed, nextTick, onBeforeUnmount, watch } from 'vue';

import ManagementPagedTable from '@/shared/components/management/ManagementPagedTable.vue';

import type { PagedMultiSelectSearch } from './paged-multi-select';
import { type ExplicitSelection, replaceExplicitPageSelection, type SelectionId } from './selection-model';

export type PagedMultiSelectSelectionLimit = {
  attemptedCount: number;
  maxSelection: number;
};

const props = defineProps<{
  cancelLabel: string;
  cellSlotNames?: string[];
  columns: TdBaseTableProps['columns'];
  confirmLabel: string;
  confirmLoading?: boolean;
  confirmWithoutSelection?: boolean;
  description?: string;
  emptyDescription: string;
  emptyTitle: string;
  errorMessage?: string;
  loading?: boolean;
  maxSelection?: number;
  rowClassName?: TdBaseTableProps['rowClassName'];
  rowKey: string;
  rows: TableRowData[];
  search?: PagedMultiSelectSearch;
  searchEmptyDescription?: string;
  searchEmptyTitle?: string;
  selectedCountLabel: (count: number) => string;
  showTotal?: boolean;
  title?: string;
  total: number;
  totalLabel?: (total: number) => string;
}>();

// 选择弹窗只管理交互状态，候选数据和服务端查询始终由调用模块拥有。
const emit = defineEmits<{
  (event: 'cancel'): void;
  (event: 'confirm'): void;
  (event: 'search', keyword: string): void;
  (event: 'page-change', pageInfo: PageInfo): void;
  (event: 'selection-limit', detail: PagedMultiSelectSelectionLimit): void;
}>();

const selection = defineModel<ExplicitSelection>('selection', { required: true });
const current = defineModel<number>('current', { required: true });
const pageSize = defineModel<number>('pageSize', { required: true });
const keyword = defineModel<string>('keyword', { default: '' });
const visible = defineModel<boolean>('visible', { default: false });

const cellSlotNames = computed(() => props.cellSlotNames ?? []);
const selectedIds = computed(() => Array.from(selection.value.selectedIds));
const selectionSummary = computed(() => props.selectedCountLabel(selectedIds.value.length));
const showTotal = computed(() => props.showTotal ?? props.total > pageSize.value);
const hasKeyword = computed(() => keyword.value.trim().length > 0);
const resolvedEmptyTitle = computed(() =>
  hasKeyword.value ? (props.searchEmptyTitle ?? props.emptyTitle) : props.emptyTitle,
);
const resolvedEmptyDescription = computed(() =>
  hasKeyword.value ? (props.searchEmptyDescription ?? props.emptyDescription) : props.emptyDescription,
);
let searchTimer: ReturnType<typeof setTimeout> | undefined;

watch(keyword, () => {
  if (!props.search) return;
  cancelScheduledSearch();
  searchTimer = setTimeout(runSearch, 300);
});

onBeforeUnmount(cancelScheduledSearch);

function cancelScheduledSearch() {
  if (searchTimer === undefined) return;
  clearTimeout(searchTimer);
  searchTimer = undefined;
}

function runSearch() {
  cancelScheduledSearch();
  current.value = 1;
  emit('search', keyword.value);
}

function searchImmediately() {
  runSearch();
}

function clearSearch() {
  // 等待 v-model watcher 安排当前清空的防抖任务，再立即取消它，避免跨交互的跳过状态。
  void nextTick(runSearch);
}

function emitPageChange(pageInfo: PageInfo) {
  emit('page-change', pageInfo);
}

function updateSelection(currentPageKeys: SelectionId[]) {
  const pageKeys = props.rows.map((row) => row[props.rowKey] as SelectionId);
  const nextSelection = replaceExplicitPageSelection(selection.value, pageKeys, currentPageKeys);
  if (props.maxSelection === undefined || nextSelection.selectedIds.size <= props.maxSelection) {
    selection.value = nextSelection;
    return;
  }
  emit('selection-limit', { attemptedCount: nextSelection.selectedIds.size, maxSelection: props.maxSelection });
}
</script>
<style scoped lang="less">
.paged-multi-select {
  container-type: inline-size;
  display: grid;
  gap: var(--graft-density-gap-12);
  max-block-size: min(70vh, 680px);
}

.paged-multi-select__description {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: calc(var(--graft-density-gap-8) * -1) 0 0;
}

.paged-multi-select__toolbar {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--td-comp-margin-s);
}

.paged-multi-select__toolbar :deep(.t-input) {
  inline-size: min(460px, 100%);
}

.paged-multi-select :deep(.management-table-card) {
  min-block-size: 360px;
}

.paged-multi-select :deep(.management-paged-table__table-host) {
  min-block-size: 260px;
}

.paged-multi-select__data-footer {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-16);
  inline-size: 100%;
  justify-content: space-between;
  min-block-size: 52px;
}

.paged-multi-select__summary {
  color: var(--td-text-color-secondary);
  display: flex;
  flex: 0 0 auto;
  gap: var(--graft-density-gap-8);
  white-space: nowrap;
}

.paged-multi-select__total::before {
  content: '·';
  margin-inline-end: var(--graft-density-gap-8);
}

.paged-multi-select__actions {
  display: flex;
  justify-content: flex-end;
}

.paged-multi-select__data-footer :deep(.t-pagination) {
  margin-inline-start: auto;
}

@container (max-width: 480px) {
  .paged-multi-select__toolbar,
  .paged-multi-select__data-footer {
    align-items: stretch;
    flex-direction: column;
  }

  .paged-multi-select__data-footer :deep(.t-pagination),
  .paged-multi-select__actions :deep(.t-space) {
    justify-content: flex-end;
  }

  .paged-multi-select__summary {
    white-space: normal;
  }
}
</style>
