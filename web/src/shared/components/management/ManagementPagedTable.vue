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
          :class="[
            'management-paged-table__table-host',
            { 'graft-scrollbar--horizontal': resolveTableModeFor(variant.density) === 'scroll' },
          ]"
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
import { computed, nextTick, onActivated, onBeforeUnmount, onDeactivated, onMounted, useSlots, watch } from 'vue';

import ResponsiveTable from '@/shared/components/responsive/ResponsiveTable.vue';
import { emitDebugLog, isDebugFlagEnabled } from '@/shared/debug/runtime';
import type { ResponsiveDensity, ResponsivePresentation } from '@/shared/responsive';

import ManagementTableCard from './ManagementTableCard.vue';
import ManagementTablePagination from './ManagementTablePagination.vue';
import { resolveEmptyManagedColumns, resolveManagedColumns, resolveTableWidthPolicy } from './table-columns';
import { useTableHostWidth } from './use-table-host-width';

// 管理列表统一在此收口响应式列宽、空态和分页，调用方只提供领域数据与单元格插槽。
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
  const columns = resolveManagedColumns(props.columns, props.columnSets[density]);
  return props.rows.length === 0 ? resolveEmptyManagedColumns(columns) : columns;
}

function resolveTableWidthPolicyFor(density: ResponsiveDensity) {
  return resolveTableWidthPolicy(resolveColumns(density), tableHostWidth.value);
}

function resolveTableModeFor(density: ResponsiveDensity) {
  return props.rows.length > 0 ? resolveTableWidthPolicyFor(density).mode : 'fill';
}

function resolveTableContentWidthFor(density: ResponsiveDensity) {
  if (props.rows.length > 0) {
    return resolveTableWidthPolicyFor(density).tableContentWidth;
  }

  // 空态列定义不保留数据态宽表约束，使 TDesign 直接按宿主宽度布局。
  return undefined;
}

const tableLayoutDebugFrames = new Set<number>();
const tableLayoutDebugTimers = new Set<number>();

function sanitizeDebugPath() {
  return typeof window === 'undefined' ? '' : window.location.pathname;
}

function roundLayoutMetric(value: number) {
  return Math.round(value * 10) / 10;
}

function measureTableLayout(stage: string) {
  if (!isDebugFlagEnabled('management.table-layout')) {
    return;
  }

  const host = tableHostRef.value;
  const tableContent = host?.querySelector<HTMLElement>('.t-table__content');
  const table = tableContent?.querySelector<HTMLTableElement>('table');
  const empty = host?.querySelector<HTMLElement>('.t-table__empty');
  const emptyContent = host?.querySelector<HTMLElement>('.management-paged-table__empty');
  const hostRect = host?.getBoundingClientRect();
  const tableContentRect = tableContent?.getBoundingClientRect();
  const tableRect = table?.getBoundingClientRect();
  const emptyRect = empty?.getBoundingClientRect();
  const emptyContentRect = emptyContent?.getBoundingClientRect();
  const tableContentStyle = tableContent ? window.getComputedStyle(tableContent) : undefined;

  emitDebugLog('management.table-layout', 'geometry', {
    stage,
    path: sanitizeDebugPath(),
    rows: props.rows.length,
    hostClientW: host?.clientWidth ?? 0,
    hostScrollW: host?.scrollWidth ?? 0,
    hostX: roundLayoutMetric(hostRect?.x ?? 0),
    hostW: roundLayoutMetric(hostRect?.width ?? 0),
    contentClientW: tableContent?.clientWidth ?? 0,
    contentScrollW: tableContent?.scrollWidth ?? 0,
    contentX: roundLayoutMetric(tableContentRect?.x ?? 0),
    contentW: roundLayoutMetric(tableContentRect?.width ?? 0),
    contentOverflowX: tableContentStyle?.overflowX ?? '',
    tableX: roundLayoutMetric(tableRect?.x ?? 0),
    tableW: roundLayoutMetric(tableRect?.width ?? 0),
    emptyX: roundLayoutMetric(emptyRect?.x ?? 0),
    emptyW: roundLayoutMetric(emptyRect?.width ?? 0),
    emptyInlineW: empty?.style.width || '',
    emptyContentX: roundLayoutMetric(emptyContentRect?.x ?? 0),
    emptyContentW: roundLayoutMetric(emptyContentRect?.width ?? 0),
    tableMode: host?.dataset.tableMode ?? '',
  });
}

function clearTableLayoutDebugSchedule() {
  tableLayoutDebugFrames.forEach((frameId) => window.cancelAnimationFrame(frameId));
  tableLayoutDebugFrames.clear();
  tableLayoutDebugTimers.forEach((timerId) => window.clearTimeout(timerId));
  tableLayoutDebugTimers.clear();
}

// 覆盖 Vue 提交与 TDesign 定时溢出校准窗口，比较同一实例在相邻帧的真实几何变化。
function scheduleTableLayoutMeasurements(reason: string) {
  if (typeof window === 'undefined' || !isDebugFlagEnabled('management.table-layout')) {
    return;
  }

  clearTableLayoutDebugSchedule();
  measureTableLayout(`${reason}:sync`);
  void nextTick(() => {
    measureTableLayout(`${reason}:next-tick`);
    const firstFrameId = window.requestAnimationFrame(() => {
      tableLayoutDebugFrames.delete(firstFrameId);
      measureTableLayout(`${reason}:frame-1`);
      const secondFrameId = window.requestAnimationFrame(() => {
        tableLayoutDebugFrames.delete(secondFrameId);
        measureTableLayout(`${reason}:frame-2`);
      });
      tableLayoutDebugFrames.add(secondFrameId);
    });
    tableLayoutDebugFrames.add(firstFrameId);
  });
  [0, 32, 160].forEach((delay) => {
    const timerId = window.setTimeout(() => {
      tableLayoutDebugTimers.delete(timerId);
      measureTableLayout(`${reason}:timeout-${delay}`);
    }, delay);
    tableLayoutDebugTimers.add(timerId);
  });
}

onMounted(() => scheduleTableLayoutMeasurements('mounted'));
onActivated(() => scheduleTableLayoutMeasurements('activated'));
onDeactivated(clearTableLayoutDebugSchedule);
onBeforeUnmount(clearTableLayoutDebugSchedule);

watch([tableHostWidth, () => props.rows.length, () => props.loading], () =>
  scheduleTableLayoutMeasurements('layout-input-changed'),
);

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
