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
    <template #cards>
      <article
        v-for="row in rows"
        :key="row.id"
        class="log-card app-log-card"
        :class="{ 'log-card--selecting': selectionMode }"
        tabindex="0"
        @click="$emit('detail', row)"
        @keydown.enter="$emit('detail', row)"
      >
        <t-checkbox
          v-if="selectionMode"
          class="log-card__selection"
          :checked="isSelected(row)"
          @click.stop
          @change="toggleCardSelection(row, $event)"
        />
        <div class="log-card__header">
          <t-tag :theme="appLogSeverityTheme(row.severity)" variant="light-outline" size="small">
            {{ row.severity.toUpperCase() }}
          </t-tag>
          <time class="log-card__time">{{ formatCompactDateTime(row.occurred_at, locale) }}</time>
        </div>
        <p class="log-card__title">{{ row.message }}</p>
        <p class="log-card__metadata">
          {{ row.component || '-' }} <span aria-hidden="true">/</span> {{ row.category || '-' }}
        </p>
        <div class="log-card__technical">
          <span>{{ t('appLog.columns.operation') }}</span>
          <log-id-text v-bind="technicalTextProps(appLogOperationText(row, t))" />
        </div>
        <div class="log-card__actions" @click.stop>
          <table-action-menu
            :actions="cardActions(row)"
            :more-label="t('appLog.actions.more')"
            :more-label-fallback="t('appLog.actions.more')"
            @action="(action) => handleCardAction(action, row)"
          />
        </div>
      </article>
    </template>
    <template v-if="filteredEmpty" #empty-action>
      <t-button size="small" theme="default" variant="outline" @click="$emit('clear-filters')">
        {{ t('appLog.actions.reset') }}
      </t-button>
    </template>
    <template #occurred_at="{ row }">
      <span>{{ formatCompactDateTime(appLogRow(row).occurred_at, locale) }}</span>
    </template>
    <template #severity="{ row }">
      <t-tag :theme="appLogSeverityTheme(appLogRow(row).severity)" variant="light-outline" size="small">
        {{ appLogRow(row).severity.toUpperCase() }}
      </t-tag>
    </template>
    <template #category="{ row }">
      <log-id-text :display-value="appLogRow(row).category" :tooltip="appLogRow(row).category" />
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
// 表格呈现应用日志并派发选择/删除事件，批量操作的权限和请求由上层列表负责。
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
  value: 'delete' | 'detail' | 'select';
};

const props = defineProps<{
  emptyDescription: string;
  emptyTitle?: string;
  footerSummary: string;
  filteredEmpty?: boolean;
  loading?: boolean;
  rows: AppLogItem[];
  selectionMode?: boolean;
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
  (e: 'clear-filters'): void;
  (e: 'delete', row: AppLogItem): void;
  (e: 'select-change', rowKeys: Array<string | number>): void;
  (e: 'enter-selection'): void;
}>();
const pageSize = defineModel<number>('pageSize', { required: true });
const cellSlotNames = [
  'occurred_at',
  'severity',
  'category',
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
    createTechnicalColumn(t('appLog.columns.category'), 'category', 184),
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
  cardsVisible: true,
  cellSlotNames,
  columns: columns.value,
  emptyDescription: props.emptyDescription,
  emptyTitle: props.emptyTitle ?? t('appLog.page.emptyTitle'),
  footerSummary: props.footerSummary,
  headLabel: 'app-log-table-head',
  loading: props.loading,
  presentation: 'log' as const,
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

function cardActions(row: AppLogItem): AppLogRowAction[] {
  const actions = rowActions(row);
  if (canDelete.value && !props.selectionMode) {
    actions.push({
      fallbackLabel: t('appLog.batch.select'),
      label: t('appLog.batch.select'),
      value: 'select',
    });
  }
  return actions;
}

function isSelected(row: AppLogItem) {
  return (props.selectedRowKeys ?? []).some((key) => Number(key) === row.id);
}

function toggleCardSelection(row: AppLogItem, checked: boolean | { checked?: boolean }) {
  const selected = isSelected(row);
  const nextSelected = typeof checked === 'boolean' ? checked : checked.checked === true;
  const keys = props.selectedRowKeys ?? [];
  emit('select-change', nextSelected && !selected ? [...keys, row.id] : keys.filter((key) => Number(key) !== row.id));
}

function handleCardAction(action: string, row: AppLogItem) {
  if (action === 'select') {
    emit('enter-selection');
    return;
  }
  handleRowAction(action, row);
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
.log-card-layout();

.log-card {
  position: relative;
}

.log-card__selection {
  left: var(--graft-density-gap-12);
  position: absolute;
  top: var(--graft-density-gap-12);
  z-index: 1;
}

.log-card--selecting {
  padding-left: calc(var(--graft-density-gap-12) + var(--graft-density-gap-24));
}
</style>
