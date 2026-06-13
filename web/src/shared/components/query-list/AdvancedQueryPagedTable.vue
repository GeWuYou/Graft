<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <management-table-card>
    <template #head>
      <section class="advanced-query-paged-table__head" :aria-label="headLabel">
        <p class="advanced-query-paged-table__description">{{ description }}</p>
        <p class="advanced-query-paged-table__summary">{{ summary }}</p>
      </section>
    </template>
    <template v-if="$slots.toolbar" #toolbar>
      <table-view-toolbar>
        <slot name="toolbar" />
      </table-view-toolbar>
    </template>

    <div>
      <t-table
        row-key="id"
        :columns="columns"
        :data="rows"
        :loading="loading"
        table-layout="fixed"
        :table-content-width="tableContentWidth"
        cell-empty-content="-"
        hover
      >
        <template v-for="slotName in cellSlotNames" #[slotName]="slotProps" :key="slotName">
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
  calculateTableContentWidth,
  ManagementTableCard,
  ManagementTablePagination,
  TableViewToolbar,
} from '@/shared/components/management';

const props = defineProps<{
  cellSlotNames: string[];
  columns: TdBaseTableProps['columns'];
  description: string;
  emptyDescription: string;
  emptyTitle: string;
  footerSummary: string;
  headLabel: string;
  loading?: boolean;
  rows: TableRowData[];
  summary: string;
  total: number;
}>();

defineEmits<{
  (e: 'page-change'): void;
}>();

const current = defineModel<number>('current', { required: true });
const pageSize = defineModel<number>('pageSize', { required: true });

const tableContentWidth = computed(() => calculateTableContentWidth(props.columns));
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
</style>
