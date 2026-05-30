<template>
  <div data-page-type="list-form-detail">
    <management-page-content>
      <management-page-header :title="t('accessLog.page.title')" :description="t('accessLog.page.description')">
        <template #eyebrow>{{ t('menu.logCenter.title') }}</template>
        <template #actions>
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
        @detail="openDetail"
        @page-change="fetchAccessLogs"
      />
    </management-page-content>

    <access-log-detail-drawer v-model:visible="detailVisible" :record="detailRecord" />
  </div>
</template>
<script setup lang="ts">
import { MessagePlugin } from 'tdesign-vue-next';
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { useAuthSessionStore } from '@/modules/auth/store';
import { resolveLocalizedErrorMessage as resolveAccessLogErrorMessage } from '@/modules/shared/localized-api-error';
import { ManagementEmptyState, ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import { createLogger as createModuleLogger } from '@/utils/logger';

import { getAccessLogDetail, getAccessLogs } from '../../api/access-log';
import AccessLogDetailDrawer from '../../components/AccessLogDetailDrawer.vue';
import AccessLogFilters from '../../components/AccessLogFilters.vue';
import AccessLogTable from '../../components/AccessLogTable.vue';
import { buildAccessLogLocation, parseAccessLogRouteQuery } from '../../contract/deep-link';
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
const pagination = ref({
  current: 1,
  pageSize: 20,
});
const filters = ref<AccessLogFilterState>(createDefaultFilters());
const deepLinkCorrelation = ref<'requestId' | 'traceId' | null>(null);

const presetViews = computed(() => [
  { key: 'all' as const, title: t('accessLog.presets.all') },
  { key: 'todayErrors' as const, title: t('accessLog.presets.todayErrors') },
  { key: 'status4xx' as const, title: t('accessLog.presets.status4xx') },
  { key: 'status5xx' as const, title: t('accessLog.presets.status5xx') },
  { key: 'slowRequests' as const, title: t('accessLog.presets.slowRequests') },
  { key: 'currentUser' as const, title: t('accessLog.presets.currentUser') },
  { key: 'lastHour' as const, title: t('accessLog.presets.lastHour') },
]);

const displayRows = computed(() => rows.value.filter((row) => matchesClientFilters(row, filters.value)));
const tableTotal = computed(() => displayRows.value.length);
const tableSummary = computed(() => t('accessLog.page.summary', { count: displayRows.value.length }));
const footerSummary = computed(() => t('accessLog.page.footerTotal', { count: total.value }));
const emptyDescription = computed(() => {
  if (deepLinkCorrelation.value === 'requestId') {
    return t('accessLog.page.emptyRequestDescription');
  }
  if (deepLinkCorrelation.value === 'traceId') {
    return t('accessLog.page.emptyTraceDescription');
  }
  return t('accessLog.page.emptyDescription');
});

function createDefaultFilters(): AccessLogFilterState {
  return {
    keyword: '',
    requestId: '',
    traceId: '',
    userId: '',
    username: '',
    method: '',
    path: '',
    pathMatch: 'exact',
    route: '',
    statusCode: '',
    durationMinMs: '',
    durationMaxMs: '',
    occurredRange: [],
    sortBy: 'occurred_at',
    sortOrder: 'desc',
  };
}

function buildQuery(): AccessLogQuery {
  const query: AccessLogQuery = {
    page: pagination.value.current,
    page_size: pagination.value.pageSize,
    sort_by: filters.value.sortBy,
    sort_order: filters.value.sortOrder,
    path_match: filters.value.pathMatch,
  };

  if (filters.value.requestId) query.request_id = filters.value.requestId;
  if (filters.value.traceId) query.trace_id = filters.value.traceId;
  if (filters.value.userId) query.user_id = Number(filters.value.userId);
  if (filters.value.username) query.username = filters.value.username;
  if (filters.value.method) query.method = filters.value.method;
  if (filters.value.path) query.path = filters.value.path;
  if (filters.value.route) query.route = filters.value.route;
  if (filters.value.statusCode) query.status_code = Number(filters.value.statusCode);
  if (filters.value.durationMinMs) query.duration_min_ms = Number(filters.value.durationMinMs);
  if (filters.value.durationMaxMs) query.duration_max_ms = Number(filters.value.durationMaxMs);
  if (filters.value.occurredRange[0]) query.occurred_from = normalizeOccurredAt(filters.value.occurredRange[0]);
  if (filters.value.occurredRange[1]) query.occurred_to = normalizeOccurredAt(filters.value.occurredRange[1]);

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
    rows.value = [];
    total.value = 0;
    logger.error('failed to fetch access logs', error);
    listError.value = resolveAccessLogErrorMessage(t, error, t('accessLog.page.loadFailed'));
    MessagePlugin.error(listError.value);
  } finally {
    loading.value = false;
  }
}

async function openDetail(row: AccessLogItem) {
  try {
    detailRecord.value = await getAccessLogDetail(Number(row.id));
    detailVisible.value = true;
  } catch (error) {
    MessagePlugin.error(resolveAccessLogErrorMessage(t, error, t('accessLog.page.loadFailed')));
  }
}

function resetFilters() {
  activePreset.value = 'all';
  filters.value = createDefaultFilters();
  pagination.value.current = 1;
  void updateRouteQuery();
}

function handleSearch() {
  activePreset.value = 'all';
  pagination.value.current = 1;
  void updateRouteQuery();
}

function applyPreset(preset: AccessLogPresetKey) {
  activePreset.value = preset;
  filters.value = {
    ...createDefaultFilters(),
    ...buildPresetFilters(preset),
    requestId: filters.value.requestId,
    traceId: filters.value.traceId,
  };
  pagination.value.current = 1;
  void updateRouteQuery();
}

function buildPresetFilters(preset: AccessLogPresetKey): Partial<AccessLogFilterState> {
  const now = new Date();
  const currentUsername = authSessionStore.userInfo.username;
  switch (preset) {
    case 'todayErrors': {
      const start = new Date(now);
      start.setHours(0, 0, 0, 0);
      return { statusCode: '400', occurredRange: [start.toISOString(), now.toISOString()] };
    }
    case 'status4xx':
      return { statusCode: '400' };
    case 'status5xx':
      return { statusCode: '500' };
    case 'slowRequests':
      return { durationMinMs: '3000' };
    case 'currentUser':
      return { username: currentUsername || '' };
    case 'lastHour': {
      const start = new Date(now.getTime() - 60 * 60 * 1000);
      return { occurredRange: [start.toISOString(), now.toISOString()] };
    }
    default:
      return {};
  }
}

function normalizeOccurredAt(value: string) {
  const date = new Date(value.replace(' ', 'T'));
  return Number.isFinite(date.getTime()) ? date.toISOString() : value;
}

function applyRouteFilters() {
  const { request_id: requestId = '', trace_id: traceId = '' } = parseAccessLogRouteQuery(route.query);
  filters.value = {
    ...filters.value,
    requestId,
    traceId,
  };
  deepLinkCorrelation.value = requestId ? 'requestId' : traceId ? 'traceId' : null;
}

function buildRouteQuery() {
  return buildAccessLogLocation({
    request_id: filters.value.requestId,
    trace_id: filters.value.traceId,
  });
}

async function updateRouteQuery() {
  if (applyingRoute.value) {
    return;
  }

  const targetLocation = buildRouteQuery();
  const currentRequestId = typeof route.query.request_id === 'string' ? route.query.request_id : '';
  const currentTraceId = typeof route.query.trace_id === 'string' ? route.query.trace_id : '';
  const nextQuery = targetLocation.query as Record<string, string>;

  if (currentRequestId === (nextQuery.request_id ?? '') && currentTraceId === (nextQuery.trace_id ?? '')) {
    await fetchAccessLogs();
    return;
  }

  await router.replace(targetLocation);
}

function matchesClientFilters(row: AccessLogItem, state: AccessLogFilterState) {
  if (state.keyword) {
    const keyword = state.keyword.toLowerCase();
    const haystack = [row.request_id, row.trace_id, row.path, row.route, row.username, row.method]
      .filter(Boolean)
      .join(' ')
      .toLowerCase();
    if (!haystack.includes(keyword)) {
      return false;
    }
  }

  if (state.requestId && row.request_id !== state.requestId) {
    return false;
  }
  if (state.traceId && row.trace_id !== state.traceId) {
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
    if (state.statusCode === '400' && (row.status_code < 400 || row.status_code >= 500)) {
      return false;
    }
    if (state.statusCode === '500' && row.status_code < 500) {
      return false;
    }
    if (state.statusCode !== '400' && state.statusCode !== '500' && row.status_code !== Number(state.statusCode)) {
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
  () => [route.query.request_id, route.query.trace_id],
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
</script>
