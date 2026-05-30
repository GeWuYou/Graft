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
          <span v-if="accessLogPathSecondary(row)" class="stack-cell__secondary">
            {{ t('accessLog.path.routeTemplateValue', { route: accessLogPathSecondary(row) }) }}
          </span>
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
          <strong>{{ accessLogUserPrimary(row, t) }}</strong>
          <span class="stack-cell__secondary">{{ accessLogUserSecondary(row, t) }}</span>
        </div>
      </template>
      <template #request_id="{ row }">
        <log-id-text :display-value="row.request_id || '-'" :tooltip="row.request_id || '-'" />
      </template>
      <template #occurred_at="{ row }">
        <span>{{ formatCompactDateTime(row.occurred_at, locale) }}</span>
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
  resolveManagedColumns,
  TableActionMenu,
} from '@/shared/components/management';
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

const columns = computed<TdBaseTableProps['columns']>(() => {
  void locale.value;
  const allColumns: TdBaseTableProps['columns'] = [
    createTimeColumn(t('accessLog.columns.occurredAt'), 'occurred_at', 176),
    createTextColumn(t('accessLog.columns.method'), 'method', { width: 110, fixed: 'left' }),
    createTextColumn(t('accessLog.columns.path'), 'path', { minWidth: 320 }),
    createTextColumn(t('accessLog.columns.statusCode'), 'status_code', { width: 110 }),
    createTextColumn(t('accessLog.columns.durationMs'), 'duration_ms', { width: 120 }),
    createTextColumn(t('accessLog.columns.user'), 'user', { width: 190 }),
    createTextColumn(t('accessLog.columns.requestId'), 'request_id', { width: 240 }),
    createActionColumn(t('accessLog.columns.operation'), 104),
  ];

  return resolveManagedColumns(allColumns, props.visibleColumnKeys, ['operation']);
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

void LogIdText;
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

.table-empty-state {
  padding: 24px 0 8px;
}

.duration-danger {
  color: var(--td-error-color);
  font-weight: 600;
}
</style>
