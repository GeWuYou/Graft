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
    :empty-title="t('appLog.page.emptyTitle')"
    :footer-summary="footerSummary"
    head-label="app-log-table-head"
    :loading="loading"
    :rows="rows"
    :summary="summary"
    :total="total"
    @page-change="$emit('page-change')"
  >
    <template #occurred_at="{ row }">
      <span>{{ formatCompactDateTime(appLogRow(row).occurred_at, locale) }}</span>
    </template>
    <template #severity="{ row }">
      <t-tag :theme="appLogSeverityTheme(appLogRow(row).severity)" variant="light-outline" size="small">
        {{ appLogRow(row).severity.toUpperCase() }}
      </t-tag>
    </template>
    <template #message="{ row }">
      <div class="stack-cell">
        <strong>{{ appLogRow(row).message }}</strong>
        <span v-if="appLogRow(row).error" class="stack-cell__secondary">{{ appLogRow(row).error }}</span>
      </div>
    </template>
    <template #operation="{ row }">
      <span>{{ appLogOperationText(appLogRow(row), t) }}</span>
    </template>
    <template #correlation="{ row }">
      <log-id-text
        :display-value="appLogCorrelationText(appLogRow(row), t)"
        :tooltip="appLogCorrelationText(appLogRow(row), t)"
      />
    </template>
    <template #fields="{ row }">
      <span>{{ appLogFieldsCount(appLogRow(row)) }}</span>
    </template>
    <template #actions="{ row }">
      <table-action-menu
        :actions="[{ label: t('appLog.actions.detail'), testId: 'app-log-detail', value: 'detail' }]"
        :more-label="t('appLog.actions.detail')"
        @action="() => $emit('detail', appLogRow(row))"
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
  createConfiguredColumns,
  formatCompactDateTime,
  resolveManagedColumns,
  TableActionMenu,
} from '@/shared/components/management';
import { AdvancedQueryPagedTable } from '@/shared/components/query-list';
import { LogIdText } from '@/shared/observability';

import {
  appLogCorrelationText,
  appLogFieldsCount,
  appLogOperationText,
  appLogSeverityTheme,
} from '../shared/presentation';
import type { AppLogItem } from '../types/app-log';

const props = defineProps<{
  description: string;
  emptyDescription: string;
  footerSummary: string;
  loading?: boolean;
  rows: AppLogItem[];
  summary: string;
  total: number;
  visibleColumnKeys?: string[];
}>();

const current = defineModel<number>('current', { required: true });
const { t, locale } = useI18n();
const emit = defineEmits<{
  (e: 'page-change'): void;
  (e: 'detail', row: AppLogItem): void;
}>();
const pageSize = defineModel<number>('pageSize', { required: true });
const cellSlotNames = ['occurred_at', 'severity', 'message', 'operation', 'correlation', 'fields', 'actions'];

const columns = computed<TdBaseTableProps['columns']>(() => {
  void locale.value;
  const allColumns: TdBaseTableProps['columns'] = [
    ...createConfiguredColumns([
      { kind: 'time', key: 'occurred_at', title: t('appLog.columns.occurredAt'), width: 176 },
      { key: 'severity', title: t('appLog.columns.severity'), config: { width: 110 } },
      { key: 'component', title: t('appLog.columns.component'), config: { minWidth: 210 } },
      { key: 'operation', title: t('appLog.columns.operation'), config: { minWidth: 160 } },
      { key: 'message', title: t('appLog.columns.message'), config: { minWidth: 360 } },
      { key: 'correlation', title: t('appLog.columns.correlation'), config: { width: 240 } },
      { key: 'fields', title: t('appLog.columns.fields'), config: { width: 90, align: 'center' } },
    ]),
    createActionColumn(t('appLog.columns.actions'), 104),
  ];

  return resolveManagedColumns(allColumns, props.visibleColumnKeys, ['actions']);
});

function appLogRow(row: unknown) {
  return row as AppLogItem;
}

void LogIdText;
void emit;
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
</style>
