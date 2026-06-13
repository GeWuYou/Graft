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
  >
    <template v-if="$slots.toolbar" #toolbar>
      <slot name="toolbar" />
    </template>
    <template #method="{ row }">
      <t-tag theme="primary" variant="light-outline" size="small">{{ accessRow(row).method }}</t-tag>
    </template>
    <template #path="{ row }">
      <div class="stack-cell">
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
      <log-id-text :display-value="accessRow(row).request_id || '-'" :tooltip="accessRow(row).request_id || '-'" />
    </template>
    <template #started_at="{ row }">
      <span>{{ Management.formatCompactDateTime(accessRow(row).started_at, locale) }}</span>
    </template>
    <template #occurred_at="{ row }">
      <span>{{ Management.formatCompactDateTime(accessRow(row).occurred_at, locale) }}</span>
    </template>
    <template #operation="{ row }">
      <management-table-action-menu
        :actions="[{ label: t('accessLog.actions.detail'), testId: 'access-log-detail', value: 'detail' }]"
        :more-label="t('accessLog.actions.detail')"
        @action="() => $emit('detail', accessRow(row))"
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

import { accessLogPathSecondary, accessLogUserPrimary, accessLogUserSecondary } from '../shared/presentation';
import type { AccessLogItem } from '../types/access-log';

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

defineEmits<{
  (e: 'detail', row: AccessLogItem): void;
  (e: 'page-change'): void;
}>();

const current = defineModel<number>('current', { required: true });
const pageSize = defineModel<number>('pageSize', { required: true });

const { t, locale } = useI18n();
const cellSlotNames = [
  'method',
  'path',
  'status_code',
  'duration_ms',
  'user',
  'request_id',
  'started_at',
  'occurred_at',
  'operation',
];

const columns = computed<TdBaseTableProps['columns']>(() => {
  void locale.value;
  const allColumns: TdBaseTableProps['columns'] = [
    ...Management.createConfiguredColumns([
      { kind: 'time', key: 'started_at', title: t('accessLog.columns.startedAt'), width: 176 },
      { key: 'method', title: t('accessLog.columns.method'), config: { width: 110, fixed: 'left' } },
      { key: 'path', title: t('accessLog.columns.path'), config: { minWidth: 320 } },
      { key: 'status_code', title: t('accessLog.columns.statusCode'), config: { width: 110 } },
      { key: 'duration_ms', title: t('accessLog.columns.durationMs'), config: { width: 120 } },
      { key: 'user', title: t('accessLog.columns.user'), config: { width: 190 } },
      { key: 'request_id', title: t('accessLog.columns.requestId'), config: { width: 240 } },
      { kind: 'time', key: 'occurred_at', title: t('accessLog.columns.occurredAt'), width: 176 },
    ]),
    Management.createActionColumn(t('accessLog.columns.operation'), 104),
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

void LogIdText;
const ManagementTableActionMenu = Management.TableActionMenu;
</script>
<style scoped lang="less">
.stack-cell__secondary {
  color: var(--td-text-color-secondary);
  margin: 0;
}

.stack-cell {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
}

.duration-danger {
  color: var(--td-error-color);
  font-weight: 600;
}
</style>
