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
          <strong>{{ row.action }}</strong>
          <span class="stack-cell__secondary">{{ sourceLabel(row, t) }}</span>
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
        <div class="stack-cell">
          <strong>{{ requestIdForRecord(row) }}</strong>
          <span class="stack-cell__secondary">{{ traceIdForRecord(row) }}</span>
        </div>
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
        <span>{{ formatAuditTimestamp(row.created_at) }}</span>
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
          :page-size-options="[10, 20, 50]"
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
  TableActionMenu,
} from '@/shared/components/management';

import {
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
  sourceLabel,
  traceIdForRecord,
} from '../shared/presentation';
import type { AuditLogListItem } from '../types/audit';

defineProps<{
  description: string;
  footerSummary: string;
  loading?: boolean;
  localFilterActive?: boolean;
  rows: AuditLogListItem[];
  summary: string;
  total: number;
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

  return [
    createTextColumn(t('audit.logList.columns.action'), 'action', { fixed: 'left', minWidth: 260 }),
    createTextColumn(t('audit.logList.columns.actor'), 'actor', { width: 180 }),
    createTextColumn(t('audit.logList.columns.resource'), 'resource', { width: 220 }),
    createTextColumn(t('audit.logList.columns.correlation'), 'correlation', { width: 220 }),
    createStatusColumn(t('audit.logList.columns.result'), 'result', 110),
    createStatusColumn(t('audit.logList.columns.risk'), 'risk', 120),
    createTimeColumn(t('audit.logList.columns.createdAt'), 'created_at', 168),
    createActionColumn(t('components.commonTable.operation'), 96),
  ];
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
  gap: 12px;
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
  gap: 4px;
}

.table-empty-state {
  padding: 24px 0 8px;
}

@media (width <= 768px) {
  .table-head {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
