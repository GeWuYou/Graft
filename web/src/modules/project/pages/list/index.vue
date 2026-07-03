<template>
  <div class="project-page" data-page-type="list-form-detail">
    <management-page-content>
      <management-page-header
        title-key="project.list.title"
        description-key="project.list.description"
        :source="{ labelKey: 'menu.ops.title', fallback: t('project.list.eyebrow') }"
      >
        <template #meta>
          <div class="project-header-summary">
            <span class="project-header-summary__total" data-testid="project-status-summary-total">
              {{ t('project.list.projectCount', { count: totalCount }) }}
            </span>
            <t-tooltip v-for="item in headerStatusSummaryItems" :key="item.key" :content="item.tooltip" placement="top">
              <span
                :class="['project-header-summary__status', `project-header-summary__status--${item.key}`]"
                :data-testid="`project-status-summary-${item.key}`"
              >
                {{ item.icon }}{{ item.count }}
              </span>
            </t-tooltip>
          </div>
        </template>
        <template #actions>
          <t-space size="small" break-line>
            <project-list-entry-actions
              :import-label="t('project.list.actions.import')"
              :create-label="t('project.list.actions.create')"
              :reset-label="t('project.list.clearFilters')"
              :show-reset="hasActiveFilters"
              @import="navigateToImport"
              @create="navigateToSourceChooser"
              @reset="resetFilters"
            />
            <t-button theme="primary" :loading="refreshLoading" @click="handleManualRefresh">
              <template #icon><refresh-icon /></template>
              {{ t('project.list.refresh') }}
            </t-button>
          </t-space>
        </template>
      </management-page-header>

      <management-toolbar>
        <template #filters>
          <t-input
            v-model="filters.keyword"
            class="management-list-search"
            clearable
            :placeholder="t('project.list.filters.searchPlaceholder')"
            @enter="handleFilterQuery"
          />
          <t-select
            v-model="filters.sourceKind"
            class="management-toolbar__select"
            :placeholder="t('project.list.filters.sourceKind')"
          >
            <t-option value="all" :label="t('project.list.filters.allSourceKinds')" />
            <t-option
              v-for="option in sourceKindOptions"
              :key="option"
              :value="option"
              :label="sourceKindLabel(option)"
            />
          </t-select>
          <t-select
            v-model="filters.driftStatus"
            class="management-toolbar__select"
            :placeholder="t('project.list.filters.driftStatus')"
          >
            <t-option value="all" :label="t('project.list.filters.allDriftStatuses')" />
            <t-option
              v-for="option in driftStatusOptions"
              :key="option"
              :value="option"
              :label="driftStatusLabel(option)"
            />
          </t-select>
          <t-select
            v-model="filters.lastRefreshStatus"
            class="management-toolbar__select"
            :placeholder="t('project.list.filters.refreshStatus')"
          >
            <t-option value="all" :label="t('project.list.filters.allRefreshStatuses')" />
            <t-option
              v-for="option in refreshStatusOptions"
              :key="option"
              :value="option"
              :label="refreshStatusLabel(option)"
            />
          </t-select>
          <t-button theme="primary" @click="handleFilterQuery">{{ t('project.list.filters.query') }}</t-button>
          <t-button theme="default" variant="text" @click="resetFilters">{{
            t('project.list.filters.reset')
          }}</t-button>
        </template>
      </management-toolbar>

      <management-table-card>
        <template #head>
          <div class="project-table-head">
            <div>
              <p class="project-table-head__summary">{{ t('project.list.tableSummary', { count: rows.length }) }}</p>
              <p class="project-table-head__hint">{{ t('project.list.tableHint') }}</p>
            </div>
          </div>
        </template>
        <template #toolbar>
          <table-view-toolbar
            :column-settings-label="t('project.list.columnSettings')"
            :refresh-label="t('project.list.refresh')"
            :refresh-loading="refreshLoading"
            @column-settings="columnDrawerVisible = true"
            @refresh="handleManualRefresh"
          />
        </template>

        <management-empty-state
          v-if="errorMessage && !tableLoading && !refreshing"
          tone="error"
          :title="t('project.list.title')"
          :description="errorMessage"
        >
          <template #actions>
            <t-button theme="primary" variant="outline" @click="handleManualRefresh">
              {{ t('project.list.retry') }}
            </t-button>
          </template>
        </management-empty-state>

        <div v-else ref="tableHostRef" class="project-table-host" :data-table-mode="tableWidthPolicy.mode">
          <t-table
            row-key="id"
            :columns="visibleColumns"
            :data="rows"
            :loading="tableLoading"
            table-layout="fixed"
            :table-content-width="tableWidthPolicy.tableContentWidth"
            cell-empty-content="-"
            hover
          >
            <template #name="{ row }">
              <div class="project-identity">
                <button
                  class="project-identity__main"
                  type="button"
                  @click="navigateToDetail(projectRow(row), 'overview')"
                >
                  <strong>{{ projectRow(row).display_name }}</strong>
                  <span v-if="projectSecondaryName(projectRow(row))">{{ projectSecondaryName(projectRow(row)) }}</span>
                </button>
                <code>{{ projectRow(row).working_directory }}</code>
              </div>
            </template>

            <template #source="{ row }">
              <t-tag theme="default" variant="light-outline">
                {{ sourceKindLabel(projectRow(row).source_kind) }}
              </t-tag>
            </template>

            <template #runtime="{ row }">
              <span
                :class="[
                  'project-runtime-badge',
                  `project-runtime-badge--${normalizeRuntimeStatus(projectRow(row).runtime_status)}`,
                  { 'project-runtime-badge--loading': isRowActionPending(projectRow(row).id) },
                ]"
                :data-testid="`project-runtime-status-${projectRow(row).id}`"
              >
                <span
                  v-if="isRowActionPending(projectRow(row).id)"
                  class="project-runtime-badge__spinner"
                  :data-testid="`project-runtime-status-loading-${projectRow(row).id}`"
                />
                <template v-else>
                  {{ runtimeStatusLabel(projectRow(row).runtime_status) }}
                </template>
              </span>
            </template>

            <template #resources="{ row }">
              <div class="project-resources">
                <div class="project-resources__item">
                  <span class="project-resources__label">{{ t('project.list.resources.service') }}</span>
                  <strong>{{ projectRow(row).service_count }}</strong>
                </div>
                <div class="project-resources__item">
                  <span class="project-resources__label">{{ t('project.list.resources.container') }}</span>
                  <div class="project-resource-badges">
                    <span
                      v-for="badge in projectContainerBadges(projectRow(row))"
                      :key="badge.key"
                      :class="['project-resource-badge', `project-resource-badge--${badge.key}`]"
                      :aria-label="badge.label"
                      :data-testid="`project-resource-badge-${badge.key}-${projectRow(row).id}`"
                      :title="badge.label"
                    >
                      <span class="project-resource-badge__icon" aria-hidden="true">{{ badge.icon }}</span>
                      {{ badge.count }}
                    </span>
                  </div>
                </div>
              </div>
            </template>

            <template #drift="{ row }">
              <span
                :class="[
                  'project-sync-status',
                  `project-sync-status--${normalizeDriftStatus(projectRow(row).drift_status)}`,
                ]"
              >
                {{ driftStatusLabel(projectRow(row).drift_status) }}
              </span>
            </template>

            <template #refresh="{ row }">
              <div class="project-refresh">
                <t-tag :theme="refreshStatusTheme(projectRow(row).last_refresh_status)" variant="light-outline">
                  {{ refreshStatusLabel(projectRow(row).last_refresh_status) }}
                </t-tag>
                <span>{{ formatTime(projectRow(row).last_refresh_at) }}</span>
              </div>
            </template>

            <template #operation="{ row }">
              <table-action-menu
                :actions="buildRowActions(projectRow(row))"
                :more-label="t('project.list.actions.operationMenu')"
                :more-label-fallback="t('project.list.actions.operationMenu')"
                @action="(action) => handleRowAction(action, projectRow(row))"
              />
            </template>

            <template #empty>
              <div class="project-empty">
                <t-empty
                  :title="t('project.list.emptyTitle')"
                  :description="
                    hasActiveFilters ? t('project.list.emptyFilteredDescription') : t('project.list.emptyDescription')
                  "
                >
                  <template #action>
                    <project-list-entry-actions
                      :import-label="t('project.list.actions.import')"
                      :create-label="t('project.list.actions.create')"
                      :reset-label="t('project.list.clearFilters')"
                      :show-reset="hasActiveFilters"
                      @import="navigateToImport"
                      @create="navigateToSourceChooser"
                      @reset="resetFilters"
                    />
                  </template>
                </t-empty>
              </div>
            </template>
          </t-table>
        </div>

        <template #footer>
          <management-table-pagination :summary="paginationSummary">
            <t-pagination
              v-model:current="pagination.current"
              v-model:page-size="pagination.pageSize"
              :total="pagination.total"
              :page-size-options="[10, 20, 50, 100]"
              :show-page-number="true"
              @change="handlePageChange"
            />
          </management-table-pagination>
        </template>
      </management-table-card>

      <t-drawer
        v-model:visible="columnDrawerVisible"
        :header="t('project.list.columnDrawerTitle')"
        size="420px"
        :footer="false"
      >
        <div class="project-column-drawer">
          <t-checkbox-group v-model="visibleColumnKeys">
            <t-space direction="vertical" size="small">
              <t-checkbox v-for="column in configurableColumns" :key="column.colKey" :value="String(column.colKey)">
                {{ column.title as string }}
              </t-checkbox>
            </t-space>
          </t-checkbox-group>
        </div>
      </t-drawer>
    </management-page-content>
  </div>
</template>
<script setup lang="ts">
import { RefreshIcon } from 'tdesign-icons-vue-next';
import type { DialogInstance, TableProps } from 'tdesign-vue-next';
import { DialogPlugin } from 'tdesign-vue-next/es/dialog';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, h, onActivated, onDeactivated, onMounted, onUnmounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import {
  ManagementEmptyState,
  ManagementPageContent,
  ManagementPageHeader,
  ManagementTableCard,
  ManagementTablePagination,
  ManagementToolbar,
  resolveTableWidthPolicy,
  TableActionMenu,
  TableViewToolbar,
  useTableHostWidth,
} from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { useTabsRouterStore } from '@/store/modules/tabs-router';
import { createLogger } from '@/utils/logger';
import { localizeRouteTitleKey } from '@/utils/route/title';

import {
  getProjects,
  postProjectDown,
  postProjectRefresh,
  postProjectRestart,
  postProjectUnregister,
  postProjectUp,
} from '../../api/project';
import ProjectListEntryActions from '../../components/ProjectListEntryActions.vue';
import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import {
  formatProjectTime,
  projectDriftStatusLabel,
  projectLifecycleActionVisibility,
  projectRefreshStatusLabel,
  projectRefreshStatusTheme,
  projectRuntimeStatusLabel,
  projectSourceKindLabel,
} from '../../shared/display';
import { appendResolvedTab, buildDetailTitleWithFallback } from '../../shared/navigation';
import type {
  ProjectDetailResponse,
  ProjectDriftStatus,
  ProjectFilters,
  ProjectListItem,
  ProjectRefreshStatus,
  ProjectRuntimeStatus,
  ProjectSourceKind,
} from '../../types/project';

defineOptions({
  name: 'ProjectListIndex',
});

const { locale, t } = useI18n();
const router = useRouter();
const tabsRouterStore = useTabsRouterStore();
const logger = createLogger('project.list');

type HeaderStatusSummaryKey = 'running' | 'degraded' | 'stopped' | 'transitioning' | 'unknown';
type ProjectListDriftTone = 'clean' | 'drifted' | 'unknown';
type PendingProjectAction = 'up' | 'down' | 'restart';
type ProjectResourceBadgeKey = 'running' | 'stopped' | 'transitioning' | 'issue' | 'unknown';
type PendingProjectActionState = {
  action: PendingProjectAction;
  lastRefreshAt: string | null;
  runtimeStatus: ProjectRuntimeStatus | null;
};
type ProjectResourceBadge = {
  key: ProjectResourceBadgeKey;
  count: number;
  label: string;
  icon: string;
};

const PROJECT_LIST_POLL_INTERVAL_MS = 5000;

const tableLoading = ref(false);
const refreshing = ref(false);
const errorMessage = ref('');
const rows = ref<ProjectListItem[]>([]);
const pagination = ref({
  current: 1,
  pageSize: 20,
  total: 0,
});
const filters = ref<ProjectFilters>({
  keyword: '',
  sourceKind: 'all',
  driftStatus: 'all',
  lastRefreshStatus: 'all',
});
const columnDrawerVisible = ref(false);
const { tableHostRef, tableHostWidth } = useTableHostWidth(() => visibleColumnKeys.value.join(','));

const sourceKindOptions: ProjectSourceKind[] = ['imported', 'managed', 'git', 'template'];
const driftStatusOptions: ProjectDriftStatus[] = ['unknown', 'clean', 'changed', 'missing'];
const refreshStatusOptions: ProjectRefreshStatus[] = ['never', 'success', 'failed'];

const configurableColumns = computed<TableProps['columns']>(() => [
  { colKey: 'name', title: t('project.list.columns.name'), width: 300 },
  { colKey: 'source', title: t('project.list.columns.source'), width: 112, align: 'center' },
  { colKey: 'runtime', title: t('project.list.columns.runtime'), width: 148, align: 'center' },
  { colKey: 'resources', title: t('project.list.columns.resources'), width: 236, align: 'center' },
  { colKey: 'drift', title: t('project.list.columns.drift'), width: 124, align: 'center' },
  { colKey: 'refresh', title: t('project.list.columns.refresh'), width: 168, align: 'center' },
  { colKey: 'operation', title: t('project.list.columns.operation'), width: 152, fixed: 'right', align: 'center' },
]);
const visibleColumnKeys = ref(['name', 'source', 'runtime', 'resources', 'drift', 'refresh', 'operation']);
const visibleColumns = computed(() =>
  (configurableColumns.value ?? []).filter((column) => visibleColumnKeys.value.includes(String(column?.colKey))),
);
const tableWidthPolicy = computed(() => resolveTableWidthPolicy(visibleColumns.value ?? [], tableHostWidth.value));
const confirmDialogOpen = ref(false);
const refreshLoading = computed(() => tableLoading.value || refreshing.value);
const realtimeActive = ref(false);
const pendingRowActions = ref<Record<number, PendingProjectActionState>>({});

const totalCount = computed(() => rows.value.length);
const projectStatusCounts = computed<Record<HeaderStatusSummaryKey, number>>(() => {
  const counts: Record<HeaderStatusSummaryKey, number> = {
    running: 0,
    degraded: 0,
    stopped: 0,
    transitioning: 0,
    unknown: 0,
  };

  for (const row of rows.value) {
    counts[normalizeRuntimeStatus(row.runtime_status)] += 1;
  }

  return counts;
});
const headerStatusSummaryItems = computed(() =>
  (
    [
      { key: 'running', icon: '🟢', tooltip: t('project.list.statusTooltip.runtimeRunning') },
      { key: 'degraded', icon: '🟡', tooltip: t('project.list.statusTooltip.runtimeDegraded') },
      { key: 'stopped', icon: '⚫', tooltip: t('project.list.statusTooltip.runtimeStopped') },
      { key: 'transitioning', icon: '🔵', tooltip: t('project.list.statusTooltip.runtimeTransitioning') },
      { key: 'unknown', icon: '⚪', tooltip: t('project.list.statusTooltip.runtimeUnknown') },
    ] as const
  )
    .filter((item) => projectStatusCounts.value[item.key] > 0)
    .map((item) => ({
      ...item,
      count: projectStatusCounts.value[item.key],
    })),
);
const hasActiveFilters = computed(
  () =>
    Boolean(filters.value.keyword.trim()) ||
    filters.value.sourceKind !== 'all' ||
    filters.value.driftStatus !== 'all' ||
    filters.value.lastRefreshStatus !== 'all',
);
const paginationSummary = computed(() => {
  if (!pagination.value.total || rows.value.length === 0) {
    return t('project.list.tableSummary', { count: rows.value.length });
  }
  const start = (pagination.value.current - 1) * pagination.value.pageSize + 1;
  const end = start + rows.value.length - 1;
  return `${start}-${end} / ${pagination.value.total}`;
});

onMounted(() => {
  realtimeActive.value = true;
  startPolling();
  void fetchProjects();
});

onUnmounted(() => {
  realtimeActive.value = false;
  stopPolling();
});

onActivated(() => {
  realtimeActive.value = true;
  startPolling();
});

onDeactivated(() => {
  realtimeActive.value = false;
  stopPolling();
});

function projectRow(row: unknown) {
  return row as ProjectListItem;
}

function sourceKindLabel(value: ProjectSourceKind) {
  return projectSourceKindLabel(t, value);
}

function driftStatusLabel(value: ProjectDriftStatus) {
  return projectDriftStatusLabel(t, value);
}

function refreshStatusLabel(value: ProjectRefreshStatus) {
  return projectRefreshStatusLabel(t, value);
}

function refreshStatusTheme(value: ProjectRefreshStatus) {
  return projectRefreshStatusTheme(value);
}

function runtimeStatusLabel(value?: ProjectRuntimeStatus | null) {
  return projectRuntimeStatusLabel(t, value);
}

function normalizeRuntimeStatus(value?: ProjectRuntimeStatus | null): HeaderStatusSummaryKey {
  if (value === 'running' || value === 'degraded' || value === 'stopped' || value === 'transitioning') {
    return value;
  }

  return 'unknown';
}

function normalizeDriftStatus(value: ProjectDriftStatus): ProjectListDriftTone {
  if (value === 'clean') {
    return 'clean';
  }

  if (value === 'unknown') {
    return 'unknown';
  }

  return 'drifted';
}

function projectResourceBadgeLabel(key: ProjectResourceBadgeKey, count: number) {
  return t('project.list.resources.statusValue', {
    count,
    status: t(`project.list.resources.${key}`),
  });
}

function projectResourceBadgeIcon(key: ProjectResourceBadgeKey) {
  if (key === 'running') return '🟢';
  if (key === 'stopped') return '⚫';
  if (key === 'transitioning') return '🔵';
  if (key === 'issue') return '🔴';
  return '⚪';
}

function projectContainerBadges(row: ProjectListItem): ProjectResourceBadge[] {
  const badges: ProjectResourceBadge[] = [
    {
      key: 'running',
      count: row.container_counts.running,
      label: projectResourceBadgeLabel('running', row.container_counts.running),
      icon: projectResourceBadgeIcon('running'),
    },
    {
      key: 'stopped',
      count: row.container_counts.stopped,
      label: projectResourceBadgeLabel('stopped', row.container_counts.stopped),
      icon: projectResourceBadgeIcon('stopped'),
    },
    {
      key: 'transitioning',
      count: row.container_counts.transitioning,
      label: projectResourceBadgeLabel('transitioning', row.container_counts.transitioning),
      icon: projectResourceBadgeIcon('transitioning'),
    },
    {
      key: 'issue',
      count: row.container_counts.issue,
      label: projectResourceBadgeLabel('issue', row.container_counts.issue),
      icon: projectResourceBadgeIcon('issue'),
    },
  ];

  const visible = badges.filter((badge) => badge.count > 0);
  if (visible.length > 0) {
    return visible;
  }
  if ((row.runtime_status ?? null) === 'unknown' && row.container_counts.total === 0) {
    return [
      {
        key: 'unknown',
        count: 0,
        label: projectResourceBadgeLabel('unknown', 0),
        icon: projectResourceBadgeIcon('unknown'),
      },
    ];
  }
  return visible;
}

function formatTime(value?: string | null) {
  return formatProjectTime(locale.value, value);
}

function projectSecondaryName(row: ProjectListItem) {
  const canonicalName = row.canonical_project_name?.trim() || '';
  const displayName = row.display_name?.trim() || '';

  if (!canonicalName || canonicalName === displayName) {
    return '';
  }

  return t('project.list.canonicalNameValue', { name: canonicalName });
}

let refreshRequestSeq = 0;
let pollTimer: number | undefined;

function startPolling() {
  stopPolling();
  if (!realtimeActive.value || typeof window === 'undefined') {
    return;
  }
  pollTimer = window.setInterval(() => {
    if (tableLoading.value || refreshing.value) {
      return;
    }
    void fetchProjects();
  }, PROJECT_LIST_POLL_INTERVAL_MS);
}

function stopPolling() {
  if (pollTimer === undefined || typeof window === 'undefined') {
    return;
  }
  window.clearInterval(pollTimer);
  pollTimer = undefined;
}

async function fetchProjects() {
  const requestSeq = ++refreshRequestSeq;
  const shouldBlockTable = rows.value.length === 0 && !tableLoading.value;
  if (shouldBlockTable) {
    tableLoading.value = true;
  } else {
    refreshing.value = true;
  }
  errorMessage.value = '';
  try {
    const response = await getProjects({
      limit: pagination.value.pageSize,
      offset: (pagination.value.current - 1) * pagination.value.pageSize,
      ...(filters.value.sourceKind !== 'all' ? { source_kind: filters.value.sourceKind } : {}),
      ...(filters.value.driftStatus !== 'all' ? { drift_status: filters.value.driftStatus } : {}),
      ...(filters.value.lastRefreshStatus !== 'all' ? { last_refresh_status: filters.value.lastRefreshStatus } : {}),
    });
    if (requestSeq !== refreshRequestSeq) {
      return;
    }
    syncPaginationFromResponse(response);
    const keyword = filters.value.keyword.trim().toLowerCase();
    const nextRows = keyword
      ? response.items.filter((item) =>
          [item.display_name, item.canonical_project_name, item.working_directory]
            .filter(Boolean)
            .some((candidate) => String(candidate).toLowerCase().includes(keyword)),
        )
      : response.items;
    rows.value = nextRows;
    reconcilePendingRowActions(nextRows);
  } catch (error) {
    if (requestSeq !== refreshRequestSeq) {
      return;
    }
    logger.error('failed to fetch projects', error);
    rows.value = [];
    pagination.value.total = 0;
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('project.list.retry'));
  } finally {
    if (requestSeq === refreshRequestSeq) {
      tableLoading.value = false;
      refreshing.value = false;
    }
  }
}

async function handleManualRefresh() {
  if (refreshLoading.value) {
    return;
  }

  await fetchProjects();
}

function syncPaginationFromResponse(response: { total?: number; limit?: number; offset?: number }) {
  pagination.value.total =
    typeof response.total === 'number' && response.total >= 0 ? response.total : rows.value.length;
  if (typeof response.limit === 'number' && response.limit > 0) {
    pagination.value.pageSize = response.limit;
  }
  if (typeof response.offset === 'number' && response.offset >= 0) {
    pagination.value.current = Math.floor(response.offset / pagination.value.pageSize) + 1;
  }
}

function reconcilePendingRowActions(nextRows: ProjectListItem[]) {
  const nextPending = { ...pendingRowActions.value };
  const rowMap = new Map(nextRows.map((row) => [row.id, row]));

  for (const [rawId, pending] of Object.entries(nextPending)) {
    const id = Number(rawId);
    const row = rowMap.get(id);
    if (!row) {
      delete nextPending[id];
      continue;
    }

    const runtimeChanged = (row.runtime_status ?? null) !== pending.runtimeStatus;
    const refreshChanged = (row.last_refresh_at ?? null) !== pending.lastRefreshAt;
    if (runtimeChanged || refreshChanged) {
      delete nextPending[id];
    }
  }

  pendingRowActions.value = nextPending;
}

function markPendingRowAction(row: ProjectListItem, action: PendingProjectAction) {
  pendingRowActions.value = {
    ...pendingRowActions.value,
    [row.id]: {
      action,
      lastRefreshAt: row.last_refresh_at ?? null,
      runtimeStatus: row.runtime_status ?? null,
    },
  };
}

function clearPendingRowAction(rowId: number) {
  if (!pendingRowActions.value[rowId]) {
    return;
  }
  const nextPending = { ...pendingRowActions.value };
  delete nextPending[rowId];
  pendingRowActions.value = nextPending;
}

function isRowActionPending(rowId: number) {
  return Boolean(pendingRowActions.value[rowId]);
}

function resetFilters() {
  filters.value = {
    keyword: '',
    sourceKind: 'all',
    driftStatus: 'all',
    lastRefreshStatus: 'all',
  };
  pagination.value.current = 1;
  void fetchProjects();
}

function handleFilterQuery() {
  pagination.value.current = 1;
  void fetchProjects();
}

function handlePageChange(pageInfo: { current: number; pageSize: number }) {
  pagination.value.current = pageInfo.current;
  pagination.value.pageSize = pageInfo.pageSize;
  void fetchProjects();
}

function navigateToDetail(row: ProjectListItem, tab?: string) {
  const target = {
    name: PROJECT_BOOTSTRAP_ROUTE.DETAIL.pageRouteName,
    params: { id: row.id },
    query: {
      ...(tab ? { tab } : {}),
      name: row.display_name,
    },
  };
  const resolved = router.resolve(target);
  appendResolvedTab(
    tabsRouterStore,
    resolved,
    buildDetailTitleWithFallback('project.route.detail.title', row.display_name),
  );
  return router.push(target);
}

function navigateToImport() {
  const target = {
    name: PROJECT_BOOTSTRAP_ROUTE.IMPORT.pageRouteName,
  };
  const resolved = router.resolve(target);
  appendResolvedTab(tabsRouterStore, resolved, localizeRouteTitleKey('project.route.import.title'));
  void router.push(target);
}

function navigateToSourceChooser() {
  const target = {
    name: PROJECT_BOOTSTRAP_ROUTE.CREATE.pageRouteName,
  };
  const resolved = router.resolve(target);
  appendResolvedTab(tabsRouterStore, resolved, localizeRouteTitleKey('project.route.create.title'));
  void router.push(target);
}

async function runAction(
  handler: (id: number) => Promise<ProjectDetailResponse | unknown>,
  row: ProjectListItem,
  successMessage: string,
  pendingAction?: PendingProjectAction,
) {
  if (pendingAction) {
    markPendingRowAction(row, pendingAction);
  }
  try {
    await handler(row.id);
    MessagePlugin.success(successMessage);
    await fetchProjects();
  } catch (error) {
    clearPendingRowAction(row.id);
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.list.actions.actionFailed')));
  }
}

function buildRowActions(row: ProjectListItem) {
  const visibility = projectLifecycleActionVisibility(row.runtime_status, {
    hideLifecycleActions: isRowActionPending(row.id),
  });

  return [
    { value: 'detail', label: t('project.list.actions.detail') },
    { value: 'refresh', label: t('project.list.actions.refresh') },
    ...(visibility.up ? [{ value: 'up', label: t('project.list.actions.up') }] : []),
    ...(visibility.down ? [{ value: 'down', label: t('project.list.actions.down') }] : []),
    ...(visibility.restart ? [{ value: 'restart', label: t('project.list.actions.restart') }] : []),
    { value: 'unregister', label: t('project.list.actions.unregister') },
  ];
}

function actionConfirmTitleKey(action: 'up' | 'down' | 'restart' | 'unregister') {
  return `project.list.actions.confirm${action.charAt(0).toUpperCase()}${action.slice(1)}Title`;
}

function actionConfirmDescriptionKey(action: 'up' | 'down' | 'restart' | 'unregister') {
  return `project.list.actions.confirm${action.charAt(0).toUpperCase()}${action.slice(1)}Description`;
}

function actionConfirmTheme(action: 'up' | 'down' | 'restart' | 'unregister') {
  return action === 'up' ? ('warning' as const) : ('danger' as const);
}

function confirmDangerousAction(row: ProjectListItem, action: 'up' | 'down' | 'restart' | 'unregister') {
  if (confirmDialogOpen.value) {
    return Promise.resolve(false);
  }

  return new Promise<boolean>((resolve) => {
    let settled = false;
    confirmDialogOpen.value = true;

    const finish = (dialog: DialogInstance, confirmed: boolean) => {
      if (settled) {
        return;
      }
      settled = true;
      confirmDialogOpen.value = false;
      dialog.destroy();
      resolve(confirmed);
    };

    const dialog = DialogPlugin.confirm({
      header: t(actionConfirmTitleKey(action)),
      body: () =>
        h('div', { class: 'project-action-confirm' }, [
          h('p', t(actionConfirmDescriptionKey(action), { name: row.display_name })),
        ]),
      theme: actionConfirmTheme(action),
      confirmBtn: {
        content: t('project.list.actions.confirm'),
        theme: actionConfirmTheme(action),
      },
      cancelBtn: t('project.list.actions.cancel'),
      onCancel: () => finish(dialog, false),
      onClose: () => finish(dialog, false),
      onConfirm: () => finish(dialog, true),
    });
  });
}

async function handleRowAction(action: string, row: ProjectListItem) {
  if (action === 'detail') {
    await navigateToDetail(row);
    return;
  }
  if (action === 'refresh') {
    await runAction(postProjectRefresh, row, t('project.list.actions.refreshSuccess'));
    return;
  }
  if (action === 'up') {
    if (!(await confirmDangerousAction(row, 'up'))) {
      return;
    }
    await runAction(postProjectUp, row, t('project.list.actions.actionSuccess'), 'up');
    return;
  }
  if (action === 'down') {
    if (!(await confirmDangerousAction(row, 'down'))) {
      return;
    }
    await runAction(postProjectDown, row, t('project.list.actions.actionSuccess'), 'down');
    return;
  }
  if (action === 'restart') {
    if (!(await confirmDangerousAction(row, 'restart'))) {
      return;
    }
    await runAction(postProjectRestart, row, t('project.list.actions.actionSuccess'), 'restart');
    return;
  }
  if (action === 'unregister') {
    if (!(await confirmDangerousAction(row, 'unregister'))) {
      return;
    }
    await runAction(postProjectUnregister, row, t('project.list.actions.actionSuccess'));
  }
}
</script>
<style scoped lang="less">
.project-page,
.project-table-head,
.project-resources,
.project-resource-badges,
.project-refresh,
.project-identity,
.project-header-summary {
  display: flex;
}

.project-page {
  flex-direction: column;
  gap: var(--graft-density-gap-16);
}

.project-table-head {
  align-items: center;
  gap: var(--graft-density-gap-16);
  justify-content: space-between;
}

.project-table-head__summary,
.project-table-head__hint {
  margin: 0;
}

.project-table-head__summary {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
}

.project-table-head__hint {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.project-header-summary {
  align-items: center;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-6) var(--graft-density-gap-10);
}

.project-header-summary__total,
.project-header-summary__status,
.project-runtime-badge,
.project-sync-status {
  align-items: center;
  display: inline-flex;
  font: var(--td-font-body-small);
  gap: var(--graft-density-gap-4);
  line-height: 1;
}

.project-runtime-badge--loading {
  color: var(--td-brand-color-6);
}

.project-runtime-badge__spinner {
  animation: project-runtime-spin 0.9s linear infinite;
  border: 2px solid color-mix(in srgb, currentcolor 18%, transparent);
  border-radius: 999px;
  border-right-color: currentcolor;
  box-sizing: border-box;
  display: inline-flex;
  flex: 0 0 auto;
  height: 12px;
  width: 12px;
}

.project-header-summary__total {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
}

.project-header-summary__status {
  color: var(--td-text-color-secondary);
  cursor: default;
}

.project-header-summary__status--running,
.project-runtime-badge--running {
  color: var(--td-success-color-6);
}

.project-header-summary__status--degraded,
.project-runtime-badge--degraded,
.project-sync-status--drifted {
  color: var(--td-warning-color-6);
}

.project-header-summary__status--stopped,
.project-runtime-badge--stopped {
  color: var(--td-text-color-secondary);
}

.project-header-summary__status--transitioning,
.project-runtime-badge--transitioning {
  color: var(--td-brand-color-6);
}

.project-header-summary__status--unknown,
.project-runtime-badge--unknown,
.project-sync-status--unknown {
  color: var(--td-text-color-placeholder);
}

.project-sync-status--clean {
  color: var(--td-success-color-6);
}

.project-table-host {
  max-width: 100%;
  min-width: 0;
  overflow-x: hidden;
  width: 100%;
}

.project-table-host[data-table-mode='scroll'] {
  overflow-x: auto;
}

.project-table-host :deep(.t-table__content) {
  min-width: 0;
}

.project-table-host :deep(.t-table__content table) {
  min-width: 100%;
}

.project-identity {
  flex-direction: column;
  gap: var(--graft-density-gap-6);
  min-width: 0;
}

.project-identity__main {
  align-items: flex-start;
  background: transparent;
  border: 0;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-2);
  min-width: 0;
  padding: 0;
  text-align: left;
}

.project-identity__main strong {
  color: var(--td-text-color-primary);
}

.project-identity__main span,
.project-refresh span,
.project-identity code {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.project-identity code {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-resources,
.project-resource-badges,
.project-refresh,
.project-column-drawer {
  gap: var(--graft-density-gap-8);
}

@keyframes project-runtime-spin {
  from {
    transform: rotate(0deg);
  }

  to {
    transform: rotate(360deg);
  }
}

.project-resources {
  align-items: center;
  flex-direction: column;
  gap: var(--graft-density-gap-6);
  justify-content: center;
  min-width: 0;
  text-align: center;
  width: 100%;
}

.project-resources__item {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
  justify-content: center;
}

.project-resources__label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.project-resource-badges {
  align-items: center;
  flex-wrap: wrap;
  justify-content: center;
}

.project-resource-badge {
  color: var(--td-text-color-secondary);
  column-gap: var(--graft-density-gap-4);
  display: inline-grid;
  font: var(--td-font-body-small);
  grid-auto-flow: column;
  place-items: center;
}

.project-resource-badge__icon {
  display: inline-flex;
  font-size: inherit;
  line-height: 1;
}

.project-resource-badge--running {
  color: var(--td-success-color-5);
}

.project-resource-badge--stopped {
  color: var(--td-text-color-secondary);
}

.project-resource-badge--transitioning {
  color: var(--td-brand-color-6);
}

.project-resource-badge--issue {
  color: var(--td-error-color-6);
}

.project-resource-badge--unknown {
  color: var(--td-text-color-placeholder);
}

.project-refresh {
  align-items: center;
  flex-direction: column;
  justify-content: center;
  text-align: center;
  width: 100%;
}

.project-empty {
  padding: var(--graft-density-gap-20) 0;
}

.project-action-confirm {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
}

.project-action-confirm p {
  margin: 0;
}

.project-column-drawer {
  display: flex;
  flex-direction: column;
}

@media (width <= 768px) {
  .project-table-head {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
