<template>
  <management-paged-table
    v-model:current="current"
    v-model:page-size="pageSize"
    :cell-slot-names="props.cellSlotNames"
    :cards-visible="props.cardsVisible"
    :column-sets="props.columnSets"
    :columns="props.columns"
    :description="props.description"
    :empty-description="props.emptyDescription"
    :empty-title="props.emptyTitle"
    :footer-summary="props.footerSummary"
    :head-label="props.headLabel"
    :loading="props.loading"
    :density-scope="props.densityScope"
    :entity-card-layout="props.entityCardLayout"
    :pagination-props="props.paginationProps"
    :pagination-visible="props.paginationVisible"
    :presentation="props.presentation"
    :preserve-inactive="props.preserveInactive"
    :row-class-name="props.rowClassName"
    :row-key="props.rowKey"
    :rows="props.rows"
    :selected-row-keys="props.selectedRowKeys"
    :summary="props.summary"
    :total="props.total"
    @page-change="$emit('page-change')"
    @row-click="(row) => emit('row-click', row)"
    @select-change="(rowKeys) => emit('select-change', rowKeys)"
  >
    <template v-if="$slots.toolbar" #toolbar>
      <slot name="toolbar" />
    </template>
    <template v-if="$slots.batch" #batch>
      <slot name="batch" />
    </template>
    <template v-if="$slots.cards" #cards>
      <slot name="cards" />
    </template>
    <template v-if="$slots.head" #head>
      <slot name="head" />
    </template>
    <template v-if="$slots.pagination" #pagination>
      <slot name="pagination" />
    </template>
    <template v-for="slotName in passthroughTableSlotNames" #[slotName]="slotProps" :key="slotName">
      <slot :name="slotName" v-bind="slotProps" />
    </template>
  </management-paged-table>
</template>
<script setup lang="ts">
import type { PaginationProps, TableRowData, TdBaseTableProps } from 'tdesign-vue-next';
import { computed } from 'vue';

import { ManagementPagedTable } from '@/shared/components/management';
import type { ResponsiveDensity, ResponsivePresentation } from '@/shared/responsive';

// 该组件只负责把查询页的分页模型与表格 slot 转交给共享表格壳，不拥有查询条件或服务端数据状态。
const props = defineProps<{
  cellSlotNames: string[];
  cardsVisible?: boolean;
  columnSets?: Partial<Record<ResponsiveDensity, string[]>>;
  columns: TdBaseTableProps['columns'];
  description?: string;
  densityScope?: 'container' | 'viewport';
  emptyDescription: string;
  emptyTitle: string;
  entityCardLayout?: 'adaptive' | 'compact';
  footerSummary: string;
  headLabel: string;
  loading?: boolean;
  paginationProps?: Partial<PaginationProps>;
  paginationVisible?: boolean;
  presentation?: ResponsivePresentation;
  preserveInactive?: boolean;
  rowClassName?: TdBaseTableProps['rowClassName'];
  rowKey?: string;
  rows: TableRowData[];
  selectedRowKeys?: Array<string | number>;
  summary?: string;
  total: number;
}>();

const emit = defineEmits<{
  (e: 'page-change'): void;
  (e: 'row-click', row: TableRowData): void;
  (e: 'select-change', rowKeys: Array<string | number>): void;
}>();

const current = defineModel<number>('current', { required: true });
const pageSize = defineModel<number>('pageSize', { required: true });

const passthroughTableSlotNames = computed(() =>
  Array.from(new Set([...props.cellSlotNames, 'empty', 'empty-action'])).filter(
    (slotName) => slotName !== 'pagination',
  ),
);
</script>
