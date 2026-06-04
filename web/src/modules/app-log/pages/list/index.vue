<template>
  <div data-page-type="list-form-detail">
    <management-page-content>
      <management-page-header :title="t('appLog.page.title')" :description="t('appLog.page.description')">
        <template #eyebrow>{{ t('menu.logCenter.title') }}</template>
        <template #actions>
          <t-button theme="default" variant="outline" :loading="loading" @click="fetchAppLogs">
            {{ t('appLog.page.refresh') }}
          </t-button>
        </template>
      </management-page-header>

      <app-log-filters v-model="filters" :loading="loading" @reset="resetFilters" @search="handleSearch" />

      <management-empty-state
        v-if="listError && !loading"
        tone="error"
        :title="t('appLog.page.errorTitle')"
        :description="listError"
      >
        <template #actions>
          <t-button theme="primary" variant="outline" @click="fetchAppLogs">
            {{ t('appLog.page.retry') }}
          </t-button>
        </template>
      </management-empty-state>

      <app-log-table
        v-else
        v-model:current="pagination.current"
        v-model:page-size="pagination.pageSize"
        :description="t('appLog.page.tableHint')"
        :empty-description="t('appLog.page.emptyDescription')"
        :footer-summary="footerSummary"
        :loading="loading"
        :rows="rows"
        :summary="tableSummary"
        :total="total"
        @detail="openDetail"
        @page-change="fetchAppLogs"
      />
    </management-page-content>

    <app-log-detail-drawer v-model:visible="detailVisible" :record="detailRecord" />
  </div>
</template>
<script setup lang="ts">
import { MessagePlugin } from 'tdesign-vue-next';
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { resolveLocalizedErrorMessage as resolveAppLogErrorMessage } from '@/modules/shared/localized-api-error';
import { ManagementEmptyState, ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import {
  localDateTimeToUtcIso,
  normalizePageStateRangeForRoute,
  normalizeRouteRangeForPageState,
} from '@/shared/observability';
import { createLogger as createModuleLogger } from '@/utils/logger';

import { getAppLogDetail, getAppLogs } from '../../api/app-log';
import AppLogDetailDrawer from '../../components/AppLogDetailDrawer.vue';
import AppLogFilters from '../../components/AppLogFilters.vue';
import AppLogTable from '../../components/AppLogTable.vue';
import { buildAppLogLocation, parseAppLogRouteQuery } from '../../contract/deep-link';
import type { AppLogFilterState, AppLogItem, AppLogQuery } from '../../types/app-log';

defineOptions({
  name: 'AppLogListIndex',
});

const { t } = useI18n();
const logger = createModuleLogger('app-log.list');
const route = useRoute();
const router = useRouter();

const loading = ref(false);
const listError = ref('');
const rows = ref<AppLogItem[]>([]);
const total = ref(0);
const detailVisible = ref(false);
const detailRecord = ref<AppLogItem | null>(null);
const applyingRoute = ref(false);
const routeHydrated = ref(false);
const pagination = ref({
  current: 1,
  pageSize: 20,
});
const filters = ref<AppLogFilterState>(createDefaultFilters());

const tableSummary = computed(() => t('appLog.page.summary', { count: rows.value.length }));
const footerSummary = computed(() => t('appLog.page.footerTotal', { count: total.value }));

function createDefaultFilters(): AppLogFilterState {
  return {
    keyword: '',
    occurredRange: [],
    severity: '',
    component: '',
    operation: '',
    requestId: '',
    traceId: '',
    message: '',
    error: '',
  };
}

function buildQuery(): AppLogQuery {
  const query: AppLogQuery = {
    page: pagination.value.current,
    page_size: pagination.value.pageSize,
  };

  if (filters.value.keyword) query.keyword = filters.value.keyword;
  if (filters.value.severity) query.severity = filters.value.severity;
  if (filters.value.component) query.component = filters.value.component;
  if (filters.value.operation) query.operation = filters.value.operation;
  if (filters.value.requestId) query.request_id = filters.value.requestId;
  if (filters.value.traceId) query.trace_id = filters.value.traceId;
  if (filters.value.message) query.message = filters.value.message;
  if (filters.value.error) query.error = filters.value.error;
  for (const [index, key] of ['occurred_from', 'occurred_to'].entries()) {
    const localValue = filters.value.occurredRange[index];
    if (localValue) {
      query[key as 'occurred_from' | 'occurred_to'] = localDateTimeToUtcIso(localValue);
    }
  }
  return query;
}

async function fetchAppLogs() {
  loading.value = true;
  listError.value = '';

  try {
    applyListResponse(await getAppLogs(buildQuery()));
  } catch (error) {
    handleListLoadError(error);
  } finally {
    loading.value = false;
  }
}

function applyListResponse(response: Awaited<ReturnType<typeof getAppLogs>>) {
  rows.value = response.items;
  total.value = response.total;
}

function handleListLoadError(error: unknown) {
  rows.value = [];
  total.value = 0;
  logger.error('failed to fetch app logs', error);
  listError.value = resolveAppLogErrorMessage(t, error, t('appLog.page.loadFailed'));
  MessagePlugin.error(listError.value);
}

async function openDetail(row: AppLogItem) {
  try {
    detailRecord.value = await getAppLogDetail(Number(row.id));
    detailVisible.value = true;
  } catch (error) {
    MessagePlugin.error(resolveAppLogErrorMessage(t, error, t('appLog.page.loadFailed')));
  }
}

function resetFilters() {
  filters.value = createDefaultFilters();
  pagination.value.current = 1;
  void updateRouteQuery();
}

function handleSearch() {
  pagination.value.current = 1;
  void updateRouteQuery();
}

function applyRouteFilters() {
  const {
    keyword = '',
    occurred_from: occurredFrom = '',
    occurred_to: occurredTo = '',
    severity = '',
    component = '',
    operation = '',
    request_id: requestId = '',
    trace_id: traceId = '',
    message = '',
    error = '',
  } = parseAppLogRouteQuery(route.query);

  filters.value = {
    keyword,
    occurredRange: normalizeRouteRangeForPageState([occurredFrom, occurredTo]),
    severity:
      severity === 'debug' || severity === 'info' || severity === 'warn' || severity === 'error' ? severity : '',
    component,
    operation,
    requestId,
    traceId,
    message,
    error,
  };
}

async function updateRouteQuery() {
  if (applyingRoute.value) {
    return;
  }

  const occurredRange = normalizePageStateRangeForRoute(filters.value.occurredRange);
  await router.replace(
    buildAppLogLocation({
      keyword: filters.value.keyword,
      occurred_from: occurredRange[0],
      occurred_to: occurredRange[1],
      severity: filters.value.severity,
      component: filters.value.component,
      operation: filters.value.operation,
      request_id: filters.value.requestId,
      trace_id: filters.value.traceId,
      message: filters.value.message,
      error: filters.value.error,
    }),
  );
  await fetchAppLogs();
}

watch(
  () => route.query,
  () => {
    applyingRoute.value = true;
    applyRouteFilters();
    applyingRoute.value = false;
    if (!routeHydrated.value) {
      routeHydrated.value = true;
      void fetchAppLogs();
    }
  },
  { immediate: true },
);
</script>
