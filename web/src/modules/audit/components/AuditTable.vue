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

    <div ref="tableHostRef" class="audit-log-table__host" :data-table-mode="tableWidthPolicy.mode">
      <t-table
        row-key="id"
        :columns="columns"
        :data="rows"
        :loading="loading"
        table-layout="fixed"
        :table-content-width="tableWidthPolicy.tableContentWidth"
        cell-empty-content="-"
        hover
        @row-click="handleRowClick"
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
          <log-id-text
            :display-value="requestIdForRecord(row)"
            :tooltip="requestIdForRecord(row)"
            v-bind="technicalCopyLabels"
          />
        </template>

        <template #session_id="{ row }">
          <log-id-text
            :display-value="row.session_id || '-'"
            :tooltip="row.session_id || '-'"
            v-bind="technicalCopyLabels"
          />
        </template>

        <template #ip="{ row }">
          <log-id-text :display-value="row.ip || '-'" :tooltip="row.ip || '-'" v-bind="technicalCopyLabels" />
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
            :actions="rowActions(row)"
            :more-label="t('audit.logList.more')"
            :more-label-fallback="t('audit.logList.more')"
            @action="(action) => handleRowAction(action, row)"
          />
        </template>

        <template #empty>
          <div class="table-empty-state">
            <t-empty :title="t('audit.logList.emptyTitle')" :description="t('audit.logList.emptyDescription')" />
          </div>
        </template>
      </t-table>
    </div>

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
  createActionColumn,
  createIdentifierColumn,
  createMainTextColumn,
  createStatusColumn,
  createTechnicalColumn,
  createTimeColumn,
  ManagementTableCard,
  ManagementTablePagination,
  resolveManagedColumns,
  resolveTableWidthPolicy,
  TableActionMenu,
  useTableHostWidth,
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
import { copyAuditRequestId } from '../shared/request-id-copy';
import type { AuditLogListItem } from '../types/audit';

type AuditRowAction = {
  fallbackLabel: string;
  label: string;
  testId?: string;
  value: 'copy-request-id' | 'detail' | 'view-access-log' | 'view-app-log' | 'view-security-event';
};

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
  (e: 'view-access-log', row: AuditLogListItem): void;
  (e: 'view-app-log', row: AuditLogListItem): void;
  (e: 'view-security-event', row: AuditLogListItem): void;
}>();

const { t, locale } = useI18n();
const technicalCopyLabels = computed(() => ({
  copyable: true,
  copyLabel: t('audit.logList.drawer.actions.copyRequestId'),
  copySuccessLabel: t('audit.logList.drawer.actions.copyRequestIdSuccess'),
  copyFailLabel: t('audit.logList.drawer.actions.copyRequestIdFail'),
}));

const current = defineModel<number>('current', { required: true });
const pageSize = defineModel<number>('pageSize', { required: true });

const columns = computed<TdBaseTableProps['columns']>(() => {
  void locale.value;

  const allColumns: TdBaseTableProps['columns'] = [
    createMainTextColumn(t('audit.logList.columns.action'), 'action', 260),
    createIdentifierColumn(t('audit.logList.columns.actor'), 'actor', 168),
    createIdentifierColumn(t('audit.logList.columns.resource'), 'resource', 208),
    createTechnicalColumn(t('audit.logList.columns.correlation'), 'correlation', 248),
    createTechnicalColumn(t('audit.logList.columns.sessionId'), 'session_id', 220),
    createIdentifierColumn(t('audit.logList.columns.ip'), 'ip', 160),
    createStatusColumn(t('audit.logList.columns.result'), 'result', 132),
    createStatusColumn(t('audit.logList.columns.risk'), 'risk', 120),
    createTimeColumn(t('audit.logList.columns.createdAt'), 'created_at', 200),
    createActionColumn(t('audit.logList.columns.operation'), 156, 'center', 'operation'),
  ];

  return resolveManagedColumns(allColumns, props.visibleColumnKeys, ['operation']);
});

const { tableHostRef, tableHostWidth } = useTableHostWidth(() => columns.value);

const tableWidthPolicy = computed(() => resolveTableWidthPolicy(columns.value, tableHostWidth.value));

function emitPageChange() {
  emit('page-change');
}

function handleRowClick(context: { row: unknown }) {
  emit('detail', context.row as AuditLogListItem);
}

function rowActions(row: AuditLogListItem): AuditRowAction[] {
  return [
    {
      fallbackLabel: t('audit.logList.detail'),
      label: t('audit.logList.detail'),
      testId: `audit-log-detail-${row.id}`,
      value: 'detail',
    },
    {
      fallbackLabel: t('audit.logList.drawer.actions.copyRequestId'),
      label: t('audit.logList.drawer.actions.copyRequestId'),
      value: 'copy-request-id',
    },
    {
      fallbackLabel: t('audit.logList.actions.viewAccessLog'),
      label: t('audit.logList.actions.viewAccessLog'),
      value: 'view-access-log',
    },
    {
      fallbackLabel: t('audit.logList.actions.viewAppLog'),
      label: t('audit.logList.actions.viewAppLog'),
      value: 'view-app-log',
    },
    {
      fallbackLabel: t('audit.logList.actions.viewSecurityEvent'),
      label: t('audit.logList.actions.viewSecurityEvent'),
      value: 'view-security-event',
    },
  ];
}

async function copyRequestId(row: AuditLogListItem) {
  await copyAuditRequestId(requestIdForRecord(row), t, { warnWhenMissing: true });
}

function handleRowAction(action: string, row: AuditLogListItem) {
  if (action === 'detail') {
    emit('detail', row);
    return;
  }
  if (action === 'copy-request-id') {
    void copyRequestId(row);
    return;
  }
  if (action === 'view-access-log') {
    emit('view-access-log', row);
    return;
  }
  if (action === 'view-app-log') {
    emit('view-app-log', row);
    return;
  }
  if (action === 'view-security-event') {
    emit('view-security-event', row);
  }
}

void TableActionMenu;
</script>
<style scoped lang="less">
@import '@/shared/observability/log-table-cells.less';

.table-head {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
}

.table-head__summary,
.table-head__description {
  color: var(--td-text-color-secondary);
  margin: 0;
}

.log-table-stack-cells();

.table-empty-state {
  padding: var(--graft-density-gap-24) 0 var(--graft-density-gap-8);
}

.audit-log-table__host {
  max-width: 100%;
  min-width: 0;
  overflow-x: hidden;
}

.audit-log-table__host[data-table-mode='scroll'] {
  overflow-x: auto;
}

@media (width <= 768px) {
  .table-head {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
