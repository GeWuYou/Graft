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
      <div v-if="search" class="paged-multi-select__toolbar">
        <t-input
          v-model="keyword"
          clearable
          :disabled="loading"
          :placeholder="search.placeholder"
          type="search"
          @clear="clearSearch"
          @enter="searchImmediately"
        />
        <t-button :loading="loading" theme="primary" variant="outline" @click="searchImmediately">
          {{ search.label }}
        </t-button>
        <slot name="toolbar" />
      </div>

      <management-paged-table
        v-model:current="current"
        v-model:page-size="pageSize"
        :cell-slot-names="cellSlotNames"
        :columns="columns"
        :empty-description="emptyDescription"
        :empty-title="emptyTitle"
        footer-summary=""
        :loading="loading"
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
        <template #empty><slot name="empty" /></template>
      </management-paged-table>

      <footer class="paged-multi-select__footer">
        <span class="paged-multi-select__summary">{{ selectionSummary }}</span>
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
import { computed, onBeforeUnmount, watch } from 'vue';

import ManagementPagedTable from '@/shared/components/management/ManagementPagedTable.vue';

import { type ExplicitSelection, replaceExplicitPageSelection, type SelectionId } from './selection-model';

/** PagedMultiSelectSearch 声明调用方已具备服务端关键词查询能力时显示的搜索文案。 */
export type PagedMultiSelectSearch = {
  label: string;
  placeholder: string;
};

const props = defineProps<{
  cancelLabel: string;
  cellSlotNames?: string[];
  columns: TdBaseTableProps['columns'];
  confirmLabel: string;
  confirmLoading?: boolean;
  confirmWithoutSelection?: boolean;
  emptyDescription: string;
  emptyTitle: string;
  loading?: boolean;
  rowClassName?: TdBaseTableProps['rowClassName'];
  rowKey: string;
  rows: TableRowData[];
  search?: PagedMultiSelectSearch;
  selectedCountLabel: (count: number) => string;
  title?: string;
  total: number;
}>();

// 选择弹窗只管理交互状态，候选数据和服务端查询始终由调用模块拥有。
const emit = defineEmits<{
  (event: 'cancel'): void;
  (event: 'confirm'): void;
  (event: 'search', keyword: string): void;
  (event: 'page-change', pageInfo: PageInfo): void;
}>();

const selection = defineModel<ExplicitSelection>('selection', { required: true });
const current = defineModel<number>('current', { required: true });
const pageSize = defineModel<number>('pageSize', { required: true });
const keyword = defineModel<string>('keyword', { default: '' });
const visible = defineModel<boolean>('visible', { default: false });

const cellSlotNames = computed(() => props.cellSlotNames ?? []);
const selectedIds = computed(() => Array.from(selection.value.selectedIds));
const selectionSummary = computed(() => props.selectedCountLabel(selectedIds.value.length));
let searchTimer: ReturnType<typeof setTimeout> | undefined;
let skipNextKeywordSearch = false;

watch(keyword, () => {
  if (!props.search) return;
  if (skipNextKeywordSearch) {
    skipNextKeywordSearch = false;
    return;
  }
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
  // TInput 先同步 v-model，再异步派发 clear；跳过随后的同值防抖调度，避免重复查询。
  skipNextKeywordSearch = true;
  runSearch();
}

function emitPageChange(pageInfo: PageInfo) {
  emit('page-change', pageInfo);
}

function updateSelection(currentPageKeys: SelectionId[]) {
  const pageKeys = props.rows.map((row) => row[props.rowKey] as SelectionId);
  selection.value = replaceExplicitPageSelection(selection.value, pageKeys, currentPageKeys);
}
</script>
<style scoped lang="less">
.paged-multi-select {
  display: grid;
  gap: var(--td-comp-paddingTB-l);
  max-block-size: min(70vh, 680px);
}

.paged-multi-select__toolbar {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--td-comp-margin-s);
}

.paged-multi-select__toolbar :deep(.t-input) {
  inline-size: min(420px, 100%);
}

.paged-multi-select__footer {
  align-items: center;
  border-top: 1px solid var(--td-component-stroke);
  display: flex;
  gap: var(--td-comp-margin-l);
  justify-content: space-between;
  padding-top: var(--td-comp-paddingTB-l);
}

.paged-multi-select__summary {
  color: var(--td-text-color-secondary);
}

@container (max-width: 480px) {
  .paged-multi-select__toolbar,
  .paged-multi-select__footer {
    align-items: stretch;
    flex-direction: column;
  }

  .paged-multi-select__footer :deep(.t-space) {
    justify-content: flex-end;
  }
}
</style>
