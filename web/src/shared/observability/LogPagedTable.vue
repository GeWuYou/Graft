<template>
  <management-table-card>
    <template #head>
      <section class="log-paged-table__head" :aria-label="headLabel">
        <p class="log-paged-table__description">{{ description }}</p>
        <p class="log-paged-table__summary">{{ summary }}</p>
      </section>
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
          <div class="log-paged-table__empty">
            <t-empty :title="emptyTitle" :description="emptyDescription" />
          </div>
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
.log-paged-table__summary,
.log-paged-table__description {
  color: var(--td-text-color-secondary);
  margin: 0;
}

.log-paged-table__empty {
  padding: 24px 0 8px;
}
</style>
