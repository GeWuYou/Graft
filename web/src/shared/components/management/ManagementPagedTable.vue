<template>
  <management-table-card>
    <template v-if="hasHeadContent" #head>
      <slot name="head">
        <section class="management-paged-table__head" :aria-label="props.headLabel">
          <p v-if="props.description" class="management-paged-table__description">{{ props.description }}</p>
          <p v-if="props.summary" class="management-paged-table__summary">{{ props.summary }}</p>
        </section>
      </slot>
    </template>
    <template v-if="$slots.toolbar" #toolbar>
      <slot name="toolbar" />
    </template>
    <template v-if="$slots.batch" #batch>
      <slot name="batch" />
    </template>

    <slot name="feedback" />

    <div
      ref="tableHostRef"
      class="management-paged-table__table-host graft-scrollbar"
      :data-table-mode="tableWidthPolicy.mode"
    >
      <t-table
        :key="props.size ?? 'default-size'"
        :row-key="resolvedRowKey"
        :columns="props.columns"
        :data="props.rows"
        :loading="props.loading"
        :row-class-name="props.rowClassName"
        :selected-row-keys="props.selectedRowKeys"
        :size="props.size"
        :sort="props.sort"
        table-layout="fixed"
        :table-content-width="tableWidthPolicy.tableContentWidth"
        cell-empty-content="-"
        hover
        @row-click="emitRowClick"
        @select-change="emitSelectChange"
        @sort-change="emitSortChange"
      >
        <template v-for="slotName in tableSlotNames" #[slotName]="slotProps" :key="slotName">
          <slot :name="slotName" v-bind="slotProps" />
        </template>
        <template #empty>
          <slot name="empty">
            <div class="management-paged-table__empty">
              <t-empty :title="props.emptyTitle" :description="props.emptyDescription">
                <template v-if="$slots['empty-action']" #action>
                  <slot name="empty-action" />
                </template>
              </t-empty>
            </div>
          </slot>
        </template>
      </t-table>
    </div>

    <template v-if="props.paginationVisible" #footer>
      <slot name="footer">
        <management-table-pagination :summary="props.footerSummary">
          <slot name="pagination">
            <t-pagination
              v-model:current="current"
              v-model:page-size="pageSize"
              v-bind="resolvedPaginationProps"
              :total="props.total"
              @change="emitPageChange"
            />
          </slot>
        </management-table-pagination>
      </slot>
    </template>
  </management-table-card>
</template>
<script setup lang="ts">
import type { PageInfo, PaginationProps, TableRowData, TableSort, TdBaseTableProps } from 'tdesign-vue-next';
import { computed, useSlots } from 'vue';

import ManagementTableCard from './ManagementTableCard.vue';
import ManagementTablePagination from './ManagementTablePagination.vue';
import { resolveTableWidthPolicy } from './table-columns';
import { useTableHostWidth } from './use-table-host-width';

const RESERVED_SLOT_NAMES = new Set([
  'batch',
  'default',
  'empty',
  'empty-action',
  'feedback',
  'footer',
  'head',
  'pagination',
  'toolbar',
]);

const props = withDefaults(
  defineProps<{
    cellSlotNames?: string[];
    columns: TdBaseTableProps['columns'];
    description?: string;
    emptyDescription: string;
    emptyTitle: string;
    footerSummary: string;
    headLabel?: string;
    loading?: boolean;
    pageSizeOptions?: PaginationProps['pageSizeOptions'];
    paginationProps?: Partial<PaginationProps>;
    paginationVisible?: boolean;
    rowClassName?: TdBaseTableProps['rowClassName'];
    rowKey?: string;
    rows: TableRowData[];
    selectedRowKeys?: Array<string | number>;
    size?: TdBaseTableProps['size'];
    sort?: TableSort;
    summary?: string;
    total: number;
  }>(),
  {
    cellSlotNames: () => [],
    description: '',
    headLabel: '',
    loading: false,
    pageSizeOptions: () => [10, 20, 50, 100],
    paginationProps: () => ({}),
    paginationVisible: true,
    rowClassName: undefined,
    rowKey: 'id',
    selectedRowKeys: () => [],
    size: undefined,
    sort: undefined,
    summary: '',
  },
);

const emit = defineEmits<{
  (e: 'page-change', pageInfo: PageInfo): void;
  (e: 'row-click', row: TableRowData): void;
  (e: 'select-change', rowKeys: Array<string | number>): void;
  (e: 'sort-change', sort: TableSort): void;
}>();

const current = defineModel<number>('current', { required: true });
const pageSize = defineModel<number>('pageSize', { required: true });

const slots = useSlots();
const hasHeadContent = computed(() => Boolean(slots.head || props.description || props.summary));
const tableSlotNames = computed(() => {
  if (props.cellSlotNames.length > 0) {
    return props.cellSlotNames;
  }

  return Object.keys(slots).filter((slotName) => !RESERVED_SLOT_NAMES.has(slotName));
});
const resolvedPaginationProps = computed<Partial<PaginationProps>>(() => ({
  pageSizeOptions: props.pageSizeOptions,
  ...props.paginationProps,
}));
const resolvedRowKey = computed(() => props.rowKey || 'id');
const { tableHostRef, tableHostWidth } = useTableHostWidth(() => props.columns);
const tableWidthPolicy = computed(() => resolveTableWidthPolicy(props.columns, tableHostWidth.value));

function emitPageChange(pageInfo: PageInfo) {
  emit('page-change', pageInfo);
}

function emitRowClick(context: { row: TableRowData }) {
  emit('row-click', context.row);
}

function emitSelectChange(rowKeys: Array<string | number>) {
  emit('select-change', rowKeys);
}

function emitSortChange(sort: TableSort) {
  emit('sort-change', sort);
}
</script>
<style scoped lang="less">
.management-paged-table__summary,
.management-paged-table__description {
  color: var(--td-text-color-secondary);
  margin: 0;
}

.management-paged-table__empty {
  padding: var(--graft-density-gap-24) 0 var(--graft-density-gap-8);
}

.management-paged-table__table-host {
  max-width: 100%;
  min-width: 0;
  overflow-x: hidden;
  width: 100%;
}

.management-paged-table__table-host[data-table-mode='scroll'] {
  overflow-x: auto;
}

.management-paged-table__table-host :deep(.t-table__content) {
  min-width: 0;
}
</style>
