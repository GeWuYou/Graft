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
          <t-input
            v-model="filters.request_id"
            clearable
            class="toolbar__select"
            :placeholder="t('audit.logList.filters.requestIdPlaceholder')"
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
          table-content-width="100%"
          cell-empty-content="-"
          hover
        >
          <template #action="{ row }">
            <div class="action-cell">
              <strong class="action-cell__primary">{{ row.action }}</strong>
              <span class="action-cell__secondary">{{ row.resource_type }}</span>
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
              <span class="stack-cell__secondary">{{ row.resource_id || '-' }}</span>
            </div>
          </template>

          <template #result="{ row }">
            <t-tag :theme="row.success ? 'success' : 'danger'" variant="light-outline" size="small" shape="round">
              {{ row.success ? t('audit.logList.result.success') : t('audit.logList.result.failed') }}
            </t-tag>
          </template>

          <template #request_id="{ row }">
            <span class="request-id">{{ row.request_id }}</span>
          </template>

          <template #created_at="{ row }">
            <span>{{ formatTimestamp(row.created_at) }}</span>
          </template>

          <template #context="{ row }">
            <div class="context-cell">
              <div v-if="row.ip" class="context-line">{{ t('audit.logList.context.ip') }}: {{ row.ip }}</div>
              <div v-if="row.user_agent" class="context-line">
                {{ t('audit.logList.context.userAgent') }}: {{ row.user_agent }}
              </div>
              <div v-if="row.message" class="context-line">
                {{ t('audit.logList.context.message') }}: {{ row.message }}
              </div>
              <div v-if="metadataLabel(row.metadata)" class="context-line">
                {{ t('audit.logList.context.metadata') }}: {{ metadataLabel(row.metadata) }}
              </div>
              <span v-if="!hasContext(row)" class="stack-cell__secondary">{{ t('audit.logList.context.none') }}</span>
            </div>
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
  </div>
</template>
<script setup lang="ts">
import type { TdBaseTableProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next';
import { computed, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import { resolveLocalizedErrorMessage } from '@/modules/shared/localized-api-error';
import { ManagementEmptyState, ManagementPageContent } from '@/shared/components/management';
import {
  ManagementPageHeader,
  ManagementTableCard,
  ManagementTablePagination,
  ManagementToolbar,
} from '@/shared/components/management';
import { createLogger } from '@/utils/logger';

import { getAuditLogs } from '../api/audit';
import { AUDIT_PERMISSION_CODE } from '../contract/permissions';
import type { AuditLogListItem, AuditLogQuery } from '../types/audit';

defineOptions({
  name: 'AuditLogListIndex',
});

type AuditFilterState = {
  action: string;
  resource_type: string;
  resource_name: string;
  request_id: string;
  successValue: '' | 'true' | 'false';
};

const logger = createLogger('audit.logList');
const { t, locale } = useI18n();

const loading = ref(false);
const listError = ref('');
const rows = ref<AuditLogListItem[]>([]);
const total = ref(0);
const createdRange = ref<string[]>([]);
const filters = ref<AuditFilterState>({
  action: '',
  resource_type: '',
  resource_name: '',
  request_id: '',
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
    filters.value.request_id.trim() ||
    filters.value.successValue ||
    createdRange.value.length,
  );
});

const columns = computed<TdBaseTableProps['columns']>(() => {
  void locale.value;

  return [
    { title: t('audit.logList.columns.action'), colKey: 'action', minWidth: 180, fixed: 'left' },
    { title: t('audit.logList.columns.actor'), colKey: 'actor', minWidth: 220 },
    { title: t('audit.logList.columns.resource'), colKey: 'resource', minWidth: 220 },
    { title: t('audit.logList.columns.result'), colKey: 'result', width: 120 },
    { title: t('audit.logList.columns.requestId'), colKey: 'request_id', minWidth: 200 },
    { title: t('audit.logList.columns.createdAt'), colKey: 'created_at', width: 180 },
    { title: t('audit.logList.columns.context'), colKey: 'context', minWidth: 360 },
  ];
});

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
  if (filters.value.request_id.trim()) {
    query.request_id = filters.value.request_id.trim();
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
    request_id: '',
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

function resourceLabel(row: AuditLogListItem) {
  return row.resource_name || t('audit.logList.resource.unknown');
}

function metadataLabel(metadata: AuditLogListItem['metadata']) {
  if (!metadata || typeof metadata !== 'object') {
    return '';
  }

  const entries = Object.entries(metadata).slice(0, 3);
  if (entries.length === 0) {
    return '';
  }

  return entries
    .map(([key, value]) => `${key}=${typeof value === 'string' ? value : JSON.stringify(value)}`)
    .join(', ');
}

function hasContext(row: AuditLogListItem) {
  return Boolean(row.ip || row.user_agent || row.message || metadataLabel(row.metadata));
}

function formatTimestamp(value?: string | null) {
  if (!value) {
    return '-';
  }

  const parsedAt = Date.parse(value);
  if (Number.isNaN(parsedAt)) {
    return value;
  }

  const languageTag = locale.value.startsWith('zh') ? 'zh-CN' : 'en-US';
  const formatter = new Intl.DateTimeFormat(languageTag, {
    dateStyle: resolveAuditDateStyle(languageTag),
    timeStyle: 'short',
  });

  return formatter.format(new Date(parsedAt));
}

function toISOStringOrRaw(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toISOString();
}

function resolveAuditDateStyle(languageTag: 'zh-CN' | 'en-US'): 'medium' {
  void languageTag;
  return 'medium';
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
      filters.value.request_id,
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
@import '../../rbac/shared/list-page.less';

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
.context-cell {
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
.context-line,
.request-id {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.request-id {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
}

@media (width <= 768px) {
  .toolbar__date {
    min-width: 100%;
  }
}
</style>
