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

    <responsive-table
      :density-scope="densityScope"
      :entity-card-layout="entityCardLayout"
      :preserve-inactive="preserveInactive"
      :presentation="responsivePresentation"
    >
      <template v-if="$slots.cards" #cards>
        <div class="management-paged-table__cards"><slot name="cards" /></div>
      </template>
      <template #default="{ variant }">
        <div
          ref="tableHostRef"
          class="management-paged-table__table-host graft-scrollbar"
          :data-table-mode="resolveTableModeFor(variant.density)"
        >
          <t-table
            :key="props.size ?? 'default-size'"
            :row-key="resolvedRowKey"
            :columns="resolveColumns(variant.density)"
            :data="props.rows"
            :loading="props.loading"
            :row-class-name="props.rowClassName"
            :selected-row-keys="resolveSelectedRowKeys(variant.density)"
            :size="props.size"
            :sort="props.sort"
            table-layout="fixed"
            :table-content-width="resolveTableContentWidthFor(variant.density)"
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
      </template>
    </responsive-table>

    <template v-if="paginationVisible" #footer>
      <slot name="footer">
        <management-table-pagination
          :hide-summary-on-compact="props.hideFooterSummaryOnCompact"
          :summary="props.footerSummary"
        >
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

import ResponsiveTable from '@/shared/components/responsive/ResponsiveTable.vue';
import type { ResponsiveDensity, ResponsivePresentation } from '@/shared/responsive';

import ManagementTableCard from './ManagementTableCard.vue';
import ManagementTablePagination from './ManagementTablePagination.vue';
import { resolveManagedColumns, resolveTableWidthPolicy } from './table-columns';
import { useTableHostWidth } from './use-table-host-width';

const RESERVED_SLOT_NAMES = new Set([
  'batch',
  'cards',
  'default',
  'empty',
  'empty-action',
  'feedback',
  'footer',
  'head',
  'pagination',
  'toolbar',
]);

type ResponsiveEntityCardLayout = 'adaptive' | 'compact';
type ResponsiveColumnSets = Partial<Record<ResponsiveDensity, string[]>>;

const props = withDefaults(
  defineProps<{
    cellSlotNames?: string[];
    cardsVisible?: boolean;
    columnSets?: ResponsiveColumnSets;
    densityScope?: 'container' | 'viewport';
    columns: TdBaseTableProps['columns'];
    description?: string;
    emptyDescription: string;
    emptyTitle: string;
    entityCardLayout?: ResponsiveEntityCardLayout;
    footerSummary: string;
    headLabel?: string;
    hideFooterSummaryOnCompact?: boolean;
    loading?: boolean;
    pageSizeOptions?: PaginationProps['pageSizeOptions'];
    paginationProps?: Partial<PaginationProps>;
    paginationVisible?: boolean;
    presentation?: ResponsivePresentation;
    preserveInactive?: boolean;
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
    cardsVisible: false,
    columnSets: () => ({}),
    densityScope: 'container',
    description: '',
    entityCardLayout: 'compact',
    headLabel: '',
    hideFooterSummaryOnCompact: false,
    loading: false,
    pageSizeOptions: () => [10, 20, 50, 100],
    paginationProps: () => ({}),
    paginationVisible: true,
    presentation: 'data',
    preserveInactive: false,
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
  (e: 'sort-change', sort: TableSort | undefined): void;
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
  // 管理列表的总数由 footerSummary 统一呈现，避免 Pagination 重复渲染同义统计。
  totalContent: false,
  ...props.paginationProps,
}));
// 分页是管理列表的默认结果面；只有调用方明确关闭时才隐藏，避免包装组件传递 undefined 时丢失 footer。
const paginationVisible = computed(() => props.paginationVisible !== false);
const resolvedRowKey = computed(() => props.rowKey || 'id');
const responsivePresentation = computed<ResponsivePresentation>(() =>
  props.cardsVisible && props.presentation === 'data' ? 'entity' : props.presentation,
);
function resolveSelectedRowKeys(density: ResponsiveDensity) {
  return density === 'compact' && responsivePresentation.value === 'entity' ? [] : props.selectedRowKeys;
}
const { tableHostRef, tableHostWidth } = useTableHostWidth(() => props.columns);
function resolveColumns(density: ResponsiveDensity) {
  return resolveManagedColumns(props.columns, props.columnSets[density]);
}

function resolveTableWidthPolicyFor(density: ResponsiveDensity) {
  return resolveTableWidthPolicy(resolveColumns(density), tableHostWidth.value);
}

function resolveTableModeFor(density: ResponsiveDensity) {
  return props.rows.length > 0 ? resolveTableWidthPolicyFor(density).mode : 'fill';
}

function resolveTableContentWidthFor(density: ResponsiveDensity) {
  // 空态没有需要横向查看的行，使用宿主宽度避免 TDesign 以超宽表格内容区作为空态居中基准。
  return props.rows.length > 0 ? resolveTableWidthPolicyFor(density).tableContentWidth : undefined;
}

function emitPageChange(pageInfo: PageInfo) {
  emit('page-change', pageInfo);
}

function emitRowClick(context: { row: TableRowData }) {
  emit('row-click', context.row);
}

function emitSelectChange(rowKeys: Array<string | number>) {
  emit('select-change', rowKeys);
}

function emitSortChange(sort: TableSort | undefined) {
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

.management-paged-table__cards {
  min-width: 0;
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
