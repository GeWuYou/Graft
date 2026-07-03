<template>
  <section class="project-resources-section">
    <div class="project-section-heading">
      <div>
        <h2>{{ t('project.detail.sections.resources.title') }}</h2>
        <p>{{ t('project.detail.sections.resources.description') }}</p>
      </div>
    </div>

    <advanced-query-paged-table
      v-model:current="activePagination.current"
      v-model:page-size="activePagination.pageSize"
      :cell-slot-names="activeCellSlotNames"
      :columns="visibleColumns"
      :description="activeResourceDescription"
      :empty-description="activeEmptyDescription"
      :empty-title="activeEmptyTitle"
      :footer-summary="footerSummary"
      :head-label="activeResourceLabel"
      :loading="activeResourceLoading"
      row-key="id"
      :rows="pagedRows"
      :summary="tableSummary"
      :total="activeTotal"
      @page-change="handlePageChange"
    >
      <template #toolbar>
        <div class="project-resources-toolbar">
          <div v-if="showResourceSwitcher" class="project-resource-switcher">
            <t-button
              v-for="view in resourceViews"
              :key="view.value"
              :theme="activeResource === view.value ? 'primary' : 'default'"
              :variant="activeResource === view.value ? 'base' : 'outline'"
              @click="activateResource(view.value)"
            >
              {{ view.label }} ({{ view.count }})
            </t-button>
          </div>
          <table-view-toolbar
            :column-settings-label="t('project.detail.resources.columnSettings')"
            :refresh-label="t('project.detail.resources.refresh')"
            @column-settings="columnDrawerVisible = true"
            @refresh="refreshActiveResource"
          />
        </div>
      </template>

      <template #state="{ row }">
        <t-tag :theme="containerStateTheme(row.state)" variant="light-outline">
          {{ containerStateLabel(row.state) }}
        </t-tag>
      </template>

      <template #name="{ row }">
        <div class="project-container-identity">
          <span class="project-container-identity__name">{{ row.name || row.short_id || row.id }}</span>
          <span class="project-container-identity__id">{{ row.short_id || row.id }}</span>
        </div>
      </template>

      <template #service="{ row }">
        <span>{{ readContainerService(row) }}</span>
      </template>

      <template #image="{ row }">
        <span>{{ row.image || '-' }}</span>
      </template>

      <template #cpu="{ row }">
        <span>{{ formatContainerCpu(row) }}</span>
      </template>

      <template #memory="{ row }">
        <span>{{ formatContainerMemory(row) }}</span>
      </template>

      <template #created_at="{ row }">
        <span>{{ formatProjectTime(locale, row.created_at) }}</span>
      </template>

      <template #anonymous="{ row }">
        <t-tag :theme="row.anonymous ? 'warning' : 'default'" variant="light-outline">
          {{
            row.anonymous
              ? t('project.detail.resources.volumeAnonymousYes')
              : t('project.detail.resources.volumeAnonymousNo')
          }}
        </t-tag>
      </template>

      <template #mounted_by="{ row }">
        <span>{{ joinList(row.mountedBy) }}</span>
      </template>

      <template #services="{ row }">
        <span>{{ joinList(row.services) }}</span>
      </template>

      <template #containers="{ row }">
        <span>{{ row.containerCount }}</span>
      </template>

      <template #operation="{ row }">
        <table-action-menu
          :actions="[
            {
              label: 'components.commonTable.detail',
              value: 'detail',
            },
          ]"
          @action="handleRowAction($event, row)"
        />
      </template>
    </advanced-query-paged-table>

    <advanced-query-column-drawer
      v-model:visible="columnDrawerVisible"
      v-model:selected-keys="activeVisibleColumnKeys"
      :columns="activeColumnOptions"
      :default-selected-keys="activeDefaultColumnKeys"
      :disabled-keys="activeLockedColumnKeys"
      :reset-label="t('project.detail.resources.resetColumns')"
      :title="t('project.detail.resources.columnSettings')"
    />

    <t-dialog
      v-model:visible="detailDialogVisible"
      :close-on-overlay-click="true"
      :confirm-btn="null"
      :footer="false"
      :header="detailDialogTitle"
      width="720px"
    >
      <div v-if="detailDialogType === 'network' && activeNetworkDetail" class="project-resource-detail">
        <t-descriptions :column="1" size="small">
          <t-descriptions-item :label="t('project.detail.resources.columns.name')">
            {{ activeNetworkDetail.name }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.detail.resources.columns.driver')">
            {{ activeNetworkDetail.driver }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.detail.resources.columns.scope')">
            {{ activeNetworkDetail.scope }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.detail.resources.columns.containers')">
            {{ activeNetworkDetail.containerCount }}
          </t-descriptions-item>
        </t-descriptions>
        <div class="project-resource-detail__list">
          <strong>{{ t('project.detail.resources.detailServices') }}</strong>
          <t-space break-line size="small">
            <t-tag v-for="service in activeNetworkDetail.services" :key="service" variant="light-outline">
              {{ service }}
            </t-tag>
          </t-space>
        </div>
      </div>

      <div v-else-if="detailDialogType === 'volume' && activeVolumeDetail" class="project-resource-detail">
        <t-descriptions :column="1" size="small">
          <t-descriptions-item :label="t('project.detail.resources.columns.name')">
            {{ activeVolumeDetail.name }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.detail.resources.columns.driver')">
            {{ activeVolumeDetail.driver }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.detail.resources.columns.mountTarget')">
            {{ activeVolumeDetail.mountTarget }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.detail.resources.columns.anonymous')">
            {{
              activeVolumeDetail.anonymous
                ? t('project.detail.resources.volumeAnonymousYes')
                : t('project.detail.resources.volumeAnonymousNo')
            }}
          </t-descriptions-item>
        </t-descriptions>
        <div class="project-resource-detail__list">
          <strong>{{ t('project.detail.resources.detailMountedBy') }}</strong>
          <t-space break-line size="small">
            <t-tag v-for="service in activeVolumeDetail.mountedBy" :key="service" variant="light-outline">
              {{ service }}
            </t-tag>
          </t-space>
        </div>
      </div>
    </t-dialog>
  </section>
</template>
<script setup lang="ts">
import type { TableRowData, TdBaseTableProps } from 'tdesign-vue-next';
import { computed, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import { getContainers } from '@/modules/container/api/container';
import type { ContainerState, ContainerSummaryRecord } from '@/modules/container/types/container';
import { TableActionMenu, TableViewToolbar } from '@/shared/components/management';
import { AdvancedQueryColumnDrawer, AdvancedQueryPagedTable } from '@/shared/components/query-list';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { formatBytes, formatPercent } from '@/shared/observability';
import { createLogger } from '@/utils/logger';

import { getProjectServices } from '../api/project';
import {
  buildProjectNetworkResourceRows,
  buildProjectVolumeResourceRows,
  paginateProjectResourceRows,
  type ProjectNetworkResourceRow,
  type ProjectResourceKind,
  type ProjectVolumeResourceRow,
} from '../shared/detail-resources';
import { formatProjectTime } from '../shared/display';

defineOptions({
  name: 'ProjectResourcesSection',
});

type ProjectContainerDetailMember = {
  container_id: string;
  container_name: string;
  state: string;
};

type ResourcePaginationState = {
  current: number;
  pageSize: number;
};

const props = withDefaults(
  defineProps<{
    activeResource?: ProjectResourceKind;
    canonicalProjectName: string;
    projectId: number;
    showResourceSwitcher?: boolean;
  }>(),
  {
    activeResource: undefined,
    showResourceSwitcher: true,
  },
);

const emit = defineEmits<{
  (e: 'open-container-detail', member: ProjectContainerDetailMember): void;
}>();

const { locale, t } = useI18n();
const logger = createLogger('project.detail.resources');

const activeResource = ref<ProjectResourceKind>(props.activeResource ?? 'containers');
const showResourceSwitcher = computed(() => props.showResourceSwitcher);
const columnDrawerVisible = ref(false);
const detailDialogVisible = ref(false);
const detailDialogType = ref<'network' | 'volume' | ''>('');
const activeNetworkDetail = ref<ProjectNetworkResourceRow | null>(null);
const activeVolumeDetail = ref<ProjectVolumeResourceRow | null>(null);
const containerLoading = ref(false);
const servicesLoading = ref(false);
const resourceError = ref('');
const servicesLoaded = ref(false);
const containerRows = ref<ContainerSummaryRecord[]>([]);
const containerTotal = ref(0);
const networkRows = ref<ProjectNetworkResourceRow[]>([]);
const volumeRows = ref<ProjectVolumeResourceRow[]>([]);

function buildOperationColumn() {
  return {
    title: t('components.commonTable.operation'),
    colKey: 'operation',
    width: 112,
    align: 'center',
    fixed: 'right',
    ellipsis: false,
  } satisfies NonNullable<TdBaseTableProps['columns']>[number];
}

const resourcePagination = reactive<Record<ProjectResourceKind, ResourcePaginationState>>({
  containers: { current: 1, pageSize: 10 },
  networks: { current: 1, pageSize: 10 },
  volumes: { current: 1, pageSize: 10 },
});

const defaultVisibleColumnKeys = {
  containers: ['state', 'name', 'service', 'image', 'cpu', 'memory', 'created_at', 'operation'],
  networks: ['name', 'driver', 'scope', 'services', 'containers', 'operation'],
  volumes: ['name', 'driver', 'anonymous', 'mounted_by', 'mountTarget', 'containers', 'operation'],
} satisfies Record<ProjectResourceKind, string[]>;

const lockedColumnKeys = {
  containers: ['name', 'operation'],
  networks: ['name', 'operation'],
  volumes: ['name', 'operation'],
} satisfies Record<ProjectResourceKind, string[]>;

const visibleColumnState = reactive<Record<ProjectResourceKind, string[]>>({
  containers: loadVisibleColumnKeys('containers', defaultVisibleColumnKeys.containers),
  networks: loadVisibleColumnKeys('networks', defaultVisibleColumnKeys.networks),
  volumes: loadVisibleColumnKeys('volumes', defaultVisibleColumnKeys.volumes),
});

const containerColumns = computed<TdBaseTableProps['columns']>(() => [
  {
    title: t('project.detail.resources.columns.status'),
    colKey: 'state',
    width: 112,
    align: 'center',
    ellipsis: false,
  },
  { title: t('project.detail.resources.columns.name'), colKey: 'name', minWidth: 220 },
  { title: t('project.detail.resources.columns.service'), colKey: 'service', width: 180 },
  { title: t('project.detail.resources.columns.image'), colKey: 'image', minWidth: 220 },
  { title: t('project.detail.resources.columns.cpu'), colKey: 'cpu', width: 120, align: 'center', ellipsis: false },
  {
    title: t('project.detail.resources.columns.memory'),
    colKey: 'memory',
    width: 160,
    align: 'center',
    ellipsis: false,
  },
  {
    title: t('project.detail.resources.columns.createdAt'),
    colKey: 'created_at',
    width: 176,
    align: 'center',
  },
  {
    title: t('components.commonTable.operation'),
    colKey: 'operation',
    width: 112,
    align: 'center',
    fixed: 'right',
    ellipsis: false,
  },
]);

const networkColumns = computed<TdBaseTableProps['columns']>(() => [
  { title: t('project.detail.resources.columns.name'), colKey: 'name', minWidth: 220 },
  { title: t('project.detail.resources.columns.driver'), colKey: 'driver', width: 120, align: 'center' },
  { title: t('project.detail.resources.columns.scope'), colKey: 'scope', width: 120, align: 'center' },
  { title: t('project.detail.resources.columns.services'), colKey: 'services', minWidth: 220 },
  { title: t('project.detail.resources.columns.containers'), colKey: 'containers', width: 120, align: 'center' },
  buildOperationColumn(),
]);

const volumeColumns = computed<TdBaseTableProps['columns']>(() => [
  { title: t('project.detail.resources.columns.name'), colKey: 'name', minWidth: 220 },
  { title: t('project.detail.resources.columns.driver'), colKey: 'driver', width: 120, align: 'center' },
  { title: t('project.detail.resources.columns.anonymous'), colKey: 'anonymous', width: 132, align: 'center' },
  { title: t('project.detail.resources.columns.mountedBy'), colKey: 'mounted_by', minWidth: 220 },
  { title: t('project.detail.resources.columns.mountTarget'), colKey: 'mountTarget', minWidth: 220 },
  { title: t('project.detail.resources.columns.containers'), colKey: 'containers', width: 120, align: 'center' },
  buildOperationColumn(),
]);

const activePagination = computed(() => resourcePagination[activeResource.value]);
const activeAllColumns = computed(() => {
  if (activeResource.value === 'containers') {
    return containerColumns.value;
  }
  if (activeResource.value === 'networks') {
    return networkColumns.value;
  }
  return volumeColumns.value;
});
const activeVisibleColumnKeys = computed({
  get: () => visibleColumnState[activeResource.value],
  set: (keys: string[]) => {
    const nextKeys = normalizeVisibleColumnKeys(activeResource.value, keys);
    visibleColumnState[activeResource.value] = nextKeys;
    persistVisibleColumnKeys(activeResource.value, nextKeys);
  },
});
const visibleColumns = computed<TdBaseTableProps['columns']>(() => {
  const keys = new Set(activeVisibleColumnKeys.value);
  return (activeAllColumns.value ?? []).filter((column) => keys.has(String(column?.colKey)));
});
const activeDefaultColumnKeys = computed(() => defaultVisibleColumnKeys[activeResource.value]);
const activeLockedColumnKeys = computed(() => lockedColumnKeys[activeResource.value]);
const activeColumnOptions = computed(() =>
  (activeAllColumns.value ?? [])
    .filter((column) => typeof column?.colKey === 'string')
    .map((column) => ({
      label: String(column?.title ?? column?.colKey ?? ''),
      value: String(column?.colKey),
    })),
);
const activeCellSlotNames = computed(() => {
  if (activeResource.value === 'containers') {
    return ['state', 'name', 'service', 'image', 'cpu', 'memory', 'created_at', 'operation'];
  }
  if (activeResource.value === 'networks') {
    return ['services', 'containers', 'operation'];
  }
  return ['anonymous', 'mounted_by', 'containers', 'operation'];
});
const activeResourceLoading = computed(() =>
  activeResource.value === 'containers' ? containerLoading.value : servicesLoading.value,
);
const activeTotal = computed(() => {
  if (activeResource.value === 'containers') {
    return containerTotal.value;
  }
  if (activeResource.value === 'networks') {
    return networkRows.value.length;
  }
  return volumeRows.value.length;
});
const resourceViews = computed(() => [
  {
    count: containerTotal.value,
    label: t('project.detail.resources.tabs.containers'),
    value: 'containers' as const,
  },
  {
    count: networkRows.value.length,
    label: t('project.detail.resources.tabs.networks'),
    value: 'networks' as const,
  },
  {
    count: volumeRows.value.length,
    label: t('project.detail.resources.tabs.volumes'),
    value: 'volumes' as const,
  },
]);
const activeResourceLabel = computed(
  () => resourceViews.value.find((view) => view.value === activeResource.value)?.label ?? '',
);
const activeResourceDescription = computed(() => t(`project.detail.resources.descriptions.${activeResource.value}`));
const activeEmptyTitle = computed(() => t(`project.detail.resources.empty.${activeResource.value}.title`));
const activeEmptyDescription = computed(() => {
  if (resourceError.value) {
    return resourceError.value;
  }
  return t(`project.detail.resources.empty.${activeResource.value}.description`);
});
const tableSummary = computed(() =>
  t('project.detail.resources.summary', {
    count: activeTotal.value,
    resource: activeResourceLabel.value,
  }),
);
const footerSummary = computed(() => {
  if (!activeTotal.value) {
    return t('project.detail.resources.pagination.empty');
  }

  const start = (activePagination.value.current - 1) * activePagination.value.pageSize + 1;
  const end = Math.min(activePagination.value.current * activePagination.value.pageSize, activeTotal.value);
  return t('project.detail.resources.pagination.summary', {
    end,
    start,
    total: activeTotal.value,
  });
});
const pagedRows = computed<TableRowData[]>(() => {
  if (activeResource.value === 'containers') {
    return containerRows.value as TableRowData[];
  }
  if (activeResource.value === 'networks') {
    return paginateProjectResourceRows(
      networkRows.value,
      activePagination.value.current,
      activePagination.value.pageSize,
    ) as TableRowData[];
  }
  return paginateProjectResourceRows(
    volumeRows.value,
    activePagination.value.current,
    activePagination.value.pageSize,
  ) as TableRowData[];
});
const detailDialogTitle = computed(() => {
  if (detailDialogType.value === 'network') {
    return activeNetworkDetail.value?.name ?? '';
  }
  if (detailDialogType.value === 'volume') {
    return activeVolumeDetail.value?.name ?? '';
  }
  return '';
});

watch(
  () => [props.projectId, props.canonicalProjectName] as const,
  () => {
    resetResourceState();
    resourceError.value = '';
    void loadActiveResource();
  },
  { immediate: true },
);

watch(
  () => props.activeResource,
  (value) => {
    if (value && activeResource.value !== value) {
      activeResource.value = value;
    }
  },
  { immediate: true },
);

watch(activeResource, async () => {
  resourcePagination[activeResource.value].current = 1;
  resourceError.value = '';
  await loadActiveResource();
});

watch(
  () => [resourcePagination.containers.current, resourcePagination.containers.pageSize],
  async () => {
    if (activeResource.value === 'containers') {
      await loadContainerRows();
    }
  },
);

function resetResourceState() {
  containerRows.value = [];
  containerTotal.value = 0;
  networkRows.value = [];
  volumeRows.value = [];
  servicesLoaded.value = false;
  resourcePagination.containers.current = 1;
  resourcePagination.networks.current = 1;
  resourcePagination.volumes.current = 1;
}

async function refreshActiveResource() {
  await loadActiveResource(true);
}

async function loadActiveResource(forceRefresh = false) {
  if (activeResource.value === 'containers') {
    await loadContainerRows();
    return;
  }

  await loadDerivedResourceRows(forceRefresh);
}

async function loadContainerRows() {
  if (!props.canonicalProjectName.trim()) {
    containerRows.value = [];
    containerTotal.value = 0;
    return;
  }

  containerLoading.value = true;
  resourceError.value = '';
  try {
    const response = await getContainers({
      limit: activePagination.value.pageSize,
      offset: (activePagination.value.current - 1) * activePagination.value.pageSize,
      orchestrator: 'compose',
      source_scope: props.canonicalProjectName,
      source_scope_kind: 'compose_project',
    });
    containerRows.value = response.items;
    containerTotal.value = response.total;
  } catch (error) {
    logger.error('failed to load project container rows', error);
    containerRows.value = [];
    containerTotal.value = 0;
    resourceError.value = resolveLocalizedErrorMessage(t, error, t('project.detail.resources.loadFailed'));
  } finally {
    containerLoading.value = false;
  }
}

async function loadDerivedResourceRows(forceRefresh = false) {
  if (!Number.isFinite(props.projectId)) {
    networkRows.value = [];
    volumeRows.value = [];
    return;
  }

  if (servicesLoaded.value && !forceRefresh) {
    return;
  }

  servicesLoading.value = true;
  resourceError.value = '';
  try {
    const response = await getProjectServices(props.projectId);
    networkRows.value = buildProjectNetworkResourceRows(response.items);
    volumeRows.value = buildProjectVolumeResourceRows(response.items);
    servicesLoaded.value = true;
  } catch (error) {
    logger.error('failed to load project service resources', error);
    networkRows.value = [];
    volumeRows.value = [];
    resourceError.value = resolveLocalizedErrorMessage(t, error, t('project.detail.resources.loadFailed'));
  } finally {
    servicesLoading.value = false;
  }
}

function activateResource(value: ProjectResourceKind) {
  if (activeResource.value === value) {
    return;
  }
  activeResource.value = value;
}

function handlePageChange() {
  if (activeResource.value !== 'containers') {
    return;
  }
  void loadContainerRows();
}

function handleRowAction(
  action: string,
  row: ContainerSummaryRecord | ProjectNetworkResourceRow | ProjectVolumeResourceRow,
) {
  if (action !== 'detail') {
    return;
  }

  if (activeResource.value === 'containers') {
    const containerRow = row as ContainerSummaryRecord;
    emit('open-container-detail', {
      container_id: containerRow.id,
      container_name: containerRow.name || containerRow.short_id || containerRow.id,
      state: containerRow.state,
    });
    return;
  }

  if (activeResource.value === 'networks') {
    activeNetworkDetail.value = row as ProjectNetworkResourceRow;
    activeVolumeDetail.value = null;
    detailDialogType.value = 'network';
    detailDialogVisible.value = true;
    return;
  }

  activeVolumeDetail.value = row as ProjectVolumeResourceRow;
  activeNetworkDetail.value = null;
  detailDialogType.value = 'volume';
  detailDialogVisible.value = true;
}

function persistVisibleColumnKeys(resource: ProjectResourceKind, keys: string[]) {
  window.localStorage.setItem(buildStorageKey(resource), JSON.stringify(keys));
}

function loadVisibleColumnKeys(resource: ProjectResourceKind, fallback: string[]) {
  const stored = window.localStorage.getItem(buildStorageKey(resource));
  if (!stored) {
    return normalizeVisibleColumnKeys(resource, fallback);
  }

  try {
    const parsed = JSON.parse(stored) as unknown;
    if (Array.isArray(parsed) && parsed.every((item) => typeof item === 'string')) {
      return normalizeVisibleColumnKeys(resource, parsed);
    }
  } catch (error) {
    logger.warn('failed to restore project resource visible columns', error);
  }

  return normalizeVisibleColumnKeys(resource, fallback);
}

function buildStorageKey(resource: ProjectResourceKind) {
  return `graft.project.detail.resources.${resource}.visibleColumns`;
}

function normalizeVisibleColumnKeys(resource: ProjectResourceKind, keys: string[]) {
  const allowedKeys = new Set((defaultVisibleColumnKeys[resource] ?? []).concat(lockedColumnKeys[resource] ?? []));
  const nextKeys = keys.filter((key) => allowedKeys.has(key));
  for (const key of lockedColumnKeys[resource]) {
    if (!nextKeys.includes(key)) {
      nextKeys.push(key);
    }
  }
  return nextKeys.length > 0 ? nextKeys : [...defaultVisibleColumnKeys[resource]];
}

function containerStateLabel(state: ContainerState) {
  return t(`container.list.states.${state}`);
}

function containerStateTheme(state: ContainerState) {
  if (state === 'running') return 'success';
  if (state === 'paused' || state === 'created') return 'warning';
  if (state === 'restarting' || state === 'removing') return 'primary';
  if (state === 'unknown') return 'danger';
  return 'default';
}

function readContainerService(row: ContainerSummaryRecord) {
  return row.compose_service || row.orchestrator?.service || '-';
}

function formatContainerCpu(row: ContainerSummaryRecord) {
  if (row.resource?.cpu_percent === undefined) {
    return '-';
  }

  return formatPercent(row.resource.cpu_percent);
}

function formatContainerMemory(row: ContainerSummaryRecord) {
  if (row.resource?.memory_usage_bytes === undefined) {
    return '-';
  }

  return formatBytes(row.resource.memory_usage_bytes);
}

function joinList(items: string[]) {
  return items.length > 0 ? items.join(', ') : '-';
}
</script>
<style scoped lang="less">
.project-resources-section {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
}

.project-section-heading {
  align-items: flex-start;
  display: flex;
  justify-content: space-between;
}

.project-section-heading h2,
.project-section-heading p {
  margin: 0;
}

.project-section-heading h2 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
}

.project-section-heading p {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin-top: var(--graft-density-gap-4);
}

.project-resources-toolbar {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  width: 100%;
}

.project-resource-switcher {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
}

.project-container-identity {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
}

.project-container-identity__name {
  color: var(--td-text-color-primary);
  font-weight: 600;
}

.project-container-identity__id {
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
}

.project-resource-detail {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
}

.project-resource-detail__list {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
}

@media (width <= 768px) {
  .project-resources-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .project-resource-switcher {
    width: 100%;
  }
}
</style>
