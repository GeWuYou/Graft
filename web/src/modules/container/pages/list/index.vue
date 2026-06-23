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
          v-model="filters.orchestrator"
          class="management-toolbar__select"
          data-testid="container-filter-orchestrator"
          :placeholder="t('container.list.filters.orchestrator')"
        >
          <t-option value="all" :label="t('container.list.filters.allOrchestrators')" />
          <t-option
            v-for="orchestrator in orchestratorOptions"
            :key="orchestrator"
            :value="orchestrator"
            :label="orchestratorLabel(orchestrator)"
          />
        </t-select>
        <t-select
          v-model="filters.sourceScopeKind"
          class="management-toolbar__select"
          data-testid="container-filter-source-scope-kind"
          :disabled="!availableSourceScopeKinds.length"
          :placeholder="t('container.list.filters.sourceScopeKind')"
        >
          <t-option value="all" :label="t('container.list.filters.allSourceScopeKinds')" />
          <t-option
            v-for="scopeKind in availableSourceScopeKinds"
            :key="scopeKind"
            :value="scopeKind"
            :label="t(`container.list.sourceKinds.${scopeKind}`)"
          />
        </t-select>
        <t-input
          v-model="filters.sourceScope"
          class="management-toolbar__select management-toolbar__scope-input"
          clearable
          data-testid="container-filter-source-scope"
          :disabled="filters.sourceScopeKind === 'all'"
          :placeholder="sourceScopePlaceholder"
          @enter="applyFilters"
        />
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

    <management-table-card>
      <template #head>
        <div class="container-table-head">
          <p class="container-table-head__summary">
            {{ t('container.list.tableSummary', { count: listTotal }) }}
          </p>
          <p>{{ t('container.list.tableHint') }}</p>
        </div>
      </template>
      <template #toolbar>
        <table-view-toolbar
          :column-settings-label="t('container.list.columnSettings')"
          :density-label="tableDensityLabel"
          :refresh-label="t('container.list.refresh')"
          :refresh-loading="loading"
          @column-settings="columnDrawerVisible = true"
          @density="toggleTableDensity"
          @refresh="refreshContainers"
        />
      </template>
      <template #batch>
        <div v-if="selectedRows.length > 0" class="container-batch-bar">
          <span>{{ t('container.list.batch.selected', { count: selectedRows.length }) }}</span>
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

      <t-alert v-if="listError.title" class="container-alert" theme="error" :title="listError.title">
        <p v-if="listError.hint" class="container-alert__hint">{{ listError.hint }}</p>
        <template #operation>
          <t-button theme="danger" variant="text" @click="refreshContainers">
            {{ t('container.list.retry') }}
          </t-button>
        </template>
      </t-alert>

      <div ref="tableHostRef" class="container-table-host" :data-table-mode="tableWidthPolicy.mode">
        <t-table
          row-key="id"
          :columns="visibleColumns"
          :data="rows"
          :loading="loading"
          :size="tableDensity"
          :table-content-width="tableWidthPolicy.tableContentWidth"
          cell-empty-content="-"
          table-layout="fixed"
          :selected-row-keys="selectedRowKeys"
          hover
          @select-change="handleSelectChange"
        >
          <template #state="{ row }">
            <t-tag :theme="stateTheme(row.state)" variant="light-outline">
              {{ stateLabel(row.state) }}
            </t-tag>
          </template>

          <template #name="{ row }">
            <div class="container-identity">
              <span class="container-identity__name">{{ displayName(row) }}</span>
              <t-tooltip :content="row.id" placement="top-left">
                <span class="container-identity__id">{{ row.short_id || shortContainerId(row.id) }}</span>
              </t-tooltip>
            </div>
          </template>

          <template #image="{ row }">
            <div class="container-image">
              <span>{{ row.image }}</span>
              <span v-if="row.runtime" class="container-muted">{{ row.runtime }}</span>
            </div>
          </template>

          <template #ports="{ row }">
            <div v-if="visiblePortLabels(row).length" class="container-port-list">
              <t-tag v-for="port in visiblePortLabels(row)" :key="port" size="small" theme="default" variant="light">
                {{ port }}
              </t-tag>
              <t-tooltip
                v-if="hiddenPortLabels(row).length"
                :content="hiddenPortLabels(row).join(' / ')"
                placement="top"
              >
                <t-tag size="small" theme="primary" variant="light">
                  {{ t('container.list.morePorts', { count: hiddenPortLabels(row).length }) }}
                </t-tag>
              </t-tooltip>
            </div>
            <span v-else>-</span>
          </template>

          <template #runtime_status="{ row }">
            <div class="container-runtime-status">
              <span class="container-runtime-status__text">{{ row.status || '-' }}</span>
              <t-tag
                v-if="shouldShowHealthTag(row.health)"
                :theme="healthTheme(row.health)"
                size="small"
                variant="light"
              >
                {{ healthLabel(row.health) }}
              </t-tag>
            </div>
          </template>

          <template #network="{ row }">
            <div class="container-runtime-status">
              <span>{{ row.primary_ip || '-' }}</span>
              <span>{{ row.network_summary || '-' }}</span>
            </div>
          </template>

          <template #source="{ row }">
            <div class="container-source-cell">
              <div class="container-source-cell__header">
                <t-tag :theme="orchestratorTheme(row)" size="small" variant="light-outline">
                  {{ orchestratorLabel(readOrchestratorType(row)) }}
                </t-tag>
              </div>
              <div v-if="sourceGroupFilter(row)" class="container-source-cell__line">
                <span class="container-source-cell__label">{{ sourceGroupLabel(row) }}</span>
                <t-button
                  data-testid="container-source-group-filter"
                  size="small"
                  theme="primary"
                  variant="text"
                  @click="applySourceQuickFilter(row, 'group')"
                >
                  {{ sourceGroupFilter(row)?.value }}
                </t-button>
              </div>
              <div v-if="sourceMemberFilter(row)" class="container-source-cell__line">
                <span class="container-source-cell__label">{{ sourceMemberLabel(row) }}</span>
                <t-button
                  data-testid="container-source-member-filter"
                  size="small"
                  theme="default"
                  variant="text"
                  @click="applySourceQuickFilter(row, 'member')"
                >
                  {{ sourceMemberFilter(row)?.value }}
                </t-button>
              </div>
              <span v-if="!sourceGroupFilter(row) && !sourceMemberFilter(row)" class="container-muted">
                {{ orchestratorSummary(row) }}
              </span>
            </div>
          </template>

          <template #cpu="{ row }">
            <t-tooltip
              v-for="metric in [cpuMetric(row)]"
              :key="`cpu:${metric.value}`"
              :content="metric.tooltip"
              placement="top"
            >
              <div class="container-resource-meter" :data-available="metric.available">
                <t-progress
                  v-if="metric.available"
                  theme="circle"
                  :label="false"
                  :percentage="metric.percentage"
                  :size="36"
                  :stroke-width="4"
                />
                <span v-else class="container-resource-meter__empty"></span>
                <span>{{ metric.value }}</span>
              </div>
            </t-tooltip>
          </template>

          <template #memory="{ row }">
            <t-tooltip
              v-for="metric in [memoryMetric(row)]"
              :key="`memory:${metric.value}`"
              :content="metric.tooltip"
              placement="top"
            >
              <div class="container-resource-meter" :data-available="metric.available">
                <t-progress
                  v-if="metric.available"
                  theme="circle"
                  :label="false"
                  :percentage="metric.percentage"
                  :size="36"
                  :stroke-width="4"
                />
                <span v-else class="container-resource-meter__empty"></span>
                <span>{{ metric.value }}</span>
              </div>
            </t-tooltip>
          </template>

          <template #resource="{ row }">
            <span>{{ resourceSummary(row) }}</span>
          </template>

          <template #image_id="{ row }">
            <span>{{ row.image_id || '-' }}</span>
          </template>

          <template #labels="{ row }">
            <span>{{ labelSummary(row) }}</span>
          </template>

          <template #created_at="{ row }">
            {{ formatTime(row.created_at) }}
          </template>

          <template #started_at="{ row }">
            {{ formatTime(row.started_at) }}
          </template>

          <template #restart_policy="{ row }">
            {{ row.restart_policy || '-' }}
          </template>

          <template #operation="{ row }">
            <div class="container-actions" @click.stop>
              <t-button
                v-permission="permissionCodes.DETAIL"
                data-testid="container-action-detail"
                theme="default"
                variant="outline"
                size="small"
                @click="openDetail(row)"
              >
                {{ t('container.list.actions.detail') }}
              </t-button>
              <t-button
                data-testid="container-action-logs"
                theme="default"
                variant="outline"
                size="small"
                @click="navigateToDetail(row, 'logs')"
              >
                {{ t('container.list.actions.logs') }}
              </t-button>
              <t-dropdown
                v-if="moreRowActions(row).length"
                :options="moreRowActionOptions(row)"
                trigger="click"
                @click="(payload, context) => handleMoreRowAction(payload, context, row)"
              >
                <t-button data-testid="container-action-more" theme="default" variant="outline" size="small">
                  {{ t('container.list.actions.more') }}
                </t-button>
              </t-dropdown>
            </div>
          </template>

          <template #empty>
            <t-empty
              :title="t('container.list.emptyTitle')"
              :description="
                hasActiveFilters ? t('container.list.emptyFilteredDescription') : t('container.list.emptyDescription')
              "
            >
              <template v-if="hasActiveFilters" #action>
                <t-button theme="primary" variant="outline" @click="resetFilters">
                  {{ t('container.list.clearFilters') }}
                </t-button>
              </template>
            </t-empty>
          </template>
        </t-table>
      </div>

      <template #footer>
        <management-table-pagination :summary="footerSummary">
          <t-pagination
            v-model:current="pagination.current"
            v-model:page-size="pagination.pageSize"
            :page-size-options="[10, 20, 50, 100]"
            :show-page-number="true"
            :total="listTotal"
            @change="handlePageChange"
          />
        </management-table-pagination>
      </template>
    </management-table-card>

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
import type { DialogInstance, DropdownOption, TdBaseTableProps } from 'tdesign-vue-next';
import { DialogPlugin } from 'tdesign-vue-next/es/dialog';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { NotifyPlugin } from 'tdesign-vue-next/es/notification';
import { computed, h, onMounted, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { LOCALE, type LocalizedTitle } from '@/contracts/i18n/locales';
import {
  buildVisibleColumns,
  ManagementPageHeader,
  ManagementTableCard,
  ManagementTablePagination,
  ManagementToolbar,
  resolveTableWidthPolicy,
  TableViewToolbar,
  useTableHostWidth,
} from '@/shared/components/management';
import { AdvancedQueryColumnDrawer } from '@/shared/components/query-list';
import { useCurrentTabRefresh } from '@/shared/composables/useTabRefresh';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { formatLocaleDateTime } from '@/shared/observability';
import { useTabsRouterStore } from '@/store';
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
import { CONTAINER_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import { CONTAINER_PERMISSION_CODE } from '../../contract/permissions';
import type {
  ContainerAction,
  ContainerActionLevel,
  ContainerBatchActionItem,
  ContainerBatchActionResponse,
  ContainerFilters,
  ContainerHealth,
  ContainerListQueryWithOrchestrator,
  ContainerListSourceScopeKind,
  ContainerListSourceScopeQuery,
  ContainerListSummary,
  ContainerOrchestratorType,
  ContainerPort,
  ContainerRuntimeInfo,
  ContainerSourceGroupKind,
  ContainerSourceMemberKind,
  ContainerState,
  ContainerSummaryRecord,
} from '../../types/container';

defineOptions({
  name: 'ContainerListIndex',
});

const { locale, t } = useI18n();
const router = useRouter();
const tabsRouterStore = useTabsRouterStore();
const logger = createLogger('container.list');
const permissionCodes = CONTAINER_PERMISSION_CODE;

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
const healthOptions: ContainerHealth[] = ['healthy', 'unhealthy', 'starting', 'none', 'unavailable'];
const orchestratorOptions: ContainerOrchestratorType[] = ['standalone', 'compose', 'swarm', 'kubernetes', 'unknown'];
const CONTAINER_RUNTIME_DISABLED_MESSAGE_KEY = 'ops.container.error.runtimeDisabled';
const CONTAINER_COLUMN_STORAGE_KEY = 'graft.container.list.visibleColumns';
const DEFAULT_VISIBLE_COLUMNS = [
  'row-select',
  'state',
  'name',
  'image',
  'source',
  'cpu',
  'memory',
  'ports',
  'network',
  'runtime_status',
  'created_at',
  'operation',
];
const ALWAYS_VISIBLE_COLUMNS = ['row-select', 'state', 'name', 'operation'];
const ALL_COLUMN_KEYS = [
  'row-select',
  'state',
  'name',
  'image',
  'source',
  'cpu',
  'memory',
  'ports',
  'network',
  'runtime_status',
  'created_at',
  'started_at',
  'restart_policy',
  'image_id',
  'labels',
  'resource',
  'operation',
];
const CONTAINER_PORT_VISIBLE_LIMIT = 2;
const CONTAINER_DEFAULT_PAGE_SIZE = 20;
const BYTES_PER_MIB = 1024 * 1024;

type ListErrorState = {
  title: string;
  hint: string;
};
type DangerousContainerAction = Extract<ContainerAction, 'remove' | 'restart' | 'start' | 'stop'>;
type RowAction =
  | 'copy-id'
  | 'inspect'
  | 'remove'
  | 'restart'
  | 'start'
  | 'stop'
  | 'view-env'
  | 'view-mounts'
  | 'view-networks';
type ResourceMetric = {
  available: boolean;
  percentage: number;
  tooltip: string;
  value: string;
};
type DropdownActionValue = { value?: string | number | Record<string, unknown> } | string | number;
type DropdownActionContext = { e?: MouseEvent };
type SourceQuickFilterTarget = 'group' | 'member';
type SourceQuickFilterValue = {
  kind: ContainerSourceGroupKind | ContainerSourceMemberKind;
  value: string;
};

const loading = ref(false);
const listError = ref<ListErrorState>({ title: '', hint: '' });
const rows = ref<ContainerSummaryRecord[]>([]);
const runtime = ref<ContainerRuntimeInfo | null>(null);
const listSummary = ref<ContainerListSummary | null>(null);
const listTotal = ref(0);
const columnDrawerVisible = ref(false);
const visibleColumnKeys = ref<string[]>(loadVisibleColumnKeys());
const tableDensity = ref<'medium' | 'small'>('medium');
const selectedRowKeys = ref<Array<string | number>>([]);
const batchActionLoading = ref<DangerousContainerAction | ''>('');
const activeDangerousDialog = ref<DialogInstance | null>(null);
const dangerousDialogOpen = ref(false);
const pendingSourceScopeFilter = ref<SourceQuickFilterValue | null>(null);
const filters = reactive<ContainerFilters>({
  keyword: '',
  orchestrator: 'all',
  sourceScopeKind: 'all',
  sourceScope: '',
  status: 'all',
  health: 'all',
});
const pagination = reactive({
  current: 1,
  pageSize: CONTAINER_DEFAULT_PAGE_SIZE,
});

const allColumns = computed<TdBaseTableProps['columns']>(() => [
  { colKey: 'row-select', type: 'multiple', width: 48, fixed: 'left', align: 'center' },
  { title: t('container.list.columns.status'), colKey: 'state', width: 104, align: 'center', ellipsis: false },
  {
    title: t('container.list.columns.name'),
    colKey: 'name',
    minWidth: 260,
    ellipsis: { theme: 'default', placement: 'top-left' },
  },
  {
    title: t('container.list.columns.image'),
    colKey: 'image',
    minWidth: 280,
    ellipsis: { theme: 'default', placement: 'top-left' },
  },
  { title: t('container.list.columns.source'), colKey: 'source', width: 188, ellipsis: false },
  { title: t('container.list.columns.cpu'), colKey: 'cpu', width: 132, align: 'center', ellipsis: false },
  { title: t('container.list.columns.memory'), colKey: 'memory', width: 180, align: 'center', ellipsis: false },
  { title: t('container.list.columns.ports'), colKey: 'ports', width: 220, ellipsis: false },
  { title: t('container.list.columns.network'), colKey: 'network', width: 176, ellipsis: false },
  { title: t('container.list.columns.resource'), colKey: 'resource', width: 168, ellipsis: false },
  {
    title: t('container.list.columns.runtimeStatus'),
    colKey: 'runtime_status',
    minWidth: 220,
    ellipsis: { theme: 'default', placement: 'top-left' },
  },
  { title: t('container.list.columns.createdAt'), colKey: 'created_at', width: 168, align: 'center' },
  { title: t('container.list.columns.startedAt'), colKey: 'started_at', width: 168, align: 'center' },
  { title: t('container.list.columns.restartPolicy'), colKey: 'restart_policy', width: 140, align: 'center' },
  {
    title: t('container.list.columns.imageId'),
    colKey: 'image_id',
    width: 220,
    ellipsis: { theme: 'default', placement: 'top-left' },
  },
  {
    title: t('container.list.columns.labels'),
    colKey: 'labels',
    width: 180,
    ellipsis: { theme: 'default', placement: 'top-left' },
  },
  {
    title: t('container.list.columns.operation'),
    colKey: 'operation',
    width: 192,
    fixed: 'right',
    align: 'center',
    ellipsis: false,
  },
]);
const visibleColumns = computed<TdBaseTableProps['columns']>(() =>
  buildVisibleColumns(allColumns.value, visibleColumnKeys.value, ALWAYS_VISIBLE_COLUMNS),
);
const orchestratorSourceScopeKinds = computed<
  Record<ContainerOrchestratorType, Array<ContainerSourceGroupKind | ContainerSourceMemberKind>>
>(() => ({
  standalone: [],
  compose: ['compose_project', 'compose_service'],
  swarm: ['swarm_stack', 'swarm_task'],
  kubernetes: ['kubernetes_namespace', 'kubernetes_pod'],
  unknown: [],
}));
const { tableHostRef, tableHostWidth } = useTableHostWidth(() => visibleColumns.value);
const tableWidthPolicy = computed(() => resolveTableWidthPolicy(visibleColumns.value, tableHostWidth.value));
const availableSourceScopeKinds = computed<Array<ContainerSourceGroupKind | ContainerSourceMemberKind>>(() => {
  if (filters.orchestrator === 'all') {
    return [
      'compose_project',
      'compose_service',
      'swarm_stack',
      'swarm_task',
      'kubernetes_namespace',
      'kubernetes_pod',
    ];
  }

  return orchestratorSourceScopeKinds.value[filters.orchestrator] ?? [];
});
const sourceScopePlaceholder = computed(() => {
  if (filters.sourceScopeKind === 'all') {
    return t('container.list.filters.sourceScopePlaceholderDisabled');
  }

  return t('container.list.filters.sourceScopePlaceholder', {
    kind: t(`container.list.sourceKinds.${filters.sourceScopeKind}`),
  });
});
const hasActiveFilters = computed(
  () =>
    Boolean(filters.keyword.trim()) ||
    filters.orchestrator !== 'all' ||
    filters.sourceScopeKind !== 'all' ||
    Boolean(filters.sourceScope.trim()) ||
    filters.status !== 'all' ||
    filters.health !== 'all' ||
    Boolean(pendingSourceScopeFilter.value),
);
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
const columnSettingOptions = computed(() => [
  { label: t('container.list.columns.selection'), value: 'row-select' },
  { label: t('container.list.columns.status'), value: 'state' },
  { label: t('container.list.columns.name'), value: 'name' },
  { label: t('container.list.columns.image'), value: 'image' },
  { label: t('container.list.columns.source'), value: 'source' },
  { label: t('container.list.columns.cpu'), value: 'cpu' },
  { label: t('container.list.columns.memory'), value: 'memory' },
  { label: t('container.list.columns.ports'), value: 'ports' },
  { label: t('container.list.columns.network'), value: 'network' },
  { label: t('container.list.columns.resource'), value: 'resource' },
  { label: t('container.list.columns.runtimeStatus'), value: 'runtime_status' },
  { label: t('container.list.columns.createdAt'), value: 'created_at' },
  { label: t('container.list.columns.startedAt'), value: 'started_at' },
  { label: t('container.list.columns.restartPolicy'), value: 'restart_policy' },
  { label: t('container.list.columns.imageId'), value: 'image_id' },
  { label: t('container.list.columns.labels'), value: 'labels' },
  { label: t('container.list.columns.operation'), value: 'operation' },
]);
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
const selectedRows = computed(() => {
  const selectedKeySet = new Set(selectedRowKeys.value.map(String));
  return rows.value.filter((row) => selectedKeySet.has(row.id));
});

onMounted(() => {
  void refreshContainers();
});

useCurrentTabRefresh(async () => {
  await refreshContainers();
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

watch(
  () => filters.orchestrator,
  (nextOrchestrator) => {
    if (nextOrchestrator === 'all') {
      filters.sourceScopeKind = 'all';
      filters.sourceScope = '';
      return;
    }
    if (availableSourceScopeKinds.value.includes(filters.sourceScopeKind as ContainerListSourceScopeKind)) {
      return;
    }
    filters.sourceScopeKind = defaultSourceScopeKind(nextOrchestrator);
    filters.sourceScope = '';
  },
);

watch(
  () => filters.sourceScopeKind,
  (nextKind) => {
    if (nextKind !== 'all') {
      return;
    }
    filters.sourceScope = '';
  },
);

async function refreshContainers() {
  loading.value = true;
  listError.value = { title: '', hint: '' };
  try {
    const payload = await getContainers(buildListQuery());
    rows.value = payload.items;
    runtime.value = payload.runtime;
    listSummary.value = payload.summary;
    listTotal.value = payload.total;
    pruneSelectedRows();
  } catch (error) {
    rows.value = [];
    runtime.value = null;
    listSummary.value = null;
    listTotal.value = 0;
    listError.value = resolveListError(error);
    logger.error('failed to fetch containers', error);
  } finally {
    loading.value = false;
  }
}

function pruneSelectedRows() {
  const availableIds = new Set(rows.value.map((row) => row.id));
  selectedRowKeys.value = selectedRowKeys.value.filter((key) => availableIds.has(String(key)));
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
  filters.sourceScope = filters.sourceScope.trim();
  syncPendingSourceScopeFilter();
  requestFirstPage();
}

function resetFilters() {
  filters.keyword = '';
  filters.orchestrator = 'all';
  filters.sourceScopeKind = 'all';
  filters.sourceScope = '';
  filters.status = 'all';
  filters.health = 'all';
  pendingSourceScopeFilter.value = null;
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
  return {
    limit: pagination.pageSize,
    offset: (pagination.current - 1) * pagination.pageSize,
    keyword: filters.keyword.trim() || undefined,
    orchestrator: filters.orchestrator === 'all' ? undefined : filters.orchestrator,
    state: filters.status === 'all' ? undefined : filters.status,
    health: filters.health === 'all' ? undefined : filters.health,
    ...buildSourceScopeQuery(),
  };
}

function buildSourceScopeQuery(): ContainerListSourceScopeQuery {
  const sourceScopeFilter = pendingSourceScopeFilter.value;
  if (!sourceScopeFilter) {
    return {};
  }

  return {
    source_scope_kind: sourceScopeFilter.kind,
    source_scope: sourceScopeFilter.value,
  };
}

function syncPendingSourceScopeFilter() {
  pendingSourceScopeFilter.value =
    filters.sourceScopeKind !== 'all' && filters.sourceScope
      ? {
          kind: filters.sourceScopeKind,
          value: filters.sourceScope,
        }
      : null;
}

function defaultSourceScopeKind(orchestrator: ContainerOrchestratorType | 'all'): ContainerListSourceScopeKind | 'all' {
  if (orchestrator === 'compose') {
    return 'compose_project';
  }
  if (orchestrator === 'swarm') {
    return 'swarm_stack';
  }
  if (orchestrator === 'kubernetes') {
    return 'kubernetes_namespace';
  }
  return 'all';
}

function openDetail(row: ContainerSummaryRecord) {
  void navigateToDetail(row, 'overview');
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

function moreRowActions(row: ContainerSummaryRecord) {
  const actions: Array<{
    disabled?: boolean;
    fallbackLabel: string;
    label: string;
    testId: string;
    value: RowAction;
  }> = [];

  actions.push({
    fallbackLabel: t('container.list.actions.copyId'),
    label: 'container.list.actions.copyId',
    testId: 'container-action-copy-id',
    value: 'copy-id',
  });

  actions.push({
    fallbackLabel: t('container.list.actions.inspect'),
    label: 'container.list.actions.inspect',
    testId: 'container-action-inspect',
    value: 'inspect',
  });

  actions.push({
    fallbackLabel: t('container.list.actions.viewMounts'),
    label: 'container.list.actions.viewMounts',
    testId: 'container-action-view-mounts',
    value: 'view-mounts',
  });

  actions.push({
    fallbackLabel: t('container.list.actions.viewNetworks'),
    label: 'container.list.actions.viewNetworks',
    testId: 'container-action-view-networks',
    value: 'view-networks',
  });

  actions.push({
    fallbackLabel: t('container.list.actions.viewEnvironment'),
    label: 'container.list.actions.viewEnvironment',
    testId: 'container-action-view-env',
    value: 'view-env',
  });

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

function moreRowActionOptions(row: ContainerSummaryRecord): DropdownOption[] {
  return moreRowActions(row).map((action) => ({
    content: action.fallbackLabel,
    disabled: action.disabled,
    theme: action.value === 'remove' ? 'error' : 'default',
    title: action.disabled ? t('container.list.actions.dangerousDisabled') : undefined,
    testId: action.testId,
    value: action.value,
  }));
}

function handleMoreRowAction(
  payload: DropdownActionValue,
  context: DropdownActionContext | undefined,
  row: ContainerSummaryRecord,
) {
  context?.e?.stopPropagation();

  const action = typeof payload === 'object' && payload ? payload.value : payload;
  if (typeof action === 'string') {
    handleRowAction(action, row);
  }
}

function handleRowAction(action: string, row: ContainerSummaryRecord) {
  if (action === 'copy-id') {
    void copyContainerId(row);
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

async function performDangerousAction(row: ContainerSummaryRecord, action: DangerousContainerAction) {
  if (isDangerousActionDisabled(row, action)) {
    MessagePlugin.warning(t('container.list.actions.dangerousDisabled'));
    return;
  }

  const force = action === 'remove' ? await confirmRemoveAction(row) : await confirmRuntimeAction(row, action);
  if (force === undefined) return;

  await executeDangerousAction(row, action, force);
}

function confirmRuntimeAction(row: ContainerSummaryRecord, action: Exclude<DangerousContainerAction, 'remove'>) {
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
          h('p', t(actionConfirmKey(action), { name: displayName(row) })),
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
}

function confirmRemoveAction(row: ContainerSummaryRecord) {
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
              ? t('container.list.actions.confirmRemoveRunning', { name: displayName(row) })
              : t('container.list.actions.confirmRemove', { name: displayName(row) }),
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
}

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
}

function handleSelectChange(rowKeys: Array<string | number>) {
  selectedRowKeys.value = rowKeys.filter((key) => rows.value.some((row) => row.id === String(key)));
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
    title: buildDetailTabTitle(displayName(row)),
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

function displayName(row: ContainerSummaryRecord) {
  return row.name || row.names?.[0] || row.id;
}

function buildDetailTabTitle(name: string): LocalizedTitle {
  const baseTitle = localizeRouteTitleKey('container.detail.title');
  return {
    [LOCALE.ZH_CN]: `${baseTitle[LOCALE.ZH_CN]} - ${name}`,
    [LOCALE.EN_US]: `${baseTitle[LOCALE.EN_US]} - ${name}`,
  };
}

function shortContainerId(id: string) {
  return id.length > 12 ? id.slice(0, 12) : id;
}

function formatPorts(ports: ContainerPort[]) {
  return ports.map((port) => {
    const target = `${port.private_port}/${port.type}`;
    if (port.public_port === undefined) {
      return target;
    }
    return `${port.ip ? `${port.ip}:` : ''}${port.public_port}->${target}`;
  });
}

function visiblePortLabels(row: ContainerSummaryRecord) {
  return formatPorts(row.ports).slice(0, CONTAINER_PORT_VISIBLE_LIMIT);
}

function hiddenPortLabels(row: ContainerSummaryRecord) {
  return formatPorts(row.ports).slice(CONTAINER_PORT_VISIBLE_LIMIT);
}

function labelSummary(row: ContainerSummaryRecord) {
  const count = Object.keys(row.labels ?? {}).length;
  return count ? t('container.list.labelCount', { count }) : '-';
}

function resourceSummary(row: ContainerSummaryRecord) {
  if (!isResourceStatsAvailable(row)) {
    return resourceUnavailableSummary(row);
  }

  const resource = row.resource;
  const cpu = resource?.cpu_percent === undefined ? '-' : `${resource.cpu_percent.toFixed(1)}%`;
  const memory = resource?.memory_percent === undefined ? '-' : `${resource.memory_percent.toFixed(1)}%`;
  return `${cpu} / ${memory}`;
}

function cpuMetric(row: ContainerSummaryRecord): ResourceMetric {
  if (!isResourceStatsAvailable(row) || row.resource?.cpu_percent === undefined) {
    return {
      available: false,
      percentage: 0,
      tooltip: resourceUnavailableSummary(row),
      value: t('container.list.stats.notCollected'),
    };
  }

  const value = `${row.resource.cpu_percent.toFixed(1)}%`;
  return {
    available: true,
    percentage: clampPercentage(row.resource.cpu_percent),
    tooltip: t('container.list.stats.cpuTooltip', { percent: value }),
    value,
  };
}

function memoryMetric(row: ContainerSummaryRecord): ResourceMetric {
  if (!isResourceStatsAvailable(row) || row.resource?.memory_percent === undefined) {
    return {
      available: false,
      percentage: 0,
      tooltip: resourceUnavailableSummary(row),
      value: t('container.list.stats.notCollected'),
    };
  }

  const usage = formatBytes(row.resource.memory_usage_bytes);
  const limit = formatBytes(row.resource.memory_limit_bytes);
  const percent = `${row.resource.memory_percent.toFixed(1)}%`;

  return {
    available: true,
    percentage: clampPercentage(row.resource.memory_percent),
    tooltip: t('container.list.stats.memoryTooltip', {
      limit: limit || '-',
      percent,
      usage: usage || '-',
    }),
    value: usage || '-',
  };
}

function clampPercentage(value: number) {
  return Math.min(100, Math.max(0, Number.isFinite(value) ? value : 0));
}

function isResourceStatsAvailable(row: ContainerSummaryRecord) {
  if (row.resource?.stats_available !== undefined) {
    return row.resource.stats_available;
  }

  return Boolean(row.resource?.available);
}

function resourceUnavailableSummary(row: ContainerSummaryRecord) {
  const reason = row.resource?.stats_error_message || row.resource?.stats_error_key || row.resource?.unavailable_reason;
  return reason?.trim() || t('container.list.resourceUnavailable');
}

function formatBytes(value?: number) {
  if (value === undefined) {
    return '';
  }

  return `${(value / BYTES_PER_MIB).toFixed(value >= BYTES_PER_MIB ? 1 : 2)} MiB`;
}

function formatTime(value?: string | null) {
  return formatLocaleDateTime(value, locale);
}

function stateLabel(state: ContainerState) {
  return t(`container.list.states.${state}`);
}

function readOrchestratorType(row: ContainerSummaryRecord): ContainerOrchestratorType {
  return row.orchestrator?.type || 'standalone';
}

function orchestratorActionLevel(row: ContainerSummaryRecord): ContainerActionLevel {
  if (row.orchestrator?.action_level) {
    return row.orchestrator.action_level;
  }

  return row.can_start || row.can_stop || row.can_restart || row.can_remove ? 'allow' : 'readonly';
}

function orchestratorLabel(type: ContainerOrchestratorType) {
  return t(`container.list.orchestrators.${type}`);
}

function orchestratorTheme(row: ContainerSummaryRecord) {
  const type = readOrchestratorType(row);
  if (type === 'standalone') return 'success';
  if (type === 'compose') return 'warning';
  if (type === 'unknown') return 'danger';
  return 'default';
}

function orchestratorSummary(row: ContainerSummaryRecord) {
  const group = sourceGroupFilter(row);
  if (group) {
    return group.value;
  }

  const member = sourceMemberFilter(row);
  if (member) {
    return member.value;
  }

  return row.orchestrator?.display_name || t('container.list.sourceUnknownSummary');
}

function sourceGroupFilter(row: ContainerSummaryRecord): SourceQuickFilterValue | null {
  return toQuickFilterValue(
    row.orchestrator?.group_scope_kind || legacyGroupScopeKind(row),
    row.orchestrator?.group_value || row.orchestrator?.group_display_name || legacyGroupScopeValue(row),
  );
}

function sourceMemberFilter(row: ContainerSummaryRecord): SourceQuickFilterValue | null {
  return toQuickFilterValue(
    row.orchestrator?.member_scope_kind || legacyMemberScopeKind(row),
    row.orchestrator?.member_value || row.orchestrator?.member_display_name || legacyMemberScopeValue(row),
  );
}

function toQuickFilterValue(
  kind: ContainerSourceGroupKind | ContainerSourceMemberKind | null | undefined,
  value?: string | null,
): SourceQuickFilterValue | null {
  const normalizedValue = value?.trim();
  if (!kind || !normalizedValue) {
    return null;
  }

  return {
    kind,
    value: normalizedValue,
  };
}

function sourceGroupLabel(row: ContainerSummaryRecord) {
  const group = sourceGroupFilter(row);
  return group ? t(`container.list.sourceKinds.${group.kind}`) : '';
}

function sourceMemberLabel(row: ContainerSummaryRecord) {
  const member = sourceMemberFilter(row);
  return member ? t(`container.list.sourceKinds.${member.kind}`) : '';
}

function applySourceQuickFilter(row: ContainerSummaryRecord, target: SourceQuickFilterTarget) {
  const sourceFilter = target === 'group' ? sourceGroupFilter(row) : sourceMemberFilter(row);
  if (!sourceFilter) {
    return;
  }

  filters.orchestrator = readOrchestratorType(row);
  filters.keyword = '';
  filters.sourceScopeKind = sourceFilter.kind;
  filters.sourceScope = sourceFilter.value;
  pendingSourceScopeFilter.value = sourceFilter;
  requestFirstPage();
}

function legacyGroupScopeKind(row: ContainerSummaryRecord): ContainerSourceGroupKind | null {
  const orchestratorType = readOrchestratorType(row);
  if (orchestratorType === 'compose' && (row.orchestrator?.project || row.compose_project)) {
    return 'compose_project';
  }
  if (orchestratorType === 'swarm' && row.orchestrator?.stack) {
    return 'swarm_stack';
  }
  if (orchestratorType === 'kubernetes' && row.orchestrator?.namespace) {
    return 'kubernetes_namespace';
  }
  return null;
}

function legacyGroupScopeValue(row: ContainerSummaryRecord): string | null | undefined {
  const orchestratorType = readOrchestratorType(row);
  if (orchestratorType === 'compose') {
    return row.orchestrator?.project || row.compose_project;
  }
  if (orchestratorType === 'swarm') {
    return row.orchestrator?.stack;
  }
  if (orchestratorType === 'kubernetes') {
    return row.orchestrator?.namespace;
  }
  return null;
}

function legacyMemberScopeKind(row: ContainerSummaryRecord): ContainerSourceMemberKind | null {
  const orchestratorType = readOrchestratorType(row);
  if (orchestratorType === 'compose' && (row.orchestrator?.service || row.compose_service)) {
    return 'compose_service';
  }
  if (orchestratorType === 'swarm' && row.orchestrator?.task) {
    return 'swarm_task';
  }
  if (orchestratorType === 'kubernetes' && row.orchestrator?.pod) {
    return 'kubernetes_pod';
  }
  return null;
}

function legacyMemberScopeValue(row: ContainerSummaryRecord): string | null | undefined {
  const orchestratorType = readOrchestratorType(row);
  if (orchestratorType === 'compose') {
    return row.orchestrator?.service || row.compose_service;
  }
  if (orchestratorType === 'swarm') {
    return row.orchestrator?.task;
  }
  if (orchestratorType === 'kubernetes') {
    return row.orchestrator?.pod;
  }
  return null;
}

function rowActionRiskText(row: ContainerSummaryRecord) {
  return orchestratorActionLevel(row) === 'warn'
    ? t('container.list.actions.sourceRisk', { source: orchestratorLabel(readOrchestratorType(row)) })
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
  return !isDangerousActionDisabled(row, action) && (row.orchestrator?.batch_action_allowed ?? true);
}

function healthLabel(health?: ContainerHealth | null) {
  return t(`container.list.health.${health || 'unavailable'}`);
}

function shouldShowHealthTag(health?: ContainerHealth | null) {
  return health === 'healthy' || health === 'unhealthy' || health === 'starting';
}

function healthTheme(health?: ContainerHealth | null) {
  if (health === 'healthy') return 'success';
  if (health === 'unhealthy') return 'danger';
  if (health === 'starting') return 'warning';
  return 'default';
}

function stateTheme(state: ContainerState) {
  if (state === 'running') return 'success';
  if (state === 'created' || state === 'paused' || state === 'restarting') return 'warning';
  if (state === 'dead') return 'danger';
  return 'default';
}

function toggleTableDensity() {
  tableDensity.value = tableDensity.value === 'medium' ? 'small' : 'medium';
}

function loadVisibleColumnKeys() {
  if (typeof window === 'undefined') {
    return [...DEFAULT_VISIBLE_COLUMNS];
  }

  try {
    const stored = window.localStorage.getItem(CONTAINER_COLUMN_STORAGE_KEY);
    if (!stored) {
      return [...DEFAULT_VISIBLE_COLUMNS];
    }
    const parsed = JSON.parse(stored);
    if (!Array.isArray(parsed)) {
      return [...DEFAULT_VISIBLE_COLUMNS];
    }

    const normalizedKeys = normalizeVisibleColumnKeys(parsed);
    persistVisibleColumnKeys(normalizedKeys);
    return normalizedKeys;
  } catch {
    return [...DEFAULT_VISIBLE_COLUMNS];
  }
}

function persistVisibleColumnKeys(keys: string[]) {
  if (typeof window === 'undefined') {
    return;
  }

  try {
    window.localStorage.setItem(CONTAINER_COLUMN_STORAGE_KEY, JSON.stringify(keys));
  } catch {
    // Column settings are a convenience preference; list rendering must not depend on storage availability.
  }
}

function normalizeVisibleColumnKeys(keys: unknown[]) {
  const availableKeySet = new Set(ALL_COLUMN_KEYS);
  const nextKeys = new Set<string>();

  for (const key of keys) {
    if (typeof key === 'string' && availableKeySet.has(key)) {
      nextKeys.add(key);
    }
  }

  for (const key of ALWAYS_VISIBLE_COLUMNS) {
    nextKeys.add(key);
  }

  return ALL_COLUMN_KEYS.filter((key) => nextKeys.has(key));
}
</script>
<style scoped lang="less">
.container-page {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
  min-width: 0;
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
  display: inline-flex;
  gap: var(--graft-density-gap-8);
  justify-content: center;
  min-width: 0;
  white-space: nowrap;
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
  .container-actions {
    justify-content: flex-start;
  }
}
</style>
