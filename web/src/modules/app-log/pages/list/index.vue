<template>
  <advanced-query-list-page
    page-type="log-audit"
    title-key="appLog.page.title"
    :title="t('appLog.page.title')"
    description-key="appLog.page.description"
    :description="t('appLog.page.description')"
    :error-message="listError"
    :error-title="t('appLog.page.errorTitle')"
    :loading="loading"
    compact-header
    :reload-label="t('appLog.page.refresh')"
    :retry-label="t('appLog.page.retry')"
    :show-header-reload="false"
    :source="{ labelKey: 'menu.logCenter.title', fallback: t('menu.logCenter.title') }"
    @reload="fetchAppLogs"
  >
    <template #filters>
      <app-log-filters
        v-model="filters"
        :active-preset="activePreset"
        :loading="loading"
        :presets="presetViews"
        @apply-preset="applyPreset"
        @reset="resetFilters"
        @search="handleSearch"
      />
    </template>
    <template #table>
      <app-log-table
        v-model:current="pagination.current"
        v-model:page-size="pagination.pageSize"
        :empty-description="t('appLog.page.emptyDescription')"
        :footer-summary="footerSummary"
        :loading="loading"
        :rows="rows"
        :selected-row-keys="selectedRowKeys"
        :total="total"
        :visible-column-keys="visibleColumnKeys"
        @delete="confirmDeleteOne"
        @detail="openDetail"
        @page-change="fetchAppLogs"
        @select-change="handleSelectChange"
      >
        <template #toolbar>
          <table-view-toolbar
            :column-settings-label="t('appLog.page.columnSettings')"
            :refresh-label="t('appLog.page.refresh')"
            :refresh-loading="loading"
            @column-settings="columnDrawerVisible = true"
            @refresh="fetchAppLogs"
          />
        </template>
        <template #batch>
          <div v-if="selectedRowKeys.length > 0" class="app-log-batch-bar">
            <span>{{ t('appLog.batch.selected', { count: selectedRowKeys.length }) }}</span>
            <div class="app-log-batch-bar__actions">
              <t-button
                v-permission="permissionCodes.DELETE"
                size="small"
                theme="danger"
                variant="outline"
                :loading="deleting"
                @click="confirmBatchDelete"
              >
                {{ t('appLog.actions.batchDelete') }}
              </t-button>
              <t-button size="small" theme="default" variant="text" @click="selectedRowKeys = []">
                {{ t('appLog.batch.cancelSelection') }}
              </t-button>
            </div>
          </div>
        </template>
      </app-log-table>
    </template>
    <template #detail>
      <advanced-query-column-drawer
        v-model:visible="columnDrawerVisible"
        v-model:selected-keys="visibleColumnKeys"
        :columns="columnSettingOptions"
        :default-selected-keys="DEFAULT_VISIBLE_COLUMNS"
        :presets-label="t('appLog.columnViews.label')"
        :reset-label="t('appLog.columnViews.resetDefault')"
        :title="t('appLog.page.columnSettings')"
        :view-presets="columnViewPresets"
      />
      <app-log-detail-drawer v-model:visible="detailVisible" :initial-tab="detailInitialTab" :record="detailRecord" />
    </template>
  </advanced-query-list-page>
</template>
<script setup lang="ts">
import { DialogPlugin, MessagePlugin } from 'tdesign-vue-next';
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { TableViewToolbar } from '@/shared/components/management';
import { AdvancedQueryColumnDrawer, AdvancedQueryListPage } from '@/shared/components/query-list';
import { resolveLocalizedErrorMessage as resolveAppLogErrorMessage } from '@/shared/localized-api-error';
import {
  assignEncodedSorters,
  buildRecentHoursLocalRange,
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
import { usePermissionStore } from '@/store';
import { createLogger as createModuleLogger } from '@/utils/logger';

import { deleteAppLog, deleteAppLogs, getAppLogDetail, getAppLogs } from '../../api/app-log';
import AppLogDetailDrawer from '../../components/AppLogDetailDrawer.vue';
import AppLogFilters from '../../components/AppLogFilters.vue';
import AppLogTable from '../../components/AppLogTable.vue';
import { buildAppLogLocation, parseAppLogRouteQuery } from '../../contract/deep-link';
import { APP_LOG_PERMISSION_CODE } from '../../contract/permissions';
import type { AppLogFilterState, AppLogItem, AppLogQuery, AppLogSortBy, AppLogSortOrder } from '../../types/app-log';

defineOptions({
  name: 'AppLogListIndex',
});

const { t } = useI18n();
const logger = createModuleLogger('app-log.list');
const route = useRoute();
const router = useRouter();
const permissionStore = usePermissionStore();

type AppLogPresetKey = 'all' | 'errors' | 'warnings' | 'lastHour';
const DEFAULT_VISIBLE_COLUMNS = ['occurred_at', 'severity', 'component', 'operation', 'message'];
const TROUBLESHOOTING_VISIBLE_COLUMNS = [
  'occurred_at',
  'severity',
  'component',
  'operation',
  'message',
  'correlation',
  'request_id',
];
const TECHNICAL_VISIBLE_COLUMNS = [
  'occurred_at',
  'severity',
  'component',
  'operation',
  'message',
  'correlation',
  'request_id',
  'fields',
];

const loading = ref(false);
const deleting = ref(false);
const listError = ref('');
const rows = ref<AppLogItem[]>([]);
const total = ref(0);
const detailVisible = ref(false);
const detailRecord = ref<AppLogItem | null>(null);
const detailInitialTab = ref<'fields' | 'raw'>('fields');
const applyingRoute = ref(false);
const activePreset = ref<AppLogPresetKey>('all');
const columnDrawerVisible = ref(false);
const visibleColumnKeys = ref([...DEFAULT_VISIBLE_COLUMNS]);
const selectedRowKeys = ref<Array<string | number>>([]);
const pagination = ref({
  current: 1,
  pageSize: 20,
});
const filters = ref<AppLogFilterState>(createDefaultFilters());
const permissionCodes = APP_LOG_PERMISSION_CODE;

const presetViews = computed(() => [
  { key: 'all' as const, title: t('appLog.presets.all') },
  { key: 'errors' as const, title: t('appLog.presets.errors') },
  { key: 'warnings' as const, title: t('appLog.presets.warnings') },
  { key: 'lastHour' as const, title: t('appLog.presets.lastHour') },
]);
const sortOptions = computed(() => [
  { label: t('appLog.filters.sortOccurredAt'), value: 'occurred_at' as const },
  { label: t('appLog.filters.sortSeverity'), value: 'severity' as const },
  { label: t('appLog.filters.sortComponent'), value: 'component' as const },
]);
const columnSettingOptions = computed(() => [
  { label: t('appLog.columns.occurredAt'), value: 'occurred_at' },
  { label: t('appLog.columns.severity'), value: 'severity' },
  { label: t('appLog.columns.component'), value: 'component' },
  { label: t('appLog.columns.operation'), value: 'operation' },
  { label: t('appLog.columns.message'), value: 'message' },
  { label: t('appLog.columns.correlation'), value: 'correlation' },
  { label: t('appLog.columns.requestId'), value: 'request_id' },
  { label: t('appLog.columns.fields'), value: 'fields' },
]);
const columnViewPresets = computed(() => [
  { value: 'default', label: t('appLog.columnViews.default'), keys: DEFAULT_VISIBLE_COLUMNS },
  { value: 'troubleshooting', label: t('appLog.columnViews.troubleshooting'), keys: TROUBLESHOOTING_VISIBLE_COLUMNS },
  { value: 'technical', label: t('appLog.columnViews.technical'), keys: TECHNICAL_VISIBLE_COLUMNS },
]);
const footerSummary = computed(() => t('appLog.page.footerTotal', { count: total.value }));
const reportListLoadError = createLogListErrorReporter<AppLogItem>({
  fallbackMessage: () => t('appLog.page.loadFailed'),
  listError,
  logger,
  logMessage: 'failed to fetch app logs',
  resolveMessage: (cause, fallback) => resolveAppLogErrorMessage(t, cause, fallback),
  rows,
  total,
});
const reportDetailLoadError = createLogDetailErrorReporter({
  fallbackMessage: () => t('appLog.page.loadFailed'),
  resolveMessage: (cause, fallback) => resolveAppLogErrorMessage(t, cause, fallback),
});

function createDefaultFilters(): AppLogFilterState {
  return {
    keyword: '',
    occurredRange: [],
    severity: '',
    component: '',
    operation: '',
    requestId: '',
    message: '',
    error: '',
    sorters: createSingleSorter('occurred_at', 'desc'),
  };
}

function buildQuery(): AppLogQuery {
  const query: AppLogQuery = {
    page: pagination.value.current,
    page_size: pagination.value.pageSize,
  };
  assignEncodedSorters(query, filters.value.sorters, sortOptions.value);

  if (filters.value.keyword) query.keyword = filters.value.keyword;
  if (filters.value.severity) query.severity = filters.value.severity;
  if (filters.value.component) query.component = filters.value.component;
  if (filters.value.operation) query.operation = filters.value.operation;
  if (filters.value.requestId) query.request_id = filters.value.requestId;
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
  selectedRowKeys.value = selectedRowKeys.value.filter((key) => rows.value.some((row) => row.id === Number(key)));
}

function handleListLoadError(error: unknown) {
  reportListLoadError(error);
}

async function openDetail(row: AppLogItem) {
  detailInitialTab.value = 'fields';
  await openLogDetailRow(row, getAppLogDetail, detailRecord, detailVisible, reportDetailLoadError);
}

function handleSelectChange(keys: Array<string | number>) {
  selectedRowKeys.value = keys;
}

function confirmDeleteOne(row: AppLogItem) {
  if (!permissionStore.hasPermission(permissionCodes.DELETE)) {
    return;
  }
  const dialog = DialogPlugin.confirm({
    header: t('appLog.deleteDialog.title'),
    body: t('appLog.deleteDialog.description', { id: row.id }),
    theme: 'danger',
    confirmBtn: t('appLog.deleteDialog.confirm'),
    cancelBtn: t('appLog.deleteDialog.cancel'),
    onConfirm: async () => {
      dialog.setConfirmLoading(true);
      try {
        if (await deleteOne(row)) {
          dialog.hide();
        }
      } finally {
        dialog.setConfirmLoading(false);
      }
    },
  });
}

function confirmBatchDelete() {
  if (!permissionStore.hasPermission(permissionCodes.DELETE) || selectedRowKeys.value.length === 0) {
    return;
  }
  const dialog = DialogPlugin.confirm({
    header: t('appLog.deleteDialog.batchTitle'),
    body: t('appLog.deleteDialog.batchDescription', { count: selectedRowKeys.value.length }),
    theme: 'danger',
    confirmBtn: t('appLog.deleteDialog.confirm'),
    cancelBtn: t('appLog.deleteDialog.cancel'),
    onConfirm: async () => {
      dialog.setConfirmLoading(true);
      try {
        if (await deleteSelected()) {
          dialog.hide();
        }
      } finally {
        dialog.setConfirmLoading(false);
      }
    },
  });
}

async function deleteOne(row: AppLogItem) {
  deleting.value = true;
  try {
    await deleteAppLog(row.id);
    selectedRowKeys.value = selectedRowKeys.value.filter((key) => Number(key) !== row.id);
    MessagePlugin.success(t('appLog.actions.deleteSuccess'));
    await fetchAppLogs();
    return true;
  } catch (error) {
    logger.error('failed to delete app log', error);
    MessagePlugin.error(resolveAppLogErrorMessage(t, error, t('appLog.actions.deleteFail')));
    return false;
  } finally {
    deleting.value = false;
  }
}

async function deleteSelected() {
  const ids = selectedRowKeys.value.map((key) => Number(key)).filter((id) => Number.isInteger(id) && id > 0);
  if (ids.length === 0) {
    return false;
  }

  deleting.value = true;
  try {
    await deleteAppLogs({ ids });
    selectedRowKeys.value = [];
    MessagePlugin.success(t('appLog.actions.batchDeleteSuccess'));
    await fetchAppLogs();
    return true;
  } catch (error) {
    logger.error('failed to batch delete app logs', error);
    MessagePlugin.error(resolveAppLogErrorMessage(t, error, t('appLog.actions.batchDeleteFail')));
    return false;
  } finally {
    deleting.value = false;
  }
}

function resetFilters() {
  filters.value = createDefaultFilters();
  restartQuery();
}

function handleSearch() {
  restartQuery();
}

function applyPreset(preset: AppLogPresetKey) {
  filters.value = {
    ...createDefaultFilters(),
    ...buildPresetFilters(preset),
    sorters: filters.value.sorters,
  };
  restartQuery(preset);
}

function restartQuery(preset?: AppLogPresetKey) {
  restartLogListQuery({ activePreset, pagination, preset, updateRouteQuery });
}

function buildPresetFilters(preset: AppLogPresetKey): Partial<AppLogFilterState> {
  const now = new Date();
  switch (preset) {
    case 'errors':
      return { severity: 'error' };
    case 'warnings':
      return { severity: 'warn' };
    case 'lastHour':
      return { occurredRange: buildRecentHoursLocalRange(now, 1) };
    default:
      return {};
  }
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
    message = '',
    error = '',
    sort = [],
  } = parseAppLogRouteQuery(route.query);
  const parsedSorters = decodeSorters(sort, normalizeSortBy, normalizeSortOrder);

  filters.value = {
    keyword,
    occurredRange: normalizeRouteRangeForPageState([occurredFrom, occurredTo]),
    severity:
      severity === 'debug' || severity === 'info' || severity === 'warn' || severity === 'error' ? severity : '',
    component,
    operation,
    requestId,
    message,
    error,
    sorters: (() => {
      const normalized = normalizeSorters(parsedSorters, sortOptions.value);
      return normalized.length ? normalized : createSingleSorter('occurred_at', 'desc');
    })(),
  };
}

function buildRouteQuery() {
  const normalizedSorters = normalizeSorters(filters.value.sorters, sortOptions.value);
  const occurredRange = normalizePageStateRangeForRoute(filters.value.occurredRange);

  return buildAppLogLocation({
    keyword: filters.value.keyword,
    occurred_from: occurredRange[0],
    occurred_to: occurredRange[1],
    severity: filters.value.severity,
    component: filters.value.component,
    operation: filters.value.operation,
    request_id: filters.value.requestId,
    message: filters.value.message,
    error: filters.value.error,
    sort: encodeSorters(normalizedSorters, sortOptions.value),
  });
}

async function updateRouteQuery() {
  if (applyingRoute.value) {
    return;
  }

  const targetLocation = buildRouteQuery();
  const currentLocation = buildAppLogLocation(route.query);
  if (JSON.stringify(targetLocation.query) === JSON.stringify(currentLocation.query)) {
    await fetchAppLogs();
    return;
  }

  await router.replace(targetLocation);
}

watch(
  () => [
    route.query.keyword,
    route.query.occurred_from,
    route.query.occurred_to,
    route.query.severity,
    route.query.component,
    route.query.operation,
    route.query.request_id,
    route.query.message,
    route.query.error,
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
    void fetchAppLogs();
  },
  { immediate: true },
);

function normalizeSortBy(value: string): AppLogSortBy | '' {
  return value === 'severity' || value === 'component' ? value : value === 'occurred_at' ? 'occurred_at' : '';
}

function normalizeSortOrder(value: string): AppLogSortOrder {
  return value === 'asc' ? 'asc' : 'desc';
}
</script>
<style scoped lang="less">
.app-log-batch-bar {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  width: 100%;
}

.app-log-batch-bar__actions {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
  justify-content: flex-end;
}
</style>
