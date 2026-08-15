<template>
  <div class="runtime-target-page-root">
    <advanced-query-list-page
      root-class="runtime-target-page"
      page-type="query-builder-list-detail"
      :title="t('runtimeTarget.list.title')"
      :description="t('runtimeTarget.list.description')"
      :error-message="errorMessage"
      :error-title="t('runtimeTarget.list.emptyTitle')"
      :loading="loading"
      :reload-label="t('runtimeTarget.list.reload')"
      :retry-label="t('runtimeTarget.list.reload')"
      :source="{ labelKey: 'runtimeTarget.list.eyebrow', fallback: t('runtimeTarget.list.eyebrow') }"
      @reload="load"
    >
      <template #actions>
        <template v-if="total > 0">
          <t-tooltip :content="t('runtimeTarget.list.discoverLocalDocker')" placement="bottom">
            <t-button
              class="runtime-target-discover-button"
              shape="square"
              theme="default"
              variant="outline"
              :aria-label="t('runtimeTarget.list.discoverLocalDocker')"
              :loading="discovering"
              data-testid="runtime-target-discover-local"
              @click="discoverLocal"
            >
              <template #icon><search-icon /></template>
            </t-button>
          </t-tooltip>
        </template>
      </template>
      <template #feedback-extra>
        <management-statistics-bar
          layout="summary"
          :items="statistics"
          :label="t('runtimeTarget.list.summary', { count: total })"
        />
      </template>
      <template #filters>
        <resource-query-panel
          v-model="queryState"
          :config="queryConfig"
          :loading="loading"
          @reset="resetQuery"
          @search="applyQuery"
        >
          <template #toolbar-after-search>
            <t-select v-model="filters.sort" class="runtime-target-sort" :options="sortOptions" @change="applySort" />
          </template>
          <template #toolbar-actions><saved-query-view-control :controller="savedViews" /></template>
        </resource-query-panel>
      </template>
      <template #table>
        <management-table-card>
          <template #toolbar>
            <table-view-toolbar
              :column-settings-label="t('runtimeTarget.list.columnSettings')"
              :refresh-label="t('runtimeTarget.list.reload')"
              :refresh-loading="loading"
              @column-settings="columnDrawerVisible = true"
              @refresh="load"
            />
          </template>
          <template v-if="selectedTargetIds.length" #batch>
            <management-batch-bar
              :selected-label="t('runtimeTarget.list.batchSelected', { count: selectedTargetIds.length })"
              :clear-label="t('runtimeTarget.list.cancel')"
              @clear="selectedTargetIds = []"
            >
              <t-button
                theme="primary"
                variant="outline"
                :loading="batchAuthorizationLoading"
                @click="openBatchAuthorization"
              >
                {{ t('runtimeTarget.list.batchAuthorize') }}
              </t-button>
              <t-button
                theme="danger"
                variant="outline"
                :loading="batchAuthorizationRevoking"
                @click="openBatchAuthorizationRevocation"
              >
                {{ t('runtimeTarget.list.batchRevoke') }}
              </t-button>
            </management-batch-bar>
          </template>
          <responsive-table entity-card-layout="adaptive" presentation="entity">
            <template #cards>
              <t-empty
                v-if="!items.length"
                :title="t('runtimeTarget.list.emptyTitle')"
                :description="t('runtimeTarget.list.emptyDescription')"
              >
                <template #action>
                  <t-button
                    theme="primary"
                    :loading="discovering"
                    data-testid="runtime-target-discover-local-empty"
                    @click="discoverLocal"
                  >
                    {{ t('runtimeTarget.list.discoverLocalDocker') }}
                  </t-button>
                </template>
              </t-empty>
              <article
                v-for="row in items"
                :key="row.id"
                class="runtime-target-card"
                :data-testid="`runtime-target-card-${row.id}`"
              >
                <header class="runtime-target-card__header">
                  <div class="runtime-target-card__identity">
                    <router-link :to="runtimeTargetDetailPath(row.id)">{{ row.displayName }}</router-link>
                    <span>{{ row.connection.endpoint }}</span>
                  </div>
                  <t-tag :theme="healthTheme(row)" variant="light">
                    {{ healthLabel(row) }}
                  </t-tag>
                </header>
                <div class="runtime-target-card__provider">
                  <span>{{ t('runtimeTarget.columns.provider') }}</span>
                  <strong>{{ row.runtime.provider }}</strong>
                </div>
                <dl class="runtime-target-card__metrics">
                  <div class="runtime-target-card__metric">
                    <dt>{{ t('runtimeTarget.metrics.workloads') }}</dt>
                    <dd>
                      <strong>{{ workloadValue(row) }}</strong>
                      <span v-if="row.resources.workloads.available">
                        {{ t('runtimeTarget.metrics.active') }} {{ row.resources.workloads.active }}
                      </span>
                    </dd>
                  </div>
                  <div v-for="metric in resourceMetrics(row)" :key="metric.key" class="runtime-target-card__metric">
                    <dt>{{ metric.label }}</dt>
                    <dd>
                      <realtime-resource-metric-cell
                        :available="metric.value.available"
                        :change="changeFor(row.id, metric.key)"
                        :percentage="metricPercentage(metric.value)"
                        :tooltip="metricText(metric.value)"
                        :value="metricValue(metric.value)"
                      />
                    </dd>
                  </div>
                </dl>
                <router-link class="runtime-target-card__detail" :to="runtimeTargetDetailPath(row.id)">
                  {{ t('runtimeTarget.list.viewDetail') }}
                </router-link>
              </article>
            </template>
            <t-table
              row-key="id"
              :data="items"
              :columns="tableColumns"
              :loading="loading"
              :selected-row-keys="selectedTargetIds"
              @select-change="handleTargetSelectionChange"
            >
              <template #empty>
                <t-empty
                  :title="t('runtimeTarget.list.emptyTitle')"
                  :description="t('runtimeTarget.list.emptyDescription')"
                >
                  <template #action>
                    <t-button
                      theme="primary"
                      :loading="discovering"
                      data-testid="runtime-target-discover-local-empty"
                      @click="discoverLocal"
                    >
                      {{ t('runtimeTarget.list.discoverLocalDocker') }}
                    </t-button>
                  </template>
                </t-empty>
              </template>
            </t-table>
          </responsive-table>
          <template #footer>
            <management-table-pagination :summary="t('runtimeTarget.list.summary', { count: total })">
              <t-pagination
                v-model:current="pagination.current"
                v-model:page-size="pagination.pageSize"
                :total="total"
                :total-content="false"
                :page-size-options="[10, 20, 50, 100]"
                :show-page-number="true"
                @change="load"
              />
            </management-table-pagination>
          </template>
        </management-table-card>
      </template>
      <template #detail>
        <advanced-query-column-drawer
          v-model:visible="columnDrawerVisible"
          v-model:selected-keys="visibleColumnKeys"
          :columns="columnOptions"
          :default-selected-keys="DEFAULT_VISIBLE_COLUMNS"
          :reset-label="t('runtimeTarget.list.resetColumns')"
          :title="t('runtimeTarget.list.columnSettings')"
        />
      </template>
    </advanced-query-list-page>
    <paged-multi-select
      v-model:visible="batchAuthorizationDialogVisible"
      v-model:current="batchCandidatePagination.current"
      v-model:keyword="batchCandidateSearch"
      v-model:page-size="batchCandidatePagination.pageSize"
      v-model:selection="batchUserSelection"
      :cancel-label="t('runtimeTarget.list.cancel')"
      :cell-slot-names="['authorizationState']"
      :columns="batchCandidateColumns"
      :confirm-label="t('runtimeTarget.list.batchAuthorize')"
      :confirm-loading="batchAuthorizationLoading"
      :empty-description="t('runtimeTarget.list.candidatesEmpty')"
      :empty-title="t('runtimeTarget.list.candidatesEmpty')"
      :error-message="batchAuthorizationError"
      :loading="batchCandidatesLoading"
      row-key="id"
      :rows="batchCandidates"
      :search="{
        placeholder: t('runtimeTarget.list.candidateSearchPlaceholder'),
        clearLabel: t('runtimeTarget.list.clearSearch'),
      }"
      :selected-count-label="(count) => t('runtimeTarget.list.selectedUsers', { count })"
      :title="t('runtimeTarget.list.batchAuthorizeTitle')"
      :total="batchCandidateTotal"
      :total-label="(count) => t('runtimeTarget.list.candidateTotal', { count })"
      @cancel="closeBatchAuthorization"
      @confirm="saveBatchAuthorization"
      @page-change="loadBatchCandidates"
      @search="searchBatchCandidates"
    >
      <template #authorizationState="{ row }">
        <t-tag :theme="candidateAuthorizationTheme(row.authorization_state)" variant="light">
          {{ candidateAuthorizationLabel(row.authorization_state) }}
        </t-tag>
      </template>
    </paged-multi-select>
    <paged-multi-select
      v-model:visible="batchAuthorizationRevokeDialogVisible"
      v-model:current="batchCandidatePagination.current"
      v-model:keyword="batchCandidateSearch"
      v-model:page-size="batchCandidatePagination.pageSize"
      v-model:selection="batchRevokeUserSelection"
      :cancel-label="t('runtimeTarget.list.cancel')"
      :cell-slot-names="['authorizationState']"
      :columns="batchRevokeCandidateColumns"
      :confirm-label="t('runtimeTarget.list.batchRevoke')"
      :confirm-loading="batchAuthorizationRevoking"
      :empty-description="t('runtimeTarget.list.candidatesEmpty')"
      :empty-title="t('runtimeTarget.list.candidatesEmpty')"
      :error-message="batchAuthorizationError"
      :loading="batchCandidatesLoading"
      row-key="id"
      :rows="batchCandidates"
      :search="{
        placeholder: t('runtimeTarget.list.candidateSearchPlaceholder'),
        clearLabel: t('runtimeTarget.list.clearSearch'),
      }"
      :selected-count-label="(count) => t('runtimeTarget.list.selectedUsers', { count })"
      :title="t('runtimeTarget.list.batchRevokeTitle')"
      :total="batchCandidateTotal"
      :total-label="(count) => t('runtimeTarget.list.candidateTotal', { count })"
      @cancel="closeBatchAuthorizationRevocation"
      @confirm="revokeBatchAuthorization"
      @page-change="loadBatchCandidates"
      @search="searchBatchCandidates"
    >
      <template #authorizationState="{ row }">
        <t-tag :theme="candidateAuthorizationTheme(row.authorization_state)" variant="light">
          {{ candidateAuthorizationLabel(row.authorization_state) }}
        </t-tag>
      </template>
    </paged-multi-select>
  </div>
</template>
<script setup lang="ts">
// 列表页负责运行目标的发现、列表状态与批量授权；详情数据和单目标授权由详情路由独立加载。
import { SearchIcon } from 'tdesign-icons-vue-next';
import type { PrimaryTableCol, TableProps } from 'tdesign-vue-next';
import { DialogPlugin } from 'tdesign-vue-next/es/dialog';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import {
  computed,
  h,
  nextTick,
  onActivated,
  onDeactivated,
  onMounted,
  onUnmounted,
  reactive,
  ref,
  resolveComponent,
  watch,
} from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { RUNTIME_TARGET_REALTIME_TOPIC } from '@/contracts/generated/modules/runtime-target';
import {
  ManagementBatchBar,
  ManagementStatisticsBar,
  ManagementTableCard,
  ManagementTablePagination,
  TableViewToolbar,
} from '@/shared/components/management';
import { RealtimeResourceMetricCell } from '@/shared/components/metrics';
import {
  AdvancedQueryColumnDrawer,
  AdvancedQueryListPage,
  applySavedQueryViewPresentation,
  normalizeSavedQueryView,
  type ResourceQueryConfig,
  ResourceQueryPanel,
  type ResourceQueryState,
  SavedQueryViewControl,
  useSavedQueryViews,
} from '@/shared/components/query-list';
import ResponsiveTable from '@/shared/components/responsive/ResponsiveTable.vue';
import { createExplicitSelection, type ExplicitSelection, PagedMultiSelect } from '@/shared/components/selection';
import { formatBytes } from '@/shared/observability';
import { openRealtimeTopicSocket, type RealtimeTopicSocketController } from '@/shared/realtime';

import {
  applyRuntimeTargetAssignmentBatch,
  deleteRuntimeTargetSavedView,
  discoverLocalDocker,
  getRuntimeTargetAssignmentCandidates,
  getRuntimeTargetAssignmentsForTargets,
  getRuntimeTargetSavedViews,
  listRuntimeTargetPage,
  postRuntimeTargetSavedView,
  putRuntimeTargetSavedView,
  type RuntimeTarget,
  type RuntimeTargetAssignmentCandidate,
  type RuntimeTargetUsageMetric,
} from '../../api/runtime-target';
import { runtimeTargetDetailPath } from '../../contract/paths';
import { parseRuntimeTargetSummaryPayload } from '../../contract/realtime';

const { t } = useI18n();
type Change = 'up' | 'down' | 'none';
type MetricChanges = Record<'cpu' | 'memory' | 'storage', Change>;
const CHANGE_HIGHLIGHT_MS = 800;
const route = useRoute();
const router = useRouter();
const loading = ref(false);
const discovering = ref(false);
const errorMessage = ref('');
const items = ref<RuntimeTarget[]>([]);
const total = ref(0);
const summary = ref({ total: 0, healthy: 0, unavailable: 0 });
const pagination = reactive({ current: 1, pageSize: 10 });
const selectedTargetIds = ref<number[]>([]);
const batchAuthorizationDialogVisible = ref(false);
const batchAuthorizationRevokeDialogVisible = ref(false);
const batchAuthorizationLoading = ref(false);
const batchAuthorizationRevoking = ref(false);
const batchAuthorizationError = ref('');
const batchCandidatesLoading = ref(false);
const batchCandidates = ref<BatchAuthorizationCandidate[]>([]);
const batchCandidateTotal = ref(0);
const batchCandidateSearch = ref('');
const batchCandidatePagination = reactive({ current: 1, pageSize: 20 });
// 只允许当前对话框会话提交异步候选结果，关闭后使仍在途的请求失效。
let batchCandidateRequestVersion = 0;
const batchUserSelection = ref<ExplicitSelection<number>>(createExplicitSelection());
const batchRevokeUserSelection = ref<ExplicitSelection<number>>(createExplicitSelection());
const batchTargetAssignments = ref<Map<number, Set<number>>>(new Map());
const DEFAULT_VISIBLE_COLUMNS = ['displayName', 'provider', 'health', 'workloads', 'cpu', 'memory', 'storage'];
type RuntimeTargetFilters = {
  keyword: string;
  provider: '' | 'docker';
  connectionKind: '' | 'unix_socket';
  health: '' | 'healthy' | 'unavailable';
  sort: 'display_name:asc' | 'display_name:desc' | 'provider:asc' | 'provider:desc' | 'health:asc' | 'health:desc';
};
type RuntimeTargetSavedQueryState = {
  keyword?: string;
  provider?: 'docker';
  connection_kind?: 'unix_socket';
  health?: 'healthy' | 'unavailable';
  sort?: RuntimeTargetFilters['sort'];
};
type RuntimeTargetSavedViewState = {
  pageSize: number;
  queryState: RuntimeTargetSavedQueryState;
  visibleColumns: string[];
};
const filters = reactive<RuntimeTargetFilters>(createDefaultFilters());
const visibleColumnKeys = ref([...DEFAULT_VISIBLE_COLUMNS]);
const columnDrawerVisible = ref(false);
const applyingRoute = ref(false);
const statistics = computed(() => [
  { label: t('runtimeTarget.list.targetCount'), value: summary.value.total },
  {
    label: t('runtimeTarget.status.healthy'),
    marker: '🟢',
    value: summary.value.healthy,
  },
  {
    label: t('runtimeTarget.status.unavailable'),
    marker: '🔴',
    value: summary.value.unavailable,
  },
]);
const active = ref(false);
let realtimeController: RealtimeTopicSocketController | null = null;
const changes = ref<Record<number, MetricChanges>>({});
const changeExpiryByID = new Map<number, number>();
let changeExpiryTimer: number | null = null;

const sortOptions = computed(() => [
  { label: t('runtimeTarget.sort.nameAsc'), value: 'display_name:asc' },
  { label: t('runtimeTarget.sort.nameDesc'), value: 'display_name:desc' },
  { label: t('runtimeTarget.sort.providerAsc'), value: 'provider:asc' },
  { label: t('runtimeTarget.sort.providerDesc'), value: 'provider:desc' },
  { label: t('runtimeTarget.sort.healthAsc'), value: 'health:asc' },
  { label: t('runtimeTarget.sort.healthDesc'), value: 'health:desc' },
]);
const queryConfig = computed<ResourceQueryConfig>(() => ({
  resource: 'runtime-target.list',
  placeholder: t('runtimeTarget.list.searchPlaceholder'),
  filters: [
    {
      key: 'provider',
      label: t('runtimeTarget.filters.provider'),
      type: 'select',
      options: [{ value: 'docker', label: t('runtimeTarget.providers.docker') }],
    },
    {
      key: 'connectionKind',
      label: t('runtimeTarget.filters.connectionKind'),
      type: 'select',
      options: [{ value: 'unix_socket', label: t('runtimeTarget.connectionKinds.unixSocket') }],
    },
    {
      key: 'health',
      label: t('runtimeTarget.columns.health'),
      type: 'select',
      options: [
        { value: 'healthy', label: t('runtimeTarget.status.healthy') },
        { value: 'unavailable', label: t('runtimeTarget.status.unavailable') },
      ],
    },
  ],
  quickFilters: [
    { key: 'all', label: t('runtimeTarget.presets.all'), patch: { provider: '', connectionKind: '', health: '' } },
    { key: 'healthy', label: t('runtimeTarget.status.healthy'), patch: { health: 'healthy' } },
    { key: 'unavailable', label: t('runtimeTarget.status.unavailable'), patch: { health: 'unavailable' } },
  ],
}));
const queryState = computed<ResourceQueryState>({
  get: () => ({
    keyword: filters.keyword,
    filters: { provider: filters.provider, connectionKind: filters.connectionKind, health: filters.health },
    page: pagination.current,
    pageSize: pagination.pageSize,
  }),
  set: (value) => {
    filters.keyword = value.keyword;
    filters.provider = value.filters.provider === 'docker' ? 'docker' : '';
    filters.connectionKind = value.filters.connectionKind === 'unix_socket' ? 'unix_socket' : '';
    filters.health =
      value.filters.health === 'healthy' || value.filters.health === 'unavailable' ? value.filters.health : '';
    pagination.current = value.page;
    pagination.pageSize = value.pageSize;
  },
});
const columnOptions = computed(() => [
  { label: t('runtimeTarget.columns.name'), value: 'displayName' },
  { label: t('runtimeTarget.columns.provider'), value: 'provider' },
  { label: t('runtimeTarget.columns.health'), value: 'health' },
  { label: t('runtimeTarget.metrics.workloads'), value: 'workloads' },
  { label: t('runtimeTarget.metrics.cpu'), value: 'cpu' },
  { label: t('runtimeTarget.metrics.memory'), value: 'memory' },
  { label: t('runtimeTarget.metrics.storage'), value: 'storage' },
]);
type BatchAuthorizationState = 'all' | 'partial' | 'none';
type BatchAuthorizationCandidate = RuntimeTargetAssignmentCandidate & { authorization_state: BatchAuthorizationState };
function buildBatchCandidateColumns(disabledState: BatchAuthorizationState) {
  return [
    {
      colKey: 'row-select',
      type: 'multiple' as const,
      width: 48,
      checkProps: ({ row }: { row: unknown }) => ({
        disabled: (row as BatchAuthorizationCandidate).authorization_state === disabledState,
      }),
    },
    { colKey: 'display', title: t('runtimeTarget.list.candidateUser'), minWidth: 180 },
    { colKey: 'username', title: t('runtimeTarget.list.candidateUsername'), minWidth: 150 },
    { colKey: 'authorizationState', title: t('runtimeTarget.list.authorizationState'), width: 130 },
  ] satisfies TableProps['columns'];
}
const batchCandidateColumns = computed<TableProps['columns']>(() => buildBatchCandidateColumns('all'));
const batchRevokeCandidateColumns = computed<TableProps['columns']>(() => buildBatchCandidateColumns('none'));
const savedViews = useSavedQueryViews<RuntimeTargetSavedViewState, number>({
  adapter: {
    list: async () =>
      (await getRuntimeTargetSavedViews()).map((view) =>
        normalizeSavedQueryView<RuntimeTargetSavedQueryState, number>(view),
      ),
    create: async (input) =>
      normalizeSavedQueryView<RuntimeTargetSavedQueryState, number>(
        await postRuntimeTargetSavedView(toSavedViewInput(input)),
      ),
    update: async (id, input) =>
      normalizeSavedQueryView<RuntimeTargetSavedQueryState, number>(
        await putRuntimeTargetSavedView(id, toSavedViewInput(input)),
      ),
    remove: async (id) => {
      await deleteRuntimeTargetSavedView(id);
    },
  },
  applyView: async (view) => {
    applySavedState(view.state);
    await replaceRoute();
    await load();
  },
  onError: (_error, operation) =>
    MessagePlugin.error(
      t(operation === 'delete' ? 'runtimeTarget.list.savedViewDeleteError' : 'runtimeTarget.list.savedViewError'),
    ),
  serializeCurrentState: () => ({
    pageSize: pagination.pageSize,
    queryState: currentSavedQueryState(),
    visibleColumns: [...visibleColumnKeys.value],
  }),
});

function createDefaultFilters(): RuntimeTargetFilters {
  return { keyword: '', provider: '', connectionKind: '', health: '', sort: 'display_name:asc' };
}
function currentSavedQueryState(): RuntimeTargetSavedQueryState {
  return {
    ...(filters.keyword.trim() ? { keyword: filters.keyword.trim() } : {}),
    ...(filters.provider ? { provider: filters.provider } : {}),
    ...(filters.connectionKind ? { connection_kind: filters.connectionKind } : {}),
    ...(filters.health ? { health: filters.health } : {}),
    sort: filters.sort,
  };
}
function toSavedViewInput(input: { name: string; isDefault: boolean; state: RuntimeTargetSavedViewState }) {
  return {
    name: input.name,
    pageSize: input.state.pageSize,
    queryState: input.state.queryState,
    visibleColumns: input.state.visibleColumns,
    isDefault: input.isDefault,
  };
}
function applySavedState(state: RuntimeTargetSavedViewState) {
  const query = state.queryState;
  filters.keyword = query.keyword ?? '';
  filters.provider = query.provider ?? '';
  filters.connectionKind = query.connection_kind ?? '';
  filters.health = query.health ?? '';
  filters.sort = isSort(query.sort) ? query.sort : 'display_name:asc';
  applySavedQueryViewPresentation(state, { pagination, supportedColumns: DEFAULT_VISIBLE_COLUMNS, visibleColumnKeys });
}
function applyQuery(value: ResourceQueryState) {
  queryState.value = value;
  pagination.current = 1;
  void load();
}
function applySort() {
  if (applyingRoute.value) return;
  pagination.current = 1;
  void load();
}
function resetQuery() {
  Object.assign(filters, createDefaultFilters());
  pagination.current = 1;
  void load();
}
function isSort(value: unknown): value is RuntimeTargetFilters['sort'] {
  return typeof value === 'string' && sortOptions.value.some((option) => option.value === value);
}
function stringQuery(value: unknown) {
  return typeof value === 'string' ? value : '';
}
function routeQuery() {
  return route?.query ?? {};
}
function hasExplicitRouteState() {
  const query = routeQuery();
  return ['keyword', 'provider', 'connection_kind', 'health', 'sort', 'page', 'page_size', 'columns'].some(
    (key) => query[key] !== undefined,
  );
}
function hydrateFromRoute() {
  applyingRoute.value = true;
  const query = routeQuery();
  filters.keyword = stringQuery(query.keyword);
  filters.provider = stringQuery(query.provider) === 'docker' ? 'docker' : '';
  filters.connectionKind = stringQuery(query.connection_kind) === 'unix_socket' ? 'unix_socket' : '';
  const routeHealth = stringQuery(query.health);
  filters.health = routeHealth === 'healthy' || routeHealth === 'unavailable' ? routeHealth : '';
  const routeSort = stringQuery(query.sort);
  filters.sort = isSort(routeSort) ? routeSort : 'display_name:asc';
  pagination.current = Math.max(1, Number(stringQuery(query.page)) || 1);
  pagination.pageSize = [10, 20, 50, 100].includes(Number(stringQuery(query.page_size)))
    ? Number(stringQuery(query.page_size))
    : 10;
  const columns = stringQuery(query.columns)
    .split(',')
    .filter((key) => DEFAULT_VISIBLE_COLUMNS.includes(key));
  visibleColumnKeys.value = columns.length ? columns : [...DEFAULT_VISIBLE_COLUMNS];
  void nextTick(() => {
    applyingRoute.value = false;
  });
}
async function replaceRoute() {
  if (!router) return;
  await router.replace({
    query: {
      ...currentSavedQueryState(),
      page: String(pagination.current),
      page_size: String(pagination.pageSize),
      columns: visibleColumnKeys.value.join(','),
    },
  });
}

function metricText(metric: RuntimeTargetUsageMetric) {
  if (!metric.available) return t('runtimeTarget.metrics.unavailable');
  const percent = `${metric.usagePercent.toFixed(1)}%`;
  return metric.totalBytes > 0
    ? `${percent} · ${formatBytes(metric.usedBytes)} / ${formatBytes(metric.totalBytes)}`
    : percent;
}
function metricPercentage(metric: RuntimeTargetUsageMetric) {
  return Math.max(0, Math.min(100, metric.usagePercent));
}
function metricValue(metric: RuntimeTargetUsageMetric) {
  return metric.available ? `${metricPercentage(metric).toFixed(1)}%` : t('runtimeTarget.metrics.unavailable');
}
function changeFor(id: number, metric: keyof MetricChanges) {
  return changes.value[id]?.[metric] ?? 'none';
}
function metricCell(id: number, metric: keyof MetricChanges, value: RuntimeTargetUsageMetric) {
  const percentage = metricPercentage(value);
  return h(RealtimeResourceMetricCell, {
    available: value.available,
    change: changeFor(id, metric),
    percentage,
    tooltip: value.available
      ? metricText(value)
      : value.unavailableReason || t('runtimeTarget.metrics.unavailableHint'),
    value: metricValue(value),
  });
}
function healthLabel(row: RuntimeTarget) {
  return row.health.status === 'healthy' ? t('runtimeTarget.status.healthy') : t('runtimeTarget.status.unavailable');
}
function healthTheme(row: RuntimeTarget) {
  return row.health.status === 'healthy' ? 'success' : 'danger';
}
function workloadValue(row: RuntimeTarget) {
  return row.resources.workloads.available ? row.resources.workloads.total : t('runtimeTarget.metrics.unavailable');
}
function resourceMetrics(row: RuntimeTarget) {
  return [
    { key: 'cpu' as const, label: t('runtimeTarget.metrics.cpu'), value: row.resources.cpu },
    { key: 'memory' as const, label: t('runtimeTarget.metrics.memory'), value: row.resources.memory },
    { key: 'storage' as const, label: t('runtimeTarget.metrics.storage'), value: row.resources.storage },
  ];
}
function workloadCell(row: RuntimeTarget) {
  const workloads = row.resources.workloads;
  if (!workloads.available) return t('runtimeTarget.metrics.unavailable');
  return h('div', { class: 'runtime-target-counts' }, [
    h('strong', workloads.total),
    h('span', [h('small', t('runtimeTarget.metrics.active')), h('b', workloads.active)]),
  ]);
}
const columns = computed<PrimaryTableCol<RuntimeTarget>[]>(() => [
  {
    colKey: 'displayName',
    title: t('runtimeTarget.columns.name'),
    minWidth: 230,
    cell: (_h, { row }) =>
      h('div', { class: 'runtime-target-identity' }, [
        h(resolveComponent('router-link'), { to: runtimeTargetDetailPath(row.id) }, () => row.displayName),
        h('small', row.connection.endpoint),
      ]),
  },
  {
    colKey: 'provider',
    title: t('runtimeTarget.columns.provider'),
    width: 140,
    cell: (_h, { row }) => row.runtime.provider,
  },
  {
    colKey: 'health',
    title: t('runtimeTarget.columns.health'),
    width: 120,
    cell: (_h, { row }) =>
      h(
        resolveComponent('t-tag'),
        { theme: row.health.status === 'healthy' ? 'success' : 'danger', variant: 'light' },
        () => healthLabel(row),
      ),
  },
  {
    colKey: 'workloads',
    title: t('runtimeTarget.metrics.workloads'),
    width: 130,
    cell: (_h, { row }) => workloadCell(row),
  },
  {
    colKey: 'cpu',
    title: t('runtimeTarget.metrics.cpu'),
    width: 142,
    cell: (_h, { row }) => metricCell(row.id, 'cpu', row.resources.cpu),
  },
  {
    colKey: 'memory',
    title: t('runtimeTarget.metrics.memory'),
    width: 142,
    cell: (_h, { row }) => metricCell(row.id, 'memory', row.resources.memory),
  },
  {
    colKey: 'storage',
    title: t('runtimeTarget.metrics.storage'),
    width: 142,
    cell: (_h, { row }) => metricCell(row.id, 'storage', row.resources.storage),
  },
]);
const tableColumns = computed(
  () =>
    [
      { colKey: 'row-select', type: 'multiple' as const, width: 48 },
      ...columns.value.filter((column) => visibleColumnKeys.value.includes(String(column.colKey))),
    ] as unknown as PrimaryTableCol[],
);

function handleTargetSelectionChange(keys: Array<string | number>) {
  selectedTargetIds.value = keys.map(Number).filter((id) => Number.isInteger(id) && id > 0);
}

function openBatchAuthorization() {
  batchCandidateSearch.value = '';
  batchCandidatePagination.current = 1;
  batchUserSelection.value = createExplicitSelection();
  batchAuthorizationError.value = '';
  batchAuthorizationDialogVisible.value = true;
  void loadBatchCandidates();
}

function closeBatchAuthorization() {
  if (batchAuthorizationLoading.value) return;
  batchCandidateRequestVersion += 1;
  batchAuthorizationDialogVisible.value = false;
  batchUserSelection.value = createExplicitSelection();
}

function openBatchAuthorizationRevocation() {
  batchCandidateSearch.value = '';
  batchCandidatePagination.current = 1;
  batchRevokeUserSelection.value = createExplicitSelection();
  batchAuthorizationError.value = '';
  batchAuthorizationRevokeDialogVisible.value = true;
  void loadBatchCandidates();
}

function closeBatchAuthorizationRevocation() {
  if (batchAuthorizationRevoking.value) return;
  batchCandidateRequestVersion += 1;
  batchAuthorizationRevokeDialogVisible.value = false;
  batchRevokeUserSelection.value = createExplicitSelection();
}

async function loadBatchCandidates() {
  const targetIds = [...selectedTargetIds.value];
  if (!targetIds.length) return;
  const requestVersion = ++batchCandidateRequestVersion;
  batchCandidatesLoading.value = true;
  batchAuthorizationError.value = '';
  try {
    const [candidatePage, assignments] = await Promise.all([
      getRuntimeTargetAssignmentCandidates(targetIds[0], {
        search: batchCandidateSearch.value.trim() || undefined,
        limit: batchCandidatePagination.pageSize,
        offset: (batchCandidatePagination.current - 1) * batchCandidatePagination.pageSize,
      }),
      getRuntimeTargetAssignmentsForTargets(targetIds),
    ]);
    if (
      requestVersion !== batchCandidateRequestVersion ||
      (!batchAuthorizationDialogVisible.value && !batchAuthorizationRevokeDialogVisible.value)
    ) {
      return;
    }
    batchTargetAssignments.value = assignments;
    batchCandidates.value = candidatePage.items.map((candidate) => {
      const assignedCount = targetIds.reduce(
        (count, targetId) => count + (assignments.get(targetId)?.has(candidate.id) ? 1 : 0),
        0,
      );
      return {
        ...candidate,
        authorization_state: assignedCount === targetIds.length ? 'all' : assignedCount > 0 ? 'partial' : 'none',
      };
    });
    batchCandidateTotal.value = candidatePage.total;
  } catch {
    if (requestVersion === batchCandidateRequestVersion) {
      batchAuthorizationError.value = t('runtimeTarget.list.authorizationLoadError');
    }
  } finally {
    if (requestVersion === batchCandidateRequestVersion) {
      batchCandidatesLoading.value = false;
    }
  }
}

function searchBatchCandidates() {
  batchCandidatePagination.current = 1;
  void loadBatchCandidates();
}

async function saveBatchAuthorization() {
  const targetIds = [...selectedTargetIds.value];
  const userIds = Array.from(batchUserSelection.value.selectedIds).map(Number);
  if (!targetIds.length || !userIds.length) return;
  batchAuthorizationLoading.value = true;
  batchAuthorizationError.value = '';
  let saved = false;
  try {
    await applyRuntimeTargetAssignmentBatch(targetIds, userIds, 'grant');
    MessagePlugin.success(t('runtimeTarget.list.batchAuthorizeSuccess'));
    saved = true;
  } catch {
    batchAuthorizationError.value = t('runtimeTarget.list.batchAuthorizeError');
  } finally {
    batchAuthorizationLoading.value = false;
    if (saved) {
      closeBatchAuthorization();
      selectedTargetIds.value = [];
    }
  }
}

async function revokeBatchAuthorization() {
  const targetIds = [...selectedTargetIds.value];
  const userIds = Array.from(batchRevokeUserSelection.value.selectedIds).map(Number);
  if (!targetIds.length || !userIds.length) return;
  const containsPartialAuthorization = batchCandidates.value.some(
    (candidate) => userIds.includes(candidate.id) && candidate.authorization_state === 'partial',
  );
  const revoke = async () => {
    batchAuthorizationRevoking.value = true;
    batchAuthorizationError.value = '';
    let revoked = false;
    try {
      await applyRuntimeTargetAssignmentBatch(targetIds, userIds, 'revoke');
      MessagePlugin.success(t('runtimeTarget.list.batchRevokeSuccess'));
      revoked = true;
    } catch {
      batchAuthorizationError.value = t('runtimeTarget.list.batchRevokeError');
    } finally {
      batchAuthorizationRevoking.value = false;
      if (revoked) {
        closeBatchAuthorizationRevocation();
        selectedTargetIds.value = [];
      }
    }
  };

  if (!containsPartialAuthorization) {
    await revoke();
    return;
  }

  const dialog = DialogPlugin.confirm({
    header: t('runtimeTarget.list.batchRevokeConfirmTitle'),
    body: t('runtimeTarget.list.batchRevokePartialWarning'),
    confirmBtn: t('runtimeTarget.list.batchRevoke'),
    cancelBtn: t('runtimeTarget.list.cancel'),
    onConfirm: () => {
      dialog.destroy();
      void revoke();
    },
    onCancel: () => dialog.destroy(),
  });
}

function candidateAuthorizationLabel(state: BatchAuthorizationState) {
  if (state === 'all') return t('runtimeTarget.list.alreadyAuthorized');
  if (state === 'partial') return t('runtimeTarget.list.partiallyAuthorized');
  return t('runtimeTarget.list.notAuthorized');
}

function candidateAuthorizationTheme(state: BatchAuthorizationState) {
  if (state === 'all') return 'success';
  if (state === 'partial') return 'warning';
  return 'default';
}

function compare(previous: number, next: number): Change {
  return next > previous ? 'up' : next < previous ? 'down' : 'none';
}

function hasMetricChanged(previous: RuntimeTargetUsageMetric, next: RuntimeTargetUsageMetric) {
  return (
    previous.available !== next.available ||
    previous.usagePercent !== next.usagePercent ||
    previous.usedBytes !== next.usedBytes ||
    previous.totalBytes !== next.totalBytes ||
    previous.unavailableReason !== next.unavailableReason
  );
}

function hasRuntimeTargetChanged(previous: RuntimeTarget, next: RuntimeTarget) {
  return (
    previous.id !== next.id ||
    previous.displayName !== next.displayName ||
    previous.runtime.provider !== next.runtime.provider ||
    previous.runtime.type !== next.runtime.type ||
    previous.runtime.version !== next.runtime.version ||
    previous.runtime.apiVersion !== next.runtime.apiVersion ||
    previous.connection.endpoint !== next.connection.endpoint ||
    previous.connection.kind !== next.connection.kind ||
    previous.health.status !== next.health.status ||
    previous.health.lastCheckedAt !== next.health.lastCheckedAt ||
    previous.health.diagnostic !== next.health.diagnostic ||
    previous.resources.workloads.available !== next.resources.workloads.available ||
    previous.resources.workloads.total !== next.resources.workloads.total ||
    previous.resources.workloads.active !== next.resources.workloads.active ||
    previous.resources.workloads.unavailableReason !== next.resources.workloads.unavailableReason ||
    hasMetricChanged(previous.resources.cpu, next.resources.cpu) ||
    hasMetricChanged(previous.resources.memory, next.resources.memory) ||
    hasMetricChanged(previous.resources.storage, next.resources.storage)
  );
}

function clearChangeExpiryScheduler() {
  if (changeExpiryTimer !== null) {
    window.clearTimeout(changeExpiryTimer);
    changeExpiryTimer = null;
  }
  changeExpiryByID.clear();
}

// 合并同一快照中的多个高亮状态，避免每个目标分别触发一次 Vue 更新。
function scheduleChangeExpiry() {
  if (changeExpiryTimer !== null || changeExpiryByID.size === 0) {
    return;
  }

  const nextExpiry = Math.min(...changeExpiryByID.values());
  const now = Date.now();
  changeExpiryTimer = window.setTimeout(
    () => {
      changeExpiryTimer = null;
      const currentTime = Date.now();
      const nextChanges = { ...changes.value };
      let changed = false;

      changeExpiryByID.forEach((expiresAt, id) => {
        if (expiresAt > currentTime) {
          return;
        }

        changeExpiryByID.delete(id);
        if (id in nextChanges) {
          delete nextChanges[id];
          changed = true;
        }
      });

      if (changed) {
        changes.value = nextChanges;
      }
      scheduleChangeExpiry();
    },
    Math.max(0, nextExpiry - now),
  );
}

function markChanged(id: number, nextChanges: MetricChanges) {
  changes.value = { ...changes.value, [id]: nextChanges };
  changeExpiryByID.set(id, Date.now() + CHANGE_HIGHLIGHT_MS);
  scheduleChangeExpiry();
}

function reconcileRealtimePage(nextItems: RuntimeTarget[]) {
  const currentByID = new Map(items.value.map((item) => [item.id, item]));
  const nextPage = nextItems.map((next) => {
    const current = currentByID.get(next.id);
    if (!current) return next;

    const nextChanges: MetricChanges = {
      cpu: compare(current.resources.cpu.usagePercent, next.resources.cpu.usagePercent),
      memory: compare(current.resources.memory.usagePercent, next.resources.memory.usagePercent),
      storage: compare(current.resources.storage.usagePercent, next.resources.storage.usagePercent),
    };
    if (Object.values(nextChanges).some((value) => value !== 'none')) {
      markChanged(current.id, nextChanges);
    }
    return hasRuntimeTargetChanged(current, next) ? next : current;
  });

  const samePage =
    nextPage.length === items.value.length && nextPage.every((item, index) => item === items.value[index]);
  if (!samePage) {
    items.value = nextPage;
  }

  const activeIDs = new Set(nextItems.map((item) => item.id));
  const staleChangeIDs = Object.keys(changes.value).filter((id) => !activeIDs.has(Number(id)));
  if (staleChangeIDs.length > 0) {
    const retainedChanges = { ...changes.value };
    staleChangeIDs.forEach((id) => delete retainedChanges[Number(id)]);
    changes.value = retainedChanges;
  }
  changeExpiryByID.forEach((_expiresAt, id) => {
    if (!activeIDs.has(id)) {
      changeExpiryByID.delete(id);
    }
  });
}

function applyRealtime(itemsUpdate: RuntimeTarget[]) {
  const offset = (pagination.current - 1) * pagination.pageSize;
  if (offset >= itemsUpdate.length) return;
  total.value = Math.max(total.value, itemsUpdate.length);
  reconcileRealtimePage(itemsUpdate.slice(offset, offset + pagination.pageSize));
}
function startRealtime() {
  if (!active.value || realtimeController) return;
  realtimeController = openRealtimeTopicSocket({
    topic: RUNTIME_TARGET_REALTIME_TOPIC.SUMMARY,
    parseMessage: parseRuntimeTargetSummaryPayload,
    onMessage: (payload) => applyRealtime(payload.items),
  });
}
function stopRealtime() {
  realtimeController?.close();
  realtimeController = null;
}
async function load() {
  loading.value = true;
  errorMessage.value = '';
  try {
    const page = await listRuntimeTargetPage({
      limit: pagination.pageSize as 10 | 20 | 50 | 100,
      offset: (pagination.current - 1) * pagination.pageSize,
      ...currentSavedQueryState(),
    });
    items.value = page.items;
    total.value = page.total;
    summary.value = page.summary;
    startRealtime();
  } catch {
    errorMessage.value = t('runtimeTarget.list.loadError');
  } finally {
    loading.value = false;
  }
}
async function discoverLocal() {
  discovering.value = true;
  errorMessage.value = '';
  try {
    await discoverLocalDocker();
    pagination.current = 1;
    await load();
    MessagePlugin.success(t('runtimeTarget.list.discoverSuccess'));
  } catch {
    errorMessage.value = t('runtimeTarget.list.discoverError');
  } finally {
    discovering.value = false;
  }
}
onMounted(() => {
  active.value = true;
  hydrateFromRoute();
  void (async () => {
    await savedViews.load({ hasExplicitState: hasExplicitRouteState() });
    await load();
  })();
});
watch(
  [filters, () => pagination.current, () => pagination.pageSize, visibleColumnKeys],
  () => {
    if (!applyingRoute.value) void replaceRoute();
  },
  { deep: true },
);
onActivated(() => {
  active.value = true;
  startRealtime();
});
onDeactivated(() => {
  active.value = false;
  stopRealtime();
});
onUnmounted(() => {
  stopRealtime();
  clearChangeExpiryScheduler();
});
</script>
<style scoped lang="less">
@import '@/shared/components/card-surface.less';

.runtime-target-page-root {
  display: flex;
  flex: 1 0 auto;
  flex-direction: column;
  min-width: 0;
}

.runtime-target-feedback {
  margin-bottom: var(--td-comp-margin-l);
}

:deep(.runtime-target-identity),
:deep(.runtime-target-counts) {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
}

:deep(.runtime-target-identity a),
:deep(.runtime-target-counts strong) {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
}

:deep(.runtime-target-identity small),
:deep(.runtime-target-counts small) {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  overflow-wrap: anywhere;
}

:deep(.runtime-target-counts span) {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-6);
  justify-content: space-between;
}

:deep(.runtime-target-counts b) {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-small);
}

.runtime-target-discover-button {
  min-height: var(--td-comp-size-xxxl);
  min-width: var(--td-comp-size-xxxl);
}

.runtime-target-card {
  .graft-entity-card-surface();

  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-14);
  min-width: 0;
  padding: var(--graft-density-gap-16);
}

.runtime-target-card__header {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  min-width: 0;
}

.runtime-target-card__identity {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.runtime-target-card__identity a {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  overflow-wrap: anywhere;
}

.runtime-target-card__identity span,
.runtime-target-card__provider span,
.runtime-target-card__metric dt,
.runtime-target-card__metric dd span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  overflow-wrap: anywhere;
}

.runtime-target-card__provider {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-8);
}

.runtime-target-card__provider strong,
.runtime-target-card__metric dd strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
}

.runtime-target-card__metrics {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin: 0;
}

.runtime-target-card__metric {
  background: var(--td-bg-color-container-hover);
  border-radius: var(--td-radius-default);
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
  min-width: 0;
  padding: var(--graft-density-gap-10);
}

.runtime-target-card__metric dt,
.runtime-target-card__metric dd {
  margin: 0;
}

.runtime-target-card__metric dd {
  align-items: flex-start;
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.runtime-target-card__detail {
  align-items: center;
  color: var(--td-brand-color);
  display: inline-flex;
  font: var(--td-font-body-medium);
  justify-content: center;
  min-height: var(--td-comp-size-xxxl);
  padding: 0 var(--graft-density-gap-12);
  text-decoration: none;
}

.runtime-target-card__detail:focus-visible {
  border-radius: var(--td-radius-default);
  outline: 2px solid var(--td-brand-color);
  outline-offset: 2px;
}
</style>
