<template>
  <div class="audit-page" data-page-type="list-form-detail">
    <management-page-content>
      <management-page-header :title="t('audit.logList.title')" :description="t('audit.logList.description')">
        <template #eyebrow>{{ t('menu.audit.title') }}</template>
        <template #actions>
          <t-button theme="default" variant="outline" :loading="loading" @click="fetchAuditLogs">
            {{ t('audit.logList.refresh') }}
          </t-button>
        </template>
      </management-page-header>

      <audit-filters
        v-model="filters"
        :active-preset="activePreset"
        :loading="loading"
        :presets="presetViews"
        @apply-preset="applyPreset"
        @reset="resetFilters"
        @search="handleSearch"
      />

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

      <audit-table
        v-else
        v-model:current="pagination.current"
        v-model:page-size="pagination.pageSize"
        :description="t('audit.logList.tableHint')"
        :footer-summary="footerSummary"
        :loading="loading"
        :local-filter-active="hasClientOnlyFilters"
        :rows="displayRows"
        :summary="tableSummary"
        :total="tableTotal"
        @detail="openDetailDrawer"
        @page-change="fetchAuditLogs"
      />
    </management-page-content>

    <audit-detail-drawer v-model:visible="detailDrawerVisible" :record="detailRecord" :rows="rows" />
  </div>
</template>
<script setup lang="ts">
import { MessagePlugin } from 'tdesign-vue-next';
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';

import { resolveLocalizedErrorMessage } from '@/modules/shared/localized-api-error';
import { ManagementEmptyState, ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import { describeCorrelationId, formatMessageWithCorrelation } from '@/shared/correlation';
import { createLogger } from '@/utils/logger';

import { getAuditLogs } from '../../api/audit';
import AuditDetailDrawer from '../../components/AuditDetailDrawer.vue';
import AuditFilters from '../../components/AuditFilters.vue';
import AuditTable from '../../components/AuditTable.vue';
import type { AuditClientFilterState } from '../../shared/presentation';
import { matchesAuditRow } from '../../shared/presentation';
import type { AuditLogListItem, AuditLogQuery } from '../../types/audit';

defineOptions({
  name: 'AuditLogListIndex',
});

type PresetKey = 'all' | 'today-anomalies' | 'permission-denied' | 'sensitive-ops' | 'auth-failed' | 'high-risk';

const logger = createLogger('audit.logs');
const { t } = useI18n();
const route = useRoute();

const loading = ref(false);
const listError = ref('');
const rows = ref<AuditLogListItem[]>([]);
const total = ref(0);
const activePreset = ref<PresetKey>('all');
const detailDrawerVisible = ref(false);
const detailRecord = ref<AuditLogListItem | null>(null);
const latestRequestSeq = ref(0);
const pagination = ref({
  current: 1,
  pageSize: 10,
});
const filters = ref<AuditClientFilterState>({
  ...createDefaultFilters(),
});

const presetViews = computed(() => [
  { key: 'all' as const, title: t('audit.logList.presets.all') },
  { key: 'today-anomalies' as const, title: t('audit.logList.presets.todayAnomalies') },
  { key: 'permission-denied' as const, title: t('audit.logList.presets.permissionDenied') },
  { key: 'sensitive-ops' as const, title: t('audit.logList.presets.sensitiveOps') },
  { key: 'auth-failed' as const, title: t('audit.logList.presets.authFailed') },
  { key: 'high-risk' as const, title: t('audit.logList.presets.highRisk') },
]);

const hasClientOnlyFilters = computed(() =>
  Boolean(
    filters.value.keyword ||
    filters.value.actor ||
    filters.value.resourceId ||
    filters.value.session ||
    filters.value.requestId ||
    filters.value.traceId,
  ),
);

const displayRows = computed(() => rows.value.filter((row) => matchesAuditRow(row, filters.value, t)));
const tableTotal = computed(() => (hasClientOnlyFilters.value ? displayRows.value.length : total.value));
const tableSummary = computed(() => t('audit.logList.summary', { count: displayRows.value.length }));
const footerSummary = computed(() =>
  hasClientOnlyFilters.value
    ? t('audit.logList.footerFiltered', { count: displayRows.value.length })
    : t('audit.logList.footerTotal', { count: total.value }),
);

function buildQuery(): AuditLogQuery {
  const query: AuditLogQuery = {
    page: pagination.value.current,
    page_size: pagination.value.pageSize,
  };

  if (filters.value.action) {
    query.action = filters.value.action;
  }
  if (filters.value.source) {
    query.source = filters.value.source as AuditLogQuery['source'];
  }
  if (filters.value.resourceType) {
    query.resource_type = filters.value.resourceType;
  }
  if (filters.value.resourceName) {
    query.resource_name = filters.value.resourceName;
  }
  if (filters.value.resourceId) {
    query.resource_id = filters.value.resourceId;
  }
  if (filters.value.requestId) {
    query.request_id = filters.value.requestId;
  } else if (filters.value.traceId) {
    query.request_id = filters.value.traceId;
  }
  if (filters.value.result !== 'all') {
    query.result = filters.value.result;
  }
  if (filters.value.riskLevel !== 'all') {
    query.risk_level = filters.value.riskLevel;
  }
  if (filters.value.createdRange[0]) {
    query.created_from = toISOStringOrRaw(filters.value.createdRange[0]);
  }
  if (filters.value.createdRange[1]) {
    query.created_to = toISOStringOrRaw(filters.value.createdRange[1]);
  }

  return query;
}

async function fetchAuditLogs() {
  const requestSeq = ++latestRequestSeq.value;
  loading.value = true;
  listError.value = '';

  try {
    const response = await getAuditLogs(buildQuery());
    if (requestSeq !== latestRequestSeq.value) {
      return;
    }
    rows.value = response.items;
    total.value = response.total;
  } catch (error) {
    if (requestSeq !== latestRequestSeq.value) {
      return;
    }
    rows.value = [];
    total.value = 0;
    logger.error('failed to fetch audit logs', error);
    listError.value = resolveLocalizedErrorMessage(t, error, t('audit.logList.loadFailed'));
    MessagePlugin.error(
      formatMessageWithCorrelation(
        listError.value,
        describeCorrelationId(t, filters.value.requestId || filters.value.traceId),
      ),
    );
  } finally {
    if (requestSeq === latestRequestSeq.value) {
      loading.value = false;
    }
  }
}

function applyPreset(preset: PresetKey) {
  activePreset.value = preset;
  filters.value = {
    ...createDefaultFilters(),
    ...presetFilterOverrides(preset),
  };

  pagination.value.current = 1;
  syncRouteQuery();
  fetchAuditLogs();
}

function handleSearch() {
  pagination.value.current = 1;
  fetchAuditLogs();
}

function resetFilters() {
  filters.value = createDefaultFilters();
  activePreset.value = 'all';
  pagination.value.current = 1;
  syncRouteQuery();
  fetchAuditLogs();
}

function createDefaultFilters(): AuditClientFilterState {
  return {
    keyword: '',
    actor: '',
    action: '',
    source: '',
    createdRange: [],
    resourceType: '',
    resourceName: '',
    resourceId: '',
    result: 'all',
    riskLevel: 'all',
    session: '',
    requestId: '',
    traceId: '',
  };
}

function presetFilterOverrides(preset: PresetKey): Partial<AuditClientFilterState> {
  switch (preset) {
    case 'today-anomalies':
      return { source: 'SECURITY_EVENT', result: 'ERROR', riskLevel: 'HIGH' };
    case 'permission-denied':
      return { source: 'SECURITY_EVENT', result: 'DENIED', riskLevel: 'CRITICAL' };
    case 'auth-failed':
      return { source: 'REQUEST', result: 'FAILED', resourceType: 'auth', riskLevel: 'HIGH' };
    case 'sensitive-ops':
      return { riskLevel: 'HIGH' };
    case 'high-risk':
      return { source: 'SECURITY_EVENT', riskLevel: 'CRITICAL' };
    default:
      return {};
  }
}

function openDetailDrawer(row: AuditLogListItem) {
  detailRecord.value = row;
  detailDrawerVisible.value = true;
}

function toISOStringOrRaw(value: string) {
  const date = new Date(value.replace(' ', 'T'));
  return Number.isNaN(date.getTime()) ? value : date.toISOString();
}

function applyRoutePreset() {
  const preset = route.query.preset;
  if (
    preset === 'today-anomalies' ||
    preset === 'permission-denied' ||
    preset === 'sensitive-ops' ||
    preset === 'auth-failed' ||
    preset === 'high-risk'
  ) {
    applyPreset(preset);
    return true;
  }
  return false;
}

function firstQueryValue(value: unknown) {
  return typeof value === 'string' ? value : '';
}

function applyRouteFilters() {
  const nextPreset = firstQueryValue(route.query.preset) as PresetKey | '';
  const nextFilters: AuditClientFilterState = {
    keyword: firstQueryValue(route.query.keyword),
    actor: firstQueryValue(route.query.actor),
    action: firstQueryValue(route.query.action),
    source: firstQueryValue(route.query.source),
    createdRange: [],
    resourceType: firstQueryValue(route.query.resourceType),
    resourceName: firstQueryValue(route.query.resourceName),
    resourceId: firstQueryValue(route.query.resourceId),
    result: (firstQueryValue(route.query.result) as AuditClientFilterState['result']) || 'all',
    riskLevel: (firstQueryValue(route.query.riskLevel) as AuditClientFilterState['riskLevel']) || 'all',
    session: firstQueryValue(route.query.session),
    requestId: firstQueryValue(route.query.requestId),
    traceId: firstQueryValue(route.query.traceId),
  };

  filters.value = nextFilters;
  activePreset.value = nextPreset || 'all';
}

function syncRouteQuery() {
  const query: Record<string, string> = {};

  if (activePreset.value !== 'all') {
    query.preset = activePreset.value;
  }
  if (filters.value.keyword) {
    query.keyword = filters.value.keyword;
  }
  if (filters.value.actor) {
    query.actor = filters.value.actor;
  }
  if (filters.value.action) {
    query.action = filters.value.action;
  }
  if (filters.value.source) {
    query.source = filters.value.source;
  }
  if (filters.value.resourceType) {
    query.resourceType = filters.value.resourceType;
  }
  if (filters.value.resourceName) {
    query.resourceName = filters.value.resourceName;
  }
  if (filters.value.resourceId) {
    query.resourceId = filters.value.resourceId;
  }
  if (filters.value.result !== 'all') {
    query.result = filters.value.result;
  }
  if (filters.value.riskLevel !== 'all') {
    query.riskLevel = filters.value.riskLevel;
  }
  if (filters.value.session) {
    query.session = filters.value.session;
  }
  if (filters.value.requestId) {
    query.requestId = filters.value.requestId;
  }
  if (filters.value.traceId) {
    query.traceId = filters.value.traceId;
  }

  void history.replaceState(
    history.state,
    '',
    `${route.path}${new URLSearchParams(query).toString() ? `?${new URLSearchParams(query).toString()}` : ''}`,
  );
}

onMounted(() => {
  applyRouteFilters();
  if (!applyRoutePreset()) {
    fetchAuditLogs();
  }
});
</script>
<style scoped lang="less">
@import '../../../rbac/shared/list-page.less';

.audit-page {
  .management-list-header();
  .management-list-toolbar();
  .management-list-table-empty();
  .management-list-table-shell();
  .management-list-mobile();

  display: flex;
  flex-direction: column;
  gap: 16px;
}
</style>
