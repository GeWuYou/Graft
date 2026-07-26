<template>
  <div ref="projectPageRef" class="project-page" data-page-type="list-form-detail">
    <management-page-content>
      <management-page-header
        :compact="usesCompactApplicationList"
        title-key="project.list.title"
        description-key="project.list.description"
        :source="{ labelKey: 'project.list.eyebrow', fallback: t('project.list.eyebrow') }"
      >
        <template #actions>
          <t-space size="small" break-line>
            <project-list-entry-actions
              :create-label="t('project.list.actions.create')"
              :reset-label="t('project.list.clearFilters')"
              :show-reset="hasActiveFilters"
              @create="navigateToSourceChooser"
              @reset="resetFilters"
            />
          </t-space>
        </template>
      </management-page-header>

      <management-statistics-bar
        :items="[
          { label: t('project.list.projectCount', { count: '' }).trim(), value: summaryTotalCount },
          ...headerStatusSummaryItems.map((item) => ({
            label: runtimeStatusLabel(item.key),
            value: item.count,
          })),
        ]"
      />

      <advanced-query-filter-builder
        active-preset="all"
        :add-filter-label="`+ ${t('project.list.filters.query')}`"
        :add-sorter-label="t('project.list.filters.sortAdd')"
        :builder-hint="t('project.list.description')"
        :builder-title="t('project.list.filters.query')"
        :compact-mode="usesCompactApplicationList"
        :compact-toggle-label="t('project.list.filters.filter')"
        :field-values="projectFilterFieldValues"
        :fields="projectFilterDefinitions"
        :filters-group-label="t('project.list.filters.query')"
        :keyword="filters.keyword"
        :keyword-placeholder="t('project.list.filters.searchPlaceholder')"
        :loading="tableLoading"
        :move-down-label="t('project.list.filters.sortMoveDown')"
        :move-up-label="t('project.list.filters.sortMoveUp')"
        :preset-label="t('project.list.filters.query')"
        :presets="[]"
        :remove-sorter-label="t('project.list.filters.sortRemove')"
        :reset-label="t('project.list.filters.reset')"
        :search-label="t('project.list.filters.query')"
        :selected-field-key="selectedApplicationFilterField"
        :show-sorter-builder="true"
        :sort-add-disabled="sortAddDisabled"
        :sort-direction-options="projectSortDirectionOptions"
        :sort-direction-placeholder="t('project.list.filters.sortDirectionPlaceholder')"
        sort-field-key="sorterBuilder"
        :sort-field-options-by-index="sortFieldOptionsByIndex"
        :sort-field-placeholder="t('project.list.filters.sortFieldPlaceholder')"
        :sort-move-down-disabled="sortMoveDownDisabled"
        :sort-move-up-disabled="sortMoveUpDisabled"
        :sorters="normalizedSorters"
        :tags="projectFilterTags"
        time-field-key="timeRange"
        :time-fields="[]"
        @reset="resetFilters"
        @search="handleFilterQuery"
        @add-sorter="addApplicationSorter"
        @move-sorter-down="moveApplicationSorterDown"
        @move-sorter-up="moveApplicationSorterUp"
        @remove-sorter="removeApplicationSorter"
        @update:field="updateApplicationFilterField"
        @update:keyword="filters.keyword = $event"
        @update:selected-field-key="updateSelectedApplicationFilterField"
        @update:sort-direction="updateApplicationSortDirection"
        @update:sort-field="updateApplicationSortField"
      >
        <template #saved-query-views>
          <saved-query-view-control :controller="projectSavedViews" />
        </template>
      </advanced-query-filter-builder>

      <management-paged-table
        v-model:current="pagination.current"
        v-model:page-size="pagination.pageSize"
        :columns="visibleColumns"
        :cards-visible="usesCompactApplicationList"
        :empty-description="
          hasActiveFilters ? t('project.list.emptyFilteredDescription') : t('project.list.emptyDescription')
        "
        :empty-title="t('project.list.emptyTitle')"
        :footer-summary="paginationSummary"
        :loading="tableLoading"
        row-key="application_id"
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
              <p v-if="!usesCompactApplicationList" class="project-table-head__hint">
                {{ t('project.list.tableHint') }}
              </p>
            </div>
          </div>
        </template>
        <template v-if="!usesCompactApplicationList" #toolbar>
          <div class="project-list-toolbar">
            <table-view-toolbar
              :column-settings-label="t('project.list.columnSettings')"
              @column-settings="columnDrawerVisible = true"
            />
          </div>
        </template>
        <template v-if="!usesCompactApplicationList && selectedRows.length > 0" #batch>
          <management-batch-bar
            :selected-label="t('project.list.batch.selected', { count: selectedRows.length })"
            :clear-label="t('project.list.batch.cancelSelection')"
            clear-test-id="project-batch-clear"
            @clear="clearSelection"
          >
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
          </management-batch-bar>
        </template>
        <template #feedback>
          <management-empty-state
            v-if="errorMessage && !tableLoading && !refreshing"
            tone="error"
            :title="t('project.list.title')"
            :description="errorMessage"
          />
        </template>
        <template #cards>
          <responsive-card-list class="project-mobile-list">
            <article
              v-for="row in rows"
              :key="row.application_id"
              class="project-mobile-card"
              role="button"
              tabindex="0"
              :data-testid="`project-mobile-card-${row.application_id}`"
              @click="navigateToDetail(row)"
              @keydown.enter.prevent="navigateToDetail(row)"
              @keydown.space.prevent="navigateToDetail(row)"
            >
              <div class="project-mobile-card__main">
                <strong class="project-mobile-card__title">{{ row.display_name }}</strong>
                <div class="project-mobile-card__meta">
                  <t-tag theme="primary" variant="light-outline" size="small">
                    {{ deploymentAdapterLabel(row.deployment_adapter_kind) }}
                  </t-tag>
                  <button
                    type="button"
                    :class="[
                      'project-runtime-badge',
                      `project-runtime-badge--${normalizeRuntimeStatus(row.runtime_status)}`,
                      { 'project-runtime-badge--loading': isRowActionPending(row.application_id) },
                    ]"
                    :aria-busy="isRowActionPending(row.application_id)"
                    :aria-label="runtimeStatusActionTooltip(row)"
                    :disabled="openingTaskRowIds.has(row.application_id)"
                    @click.stop="openApplicationTask(row)"
                  >
                    <span v-if="isRowActionPending(row.application_id)" class="project-runtime-badge__spinner" />
                    <template v-else>{{ runtimeStatusLabel(row.runtime_status) }}</template>
                  </button>
                </div>
              </div>
              <table-action-menu
                class="project-mobile-card__actions"
                :actions="buildMobileRowActions(row)"
                :more-label="t('project.list.actions.operationMenu')"
                :more-label-fallback="t('project.list.actions.operationMenu')"
                @action="(action) => handleRowAction(action, row)"
              />
            </article>
          </responsive-card-list>
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
            <code>{{ projectRow(row).workspace_path }}</code>
          </div>
        </template>

        <template #source="{ row }">
          <t-tag theme="default" variant="light-outline">
            {{ sourceTypeLabel(projectRow(row).source_type) }}
          </t-tag>
        </template>

        <template #deploymentAdapterKind="{ row }">
          <t-tag theme="primary" variant="light-outline">
            {{ deploymentAdapterLabel(projectRow(row).deployment_adapter_kind) }}
          </t-tag>
        </template>

        <template #runtimeTarget="{ row }">
          <span>{{ projectRow(row).runtime_target?.display_name || t('project.list.runtimeTargetUnavailable') }}</span>
        </template>

        <template #provider="{ row }">
          <span>{{ projectProviderLabel(projectRow(row)) }}</span>
        </template>

        <template #runtime="{ row }">
          <t-tooltip :content="runtimeStatusActionTooltip(projectRow(row))" placement="top" theme="default">
            <button
              type="button"
              :class="[
                'project-runtime-badge',
                `project-runtime-badge--${normalizeRuntimeStatus(projectRow(row).runtime_status)}`,
                { 'project-runtime-badge--loading': isRowActionPending(projectRow(row).application_id) },
              ]"
              :aria-busy="isRowActionPending(projectRow(row).application_id)"
              :aria-label="runtimeStatusActionTooltip(projectRow(row))"
              :disabled="openingTaskRowIds.has(projectRow(row).application_id)"
              :data-testid="`project-runtime-status-${projectRow(row).application_id}`"
              @click="openApplicationTask(projectRow(row))"
            >
              <span
                v-if="isRowActionPending(projectRow(row).application_id)"
                class="project-runtime-badge__spinner"
                :data-testid="`project-runtime-status-loading-${projectRow(row).application_id}`"
              />
              <template v-else>
                {{ runtimeStatusLabel(projectRow(row).runtime_status) }}
              </template>
            </button>
          </t-tooltip>
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
                  :class="['application-resource-badge', `project-resource-badge--${badge.key}`]"
                  :aria-label="badge.label"
                  :data-testid="`project-resource-badge-${badge.key}-${projectRow(row).application_id}`"
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
                  :create-label="t('project.list.actions.create')"
                  :reset-label="t('project.list.clearFilters')"
                  :show-reset="hasActiveFilters"
                  @create="navigateToSourceChooser"
                  @reset="resetFilters"
                />
              </template>
            </t-empty>
          </div>
        </template>
      </management-paged-table>

      <t-drawer
        v-if="!usesCompactApplicationList"
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
    <task-detail-drawer
      v-model:visible="taskDrawerVisible"
      :resolve-task-type="(taskType) => projectTaskTypeLabel(t, taskType)"
      :task-id="activeTaskId"
    />
  </div>
</template>
<script setup lang="ts">
import type { DialogInstance, TableProps } from 'tdesign-vue-next';
import { DialogPlugin } from 'tdesign-vue-next/es/dialog';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, h, onActivated, onDeactivated, onMounted, onUnmounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { listRuntimeTargets, type RuntimeTarget } from '@/modules/runtime-target/api/runtime-target';
import { getLatestTaskForOwner } from '@/modules/task/contract/latest-task';
import { isTerminalTaskStatus, observeTask, type TaskObserver } from '@/modules/task/contract/task-observer';
import { TaskDetailDrawer } from '@/modules/task/contract/task-ui';
import {
  ManagementBatchBar,
  ManagementEmptyState,
  ManagementPageContent,
  ManagementPagedTable,
  ManagementPageHeader,
  ManagementStatisticsBar,
  TableActionMenu,
  TableViewToolbar,
} from '@/shared/components/management';
import {
  AdvancedQueryFilterBuilder,
  type AdvancedQueryFilterFieldDefinition,
  type AdvancedQueryFilterTag,
  applySavedQueryViewPresentation,
  normalizeSavedQueryView,
  SavedQueryViewControl,
  useAdvancedQuerySorterUiState,
  useSavedQueryViews,
} from '@/shared/components/query-list';
import ResponsiveCardList from '@/shared/components/responsive/ResponsiveCardList.vue';
import { useResponsiveVariant } from '@/shared/composables';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import {
  assignEncodedSorters,
  createSingleSorter,
  decodeSorters,
  encodeSorters,
  moveSorterInState,
  normalizeSorters,
  prependSorterTags,
  removeSorterFromState,
  withSorterDirectionFromInput,
  withSorterFieldFromInput,
} from '@/shared/observability';
import { useRealtimeSchedulerStore } from '@/store';
import { useTabsRouterStore } from '@/store/modules/tabs-router';
import { createLogger } from '@/utils/logger';
import { localizeRouteTitleKey } from '@/utils/route/title';

import {
  deleteApplicationSavedView,
  getApplications,
  getApplicationSavedViews,
  postApplicationBatchActions,
  postApplicationDestroy,
  postApplicationRedeploy,
  postApplicationRestart,
  postApplicationSavedView,
  postApplicationStop,
  postApplicationUnregister,
  postApplicationUp,
  putApplicationSavedView,
} from '../../api/project';
import ProjectListEntryActions from '../../components/ProjectListEntryActions.vue';
import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import {
  projectDriftStatusLabel,
  projectLifecycleActionVisibility,
  projectRuntimeStatusLabel,
  projectSourceTypeLabel,
  projectTaskTypeLabel,
} from '../../shared/display';
import {
  projectLifecycleReviewStatusLabel,
  projectLifecycleReviewStatusTheme,
  projectRequiresLifecycleReview,
} from '../../shared/lifecycle';
import { acquireApplicationListRealtime, releaseApplicationListRealtime } from '../../shared/list-realtime';
import { appendResolvedTab, buildDetailTitleWithFallback } from '../../shared/navigation';
import type {
  ApplicationBatchAction,
  ApplicationBatchActionItem,
  ApplicationBatchActionResponse,
  ApplicationDeploymentAdapterKind,
  ApplicationDestroyRequest,
  ApplicationDetailResponse,
  ApplicationDriftStatus,
  ApplicationFilters,
  ApplicationListItemWithLifecycle,
  ApplicationListQuery,
  ApplicationProvider,
  ApplicationRuntimeStatus,
  ApplicationSavedViewQueryState,
  ApplicationSavedViewRequest,
  ApplicationSourceType,
  ApplicationTaskReceipt,
} from '../../types/project';

// 项目列表以 Query 管理服务端项目快照，筛选/保存视图属于页面状态；任务观察器只负责异步操作进度反馈。
defineOptions({
  name: 'ApplicationListIndex',
});

const { t } = useI18n();
const router = useRouter();
const realtimeSchedulerStore = useRealtimeSchedulerStore();
const tabsRouterStore = useTabsRouterStore();
const logger = createLogger('project.list');
const projectPageRef = ref<HTMLElement | null>(null);
const projectPageVariant = useResponsiveVariant(projectPageRef, { presentation: 'entity' });
const usesCompactApplicationList = computed(() => projectPageVariant.value.density === 'compact');
const activeTaskId = ref<number | null>(null);
const taskDrawerVisible = ref(false);
const openingTaskRowIds = ref(new Set<string>());
let taskOpenRequestVersion = 0;

type HeaderStatusSummaryKey = 'running' | 'degraded' | 'stopped' | 'transitioning' | 'unknown';
type ApplicationListDriftTone = 'clean' | 'drifted' | 'unknown';
type PendingApplicationAction = 'up' | 'stop' | 'restart' | 'redeploy';
type ApplicationResourceBadgeKey = 'running' | 'stopped' | 'transitioning' | 'issue' | 'unknown';
type ApplicationBatchActionUi = ApplicationBatchAction;
type PendingApplicationActionState = {
  action: PendingApplicationAction;
  awaitingVisibleChange: boolean;
  deadlineAt: number | null;
  runtimeStatus: ApplicationRuntimeStatus | null;
  taskId?: number;
};
type ApplicationResourceBadge = {
  key: ApplicationResourceBadgeKey;
  count: number;
  label: string;
  icon: string;
};
type ApplicationFilterFieldKey =
  'deploymentAdapterKind' | 'provider' | 'runtimeTargetId' | 'sourceType' | 'driftStatus' | 'runtimeStatus';
type ApplicationBuilderFieldKey = 'sorterBuilder' | ApplicationFilterFieldKey;
type ApplicationSortBy = 'created_at';
type ApplicationFilterState = ApplicationFilters & {
  sorters: Array<{ field: ApplicationSortBy; direction?: 'asc' | 'desc' }>;
};
type ApplicationSavedQueryViewState = {
  pageSize: number;
  queryState: ApplicationSavedViewQueryState;
  visibleColumns: string[];
};

const tableLoading = ref(false);
const refreshing = ref(false);
const errorMessage = ref('');
const rows = ref<ApplicationListItemWithLifecycle[]>([]);
const pagination = ref({
  current: 1,
  pageSize: 20,
  total: 0,
});
const filters = ref<ApplicationFilterState>(createDefaultFilters());
const runtimeTargets = ref<RuntimeTarget[]>([]);
const columnDrawerVisible = ref(false);
const selectedApplicationFilterField = ref<ApplicationBuilderFieldKey>('deploymentAdapterKind');

watch(usesCompactApplicationList, (usesCompact) => {
  if (usesCompact) {
    clearSelection();
    columnDrawerVisible.value = false;
  }
});

const sourceTypeOptions: ApplicationSourceType[] = ['imported', 'managed', 'template'];
const driftStatusOptions: ApplicationDriftStatus[] = ['unknown', 'clean', 'changed', 'missing'];
const runtimeStatusOptions: ApplicationRuntimeStatus[] = ['running', 'degraded', 'stopped', 'transitioning', 'unknown'];
const projectSortOptions = computed(() => [
  { label: t('project.list.filters.sortCreatedAt'), value: 'created_at' as const },
]);
const projectSortDirectionOptions = computed(() => [
  { label: t('project.list.filters.sortDesc'), value: 'desc' as const },
  { label: t('project.list.filters.sortAsc'), value: 'asc' as const },
]);
const { normalizedSorters, sortFieldOptionsByIndex, sortAddDisabled, sortMoveUpDisabled, sortMoveDownDisabled } =
  useAdvancedQuerySorterUiState<ApplicationSortBy>(
    () => filters.value.sorters,
    () => projectSortOptions.value,
  );
const providerOptions = computed(() =>
  [...new Set(runtimeTargets.value.map((target) => target.runtime.provider))].filter(
    (provider): provider is ApplicationProvider => provider === 'docker',
  ),
);
const filteredRuntimeTargets = computed(() =>
  filters.value.provider === 'all'
    ? runtimeTargets.value
    : runtimeTargets.value.filter((target) => target.runtime.provider === filters.value.provider),
);
const projectFilterDefinitions = computed<AdvancedQueryFilterFieldDefinition[]>(() => [
  { key: 'sorterBuilder', kind: 'special', label: t('project.list.filters.sorterBuilder') },
  {
    key: 'deploymentAdapterKind',
    kind: 'select',
    label: t('project.list.filters.deploymentAdapterKind'),
    options: [
      { label: t('project.list.filters.allDeploymentAdapterKinds'), value: 'all' },
      { label: t('project.list.deploymentAdapterKind.compose'), value: 'compose' },
    ],
    placeholder: t('project.list.filters.deploymentAdapterKind'),
  },
  {
    key: 'provider',
    kind: 'select',
    label: t('project.list.filters.provider'),
    options: [
      { label: t('project.list.filters.allProviders'), value: 'all' },
      ...providerOptions.value.map((value) => ({ label: providerLabel(value), value })),
    ],
    placeholder: t('project.list.filters.provider'),
  },
  {
    key: 'runtimeTargetId',
    kind: 'select',
    label: t('project.list.filters.runtimeTarget'),
    options: [
      { label: t('project.list.filters.allRuntimeTargets'), value: '' },
      ...filteredRuntimeTargets.value.map((target) => ({ label: target.displayName, value: String(target.id) })),
    ],
    placeholder: t('project.list.filters.runtimeTarget'),
  },
  {
    key: 'sourceType',
    kind: 'select',
    label: t('project.list.filters.sourceType'),
    options: [
      { label: t('project.list.filters.allSourceTypes'), value: 'all' },
      ...sourceTypeOptions.map((value) => ({ label: sourceTypeLabel(value), value })),
    ],
    placeholder: t('project.list.filters.sourceType'),
  },
  {
    key: 'driftStatus',
    kind: 'select',
    label: t('project.list.filters.driftStatus'),
    options: [
      { label: t('project.list.filters.allDriftStatuses'), value: 'all' },
      ...driftStatusOptions.map((value) => ({ label: driftStatusLabel(value), value })),
    ],
    placeholder: t('project.list.filters.driftStatus'),
  },
  {
    key: 'runtimeStatus',
    kind: 'select',
    label: t('project.list.filters.runtimeStatus'),
    options: [
      { label: t('project.list.filters.allRuntimeStatuses'), value: 'all' },
      ...runtimeStatusOptions.map((value) => ({ label: runtimeStatusLabel(value), value: value ?? 'unknown' })),
    ],
    placeholder: t('project.list.filters.runtimeStatus'),
  },
]);
const projectFilterFieldValues = computed<Record<ApplicationFilterFieldKey, string>>(() => ({
  deploymentAdapterKind: filters.value.deploymentAdapterKind,
  provider: filters.value.provider,
  runtimeTargetId: filters.value.runtimeTargetId ? String(filters.value.runtimeTargetId) : '',
  sourceType: filters.value.sourceType,
  driftStatus: filters.value.driftStatus,
  runtimeStatus: filters.value.runtimeStatus ?? 'unknown',
}));
const projectFilterTags = computed<AdvancedQueryFilterTag[]>(() => {
  const fieldMap = new Map(projectFilterDefinitions.value.map((field) => [field.key, field]));
  const tags = (Object.entries(projectFilterFieldValues.value) as Array<[ApplicationFilterFieldKey, string]>).reduce<
    AdvancedQueryFilterTag[]
  >((tags, [key, value]) => {
    if (!value || value === 'all') return tags;
    const field = fieldMap.get(key);
    const label = field?.options?.find((option) => option.value === value)?.label ?? value;
    tags.push({ key, label: `${field?.label ?? key}: ${label}` });
    return tags;
  }, []);

  return prependSorterTags(
    tags,
    normalizedSorters.value,
    projectSortOptions.value,
    t('project.list.filters.sortTagPrefix'),
  );
});

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
  {
    colKey: 'deploymentAdapterKind',
    title: t('project.list.columns.deploymentAdapterKind'),
    width: 144,
    align: 'center',
  },
  { colKey: 'runtimeTarget', title: t('project.list.columns.runtimeTarget'), width: 180 },
  { colKey: 'provider', title: t('project.list.columns.provider'), width: 128, align: 'center' },
  { colKey: 'source', title: t('project.list.columns.source'), width: 112, align: 'center' },
  { colKey: 'runtime', title: t('project.list.columns.runtime'), width: 148, align: 'center' },
  { colKey: 'resources', title: t('project.list.columns.resources'), width: 236, align: 'center' },
  { colKey: 'drift', title: t('project.list.columns.drift'), width: 124, align: 'center' },
  { colKey: 'operation', title: t('project.list.columns.operation'), width: 152, fixed: 'right', align: 'center' },
]);
const visibleColumnKeys = ref([
  'row-select',
  'name',
  'deploymentAdapterKind',
  'runtimeTarget',
  'provider',
  'source',
  'runtime',
  'resources',
  'drift',
  'operation',
]);
const visibleColumns = computed(() =>
  (configurableColumns.value ?? []).filter((column) => visibleColumnKeys.value.includes(String(column?.colKey))),
);
const projectSavedViews = useSavedQueryViews<ApplicationSavedQueryViewState, number>({
  adapter: {
    list: async () =>
      (await getApplicationSavedViews()).map((view) =>
        normalizeSavedQueryView<ApplicationSavedViewQueryState, number>(view),
      ),
    create: async (input) =>
      normalizeSavedQueryView<ApplicationSavedViewQueryState, number>(
        await postApplicationSavedView(toApplicationSavedViewRequest(input)),
      ),
    update: async (id, input) =>
      normalizeSavedQueryView<ApplicationSavedViewQueryState, number>(
        await putApplicationSavedView(id, toApplicationSavedViewRequest(input)),
      ),
    remove: async (id) => {
      await deleteApplicationSavedView(id);
    },
  },
  applyView: async (view) => {
    applyApplicationSavedQueryView(view.state);
    await fetchApplications();
  },
  onError: (error, operation) => {
    const fallback =
      operation === 'delete' ? t('project.list.savedViews.deleteFailed') : t('project.list.savedViews.conflict');
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, fallback));
  },
  serializeCurrentState: () => ({
    pageSize: pagination.value.pageSize,
    queryState: currentSavedViewQueryState(),
    visibleColumns: [...visibleColumnKeys.value],
  }),
});
const confirmDialogOpen = ref(false);
const realtimeActive = ref(false);
const pendingRowActions = ref<Record<string, PendingApplicationActionState>>({});
const selectedRowKeys = ref<string[]>([]);
const batchActionLoading = ref<ApplicationBatchActionUi | ''>('');

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
  ([{ key: 'running' }, { key: 'degraded' }, { key: 'stopped' }, { key: 'transitioning' }, { key: 'unknown' }] as const)
    .filter((item) => projectStatusCounts.value[item.key] > 0)
    .map((item) => ({
      ...item,
      count: projectStatusCounts.value[item.key],
    })),
);
const hasActiveFilters = computed(
  () =>
    Boolean(filters.value.keyword.trim()) ||
    filters.value.deploymentAdapterKind !== 'all' ||
    (typeof filters.value.runtimeTargetId === 'number' && filters.value.runtimeTargetId > 0) ||
    filters.value.provider !== 'all' ||
    filters.value.sourceType !== 'all' ||
    filters.value.runtimeStatus !== 'all' ||
    filters.value.driftStatus !== 'all',
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
  const rowMap = new Map(rows.value.map((row) => [row.application_id, row]));
  return selectedRowKeys.value
    .map((id) => rowMap.get(id))
    .filter((row): row is ApplicationListItemWithLifecycle => Boolean(row));
});

onMounted(() => {
  realtimeActive.value = true;
  syncApplicationListRealtimeSubscription();
  void fetchApplications();
  void loadRuntimeTargets();
  void loadSavedViews();
});

onUnmounted(() => {
  realtimeActive.value = false;
  syncApplicationListRealtimeSubscription();
  clearPendingRowActionTimeouts();
  clearPendingTaskObservers();
});

onActivated(() => {
  realtimeActive.value = true;
  syncApplicationListRealtimeSubscription();
  void fetchApplications();
});

onDeactivated(() => {
  realtimeActive.value = false;
  syncApplicationListRealtimeSubscription();
});

watch(
  () => realtimeSchedulerStore.allowPolling,
  () => {
    syncApplicationListRealtimeSubscription();
  },
);

function projectRow(row: unknown) {
  return row as ApplicationListItemWithLifecycle;
}

function sourceTypeLabel(value: ApplicationSourceType) {
  return projectSourceTypeLabel(t, value);
}

function deploymentAdapterLabel(value: ApplicationDeploymentAdapterKind) {
  return t(`project.list.deploymentAdapterKind.${value}`);
}

function providerLabel(value: ApplicationProvider) {
  return t(`project.list.provider.${value}`);
}

function projectProviderLabel(row: ApplicationListItemWithLifecycle) {
  return row.runtime_target ? providerLabel(row.runtime_target.provider) : '-';
}

function driftStatusLabel(value: ApplicationDriftStatus) {
  return projectDriftStatusLabel(t, value);
}

function runtimeStatusLabel(value?: ApplicationRuntimeStatus | null) {
  return projectRuntimeStatusLabel(t, value);
}

function normalizeRuntimeStatus(value?: ApplicationRuntimeStatus | null): HeaderStatusSummaryKey {
  if (value === 'running' || value === 'degraded' || value === 'stopped' || value === 'transitioning') {
    return value;
  }

  return 'unknown';
}

function normalizeDriftStatus(value: ApplicationDriftStatus): ApplicationListDriftTone {
  if (value === 'clean') {
    return 'clean';
  }

  if (value === 'unknown') {
    return 'unknown';
  }

  return 'drifted';
}

function projectResourceBadgeLabel(key: ApplicationResourceBadgeKey, count: number) {
  return t('project.list.resources.statusValue', {
    count,
    status: t(`project.list.resources.${key}`),
  });
}

function projectResourceBadgeIcon(key: ApplicationResourceBadgeKey) {
  if (key === 'running') return '🟢';
  if (key === 'stopped') return '⚫';
  if (key === 'transitioning') return '🟠';
  if (key === 'issue') return '🔴';
  return '⚪';
}

function projectContainerBadges(row: ApplicationListItemWithLifecycle): ApplicationResourceBadge[] {
  const badges: ApplicationResourceBadge[] = [
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
  const fallbackKey: ApplicationResourceBadgeKey =
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

function projectSecondaryName(row: ApplicationListItemWithLifecycle) {
  const canonicalName = row.compose_project_name?.trim() || '';
  const displayName = row.display_name?.trim() || '';

  if (!canonicalName || canonicalName === displayName) {
    return '';
  }

  return t('project.list.canonicalNameValue', { name: canonicalName });
}

let refreshRequestSeq = 0;
const pendingRowTimeouts = new Map<string, number>();
const pendingTaskObservers = new Map<string, TaskObserver>();

function syncApplicationListRealtimeSubscription() {
  if (realtimeActive.value && realtimeSchedulerStore.allowPolling) {
    acquireApplicationListRealtime(handleApplicationListRealtimeItems);
    return;
  }
  releaseApplicationListRealtime(handleApplicationListRealtimeItems);
}

async function fetchApplications() {
  const requestSeq = ++refreshRequestSeq;
  const shouldBlockTable = rows.value.length === 0 && !tableLoading.value;
  if (shouldBlockTable) {
    tableLoading.value = true;
  } else {
    refreshing.value = true;
  }
  errorMessage.value = '';
  try {
    const query: ApplicationListQuery = {
      limit: pagination.value.pageSize,
      offset: (pagination.value.current - 1) * pagination.value.pageSize,
    };
    assignEncodedSorters(query, filters.value.sorters, projectSortOptions.value);
    if (filters.value.keyword.trim()) query.keyword = filters.value.keyword.trim();
    if (filters.value.deploymentAdapterKind !== 'all')
      query.deployment_adapter_kind = filters.value.deploymentAdapterKind;
    if (filters.value.runtimeTargetId && filters.value.runtimeTargetId > 0) {
      query.runtime_target_id = filters.value.runtimeTargetId;
    }
    if (filters.value.provider !== 'all') query.provider = filters.value.provider;
    if (filters.value.sourceType !== 'all') query.source_type = filters.value.sourceType;
    if (filters.value.runtimeStatus !== 'all') query.runtime_status = filters.value.runtimeStatus;
    if (filters.value.driftStatus !== 'all') query.drift_status = filters.value.driftStatus;
    const response = await getApplications(query);
    if (requestSeq !== refreshRequestSeq) {
      return;
    }
    syncPaginationFromResponse(response);
    const nextRows = response.items;
    rows.value = nextRows;
    reconcilePendingRowActions(nextRows);
    selectedRowKeys.value = selectedRowKeys.value.filter((id) => nextRows.some((row) => row.application_id === id));
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

async function loadRuntimeTargets() {
  try {
    runtimeTargets.value = await listRuntimeTargets();
  } catch (error) {
    logger.error('failed to load runtime targets', error);
    runtimeTargets.value = [];
    MessagePlugin.error(t('project.list.runtimeTargetsLoadFailed'));
  }
}

async function loadSavedViews() {
  const loaded = await projectSavedViews.load();
  if (!loaded) {
    logger.error('failed to load project saved views');
    MessagePlugin.error(t('project.list.savedViews.loadFailed'));
  }
}

function handleApplicationListRealtimeItems(
  items: Array<{
    application_id: string;
    runtime_status: ApplicationRuntimeStatus;
    service_count: number;
    container_counts: ApplicationListItemWithLifecycle['container_counts'];
    drift_status: ApplicationDriftStatus;
  }>,
) {
  if (!realtimeActive.value || !realtimeSchedulerStore.allowPolling || rows.value.length === 0) {
    return;
  }
  const patchById = new Map(items.map((item) => [item.application_id, item]));
  let changed = false;
  const nextRows = rows.value.map((row) => {
    const patch = patchById.get(row.application_id);
    if (!patch) {
      return row;
    }
    const nextRow: ApplicationListItemWithLifecycle = {
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

function reconcilePendingRowActions(nextRows: ApplicationListItemWithLifecycle[]) {
  const nextPending = { ...pendingRowActions.value };
  const rowMap = new Map(nextRows.map((row) => [row.application_id, row]));

  for (const [rawId, pending] of Object.entries(nextPending)) {
    const id = rawId;
    const row = rowMap.get(id);
    if (!row) {
      clearPendingRowActionTimeout(id);
      pendingTaskObservers.get(id)?.stop();
      pendingTaskObservers.delete(id);
      delete nextPending[id];
      continue;
    }

    if (pending.taskId) continue;
    const runtimeChanged = (row.runtime_status ?? null) !== pending.runtimeStatus;
    if (pending.awaitingVisibleChange && runtimeChanged) {
      clearPendingRowActionTimeout(id);
      delete nextPending[id];
    }
  }

  pendingRowActions.value = nextPending;
}

function markPendingRowAction(row: ApplicationListItemWithLifecycle, action: PendingApplicationAction) {
  clearPendingRowActionTimeout(row.application_id);
  pendingRowActions.value = {
    ...pendingRowActions.value,
    [row.application_id]: {
      action,
      awaitingVisibleChange: false,
      deadlineAt: null,
      runtimeStatus: row.runtime_status ?? null,
    },
  };
}

function markPendingRowActions(rowsToMark: ApplicationListItemWithLifecycle[], action: PendingApplicationAction) {
  if (rowsToMark.length === 0) {
    return;
  }

  const nextPending = { ...pendingRowActions.value };
  for (const row of rowsToMark) {
    clearPendingRowActionTimeout(row.application_id);
    nextPending[row.application_id] = {
      action,
      awaitingVisibleChange: false,
      deadlineAt: null,
      runtimeStatus: row.runtime_status ?? null,
    };
  }
  pendingRowActions.value = nextPending;
}

function markPendingRowActionAwaitingChange(rowId: string) {
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

function observePendingTask(rowId: string, taskId: number) {
  pendingTaskObservers.get(rowId)?.stop();
  const observer = observeTask(taskId, {
    onError: (error) => logger.warn('task status observation failed', { error, rowId, taskId }),
    onTask: (task) => {
      if (!isTerminalTaskStatus(task.status)) return;
      if (task.status === 'success') {
        awaitPendingRowRuntimeChange(rowId);
        void fetchApplications();
        return;
      }
      clearPendingRowAction(rowId);
    },
  });
  pendingTaskObservers.set(rowId, observer);
}

function markPendingRowActionTask(rowId: string, taskId: number) {
  const pending = pendingRowActions.value[rowId];
  if (!pending) return;
  clearPendingRowActionTimeout(rowId);
  pendingRowActions.value = {
    ...pendingRowActions.value,
    [rowId]: { ...pending, awaitingVisibleChange: false, deadlineAt: null, taskId },
  };
  observePendingTask(rowId, taskId);
}

// 任务完成与运行快照更新是两个独立事件；成功后等待列表确认，避免先显示操作前状态。
function awaitPendingRowRuntimeChange(rowId: string) {
  const pending = pendingRowActions.value[rowId];
  if (!pending) return;
  pendingTaskObservers.get(rowId)?.stop();
  pendingTaskObservers.delete(rowId);
  const deadlineAt = Date.now() + 15_000;
  pendingRowActions.value = {
    ...pendingRowActions.value,
    [rowId]: {
      ...pendingRowActions.value[rowId],
      awaitingVisibleChange: true,
      deadlineAt,
      taskId: undefined,
    },
  };
  schedulePendingRowActionTimeout(rowId, deadlineAt);
}

function markPendingRowActionsAwaitingChange(rowIds: string[]) {
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

function clearPendingRowAction(rowId: string) {
  if (!pendingRowActions.value[rowId]) {
    return;
  }
  clearPendingRowActionTimeout(rowId);
  pendingTaskObservers.get(rowId)?.stop();
  pendingTaskObservers.delete(rowId);
  const nextPending = { ...pendingRowActions.value };
  delete nextPending[rowId];
  pendingRowActions.value = nextPending;
}

function clearPendingRowActions(rowIds: string[]) {
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
    pendingTaskObservers.get(rowId)?.stop();
    pendingTaskObservers.delete(rowId);
    delete nextPending[rowId];
    changed = true;
  }

  if (changed) {
    pendingRowActions.value = nextPending;
  }
}

function isRowActionPending(rowId: string) {
  return Boolean(pendingRowActions.value[rowId]);
}

function schedulePendingRowActionTimeout(rowId: string, deadlineAt: number) {
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

function clearPendingRowActionTimeout(rowId: string) {
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

function clearPendingTaskObservers() {
  for (const observer of pendingTaskObservers.values()) observer.stop();
  pendingTaskObservers.clear();
}

function clearSelection() {
  selectedRowKeys.value = [];
}

function handleSelectChange(rowKeys: Array<string | number>) {
  selectedRowKeys.value = rowKeys.map(String).filter(Boolean);
}

function resetFilters() {
  filters.value = createDefaultFilters();
  projectSavedViews.selectedId.value = undefined;
  pagination.value.current = 1;
  void fetchApplications();
}

function createDefaultFilters(): ApplicationFilterState {
  return {
    keyword: '',
    deploymentAdapterKind: 'all',
    runtimeTargetId: undefined,
    provider: 'all',
    sourceType: 'all',
    runtimeStatus: 'all',
    driftStatus: 'all',
    sorters: createSingleSorter('created_at', 'desc'),
  };
}

function handleFilterQuery() {
  pagination.value.current = 1;
  void fetchApplications();
}

function handlePageChange(pageInfo: { current: number; pageSize: number }) {
  pagination.value.current = pageInfo.current;
  pagination.value.pageSize = pageInfo.pageSize;
  void fetchApplications();
}

function currentSavedViewQueryState(): ApplicationSavedViewQueryState {
  return {
    ...(filters.value.keyword.trim() ? { keyword: filters.value.keyword.trim() } : {}),
    ...(filters.value.deploymentAdapterKind !== 'all'
      ? { deployment_adapter_kind: filters.value.deploymentAdapterKind }
      : {}),
    ...(filters.value.runtimeTargetId && filters.value.runtimeTargetId > 0
      ? { runtime_target_id: filters.value.runtimeTargetId }
      : {}),
    ...(filters.value.provider !== 'all' ? { provider: filters.value.provider } : {}),
    ...(filters.value.sourceType !== 'all' ? { source_type: filters.value.sourceType } : {}),
    ...(filters.value.runtimeStatus !== 'all' ? { runtime_status: filters.value.runtimeStatus } : {}),
    ...(filters.value.driftStatus !== 'all' ? { drift_status: filters.value.driftStatus } : {}),
    sort: encodeSorters(normalizedSorters.value, projectSortOptions.value) as NonNullable<
      ApplicationSavedViewQueryState['sort']
    >,
  };
}

function toApplicationSavedViewRequest(input: {
  name: string;
  state: ApplicationSavedQueryViewState;
}): ApplicationSavedViewRequest {
  return {
    name: input.name,
    page_size: input.state.pageSize,
    query_state: input.state.queryState,
    visible_columns: input.state.visibleColumns as ApplicationSavedViewRequest['visible_columns'],
  };
}

function applyApplicationSavedQueryView(savedState: ApplicationSavedQueryViewState) {
  const state = savedState.queryState;
  const restoredSorters = normalizeSorters(
    decodeSorters(state.sort ?? [], normalizeApplicationSortField, normalizeApplicationSortDirection),
    projectSortOptions.value,
  );
  filters.value = {
    keyword: state.keyword ?? '',
    deploymentAdapterKind: state.deployment_adapter_kind ?? 'all',
    runtimeTargetId: state.runtime_target_id,
    provider: state.provider ?? 'all',
    sourceType: state.source_type ?? 'all',
    runtimeStatus: state.runtime_status ?? 'all',
    driftStatus: state.drift_status ?? 'all',
    sorters: restoredSorters.length ? restoredSorters : createSingleSorter('created_at', 'desc'),
  };
  applySavedQueryViewPresentation(savedState, {
    pagination: pagination.value,
    supportedColumns: (configurableColumns.value ?? []).map((column) => String(column.colKey)),
    visibleColumnKeys,
  });
}

function updateApplicationFilterField(payload: { key: string; value: string | string[] }) {
  const value = Array.isArray(payload.value) ? (payload.value[0] ?? '') : payload.value;
  const key = payload.key as ApplicationFilterFieldKey;
  if (key === 'runtimeTargetId') {
    const parsed = Number(value);
    filters.value.runtimeTargetId = Number.isInteger(parsed) && parsed > 0 ? parsed : undefined;
    return;
  }
  if (key === 'deploymentAdapterKind') filters.value.deploymentAdapterKind = value === 'compose' ? 'compose' : 'all';
  if (key === 'provider') filters.value.provider = value === 'docker' ? 'docker' : 'all';
  if (key === 'sourceType') {
    filters.value.sourceType = sourceTypeOptions.includes(value as ApplicationSourceType)
      ? (value as ApplicationSourceType)
      : 'all';
  }
  if (key === 'driftStatus') {
    filters.value.driftStatus = driftStatusOptions.includes(value as ApplicationDriftStatus)
      ? (value as ApplicationDriftStatus)
      : 'all';
  }
  if (key === 'runtimeStatus') {
    filters.value.runtimeStatus = runtimeStatusOptions.includes(value as ApplicationRuntimeStatus)
      ? (value as ApplicationRuntimeStatus)
      : 'all';
  }
}

function addApplicationSorter() {
  filters.value = {
    ...filters.value,
    sorters: filters.value.sorters.length ? filters.value.sorters : createSingleSorter('created_at', 'desc'),
  };
}

function removeApplicationSorter(index: number) {
  filters.value = removeSorterFromState(filters.value, index, projectSortOptions.value);
}

function moveApplicationSorterUp(index: number) {
  filters.value = moveSorterInState(filters.value, index, -1, projectSortOptions.value);
}

function moveApplicationSorterDown(index: number) {
  filters.value = moveSorterInState(filters.value, index, 1, projectSortOptions.value);
}

function normalizeApplicationSortField(value: string): ApplicationSortBy | '' {
  return value === 'created_at' ? 'created_at' : '';
}

function normalizeApplicationSortDirection(value: string) {
  return value === 'asc' ? 'asc' : 'desc';
}

function updateApplicationSortField(payload: {
  index: number;
  value: string | number | Array<string | number> | undefined;
}) {
  filters.value = withSorterFieldFromInput(
    filters.value,
    payload.index,
    payload.value,
    normalizeApplicationSortField,
    projectSortOptions.value,
  );
}

function updateApplicationSortDirection(payload: {
  index: number;
  value: string | number | Array<string | number> | undefined;
}) {
  filters.value = withSorterDirectionFromInput(
    filters.value,
    payload.index,
    payload.value,
    normalizeApplicationSortDirection,
    projectSortOptions.value,
  );
}

function updateSelectedApplicationFilterField(value: string) {
  if (projectFilterDefinitions.value.some((field) => field.key === value)) {
    selectedApplicationFilterField.value = value as ApplicationBuilderFieldKey;
  }
}

watch(
  () => filters.value.runtimeTargetId,
  (targetId) => {
    const target = runtimeTargets.value.find((item) => item.id === targetId);
    if (target?.runtime.provider === 'docker') {
      filters.value.provider = target.runtime.provider;
    }
  },
);

watch(
  () => filters.value.provider,
  (provider) => {
    if (provider === 'all') return;
    const target = runtimeTargets.value.find((item) => item.id === filters.value.runtimeTargetId);
    if (target && target.runtime.provider !== provider) {
      filters.value.runtimeTargetId = undefined;
    }
  },
);

function navigateToDetail(row: ApplicationListItemWithLifecycle, tab?: string) {
  const target = {
    name: PROJECT_BOOTSTRAP_ROUTE.DETAIL.pageRouteName,
    params: { applicationId: row.application_id },
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

function navigateToSourceChooser() {
  const target = {
    name: PROJECT_BOOTSTRAP_ROUTE.CREATE.pageRouteName,
  };
  const resolved = router.resolve(target);
  appendResolvedTab(tabsRouterStore, resolved, localizeRouteTitleKey('project.route.create.title'));
  void router.push(target);
}

async function runAction(
  handler: (id: string) => Promise<ApplicationTaskReceipt | ApplicationDetailResponse | unknown>,
  row: ApplicationListItemWithLifecycle,
  successMessage: string,
  pendingAction?: PendingApplicationAction,
) {
  if (pendingAction) {
    markPendingRowAction(row, pendingAction);
  }
  try {
    const receipt = await handler(row.application_id);
    if (isTaskReceipt(receipt)) {
      if (pendingAction) markPendingRowActionTask(row.application_id, receipt.task_id);
      openTaskDrawer(receipt.task_id);
      MessagePlugin.success(t('project.list.actions.taskAccepted'));
    } else {
      if (pendingAction) markPendingRowActionAwaitingChange(row.application_id);
      MessagePlugin.success(successMessage);
    }
    await fetchApplications();
  } catch (error) {
    clearPendingRowAction(row.application_id);
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.list.actions.actionFailed')));
  }
}

function buildRowActions(row: ApplicationListItemWithLifecycle) {
  const visibility = projectLifecycleActionVisibility(row.runtime_status, {
    hideLifecycleActions: isRowActionPending(row.application_id),
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

function buildMobileRowActions(row: ApplicationListItemWithLifecycle) {
  const actions = buildRowActions(row);
  const primaryAction =
    actions.find((action) => action.value === 'stop') ?? actions.find((action) => action.value === 'up');

  if (!primaryAction) {
    return actions;
  }

  return [
    primaryAction,
    ...actions.filter((action) => action.value !== primaryAction.value && action.value !== 'detail'),
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

function isDeleteWorkspacePathAllowed(row: ApplicationListItemWithLifecycle) {
  return row.ownership_mode !== 'external';
}

function isRowBatchEligible(row: ApplicationListItemWithLifecycle, action: ApplicationBatchActionUi) {
  if (projectRequiresLifecycleReview(row) && ['start', 'stop', 'restart', 'redeploy'].includes(action)) {
    return false;
  }
  const visibility = projectLifecycleActionVisibility(row.runtime_status, {
    hideLifecycleActions: isRowActionPending(row.application_id),
  });
  if (action === 'start') return visibility.up;
  if (action === 'stop') return visibility.stop;
  if (action === 'restart') return visibility.restart;
  if (action === 'unregister') return true;
  if (action === 'redeploy') return !isRowActionPending(row.application_id);
  return true;
}

function batchActionableRows(action: ApplicationBatchActionUi) {
  return selectedRows.value.filter((row) => isRowBatchEligible(row, action));
}

function requiresSingleSelection(action: ApplicationBatchActionUi) {
  return action === 'destroy';
}

function isBatchActionDisabled(action: ApplicationBatchActionUi) {
  if (requiresSingleSelection(action) && selectedRows.value.length !== 1) {
    return true;
  }
  return batchActionableRows(action).length === 0;
}

function batchActionHint(action: ApplicationBatchActionUi) {
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
  row: ApplicationListItemWithLifecycle,
  action: 'up' | 'stop' | 'restart' | 'unregister' | 'redeploy' | 'destroy',
) {
  if (confirmDialogOpen.value) {
    return Promise.resolve(false);
  }

  return new Promise<boolean>((resolve) => {
    let settled = false;
    confirmDialogOpen.value = true;
    const deleteWorkspacePath = ref(false);
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
                    checked: deleteWorkspacePath.value,
                    disabled: !isDeleteWorkspacePathAllowed(row),
                    type: 'checkbox',
                    onInput: (event: Event) => {
                      deleteWorkspacePath.value = (event.target as HTMLInputElement).checked;
                      if (deleteWorkspacePath.value) {
                        autoUnregister.value = true;
                      }
                    },
                  }),
                  h('span', t('project.list.actions.destroyDeleteApplicationFiles')),
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
            auto_unregister: autoUnregister.value || deleteWorkspacePath.value,
            confirm_application_id: row.application_id,
            delete_workspace: deleteWorkspacePath.value,
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

async function runDestroy(row: ApplicationListItemWithLifecycle, payload: ApplicationDestroyRequest) {
  try {
    await postApplicationDestroy(row.application_id, payload);
    MessagePlugin.success(t('project.list.actions.actionSuccess'));
    await fetchApplications();
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.list.actions.actionFailed')));
  }
}

async function runRedeploy(row: ApplicationListItemWithLifecycle) {
  markPendingRowAction(row, 'redeploy');
  try {
    const receipt = await postApplicationRedeploy(row.application_id);
    markPendingRowActionTask(row.application_id, receipt.task_id);
    openTaskDrawer(receipt.task_id);
    MessagePlugin.success(t('project.list.actions.taskAccepted'));
    await fetchApplications();
  } catch (error) {
    clearPendingRowAction(row.application_id);
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.list.actions.actionFailed')));
  }
}

function isTaskReceipt(value: unknown): value is ApplicationTaskReceipt {
  return Boolean(value && typeof value === 'object' && typeof (value as ApplicationTaskReceipt).task_id === 'number');
}

function openTaskDrawer(taskId: number) {
  taskOpenRequestVersion += 1;
  activeTaskId.value = taskId;
  taskDrawerVisible.value = true;
}

async function openApplicationTask(row: ApplicationListItemWithLifecycle) {
  const pendingTaskId = pendingRowActions.value[row.application_id]?.taskId;
  if (pendingTaskId) {
    openTaskDrawer(pendingTaskId);
    return;
  }

  const requestVersion = ++taskOpenRequestVersion;
  openingTaskRowIds.value = new Set(openingTaskRowIds.value).add(row.application_id);
  try {
    const task = await getLatestTaskForOwner({ ownerId: row.application_id, ownerType: 'application' });
    if (requestVersion !== taskOpenRequestVersion) return;
    if (task) openTaskDrawer(task.id);
    else MessagePlugin.info(t('project.list.actions.noTaskHistory'));
  } catch (error) {
    if (requestVersion === taskOpenRequestVersion) {
      MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.list.actions.taskHistoryLoadFailed')));
    }
  } finally {
    const nextOpeningRows = new Set(openingTaskRowIds.value);
    nextOpeningRows.delete(row.application_id);
    openingTaskRowIds.value = nextOpeningRows;
  }
}

function runtimeStatusActionTooltip(row: ApplicationListItemWithLifecycle) {
  return isRowActionPending(row.application_id)
    ? t('project.list.statusTooltip.taskInProgress')
    : t('project.list.statusTooltip.viewLatestTask');
}

async function executeBatchAction(
  action: ApplicationBatchActionUi,
  overrides: {
    confirmComposeProjectName?: string;
    deleteWorkspacePath?: boolean;
    auto_unregister?: boolean;
    image_prune?: boolean;
    remove_named_volumes?: boolean;
  } = {},
) {
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
    const response = await postApplicationBatchActions({
      action,
      auto_unregister: overrides.auto_unregister ?? false,
      confirm_compose_project_name: overrides.confirmComposeProjectName,
      delete_workspace_path: overrides.deleteWorkspacePath ?? false,
      image_prune: overrides.image_prune ?? false,
      application_ids: actionableRows.map((row) => row.application_id),
      remove_named_volumes: overrides.remove_named_volumes ?? false,
    });
    if (pendingAction) {
      const completedRowIds = response.items
        .filter((item) => !item.skipped && item.result === 'completed')
        .map((item) => String(item.application_id));
      const blockedRowIds = response.items
        .filter((item) => item.skipped || item.result !== 'completed')
        .map((item) => String(item.application_id));
      markPendingRowActionsAwaitingChange(completedRowIds);
      clearPendingRowActions(blockedRowIds);
    }
    handleBatchActionResult(action, response);
    clearSelection();
    await fetchApplications();
  } catch (error) {
    if (pendingAction) {
      clearPendingRowActions(actionableRows.map((row) => row.application_id));
    }
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.list.batch.failed')));
  } finally {
    batchActionLoading.value = '';
  }
}

function batchFailureSummary(items: ApplicationBatchActionItem[]) {
  return items
    .filter((item) => !item.skipped && item.result !== 'completed')
    .map((item) => `${item.application_id}: ${item.message_key ? t(item.message_key) : item.message || '-'}`)
    .join('\n');
}

function batchActionLocaleSegment(action: ApplicationBatchActionUi) {
  return action;
}

function handleBatchActionResult(action: ApplicationBatchActionUi, response: ApplicationBatchActionResponse) {
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

function confirmBatchAction(action: ApplicationBatchActionUi) {
  if (isBatchActionDisabled(action) || confirmDialogOpen.value) {
    return;
  }
  confirmDialogOpen.value = true;
  const deleteWorkspacePath = ref(false);
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
                  checked: deleteWorkspacePath.value,
                  disabled: selectedRows.value.some((row) => !isDeleteWorkspacePathAllowed(row)),
                  type: 'checkbox',
                  onInput: (event: Event) => {
                    deleteWorkspacePath.value = (event.target as HTMLInputElement).checked;
                    if (deleteWorkspacePath.value) {
                      autoUnregister.value = true;
                    }
                  },
                }),
                h('span', t('project.list.actions.destroyDeleteApplicationFiles')),
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
        auto_unregister: autoUnregister.value || deleteWorkspacePath.value,
        confirmComposeProjectName:
          action === 'destroy' && selectedRows.value.length === 1 ? selectedRows.value[0]?.application_id : undefined,
        deleteWorkspacePath: deleteWorkspacePath.value,
        image_prune: false,
        remove_named_volumes: removeNamedVolumes.value,
      });
    },
  });
}

async function handleRowAction(action: string, row: ApplicationListItemWithLifecycle) {
  if (action === 'detail') {
    await navigateToDetail(row);
    return;
  }
  if (action === 'up') {
    if (!(await confirmDangerousAction(row, 'up'))) {
      return;
    }
    await runAction(postApplicationUp, row, t('project.list.actions.actionSuccess'), 'up');
    return;
  }
  if (action === 'stop') {
    if (!(await confirmDangerousAction(row, 'stop'))) {
      return;
    }
    await runAction(postApplicationStop, row, t('project.list.actions.actionSuccess'), 'stop');
    return;
  }
  if (action === 'restart') {
    if (!(await confirmDangerousAction(row, 'restart'))) {
      return;
    }
    await runAction(postApplicationRestart, row, t('project.list.actions.actionSuccess'), 'restart');
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
    await runAction(postApplicationUnregister, row, t('project.list.actions.actionSuccess'));
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
.project-header-summary {
  flex-direction: column;
  gap: var(--graft-density-gap-16);
}

.project-page :deep(.t-table__body > tr:not(.t-table__row--active)) {
  background-color: var(--td-bg-color-container);
}

.project-page :deep(.t-table__body > tr:not(.t-table__row--active):hover) {
  background-color: var(--td-bg-color-container-hover);
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

button.project-runtime-badge {
  background: transparent;
  border: 0;
  cursor: pointer;
  padding: 0;
}

button.project-runtime-badge:disabled {
  cursor: wait;
  opacity: 0.7;
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

.project-mobile-list {
  width: 100%;
}

.project-mobile-card {
  align-items: center;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  cursor: pointer;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  min-height: 104px;
  padding: var(--graft-density-gap-14);
}

.project-mobile-card:focus-visible {
  outline: 2px solid var(--td-brand-color);
  outline-offset: 2px;
}

.project-mobile-card__main {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  gap: var(--graft-density-gap-10);
  min-width: 0;
}

.project-mobile-card__title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-mobile-card__meta {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-10);
}

.project-mobile-card__actions {
  flex: 0 0 auto;
  width: auto;
}

.project-mobile-card__actions :deep(.table-action-menu) {
  width: auto;
}

@media (width <= 768px) {
  .project-table-head {
    align-items: flex-start;
    flex-direction: column;
  }

  .project-mobile-card {
    align-items: flex-start;
    flex-direction: column;
  }

  .project-mobile-card__actions,
  .project-mobile-card__actions :deep(.table-action-menu) {
    width: 100%;
  }

  .project-mobile-card__actions :deep(.t-button) {
    flex: 1 1 0;
  }
}
</style>
