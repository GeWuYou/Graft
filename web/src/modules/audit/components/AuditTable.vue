<template>
  <management-paged-table
    v-model:current="current"
    v-model:page-size="pageSize"
    :cards-visible="true"
    :columns="columns"
    :description="description"
    :empty-description="t('audit.logList.emptyDescription')"
    :empty-title="t('audit.logList.emptyTitle')"
    :footer-summary="footerSummary"
    :head-label="summary || t('audit.logList.title')"
    :loading="loading"
    presentation="log"
    row-key="id"
    :rows="rows"
    :summary="summary"
    :total="total"
    @page-change="emitPageChange"
    @row-click="handleRowClick"
  >
    <template #head>
      <div class="table-head">
        <div>
          <p v-if="summary" class="table-head__summary">{{ summary }}</p>
          <p v-if="description" class="table-head__description">{{ description }}</p>
        </div>
        <t-tag v-if="localFilterActive" theme="default" variant="light-outline" size="small">
          {{ t('audit.logList.currentPageFiltered') }}
        </t-tag>
      </div>
    </template>
    <template v-if="$slots.toolbar" #toolbar><slot name="toolbar" /></template>

    <template #action="{ row }">
      <div class="stack-cell">
        <strong>{{ actionTitle(row, t) }}</strong>
        <span v-if="actionCategoryLabel(row, t) !== actionTitle(row, t)" class="stack-cell__secondary">
          {{ actionCategoryLabel(row, t) }}
        </span>
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
        <span v-if="reasonForRecord(row, t) !== resourceLabel(row, t)" class="stack-cell__secondary">
          {{ reasonForRecord(row, t) }}
        </span>
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
    <template #ip="{ row }"
      ><log-id-text :display-value="row.ip || '-'" :tooltip="row.ip || '-'" v-bind="technicalCopyLabels"
    /></template>
    <template #result="{ row }"
      ><t-tag :theme="resultTone(row)" variant="light-outline" size="small">{{ resultLabel(row, t) }}</t-tag></template
    >
    <template #risk="{ row }"
      ><t-tag :theme="riskTone(row)" variant="light-outline" size="small">{{ riskLabel(row, t) }}</t-tag></template
    >
    <template #created_at="{ row }"
      ><span>{{ formatAuditTimestamp(row.created_at, locale) }}</span></template
    >
    <template #operation="{ row }">
      <table-action-menu
        :actions="rowActions(row)"
        :more-label="t('audit.logList.more')"
        :more-label-fallback="t('audit.logList.more')"
        @action="(action) => handleRowAction(action, row)"
      />
    </template>

    <template #cards>
      <article v-for="row in rows" :key="row.id" class="audit-log-card" @click="emit('detail', row)">
        <div class="audit-log-card__header">
          <strong class="audit-log-card__title">{{ actionTitle(row, t) }}</strong>
        </div>
        <p class="audit-log-card__target">{{ resourceLabel(row, t) }}</p>
        <div class="audit-log-card__badges">
          <t-tag :theme="resultTone(row)" variant="light-outline" size="small">{{ resultLabel(row, t) }}</t-tag>
          <t-tag :theme="riskTone(row)" variant="light-outline" size="small">{{ riskLabel(row, t) }}</t-tag>
        </div>
        <time class="audit-log-card__time">{{ formatAuditTimestamp(row.created_at, locale) }}</time>
        <p class="audit-log-card__actor">{{ t('audit.logList.columns.actor') }}: {{ actorLabel(row, t) }}</p>
        <div class="audit-log-card__request" @click.stop>
          <span>{{ t('audit.logList.columns.correlation') }}</span>
          <log-id-text
            :display-value="requestIdForRecord(row)"
            :tooltip="requestIdForRecord(row)"
            v-bind="technicalCopyLabels"
          />
        </div>
        <div class="audit-log-card__actions" @click.stop>
          <table-action-menu
            :actions="rowActions(row)"
            :more-label="t('audit.logList.more')"
            :more-label-fallback="t('audit.logList.more')"
            @action="(action) => handleRowAction(action, row)"
          />
        </div>
      </article>
    </template>
  </management-paged-table>
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
  ManagementPagedTable,
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
const current = defineModel<number>('current', { required: true });
const pageSize = defineModel<number>('pageSize', { required: true });
const technicalCopyLabels = computed(() => ({
  copyable: true,
  copyLabel: t('audit.logList.drawer.actions.copyRequestId'),
  copySuccessLabel: t('audit.logList.drawer.actions.copyRequestIdSuccess'),
  copyFailLabel: t('audit.logList.drawer.actions.copyRequestIdFail'),
}));
const columns = computed<TdBaseTableProps['columns']>(() => {
  void locale.value;
  return resolveManagedColumns(
    [
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
    ],
    props.visibleColumnKeys,
    ['operation'],
  );
});
function emitPageChange() {
  emit('page-change');
}
function handleRowClick(row: unknown) {
  emit('detail', row as AuditLogListItem);
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
  if (action === 'detail') emit('detail', row);
  else if (action === 'copy-request-id') void copyRequestId(row);
  else if (action === 'view-access-log') emit('view-access-log', row);
  else if (action === 'view-app-log') emit('view-app-log', row);
  else if (action === 'view-security-event') emit('view-security-event', row);
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
.table-head__description,
.audit-log-card__time,
.audit-log-card__actor,
.audit-log-card__request > span {
  color: var(--td-text-color-secondary);
  margin: 0;
}
.log-table-stack-cells();

.audit-log-card {
  .log-card-surface(var(--td-component-stroke));
}

.audit-log-card__header,
.audit-log-card__badges,
.audit-log-card__request,
.audit-log-card__actions {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-8);
  justify-content: space-between;
  min-width: 0;
}

.audit-log-card__title {
  -webkit-box-orient: vertical;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  overflow: hidden;
}

.audit-log-card__badges {
  flex: none;
}

.audit-log-card__target {
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.audit-log-card__actor {
  margin: 0;
}

.audit-log-card__actions {
  justify-content: flex-end;
  justify-self: end;
}
</style>
