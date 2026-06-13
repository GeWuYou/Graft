<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <advanced-query-paged-table
    v-model:current="current"
    v-model:page-size="pageSize"
    :cell-slot-names="cellSlotNames"
    :columns="columns"
    :description="description"
    :empty-description="emptyDescription"
    :empty-title="t('accessLog.page.emptyTitle')"
    :footer-summary="footerSummary"
    head-label="access-log-table-head"
    :loading="loading"
    :rows="rows"
    :summary="summary"
    :total="total"
    @page-change="$emit('page-change')"
    @row-click="(row) => $emit('detail', accessRow(row))"
  >
    <template v-if="$slots.toolbar" #toolbar>
      <slot name="toolbar" />
    </template>
    <template #method="{ row }">
      <t-tag theme="primary" variant="light-outline" size="small">{{ accessRow(row).method }}</t-tag>
    </template>
    <template #path="{ row }">
      <div class="stack-cell stack-cell--compact">
        <strong>{{ accessRow(row).path }}</strong>
        <span v-if="accessLogPathSecondary(accessRow(row))" class="stack-cell__secondary">
          {{ t('accessLog.path.routeTemplateValue', { route: accessLogPathSecondary(accessRow(row)) }) }}
        </span>
      </div>
    </template>
    <template #status_code="{ row }">
      <t-tag :theme="statusTheme(accessRow(row).status_code)" variant="light-outline" size="small">
        {{ accessRow(row).status_code }}
      </t-tag>
    </template>
    <template #duration_ms="{ row }">
      <span :class="{ 'duration-danger': accessRow(row).duration_ms >= 3000 }">
        {{ accessRow(row).duration_ms }} ms
      </span>
    </template>
    <template #user="{ row }">
      <div class="stack-cell">
        <strong>{{ accessLogUserPrimary(accessRow(row), t) }}</strong>
        <span class="stack-cell__secondary">{{ accessLogUserSecondary(accessRow(row), t) }}</span>
      </div>
    </template>
    <template #request_id="{ row }">
      <log-id-text
        :display-value="accessRow(row).request_id"
        :tooltip="accessRow(row).request_id"
        v-bind="technicalCopyLabels"
      />
    </template>
    <template #trace_id="{ row }">
      <log-id-text
        :display-value="accessRow(row).trace_id"
        :tooltip="accessRow(row).trace_id"
        v-bind="technicalCopyLabels"
      />
    </template>
    <template #client_ip="{ row }">
      <log-id-text
        :display-value="accessRow(row).client_ip"
        :tooltip="accessRow(row).client_ip"
        v-bind="technicalCopyLabels"
      />
    </template>
    <template #user_agent="{ row }">
      <log-id-text
        :display-value="accessRow(row).user_agent"
        :tooltip="accessRow(row).user_agent"
        v-bind="technicalCopyLabels"
      />
    </template>
    <template #started_at="{ row }">
      <span>{{ Management.formatCompactDateTime(accessRow(row).started_at, locale) }}</span>
    </template>
    <template #occurred_at="{ row }">
      <span>{{ Management.formatCompactDateTime(accessRow(row).occurred_at, locale) }}</span>
    </template>
    <template #operation="{ row }">
      <table-action-menu
        :actions="rowActions(accessRow(row))"
        :more-label="t('accessLog.actions.more')"
        :more-label-fallback="t('accessLog.actions.more')"
        @action="(action) => handleRowAction(action, accessRow(row))"
      />
    </template>
  </advanced-query-paged-table>
</template>
<script setup lang="ts">
import type { TdBaseTableProps } from 'tdesign-vue-next';
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import * as Management from '@/shared/components/management';
import { AdvancedQueryPagedTable } from '@/shared/components/query-list';
import { LogIdText } from '@/shared/observability';

import { copyAccessLogValue } from '../shared/clipboard';
import { accessLogPathSecondary, accessLogUserPrimary, accessLogUserSecondary } from '../shared/presentation';
import type { AccessLogItem } from '../types/access-log';

type AccessLogRowAction = {
  fallbackLabel: string;
  label: string;
  testId?: string;
  value: 'copy-path' | 'copy-request-id' | 'detail' | 'view-app-log' | 'view-audit';
};

const props = defineProps<{
  description: string;
  emptyDescription: string;
  footerSummary: string;
  loading?: boolean;
  rows: AccessLogItem[];
  summary: string;
  total: number;
  visibleColumnKeys?: string[];
}>();

const emit = defineEmits<{
  (e: 'detail', row: AccessLogItem): void;
  (e: 'page-change'): void;
  (e: 'view-app-log', row: AccessLogItem): void;
  (e: 'view-audit', row: AccessLogItem): void;
}>();

const current = defineModel<number>('current', { required: true });
const pageSize = defineModel<number>('pageSize', { required: true });

const { t, locale } = useI18n();
const TableActionMenu = Management.TableActionMenu;
const technicalCopyLabels = computed(() => ({
  copyable: true,
  copyLabel: t('accessLog.actions.copy'),
  copySuccessLabel: t('accessLog.actions.copySuccess'),
  copyFailLabel: t('accessLog.actions.copyFail'),
}));
const cellSlotNames = [
  'method',
  'path',
  'status_code',
  'duration_ms',
  'user',
  'request_id',
  'trace_id',
  'client_ip',
  'user_agent',
  'started_at',
  'occurred_at',
  'operation',
];

const columns = computed<TdBaseTableProps['columns']>(() => {
  void locale.value;
  const allColumns: TdBaseTableProps['columns'] = [
    Management.createTimeColumn(t('accessLog.columns.startedAt'), 'started_at', 176),
    Management.createStatusColumn(t('accessLog.columns.method'), 'method', 96),
    Management.createMainTextColumn(t('accessLog.columns.path'), 'path', 360),
    Management.createStatusColumn(t('accessLog.columns.statusCode'), 'status_code', 112),
    Management.createCountColumn(t('accessLog.columns.durationMs'), 'duration_ms', 112),
    Management.createIdentifierColumn(t('accessLog.columns.user'), 'user', 170),
    Management.createTechnicalColumn(t('accessLog.columns.requestId'), 'request_id', 260),
    Management.createTechnicalColumn(t('accessLog.columns.traceId'), 'trace_id', 260),
    Management.createIdentifierColumn(t('accessLog.columns.clientIp'), 'client_ip', 160),
    Management.createTechnicalColumn(t('accessLog.columns.userAgent'), 'user_agent', 280),
    Management.createTimeColumn(t('accessLog.columns.occurredAt'), 'occurred_at', 176),
    Management.createActionColumn(t('accessLog.columns.operation'), 148, 'center', 'operation'),
  ];

  return Management.resolveManagedColumns(allColumns, props.visibleColumnKeys, ['operation']);
});

function statusTheme(statusCode: number) {
  if (statusCode >= 500) {
    return 'danger';
  }
  if (statusCode >= 400) {
    return 'warning';
  }
  return 'success';
}

function accessRow(row: unknown) {
  return row as AccessLogItem;
}

function rowActions(row: AccessLogItem): AccessLogRowAction[] {
  return [
    {
      fallbackLabel: t('accessLog.actions.detail'),
      label: t('accessLog.actions.detail'),
      testId: `access-log-detail-${row.id}`,
      value: 'detail',
    },
    {
      fallbackLabel: t('accessLog.actions.copyRequestId'),
      label: t('accessLog.actions.copyRequestId'),
      value: 'copy-request-id',
    },
    {
      fallbackLabel: t('accessLog.actions.copyPath'),
      label: t('accessLog.actions.copyPath'),
      value: 'copy-path',
    },
    {
      fallbackLabel: t('accessLog.actions.viewRelatedAppLogs'),
      label: t('accessLog.actions.viewRelatedAppLogs'),
      value: 'view-app-log',
    },
    {
      fallbackLabel: t('accessLog.actions.viewRelatedAuditEvents'),
      label: t('accessLog.actions.viewRelatedAuditEvents'),
      value: 'view-audit',
    },
  ];
}

function handleRowAction(action: string, row: AccessLogItem) {
  if (action === 'detail') {
    emit('detail', row);
    return;
  }
  if (action === 'copy-request-id') {
    void copyAccessLogValue(row.request_id, t);
    return;
  }
  if (action === 'copy-path') {
    void copyAccessLogValue(row.path, t);
    return;
  }
  if (action === 'view-app-log') {
    emit('view-app-log', row);
    return;
  }
  if (action === 'view-audit') {
    emit('view-audit', row);
  }
}

void LogIdText;
void TableActionMenu;
void emit;
</script>
<style scoped lang="less">
@import '@/shared/observability/log-table-cells.less';

.log-table-stack-cells();

.duration-danger {
  color: var(--td-error-color);
  font-weight: 600;
}
</style>
