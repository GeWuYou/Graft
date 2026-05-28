<template>
  <div class="audit-page" data-page-type="log-audit">
    <management-page-content>
      <management-page-header :title="t('audit.logList.listTitle')" :description="t('audit.logList.hint')">
        <template #eyebrow>{{ t('menu.audit.logs.title') }}</template>
        <template #actions>
          <t-button
            v-permission="AUDIT_PERMISSION_CODE.READ"
            theme="default"
            variant="outline"
            :loading="loading"
            @click="fetchAuditLogs"
          >
            {{ t('audit.logList.refresh') }}
          </t-button>
        </template>
      </management-page-header>

      <management-toolbar>
        <template #filters>
          <t-input
            v-model="filters.action"
            clearable
            class="toolbar__search"
            :placeholder="t('audit.logList.filters.actionPlaceholder')"
          />
          <t-input
            v-model="filters.resource_type"
            clearable
            class="toolbar__select"
            :placeholder="t('audit.logList.filters.resourceTypePlaceholder')"
          />
          <t-input
            v-model="filters.resource_name"
            clearable
            class="toolbar__select"
            :placeholder="t('audit.logList.filters.resourceNamePlaceholder')"
          />
          <t-select
            v-model="filters.successValue"
            clearable
            class="toolbar__select"
            :options="successOptions"
            :placeholder="t('audit.logList.filters.successPlaceholder')"
          />
          <t-date-range-picker
            v-model="createdRange"
            allow-input
            clearable
            class="toolbar__date"
            enable-time-picker
            format="YYYY-MM-DD HH:mm:ss"
            :placeholder="[
              t('audit.logList.filters.createdRangePlaceholder'),
              t('audit.logList.filters.createdRangePlaceholder'),
            ]"
          />
        </template>
        <template #actions>
          <t-button v-permission="AUDIT_PERMISSION_CODE.READ" theme="default" variant="text" @click="resetFilters">
            {{ t('audit.logList.clearFilters') }}
          </t-button>
        </template>
      </management-toolbar>

      <div class="inline-note">
        <p>{{ t('audit.logList.readonlyNotice') }}</p>
        <p>{{ t('audit.logList.factSourceHint') }}</p>
      </div>

      <management-table-card>
        <template #head>
          <div class="table-head">
            <div>
              <p class="table-head__summary">{{ t('audit.logList.summary', { count: rows.length }) }}</p>
              <p class="table-head__description">{{ t('audit.logList.tableHint') }}</p>
            </div>
            <t-button
              v-if="hasActiveFilters"
              v-permission="AUDIT_PERMISSION_CODE.READ"
              theme="default"
              variant="text"
              @click="resetFilters"
            >
              {{ t('audit.logList.clearFilters') }}
            </t-button>
          </div>
        </template>

        <management-empty-state
          v-if="listError && !loading"
          tone="error"
          :title="t('audit.logList.errorTitle')"
          :description="listError"
        >
          <template #actions>
            <t-button theme="primary" variant="outline" @click="fetchAuditLogs">
              {{ t('audit.logList.retry') }}
            </t-button>
          </template>
        </management-empty-state>

        <t-table
          v-else
          row-key="id"
          :data="rows"
          :columns="columns"
          :loading="loading"
          table-layout="fixed"
          :table-content-width="tableContentWidth"
          cell-empty-content="-"
          hover
        >
          <template #action="{ row }">
            <div class="action-cell">
              <strong class="action-cell__primary">{{ auditActionTitle(row) }}</strong>
              <span class="action-cell__secondary">{{ row.resource_type || '-' }}</span>
            </div>
          </template>

          <template #actor="{ row }">
            <div class="stack-cell">
              <strong>{{ actorLabel(row) }}</strong>
              <span class="stack-cell__secondary">{{ row.actor_username || '-' }}</span>
            </div>
          </template>

          <template #resource="{ row }">
            <div class="stack-cell">
              <strong>{{ resourceLabel(row) }}</strong>
              <span class="stack-cell__secondary">{{ resourceSecondaryLabel(row) }}</span>
            </div>
          </template>

          <template #result="{ row }">
            <t-tag :theme="row.success ? 'success' : 'danger'" variant="light-outline" size="small" shape="round">
              {{ row.success ? t('audit.logList.result.success') : t('audit.logList.result.failed') }}
            </t-tag>
          </template>

          <template #created_at="{ row }">
            <span>{{ formatTimestamp(row.created_at) }}</span>
          </template>

          <template #operation="{ row }">
            <table-action-menu
              :actions="[
                {
                  label: t('audit.logList.detail'),
                  testId: 'audit-detail',
                  value: 'detail',
                },
              ]"
              :more-label="t('audit.logList.more')"
              @action="() => openDetailDrawer(row)"
            />
          </template>

          <template #empty>
            <div class="table-empty-state">
              <t-empty :title="t('audit.logList.emptyTitle')" :description="t('audit.logList.emptyDescription')">
                <template #action>
                  <div class="table-empty-state__actions">
                    <t-button
                      v-if="hasActiveFilters"
                      v-permission="AUDIT_PERMISSION_CODE.READ"
                      theme="default"
                      variant="outline"
                      @click="resetFilters"
                    >
                      {{ t('audit.logList.clearFilters') }}
                    </t-button>
                  </div>
                </template>
              </t-empty>
            </div>
          </template>
        </t-table>

        <template #footer>
          <management-table-pagination :summary="t('audit.logList.footerTotal', { count: total })">
            <t-pagination
              v-model:current="pagination.current"
              v-model:page-size="pagination.pageSize"
              :total="total"
              :page-size-options="[10, 20, 50]"
              @change="handlePageChange"
            />
          </management-table-pagination>
        </template>
      </management-table-card>
    </management-page-content>

    <t-drawer
      v-model:visible="detailDrawerVisible"
      :header="t('audit.logList.detailTitle')"
      size="560px"
      placement="right"
      destroy-on-close
    >
      <div v-if="detailRecord" class="drawer-panel audit-detail-panel">
        <div class="detail-section">
          <h4 class="detail-section__title">{{ t('audit.logList.detailSections.basic') }}</h4>
          <div class="detail-grid">
            <div class="detail-item">
              <span class="detail-item__label">{{ t('audit.logList.columns.action') }}</span>
              <span class="detail-item__value">{{ auditActionTitle(detailRecord) }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-item__label">{{ t('audit.logList.columns.result') }}</span>
              <span class="detail-item__value">
                {{ detailRecord.success ? t('audit.logList.result.success') : t('audit.logList.result.failed') }}
              </span>
            </div>
            <div class="detail-item">
              <span class="detail-item__label">{{ t('audit.logList.columns.createdAt') }}</span>
              <span class="detail-item__value">{{ formatTimestamp(detailRecord.created_at) }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-item__label">{{ t('audit.logList.columns.actor') }}</span>
              <span class="detail-item__value">{{ actorLabel(detailRecord) }}</span>
            </div>
            <div class="detail-item detail-item--full">
              <span class="detail-item__label">{{ t('audit.logList.columns.resource') }}</span>
              <span class="detail-item__value">{{ resourceDetailLabel(detailRecord) }}</span>
            </div>
          </div>
        </div>

        <div class="detail-section">
          <h4 class="detail-section__title">{{ t('audit.logList.detailSections.request') }}</h4>
          <div class="detail-grid">
            <div class="detail-item">
              <span class="detail-item__label">{{ t('audit.logList.detailFields.requestId') }}</span>
              <span class="detail-item__value detail-item__value--mono">{{ detailRecord.request_id || '-' }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-item__label">{{ t('audit.logList.detailFields.ip') }}</span>
              <span class="detail-item__value">{{ detailRecord.ip || '-' }}</span>
            </div>
            <div class="detail-item detail-item--full">
              <span class="detail-item__label">{{ t('audit.logList.detailFields.userAgent') }}</span>
              <span class="detail-item__value">{{ detailRecord.user_agent || '-' }}</span>
            </div>
            <div class="detail-item detail-item--full">
              <span class="detail-item__label">{{ t('audit.logList.detailFields.message') }}</span>
              <span class="detail-item__value">{{ detailRecord.message || '-' }}</span>
            </div>
          </div>
        </div>

        <div class="detail-section">
          <div class="detail-section__header">
            <h4 class="detail-section__title">{{ t('audit.logList.detailSections.metadata') }}</h4>
            <t-button theme="default" variant="text" size="small" @click="copyMetadata(detailRecord)">
              {{ t('audit.logList.copyMetadata') }}
            </t-button>
          </div>
          <pre class="detail-code">{{ metadataDetail(detailRecord.metadata) }}</pre>
        </div>
      </div>
    </t-drawer>
  </div>
</template>
<script setup lang="ts">
import type { TdBaseTableProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next';
import { computed, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import { resolveLocalizedErrorMessage } from '@/modules/shared/localized-api-error';
import {
  calculateTableContentWidth,
  createActionColumn,
  createStatusColumn,
  createTextColumn,
  createTimeColumn,
  formatCompactDateTime,
  ManagementEmptyState,
  ManagementPageContent,
  ManagementPageHeader,
  ManagementTableCard,
  ManagementTablePagination,
  ManagementToolbar,
  TableActionMenu,
} from '@/shared/components/management';
import { createLogger } from '@/utils/logger';

import { getAuditLogs } from '../../api/audit';
import { AUDIT_PERMISSION_CODE } from '../../contract/permissions';
import type { AuditLogListItem, AuditLogQuery } from '../../types/audit';

defineOptions({
  name: 'AuditLogListIndex',
});

type AuditFilterState = {
  action: string;
  resource_type: string;
  resource_name: string;
  successValue: '' | 'true' | 'false';
};

const logger = createLogger('audit.logs');
const { t, locale } = useI18n();

const loading = ref(false);
const listError = ref('');
const rows = ref<AuditLogListItem[]>([]);
const total = ref(0);
const createdRange = ref<string[]>([]);
const detailDrawerVisible = ref(false);
const detailRecord = ref<AuditLogListItem | null>(null);
const filters = ref<AuditFilterState>({
  action: '',
  resource_type: '',
  resource_name: '',
  successValue: '',
});
const pagination = ref({
  current: 1,
  pageSize: 10,
});

const successOptions = computed(() => [
  { label: t('audit.logList.filters.successAll'), value: '' },
  { label: t('audit.logList.filters.successTrue'), value: 'true' },
  { label: t('audit.logList.filters.successFalse'), value: 'false' },
]);

const hasActiveFilters = computed(() => {
  return Boolean(
    filters.value.action.trim() ||
    filters.value.resource_type.trim() ||
    filters.value.resource_name.trim() ||
    filters.value.successValue ||
    createdRange.value.length,
  );
});

const columns = computed<TdBaseTableProps['columns']>(() => {
  void locale.value;

  return [
    createTextColumn(t('audit.logList.columns.action'), 'action', {
      fixed: 'left',
      minWidth: 360,
    }),
    createTextColumn(t('audit.logList.columns.actor'), 'actor', {
      width: 200,
    }),
    createTextColumn(t('audit.logList.columns.resource'), 'resource', {
      width: 240,
    }),
    createStatusColumn(t('audit.logList.columns.result'), 'result', 100),
    createTimeColumn(t('audit.logList.columns.createdAt'), 'created_at', 180),
    createActionColumn(t('components.commonTable.operation'), 108),
  ];
});

const tableContentWidth = computed(() => calculateTableContentWidth(columns.value));

function toQuery(): AuditLogQuery {
  const query: AuditLogQuery = {
    page: pagination.value.current,
    page_size: pagination.value.pageSize,
  };

  if (filters.value.action.trim()) {
    query.action = filters.value.action.trim();
  }
  if (filters.value.resource_type.trim()) {
    query.resource_type = filters.value.resource_type.trim();
  }
  if (filters.value.resource_name.trim()) {
    query.resource_name = filters.value.resource_name.trim();
  }
  if (filters.value.successValue === 'true') {
    query.success = true;
  } else if (filters.value.successValue === 'false') {
    query.success = false;
  }
  if (createdRange.value[0]) {
    query.created_from = toISOStringOrRaw(createdRange.value[0]);
  }
  if (createdRange.value[1]) {
    query.created_to = toISOStringOrRaw(createdRange.value[1]);
  }

  return query;
}

async function fetchAuditLogs() {
  loading.value = true;
  listError.value = '';

  try {
    const response = await getAuditLogs(toQuery());
    rows.value = response.items;
    total.value = response.total;
  } catch (error) {
    rows.value = [];
    total.value = 0;
    logger.error('failed to fetch audit logs', error);
    listError.value = resolveLocalizedErrorMessage(t, error, t('audit.logList.loadFailed'));
    MessagePlugin.error(listError.value);
  } finally {
    loading.value = false;
  }
}

function resetFilters() {
  filters.value = {
    action: '',
    resource_type: '',
    resource_name: '',
    successValue: '',
  };
  createdRange.value = [];
  pagination.value.current = 1;
}

function handlePageChange() {
  fetchAuditLogs();
}

function actorLabel(row: AuditLogListItem) {
  return row.actor_display_name || row.actor_username || t('audit.logList.actor.anonymous');
}

function auditActionTitle(row: AuditLogListItem) {
  return row.action || row.request_id;
}

function resourceLabel(row: AuditLogListItem) {
  return row.resource_name || t('audit.logList.resource.unknown');
}

function resourceSecondaryLabel(row: AuditLogListItem) {
  return row.resource_id ? `${row.resource_type || '-'} / ${row.resource_id}` : row.resource_type || '-';
}

function resourceDetailLabel(row: AuditLogListItem) {
  const detailParts = [resourceLabel(row)];

  if (row.resource_type) {
    detailParts.push(row.resource_type);
  }
  if (row.resource_id) {
    detailParts.push(row.resource_id);
  }

  return detailParts.join(' / ');
}

function metadataDetail(metadata: AuditLogListItem['metadata']) {
  if (!metadata || typeof metadata !== 'object' || Object.keys(metadata).length === 0) {
    return '-';
  }

  return JSON.stringify(metadata, null, 2);
}

async function copyMetadata(row: AuditLogListItem) {
  try {
    await navigator.clipboard.writeText(metadataDetail(row.metadata));
    MessagePlugin.success(t('audit.logList.copyMetadataSuccess'));
  } catch (error) {
    logger.error('failed to copy audit metadata', error);
    MessagePlugin.error(t('audit.logList.copyMetadataFailed'));
  }
}

function openDetailDrawer(row: AuditLogListItem) {
  detailRecord.value = row;
  detailDrawerVisible.value = true;
}

function formatTimestamp(value?: string | null) {
  return formatCompactDateTime(value);
}

function toISOStringOrRaw(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toISOString();
}

onMounted(() => {
  fetchAuditLogs();
});

watch(
  () =>
    [
      filters.value.action,
      filters.value.resource_type,
      filters.value.resource_name,
      filters.value.successValue,
      createdRange.value[0],
      createdRange.value[1],
    ] as const,
  () => {
    pagination.value.current = 1;
    fetchAuditLogs();
  },
);
</script>
<style scoped lang="less">
@import '../../../rbac/shared/list-page.less';

.audit-page {
  display: flex;
  flex-direction: column;
  gap: 16px;

  .management-list-toolbar();
  .management-list-header();
  .management-list-table-empty();
  .management-list-table-shell();
  .management-list-mobile();
}

.toolbar__date {
  min-width: min(100%, 320px);
}

.inline-note {
  --audit-note-bg: color-mix(in srgb, var(--td-brand-color) 4%, var(--td-bg-color-container));

  background: var(--audit-note-bg);
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 92%, var(--td-brand-color));
  border-inline-start: 3px solid var(--td-brand-color);
  box-shadow: inset 0 1px 0 color-mix(in srgb, var(--td-brand-color) 8%, transparent);
  color: var(--td-text-color-placeholder);
  display: grid;
  gap: 6px;
  padding: 12px 14px 12px 16px;
}

.inline-note p,
.table-head__summary,
.table-head__description {
  margin: 0;
}

.action-cell,
.stack-cell,
.detail-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.action-cell__primary,
.stack-cell strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
}

.action-cell__secondary,
.stack-cell__secondary,
.detail-item__label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.detail-item__value {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
}

.detail-item__value--mono,
.detail-code {
  font-family: var(--td-font-family-medium);
}

.audit-detail-panel,
.detail-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.detail-section__header {
  align-items: center;
  display: flex;
  justify-content: space-between;
}

.detail-section__title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  margin: 0;
}

.detail-grid {
  display: grid;
  gap: 16px;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.detail-item--full {
  grid-column: 1 / -1;
}

.detail-code {
  background: var(--td-bg-color-page);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  margin: 0;
  max-height: 240px;
  overflow: auto;
  padding: 12px;
  white-space: pre-wrap;
}

@media (width <= 768px) {
  .toolbar__date {
    min-width: 100%;
  }

  .detail-grid {
    grid-template-columns: 1fr;
  }
}
</style>
