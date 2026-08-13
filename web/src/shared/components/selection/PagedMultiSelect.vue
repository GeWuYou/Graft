<template>
  <section class="paged-multi-select">
    <div class="paged-multi-select__toolbar">
      <t-input
        v-model="keyword"
        clearable
        :disabled="loading"
        :placeholder="searchPlaceholder"
        type="search"
        @enter="emitSearch"
        @clear="emitSearch"
      />
      <t-button :loading="loading" theme="primary" variant="outline" @click="emitSearch">{{ searchLabel }}</t-button>
    </div>

    <management-paged-table
      v-model:current="current"
      v-model:page-size="pageSize"
      :cell-slot-names="cellSlotNames"
      :columns="columns"
      :empty-description="emptyDescription"
      :empty-title="emptyTitle"
      :footer-summary="selectionSummary"
      :loading="loading"
      :row-class-name="rowClassName"
      :row-key="rowKey"
      :rows="rows"
      :selected-row-keys="selectedKeys"
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
        <t-button :disabled="loading" variant="outline" @click="emit('cancel')">{{ cancelLabel }}</t-button>
        <t-button
          :disabled="selectedKeys.length === 0"
          :loading="confirmLoading"
          theme="primary"
          @click="emit('confirm')"
        >
          {{ confirmLabel }}
        </t-button>
      </t-space>
    </footer>
  </section>
</template>
<script setup lang="ts">
import type { PageInfo, TableRowData, TdBaseTableProps } from 'tdesign-vue-next';
import { computed } from 'vue';

import ManagementPagedTable from '@/shared/components/management/ManagementPagedTable.vue';

type SelectionKey = string | number;

const props = defineProps<{
  cancelLabel: string;
  cellSlotNames?: string[];
  columns: TdBaseTableProps['columns'];
  confirmLabel: string;
  confirmLoading?: boolean;
  emptyDescription: string;
  emptyTitle: string;
  loading?: boolean;
  rowClassName?: TdBaseTableProps['rowClassName'];
  rowKey: string;
  rows: TableRowData[];
  searchLabel: string;
  searchPlaceholder: string;
  selectedCountLabel: (count: number) => string;
  total: number;
}>();

const emit = defineEmits<{
  (event: 'cancel'): void;
  (event: 'confirm'): void;
  (event: 'search', keyword: string): void;
  (event: 'page-change', pageInfo: PageInfo): void;
}>();

const selectedKeys = defineModel<SelectionKey[]>('selectedKeys', { required: true });
const current = defineModel<number>('current', { required: true });
const pageSize = defineModel<number>('pageSize', { required: true });
const keyword = defineModel<string>('keyword', { required: true });

const cellSlotNames = computed(() => props.cellSlotNames ?? []);
const selectionSummary = computed(() => props.selectedCountLabel(selectedKeys.value.length));

function emitSearch() {
  emit('search', keyword.value);
}

function emitPageChange(pageInfo: PageInfo) {
  emit('page-change', pageInfo);
}

function updateSelection(currentPageKeys: SelectionKey[]) {
  const pageKeys = new Set(props.rows.map((row) => row[props.rowKey] as SelectionKey));
  const retainedKeys = selectedKeys.value.filter((key) => !pageKeys.has(key));
  selectedKeys.value = [...retainedKeys, ...currentPageKeys];
}
</script>
<style scoped lang="less">
.paged-multi-select {
  display: grid;
  gap: var(--td-comp-paddingTB-l);
}

.paged-multi-select__toolbar {
  display: flex;
  gap: var(--td-comp-margin-s);
}

.paged-multi-select__toolbar :deep(.t-input) {
  flex: 1;
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
</style>
