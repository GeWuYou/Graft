<template>
  <management-table-card>
    <template #head>
      <section class="table-head" aria-label="access-log-table-head">
        <p class="table-head__description">{{ description }}</p>
        <p class="table-head__summary">{{ summary }}</p>
      </section>
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
      <template #method="{ row }">
        <t-tag theme="primary" variant="light-outline" size="small">{{ row.method }}</t-tag>
      </template>
      <template #path="{ row }">
        <div class="stack-cell">
          <strong>{{ row.path }}</strong>
          <span class="stack-cell__secondary">{{
            row.trace_id ? `Trace ${truncateMiddle(row.trace_id, 24)}` : row.route || '-'
          }}</span>
        </div>
      </template>
      <template #status_code="{ row }">
        <t-tag :theme="statusTheme(row.status_code)" variant="light-outline" size="small">
          {{ row.status_code }}
        </t-tag>
      </template>
      <template #duration_ms="{ row }">
        <span :class="{ 'duration-danger': row.duration_ms >= 3000 }">{{ row.duration_ms }} ms</span>
      </template>
      <template #user="{ row }">
        <div class="stack-cell">
          <strong>{{ row.username || '-' }}</strong>
          <span class="stack-cell__secondary">{{ row.user_id ?? '-' }}</span>
        </div>
      </template>
      <template #request_id="{ row }">
        <strong class="table-mono" :title="row.request_id">{{ truncateMiddle(row.request_id, 22) }}</strong>
      </template>
      <template #operation="{ row }">
        <table-action-menu
          :actions="[{ label: t('accessLog.actions.detail'), testId: 'access-log-detail', value: 'detail' }]"
          :more-label="t('accessLog.actions.detail')"
          @action="() => $emit('detail', row)"
        />
      </template>
      <template #empty>
        <div class="table-empty-state">
          <t-empty :title="t('accessLog.page.emptyTitle')" :description="emptyDescription" />
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
          @change="$emit('page-change')"
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
  createTextColumn,
  createTimeColumn,
  formatCompactDateTime,
  ManagementTableCard,
  ManagementTablePagination,
  TableActionMenu,
} from '@/shared/components/management';

import type { AccessLogItem } from '../types/access-log';

defineProps<{
  description: string;
  emptyDescription: string;
  footerSummary: string;
  loading?: boolean;
  rows: AccessLogItem[];
  summary: string;
  total: number;
}>();

defineEmits<{
  (e: 'detail', row: AccessLogItem): void;
  (e: 'page-change'): void;
}>();

const current = defineModel<number>('current', { required: true });
const pageSize = defineModel<number>('pageSize', { required: true });

const { t, locale } = useI18n();

const columns = computed<TdBaseTableProps['columns']>(() => {
  void locale.value;
  return [
    createTimeColumn(t('accessLog.columns.occurredAt'), 'occurred_at', 176),
    createTextColumn(t('accessLog.columns.method'), 'method', { width: 110, fixed: 'left' }),
    createTextColumn(t('accessLog.columns.path'), 'path', { minWidth: 320 }),
    createTextColumn(t('accessLog.columns.statusCode'), 'status_code', { width: 110 }),
    createTextColumn(t('accessLog.columns.durationMs'), 'duration_ms', { width: 120 }),
    createTextColumn(t('accessLog.columns.user'), 'user', { width: 160 }),
    createTextColumn(t('accessLog.columns.requestId'), 'request_id', { width: 220 }),
    createActionColumn(t('accessLog.columns.operation'), 96),
  ];
});

const tableContentWidth = computed(() => calculateTableContentWidth(columns.value));

function statusTheme(statusCode: number) {
  if (statusCode >= 500) {
    return 'danger';
  }
  if (statusCode >= 400) {
    return 'warning';
  }
  return 'success';
}

function truncateMiddle(value: string, maxLength: number) {
  if (!value || value.length <= maxLength) {
    return value || '-';
  }

  const prefixLength = Math.max(6, Math.floor((maxLength - 1) / 2));
  const suffixLength = Math.max(6, maxLength - prefixLength - 1);
  return `${value.slice(0, prefixLength)}…${value.slice(-suffixLength)}`;
}

void formatCompactDateTime;
</script>
<style scoped lang="less">
.table-head__summary,
.table-head__description,
.stack-cell__secondary {
  color: var(--td-text-color-secondary);
  margin: 0;
}

.stack-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.table-mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

.table-empty-state {
  padding: 24px 0 8px;
}

.duration-danger {
  color: var(--td-error-color);
  font-weight: 600;
}
</style>
