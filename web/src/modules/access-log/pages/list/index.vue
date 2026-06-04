<template>
  <div data-page-type="query-builder-list-detail">
    <management-page-content>
      <management-page-header :title="t('accessLog.page.title')" :description="t('accessLog.page.description')">
        <template #eyebrow>{{ t('menu.logCenter.title') }}</template>
        <template #actions>
          <t-button theme="default" variant="outline" @click="columnDrawerVisible = true">
            {{ t('accessLog.page.columnSettings') }}
          </t-button>
          <t-button theme="default" variant="outline" :loading="loading" @click="fetchAccessLogs">
            {{ t('accessLog.page.refresh') }}
          </t-button>
        </template>
      </management-page-header>

      <access-log-filters
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
        :title="t('accessLog.page.errorTitle')"
        :description="listError"
      >
        <template #actions>
          <t-button theme="primary" variant="outline" @click="fetchAccessLogs">
            {{ t('accessLog.page.retry') }}
          </t-button>
        </template>
      </management-empty-state>

      <access-log-table
        v-else
        v-model:current="pagination.current"
        v-model:page-size="pagination.pageSize"
        :description="t('accessLog.page.tableHint')"
        :empty-description="emptyDescription"
        :footer-summary="footerSummary"
        :loading="loading"
        :rows="displayRows"
        :summary="tableSummary"
        :total="tableTotal"
        :visible-column-keys="visibleColumnKeys"
        @detail="openDetail"
        @page-change="fetchAccessLogs"
      />
    </management-page-content>

    <t-drawer
      v-model:visible="columnDrawerVisible"
      :header="t('accessLog.page.columnSettings')"
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

    <access-log-detail-drawer v-model:visible="detailVisible" :record="detailRecord" />
  </div>
</template>
<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { useAuthSessionStore } from '@/modules/auth/store';
import { resolveLocalizedErrorMessage as resolveAccessLogErrorMessage } from '@/modules/shared/localized-api-error';
import { ManagementEmptyState, ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import {
  assignEncodedSorters,
  buildRecentHoursLocalRange,
  buildTodayLocalRange,
  createLogDetailErrorReporter,
  createLogListErrorReporter,
  createSingleSorter,
  decodeSorters,
  encodeSorters,
  localDateTimeToUtcIso,
  normalizePageStateRangeForRoute,
  normalizeRouteRangeForPageState,
  normalizeSorters,
  openLogDetailRow,
  restartLogListQuery,
} from '@/shared/observability';
import { createLogger as createModuleLogger } from '@/utils/logger';

import { getAccessLogDetail, getAccessLogs } from '../../api/access-log';
import AccessLogDetailDrawer from '../../components/AccessLogDetailDrawer.vue';
import AccessLogFilters from '../../components/AccessLogFilters.vue';
import AccessLogTable from '../../components/AccessLogTable.vue';
import { buildAccessLogLocation, parseAccessLogRouteQuery } from '../../contract/deep-link';
import { buildAccessLogSortOptions } from '../../shared/presentation';
import type { AccessLogFilterState, AccessLogItem, AccessLogQuery } from '../../types/access-log';

defineOptions({
  name: 'AccessLogListIndex',
});

type AccessLogPresetKey =
  | 'all'
  | 'todayErrors'
  | 'status4xx'
  | 'status5xx'
  | 'slowRequests'
  | 'currentUser'
  | 'lastHour';

const { t } = useI18n();
const logger = createModuleLogger('access-log.list');
const route = useRoute();
const router = useRouter();
const authSessionStore = useAuthSessionStore();

const loading = ref(false);
const listError = ref('');
const rows = ref<AccessLogItem[]>([]);
const total = ref(0);
const detailVisible = ref(false);
const detailRecord = ref<AccessLogItem | null>(null);
const applyingRoute = ref(false);
const activePreset = ref<AccessLogPresetKey>('all');
const columnDrawerVisible = ref(false);
const visibleColumnKeys = ref(['started_at', 'method', 'path', 'status_code', 'duration_ms', 'user', 'request_id']);
const pagination = ref({
  current: 1,
  pageSize: 20,
});
const filters = ref<AccessLogFilterState>(createDefaultFilters());
const deepLinkCorrelation = ref<'requestId' | null>(null);
const routeHydrated = ref(false);

const presetViews = computed(() => [
  { key: 'all' as const, title: t('accessLog.presets.all') },
  { key: 'todayErrors' as const, title: t('accessLog.presets.todayErrors') },
  { key: 'status4xx' as const, title: t('accessLog.presets.status4xx') },
  { key: 'status5xx' as const, title: t('accessLog.presets.status5xx') },
  { key: 'slowRequests' as const, title: t('accessLog.presets.slowRequests') },
  { key: 'currentUser' as const, title: t('accessLog.presets.currentUser') },
  { key: 'lastHour' as const, title: t('accessLog.presets.lastHour') },
]);
const sortOptions = computed(() => buildAccessLogSortOptions(t));
const columnSettingOptions = computed(() => [
  { label: t('accessLog.columns.occurredAt'), value: 'occurred_at' },
  { label: t('accessLog.columns.method'), value: 'method' },
  { label: t('accessLog.columns.path'), value: 'path' },
  { label: t('accessLog.columns.statusCode'), value: 'status_code' },
  { label: t('accessLog.columns.durationMs'), value: 'duration_ms' },
  { label: t('accessLog.columns.user'), value: 'user' },
  { label: t('accessLog.columns.requestId'), value: 'request_id' },
]);

const hasClientOnlyFilters = computed(() =>
  Boolean(filters.value.username && filters.value.username !== authSessionStore.userInfo.username),
);
const displayRows = computed(() => rows.value.filter((row) => matchesClientFilters(row, filters.value)));
const tableTotal = computed(() => (hasClientOnlyFilters.value ? displayRows.value.length : total.value));
const tableSummary = computed(() => t('accessLog.page.summary', { count: displayRows.value.length }));
const footerSummary = computed(() => t('accessLog.page.footerTotal', { count: total.value }));
const emptyDescription = computed(() => {
  if (deepLinkCorrelation.value === 'requestId') {
    return t('accessLog.page.emptyRequestDescription');
  }
  return t('accessLog.page.emptyDescription');
});
const reportListLoadError = createLogListErrorReporter<AccessLogItem>({
  fallbackMessage: () => t('accessLog.page.loadFailed'),
  listError,
  logger,
  logMessage: 'failed to fetch access logs',
  resolveMessage: (cause, fallback) => resolveAccessLogErrorMessage(t, cause, fallback),
  rows,
  total,
});
const reportDetailLoadError = createLogDetailErrorReporter({
  fallbackMessage: () => t('accessLog.page.loadFailed'),
  resolveMessage: (cause, fallback) => resolveAccessLogErrorMessage(t, cause, fallback),
});

function createDefaultFilters(): AccessLogFilterState {
  return {
    keyword: '',
    requestId: '',
    userId: '',
    username: '',
    method: '',
    path: '',
    pathMatch: 'exact',
    route: '',
    statusCode: '',
    durationMinMs: '',
    durationMaxMs: '',
    startedRange: [],
    occurredRange: [],
    sorters: createSingleSorter('started_at', 'desc'),
  };
}

function buildQuery(): AccessLogQuery {
  const query: AccessLogQuery = {
    page: pagination.value.current,
    page_size: pagination.value.pageSize,
    path_match: filters.value.pathMatch,
  };
  assignEncodedSorters(query, filters.value.sorters, sortOptions.value);

  if (filters.value.keyword) query.keyword = filters.value.keyword;
  if (filters.value.requestId) query.request_id = filters.value.requestId;
  if (filters.value.userId) query.user_id = Number(filters.value.userId);
  if (filters.value.username) query.username = filters.value.username;
  if (filters.value.method) query.method = filters.value.method;
  if (filters.value.path) query.path = filters.value.path;
  if (filters.value.route) query.route = filters.value.route;
  if (filters.value.statusCode === '4xx' || filters.value.statusCode === '5xx') {
    query.status_group = filters.value.statusCode;
  } else if (filters.value.statusCode) {
    query.status_code = Number(filters.value.statusCode);
  }
  if (filters.value.durationMinMs) query.duration_min_ms = Number(filters.value.durationMinMs);
  if (filters.value.durationMaxMs) query.duration_max_ms = Number(filters.value.durationMaxMs);
  if (filters.value.startedRange[0]) query.started_from = localDateTimeToUtcIso(filters.value.startedRange[0]);
  if (filters.value.startedRange[1]) query.started_to = localDateTimeToUtcIso(filters.value.startedRange[1]);
  if (filters.value.occurredRange[0]) query.occurred_from = localDateTimeToUtcIso(filters.value.occurredRange[0]);
  if (filters.value.occurredRange[1]) query.occurred_to = localDateTimeToUtcIso(filters.value.occurredRange[1]);
  return query;
}

async function fetchAccessLogs() {
  loading.value = true;
  listError.value = '';

  try {
    const response = await getAccessLogs(buildQuery());
    rows.value = response.items;
    total.value = response.total;
  } catch (error) {
    reportListLoadError(error);
  } finally {
    loading.value = false;
  }
}

async function openDetail(row: AccessLogItem) {
  await openLogDetailRow(row, getAccessLogDetail, detailRecord, detailVisible, reportDetailLoadError);
}

function resetFilters() {
  filters.value = createDefaultFilters();
  restartQuery();
}

function handleSearch() {
  restartQuery();
}

function applyPreset(preset: AccessLogPresetKey) {
  filters.value = {
    ...createDefaultFilters(),
    ...buildPresetFilters(preset),
    requestId: filters.value.requestId,
    sorters: filters.value.sorters,
  };
  restartQuery(preset);
}

function restartQuery(preset?: AccessLogPresetKey) {
  restartLogListQuery({ activePreset, pagination, preset, updateRouteQuery });
}

function buildPresetFilters(preset: AccessLogPresetKey): Partial<AccessLogFilterState> {
  const now = new Date();
  const currentUsername = authSessionStore.userInfo.username;
  switch (preset) {
    case 'todayErrors': {
      return { statusCode: '4xx', startedRange: buildTodayLocalRange(now) };
    }
    case 'status4xx':
      return { statusCode: '4xx' };
    case 'status5xx':
      return { statusCode: '5xx' };
    case 'slowRequests':
      return { durationMinMs: '3000' };
    case 'currentUser':
      return { username: currentUsername || '' };
    case 'lastHour': {
      return { startedRange: buildRecentHoursLocalRange(now, 1) };
    }
    default:
      return {};
  }
}

function applyRouteFilters() {
  const {
    keyword = '',
    request_id: requestId = '',
    user_id: userId = '',
    username = '',
    method = '',
    path = '',
    path_match: pathMatch = '',
    route: routeValue = '',
    status_code: statusCode = '',
    status_group: statusGroup = '',
    duration_min_ms: durationMinMs = '',
    duration_max_ms: durationMaxMs = '',
    started_from: startedFrom = '',
    started_to: startedTo = '',
    occurred_from: occurredFrom = '',
    occurred_to: occurredTo = '',
    sort = [],
  } = parseAccessLogRouteQuery(route.query);
  const parsedSorters = decodeSorters(sort, normalizeSortBy, normalizeSortOrder);
  const nextStatusCode = statusGroup || statusCode;
  filters.value = {
    ...filters.value,
    keyword,
    requestId,
    userId,
    username,
    method,
    path,
    pathMatch: pathMatch === 'prefix' ? 'prefix' : 'exact',
    route: routeValue,
    statusCode: nextStatusCode,
    durationMinMs,
    durationMaxMs,
    startedRange: normalizeRouteRangeForPageState([startedFrom, startedTo]),
    occurredRange: normalizeRouteRangeForPageState([occurredFrom, occurredTo]),
    sorters: (() => {
      const normalized = normalizeSorters(parsedSorters, sortOptions.value);
      return normalized.length ? normalized : createSingleSorter('started_at', 'desc');
    })(),
  };
  deepLinkCorrelation.value = requestId ? 'requestId' : null;
  routeHydrated.value = true;
}

function buildRouteQuery() {
  const normalizedSorters = normalizeSorters(filters.value.sorters, sortOptions.value);
  const [startedFrom = '', startedTo = ''] = normalizePageStateRangeForRoute(filters.value.startedRange);
  const [occurredFrom = '', occurredTo = ''] = normalizePageStateRangeForRoute(filters.value.occurredRange);
  const isGroupedStatusCode = filters.value.statusCode === '4xx' || filters.value.statusCode === '5xx';
  return buildAccessLogLocation({
    keyword: filters.value.keyword,
    request_id: filters.value.requestId,
    user_id: filters.value.userId,
    username: filters.value.username,
    method: filters.value.method,
    path: filters.value.path,
    path_match: filters.value.pathMatch === 'prefix' ? filters.value.pathMatch : '',
    route: filters.value.route,
    status_code: isGroupedStatusCode ? '' : filters.value.statusCode,
    status_group: isGroupedStatusCode ? filters.value.statusCode : '',
    duration_min_ms: filters.value.durationMinMs,
    duration_max_ms: filters.value.durationMaxMs,
    started_from: startedFrom,
    started_to: startedTo,
    occurred_from: occurredFrom,
    occurred_to: occurredTo,
    sort: encodeSorters(normalizedSorters, sortOptions.value),
  });
}

async function updateRouteQuery() {
  if (applyingRoute.value) {
    return;
  }

  const targetLocation = buildRouteQuery();
  const currentKeyword = typeof route.query.keyword === 'string' ? route.query.keyword : '';
  const currentRequestId = typeof route.query.request_id === 'string' ? route.query.request_id : '';
  const currentUserId = typeof route.query.user_id === 'string' ? route.query.user_id : '';
  const currentUsername = typeof route.query.username === 'string' ? route.query.username : '';
  const currentMethod = typeof route.query.method === 'string' ? route.query.method : '';
  const currentPath = typeof route.query.path === 'string' ? route.query.path : '';
  const currentPathMatch = typeof route.query.path_match === 'string' ? route.query.path_match : '';
  const currentRouteValue = typeof route.query.route === 'string' ? route.query.route : '';
  const currentStatusCode = typeof route.query.status_code === 'string' ? route.query.status_code : '';
  const currentStatusGroup = typeof route.query.status_group === 'string' ? route.query.status_group : '';
  const currentDurationMinMs = typeof route.query.duration_min_ms === 'string' ? route.query.duration_min_ms : '';
  const currentDurationMaxMs = typeof route.query.duration_max_ms === 'string' ? route.query.duration_max_ms : '';
  const currentStartedFrom = typeof route.query.started_from === 'string' ? route.query.started_from : '';
  const currentStartedTo = typeof route.query.started_to === 'string' ? route.query.started_to : '';
  const currentOccurredFrom = typeof route.query.occurred_from === 'string' ? route.query.occurred_from : '';
  const currentOccurredTo = typeof route.query.occurred_to === 'string' ? route.query.occurred_to : '';
  const currentSort = Array.isArray(route.query.sort)
    ? route.query.sort.map((item) => String(item))
    : typeof route.query.sort === 'string'
      ? [route.query.sort]
      : [];
  const nextQuery = targetLocation.query as Record<string, string | string[]>;
  const nextSort = Array.isArray(nextQuery.sort) ? nextQuery.sort : nextQuery.sort ? [nextQuery.sort] : [];

  if (
    currentKeyword === (nextQuery.keyword ?? '') &&
    currentRequestId === (nextQuery.request_id ?? '') &&
    currentUserId === (nextQuery.user_id ?? '') &&
    currentUsername === (nextQuery.username ?? '') &&
    currentMethod === (nextQuery.method ?? '') &&
    currentPath === (nextQuery.path ?? '') &&
    currentPathMatch === (nextQuery.path_match ?? '') &&
    currentRouteValue === (nextQuery.route ?? '') &&
    currentStatusCode === (nextQuery.status_code ?? '') &&
    currentStatusGroup === (nextQuery.status_group ?? '') &&
    currentDurationMinMs === (nextQuery.duration_min_ms ?? '') &&
    currentDurationMaxMs === (nextQuery.duration_max_ms ?? '') &&
    currentStartedFrom === (nextQuery.started_from ?? '') &&
    currentStartedTo === (nextQuery.started_to ?? '') &&
    currentOccurredFrom === (nextQuery.occurred_from ?? '') &&
    currentOccurredTo === (nextQuery.occurred_to ?? '') &&
    JSON.stringify(currentSort) === JSON.stringify(nextSort)
  ) {
    await fetchAccessLogs();
    return;
  }

  await router.replace(targetLocation);
}

function matchesClientFilters(row: AccessLogItem, state: AccessLogFilterState) {
  if (state.requestId && row.request_id !== state.requestId) {
    return false;
  }
  if (state.userId && String(row.user_id ?? '') !== state.userId) {
    return false;
  }
  if (state.username && !(row.username || '').toLowerCase().includes(state.username.toLowerCase())) {
    return false;
  }
  if (state.method && row.method !== state.method) {
    return false;
  }
  if (state.path) {
    const candidate = row.path || '';
    if (state.pathMatch === 'prefix' ? !candidate.startsWith(state.path) : candidate !== state.path) {
      return false;
    }
  }
  if (state.statusCode) {
    if (state.statusCode === '4xx') {
      if (row.status_code < 400 || row.status_code >= 500) {
        return false;
      }
    } else if (state.statusCode === '5xx') {
      if (row.status_code < 500 || row.status_code >= 600) {
        return false;
      }
    } else if (row.status_code !== Number(state.statusCode)) {
      return false;
    }
  }
  if (state.durationMinMs && row.duration_ms < Number(state.durationMinMs)) {
    return false;
  }
  if (state.durationMaxMs && row.duration_ms > Number(state.durationMaxMs)) {
    return false;
  }

  return true;
}

watch(
  () => [
    route.query.keyword,
    route.query.request_id,
    route.query.user_id,
    route.query.username,
    route.query.method,
    route.query.path,
    route.query.path_match,
    route.query.route,
    route.query.status_code,
    route.query.status_group,
    route.query.duration_min_ms,
    route.query.duration_max_ms,
    route.query.started_from,
    route.query.started_to,
    route.query.occurred_from,
    route.query.occurred_to,
    route.query.sort,
  ],
  () => {
    applyingRoute.value = true;
    try {
      applyRouteFilters();
    } finally {
      applyingRoute.value = false;
    }
    pagination.value.current = 1;
    void fetchAccessLogs();
  },
  { immediate: true },
);

function normalizeSortBy(value: string) {
  return value === 'occurred_at' || value === 'duration_ms' || value === 'status_code'
    ? value
    : value === 'started_at'
      ? 'started_at'
      : '';
}

function normalizeSortOrder(value: string) {
  return value === 'asc' ? 'asc' : 'desc';
}
</script>
<style scoped lang="less">
.column-grid {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
</style>
