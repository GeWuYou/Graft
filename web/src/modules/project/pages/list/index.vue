<template>
  <div class="project-page" data-page-type="list-form-detail">
    <management-page-content>
      <management-page-header
        title-key="project.list.title"
        description-key="project.list.description"
        :source="{ labelKey: 'menu.ops.title', fallback: t('project.list.eyebrow') }"
      >
        <template #meta>
          <t-space break-line size="small">
            <t-tag theme="default" variant="light-outline">
              {{ t('project.list.projectCount', { count: totalCount }) }}
            </t-tag>
            <t-tag theme="success" variant="light-outline">
              {{ t('project.list.runningContainers', { count: runningContainerCount }) }}
            </t-tag>
            <t-tag theme="warning" variant="light-outline">
              {{ t('project.list.stoppedContainers', { count: stoppedContainerCount }) }}
            </t-tag>
            <t-tag theme="danger" variant="light-outline">
              {{ t('project.list.warningProjects', { count: warningProjectCount }) }}
            </t-tag>
          </t-space>
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
            <t-button theme="primary" :loading="loading" @click="fetchProjects">
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
            :refresh-loading="loading"
            @column-settings="columnDrawerVisible = true"
            @refresh="fetchProjects"
          />
        </template>

        <management-empty-state
          v-if="errorMessage && !loading"
          tone="error"
          :title="t('project.list.title')"
          :description="errorMessage"
        >
          <template #actions>
            <t-button theme="primary" variant="outline" @click="fetchProjects">
              {{ t('project.list.retry') }}
            </t-button>
          </template>
        </management-empty-state>

        <div v-else ref="tableHostRef" class="project-table-host" :data-table-mode="tableWidthPolicy.mode">
          <t-table
            row-key="id"
            :columns="visibleColumns"
            :data="rows"
            :loading="loading"
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
                  <span>{{ projectRow(row).canonical_project_name }}</span>
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
              <t-tag :theme="runtimeStatusTheme(projectRow(row).runtime_status)" variant="light-outline">
                {{ runtimeStatusLabel(projectRow(row).runtime_status) }}
              </t-tag>
            </template>

            <template #services="{ row }">
              <span>{{ projectRow(row).service_count }}</span>
            </template>

            <template #containers="{ row }">
              <div class="project-container-counts">
                <t-tag theme="success" variant="light">{{ projectRow(row).container_counts.running }}</t-tag>
                <t-tag theme="warning" variant="light">{{ projectRow(row).container_counts.stopped }}</t-tag>
                <t-tag theme="default" variant="light">{{ projectRow(row).container_counts.total }}</t-tag>
              </div>
            </template>

            <template #drift="{ row }">
              <t-tag :theme="driftStatusTheme(projectRow(row).drift_status)" variant="light-outline">
                {{ driftStatusLabel(projectRow(row).drift_status) }}
              </t-tag>
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
                :actions="buildRowActions()"
                :more-label="t('project.list.actions.detail')"
                :more-label-fallback="t('project.list.actions.detail')"
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
import type { TableProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onMounted, ref } from 'vue';
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
  projectDriftStatusTheme,
  projectRefreshStatusLabel,
  projectRefreshStatusTheme,
  projectRuntimeStatusLabel,
  projectRuntimeStatusTheme,
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

const loading = ref(false);
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
  { colKey: 'name', title: t('project.list.columns.name'), width: 320 },
  { colKey: 'source', title: t('project.list.columns.source'), width: 120 },
  { colKey: 'runtime', title: t('project.list.columns.runtime'), width: 140 },
  { colKey: 'services', title: t('project.list.columns.services'), width: 96, align: 'center' },
  { colKey: 'containers', title: t('project.list.columns.containers'), width: 160, align: 'center' },
  { colKey: 'drift', title: t('project.list.columns.drift'), width: 120 },
  { colKey: 'refresh', title: t('project.list.columns.refresh'), width: 180 },
  { colKey: 'operation', title: t('project.list.columns.operation'), width: 120, fixed: 'right' },
]);
const visibleColumnKeys = ref(['name', 'source', 'runtime', 'services', 'containers', 'drift', 'refresh', 'operation']);
const visibleColumns = computed(() =>
  (configurableColumns.value ?? []).filter((column) => visibleColumnKeys.value.includes(String(column?.colKey))),
);
const tableWidthPolicy = computed(() => resolveTableWidthPolicy(visibleColumns.value ?? [], tableHostWidth.value));

const totalCount = computed(() => pagination.value.total);
const runningContainerCount = computed(() => rows.value.reduce((sum, item) => sum + item.container_counts.running, 0));
const stoppedContainerCount = computed(() => rows.value.reduce((sum, item) => sum + item.container_counts.stopped, 0));
const warningProjectCount = computed(
  () =>
    rows.value.filter(
      (item) =>
        item.drift_status !== 'clean' || item.last_refresh_status === 'failed' || item.runtime_status === 'partial',
    ).length,
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
  void fetchProjects();
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

function driftStatusTheme(value: ProjectDriftStatus) {
  return projectDriftStatusTheme(value);
}

function refreshStatusLabel(value: ProjectRefreshStatus) {
  return projectRefreshStatusLabel(t, value);
}

function refreshStatusTheme(value: ProjectRefreshStatus) {
  return projectRefreshStatusTheme(value);
}

function runtimeStatusTheme(value?: ProjectRuntimeStatus | null) {
  return projectRuntimeStatusTheme(value);
}

function runtimeStatusLabel(value?: ProjectRuntimeStatus | null) {
  return projectRuntimeStatusLabel(t, value);
}

function formatTime(value?: string | null) {
  return formatProjectTime(locale.value, value);
}

async function fetchProjects() {
  loading.value = true;
  errorMessage.value = '';
  try {
    const response = await getProjects({
      limit: pagination.value.pageSize,
      offset: (pagination.value.current - 1) * pagination.value.pageSize,
      ...(filters.value.sourceKind !== 'all' ? { source_kind: filters.value.sourceKind } : {}),
      ...(filters.value.driftStatus !== 'all' ? { drift_status: filters.value.driftStatus } : {}),
      ...(filters.value.lastRefreshStatus !== 'all' ? { last_refresh_status: filters.value.lastRefreshStatus } : {}),
    });
    syncPaginationFromResponse(response);
    const keyword = filters.value.keyword.trim().toLowerCase();
    rows.value = keyword
      ? response.items.filter((item) =>
          [item.display_name, item.canonical_project_name, item.working_directory]
            .filter(Boolean)
            .some((candidate) => String(candidate).toLowerCase().includes(keyword)),
        )
      : response.items;
  } catch (error) {
    logger.error('failed to fetch projects', error);
    rows.value = [];
    pagination.value.total = 0;
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('project.list.retry'));
  } finally {
    loading.value = false;
  }
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

function navigateToDetail(row: ProjectListItem, tab: string) {
  const target = {
    name: PROJECT_BOOTSTRAP_ROUTE.DETAIL.pageRouteName,
    params: { id: row.id },
    query: { tab, name: row.display_name },
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
) {
  try {
    await handler(row.id);
    MessagePlugin.success(successMessage);
    await fetchProjects();
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.list.actions.actionFailed')));
  }
}

function buildRowActions() {
  return [
    { value: 'overview', label: t('project.list.actions.overview') },
    { value: 'services', label: t('project.list.actions.services') },
    { value: 'configuration', label: t('project.list.actions.configuration') },
    { value: 'activity', label: t('project.list.actions.activity') },
    { value: 'refresh', label: t('project.list.actions.refresh') },
    { value: 'up', label: t('project.list.actions.up') },
    { value: 'down', label: t('project.list.actions.down') },
    { value: 'restart', label: t('project.list.actions.restart') },
    { value: 'unregister', label: t('project.list.actions.unregister') },
  ];
}

async function handleRowAction(action: string, row: ProjectListItem) {
  if (action === 'overview' || action === 'services' || action === 'configuration' || action === 'activity') {
    await navigateToDetail(row, action);
    return;
  }
  if (action === 'refresh') {
    await runAction(postProjectRefresh, row, t('project.list.actions.refreshSuccess'));
    return;
  }
  if (action === 'up') {
    await runAction(postProjectUp, row, t('project.list.actions.actionSuccess'));
    return;
  }
  if (action === 'down') {
    await runAction(postProjectDown, row, t('project.list.actions.actionSuccess'));
    return;
  }
  if (action === 'restart') {
    await runAction(postProjectRestart, row, t('project.list.actions.actionSuccess'));
    return;
  }
  if (action === 'unregister') {
    await runAction(postProjectUnregister, row, t('project.list.actions.actionSuccess'));
  }
}
</script>
<style scoped lang="less">
.project-page,
.project-table-head,
.project-container-counts,
.project-refresh,
.project-identity {
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

.project-table-host {
  width: 100%;
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

.project-container-counts,
.project-refresh,
.project-column-drawer {
  gap: var(--graft-density-gap-8);
}

.project-refresh {
  align-items: center;
  flex-wrap: wrap;
}

.project-empty {
  padding: var(--graft-density-gap-20) 0;
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
