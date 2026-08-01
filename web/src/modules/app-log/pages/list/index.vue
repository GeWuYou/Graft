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
        :saved-view-controller="appLogSavedViews"
        @apply-preset="applyPreset"
        @reset="resetFilters"
        @search="handleSearch"
      />
    </template>
    <template #table>
      <app-log-table
        v-model:current="pagination.current"
        v-model:page-size="pagination.pageSize"
        :empty-description="emptyDescription"
        :empty-title="emptyTitle"
        :footer-summary="footerSummary"
        :filtered-empty="hasActiveFilters && rows.length === 0"
        :loading="loading"
        :rows="rows"
        :selection-mode="selectionMode"
        :selected-row-keys="selectedRowKeys"
        :total="total"
        :visible-column-keys="visibleColumnKeys"
        @delete="confirmDeleteOne"
        @detail="openDetail"
        @clear-filters="resetFilters"
        @page-change="fetchAppLogs"
        @select-change="handleSelectChange"
        @enter-selection="selectionMode = true"
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
        <template v-if="selectedRowKeys.length > 0 || selectionMode" #batch>
          <management-batch-bar
            :selected-label="t('appLog.batch.selected', { count: selectedRowKeys.length })"
            :clear-label="t('appLog.batch.cancelSelection')"
            clear-test-id="app-log-batch-clear"
            @clear="clearSelectionMode"
          >
            <t-button v-if="selectionMode" size="small" theme="default" variant="outline" @click="selectCurrentPage">
              {{ t('appLog.batch.selectAll') }}
            </t-button>
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
          </management-batch-bar>
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
import { useQuery } from '@tanstack/vue-query';
import { DialogPlugin, MessagePlugin } from 'tdesign-vue-next';
import { computed, nextTick, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { ManagementBatchBar, TableViewToolbar } from '@/shared/components/management';
import {
  AdvancedQueryColumnDrawer,
  AdvancedQueryListPage,
  applySavedQueryViewPresentation,
  normalizeSavedQueryView,
  useSavedQueryViews,
} from '@/shared/components/query-list';
import { resolveLocalizedErrorMessage as resolveAppLogErrorMessage } from '@/shared/localized-api-error';
import {
  assignEncodedSorters,
  buildRecentHoursLocalRange,
  createLogDetailErrorReporter,
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
import { queryClient } from '@/shared/query';
import { usePermissionStore } from '@/store';
import { createLogger as createModuleLogger } from '@/utils/logger';

import {
  deleteAppLog,
  deleteAppLogs,
  deleteAppLogSavedView,
  getAppLogDetail,
  getAppLogs,
  getAppLogSavedViews,
  postAppLogSavedView,
  putAppLogSavedView,
} from '../../api/app-log';
import AppLogDetailDrawer from '../../components/AppLogDetailDrawer.vue';
import AppLogFilters from '../../components/AppLogFilters.vue';
import AppLogTable from '../../components/AppLogTable.vue';
import { buildAppLogLocation, parseAppLogRouteQuery } from '../../contract/deep-link';
import { APP_LOG_PERMISSION_CODE } from '../../contract/permissions';
import type {
  AppLogFilterState,
  AppLogItem,
  AppLogQuery,
  AppLogSavedViewRequest,
  AppLogSortBy,
  AppLogSortOrder,
} from '../../types/app-log';

defineOptions({
  name: 'AppLogListIndex',
});

const { t } = useI18n();
const logger = createModuleLogger('app-log.list');
const route = useRoute();
const router = useRouter();
const permissionStore = usePermissionStore();

type AppLogPresetKey = 'all' | 'errors' | 'warnings' | 'lastHour';
type AppLogSavedQueryState = Omit<AppLogQuery, 'page' | 'page_size'>;
type AppLogSavedQueryViewState = {
  pageSize: number;
  queryState: AppLogSavedQueryState;
  visibleColumns: string[];
};
const DEFAULT_VISIBLE_COLUMNS = ['occurred_at', 'severity', 'category', 'component', 'operation', 'message'];
const TROUBLESHOOTING_VISIBLE_COLUMNS = [
  'occurred_at',
  'severity',
  'category',
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

const deleting = ref(false);
const listError = ref('');
const rows = ref<AppLogItem[]>([]);
const total = ref(0);
const hasLoadedList = ref(false);
const detailVisible = ref(false);
const detailRecord = ref<AppLogItem | null>(null);
const detailInitialTab = ref<'fields' | 'raw'>('fields');
const applyingRoute = ref(false);
const activePreset = ref<AppLogPresetKey>('all');
const columnDrawerVisible = ref(false);
const visibleColumnKeys = ref([...DEFAULT_VISIBLE_COLUMNS]);
const selectedRowKeys = ref<Array<string | number>>([]);
const selectionMode = ref(false);
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
  { label: t('appLog.columns.category'), value: 'category' },
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
const hasActiveFilters = computed(() =>
  Boolean(
    filters.value.keyword ||
    filters.value.occurredRange.length ||
    filters.value.severity ||
    filters.value.category ||
    filters.value.component ||
    filters.value.operation ||
    filters.value.requestId ||
    filters.value.message ||
    filters.value.error,
  ),
);
const emptyTitle = computed(() =>
  hasActiveFilters.value ? t('appLog.page.emptyFilteredTitle') : t('appLog.page.emptyTitle'),
);
const emptyDescription = computed(() =>
  hasActiveFilters.value ? t('appLog.page.emptyFilteredDescription') : t('appLog.page.emptyDescription'),
);
const reportDetailLoadError = createLogDetailErrorReporter({
  fallbackMessage: () => t('appLog.page.loadFailed'),
  resolveMessage: (cause, fallback) => resolveAppLogErrorMessage(t, cause, fallback),
});
const appLogSavedViews = useSavedQueryViews<AppLogSavedQueryViewState, number>({
  adapter: {
    list: async () =>
      (
        await queryClient.fetchQuery({
          queryKey: ['app-log', 'saved-views'],
          queryFn: getAppLogSavedViews,
        })
      ).map((view) => normalizeSavedQueryView<AppLogSavedQueryState, number>(view)),
    create: async (input) => {
      const view = await postAppLogSavedView(toAppLogSavedViewRequest(input));
      await queryClient.invalidateQueries({ queryKey: ['app-log', 'saved-views'] });
      return normalizeSavedQueryView<AppLogSavedQueryState, number>(view);
    },
    update: async (id, input) => {
      const view = await putAppLogSavedView(id, toAppLogSavedViewRequest(input));
      await queryClient.invalidateQueries({ queryKey: ['app-log', 'saved-views'] });
      return normalizeSavedQueryView<AppLogSavedQueryState, number>(view);
    },
    remove: async (id) => {
      await deleteAppLogSavedView(id);
      await queryClient.invalidateQueries({ queryKey: ['app-log', 'saved-views'] });
    },
  },
  applyView: async (view) => {
    applyAppLogSavedQueryView(view.state);
    await updateRouteQuery();
  },
  onError: (error) => {
    logger.error('failed to manage app-log saved view', error);
    MessagePlugin.error(resolveAppLogErrorMessage(t, error, t('appLog.page.loadFailed')));
  },
  serializeCurrentState: () => ({
    pageSize: pagination.value.pageSize,
    queryState: currentAppLogSavedViewQueryState(),
    visibleColumns: [...visibleColumnKeys.value],
  }),
});

function createDefaultFilters(): AppLogFilterState {
  return {
    keyword: '',
    occurredRange: [],
    severity: '',
    category: '',
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
  if (filters.value.category) query.category = filters.value.category;
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

const appLogListQuery = useQuery(
  {
    queryKey: computed(() => ['app-log', 'list', buildQuery()]),
    queryFn: () => getAppLogs(buildQuery()),
  },
  queryClient,
);
const loading = computed(() => appLogListQuery.isFetching.value);

watch(appLogListQuery.data, (response) => {
  if (!response) return;
  listError.value = '';
  applyListResponse(response);
});

watch(appLogListQuery.error, (error) => {
  if (!error) return;
  logger.error('failed to fetch app logs', error);
  listError.value = resolveAppLogErrorMessage(t, error, t('appLog.page.loadFailed'));
  // 后台刷新失败时，vue-query 仍保留最近一次成功响应，避免列表因瞬时错误闪退为空。
  if (!hasLoadedList.value) {
    rows.value = [];
    total.value = 0;
  }
  MessagePlugin.error(listError.value);
});

async function fetchAppLogs() {
  listError.value = '';
  selectionMode.value = false;
  await nextTick();
  await appLogListQuery.refetch();
}

function applyListResponse(response: Awaited<ReturnType<typeof getAppLogs>>) {
  hasLoadedList.value = true;
  rows.value = response.items;
  total.value = response.total;
  selectedRowKeys.value = selectedRowKeys.value.filter((key) => rows.value.some((row) => row.id === Number(key)));
  selectionMode.value = false;
}

async function openDetail(row: AppLogItem) {
  detailInitialTab.value = 'fields';
  await openLogDetailRow(
    row,
    (id) => queryClient.fetchQuery({ queryKey: ['app-log', 'detail', id], queryFn: () => getAppLogDetail(id) }),
    detailRecord,
    detailVisible,
    reportDetailLoadError,
  );
}

function handleSelectChange(keys: Array<string | number>) {
  selectedRowKeys.value = keys;
}

function clearSelectionMode() {
  selectedRowKeys.value = [];
  selectionMode.value = false;
}

function selectCurrentPage() {
  selectedRowKeys.value = rows.value.map((row) => row.id);
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
    await queryClient.invalidateQueries({ queryKey: ['app-log', 'list'] });
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
    await queryClient.invalidateQueries({ queryKey: ['app-log', 'list'] });
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
  appLogSavedViews.selectedId.value = undefined;
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
    quick_preset: quickPreset = '',
    keyword = '',
    occurred_from: occurredFrom = '',
    occurred_to: occurredTo = '',
    severity = '',
    category = '',
    component = '',
    operation = '',
    request_id: requestId = '',
    message = '',
    error = '',
    sort = [],
  } = parseAppLogRouteQuery(route.query);
  const parsedSorters = decodeSorters(sort, normalizeSortBy, normalizeSortOrder);
  activePreset.value = normalizeQuickPreset(quickPreset);

  filters.value = {
    keyword,
    occurredRange: normalizeRouteRangeForPageState([occurredFrom, occurredTo]),
    severity:
      severity === 'debug' || severity === 'info' || severity === 'warn' || severity === 'error' ? severity : '',
    category,
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
    quick_preset: activePreset.value === 'all' ? '' : activePreset.value,
    keyword: filters.value.keyword,
    occurred_from: occurredRange[0],
    occurred_to: occurredRange[1],
    severity: filters.value.severity,
    category: filters.value.category,
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
    route.query.quick_preset,
    route.query.keyword,
    route.query.occurred_from,
    route.query.occurred_to,
    route.query.severity,
    route.query.category,
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

function normalizeQuickPreset(value: string): AppLogPresetKey {
  return value === 'errors' || value === 'warnings' || value === 'lastHour' ? value : 'all';
}

function normalizeSortOrder(value: string): AppLogSortOrder {
  return value === 'asc' ? 'asc' : 'desc';
}

function currentAppLogSavedViewQueryState(): AppLogSavedQueryState {
  const { page: _page, page_size: _pageSize, ...query } = buildQuery();
  return query;
}

function toAppLogSavedViewRequest(input: {
  name: string;
  isDefault: boolean;
  state: AppLogSavedQueryViewState;
}): AppLogSavedViewRequest {
  return {
    name: input.name,
    page_size: input.state.pageSize,
    query_state: input.state.queryState as unknown as Record<string, unknown>,
    visible_columns: input.state.visibleColumns,
    is_default: input.isDefault,
  };
}

function applyAppLogSavedQueryView(savedState: AppLogSavedQueryViewState) {
  const state = savedState.queryState;
  const normalizedSorters = normalizeSorters(
    decodeSorters(state.sort ?? [], normalizeSortBy, normalizeSortOrder),
    sortOptions.value,
  );
  filters.value = {
    ...createDefaultFilters(),
    keyword: state.keyword ?? '',
    occurredRange: normalizeRouteRangeForPageState([state.occurred_from ?? '', state.occurred_to ?? '']),
    severity:
      state.severity === 'debug' || state.severity === 'info' || state.severity === 'warn' || state.severity === 'error'
        ? state.severity
        : '',
    category: state.category ?? '',
    component: state.component ?? '',
    operation: state.operation ?? '',
    requestId: state.request_id ?? '',
    message: state.message ?? '',
    error: state.error ?? '',
    sorters: normalizedSorters.length ? normalizedSorters : createSingleSorter('occurred_at', 'desc'),
  };
  activePreset.value = 'all';
  applySavedQueryViewPresentation(savedState, {
    pagination: pagination.value,
    supportedColumns: columnSettingOptions.value.map((column) => column.value),
    visibleColumnKeys,
  });
}

onMounted(() => {
  void appLogSavedViews.load();
});
</script>
