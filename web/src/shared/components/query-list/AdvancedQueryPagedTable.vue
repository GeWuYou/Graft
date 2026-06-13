<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <management-table-card>
    <template v-if="hasHeadContent" #head>
      <section class="advanced-query-paged-table__head" :aria-label="headLabel">
        <p v-if="description" class="advanced-query-paged-table__description">{{ description }}</p>
        <p v-if="summary" class="advanced-query-paged-table__summary">{{ summary }}</p>
      </section>
    </template>
    <template v-if="$slots.toolbar" #toolbar>
      <slot name="toolbar" />
    </template>
    <template v-if="$slots.batch" #batch>
      <slot name="batch" />
    </template>

    <div ref="tableHostRef" class="advanced-query-paged-table__table-host" :data-table-mode="tableWidthPolicy.mode">
      <t-table
        row-key="id"
        :columns="columns"
        :data="rows"
        :loading="loading"
        :selected-row-keys="selectedRowKeys"
        table-layout="fixed"
        :table-content-width="tableWidthPolicy.tableContentWidth"
        cell-empty-content="-"
        hover
        @row-click="emitRowClick"
        @select-change="emitSelectChange"
      >
        <template v-for="slotName in cellSlotNames" #[slotName]="slotProps" :key="slotName">
          <slot :name="slotName" v-bind="slotProps" />
        </template>
        <template v-for="slotName in passthroughTableSlotNames" #[slotName]="slotProps" :key="slotName">
          <slot :name="slotName" v-bind="slotProps" />
        </template>
        <template #empty>
          <div class="advanced-query-paged-table__empty">
            <t-empty :title="emptyTitle" :description="emptyDescription" />
          </div>
        </template>
        <template v-if="$slots.pagination" #pagination>
          <slot name="pagination" />
        </template>
      </t-table>
    </div>

    <template #footer>
      <management-table-pagination :summary="footerSummary">
        <t-pagination
          v-model:current="current"
          v-model:page-size="pageSize"
          :total="total"
          :page-size-options="[10, 20, 50, 100]"
          @change="$emit('page-change')"
        />
      </management-table-pagination>
    </template>
  </management-table-card>
</template>
<script setup lang="ts">
import type { TableRowData, TdBaseTableProps } from 'tdesign-vue-next';
import { computed } from 'vue';

import {
  ManagementTableCard,
  ManagementTablePagination,
  resolveTableWidthPolicy,
  useTableHostWidth,
} from '@/shared/components/management';

const props = defineProps<{
  cellSlotNames: string[];
  columns: TdBaseTableProps['columns'];
  description?: string;
  emptyDescription: string;
  emptyTitle: string;
  footerSummary: string;
  headLabel: string;
  loading?: boolean;
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
  ['toolbar'].filter((slotName) => !props.cellSlotNames.includes(slotName)),
);
const hasHeadContent = computed(() => Boolean(props.description || props.summary));
const { tableHostRef, tableHostWidth } = useTableHostWidth(() => props.columns);

const tableWidthPolicy = computed(() => resolveTableWidthPolicy(props.columns, tableHostWidth.value));

function emitRowClick(context: { row: TableRowData }) {
  emit('row-click', context.row);
}

function emitSelectChange(rowKeys: Array<string | number>) {
  emit('select-change', rowKeys);
}
</script>
<style scoped lang="less">
.advanced-query-paged-table__summary,
.advanced-query-paged-table__description {
  color: var(--td-text-color-secondary);
  margin: 0;
}

.advanced-query-paged-table__empty {
  padding: var(--graft-density-gap-24) 0 var(--graft-density-gap-8);
}

.advanced-query-paged-table__table-host {
  max-width: 100%;
  min-width: 0;
  overflow-x: hidden;
}

.advanced-query-paged-table__table-host[data-table-mode='scroll'] {
  overflow-x: auto;
}
</style>
