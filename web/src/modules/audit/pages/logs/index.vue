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

      <section v-if="scopeState" class="audit-scope-card">
        <div class="audit-scope-card__header">
          <div>
            <p class="audit-scope-card__eyebrow">{{ t('audit.logList.scope.eyebrow') }}</p>
            <div class="audit-scope-card__title-row">
              <strong>{{ scopeState.appliedScope.name }}</strong>
              <t-tag theme="primary" variant="light-outline" size="small">
                {{ t('audit.logList.scope.lockedTag') }}
              </t-tag>
            </div>
            <p v-if="scopeState.appliedScope.description" class="audit-scope-card__description">
              {{ scopeState.appliedScope.description }}
            </p>
          </div>
          <t-space size="small" wrap>
            <t-button theme="default" variant="outline" size="small" @click="exitDrilldown">
              {{ t('audit.logList.scope.exitAction') }}
            </t-button>
            <t-button theme="primary" variant="outline" size="small" @click="convertScopeToFilters">
              {{ t('audit.logList.scope.convertAction') }}
            </t-button>
          </t-space>
        </div>

        <p class="audit-scope-card__hint">{{ t('audit.logList.scope.hint') }}</p>

        <t-collapse
          v-if="localizedScopeProjectionItems.length"
          :value="scopePanelValue"
          @change="handleScopePanelChange"
        >
          <t-collapse-panel value="projection" :header="t('audit.logList.scope.projectionTitle')">
            <div class="audit-scope-card__projection-list">
              <article
                v-for="item in localizedScopeProjectionItems"
                :key="item.key"
                class="audit-scope-card__projection-item"
              >
                <div class="audit-scope-card__projection-head">
                  <span>{{ item.label }}</span>
                  <t-tag v-if="item.locked" theme="warning" variant="light-outline" size="small">
                    {{ t('audit.logList.scope.readonlyTag') }}
                  </t-tag>
                </div>
                <div class="audit-scope-card__projection-values">
                  <t-tag
                    v-for="value in item.localizedValues"
                    :key="`${item.key}-${value}`"
                    theme="default"
                    variant="light-outline"
                    size="small"
                  >
                    {{ value }}
                  </t-tag>
                </div>
              </article>
            </div>
          </t-collapse-panel>
        </t-collapse>
      </section>

      <audit-filters
        v-model="filters"
        :active-preset="activePreset"
        :locked-fields="scopeOwnedFilterKeys"
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
import { computed, onActivated, onDeactivated, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { resolveLocalizedErrorMessage } from '@/modules/shared/localized-api-error';
import { ManagementEmptyState, ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import { describeCorrelationId, formatMessageWithCorrelation } from '@/shared/correlation';
import {
  buildRecentHoursLocalRange,
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
import { AUDIT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import { buildAuditLogsLocation, parseAuditLogsRouteQuery } from '../../contract/deep-link';
import {
  buildMonitorReturnLocation,
  resolveAuditNavigationContext,
  withMonitorOrigin,
} from '../../contract/navigation';
import {
  AUDIT_BUSINESS_CATEGORY,
  AUDIT_DRILLDOWN_SCOPE,
  type AuditQuickPresetKey,
  listAuditPresets,
} from '../../contract/presets';
import { AUDIT_TIME_PRESET, type AuditTimePreset } from '../../contract/time-presets';
import type { AuditFilterKey } from '../../shared/filter-definitions';
import type { AuditClientFilterState } from '../../shared/presentation';
import type {
  AppliedDrilldownScope,
  AuditDrilldownScope,
  AuditLogConvertibleFilters,
  AuditLogListItem,
  AuditLogQuery,
  DrilldownScopeProjection,
} from '../../types/audit';

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
const routePreset = ref<AuditTimePreset | ''>('');
const routeScope = ref<AuditDrilldownScope | ''>('');
const appliedScope = ref<AppliedDrilldownScope | null>(null);
const scopeProjection = ref<DrilldownScopeProjection | null>(null);
const convertibleFilters = ref<AuditLogConvertibleFilters | null>(null);
const scopePanelValue = ref<Array<string | number>>(['projection']);
const applyingRoute = ref(false);
const isRouteSyncActive = ref(true);
const navigationContext = computed(() => resolveAuditNavigationContext(route.query));
const monitorReturnLocation = computed(() => buildMonitorReturnLocation(route.query));
const activePreset = computed(() => inferPresetFromState(filters.value, routeScope.value));
const scopeState = computed(() =>
  appliedScope.value && scopeProjection.value
    ? {
        appliedScope: appliedScope.value,
        projection: scopeProjection.value,
        convertibleFilters: convertibleFilters.value,
      }
    : null,
);
const scopeOwnedFilterKeys = computed(() => mapOwnedFieldsToFilterKeys(appliedScope.value?.owned_fields ?? []));

const presetViews = computed(() =>
  listAuditPresets().map((preset) => ({
    key: preset.key,
    title: t(preset.titleKey),
  })),
);
const localizedScopeProjectionItems = computed(() =>
  (scopeState.value?.projection.items ?? []).map((item) => ({
    ...item,
    localizedValues: (item.values ?? [])
      .map((value) => formatScopeProjectionValue(item.key, value))
      .filter((value, index, values) => Boolean(value) && values.indexOf(value) === index),
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

const hasClientOnlyFilters = computed(() => false);

const displayRows = computed(() => rows.value);
const tableTotal = computed(() => total.value);
const footerSummary = computed(() =>
  hasClientOnlyFilters.value
    ? t('audit.logList.footerFiltered', { count: displayRows.value.length })
    : t('audit.logList.footerTotal', { count: total.value }),
);

const isCurrentAuditLogsRoute = computed(
  () => route.path === buildAuditLogsLocation({}).path || route.name === AUDIT_BOOTSTRAP_ROUTE.LOG_LIST.routeName,
);

function serializeRouteQuery(query: Record<string, unknown> | undefined) {
  return JSON.stringify(query ?? {});
}

function canSyncAuditRoute(reason: string) {
  const allowed = isRouteSyncActive.value && isCurrentAuditLogsRoute.value;

  if (!allowed) {
    logger.debug('skip audit route sync while page is inactive or route changed', {
      reason,
      routePath: route.path,
      routeName: route.name,
      isRouteSyncActive: isRouteSyncActive.value,
      isCurrentAuditLogsRoute: isCurrentAuditLogsRoute.value,
      query: route.query,
    });
  }

  return allowed;
}

function buildQuery(): AuditLogQuery {
  const sorter = getSingleSorter(filters.value.sorters);
  const query: AuditLogQuery = {
    page: pagination.value.current,
    page_size: pagination.value.pageSize,
  };
  if (routePreset.value) {
    query.preset = routePreset.value;
  }
  if (routeScope.value) {
    query.scope = routeScope.value;
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
  if (filters.value.businessCategory) {
    query.business_category = filters.value.businessCategory;
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
  if (filters.value.requestId) {
    query.request_id = filters.value.requestId;
  }
  if (filters.value.resourceId) {
    query.resource_id = filters.value.resourceId;
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
  if (filters.value.session) {
    query.session_id = filters.value.session;
  }
  if (filters.value.requestPathPrefixes.length) {
    query.request_path_prefixes = [...filters.value.requestPathPrefixes];
  }
  const explicitCreatedRange = filters.value.createdRange;
  if (explicitCreatedRange[0]) {
    query.created_from = localDateTimeToUtcIso(explicitCreatedRange[0]);
  }
  if (explicitCreatedRange[1]) {
    query.created_to = localDateTimeToUtcIso(explicitCreatedRange[1]);
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
    appliedScope.value = response.applied_scope ?? null;
    scopeProjection.value = response.scope_projection ?? null;
    convertibleFilters.value = response.convertible_filters ?? null;
  } catch (error) {
    if (requestSeq !== latestRequestSeq.value) {
      return;
    }
    rows.value = [];
    total.value = 0;
    appliedScope.value = null;
    scopeProjection.value = null;
    convertibleFilters.value = null;
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

function applyPreset(preset: AuditQuickPresetKey) {
  filters.value = createDefaultFilters();
  routePreset.value = resolvePresetTimeWindow(preset);
  routeScope.value = resolveScopeForPreset(preset);
  if (!routeScope.value) {
    filters.value.createdRange = buildPresetCreatedRange(routePreset.value);
  }
  pagination.value.current = 1;
  updateRouteQuery();
}

function handleSearch() {
  pagination.value.current = 1;
  updateRouteQuery();
}

function resetFilters() {
  filters.value = createDefaultFilters();
  routePreset.value = '';
  routeScope.value = '';
  pagination.value.current = 1;
  updateRouteQuery();
}

function exitDrilldown() {
  routeScope.value = '';
  pagination.value.current = 1;
  updateRouteQuery();
}

function convertScopeToFilters() {
  if (!convertibleFilters.value) {
    return;
  }

  routeScope.value = '';
  routePreset.value = convertibleFilters.value.preset ?? routePreset.value;
  applyConvertibleFilters(convertibleFilters.value);
  pagination.value.current = 1;
  updateRouteQuery();
}

function createDefaultFilters(): AuditClientFilterState {
  return {
    keyword: '',
    actor: '',
    success: 'all',
    action: '',
    actionPrefix: '',
    actionPrefixes: [],
    actionKeywords: [],
    requestPathPrefixes: [],
    source: '',
    businessCategory: '',
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
  routePreset.value = normalizePreset(query.preset);
  routeScope.value = normalizeScope(query.scope);
  const nextFilters: AuditClientFilterState = {
    ...createDefaultFilters(),
    keyword: query.keyword ?? '',
    actor: query.actor ?? '',
    success: query.success === 'true' ? 'true' : query.success === 'false' ? 'false' : 'all',
    action: query.action || '',
    actionPrefix: query.action_prefix || '',
    actionPrefixes: splitRouteList(query.action_prefixes),
    actionKeywords: splitRouteList(query.action_keywords),
    requestPathPrefixes: splitRouteList(query.request_path_prefixes),
    source: query.source || '',
    businessCategory: normalizeBusinessCategory(query.business_category),
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
}

function buildRouteQuery() {
  const explicitCreatedRange = filters.value.createdRange;
  const [createdFrom = '', createdTo = ''] = normalizePageStateRangeForRoute(explicitCreatedRange);
  const sorter = getSingleSorter(filters.value.sorters);

  return {
    preset: routePreset.value,
    scope: routeScope.value,
    keyword: filters.value.keyword,
    actor: filters.value.actor,
    success: filters.value.success === 'all' ? '' : filters.value.success,
    action: filters.value.action,
    action_prefix: filters.value.actionPrefix,
    action_prefixes: joinRouteList(filters.value.actionPrefixes),
    action_keywords: joinRouteList(filters.value.actionKeywords),
    request_path_prefixes: joinRouteList(filters.value.requestPathPrefixes),
    source: filters.value.source,
    business_category: filters.value.businessCategory,
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
  if (!canSyncAuditRoute('interactive-filter-sync')) {
    return;
  }

  const nextLocation = withMonitorOrigin(
    buildAuditLogsLocation(buildRouteQuery()),
    navigationContext.value.monitorOrigin,
  );
  const currentLocation = withMonitorOrigin(buildAuditLogsLocation(route.query), navigationContext.value.monitorOrigin);

  if (serializeRouteQuery(nextLocation.query) === serializeRouteQuery(currentLocation.query)) {
    await fetchAuditLogs();
    return;
  }

  logger.debug('replace audit route query from interactive filters', {
    reason: 'interactive-filter-sync',
    routePath: route.path,
    routeName: route.name,
    currentQuery: currentLocation.query,
    nextQuery: nextLocation.query,
  });
  await router.replace(nextLocation);
}

async function syncFromCurrentRoute(reason: string) {
  logger.debug('observe route query change for audit logs', {
    reason,
    routePath: route.path,
    routeName: route.name,
    isRouteSyncActive: isRouteSyncActive.value,
    isCurrentAuditLogsRoute: isCurrentAuditLogsRoute.value,
    applyingRoute: applyingRoute.value,
    query: route.query,
  });
  if (!canSyncAuditRoute(reason)) {
    return;
  }

  applyingRoute.value = true;
  try {
    applyRouteFilters();
  } finally {
    applyingRoute.value = false;
  }
  pagination.value.current = 1;
  const canonicalLocation = withMonitorOrigin(
    buildAuditLogsLocation(buildRouteQuery()),
    navigationContext.value.monitorOrigin,
  );
  const currentLocation = withMonitorOrigin(buildAuditLogsLocation(route.query), navigationContext.value.monitorOrigin);
  if (serializeRouteQuery(canonicalLocation.query) !== serializeRouteQuery(currentLocation.query)) {
    logger.debug('canonicalize audit route query after route change', {
      reason,
      routePath: route.path,
      routeName: route.name,
      currentQuery: currentLocation.query,
      canonicalQuery: canonicalLocation.query,
    });
    await router.replace(canonicalLocation);
    return;
  }
  await fetchAuditLogs();
}

watch(
  () => route.query,
  async () => {
    await syncFromCurrentRoute('route-query-watch');
  },
  { immediate: true },
);

onActivated(() => {
  isRouteSyncActive.value = true;
  void syncFromCurrentRoute('route-activated');
});

onDeactivated(() => {
  isRouteSyncActive.value = false;
});

function returnToMonitor() {
  if (!monitorReturnLocation.value) {
    return;
  }

  void router.push(monitorReturnLocation.value);
}

function normalizeSortOrder(value: string) {
  return value === 'asc' ? 'asc' : 'desc';
}

function normalizePreset(value?: string) {
  return value === AUDIT_TIME_PRESET.LAST_24H ||
    value === AUDIT_TIME_PRESET.LAST_7D ||
    value === AUDIT_TIME_PRESET.LAST_30D
    ? value
    : '';
}

function normalizeScope(value?: string): AuditDrilldownScope | '' {
  switch (value) {
    case AUDIT_DRILLDOWN_SCOPE.FAILED_OPERATIONS:
    case AUDIT_DRILLDOWN_SCOPE.HIGH_RISK_OPERATIONS:
    case AUDIT_DRILLDOWN_SCOPE.SENSITIVE_OPERATIONS:
    case AUDIT_DRILLDOWN_SCOPE.AUTH_FAILURES:
    case AUDIT_DRILLDOWN_SCOPE.PERMISSION_DENIALS:
    case AUDIT_DRILLDOWN_SCOPE.RBAC_CHANGES:
    case AUDIT_DRILLDOWN_SCOPE.CRITICAL_SECURITY:
      return value;
    default:
      return '';
  }
}

function normalizeBusinessCategory(value?: string): AuditClientFilterState['businessCategory'] {
  switch (value) {
    case AUDIT_BUSINESS_CATEGORY.FAILED_OPERATIONS:
    case AUDIT_BUSINESS_CATEGORY.HIGH_RISK_OPERATIONS:
    case AUDIT_BUSINESS_CATEGORY.SENSITIVE_OPERATIONS:
    case AUDIT_BUSINESS_CATEGORY.AUTH_FAILURES:
    case AUDIT_BUSINESS_CATEGORY.PERMISSION_DENIALS:
    case AUDIT_BUSINESS_CATEGORY.RBAC_CHANGES:
    case AUDIT_BUSINESS_CATEGORY.CRITICAL_SECURITY:
      return value;
    default:
      return '';
  }
}

function applyConvertibleFilters(next: AuditLogConvertibleFilters) {
  filters.value = {
    ...filters.value,
    source: next.source ?? '',
    businessCategory: normalizeBusinessCategory(next.business_category),
    success: next.success === true ? 'true' : next.success === false ? 'false' : 'all',
    actionPrefixes: next.action_prefixes ? [...next.action_prefixes] : [],
    actionKeywords: next.action_keywords ? [...next.action_keywords] : [],
    resourceTypes: next.resource_types ? [...next.resource_types] : [],
    requestPathPrefixes: next.request_path_prefixes ? [...next.request_path_prefixes] : [],
    results: next.results ? [...next.results] : [],
    riskLevels: next.risk_levels ? [...next.risk_levels] : [],
  };
}

function mapOwnedFieldsToFilterKeys(fields: string[]) {
  const mapped: AuditFilterKey[] = [];

  fields.forEach((field) => {
    switch (field) {
      case 'business_category':
        mapped.push('businessCategory');
        break;
      case 'action_keywords':
        mapped.push('actionKeywords');
        break;
      case 'action_prefixes':
        mapped.push('actionPrefixes');
        break;
      case 'resource_types':
        mapped.push('resourceTypes');
        break;
      case 'request_path_prefixes':
        mapped.push('requestPathPrefixes');
        break;
      case 'results':
        mapped.push('results');
        break;
      case 'risk_levels':
        mapped.push('riskLevels');
        break;
      case 'source':
        mapped.push('source');
        break;
      case 'success':
        mapped.push('success');
        break;
      default:
        break;
    }
  });

  return mapped;
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

function inferPresetFromState(value: AuditClientFilterState, scope: string): AuditQuickPresetKey {
  if (scope === AUDIT_DRILLDOWN_SCOPE.FAILED_OPERATIONS) {
    return 'failed-operations';
  }
  if (scope === AUDIT_DRILLDOWN_SCOPE.RBAC_CHANGES) {
    return 'rbac-changes';
  }
  if (scope === AUDIT_DRILLDOWN_SCOPE.PERMISSION_DENIALS) {
    return 'permission-denied';
  }
  if (scope === AUDIT_DRILLDOWN_SCOPE.SENSITIVE_OPERATIONS) {
    return 'sensitive-ops';
  }
  if (scope === AUDIT_DRILLDOWN_SCOPE.AUTH_FAILURES) {
    return 'auth-failed';
  }
  if (scope === AUDIT_DRILLDOWN_SCOPE.HIGH_RISK_OPERATIONS || scope === AUDIT_DRILLDOWN_SCOPE.CRITICAL_SECURITY) {
    return 'high-risk';
  }
  if (value.businessCategory === AUDIT_BUSINESS_CATEGORY.FAILED_OPERATIONS) {
    return 'failed-operations';
  }
  if (value.businessCategory === AUDIT_BUSINESS_CATEGORY.RBAC_CHANGES) {
    return 'rbac-changes';
  }
  if (value.businessCategory === AUDIT_BUSINESS_CATEGORY.PERMISSION_DENIALS) {
    return 'permission-denied';
  }
  if (value.businessCategory === AUDIT_BUSINESS_CATEGORY.SENSITIVE_OPERATIONS) {
    return 'sensitive-ops';
  }
  if (value.businessCategory === AUDIT_BUSINESS_CATEGORY.AUTH_FAILURES) {
    return 'auth-failed';
  }
  if (
    value.businessCategory === AUDIT_BUSINESS_CATEGORY.HIGH_RISK_OPERATIONS ||
    value.businessCategory === AUDIT_BUSINESS_CATEGORY.CRITICAL_SECURITY
  ) {
    return 'high-risk';
  }
  return 'all';
}

function resolveScopeForPreset(preset: AuditQuickPresetKey): AuditDrilldownScope | '' {
  switch (preset) {
    case 'failed-operations':
      return AUDIT_DRILLDOWN_SCOPE.FAILED_OPERATIONS;
    case 'rbac-changes':
      return AUDIT_DRILLDOWN_SCOPE.RBAC_CHANGES;
    case 'permission-denied':
      return AUDIT_DRILLDOWN_SCOPE.PERMISSION_DENIALS;
    case 'sensitive-ops':
      return AUDIT_DRILLDOWN_SCOPE.SENSITIVE_OPERATIONS;
    case 'auth-failed':
      return AUDIT_DRILLDOWN_SCOPE.AUTH_FAILURES;
    case 'high-risk':
      return AUDIT_DRILLDOWN_SCOPE.HIGH_RISK_OPERATIONS;
    default:
      return '';
  }
}

function handleScopePanelChange(value: Array<string | number>) {
  scopePanelValue.value = [...value];
}

function buildPresetCreatedRange(preset: AuditTimePreset | '') {
  const now = new Date();
  switch (preset) {
    case 'last_24h':
      return buildRecentHoursLocalRange(now, 24);
    case 'last_7d':
      return buildRecentHoursLocalRange(now, 24 * 7);
    case 'last_30d':
      return buildRecentHoursLocalRange(now, 24 * 30);
    default:
      return [];
  }
}

function resolvePresetTimeWindow(preset: AuditQuickPresetKey): AuditTimePreset | '' {
  return preset === 'all' ? '' : AUDIT_TIME_PRESET.LAST_24H;
}

function formatScopeProjectionValue(key: string, value: string) {
  const normalized = value.trim();
  if (!normalized) {
    return '';
  }

  if (key === 'business_category') {
    switch (normalized) {
      case AUDIT_BUSINESS_CATEGORY.FAILED_OPERATIONS:
        return t('audit.logList.businessCategory.failedOperations');
      case AUDIT_BUSINESS_CATEGORY.HIGH_RISK_OPERATIONS:
        return t('audit.logList.businessCategory.highRiskOperations');
      case AUDIT_BUSINESS_CATEGORY.SENSITIVE_OPERATIONS:
        return t('audit.logList.businessCategory.sensitiveOperations');
      case AUDIT_BUSINESS_CATEGORY.AUTH_FAILURES:
        return t('audit.logList.businessCategory.authFailures');
      case AUDIT_BUSINESS_CATEGORY.PERMISSION_DENIALS:
        return t('audit.logList.businessCategory.permissionDenials');
      case AUDIT_BUSINESS_CATEGORY.RBAC_CHANGES:
        return t('audit.logList.businessCategory.rbacChanges');
      case AUDIT_BUSINESS_CATEGORY.CRITICAL_SECURITY:
        return t('audit.logList.businessCategory.criticalSecurity');
      default:
        return t('audit.logList.scope.unknownValue');
    }
  }

  return normalized;
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

.audit-scope-card {
  background: linear-gradient(135deg, rgb(250 252 255 / 95%), rgb(255 250 243 / 95%)), var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-large);
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px 18px;
}

.audit-scope-card__header {
  align-items: flex-start;
  display: flex;
  gap: 16px;
  justify-content: space-between;
}

.audit-scope-card__eyebrow {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  margin: 0 0 4px;
}

.audit-scope-card__title-row {
  align-items: center;
  display: flex;
  gap: 8px;
}

.audit-scope-card__description,
.audit-scope-card__hint {
  color: var(--td-text-color-secondary);
  margin: 0;
}

.audit-scope-card__projection-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.audit-scope-card__projection-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.audit-scope-card__projection-head,
.audit-scope-card__projection-values {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.column-grid {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
</style>
