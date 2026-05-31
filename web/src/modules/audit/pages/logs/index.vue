<template>
  <div class="audit-page" data-page-type="query-builder-list-detail">
    <management-page-content>
      <management-page-header :title="t('audit.logList.title')" :description="t('audit.logList.description')">
        <template #eyebrow>{{ t('menu.audit.title') }}</template>
        <template #actions>
          <t-button theme="default" variant="outline" @click="columnDrawerVisible = true">
            {{ t('audit.logList.columnSettings') }}
          </t-button>
          <t-button v-if="monitorReturnLocation" theme="primary" variant="outline" @click="returnToMonitor">
            {{ t('audit.logList.actions.backToMonitor') }}
          </t-button>
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
        :footer-summary="footerSummary"
        :loading="loading"
        :local-filter-active="hasClientOnlyFilters"
        :rows="displayRows"
        :total="tableTotal"
        :visible-column-keys="visibleColumnKeys"
        @detail="openDetailDrawer"
        @page-change="fetchAuditLogs"
      />
    </management-page-content>

    <t-drawer
      v-model:visible="columnDrawerVisible"
      :header="t('audit.logList.columnSettings')"
      :footer="false"
      placement="right"
      size="320px"
    >
      <t-checkbox-group v-model="visibleColumnKeys">
        <div class="column-grid">
          <t-checkbox v-for="column in columnSettingOptions" :key="column.value" :value="column.value">
            {{ column.label }}
          </t-checkbox>
        </div>
      </t-checkbox-group>
    </t-drawer>

    <audit-detail-drawer
      v-model:visible="detailDrawerVisible"
      :record="detailRecord"
      :rows="rows"
      :monitor-origin="navigationContext.monitorOrigin"
    />
  </div>
</template>
<script setup lang="ts">
import { MessagePlugin } from 'tdesign-vue-next';
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { resolveLocalizedErrorMessage } from '@/modules/shared/localized-api-error';
import { ManagementEmptyState, ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import { describeCorrelationId, formatMessageWithCorrelation } from '@/shared/correlation';
import {
  createSingleSorter,
  getSingleSorter,
  localDateTimeToUtcIso,
  normalizePageStateRangeForRoute,
  normalizeRouteRangeForPageState,
} from '@/shared/observability';
import { createLogger } from '@/utils/logger';

import { getAuditLogs } from '../../api/audit';
import AuditDetailDrawer from '../../components/AuditDetailDrawer.vue';
import AuditFilters from '../../components/AuditFilters.vue';
import AuditTable from '../../components/AuditTable.vue';
import { buildAuditLogsLocation, parseAuditLogsRouteQuery } from '../../contract/deep-link';
import {
  buildMonitorReturnLocation,
  resolveAuditNavigationContext,
  withMonitorOrigin,
} from '../../contract/navigation';
import {
  applyAuditPresetFilters,
  type AuditQuickPresetKey,
  listAuditPresets,
  resolveAuditPresetKey,
} from '../../contract/presets';
import { type AuditTimePreset, resolveAuditTimePreset } from '../../contract/time-presets';
import type { AuditClientFilterState } from '../../shared/presentation';
import { matchesAuditRow } from '../../shared/presentation';
import type { AuditLogListItem, AuditLogQuery } from '../../types/audit';

defineOptions({
  name: 'AuditLogListIndex',
});

const logger = createLogger('audit.logs');
const { t } = useI18n();
const route = useRoute();
const router = useRouter();

const loading = ref(false);
const listError = ref('');
const rows = ref<AuditLogListItem[]>([]);
const total = ref(0);
const activePreset = ref<AuditQuickPresetKey>(resolveAuditPresetKey(''));
const activeTimePreset = ref<AuditTimePreset | ''>('');
const detailDrawerVisible = ref(false);
const detailRecord = ref<AuditLogListItem | null>(null);
const latestRequestSeq = ref(0);
const columnDrawerVisible = ref(false);
const visibleColumnKeys = ref(['action', 'actor', 'resource', 'correlation', 'result', 'risk', 'created_at']);
const pagination = ref({
  current: 1,
  pageSize: 10,
});
const filters = ref<AuditClientFilterState>({
  ...createDefaultFilters(),
});
const applyingRoute = ref(false);
const navigationContext = computed(() => resolveAuditNavigationContext(route.query));
const monitorReturnLocation = computed(() => buildMonitorReturnLocation(route.query));

const presetViews = computed(() =>
  listAuditPresets().map((preset) => ({
    key: preset.key,
    title: t(preset.titleKey),
  })),
);
const columnSettingOptions = computed(() => [
  { label: t('audit.logList.columns.action'), value: 'action' },
  { label: t('audit.logList.columns.actor'), value: 'actor' },
  { label: t('audit.logList.columns.resource'), value: 'resource' },
  { label: t('audit.logList.columns.correlation'), value: 'correlation' },
  { label: t('audit.logList.columns.result'), value: 'result' },
  { label: t('audit.logList.columns.risk'), value: 'risk' },
  { label: t('audit.logList.columns.createdAt'), value: 'created_at' },
]);

const hasClientOnlyFilters = computed(() =>
  Boolean(
    filters.value.keyword ||
    filters.value.actor ||
    filters.value.actorUserId ||
    filters.value.resourceId ||
    filters.value.session ||
    filters.value.requestId,
  ),
);

const displayRows = computed(() => rows.value.filter((row) => matchesAuditRow(row, filters.value, t)));
const tableTotal = computed(() => total.value);
const footerSummary = computed(() =>
  hasClientOnlyFilters.value
    ? t('audit.logList.footerFiltered', { count: displayRows.value.length })
    : t('audit.logList.footerTotal', { count: total.value }),
);

function buildQuery(): AuditLogQuery {
  const sorter = getSingleSorter(filters.value.sorters);
  const query: AuditLogQuery = {
    page: pagination.value.current,
    page_size: pagination.value.pageSize,
  };

  if (filters.value.action) {
    query.action = filters.value.action;
  }
  if (filters.value.actionPrefix) {
    query.action_prefix = filters.value.actionPrefix;
  }
  if (filters.value.actionPrefixes.length) {
    query.action_prefixes = [...filters.value.actionPrefixes];
  }
  if (filters.value.actionKeywords.length) {
    query.action_keywords = [...filters.value.actionKeywords];
  }
  if (filters.value.source) {
    query.source = filters.value.source as AuditLogQuery['source'];
  }
  if (filters.value.resourceType) {
    query.resource_type = filters.value.resourceType;
  }
  if (filters.value.resourceTypes.length) {
    query.resource_types = [...filters.value.resourceTypes];
  }
  if (filters.value.resourceName) {
    query.resource_name = filters.value.resourceName;
  }
  if (filters.value.resourceId) {
    query.resource_id = filters.value.resourceId;
  }
  if (filters.value.requestId) {
    query.request_id = filters.value.requestId;
  }
  if (filters.value.result !== 'all') {
    query.result = filters.value.result;
  }
  if (filters.value.results.length) {
    query.results = [...filters.value.results];
  }
  if (filters.value.riskLevel !== 'all') {
    query.risk_level = filters.value.riskLevel;
  }
  if (filters.value.riskLevels.length) {
    query.risk_levels = [...filters.value.riskLevels];
  }
  if (filters.value.success !== 'all') {
    query.success = filters.value.success === 'true';
  }
  if (filters.value.requestPathPrefixes.length) {
    query.request_path_prefixes = [...filters.value.requestPathPrefixes];
  }
  if (filters.value.createdRange[0]) {
    query.created_from = localDateTimeToUtcIso(filters.value.createdRange[0]);
  }
  if (filters.value.createdRange[1]) {
    query.created_to = localDateTimeToUtcIso(filters.value.createdRange[1]);
  }
  if (!query.created_from && !query.created_to && activeTimePreset.value) {
    query.preset = activeTimePreset.value;
  }
  if (sorter?.field) {
    query.sort_by = sorter.field;
    if (sorter.direction) {
      query.sort_order = sorter.direction;
    }
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
    const correlationId = filters.value.requestId;
    MessagePlugin.error(
      correlationId
        ? formatMessageWithCorrelation(listError.value, describeCorrelationId(t, correlationId))
        : listError.value,
    );
  } finally {
    if (requestSeq === latestRequestSeq.value) {
      loading.value = false;
    }
  }
}

function applyPreset(preset: typeof activePreset.value) {
  activePreset.value = preset;
  filters.value = applyAuditPresetFilters(preset, filters.value, createDefaultFilters);

  pagination.value.current = 1;
  updateRouteQuery();
}

function handleSearch() {
  pagination.value.current = 1;
  updateRouteQuery();
}

function resetFilters() {
  filters.value = createDefaultFilters();
  activePreset.value = 'all';
  pagination.value.current = 1;
  updateRouteQuery();
}

function createDefaultFilters(): AuditClientFilterState {
  return {
    keyword: '',
    actor: '',
    actorUserId: '',
    success: 'all',
    action: '',
    actionPrefix: '',
    actionPrefixes: [],
    actionKeywords: [],
    requestPathPrefixes: [],
    source: '',
    createdRange: [],
    resourceType: '',
    resourceTypes: [],
    resourceName: '',
    resourceId: '',
    result: 'all',
    results: [],
    riskLevel: 'all',
    riskLevels: [],
    session: '',
    requestId: '',
    sorters: createSingleSorter('created_at', 'desc'),
  };
}

function openDetailDrawer(row: AuditLogListItem) {
  detailRecord.value = row;
  detailDrawerVisible.value = true;
}

function applyRouteFilters() {
  const query = parseAuditLogsRouteQuery(route.query);
  activeTimePreset.value = query.preset ? resolveAuditTimePreset(query.preset) : '';
  const nextFilters: AuditClientFilterState = {
    ...createDefaultFilters(),
    keyword: query.keyword ?? '',
    actor: query.username || query.actor || '',
    actorUserId: query.user_id ?? '',
    success: query.success === 'true' ? 'true' : query.success === 'false' ? 'false' : 'all',
    action: query.action || '',
    actionPrefix: query.action_prefix || '',
    actionPrefixes: splitRouteList(query.action_prefixes),
    actionKeywords: splitRouteList(query.action_keywords),
    requestPathPrefixes: splitRouteList(query.request_path_prefixes),
    source: query.source || '',
    createdRange: normalizeRouteRangeForPageState([query.created_from ?? '', query.created_to ?? '']),
    resourceType: query.resource_type || '',
    resourceTypes: splitRouteList(query.resource_types),
    resourceName: query.resource_name ?? '',
    resourceId: query.resource_id ?? '',
    result: (query.result as AuditClientFilterState['result']) || 'all',
    results: splitRouteList(query.results) as AuditClientFilterState['results'],
    riskLevel: (query.risk_level as AuditClientFilterState['riskLevel']) || 'all',
    riskLevels: splitRouteList(query.risk_levels) as AuditClientFilterState['riskLevels'],
    session: query.session ?? '',
    requestId: query.request_id ?? '',
    sorters: query.sort_by
      ? createSingleSorter('created_at', normalizeSortOrder(query.sort_order || 'desc'))
      : filters.value.sorters,
  };

  filters.value = nextFilters;
  activePreset.value = inferPresetFromFilters(nextFilters);
}

function buildRouteQuery() {
  const [createdFrom = '', createdTo = ''] = normalizePageStateRangeForRoute(filters.value.createdRange);
  const sorter = getSingleSorter(filters.value.sorters);

  return {
    preset: createdFrom || createdTo ? '' : activeTimePreset.value,
    keyword: filters.value.keyword,
    username: filters.value.actor,
    user_id: filters.value.actorUserId,
    success: filters.value.success === 'all' ? '' : filters.value.success,
    action: filters.value.action,
    action_prefix: filters.value.actionPrefix,
    action_prefixes: joinRouteList(filters.value.actionPrefixes),
    action_keywords: joinRouteList(filters.value.actionKeywords),
    request_path_prefixes: joinRouteList(filters.value.requestPathPrefixes),
    source: filters.value.source,
    created_from: createdFrom,
    created_to: createdTo,
    resource_type: filters.value.resourceType,
    resource_types: joinRouteList(filters.value.resourceTypes),
    resource_name: filters.value.resourceName,
    resource_id: filters.value.resourceId,
    result: filters.value.result === 'all' ? '' : filters.value.result,
    results: joinRouteList(filters.value.results),
    risk_level: filters.value.riskLevel === 'all' ? '' : filters.value.riskLevel,
    risk_levels: joinRouteList(filters.value.riskLevels),
    session: filters.value.session,
    request_id: filters.value.requestId,
    sort_by: sorter?.field ?? '',
    sort_order: sorter?.field ? (sorter.direction ?? '') : '',
  };
}

async function updateRouteQuery() {
  if (applyingRoute.value) {
    return;
  }

  const nextLocation = withMonitorOrigin(
    buildAuditLogsLocation(buildRouteQuery()),
    navigationContext.value.monitorOrigin,
  );
  const currentLocation = withMonitorOrigin(buildAuditLogsLocation(route.query), navigationContext.value.monitorOrigin);

  if (JSON.stringify(nextLocation.query) === JSON.stringify(currentLocation.query)) {
    await fetchAuditLogs();
    return;
  }

  await router.replace(nextLocation);
}

watch(
  () => route.query,
  async () => {
    applyingRoute.value = true;
    try {
      applyRouteFilters();
    } finally {
      applyingRoute.value = false;
    }
    pagination.value.current = 1;
    await fetchAuditLogs();
  },
  { immediate: true },
);

function returnToMonitor() {
  if (!monitorReturnLocation.value) {
    return;
  }

  void router.push(monitorReturnLocation.value);
}

function normalizeSortOrder(value: string) {
  return value === 'asc' ? 'asc' : 'desc';
}

function splitRouteList(value: string | undefined) {
  if (!value) {
    return [];
  }

  return value
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean);
}

function joinRouteList(values: string[]) {
  return values.length ? values.join(',') : '';
}

function inferPresetFromFilters(value: AuditClientFilterState): AuditQuickPresetKey {
  if (
    value.success === 'false' &&
    value.resourceTypes.join(',') === 'auth,session' &&
    value.actionKeywords.join(',') === 'auth,login' &&
    value.requestPathPrefixes.join(',') === '/api/auth'
  ) {
    return 'auth-failed';
  }
  if (value.success === 'false' && !value.actionKeywords.length && !value.resourceTypes.length) {
    return 'failed-operations';
  }
  if (value.actionPrefixes.join(',') === 'rbac.,role.,permission.') {
    return 'rbac-changes';
  }
  if (value.results.join(',') === 'DENIED') {
    return 'permission-denied';
  }
  if (
    value.actionKeywords.join(',') === 'delete,reset,grant,assign,revoke,remove,replace,update_role,update_permission'
  ) {
    return 'sensitive-ops';
  }
  if (value.riskLevels.join(',') === 'HIGH,CRITICAL') {
    return 'high-risk';
  }
  return 'all';
}
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

.column-grid {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
</style>
