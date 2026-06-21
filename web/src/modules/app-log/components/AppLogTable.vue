<template>
  <advanced-query-paged-table
    v-model:current="current"
    v-model:page-size="pageSize"
    v-bind="pagedTableProps"
    @page-change="$emit('page-change')"
    @row-click="(row) => $emit('detail', appLogRow(row))"
    @select-change="$emit('select-change', $event)"
  >
    <template v-if="$slots.toolbar" #toolbar>
      <slot name="toolbar" />
    </template>
    <template v-if="$slots.batch" #batch>
      <slot name="batch" />
    </template>
    <template #occurred_at="{ row }">
      <span>{{ formatCompactDateTime(appLogRow(row).occurred_at, locale) }}</span>
    </template>
    <template #severity="{ row }">
      <t-tag :theme="appLogSeverityTheme(appLogRow(row).severity)" variant="light-outline" size="small">
        {{ appLogRow(row).severity.toUpperCase() }}
      </t-tag>
    </template>
    <template #message="{ row }">
      <div class="stack-cell stack-cell--compact">
        <strong>{{ appLogRow(row).message }}</strong>
        <span v-if="appLogRow(row).error" class="stack-cell__secondary">{{ appLogRow(row).error }}</span>
      </div>
    </template>
    <template #operation="{ row }">
      <log-id-text v-bind="technicalTextProps(appLogOperationText(appLogRow(row), t))" />
    </template>
    <template #correlation="{ row }">
      <log-id-text
        :display-value="appLogCorrelationText(appLogRow(row), t)"
        :tooltip="appLogCorrelationText(appLogRow(row), t)"
        v-bind="technicalCopyLabels"
      />
    </template>
    <template #request_id="{ row }">
      <log-id-text
        :display-value="appLogRow(row).request_id"
        :tooltip="appLogRow(row).request_id"
        v-bind="technicalCopyLabels"
      />
    </template>
    <template #fields="{ row }">
      <span>{{ appLogFieldsCount(appLogRow(row)) }}</span>
    </template>
    <template #actions="{ row }">
      <table-action-menu
        :actions="rowActions(appLogRow(row))"
        :more-label="t('appLog.actions.more')"
        :more-label-fallback="t('appLog.actions.more')"
        @action="(action) => handleRowAction(action, appLogRow(row))"
      />
    </template>
  </advanced-query-paged-table>
</template>
<script setup lang="ts">
import type { TdBaseTableProps } from 'tdesign-vue-next';
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import {
  createActionColumn,
  createCountColumn,
  createIdentifierColumn,
  createMainTextColumn,
  createStatusColumn,
  createTechnicalColumn,
  createTimeColumn,
  formatCompactDateTime,
  resolveManagedColumns,
  TableActionMenu,
} from '@/shared/components/management';
import { AdvancedQueryPagedTable } from '@/shared/components/query-list';
import { LogIdText } from '@/shared/observability';
import { usePermissionStore } from '@/store';

import { APP_LOG_PERMISSION_CODE } from '../contract/permissions';
import {
  appLogCorrelationText,
  appLogFieldsCount,
  appLogOperationText,
  appLogSeverityTheme,
} from '../shared/presentation';
import type { AppLogItem } from '../types/app-log';

type AppLogRowAction = {
  fallbackLabel: string;
  label: string;
  testId?: string;
  value: 'delete' | 'detail';
};

const props = defineProps<{
  emptyDescription: string;
  footerSummary: string;
  loading?: boolean;
  rows: AppLogItem[];
  selectedRowKeys?: Array<string | number>;
  total: number;
  visibleColumnKeys?: string[];
}>();

const current = defineModel<number>('current', { required: true });
const { t, locale } = useI18n();
const permissionStore = usePermissionStore();
const emit = defineEmits<{
  (e: 'page-change'): void;
  (e: 'detail', row: AppLogItem): void;
  (e: 'delete', row: AppLogItem): void;
  (e: 'select-change', rowKeys: Array<string | number>): void;
}>();
const pageSize = defineModel<number>('pageSize', { required: true });
const cellSlotNames = [
  'occurred_at',
  'severity',
  'message',
  'operation',
  'correlation',
  'request_id',
  'fields',
  'actions',
];
const technicalCopyLabels = computed(() => ({
  copyable: true,
  copyLabel: t('appLog.actions.copy'),
  copySuccessLabel: t('appLog.actions.copySuccess'),
  copyFailLabel: t('appLog.actions.copyFail'),
}));
const canDelete = computed(() => permissionStore.hasPermission(APP_LOG_PERMISSION_CODE.DELETE));

const columns = computed<TdBaseTableProps['columns']>(() => {
  void locale.value;
  const selectionColumn = canDelete.value
    ? [
        {
          colKey: 'row-select',
          fixed: 'left' as const,
          type: 'multiple',
          width: 48,
        },
      ]
    : [];
  const allColumns: TdBaseTableProps['columns'] = [
    ...selectionColumn,
    createTimeColumn(t('appLog.columns.occurredAt'), 'occurred_at', 176),
    createStatusColumn(t('appLog.columns.severity'), 'severity', 104),
    createIdentifierColumn(t('appLog.columns.component'), 'component', 184),
    createTechnicalColumn(t('appLog.columns.operation'), 'operation', 196),
    createMainTextColumn(t('appLog.columns.message'), 'message', 420),
    createTechnicalColumn(t('appLog.columns.correlation'), 'correlation', 260),
    createTechnicalColumn(t('appLog.columns.requestId'), 'request_id', 260),
    createCountColumn(t('appLog.columns.fields'), 'fields', 92),
    createActionColumn(t('appLog.columns.actions'), 156, 'center', 'actions'),
  ];

  return resolveManagedColumns(allColumns, props.visibleColumnKeys, ['row-select', 'actions']);
});
const pagedTableProps = computed(() => ({
  cellSlotNames,
  columns: columns.value,
  emptyDescription: props.emptyDescription,
  emptyTitle: t('appLog.page.emptyTitle'),
  footerSummary: props.footerSummary,
  headLabel: 'app-log-table-head',
  loading: props.loading,
  rows: props.rows,
  selectedRowKeys: props.selectedRowKeys,
  total: props.total,
}));

function appLogRow(row: unknown) {
  return row as AppLogItem;
}

function technicalTextProps(value: string) {
  return {
    displayValue: value,
    tooltip: value,
    ...technicalCopyLabels.value,
  };
}

function rowActions(row: AppLogItem) {
  const actions: AppLogRowAction[] = [
    {
      fallbackLabel: t('appLog.actions.detail'),
      label: t('appLog.actions.detail'),
      testId: `app-log-detail-${row.id}`,
      value: 'detail',
    },
  ];

  if (permissionStore.hasPermission(APP_LOG_PERMISSION_CODE.DELETE)) {
    actions.push({
      fallbackLabel: t('appLog.actions.delete'),
      label: t('appLog.actions.delete'),
      value: 'delete',
    });
  }

  return actions;
}

function handleRowAction(action: string, row: AppLogItem) {
  if (action === 'detail') {
    emit('detail', row);
    return;
  }
  if (action === 'delete') {
    emit('delete', row);
  }
}

void LogIdText;
void TableActionMenu;
void emit;
</script>
<style scoped lang="less">
@import '@/shared/observability/log-table-cells.less';

.log-table-stack-cells();
</style>
