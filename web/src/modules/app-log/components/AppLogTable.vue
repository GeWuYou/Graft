<template>
  <management-table-card>
    <template #head>
      <section class="table-head" aria-label="app-log-table-head">
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
      <template #occurred_at="{ row }">
        <span>{{ formatCompactDateTime(row.occurred_at, locale) }}</span>
      </template>
      <template #severity="{ row }">
        <t-tag :theme="appLogSeverityTheme(row.severity)" variant="light-outline" size="small">
          {{ row.severity.toUpperCase() }}
        </t-tag>
      </template>
      <template #message="{ row }">
        <div class="stack-cell">
          <strong>{{ row.message }}</strong>
          <span v-if="row.error" class="stack-cell__secondary">{{ row.error }}</span>
        </div>
      </template>
      <template #operation="{ row }">
        <span>{{ appLogOperationText(row, t) }}</span>
      </template>
      <template #correlation="{ row }">
        <log-id-text :display-value="appLogCorrelationText(row, t)" :tooltip="appLogCorrelationText(row, t)" />
      </template>
      <template #fields="{ row }">
        <span>{{ appLogFieldsCount(row) }}</span>
      </template>
      <template #actions="{ row }">
        <table-action-menu
          :actions="[{ label: t('appLog.actions.detail'), testId: 'app-log-detail', value: 'detail' }]"
          :more-label="t('appLog.actions.detail')"
          @action="() => $emit('detail', row)"
        />
      </template>
      <template #empty>
        <div class="table-empty-state">
          <t-empty :title="t('appLog.page.emptyTitle')" :description="emptyDescription" />
        </div>
      </template>
    </t-table>
    <template #footer>
      <div class="app-log-pagination">
        <span>{{ footerSummary }}</span>
        <t-pagination
          v-model:current="current"
          v-model:page-size="pageSize"
          :total="total"
          :page-size-options="paginationSizes"
          @change="$emit('page-change')"
        />
      </div>
    </template>
  </management-table-card>
</template>
<script setup lang="ts">
import type { TdBaseTableProps } from 'tdesign-vue-next';
import { useI18n } from 'vue-i18n';

import {
  calculateTableContentWidth,
  formatCompactDateTime,
  ManagementTableCard,
  TableActionMenu,
} from '@/shared/components/management';
import { LogIdText } from '@/shared/observability';

import {
  appLogCorrelationText,
  appLogFieldsCount,
  appLogOperationText,
  appLogSeverityTheme,
} from '../shared/presentation';
import type { AppLogItem } from '../types/app-log';

type AppLogTableEmits = {
  detail: [row: AppLogItem];
  'page-change': [];
};

defineProps<{
  description: string;
  emptyDescription: string;
  footerSummary: string;
  loading?: boolean;
  rows: AppLogItem[];
  summary: string;
  total: number;
}>();

defineEmits<AppLogTableEmits>();

const { t, locale } = useI18n();
const paginationSizes = [10, 20, 50, 100];

const columns = buildAppLogColumns();
const tableContentWidth = calculateTableContentWidth(columns);
const current = defineModel<number>('current', { required: true });
const pageSize = defineModel<number>('pageSize', { required: true });

function buildAppLogColumns(): TdBaseTableProps['columns'] {
  const baseColumn = { ellipsis: { theme: 'default' as const, placement: 'top-left' as const } };
  const specs = [
    ['occurred_at', 'appLog.columns.occurredAt', { width: 176, align: 'center' as const }],
    ['severity', 'appLog.columns.severity', { width: 110 }],
    ['component', 'appLog.columns.component', { minWidth: 210 }],
    ['operation', 'appLog.columns.operation', { minWidth: 160 }],
    ['message', 'appLog.columns.message', { minWidth: 360 }],
    ['correlation', 'appLog.columns.correlation', { width: 240 }],
    ['fields', 'appLog.columns.fields', { width: 90, align: 'center' as const }],
    ['actions', 'appLog.columns.actions', { width: 104, align: 'center' as const, fixed: 'right' as const }],
  ] as const;

  void locale.value;
  return specs.map(([colKey, titleKey, options]) => ({
    ...baseColumn,
    title: t(titleKey),
    colKey,
    ...options,
    ...(colKey === 'actions' ? { ellipsis: false } : {}),
  }));
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

.app-log-pagination {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: flex;
  font: var(--td-font-body-small);
  gap: 16px;
  justify-content: space-between;
  min-height: 60px;
  width: 100%;
}

@media (width <= 768px) {
  .app-log-pagination {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
