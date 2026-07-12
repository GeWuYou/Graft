<template>
  <div class="container-page" data-page-type="list-form-detail">
    <management-page-header
      title-key="container.list.title"
      :title="t('container.list.title')"
      description-key="container.list.description"
      :description="t('container.list.description')"
      :source="{ labelKey: 'container.list.eyebrow', fallback: t('container.list.eyebrow') }"
    >
      <template #meta>
        <t-space break-line size="small">
          <t-tag :theme="runtimeStatusTheme" variant="light-outline">
            {{ t('container.list.runtimeLabel') }}: {{ runtimeSummary }}
          </t-tag>
          <t-tag theme="default" variant="light-outline">
            {{ t('container.list.totalCount', { count: totalCount }) }}
          </t-tag>
          <t-tag theme="success" variant="light-outline">
            {{ t('container.list.runningCount', { count: runningCount }) }}
          </t-tag>
          <t-tag theme="warning" variant="light-outline">
            {{ t('container.list.stoppedCount', { count: stoppedCount }) }}
          </t-tag>
          <t-tag theme="danger" variant="light-outline">
            {{ t('container.list.errorCount', { count: errorCount }) }}
          </t-tag>
          <t-tag theme="danger" variant="light-outline">
            {{ t('container.list.unhealthyCount', { count: unhealthyCount }) }}
          </t-tag>
          <t-tag :theme="readOnlyMode ? 'warning' : 'default'" variant="light-outline">
            {{ readOnlyModeStatus }}
          </t-tag>
        </t-space>
      </template>
    </management-page-header>

    <management-toolbar class="container-toolbar">
      <template #filters>
        <t-input
          v-model="filters.keyword"
          class="management-list-search"
          clearable
          data-testid="container-filter-keyword"
          :placeholder="t('container.list.filters.searchPlaceholder')"
          @enter="applyFilters"
        >
          <template #prefix-icon><search-icon /></template>
        </t-input>
        <t-select
          v-model="filters.status"
          class="management-toolbar__select"
          data-testid="container-filter-status"
          :placeholder="t('container.list.filters.status')"
        >
          <t-option value="all" :label="t('container.list.filters.allStatuses')" />
          <t-option v-for="status in statusOptions" :key="status" :value="status" :label="stateLabel(status)" />
        </t-select>
        <t-select
          v-model="filters.deploymentType"
          class="management-toolbar__select"
          data-testid="container-filter-deployment-type"
          :placeholder="t('container.list.filters.deploymentType')"
        >
          <t-option value="all" :label="t('container.list.filters.allDeploymentTypes')" />
          <t-option
            v-for="deploymentType in deploymentTypeOptions"
            :key="deploymentType"
            :value="deploymentType"
            :label="deploymentTypeLabel(deploymentType)"
          />
        </t-select>
        <t-select
          v-model="filters.runtimeTargetId"
          class="management-toolbar__select"
          data-testid="container-filter-runtime-target"
          :placeholder="t('container.list.filters.runtimeTarget')"
        >
          <t-option value="all" :label="t('container.list.filters.allRuntimeTargets')" />
          <t-option v-for="target in runtimeTargets" :key="target.id" :value="target.id" :label="target.displayName" />
        </t-select>
        <t-select
          v-model="filters.health"
          class="management-toolbar__select"
          data-testid="container-filter-health"
          :placeholder="t('container.list.filters.health')"
        >
          <t-option value="all" :label="t('container.list.filters.allHealth')" />
          <t-option v-for="health in healthOptions" :key="health" :value="health" :label="healthLabel(health)" />
        </t-select>
        <t-button data-testid="container-filter-apply" theme="primary" @click="applyFilters">
          {{ t('container.list.filters.query') }}
        </t-button>
        <t-button data-testid="container-filter-reset" theme="default" variant="text" @click="resetFilters">
          {{ t('container.list.filters.reset') }}
        </t-button>
      </template>
    </management-toolbar>

    <container-resource-table
      v-model:current="pagination.current"
      v-model:page-size="pagination.pageSize"
      :always-visible-column-keys="CONTAINER_RESOURCE_ALWAYS_VISIBLE_COLUMNS"
      :empty-description="
        hasActiveFilters ? t('container.list.emptyFilteredDescription') : t('container.list.emptyDescription')
      "
      :empty-title="t('container.list.emptyTitle')"
      :footer-summary="footerSummary"
      :head-description="t('container.list.tableHint')"
      :head-summary="t('container.list.tableSummary', { count: listTotal })"
      :loading="tableLoading"
      :more-actions-label="t('container.list.actions.more')"
      :row-actions="buildRowActions"
      :rows="rows"
      :selected-row-keys="selectedRowKeys"
      :table-density="tableDensity"
      :total="listTotal"
      :visible-column-keys="visibleColumnKeys"
      @action="handleTableAction"
      @page-change="handlePageChange"
      @project-context="openComposeProjectContext"
      @select-change="handleSelectChange"
    >
      <template #toolbar>
        <div class="container-toolbar-row">
          <table-view-toolbar
            :column-settings-label="t('container.list.columnSettings')"
            :density-label="tableDensityLabel"
            :refresh-label="t('container.list.refresh')"
            :refresh-loading="refreshing"
            @column-settings="columnDrawerVisible = true"
            @density="toggleTableDensity"
            @refresh="handleManualRefresh"
          />
        </div>
      </template>
      <template #batch>
        <div v-if="selectedRowKeys.length > 0" class="container-batch-bar">
          <span>{{ t('container.list.batch.selected', { count: selectedRowKeys.length }) }}</span>
          <div class="container-batch-bar__actions">
            <t-tooltip :content="batchActionHint('start')" placement="top">
              <t-button
                data-testid="container-batch-start"
                size="small"
                theme="primary"
                variant="outline"
                :disabled="isBatchActionDisabled('start')"
                :loading="batchActionLoading === 'start'"
                @click="confirmBatchAction('start')"
              >
                {{ t('container.list.batch.start') }}
              </t-button>
            </t-tooltip>
            <t-tooltip :content="batchActionHint('stop')" placement="top">
              <t-button
                data-testid="container-batch-stop"
                size="small"
                theme="warning"
                variant="outline"
                :disabled="isBatchActionDisabled('stop')"
                :loading="batchActionLoading === 'stop'"
                @click="confirmBatchAction('stop')"
              >
                {{ t('container.list.batch.stop') }}
              </t-button>
            </t-tooltip>
            <t-tooltip :content="batchActionHint('restart')" placement="top">
              <t-button
                data-testid="container-batch-restart"
                size="small"
                theme="warning"
                variant="outline"
                :disabled="isBatchActionDisabled('restart')"
                :loading="batchActionLoading === 'restart'"
                @click="confirmBatchAction('restart')"
              >
                {{ t('container.list.batch.restart') }}
              </t-button>
            </t-tooltip>
            <t-tooltip :content="batchActionHint('remove')" placement="top">
              <t-button
                data-testid="container-batch-remove"
                size="small"
                theme="danger"
                variant="outline"
                :disabled="isBatchActionDisabled('remove')"
                :loading="batchActionLoading === 'remove'"
                @click="confirmBatchAction('remove')"
              >
                {{ t('container.list.batch.remove') }}
              </t-button>
            </t-tooltip>
            <t-button
              data-testid="container-batch-clear"
              size="small"
              theme="default"
              variant="text"
              @click="clearSelection"
            >
              {{ t('container.list.batch.cancelSelection') }}
            </t-button>
          </div>
        </div>
      </template>
      <template #feedback>
        <t-alert v-if="listError.title" class="container-alert" theme="error" :title="listError.title">
          <p v-if="listError.hint" class="container-alert__hint">{{ listError.hint }}</p>
          <template #operation>
            <t-button theme="danger" variant="text" @click="refreshContainers">
              {{ t('container.list.retry') }}
            </t-button>
          </template>
        </t-alert>
      </template>

      <template v-if="hasActiveFilters" #empty-action>
        <t-button theme="primary" variant="outline" @click="resetFilters">
          {{ t('container.list.clearFilters') }}
        </t-button>
      </template>
    </container-resource-table>

    <advanced-query-column-drawer
      v-model:visible="columnDrawerVisible"
      v-model:selected-keys="visibleColumnKeys"
      :columns="columnSettingOptions"
      :default-selected-keys="DEFAULT_VISIBLE_COLUMNS"
      :disabled-keys="ALWAYS_VISIBLE_COLUMNS"
      :reset-label="t('container.list.resetColumns')"
      :title="t('container.list.columnSettings')"
    />
  </div>
</template>
<script setup lang="ts">
import { SearchIcon } from 'tdesign-icons-vue-next';
import type { DialogInstance } from 'tdesign-vue-next';
import { DialogPlugin } from 'tdesign-vue-next/es/dialog';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { NotifyPlugin } from 'tdesign-vue-next/es/notification';
import { computed, h, onActivated, onDeactivated, onMounted, onUnmounted, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { LOCALE, type LocalizedTitle } from '@/contracts/i18n/locales';
import { buildAuditResourceLocation } from '@/modules/audit/contract/deep-link';
import { AUDIT_PERMISSION_CODE } from '@/modules/audit/contract/permissions';
import { PROJECT_BOOTSTRAP_ROUTE } from '@/modules/project/contract/bootstrap';
import { listRuntimeTargets, type RuntimeTarget } from '@/modules/runtime-target/api/runtime-target';
import { ManagementPageHeader, ManagementToolbar, TableViewToolbar } from '@/shared/components/management';
import { AdvancedQueryColumnDrawer } from '@/shared/components/query-list';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { usePermissionStore, useTabsRouterStore } from '@/store';
import { createLogger } from '@/utils/logger';
import { localizeRouteTitleKey } from '@/utils/route/title';
import type { AppRouteMeta } from '@/utils/types';

import {
  batchContainerActions,
  getContainers,
  removeContainer,
  restartContainer,
  startContainer,
  stopContainer,
} from '../../api/container';
import ContainerResourceTable from '../../components/ContainerResourceTable.vue';
import { CONTAINER_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import {
  buildContainerResourceColumnSettingOptions,
  CONTAINER_RESOURCE_ALL_COLUMN_KEYS,
  CONTAINER_RESOURCE_ALWAYS_VISIBLE_COLUMNS,
  CONTAINER_RESOURCE_COLUMN_STORAGE_KEY,
  CONTAINER_RESOURCE_DEFAULT_VISIBLE_COLUMNS,
  type ContainerResourceRowAction,
  displayContainerName,
} from '../../shared/resource-table';
import {
  acquireContainerSummaryCollectionSubscription,
  clearContainerListMetadata,
  releaseContainerSummaryCollectionSubscription,
  seedContainerList,
  selectContainerListViews,
} from '../../shared/stats-manager';
import type {
  ContainerAction,
  ContainerActionLevel,
  ContainerBatchActionItem,
  ContainerBatchActionResponse,
  ContainerFilters,
  ContainerListQueryWithOrchestrator,
  ContainerListSummary,
  ContainerRuntimeInfo,
  ContainerState,
  ContainerSummaryRecord,
} from '../../types/container';

defineOptions({
  name: 'ContainerListIndex',
});

const { t } = useI18n();
const router = useRouter();
const permissionStore = usePermissionStore();
const tabsRouterStore = useTabsRouterStore();
const logger = createLogger('container.list');
const auditPermissionCodes = AUDIT_PERMISSION_CODE;
const DEFAULT_VISIBLE_COLUMNS = CONTAINER_RESOURCE_DEFAULT_VISIBLE_COLUMNS;
const ALWAYS_VISIBLE_COLUMNS = CONTAINER_RESOURCE_ALWAYS_VISIBLE_COLUMNS;

const statusOptions: ContainerState[] = [
  'created',
  'running',
  'paused',
  'restarting',
  'removing',
  'exited',
  'dead',
  'unknown',
];
const healthOptions = ['healthy', 'unhealthy', 'starting', 'none', 'unavailable'] as const;
const deploymentTypeOptions = ['standalone', 'compose'] as const;
const CONTAINER_RUNTIME_DISABLED_MESSAGE_KEY = 'ops.container.error.runtimeDisabled';
const CONTAINER_DEFAULT_PAGE_SIZE = 20;
type ListErrorState = {
  title: string;
  hint: string;
};
type DangerousContainerAction = Extract<ContainerAction, 'remove' | 'restart' | 'start' | 'stop'>;

const tableLoading = ref(false);
const refreshing = ref(false);
const listError = ref<ListErrorState>({ title: '', hint: '' });
const runtime = ref<ContainerRuntimeInfo | null>(null);
const listSummary = ref<ContainerListSummary | null>(null);
const listTotal = ref(0);
const runtimeTargets = ref<RuntimeTarget[]>([]);
const columnDrawerVisible = ref(false);
const visibleColumnKeys = ref<string[]>(loadVisibleColumnKeys());
const tableDensity = ref<'medium' | 'small'>('medium');
const selectedRowKeys = ref<Array<string | number>>([]);
const selectedRowRecords = ref<ContainerSummaryRecord[]>([]);
const batchActionLoading = ref<DangerousContainerAction | ''>('');
const activeDangerousDialog = ref<DialogInstance | null>(null);
const dangerousDialogOpen = ref(false);
const filters = reactive<ContainerFilters>({
  keyword: '',
  deploymentType: 'all',
  runtimeTargetId: 'all',
  status: 'all',
  health: 'all',
});
const submittedFilters = ref<ContainerFilters>({
  keyword: '',
  deploymentType: 'all',
  runtimeTargetId: 'all',
  status: 'all',
  health: 'all',
});
const pagination = reactive({
  current: 1,
  pageSize: CONTAINER_DEFAULT_PAGE_SIZE,
});
const rows = computed<ContainerSummaryRecord[]>(() => selectContainerListViews());
const listRealtimeActive = ref(false);
let listRealtimeSubscribed = false;
function hasCommittedFilters(activeFilters: ContainerFilters) {
  return (
    Boolean(activeFilters.keyword.trim()) ||
    activeFilters.deploymentType !== 'all' ||
    activeFilters.runtimeTargetId !== 'all' ||
    activeFilters.status !== 'all' ||
    activeFilters.health !== 'all'
  );
}

const hasActiveFilters = computed(() => hasCommittedFilters(submittedFilters.value));
const totalCount = computed(() => listSummary.value?.total ?? listTotal.value);
const runningCount = computed(() => listSummary.value?.running ?? 0);
const stoppedCount = computed(() => listSummary.value?.stopped ?? 0);
const errorCount = computed(() => listSummary.value?.error ?? 0);
const unhealthyCount = computed(() => listSummary.value?.unhealthy ?? 0);
const readOnlyMode = computed(() => {
  if (!rows.value.length) {
    return true;
  }

  return rows.value.every((row) => !canRunAnyDangerousAction(row));
});
const readOnlyModeStatus = computed(() =>
  readOnlyMode.value ? t('container.list.readOnlyMode') : t('container.list.actionModeEnabled'),
);
const runtimeStatusTheme = computed(() => {
  if (runtime.value?.status === 'enabled') return 'success';
  if (runtime.value?.status === 'disabled') return 'warning';
  return 'danger';
});
const runtimeSummary = computed(() => {
  if (!runtime.value) return t('container.list.runtimeUnavailable');
  const version = runtime.value.server_version || runtime.value.api_version || '';
  return version ? `${runtime.value.runtime} / ${version}` : runtime.value.runtime;
});
const tableDensityLabel = computed(() =>
  tableDensity.value === 'medium' ? t('container.list.compactDensity') : t('container.list.defaultDensity'),
);
const columnSettingOptions = computed(() => buildContainerResourceColumnSettingOptions(t));
const footerSummary = computed(() => {
  if (!listTotal.value) {
    return t('container.list.pagination.empty');
  }

  const start = (pagination.current - 1) * pagination.pageSize + 1;
  const end = Math.min(pagination.current * pagination.pageSize, listTotal.value);
  return t('container.list.pagination.summary', {
    end,
    start,
    total: listTotal.value,
  });
});

function buildSelectedRowMap() {
  const selectedRowMap = new Map(selectedRowRecords.value.map((row) => [row.id, row]));
  rows.value.forEach((row) => {
    selectedRowMap.set(row.id, row);
  });
  return selectedRowMap;
}

const selectedRows = computed(() => {
  const rowMap = buildSelectedRowMap();

  return selectedRowKeys.value
    .map((key) => {
      const id = String(key);
      return rowMap.get(id) ?? null;
    })
    .filter((row): row is ContainerSummaryRecord => Boolean(row));
});
let refreshRequestSeq = 0;

onMounted(() => {
  listRealtimeActive.value = true;
  void loadRuntimeTargets();
  void refreshContainers();
});

async function loadRuntimeTargets() {
  try {
    runtimeTargets.value = (await listRuntimeTargets()).filter((target) => target.provider === 'docker');
  } catch {
    runtimeTargets.value = [];
  }
}

onUnmounted(() => {
  listRealtimeActive.value = false;
  releaseListRealtimeSubscription();
});

onActivated(() => {
  listRealtimeActive.value = true;
  acquireListRealtimeSubscription();
});

onDeactivated(() => {
  listRealtimeActive.value = false;
  releaseListRealtimeSubscription();
});

watch(
  visibleColumnKeys,
  (keys) => {
    const normalizedKeys = normalizeVisibleColumnKeys(keys);
    if (normalizedKeys.join('|') !== keys.join('|')) {
      visibleColumnKeys.value = normalizedKeys;
      return;
    }
    persistVisibleColumnKeys(normalizedKeys);
  },
  { deep: true },
);

watch(
  () => [pagination.current, pagination.pageSize],
  () => void refreshContainers(),
);

async function refreshContainers() {
  const requestSeq = ++refreshRequestSeq;
  const shouldBlockTable = rows.value.length === 0 && !tableLoading.value;
  if (shouldBlockTable) {
    tableLoading.value = true;
  } else {
    refreshing.value = true;
  }
  listError.value = { title: '', hint: '' };
  try {
    const payload = await getContainers(buildListQuery());
    if (requestSeq !== refreshRequestSeq) {
      return;
    }
    seedContainerList(payload.items);
    if (payload.items.length > 0) {
      acquireListRealtimeSubscription();
    } else {
      releaseListRealtimeSubscription();
    }
    runtime.value = payload.runtime;
    listSummary.value = payload.summary;
    listTotal.value = payload.total;
    syncSelectedRowsFromCurrentPage();
  } catch (error) {
    if (requestSeq !== refreshRequestSeq) {
      return;
    }
    releaseListRealtimeSubscription();
    clearContainerListMetadata();
    runtime.value = null;
    listSummary.value = null;
    listTotal.value = 0;
    listError.value = resolveListError(error);
    logger.error('failed to fetch containers', error);
  } finally {
    if (requestSeq === refreshRequestSeq) {
      tableLoading.value = false;
      refreshing.value = false;
    }
  }
}

async function handleManualRefresh() {
  if (tableLoading.value || refreshing.value) {
    return;
  }
  await refreshContainers();
}

function syncSelectedRowsFromCurrentPage() {
  if (!selectedRowKeys.value.length) {
    selectedRowRecords.value = [];
    return;
  }

  const rowMap = buildSelectedRowMap();
  selectedRowRecords.value = selectedRowKeys.value
    .map((key) => rowMap.get(String(key)) ?? null)
    .filter((row): row is ContainerSummaryRecord => Boolean(row));
}

function acquireListRealtimeSubscription() {
  if (!listRealtimeActive.value || rows.value.length === 0 || listRealtimeSubscribed) {
    return;
  }
  listRealtimeSubscribed = true;
  acquireContainerSummaryCollectionSubscription();
}

function releaseListRealtimeSubscription() {
  if (!listRealtimeSubscribed) {
    return;
  }
  listRealtimeSubscribed = false;
  releaseContainerSummaryCollectionSubscription();
}

function resolveListError(error: unknown): ListErrorState {
  if (isApiRequestErrorShape(error) && error.messageKey === CONTAINER_RUNTIME_DISABLED_MESSAGE_KEY) {
    return {
      title: t(CONTAINER_RUNTIME_DISABLED_MESSAGE_KEY),
      hint: t('container.list.runtimeDisabledHint'),
    };
  }

  return {
    title: resolveLocalizedErrorMessage(t, error, t('container.list.loadFailed')),
    hint: '',
  };
}

function isApiRequestErrorShape(error: unknown): error is { isApiRequestError: true; messageKey?: string } {
  return Boolean(error && typeof error === 'object' && (error as { isApiRequestError?: unknown }).isApiRequestError);
}

function applyFilters() {
  filters.keyword = filters.keyword.trim();
  commitSubmittedFilters();
  clearSelection();
  requestFirstPage();
}

function resetFilters() {
  filters.keyword = '';
  filters.deploymentType = 'all';
  filters.runtimeTargetId = 'all';
  filters.status = 'all';
  filters.health = 'all';
  commitSubmittedFilters();
  clearSelection();
  requestFirstPage();
}

function requestFirstPage() {
  if (pagination.current === 1) {
    void refreshContainers();
    return;
  }
  pagination.current = 1;
}

function buildListQuery(): ContainerListQueryWithOrchestrator {
  const activeFilters = submittedFilters.value;
  return {
    limit: pagination.pageSize,
    offset: (pagination.current - 1) * pagination.pageSize,
    keyword: activeFilters.keyword.trim() || undefined,
    deployment_type: activeFilters.deploymentType === 'all' ? undefined : activeFilters.deploymentType,
    runtime_target_id: activeFilters.runtimeTargetId === 'all' ? undefined : Number(activeFilters.runtimeTargetId),
    state: activeFilters.status === 'all' ? undefined : activeFilters.status,
    health: activeFilters.health === 'all' ? undefined : activeFilters.health,
  };
}

function commitSubmittedFilters() {
  submittedFilters.value = {
    keyword: filters.keyword,
    deploymentType: filters.deploymentType,
    runtimeTargetId: filters.runtimeTargetId,
    status: filters.status,
    health: filters.health,
  };
}

const deploymentTypeLabel = (type: (typeof deploymentTypeOptions)[number]) => t(`container.list.deployments.${type}`);

function openDetail(row: ContainerSummaryRecord) {
  void navigateToDetail(row, 'overview');
}

function openAuditLogs(row: ContainerSummaryRecord) {
  void router.push(buildAuditResourceLocation('container', row.id, displayContainerName(row)));
}

function openComposeProjectContext(projectName: string) {
  const keyword = projectName.trim();
  if (keyword) {
    void router.push({ name: PROJECT_BOOTSTRAP_ROUTE.LIST.routeName, query: { keyword } });
  }
}

async function copyContainerId(row: ContainerSummaryRecord) {
  try {
    await navigator.clipboard.writeText(row.id);
    MessagePlugin.success(t('container.list.copyIdSuccess'));
  } catch (error) {
    logger.warn('failed to copy container id', error);
    MessagePlugin.error(t('container.list.copyIdError'));
  }
}

function buildRowActions(row: ContainerSummaryRecord): ContainerResourceRowAction[] {
  const actions: ContainerResourceRowAction[] = [
    {
      fallbackLabel: t('container.list.actions.detail'),
      label: 'container.list.actions.detail',
      testId: 'container-action-detail',
      value: 'detail',
    },
    {
      fallbackLabel: t('container.list.actions.logs'),
      label: 'container.list.actions.logs',
      testId: 'container-action-logs',
      value: 'logs',
    },
  ];

  if (permissionStore.hasPermission(auditPermissionCodes.READ)) {
    actions.push({
      fallbackLabel: t('container.list.actions.viewAudit'),
      label: 'container.list.actions.viewAudit',
      testId: 'container-action-audit',
      value: 'view-audit',
    });
  }

  actions.push(
    {
      fallbackLabel: t('container.list.actions.copyId'),
      label: 'container.list.actions.copyId',
      testId: 'container-action-copy-id',
      value: 'copy-id',
    },
    {
      fallbackLabel: t('container.list.actions.inspect'),
      label: 'container.list.actions.inspect',
      testId: 'container-action-inspect',
      value: 'inspect',
    },
    {
      fallbackLabel: t('container.list.actions.viewMounts'),
      label: 'container.list.actions.viewMounts',
      testId: 'container-action-view-mounts',
      value: 'view-mounts',
    },
    {
      fallbackLabel: t('container.list.actions.viewNetworks'),
      label: 'container.list.actions.viewNetworks',
      testId: 'container-action-view-networks',
      value: 'view-networks',
    },
    {
      fallbackLabel: t('container.list.actions.viewEnvironment'),
      label: 'container.list.actions.viewEnvironment',
      testId: 'container-action-view-env',
      value: 'view-env',
    },
  );

  if (canRunDangerousAction(row, 'start')) {
    actions.push({
      fallbackLabel: t('container.list.actions.start'),
      label: 'container.list.actions.start',
      testId: 'container-action-start',
      value: 'start',
    });
  }

  if (canRunDangerousAction(row, 'stop')) {
    actions.push({
      fallbackLabel: t('container.list.actions.stop'),
      label: 'container.list.actions.stop',
      testId: 'container-action-stop',
      value: 'stop',
    });
  }

  if (canRunDangerousAction(row, 'restart')) {
    actions.push({
      fallbackLabel: t('container.list.actions.restart'),
      label: 'container.list.actions.restart',
      testId: 'container-action-restart',
      value: 'restart',
    });
  }

  if (orchestratorActionLevel(row) !== 'readonly') {
    actions.push({
      disabled: isDangerousActionDisabled(row, 'remove'),
      fallbackLabel: t('container.list.actions.remove'),
      label: 'container.list.actions.remove',
      testId: 'container-action-remove',
      value: 'remove',
    });
  }

  return actions;
}

const stateLabel = (state: ContainerState) => t(`container.list.states.${state}`);

const healthLabel = (health: (typeof healthOptions)[number]) => t(`container.list.health.${health || 'unavailable'}`);

function handleTableAction(payload: { action: string; row: ContainerSummaryRecord }) {
  const { action, row } = payload;
  if (action === 'copy-id') {
    void copyContainerId(row);
    return;
  }

  if (action === 'detail') {
    openDetail(row);
    return;
  }

  if (action === 'logs') {
    void navigateToDetail(row, 'logs');
    return;
  }

  if (action === 'view-audit') {
    openAuditLogs(row);
    return;
  }

  if (action === 'inspect') {
    void navigateToDetail(row, 'overview');
    return;
  }

  if (action === 'view-mounts') {
    void navigateToDetail(row, 'storage');
    return;
  }

  if (action === 'view-networks') {
    void navigateToDetail(row, 'network');
    return;
  }

  if (action === 'view-env') {
    void navigateToDetail(row, 'config');
    return;
  }

  if (action === 'start' || action === 'stop' || action === 'restart' || action === 'remove') {
    void performDangerousAction(row, action);
  }
}

function handleRowAction(action: string, row: ContainerSummaryRecord) {
  handleTableAction({ action, row });
}

defineExpose({
  handleRowAction,
});

const performDangerousAction = async (row: ContainerSummaryRecord, action: DangerousContainerAction) => {
  if (isDangerousActionDisabled(row, action)) {
    MessagePlugin.warning(t('container.list.actions.dangerousDisabled'));
    return;
  }

  const force = action === 'remove' ? await confirmRemoveAction(row) : await confirmRuntimeAction(row, action);
  if (force === undefined) return;

  await executeDangerousAction(row, action, force);
};

const confirmRuntimeAction = (row: ContainerSummaryRecord, action: Exclude<DangerousContainerAction, 'remove'>) => {
  if (dangerousDialogOpen.value) {
    return Promise.resolve(undefined);
  }

  return new Promise<boolean | undefined>((resolve) => {
    let resolved = false;
    dangerousDialogOpen.value = true;
    const dialog = DialogPlugin.confirm({
      header: t(actionDialogTitleKey(action)),
      body: () =>
        h('div', { class: 'container-remove-confirm' }, [
          h('p', t(actionConfirmKey(action), { name: displayContainerName(row) })),
          rowActionRiskText(row) ? h('p', { class: 'container-remove-confirm__risk' }, rowActionRiskText(row)) : null,
        ]),
      theme: action === 'start' ? 'warning' : 'danger',
      confirmBtn: t('container.list.actions.confirm'),
      cancelBtn: t('container.list.actions.cancel'),
      onCancel: () =>
        closeConfirmDialog(
          dialog,
          resolve,
          undefined,
          () => resolved,
          (value) => (resolved = value),
        ),
      onClose: () =>
        closeConfirmDialog(
          dialog,
          resolve,
          undefined,
          () => resolved,
          (value) => (resolved = value),
        ),
      onConfirm: () => {
        closeConfirmDialog(
          dialog,
          resolve,
          false,
          () => resolved,
          (value) => (resolved = value),
        );
      },
    });
    activeDangerousDialog.value = dialog;
  });
};

const confirmRemoveAction = (row: ContainerSummaryRecord) => {
  if (dangerousDialogOpen.value) {
    return Promise.resolve(undefined);
  }

  return new Promise<boolean | undefined>((resolve) => {
    let resolved = false;
    dangerousDialogOpen.value = true;
    const force = ref(false);
    const running = row.state === 'running';
    const dialog = DialogPlugin.confirm({
      header: t('container.list.actions.confirmRemoveTitle'),
      body: () =>
        h('div', { class: 'container-remove-confirm' }, [
          h(
            'p',
            running
              ? t('container.list.actions.confirmRemoveRunning', { name: displayContainerName(row) })
              : t('container.list.actions.confirmRemove', { name: displayContainerName(row) }),
          ),
          rowActionRiskText(row) ? h('p', { class: 'container-remove-confirm__risk' }, rowActionRiskText(row)) : null,
          running
            ? h('label', { class: 'container-remove-confirm__force' }, [
                h('input', {
                  checked: force.value,
                  type: 'checkbox',
                  onInput: (event: Event) => {
                    force.value = (event.target as HTMLInputElement).checked;
                  },
                }),
                h('span', t('container.list.actions.forceRemove')),
              ])
            : null,
        ]),
      theme: 'danger',
      confirmBtn: t('container.list.actions.remove'),
      cancelBtn: t('container.list.actions.cancel'),
      onCancel: () =>
        closeConfirmDialog(
          dialog,
          resolve,
          undefined,
          () => resolved,
          (value) => (resolved = value),
        ),
      onClose: () =>
        closeConfirmDialog(
          dialog,
          resolve,
          undefined,
          () => resolved,
          (value) => (resolved = value),
        ),
      onConfirm: () => {
        closeConfirmDialog(
          dialog,
          resolve,
          force.value,
          () => resolved,
          (value) => (resolved = value),
        );
      },
    });
    activeDangerousDialog.value = dialog;
  });
};

function closeConfirmDialog<T>(
  dialog: DialogInstance,
  resolve: (value: T) => void,
  value: T,
  isResolved: () => boolean,
  setResolved: (value: boolean) => void,
) {
  dangerousDialogOpen.value = false;
  if (activeDangerousDialog.value === dialog) {
    activeDangerousDialog.value = null;
  }
  if (isResolved()) return;

  setResolved(true);
  dialog.hide();
  resolve(value);
}

async function executeDangerousAction(row: ContainerSummaryRecord, action: DangerousContainerAction, force: boolean) {
  try {
    const response =
      action === 'start'
        ? await startContainer(row.id)
        : action === 'stop'
          ? await stopContainer(row.id)
          : action === 'restart'
            ? await restartContainer(row.id)
            : await removeContainer(row.id, { force });
    const messageKey = response.message_key;
    MessagePlugin.success(messageKey ? t(messageKey) : response.message || t('container.list.actionSuccess'));
    selectedRowKeys.value = selectedRowKeys.value.filter((key) => String(key) !== row.id);
    selectedRowRecords.value = selectedRowRecords.value.filter((selectedRow) => selectedRow.id !== row.id);
    await refreshContainers();
  } catch (error) {
    logger.warn(`failed to ${action} container`, error);
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('container.list.actionFailed')));
  }
}

function isDangerousActionDisabled(row: ContainerSummaryRecord, action: DangerousContainerAction) {
  if (!row.id || row.state === 'unknown' || row.state === 'removing') {
    return true;
  }
  if (orchestratorActionLevel(row) === 'readonly') {
    return true;
  }

  if (action === 'start') return !row.can_start;
  if (action === 'stop') return !row.can_stop;
  if (action === 'restart') return !row.can_restart;
  return !row.can_remove;
}

function actionDialogTitleKey(action: DangerousContainerAction) {
  return `container.list.actions.confirm${capitalizeAction(action)}Title`;
}

function actionConfirmKey(action: DangerousContainerAction) {
  return `container.list.actions.confirm${capitalizeAction(action)}`;
}

function capitalizeAction(action: DangerousContainerAction) {
  return `${action.charAt(0).toUpperCase()}${action.slice(1)}`;
}

function batchActionHint(action: DangerousContainerAction) {
  if (!selectedRows.value.length) {
    return t('container.list.batch.noSelection');
  }

  const actionableCount = batchActionableRows(action).length;
  return isBatchActionDisabled(action)
    ? t('container.list.actions.dangerousDisabled')
    : t(`container.list.batch.${action}Hint`, { count: actionableCount });
}

function isBatchActionDisabled(action: DangerousContainerAction) {
  return batchActionableRows(action).length === 0;
}

function batchActionableRows(action: DangerousContainerAction) {
  return selectedRows.value.filter((row) => isBatchActionEligible(row, action));
}

function clearSelection() {
  selectedRowKeys.value = [];
  selectedRowRecords.value = [];
}

function handleSelectChange(rowKeys: Array<string | number>) {
  const rowMap = new Map(rows.value.map((row) => [row.id, row]));
  const currentPageIds = new Set(rowMap.keys());
  const preservedRowKeys = selectedRowKeys.value.filter((key) => !currentPageIds.has(String(key)));
  const normalizedCurrentPageKeys = rowKeys.filter((key) => rowMap.has(String(key)));

  selectedRowKeys.value = [...preservedRowKeys, ...normalizedCurrentPageKeys];
  syncSelectedRowsFromCurrentPage();
}

function confirmBatchAction(action: DangerousContainerAction) {
  if (isBatchActionDisabled(action)) {
    MessagePlugin.warning(t('container.list.actions.dangerousDisabled'));
    return;
  }
  if (dangerousDialogOpen.value) {
    return;
  }

  dangerousDialogOpen.value = true;
  const force = ref(false);
  const selectedCount = selectedRows.value.length;
  const actionableRows = batchActionableRows(action);
  const actionableCount = actionableRows.length;
  const skippedCount = selectedCount - actionableCount;
  const sourceBlockedCount = selectedRows.value.filter(
    (row) => !isDangerousActionDisabled(row, action) && !isBatchActionEligible(row, action),
  ).length;
  const runningCountForRemove =
    action === 'remove' ? actionableRows.filter((row) => row.state === 'running').length : 0;
  let resolved = false;
  const dialog = DialogPlugin.confirm({
    header: t(`container.list.batch.confirm${capitalizeAction(action)}Title`),
    body: () =>
      h('div', { class: 'container-remove-confirm' }, [
        h('p', t(`container.list.batch.confirm${capitalizeAction(action)}`, { count: actionableCount })),
        h(
          'p',
          t('container.list.batch.confirmScope', {
            actionableCount,
            selectedCount,
            skippedCount,
          }),
        ),
        skippedCount > 0 ? h('p', t('container.list.batch.skipInapplicable')) : null,
        sourceBlockedCount > 0
          ? h('p', t('container.list.batch.skipSourceRestricted', { count: sourceBlockedCount }))
          : null,
        action === 'remove' && runningCountForRemove > 0
          ? h('p', t('container.list.batch.confirmRemoveRunning', { count: runningCountForRemove }))
          : null,
        action === 'remove' && runningCountForRemove > 0
          ? h('label', { class: 'container-remove-confirm__force' }, [
              h('input', {
                checked: force.value,
                type: 'checkbox',
                onInput: (event: Event) => {
                  force.value = (event.target as HTMLInputElement).checked;
                },
              }),
              h('span', t('container.list.actions.forceRemove')),
            ])
          : null,
      ]),
    theme: action === 'start' ? 'warning' : 'danger',
    confirmBtn: t('container.list.actions.confirm'),
    cancelBtn: t('container.list.actions.cancel'),
    onCancel: () =>
      closeConfirmDialog(
        dialog,
        () => undefined,
        undefined,
        () => resolved,
        (value) => (resolved = value),
      ),
    onClose: () =>
      closeConfirmDialog(
        dialog,
        () => undefined,
        undefined,
        () => resolved,
        (value) => (resolved = value),
      ),
    onConfirm: async () => {
      dialog.setConfirmLoading(true);
      try {
        const completed = await executeBatchAction(action, force.value, actionableRows);
        if (completed) {
          closeConfirmDialog(
            dialog,
            () => undefined,
            undefined,
            () => resolved,
            (value) => (resolved = value),
          );
        }
      } finally {
        dialog.setConfirmLoading(false);
      }
    },
  });
  activeDangerousDialog.value = dialog;
}

async function executeBatchAction(
  action: DangerousContainerAction,
  force: boolean,
  actionRows = batchActionableRows(action),
) {
  const ids = actionRows.map((row) => row.id);
  if (!ids.length) return false;

  batchActionLoading.value = action;
  try {
    const response = await batchContainerActions({ action, ids, force: action === 'remove' ? force : false });
    syncSelectionAfterBatchAction(action, response, ids);
    handleBatchActionResult(response);
    await refreshContainers();
    return true;
  } catch (error) {
    logger.warn(`failed to batch ${action} containers`, error);
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('container.list.batch.failed')));
    return false;
  } finally {
    batchActionLoading.value = '';
  }
}

function syncSelectionAfterBatchAction(
  action: DangerousContainerAction,
  response: ContainerBatchActionResponse,
  attemptedIds: string[],
) {
  if (action !== 'remove') {
    return;
  }

  const removedIds = new Set(
    response.items
      .filter((item) => item.success)
      .map((item) => item.id)
      .filter((id): id is string => Boolean(id)),
  );
  if (!removedIds.size && response.failed_count === 0) {
    attemptedIds.forEach((id) => removedIds.add(id));
  }
  if (!removedIds.size) {
    return;
  }

  selectedRowKeys.value = selectedRowKeys.value.filter((key) => !removedIds.has(String(key)));
  selectedRowRecords.value = selectedRowRecords.value.filter((row) => !removedIds.has(row.id));
}

function handleBatchActionResult(response: ContainerBatchActionResponse) {
  if (response.failed_count === 0) {
    MessagePlugin.success(t('container.list.batch.success', { count: response.success_count }));
    return;
  }

  if (response.success_count > 0) {
    void NotifyPlugin.warning({
      title: t('container.list.batch.partialTitle'),
      content: batchFailureSummary(response.items),
      duration: 0,
      closeBtn: true,
    });
    return;
  }

  MessagePlugin.error(t('container.list.batch.failed'));
  DialogPlugin.alert({
    header: t('container.list.batch.failureDetailTitle'),
    body: batchFailureSummary(response.items),
    confirmBtn: t('container.list.actions.confirm'),
    theme: 'danger',
  });
}

function batchFailureSummary(items: ContainerBatchActionItem[]) {
  const failedItems = items.filter((item) => !item.success);
  if (!failedItems.length) {
    return t('container.list.batch.noFailureDetail');
  }

  return failedItems
    .slice(0, 5)
    .map((item) => `${item.name || item.id}: ${item.message_key ? t(item.message_key) : item.message || '-'}`)
    .join('\n');
}

function navigateToDetail(row: ContainerSummaryRecord, tab: string) {
  const target = {
    name: CONTAINER_BOOTSTRAP_ROUTE.DETAIL.pageRouteName,
    params: { id: row.id },
    query: { tab },
  };
  const resolved = router.resolve(target);
  tabsRouterStore.appendTabRouterList({
    tabKey: resolved.path,
    path: resolved.path,
    fullPath: resolved.fullPath,
    query: resolved.query,
    params: resolved.params,
    title: buildDetailTabTitle(displayContainerName(row)),
    name: resolved.name,
    isAlive: true,
    meta: resolved.meta as AppRouteMeta,
  });
  tabsRouterStore.setActiveTabKey(resolved.path);

  return router.push(target);
}

function handlePageChange(pageInfo: { current?: number; pageSize?: number }) {
  if (pageInfo.current) {
    pagination.current = pageInfo.current;
  }
  if (pageInfo.pageSize) {
    pagination.pageSize = pageInfo.pageSize;
  }
}

function buildDetailTabTitle(name: string): LocalizedTitle {
  const baseTitle = localizeRouteTitleKey('container.detail.title');
  return {
    [LOCALE.ZH_CN]: `${baseTitle[LOCALE.ZH_CN]} - ${name}`,
    [LOCALE.EN_US]: `${baseTitle[LOCALE.EN_US]} - ${name}`,
  };
}

function orchestratorActionLevel(row: ContainerSummaryRecord): ContainerActionLevel {
  if (row.deployment?.action_level) {
    return row.deployment.action_level;
  }

  return row.can_start || row.can_stop || row.can_restart || row.can_remove ? 'allow' : 'readonly';
}

function rowActionRiskText(row: ContainerSummaryRecord) {
  return orchestratorActionLevel(row) === 'warn'
    ? t('container.list.actions.sourceRisk', {
        source: t(`container.list.deployments.${row.deployment?.type || 'standalone'}`),
      })
    : '';
}

function canRunDangerousAction(row: ContainerSummaryRecord, action: DangerousContainerAction) {
  if (orchestratorActionLevel(row) === 'readonly') {
    return false;
  }
  if (action === 'start') return Boolean(row.can_start);
  if (action === 'stop') return Boolean(row.can_stop);
  if (action === 'restart') return Boolean(row.can_restart);
  return Boolean(row.can_remove);
}

function canRunAnyDangerousAction(row: ContainerSummaryRecord) {
  return (['start', 'stop', 'restart', 'remove'] as DangerousContainerAction[]).some((action) =>
    canRunDangerousAction(row, action),
  );
}

function isBatchActionEligible(row: ContainerSummaryRecord, action: DangerousContainerAction) {
  return !isDangerousActionDisabled(row, action) && (row.deployment?.batch_action_allowed ?? true);
}

function toggleTableDensity() {
  tableDensity.value = tableDensity.value === 'medium' ? 'small' : 'medium';
}

function loadVisibleColumnKeys() {
  if (typeof window === 'undefined') {
    return [...CONTAINER_RESOURCE_DEFAULT_VISIBLE_COLUMNS];
  }

  try {
    const stored = window.localStorage.getItem(CONTAINER_RESOURCE_COLUMN_STORAGE_KEY);
    if (!stored) {
      return [...CONTAINER_RESOURCE_DEFAULT_VISIBLE_COLUMNS];
    }
    const parsed = JSON.parse(stored);
    if (!Array.isArray(parsed)) {
      return [...CONTAINER_RESOURCE_DEFAULT_VISIBLE_COLUMNS];
    }

    const normalizedKeys = normalizeVisibleColumnKeys(parsed);
    persistVisibleColumnKeys(normalizedKeys);
    return normalizedKeys;
  } catch {
    return [...CONTAINER_RESOURCE_DEFAULT_VISIBLE_COLUMNS];
  }
}

function persistVisibleColumnKeys(keys: string[]) {
  if (typeof window === 'undefined') {
    return;
  }

  try {
    window.localStorage.setItem(CONTAINER_RESOURCE_COLUMN_STORAGE_KEY, JSON.stringify(keys));
  } catch {
    // Column settings are a convenience preference; list rendering must not depend on storage availability.
  }
}

function normalizeVisibleColumnKeys(keys: unknown[]) {
  const availableKeySet = new Set(CONTAINER_RESOURCE_ALL_COLUMN_KEYS);
  const nextKeys = new Set<string>();

  for (const key of keys) {
    if (typeof key === 'string' && availableKeySet.has(key)) {
      nextKeys.add(key);
    }
  }

  for (const key of CONTAINER_RESOURCE_ALWAYS_VISIBLE_COLUMNS) {
    nextKeys.add(key);
  }

  return CONTAINER_RESOURCE_ALL_COLUMN_KEYS.filter((key) => nextKeys.has(key));
}
</script>
<style scoped lang="less">
.container-page {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
  min-width: 0;
}

.container-toolbar-row {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
}

.container-table-head,
.container-image,
.container-identity,
.container-source-cell {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.container-table-head p,
.container-detail-section h3,
.container-detail-section h4 {
  margin: 0;
}

.container-table-head__summary,
.container-identity__name {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
}

.container-table-head p:not(.container-table-head__summary),
.container-identity__id,
.container-muted {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.container-identity__id {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.container-port-list,
.container-actions,
.container-batch-bar,
.container-batch-bar__actions,
.container-remove-confirm__force {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-6);
}

.container-batch-bar {
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.container-batch-bar > span {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
}

.container-batch-bar__actions,
.container-remove-confirm__force {
  align-items: center;
}

.container-remove-confirm {
  color: var(--td-text-color-primary);
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
}

.container-remove-confirm p {
  margin: 0;
}

.container-remove-confirm__risk {
  color: var(--td-warning-color-7);
}

.container-runtime-status {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.container-runtime-status__text {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.container-runtime-status .t-tag {
  align-self: flex-start;
}

.container-source-cell {
  align-items: flex-start;
}

.container-source-cell__header {
  align-items: center;
  display: flex;
}

.container-source-cell__line {
  align-items: center;
  display: inline-flex;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.container-source-cell__label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.container-source-cell .t-tag,
.container-source-cell :deep(.t-button) {
  align-self: flex-start;
}

.container-source-cell :deep(.t-button) {
  min-width: auto;
  padding: 0;
}

.management-toolbar__scope-input {
  flex: 0 1 var(--graft-list-select-width);
  min-width: 180px;
  width: var(--graft-list-select-width);
}

.container-resource-meter {
  align-items: center;
  border-radius: 999px;
  display: inline-flex;
  gap: var(--graft-density-gap-8);
  justify-content: center;
  min-width: 0;
  overflow: hidden;
  padding: var(--graft-density-gap-2) var(--graft-density-gap-8) var(--graft-density-gap-2) var(--graft-density-gap-2);
  position: relative;
  transform: translateZ(0);
  transition:
    background-color 180ms ease,
    color 180ms ease,
    opacity 180ms ease,
    transform 180ms ease;
  white-space: nowrap;
  will-change: background-color, opacity, transform;
}

.container-resource-meter::after {
  border-radius: inherit;
  content: '';
  inset: 0;
  opacity: 0;
  pointer-events: none;
  position: absolute;
  transform: scaleX(0.96);
}

.container-resource-meter > span:last-child {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-small);
  overflow: hidden;
  text-overflow: ellipsis;
}

.container-resource-meter[data-available='false'] > span:last-child {
  color: var(--td-text-color-secondary);
}

.container-resource-meter__empty {
  border: 1px dashed var(--td-component-stroke);
  border-radius: 50%;
  display: inline-block;
  flex: 0 0 36px;
  height: 36px;
  width: 36px;
}

.container-resource-meter.container-metric-change--up {
  animation: container-resource-shift-up 480ms cubic-bezier(0.2, 0.8, 0.2, 1);
  background: color-mix(in srgb, var(--td-warning-color-1) 58%, transparent);
  transform: translateY(-1px);
}

.container-resource-meter.container-metric-change--down {
  animation: container-resource-shift-down 480ms cubic-bezier(0.2, 0.8, 0.2, 1);
  background: color-mix(in srgb, var(--td-success-color-1) 60%, transparent);
}

.container-resource-meter.container-metric-change--up::after {
  animation: container-resource-overlay-up 520ms ease-out;
  background: linear-gradient(
    90deg,
    color-mix(in srgb, var(--td-warning-color-3) 0%, transparent) 0%,
    color-mix(in srgb, var(--td-warning-color-4) 18%, transparent) 28%,
    color-mix(in srgb, var(--td-warning-color-5) 24%, transparent) 52%,
    color-mix(in srgb, var(--td-warning-color-3) 0%, transparent) 100%
  );
}

.container-resource-meter.container-metric-change--down::after {
  animation: container-resource-overlay-down 520ms ease-out;
  background: linear-gradient(
    90deg,
    color-mix(in srgb, var(--td-success-color-3) 0%, transparent) 0%,
    color-mix(in srgb, var(--td-success-color-4) 16%, transparent) 28%,
    color-mix(in srgb, var(--td-success-color-5) 20%, transparent) 52%,
    color-mix(in srgb, var(--td-success-color-3) 0%, transparent) 100%
  );
}

@keyframes container-resource-shift-up {
  0% {
    opacity: 0.92;
    transform: translate3d(0, 1px, 0);
  }

  40% {
    opacity: 1;
    transform: translate3d(0, -1px, 0);
  }

  100% {
    opacity: 1;
    transform: translate3d(0, -1px, 0);
  }
}

@keyframes container-resource-shift-down {
  0% {
    opacity: 0.92;
    transform: translate3d(0, -1px, 0);
  }

  40% {
    opacity: 1;
    transform: translate3d(0, 0, 0);
  }

  100% {
    opacity: 1;
    transform: translate3d(0, 0, 0);
  }
}

@keyframes container-resource-overlay-up {
  0% {
    opacity: 0;
    transform: scaleX(0.94) translateX(-8%);
  }

  30% {
    opacity: 1;
  }

  100% {
    opacity: 0;
    transform: scaleX(1.02) translateX(8%);
  }
}

@keyframes container-resource-overlay-down {
  0% {
    opacity: 0;
    transform: scaleX(0.94) translateX(8%);
  }

  30% {
    opacity: 1;
  }

  100% {
    opacity: 0;
    transform: scaleX(1.02) translateX(-8%);
  }
}

@media (prefers-reduced-motion: reduce) {
  .container-resource-meter,
  .container-resource-meter::after,
  .container-resource-meter.container-metric-change--up,
  .container-resource-meter.container-metric-change--down {
    animation: none;
    transform: none;
    transition-duration: 0ms;
  }
}

.container-actions {
  flex-wrap: nowrap;
  justify-content: center;
}

.container-alert {
  margin-bottom: var(--graft-density-gap-12);
}

.container-table-host {
  max-width: 100%;
  min-width: 0;
  overflow-x: hidden;
  width: 100%;
}

.container-table-host[data-table-mode='scroll'] {
  overflow-x: auto;
}

.container-table-host :deep(.t-table__content) {
  min-width: 0;
}

.container-table-host :deep(.t-table__content table) {
  min-width: 100%;
}

.container-alert__hint {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: var(--graft-density-gap-4) 0 0;
}

@media (width <= 768px) {
  .container-toolbar-row {
    align-items: stretch;
  }

  .container-actions {
    justify-content: flex-start;
  }
}
</style>
