<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <management-table-card>
    <template #head>
      <div class="table-head">
        <div>
          <p class="table-head__summary">{{ summary }}</p>
          <p class="table-head__description">{{ description }}</p>
        </div>
        <t-tag v-if="localFilterActive" theme="default" variant="light-outline" size="small">
          {{ t('audit.logList.currentPageFiltered') }}
        </t-tag>
      </div>
    </template>
    <template v-if="$slots.toolbar" #toolbar>
      <slot name="toolbar" />
    </template>

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
      <template #action="{ row }">
        <div class="stack-cell">
          <strong>{{ actionTitle(row, t) }}</strong>
          <span class="stack-cell__secondary">{{ actionCategoryLabel(row, t) }}</span>
        </div>
      </template>

      <template #actor="{ row }">
        <div class="stack-cell">
          <strong>{{ actorLabel(row, t) }}</strong>
          <span class="stack-cell__secondary">{{ actorSecondaryLabel(row) }}</span>
        </div>
      </template>

      <template #resource="{ row }">
        <div class="stack-cell">
          <strong>{{ resourceLabel(row, t) }}</strong>
          <span class="stack-cell__secondary">{{ reasonForRecord(row, t) }}</span>
        </div>
      </template>

      <template #correlation="{ row }">
        <log-id-text :display-value="requestIdForRecord(row)" :tooltip="requestIdForRecord(row)" />
      </template>

      <template #result="{ row }">
        <t-tag :theme="resultTone(row)" variant="light-outline" size="small">
          {{ resultLabel(row, t) }}
        </t-tag>
      </template>

      <template #risk="{ row }">
        <t-tag :theme="riskTone(row)" variant="light-outline" size="small">
          {{ riskLabel(row, t) }}
        </t-tag>
      </template>

      <template #created_at="{ row }">
        <span>{{ formatAuditTimestamp(row.created_at, locale) }}</span>
      </template>

      <template #operation="{ row }">
        <table-action-menu
          :actions="[{ label: t('audit.logList.detail'), testId: 'audit-detail', value: 'detail' }]"
          :more-label="t('audit.logList.more')"
          @action="() => $emit('detail', row)"
        />
      </template>

      <template #empty>
        <div class="table-empty-state">
          <t-empty :title="t('audit.logList.emptyTitle')" :description="t('audit.logList.emptyDescription')" />
        </div>
      </template>
    </t-table>

    <template #footer>
      <management-table-pagination :summary="footerSummary">
        <t-pagination
          v-model:current="current"
          v-model:page-size="pageSize"
          :total="total"
          :page-size-options="[10, 20, 50, 100]"
          @change="emitPageChange"
        />
      </management-table-pagination>
    </template>
  </management-table-card>
</template>
<script setup lang="ts">
import type { TdBaseTableProps } from 'tdesign-vue-next';
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import {
  calculateTableContentWidth,
  createActionColumn,
  createStatusColumn,
  createTextColumn,
  createTimeColumn,
  ManagementTableCard,
  ManagementTablePagination,
  resolveManagedColumns,
  TableActionMenu,
} from '@/shared/components/management';
import { LogIdText } from '@/shared/observability';

import {
  actionCategoryLabel,
  actionTitle,
  actorLabel,
  actorSecondaryLabel,
  formatAuditTimestamp,
  reasonForRecord,
  requestIdForRecord,
  resourceLabel,
  resultLabel,
  resultTone,
  riskLabel,
  riskTone,
} from '../shared/presentation';
import type { AuditLogListItem } from '../types/audit';

const props = defineProps<{
  description?: string;
  footerSummary: string;
  loading?: boolean;
  localFilterActive?: boolean;
  rows: AuditLogListItem[];
  summary?: string;
  total: number;
  visibleColumnKeys?: string[];
}>();

const emit = defineEmits<{
  (e: 'detail', row: AuditLogListItem): void;
  (e: 'update:current', value: number): void;
  (e: 'update:pageSize', value: number): void;
  (e: 'page-change'): void;
}>();

const { t, locale } = useI18n();

const current = defineModel<number>('current', { required: true });
const pageSize = defineModel<number>('pageSize', { required: true });

const columns = computed<TdBaseTableProps['columns']>(() => {
  void locale.value;

  const allColumns: TdBaseTableProps['columns'] = [
    createTextColumn(t('audit.logList.columns.action'), 'action', { fixed: 'left', minWidth: 260 }),
    createTextColumn(t('audit.logList.columns.actor'), 'actor', { width: 180 }),
    createTextColumn(t('audit.logList.columns.resource'), 'resource', { width: 220 }),
    createTextColumn(t('audit.logList.columns.correlation'), 'correlation', { width: 240 }),
    createStatusColumn(t('audit.logList.columns.result'), 'result', 132),
    createStatusColumn(t('audit.logList.columns.risk'), 'risk', 120),
    createTimeColumn(t('audit.logList.columns.createdAt'), 'created_at', 220),
    createActionColumn(t('components.commonTable.operation'), 104),
  ];

  return resolveManagedColumns(allColumns, props.visibleColumnKeys, ['operation']);
});

const tableContentWidth = computed(() => calculateTableContentWidth(columns.value));

function emitPageChange() {
  emit('page-change');
}
</script>
<style scoped lang="less">
.table-head {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
}

.table-head__summary,
.table-head__description,
.stack-cell__secondary {
  color: var(--td-text-color-secondary);
  margin: 0;
}

.stack-cell {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.table-empty-state {
  padding: var(--graft-density-gap-24) 0 var(--graft-density-gap-8);
}

@media (width <= 768px) {
  .table-head {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
