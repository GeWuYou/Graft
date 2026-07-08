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
              {{ t('project.list.projectCount', { count: summaryTotalCount }) }}
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
          <t-button theme="primary" @click="handleFilterQuery">{{ t('project.list.filters.query') }}</t-button>
          <t-button theme="default" variant="text" @click="resetFilters">{{
            t('project.list.filters.reset')
          }}</t-button>
        </template>
      </management-toolbar>

      <management-paged-table
        v-model:current="pagination.current"
        v-model:page-size="pagination.pageSize"
        :columns="visibleColumns"
        :empty-description="
          hasActiveFilters ? t('project.list.emptyFilteredDescription') : t('project.list.emptyDescription')
        "
        :empty-title="t('project.list.emptyTitle')"
        :footer-summary="paginationSummary"
        :loading="tableLoading"
        row-key="id"
        :rows="rows"
        :selected-row-keys="selectedRowKeys"
        :summary="t('project.list.tableSummary', { count: summaryTotalCount })"
        :total="pagination.total"
        @page-change="handlePageChange"
        @select-change="handleSelectChange"
      >
        <template #head>
          <div class="project-table-head">
            <div>
              <p class="project-table-head__summary" data-testid="project-table-summary">
                {{ t('project.list.tableSummary', { count: summaryTotalCount }) }}
              </p>
              <p class="project-table-head__hint">{{ t('project.list.tableHint') }}</p>
            </div>
          </div>
        </template>
        <template #toolbar>
          <table-view-toolbar
            :column-settings-label="t('project.list.columnSettings')"
            @column-settings="columnDrawerVisible = true"
          />
        </template>
        <template v-if="selectedRows.length > 0" #batch>
          <div class="project-batch-bar">
            <span data-testid="project-batch-selected">
              {{ t('project.list.batch.selected', { count: selectedRows.length }) }}
            </span>
            <div class="project-batch-bar__actions">
              <t-tooltip :content="batchActionHint('start')" placement="top">
                <t-button
                  data-testid="project-batch-start"
                  size="small"
                  theme="primary"
                  variant="outline"
                  :disabled="isBatchActionDisabled('start')"
                  :loading="batchActionLoading === 'start'"
                  @click="confirmBatchAction('start')"
                >
                  {{ t('project.list.batch.start') }}
                </t-button>
              </t-tooltip>
              <t-tooltip :content="batchActionHint('stop')" placement="top">
                <t-button
                  data-testid="project-batch-stop"
                  size="small"
                  theme="warning"
                  variant="outline"
                  :disabled="isBatchActionDisabled('stop')"
                  :loading="batchActionLoading === 'stop'"
                  @click="confirmBatchAction('stop')"
                >
                  {{ t('project.list.batch.stop') }}
                </t-button>
              </t-tooltip>
              <t-tooltip :content="batchActionHint('restart')" placement="top">
                <t-button
                  data-testid="project-batch-restart"
                  size="small"
                  theme="warning"
                  variant="outline"
                  :disabled="isBatchActionDisabled('restart')"
                  :loading="batchActionLoading === 'restart'"
                  @click="confirmBatchAction('restart')"
                >
                  {{ t('project.list.batch.restart') }}
                </t-button>
              </t-tooltip>
              <t-tooltip :content="batchActionHint('unregister')" placement="top">
                <t-button
                  data-testid="project-batch-unregister"
                  size="small"
                  theme="default"
                  variant="outline"
                  :disabled="isBatchActionDisabled('unregister')"
                  :loading="batchActionLoading === 'unregister'"
                  @click="confirmBatchAction('unregister')"
                >
                  {{ t('project.list.batch.unregister') }}
                </t-button>
              </t-tooltip>
              <t-tooltip :content="batchActionHint('redeploy')" placement="top">
                <t-button
                  data-testid="project-batch-redeploy"
                  size="small"
                  theme="default"
                  variant="outline"
                  :disabled="isBatchActionDisabled('redeploy')"
                  :loading="batchActionLoading === 'redeploy'"
                  @click="confirmBatchAction('redeploy')"
                >
                  {{ t('project.list.batch.redeploy') }}
                </t-button>
              </t-tooltip>
              <t-tooltip :content="batchActionHint('destroy')" placement="top">
                <t-button
                  data-testid="project-batch-destroy"
                  size="small"
                  theme="danger"
                  variant="outline"
                  :disabled="isBatchActionDisabled('destroy')"
                  :loading="batchActionLoading === 'destroy'"
                  @click="confirmBatchAction('destroy')"
                >
                  {{ t('project.list.batch.destroy') }}
                </t-button>
              </t-tooltip>
              <t-button
                data-testid="project-batch-clear"
                size="small"
                theme="default"
                variant="text"
                @click="clearSelection"
              >
                {{ t('project.list.batch.cancelSelection') }}
              </t-button>
            </div>
          </div>
        </template>
        <template #feedback>
          <management-empty-state
            v-if="errorMessage && !tableLoading && !refreshing"
            tone="error"
            :title="t('project.list.title')"
            :description="errorMessage"
          />
        </template>
        <template #name="{ row }">
          <div class="project-identity">
            <div class="project-identity__title-row">
              <button
                class="project-identity__main"
                type="button"
                @click="navigateToDetail(projectRow(row), 'overview')"
              >
                <strong>{{ projectRow(row).display_name }}</strong>
              </button>
              <t-tooltip
                v-if="projectRequiresLifecycleReview(projectRow(row))"
                :content="t('project.list.statusTooltip.lifecycleReviewRequired')"
                placement="top"
                theme="default"
              >
                <t-tag
                  size="small"
                  :theme="projectLifecycleReviewStatusTheme(projectRow(row).lifecycle_review_status)"
                  variant="light-outline"
                  data-testid="project-lifecycle-review-tag"
                  @click="navigateToDetail(projectRow(row), 'lifecycle')"
                >
                  {{ projectLifecycleReviewStatusLabel(t, projectRow(row).lifecycle_review_status) }}
                </t-tag>
              </t-tooltip>
            </div>
            <div class="project-identity__meta">
              <span v-if="projectSecondaryName(projectRow(row))">{{ projectSecondaryName(projectRow(row)) }}</span>
            </div>
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

        <template #operation="{ row }">
          <table-action-menu
            :actions="buildRowActions(projectRow(row))"
            :more-label="t('project.list.actions.operationMenu')"
            :more-label-fallback="t('project.list.actions.operationMenu')"
            @action="(action) => handleRowAction(action, projectRow(row))"
          />
        </template>

        <template #empty>
          <div v-if="!errorMessage" class="project-empty">
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
      </management-paged-table>

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
import type { DialogInstance, TableProps } from 'tdesign-vue-next';
import { DialogPlugin } from 'tdesign-vue-next/es/dialog';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, h, onActivated, onDeactivated, onMounted, onUnmounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import {
  ManagementEmptyState,
  ManagementPageContent,
  ManagementPagedTable,
  ManagementPageHeader,
  ManagementToolbar,
  TableActionMenu,
  TableViewToolbar,
} from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { useRealtimeSchedulerStore } from '@/store';
import { useTabsRouterStore } from '@/store/modules/tabs-router';
import { createLogger } from '@/utils/logger';
import { localizeRouteTitleKey } from '@/utils/route/title';

import {
  getProjects,
  postProjectBatchActions,
  postProjectDestroy,
  postProjectRedeploy,
  postProjectRestart,
  postProjectStop,
  postProjectUnregister,
  postProjectUp,
} from '../../api/project';
import ProjectListEntryActions from '../../components/ProjectListEntryActions.vue';
import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import {
  projectDriftStatusLabel,
  projectLifecycleActionVisibility,
  projectRuntimeStatusLabel,
  projectSourceKindLabel,
} from '../../shared/display';
import {
  projectLifecycleReviewStatusLabel,
  projectLifecycleReviewStatusTheme,
  projectRequiresLifecycleReview,
} from '../../shared/lifecycle';
import { acquireProjectListRealtime, releaseProjectListRealtime } from '../../shared/list-realtime';
import { appendResolvedTab, buildDetailTitleWithFallback } from '../../shared/navigation';
import type {
  ProjectBatchAction,
  ProjectBatchActionItem,
  ProjectBatchActionResponse,
  ProjectDestroyRequest,
  ProjectDetailResponse,
  ProjectDriftStatus,
  ProjectFilters,
  ProjectListItemWithLifecycle,
  ProjectRuntimeStatus,
  ProjectSourceKind,
} from '../../types/project';

defineOptions({
  name: 'ProjectListIndex',
});

const { t } = useI18n();
const router = useRouter();
const realtimeSchedulerStore = useRealtimeSchedulerStore();
const tabsRouterStore = useTabsRouterStore();
const logger = createLogger('project.list');

type HeaderStatusSummaryKey = 'running' | 'degraded' | 'stopped' | 'transitioning' | 'unknown';
type ProjectListDriftTone = 'clean' | 'drifted' | 'unknown';
type PendingProjectAction = 'up' | 'stop' | 'restart' | 'redeploy';
type ProjectResourceBadgeKey = 'running' | 'stopped' | 'transitioning' | 'issue' | 'unknown';
type ProjectBatchActionUi = ProjectBatchAction;
type PendingProjectActionState = {
  action: PendingProjectAction;
  awaitingVisibleChange: boolean;
  deadlineAt: number | null;
  runtimeStatus: ProjectRuntimeStatus | null;
};
type ProjectResourceBadge = {
  key: ProjectResourceBadgeKey;
  count: number;
  label: string;
  icon: string;
};

const tableLoading = ref(false);
const refreshing = ref(false);
const errorMessage = ref('');
const rows = ref<ProjectListItemWithLifecycle[]>([]);
const pagination = ref({
  current: 1,
  pageSize: 20,
  total: 0,
});
const filters = ref<ProjectFilters>({
  keyword: '',
  sourceKind: 'all',
  driftStatus: 'all',
});
const columnDrawerVisible = ref(false);

const sourceKindOptions: ProjectSourceKind[] = ['imported', 'managed', 'git', 'template'];
const driftStatusOptions: ProjectDriftStatus[] = ['unknown', 'clean', 'changed', 'missing'];

const configurableColumns = computed<TableProps['columns']>(() => [
  {
    colKey: 'row-select',
    title: t('project.list.columns.selection'),
    type: 'multiple',
    width: 48,
    fixed: 'left',
    align: 'center',
  },
  { colKey: 'name', title: t('project.list.columns.name'), width: 300 },
  { colKey: 'source', title: t('project.list.columns.source'), width: 112, align: 'center' },
  { colKey: 'runtime', title: t('project.list.columns.runtime'), width: 148, align: 'center' },
  { colKey: 'resources', title: t('project.list.columns.resources'), width: 236, align: 'center' },
  { colKey: 'drift', title: t('project.list.columns.drift'), width: 124, align: 'center' },
  { colKey: 'operation', title: t('project.list.columns.operation'), width: 152, fixed: 'right', align: 'center' },
]);
const visibleColumnKeys = ref(['row-select', 'name', 'source', 'runtime', 'resources', 'drift', 'operation']);
const visibleColumns = computed(() =>
  (configurableColumns.value ?? []).filter((column) => visibleColumnKeys.value.includes(String(column?.colKey))),
);
const confirmDialogOpen = ref(false);
const realtimeActive = ref(false);
const pendingRowActions = ref<Record<number, PendingProjectActionState>>({});
const selectedRowKeys = ref<number[]>([]);
const batchActionLoading = ref<ProjectBatchActionUi | ''>('');

const summaryTotalCount = computed(() => (pagination.value.total > 0 ? pagination.value.total : rows.value.length));
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
    Boolean(filters.value.keyword.trim()) || filters.value.sourceKind !== 'all' || filters.value.driftStatus !== 'all',
);
const paginationSummary = computed(() => {
  if (!pagination.value.total || rows.value.length === 0) {
    return t('project.list.tableSummary', { count: rows.value.length });
  }
  const start = (pagination.value.current - 1) * pagination.value.pageSize + 1;
  const end = start + rows.value.length - 1;
  return `${start}-${end} / ${pagination.value.total}`;
});
const selectedRows = computed(() => {
  const rowMap = new Map(rows.value.map((row) => [row.id, row]));
  return selectedRowKeys.value
    .map((id) => rowMap.get(id))
    .filter((row): row is ProjectListItemWithLifecycle => Boolean(row));
});

onMounted(() => {
  realtimeActive.value = true;
  syncProjectListRealtimeSubscription();
  void fetchProjects();
});

onUnmounted(() => {
  realtimeActive.value = false;
  syncProjectListRealtimeSubscription();
  clearPendingRowActionTimeouts();
});

onActivated(() => {
  realtimeActive.value = true;
  syncProjectListRealtimeSubscription();
  void fetchProjects();
});

onDeactivated(() => {
  realtimeActive.value = false;
  syncProjectListRealtimeSubscription();
});

watch(
  () => realtimeSchedulerStore.allowPolling,
  () => {
    syncProjectListRealtimeSubscription();
  },
);

function projectRow(row: unknown) {
  return row as ProjectListItemWithLifecycle;
}

function sourceKindLabel(value: ProjectSourceKind) {
  return projectSourceKindLabel(t, value);
}

function driftStatusLabel(value: ProjectDriftStatus) {
  return projectDriftStatusLabel(t, value);
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

function projectContainerBadges(row: ProjectListItemWithLifecycle): ProjectResourceBadge[] {
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
  const fallbackKey: ProjectResourceBadgeKey =
    row.runtime_status === 'running'
      ? 'running'
      : row.runtime_status === 'stopped'
        ? 'stopped'
        : row.runtime_status === 'transitioning'
          ? 'transitioning'
          : row.runtime_status === 'degraded'
            ? 'issue'
            : 'unknown';
  return [
    {
      key: fallbackKey,
      count: 0,
      label: projectResourceBadgeLabel(fallbackKey, 0),
      icon: projectResourceBadgeIcon(fallbackKey),
    },
  ];
}

function projectSecondaryName(row: ProjectListItemWithLifecycle) {
  const canonicalName = row.canonical_project_name?.trim() || '';
  const displayName = row.display_name?.trim() || '';

  if (!canonicalName || canonicalName === displayName) {
    return '';
  }

  return t('project.list.canonicalNameValue', { name: canonicalName });
}

let refreshRequestSeq = 0;
const pendingRowTimeouts = new Map<number, number>();

function syncProjectListRealtimeSubscription() {
  if (realtimeActive.value && realtimeSchedulerStore.allowPolling) {
    acquireProjectListRealtime(handleProjectListRealtimeItems);
    return;
  }
  releaseProjectListRealtime(handleProjectListRealtimeItems);
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
    selectedRowKeys.value = selectedRowKeys.value.filter((id) => nextRows.some((row) => row.id === id));
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

function handleProjectListRealtimeItems(
  items: Array<{
    project_id: number;
    runtime_status: ProjectRuntimeStatus;
    service_count: number;
    container_counts: ProjectListItemWithLifecycle['container_counts'];
    drift_status: ProjectDriftStatus;
  }>,
) {
  if (!realtimeActive.value || !realtimeSchedulerStore.allowPolling || rows.value.length === 0) {
    return;
  }
  const patchById = new Map(items.map((item) => [item.project_id, item]));
  let changed = false;
  const nextRows = rows.value.map((row) => {
    const patch = patchById.get(row.id);
    if (!patch) {
      return row;
    }
    const nextRow: ProjectListItemWithLifecycle = {
      ...row,
      runtime_status: patch.runtime_status,
      service_count: patch.service_count,
      container_counts: patch.container_counts,
      drift_status: patch.drift_status,
    };
    if (
      nextRow.runtime_status !== row.runtime_status ||
      nextRow.service_count !== row.service_count ||
      nextRow.drift_status !== row.drift_status ||
      JSON.stringify(nextRow.container_counts) !== JSON.stringify(row.container_counts)
    ) {
      changed = true;
      return nextRow;
    }
    return row;
  });
  if (!changed) {
    return;
  }
  rows.value = nextRows;
  reconcilePendingRowActions(nextRows);
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

function reconcilePendingRowActions(nextRows: ProjectListItemWithLifecycle[]) {
  const nextPending = { ...pendingRowActions.value };
  const rowMap = new Map(nextRows.map((row) => [row.id, row]));

  for (const [rawId, pending] of Object.entries(nextPending)) {
    const id = Number(rawId);
    const row = rowMap.get(id);
    if (!row) {
      clearPendingRowActionTimeout(id);
      delete nextPending[id];
      continue;
    }

    const runtimeChanged = (row.runtime_status ?? null) !== pending.runtimeStatus;
    if (pending.awaitingVisibleChange && runtimeChanged) {
      clearPendingRowActionTimeout(id);
      delete nextPending[id];
    }
  }

  pendingRowActions.value = nextPending;
}

function markPendingRowAction(row: ProjectListItemWithLifecycle, action: PendingProjectAction) {
  clearPendingRowActionTimeout(row.id);
  pendingRowActions.value = {
    ...pendingRowActions.value,
    [row.id]: {
      action,
      awaitingVisibleChange: false,
      deadlineAt: null,
      runtimeStatus: row.runtime_status ?? null,
    },
  };
}

function markPendingRowActions(rowsToMark: ProjectListItemWithLifecycle[], action: PendingProjectAction) {
  if (rowsToMark.length === 0) {
    return;
  }

  const nextPending = { ...pendingRowActions.value };
  for (const row of rowsToMark) {
    clearPendingRowActionTimeout(row.id);
    nextPending[row.id] = {
      action,
      awaitingVisibleChange: false,
      deadlineAt: null,
      runtimeStatus: row.runtime_status ?? null,
    };
  }
  pendingRowActions.value = nextPending;
}

function markPendingRowActionAwaitingChange(rowId: number) {
  const pending = pendingRowActions.value[rowId];
  if (!pending) {
    return;
  }

  const deadlineAt = Date.now() + 15_000;
  pendingRowActions.value = {
    ...pendingRowActions.value,
    [rowId]: {
      ...pending,
      awaitingVisibleChange: true,
      deadlineAt,
    },
  };
  schedulePendingRowActionTimeout(rowId, deadlineAt);
}

function markPendingRowActionsAwaitingChange(rowIds: number[]) {
  if (rowIds.length === 0) {
    return;
  }

  const deadlineAt = Date.now() + 15_000;
  const nextPending = { ...pendingRowActions.value };
  let changed = false;

  for (const rowId of rowIds) {
    const pending = nextPending[rowId];
    if (!pending) {
      continue;
    }
    nextPending[rowId] = {
      ...pending,
      awaitingVisibleChange: true,
      deadlineAt,
    };
    schedulePendingRowActionTimeout(rowId, deadlineAt);
    changed = true;
  }

  if (changed) {
    pendingRowActions.value = nextPending;
  }
}

function clearPendingRowAction(rowId: number) {
  if (!pendingRowActions.value[rowId]) {
    return;
  }
  clearPendingRowActionTimeout(rowId);
  const nextPending = { ...pendingRowActions.value };
  delete nextPending[rowId];
  pendingRowActions.value = nextPending;
}

function clearPendingRowActions(rowIds: number[]) {
  if (rowIds.length === 0) {
    return;
  }

  const nextPending = { ...pendingRowActions.value };
  let changed = false;

  for (const rowId of rowIds) {
    if (!nextPending[rowId]) {
      continue;
    }
    clearPendingRowActionTimeout(rowId);
    delete nextPending[rowId];
    changed = true;
  }

  if (changed) {
    pendingRowActions.value = nextPending;
  }
}

function isRowActionPending(rowId: number) {
  return Boolean(pendingRowActions.value[rowId]);
}

function schedulePendingRowActionTimeout(rowId: number, deadlineAt: number) {
  clearPendingRowActionTimeout(rowId);
  if (typeof window === 'undefined') {
    return;
  }
  const delay = Math.max(0, deadlineAt - Date.now());
  const timeoutId = window.setTimeout(() => {
    clearPendingRowAction(rowId);
  }, delay);
  pendingRowTimeouts.set(rowId, timeoutId);
}

function clearPendingRowActionTimeout(rowId: number) {
  if (typeof window === 'undefined') {
    pendingRowTimeouts.delete(rowId);
    return;
  }
  const timeoutId = pendingRowTimeouts.get(rowId);
  if (timeoutId === undefined) {
    return;
  }
  window.clearTimeout(timeoutId);
  pendingRowTimeouts.delete(rowId);
}

function clearPendingRowActionTimeouts() {
  if (typeof window === 'undefined') {
    pendingRowTimeouts.clear();
    return;
  }
  for (const timeoutId of pendingRowTimeouts.values()) {
    window.clearTimeout(timeoutId);
  }
  pendingRowTimeouts.clear();
}

function clearSelection() {
  selectedRowKeys.value = [];
}

function handleSelectChange(rowKeys: Array<string | number>) {
  selectedRowKeys.value = rowKeys.map((key) => Number(key)).filter((key) => Number.isInteger(key) && key > 0);
}

function resetFilters() {
  filters.value = {
    keyword: '',
    sourceKind: 'all',
    driftStatus: 'all',
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

function navigateToDetail(row: ProjectListItemWithLifecycle, tab?: string) {
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
  row: ProjectListItemWithLifecycle,
  successMessage: string,
  pendingAction?: PendingProjectAction,
) {
  if (pendingAction) {
    markPendingRowAction(row, pendingAction);
  }
  try {
    await handler(row.id);
    if (pendingAction) {
      markPendingRowActionAwaitingChange(row.id);
    }
    MessagePlugin.success(successMessage);
    await fetchProjects();
  } catch (error) {
    clearPendingRowAction(row.id);
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.list.actions.actionFailed')));
  }
}

function buildRowActions(row: ProjectListItemWithLifecycle) {
  const visibility = projectLifecycleActionVisibility(row.runtime_status, {
    hideLifecycleActions: isRowActionPending(row.id),
  });
  const lifecycleBlocked = projectRequiresLifecycleReview(row);

  return [
    { value: 'detail', label: t('project.list.actions.detail') },
    ...(!lifecycleBlocked && visibility.up ? [{ value: 'up', label: t('project.list.actions.up') }] : []),
    ...(!lifecycleBlocked && visibility.stop ? [{ value: 'stop', label: t('project.list.actions.stop') }] : []),
    ...(!lifecycleBlocked && visibility.restart
      ? [{ value: 'restart', label: t('project.list.actions.restart') }]
      : []),
    ...(!lifecycleBlocked && visibility.redeploy
      ? [{ value: 'redeploy', label: t('project.list.actions.redeploy') }]
      : []),
    { value: 'unregister', label: t('project.list.actions.unregister') },
    { value: 'destroy', label: t('project.list.actions.destroy') },
  ];
}

function actionConfirmTitleKey(action: 'up' | 'stop' | 'restart' | 'unregister' | 'redeploy' | 'destroy') {
  return `project.list.actions.confirm${action.charAt(0).toUpperCase()}${action.slice(1)}Title`;
}

function actionConfirmDescriptionKey(action: 'up' | 'stop' | 'restart' | 'unregister' | 'redeploy' | 'destroy') {
  return `project.list.actions.confirm${action.charAt(0).toUpperCase()}${action.slice(1)}Description`;
}

function actionConfirmTheme(action: 'up' | 'stop' | 'restart' | 'unregister' | 'redeploy' | 'destroy') {
  return action === 'up' ? ('warning' as const) : ('danger' as const);
}

function isDeleteWorkingDirectoryAllowed(row: ProjectListItemWithLifecycle) {
  return row.ownership_mode !== 'external';
}

function isRowBatchEligible(row: ProjectListItemWithLifecycle, action: ProjectBatchActionUi) {
  if (projectRequiresLifecycleReview(row) && ['start', 'stop', 'restart', 'redeploy'].includes(action)) {
    return false;
  }
  const visibility = projectLifecycleActionVisibility(row.runtime_status, {
    hideLifecycleActions: isRowActionPending(row.id),
  });
  if (action === 'start') return visibility.up;
  if (action === 'stop') return visibility.stop;
  if (action === 'restart') return visibility.restart;
  if (action === 'unregister') return true;
  if (action === 'redeploy') return !isRowActionPending(row.id);
  return true;
}

function batchActionableRows(action: ProjectBatchActionUi) {
  return selectedRows.value.filter((row) => isRowBatchEligible(row, action));
}

function requiresSingleSelection(action: ProjectBatchActionUi) {
  return action === 'destroy';
}

function isBatchActionDisabled(action: ProjectBatchActionUi) {
  if (requiresSingleSelection(action) && selectedRows.value.length !== 1) {
    return true;
  }
  return batchActionableRows(action).length === 0;
}

function batchActionHint(action: ProjectBatchActionUi) {
  if (!selectedRows.value.length) return t('project.list.batch.noSelection');
  if (requiresSingleSelection(action) && selectedRows.value.length !== 1) {
    return t('project.list.batch.destroySingleSelection');
  }
  if (isBatchActionDisabled(action)) return t('project.list.batch.noActionableSelection');
  return t(`project.list.batch.${action}Hint`, {
    count: batchActionableRows(action).length,
  });
}

function confirmDangerousAction(
  row: ProjectListItemWithLifecycle,
  action: 'up' | 'stop' | 'restart' | 'unregister' | 'redeploy' | 'destroy',
) {
  if (confirmDialogOpen.value) {
    return Promise.resolve(false);
  }

  return new Promise<boolean>((resolve) => {
    let settled = false;
    confirmDialogOpen.value = true;
    const deleteWorkingDirectory = ref(false);
    const autoUnregister = ref(false);
    const removeNamedVolumes = ref(false);

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
          action === 'destroy'
            ? h('div', { class: 'project-action-confirm__danger' }, [
                h('label', { class: 'project-action-confirm__option' }, [
                  h('input', {
                    checked: removeNamedVolumes.value,
                    type: 'checkbox',
                    onInput: (event: Event) => {
                      removeNamedVolumes.value = (event.target as HTMLInputElement).checked;
                    },
                  }),
                  h('span', t('project.list.actions.destroyDeleteVolumes')),
                ]),
                h('label', { class: 'project-action-confirm__option' }, [
                  h('input', {
                    checked: autoUnregister.value,
                    type: 'checkbox',
                    onInput: (event: Event) => {
                      autoUnregister.value = (event.target as HTMLInputElement).checked;
                    },
                  }),
                  h('span', t('project.list.actions.destroyAutoUnregister')),
                ]),
                h('label', { class: 'project-action-confirm__option' }, [
                  h('input', {
                    checked: deleteWorkingDirectory.value,
                    disabled: !isDeleteWorkingDirectoryAllowed(row),
                    type: 'checkbox',
                    onInput: (event: Event) => {
                      deleteWorkingDirectory.value = (event.target as HTMLInputElement).checked;
                      if (deleteWorkingDirectory.value) {
                        autoUnregister.value = true;
                      }
                    },
                  }),
                  h('span', t('project.list.actions.destroyDeleteProjectFiles')),
                ]),
              ])
            : null,
        ]),
      theme: actionConfirmTheme(action),
      confirmBtn: {
        content: t('project.list.actions.confirm'),
        theme: actionConfirmTheme(action),
      },
      cancelBtn: t('project.list.actions.cancel'),
      onCancel: () => finish(dialog, false),
      onClose: () => finish(dialog, false),
      onConfirm: async () => {
        if (action === 'destroy') {
          await runDestroy(row, {
            auto_unregister: autoUnregister.value || deleteWorkingDirectory.value,
            confirm_canonical_project_name: row.canonical_project_name,
            delete_working_directory: deleteWorkingDirectory.value,
            image_prune: false,
            remove_named_volumes: removeNamedVolumes.value,
          });
          finish(dialog, true);
          return;
        }
        finish(dialog, true);
      },
    });
  });
}

async function runDestroy(row: ProjectListItemWithLifecycle, payload: ProjectDestroyRequest) {
  try {
    await postProjectDestroy(row.id, payload);
    MessagePlugin.success(t('project.list.actions.actionSuccess'));
    await fetchProjects();
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.list.actions.actionFailed')));
  }
}

async function runRedeploy(row: ProjectListItemWithLifecycle) {
  markPendingRowAction(row, 'redeploy');
  try {
    await postProjectRedeploy(row.id);
    markPendingRowActionAwaitingChange(row.id);
    MessagePlugin.success(t('project.list.actions.actionSuccess'));
    await fetchProjects();
  } catch (error) {
    clearPendingRowAction(row.id);
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.list.actions.actionFailed')));
  }
}

async function executeBatchAction(action: ProjectBatchActionUi, overrides: Partial<ProjectDestroyRequest> = {}) {
  const actionableRows = batchActionableRows(action);
  if (requiresSingleSelection(action) && actionableRows.length !== 1) {
    return;
  }
  if (!actionableRows.length) return;
  batchActionLoading.value = action;
  const pendingAction =
    action === 'start'
      ? ('up' as const)
      : action === 'stop' || action === 'restart' || action === 'redeploy'
        ? action
        : null;
  if (pendingAction) {
    markPendingRowActions(actionableRows, pendingAction);
  }
  try {
    const response = await postProjectBatchActions({
      action,
      auto_unregister: overrides.auto_unregister ?? false,
      confirm_canonical_project_name: overrides.confirm_canonical_project_name,
      delete_working_directory: overrides.delete_working_directory ?? false,
      image_prune: overrides.image_prune ?? false,
      project_ids: actionableRows.map((row) => row.id),
      remove_named_volumes: overrides.remove_named_volumes ?? false,
    });
    if (pendingAction) {
      const completedRowIds = response.items
        .filter((item) => !item.skipped && item.result === 'completed')
        .map((item) => item.project_id);
      const blockedRowIds = response.items
        .filter((item) => item.skipped || item.result !== 'completed')
        .map((item) => item.project_id);
      markPendingRowActionsAwaitingChange(completedRowIds);
      clearPendingRowActions(blockedRowIds);
    }
    handleBatchActionResult(action, response);
    clearSelection();
    await fetchProjects();
  } catch (error) {
    if (pendingAction) {
      clearPendingRowActions(actionableRows.map((row) => row.id));
    }
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.list.batch.failed')));
  } finally {
    batchActionLoading.value = '';
  }
}

function batchFailureSummary(items: ProjectBatchActionItem[]) {
  return items
    .filter((item) => !item.skipped && item.result !== 'completed')
    .map((item) => `${item.project_id}: ${item.message_key ? t(item.message_key) : item.message || '-'}`)
    .join('\n');
}

function batchActionLocaleSegment(action: ProjectBatchActionUi) {
  return action;
}

function handleBatchActionResult(action: ProjectBatchActionUi, response: ProjectBatchActionResponse) {
  const successCount = response.completed_count;
  const skippedCount = response.skipped_count;
  const title = t(`project.list.batch.${batchActionLocaleSegment(action)}ResultTitle`);
  const dialogTheme = 'warning' as const;
  if (response.blocked_count === 0) {
    MessagePlugin.success(
      t('project.list.batch.success', {
        count: successCount,
        skippedCount,
      }),
    );
    return;
  }
  MessagePlugin.error(
    t('project.list.batch.partial', {
      count: successCount,
      skippedCount,
      blockedCount: response.blocked_count,
    }),
  );
  DialogPlugin.alert({
    body: () => h('div', { style: { whiteSpace: 'pre-line' } }, batchFailureSummary(response.items)),
    confirmBtn: t('project.list.actions.confirm'),
    header: title,
    theme: dialogTheme,
  });
}

function confirmBatchAction(action: ProjectBatchActionUi) {
  if (isBatchActionDisabled(action) || confirmDialogOpen.value) {
    return;
  }
  confirmDialogOpen.value = true;
  const deleteWorkingDirectory = ref(false);
  const autoUnregister = ref(false);
  const removeNamedVolumes = ref(false);
  const selectedCount = selectedRows.value.length;
  const actionableCount = batchActionableRows(action).length;
  const skippedCount = selectedCount - actionableCount;
  const title = t(`project.list.batch.${batchActionLocaleSegment(action)}Title`);
  const closeDialog = () => {
    confirmDialogOpen.value = false;
    dialog.destroy();
  };
  const dialog = DialogPlugin.confirm({
    header: title,
    body: () =>
      h('div', { class: 'project-action-confirm' }, [
        h(
          'p',
          t(`project.list.batch.${batchActionLocaleSegment(action)}Confirm`, {
            count: actionableCount,
          }),
        ),
        h('p', t('project.list.batch.scope', { actionableCount, selectedCount, skippedCount })),
        skippedCount > 0 ? h('p', t('project.list.batch.skipInapplicable')) : null,
        action === 'destroy'
          ? h('div', { class: 'project-action-confirm__danger' }, [
              h('label', { class: 'project-action-confirm__option' }, [
                h('input', {
                  checked: removeNamedVolumes.value,
                  type: 'checkbox',
                  onInput: (event: Event) => {
                    removeNamedVolumes.value = (event.target as HTMLInputElement).checked;
                  },
                }),
                h('span', t('project.list.actions.destroyDeleteVolumes')),
              ]),
              h('label', { class: 'project-action-confirm__option' }, [
                h('input', {
                  checked: autoUnregister.value,
                  type: 'checkbox',
                  onInput: (event: Event) => {
                    autoUnregister.value = (event.target as HTMLInputElement).checked;
                  },
                }),
                h('span', t('project.list.actions.destroyAutoUnregister')),
              ]),
              h('label', { class: 'project-action-confirm__option' }, [
                h('input', {
                  checked: deleteWorkingDirectory.value,
                  disabled: selectedRows.value.some((row) => !isDeleteWorkingDirectoryAllowed(row)),
                  type: 'checkbox',
                  onInput: (event: Event) => {
                    deleteWorkingDirectory.value = (event.target as HTMLInputElement).checked;
                    if (deleteWorkingDirectory.value) {
                      autoUnregister.value = true;
                    }
                  },
                }),
                h('span', t('project.list.actions.destroyDeleteProjectFiles')),
              ]),
            ])
          : null,
      ]),
    cancelBtn: t('project.list.actions.cancel'),
    confirmBtn: t('project.list.actions.confirm'),
    onCancel: () => {
      closeDialog();
    },
    onClose: () => {
      closeDialog();
    },
    onConfirm: async () => {
      closeDialog();
      await executeBatchAction(action, {
        auto_unregister: autoUnregister.value || deleteWorkingDirectory.value,
        confirm_canonical_project_name:
          action === 'destroy' && selectedRows.value.length === 1
            ? selectedRows.value[0]?.canonical_project_name
            : undefined,
        delete_working_directory: deleteWorkingDirectory.value,
        image_prune: false,
        remove_named_volumes: removeNamedVolumes.value,
      });
    },
  });
}

async function handleRowAction(action: string, row: ProjectListItemWithLifecycle) {
  if (action === 'detail') {
    await navigateToDetail(row);
    return;
  }
  if (action === 'up') {
    if (!(await confirmDangerousAction(row, 'up'))) {
      return;
    }
    await runAction(postProjectUp, row, t('project.list.actions.actionSuccess'), 'up');
    return;
  }
  if (action === 'stop') {
    if (!(await confirmDangerousAction(row, 'stop'))) {
      return;
    }
    await runAction(postProjectStop, row, t('project.list.actions.actionSuccess'), 'stop');
    return;
  }
  if (action === 'restart') {
    if (!(await confirmDangerousAction(row, 'restart'))) {
      return;
    }
    await runAction(postProjectRestart, row, t('project.list.actions.actionSuccess'), 'restart');
    return;
  }
  if (action === 'redeploy') {
    if (!(await confirmDangerousAction(row, 'redeploy'))) {
      return;
    }
    await runRedeploy(row);
    return;
  }
  if (action === 'unregister') {
    if (!(await confirmDangerousAction(row, 'unregister'))) {
      return;
    }
    await runAction(postProjectUnregister, row, t('project.list.actions.actionSuccess'));
    return;
  }
  if (action === 'destroy') {
    if (!(await confirmDangerousAction(row, 'destroy'))) {
      return;
    }
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
.project-header-summary,
.project-batch-bar,
.project-batch-bar__actions {
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

.project-batch-bar {
  align-items: center;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  padding: var(--graft-density-gap-8) 0;
  width: 100%;
}

.project-batch-bar__actions {
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
}

.project-batch-bar > span {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
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

.project-identity__title-row {
  align-items: center;
  display: inline-flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-6);
  min-width: 0;
}

.project-identity__meta {
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

.project-identity__title-row :deep(.t-tag) {
  cursor: pointer;
}

.project-identity__meta span,
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
.project-column-drawer,
.project-action-confirm__danger {
  gap: var(--graft-density-gap-8);
}

.project-action-confirm__danger {
  display: flex;
  flex-direction: column;
  margin-top: var(--graft-density-gap-8);
}

.project-action-confirm__option {
  align-items: center;
  display: flex;
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
