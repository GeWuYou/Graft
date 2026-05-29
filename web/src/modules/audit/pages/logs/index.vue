<template>
  <div class="audit-page" data-page-type="list-form-detail">
    <management-page-content>
      <management-page-header :title="t('audit.logList.title')" :description="t('audit.logList.description')">
        <template #eyebrow>{{ t('menu.audit.title') }}</template>
        <template #actions>
          <t-space size="small" wrap>
            <t-button
              v-for="preset in presetViews"
              :key="preset.key"
              size="small"
              :theme="activePreset === preset.key ? 'primary' : 'default'"
              :variant="activePreset === preset.key ? 'base' : 'outline'"
              @click="applyPreset(preset.key)"
            >
              {{ preset.title }}
            </t-button>
            <t-button theme="default" variant="outline" :loading="loading" @click="fetchAuditLogs">
              {{ t('audit.logList.refresh') }}
            </t-button>
          </t-space>
        </template>
      </management-page-header>

      <audit-filters
        v-model="filters"
        :advanced-visible="advancedVisible"
        :loading="loading"
        @reset="resetFilters"
        @search="handleSearch"
        @toggle-advanced="advancedVisible = !advancedVisible"
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
const advancedVisible = ref(false);
const detailDrawerVisible = ref(false);
const detailRecord = ref<AuditLogListItem | null>(null);
const latestRequestSeq = ref(0);
const pagination = ref({
  current: 1,
  pageSize: 10,
});
const filters = ref<AuditClientFilterState>({
  keyword: '',
  actor: '',
  action: '',
  createdRange: [],
  resource: '',
  resourceId: '',
  result: 'all',
  riskLevel: 'all',
  session: '',
  requestId: '',
  traceId: '',
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
    filters.value.resource ||
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
  if (filters.value.resource) {
    query.resource_type = filters.value.resource;
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

  if (preset === 'all') {
    filters.value.action = '';
    filters.value.result = 'all';
    filters.value.resource = '';
    filters.value.resourceId = '';
    filters.value.riskLevel = 'all';
  } else if (preset === 'today-anomalies') {
    filters.value.action = '';
    filters.value.result = 'ERROR';
    filters.value.resource = '';
    filters.value.resourceId = '';
    filters.value.riskLevel = 'HIGH';
  } else if (preset === 'permission-denied') {
    filters.value.action = '';
    filters.value.result = 'DENIED';
    filters.value.resource = '';
    filters.value.resourceId = '';
    filters.value.riskLevel = 'CRITICAL';
  } else if (preset === 'auth-failed') {
    filters.value.action = '';
    filters.value.result = 'FAILED';
    filters.value.resource = 'auth';
    filters.value.resourceId = '';
    filters.value.riskLevel = 'HIGH';
  } else if (preset === 'sensitive-ops') {
    filters.value.action = '';
    filters.value.result = 'all';
    filters.value.resource = '';
    filters.value.resourceId = '';
    filters.value.riskLevel = 'HIGH';
  } else if (preset === 'high-risk') {
    filters.value.action = '';
    filters.value.result = 'all';
    filters.value.resource = '';
    filters.value.resourceId = '';
    filters.value.riskLevel = 'CRITICAL';
  }

  pagination.value.current = 1;
  syncRouteQuery();
  fetchAuditLogs();
}

function handleSearch() {
  pagination.value.current = 1;
  fetchAuditLogs();
}

function resetFilters() {
  filters.value = {
    keyword: '',
    actor: '',
    action: '',
    createdRange: [],
    resource: '',
    resourceId: '',
    result: 'all',
    riskLevel: 'all',
    session: '',
    requestId: '',
    traceId: '',
  };
  activePreset.value = 'all';
  pagination.value.current = 1;
  syncRouteQuery();
  fetchAuditLogs();
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
    createdRange: [],
    resource: firstQueryValue(route.query.resourceType),
    resourceId: firstQueryValue(route.query.resourceId),
    result: (firstQueryValue(route.query.result) as AuditClientFilterState['result']) || 'all',
    riskLevel: (firstQueryValue(route.query.riskLevel) as AuditClientFilterState['riskLevel']) || 'all',
    session: firstQueryValue(route.query.session),
    requestId: firstQueryValue(route.query.requestId),
    traceId: firstQueryValue(route.query.traceId),
  };

  filters.value = nextFilters;
  activePreset.value = nextPreset || 'all';
  advancedVisible.value = Boolean(
    nextFilters.resource ||
    nextFilters.resourceId ||
    nextFilters.result !== 'all' ||
    nextFilters.riskLevel !== 'all' ||
    nextFilters.session ||
    nextFilters.requestId ||
    nextFilters.traceId,
  );
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
  if (filters.value.resource) {
    query.resourceType = filters.value.resource;
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
