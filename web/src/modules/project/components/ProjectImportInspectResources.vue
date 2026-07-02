<template>
  <section class="project-import-resources">
    <project-import-section-heading
      :description="t('project.import.preview.resourcesDescription')"
      :title="t('project.import.preview.resourcesTitle')"
    />

    <div class="project-import-resources__surface">
      <div
        class="project-import-resource-switcher"
        role="tablist"
        :aria-label="t('project.import.preview.resourcesTitle')"
      >
        <t-button
          v-for="view in resourceViews"
          :id="resourceTabId(view.value)"
          :key="view.value"
          role="tab"
          :aria-controls="resourcePanelId(view.value)"
          :aria-selected="activeResource === view.value"
          :tabindex="activeResource === view.value ? 0 : -1"
          :theme="activeResource === view.value ? 'primary' : 'default'"
          :variant="activeResource === view.value ? 'base' : 'outline'"
          @click="activateResource(view.value)"
        >
          {{ view.label }} ({{ view.count }})
        </t-button>
      </div>

      <div class="project-import-resources__toolbar">
        <t-input
          v-model="resourceSearchKeyword"
          class="management-list-search"
          clearable
          :placeholder="t('project.import.preview.resourceSearchPlaceholder')"
        >
          <template #prefix-icon><search-icon /></template>
        </t-input>
        <table-view-toolbar
          :column-settings-label="t('project.import.preview.resourceColumnSettings')"
          :refresh-label="t('project.import.actions.refreshInspect')"
          :refresh-loading="activeResource === 'containers' ? containerLoading : inspectLoading"
          @column-settings="columnDrawerVisible = true"
          @refresh="refreshActiveResource"
        >
          <template #before>
            <span class="project-import-resources__toolbar-copy">{{ activeResourceDescription }}</span>
          </template>
        </table-view-toolbar>
      </div>

      <div
        v-if="activeResource === 'containers'"
        :id="resourcePanelId('containers')"
        class="project-import-resources__table"
        role="tabpanel"
        :aria-labelledby="resourceTabId('containers')"
      >
        <container-resource-table
          v-model:current="resourcePagination.containers.current"
          v-model:page-size="resourcePagination.containers.pageSize"
          :always-visible-column-keys="CONTAINER_RESOURCE_ALWAYS_VISIBLE_COLUMNS"
          :empty-description="containerEmptyDescription"
          :empty-title="t('project.import.preview.resources.empty.containers.title')"
          :footer-summary="activeFooterSummary"
          :head-description="activeResourceDescription"
          :head-summary="activeResourceSummary"
          :loading="containerLoading"
          :readonly-mode="true"
          :row-actions="buildContainerRowActions"
          :rows="pagedContainerRows"
          :sort="resourceSort.containers"
          :total="filteredContainerRows.length"
          :visible-column-keys="visibleColumnKeys.containers"
          @action="handleContainerAction"
          @sort-change="(sort) => handleSortChange('containers', sort)"
        >
          <template #feedback>
            <t-alert v-if="containerError" theme="error" :message="containerError" />
          </template>
        </container-resource-table>
      </div>

      <div
        v-else-if="activeResource === 'networks'"
        :id="resourcePanelId('networks')"
        class="project-import-resources__table"
        role="tabpanel"
        :aria-labelledby="resourceTabId('networks')"
      >
        <management-paged-table
          v-model:current="resourcePagination.networks.current"
          v-model:page-size="resourcePagination.networks.pageSize"
          :columns="visibleNetworkColumns"
          :description="activeResourceDescription"
          :empty-description="t('project.import.preview.resources.empty.networks.description')"
          :empty-title="t('project.import.preview.resources.empty.networks.title')"
          :footer-summary="activeFooterSummary"
          head-label="project-import-networks-table"
          :rows="pagedNetworkRows"
          row-key="id"
          :sort="resourceSort.networks"
          :summary="activeResourceSummary"
          :total="filteredNetworkRows.length"
          @sort-change="(sort) => handleSortChange('networks', sort)"
        >
          <template #internal="{ row }">
            <t-tag :theme="row.internal ? 'warning' : 'default'" variant="light-outline">
              {{
                row.internal
                  ? t('project.import.preview.resources.internalYes')
                  : t('project.import.preview.resources.internalNo')
              }}
            </t-tag>
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
              @action="handleNetworkAction($event, row)"
            />
          </template>
        </management-paged-table>
      </div>

      <div
        v-else
        :id="resourcePanelId('volumes')"
        class="project-import-resources__table"
        role="tabpanel"
        :aria-labelledby="resourceTabId('volumes')"
      >
        <management-paged-table
          v-model:current="resourcePagination.volumes.current"
          v-model:page-size="resourcePagination.volumes.pageSize"
          :columns="visibleVolumeColumns"
          :description="activeResourceDescription"
          :empty-description="t('project.import.preview.resources.empty.volumes.description')"
          :empty-title="t('project.import.preview.resources.empty.volumes.title')"
          :footer-summary="activeFooterSummary"
          head-label="project-import-volumes-table"
          :rows="pagedVolumeRows"
          row-key="id"
          :sort="resourceSort.volumes"
          :summary="activeResourceSummary"
          :total="filteredVolumeRows.length"
          @sort-change="(sort) => handleSortChange('volumes', sort)"
        >
          <template #anonymous="{ row }">
            <t-tag :theme="row.anonymous ? 'warning' : 'default'" variant="light-outline">
              {{
                row.anonymous
                  ? t('project.import.preview.resources.volumeAnonymousYes')
                  : t('project.import.preview.resources.volumeAnonymousNo')
              }}
            </t-tag>
          </template>
          <template #mounted_by="{ row }">
            <span>{{ joinList(row.mountedBy) }}</span>
          </template>
          <template #operation="{ row }">
            <table-action-menu
              :actions="[
                {
                  label: 'components.commonTable.detail',
                  value: 'detail',
                },
              ]"
              @action="handleVolumeAction($event, row)"
            />
          </template>
        </management-paged-table>
      </div>
    </div>

    <advanced-query-column-drawer
      v-model:selected-keys="activeVisibleColumnKeys"
      v-model:visible="columnDrawerVisible"
      :columns="activeColumnSettingOptions"
      :default-selected-keys="activeDefaultColumnKeys"
      :disabled-keys="activeLockedColumnKeys"
      :reset-label="t('project.import.preview.resourceResetColumns')"
      :title="t('project.import.preview.resourceColumnSettings')"
    />

    <t-dialog
      v-model:visible="detailDialogVisible"
      :close-on-overlay-click="true"
      :confirm-btn="null"
      :footer="false"
      :header="detailDialogTitle"
      width="720px"
    >
      <div v-if="detailDialogType === 'network' && activeNetworkDetail" class="project-import-resource-detail">
        <t-descriptions :column="1" size="small">
          <t-descriptions-item :label="t('project.import.preview.resources.columns.name')">
            {{ activeNetworkDetail.name }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.import.preview.resources.columns.driver')">
            {{ activeNetworkDetail.driver }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.import.preview.resources.columns.scope')">
            {{ activeNetworkDetail.scope }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.import.preview.resources.columns.internal')">
            {{
              activeNetworkDetail.internal
                ? t('project.import.preview.resources.internalYes')
                : t('project.import.preview.resources.internalNo')
            }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.import.preview.resources.columns.containers')">
            {{ activeNetworkDetail.containerCount }}
          </t-descriptions-item>
        </t-descriptions>
        <div class="project-import-resource-detail__list">
          <strong>{{ t('project.import.preview.resources.detailServices') }}</strong>
          <t-space break-line size="small">
            <t-tag v-for="service in activeNetworkDetail.services" :key="service" variant="light-outline">
              {{ service }}
            </t-tag>
          </t-space>
        </div>
      </div>

      <div v-else-if="detailDialogType === 'volume' && activeVolumeDetail" class="project-import-resource-detail">
        <t-descriptions :column="1" size="small">
          <t-descriptions-item :label="t('project.import.preview.resources.columns.name')">
            {{ activeVolumeDetail.name }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.import.preview.resources.columns.driver')">
            {{ activeVolumeDetail.driver }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.import.preview.resources.columns.mountTarget')">
            {{ activeVolumeDetail.mountTarget }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.import.preview.resources.columns.anonymous')">
            {{
              activeVolumeDetail.anonymous
                ? t('project.import.preview.resources.volumeAnonymousYes')
                : t('project.import.preview.resources.volumeAnonymousNo')
            }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.import.preview.resources.columns.containers')">
            {{ activeVolumeDetail.containerCount }}
          </t-descriptions-item>
        </t-descriptions>
        <div class="project-import-resource-detail__list">
          <strong>{{ t('project.import.preview.resources.detailMountedBy') }}</strong>
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
import { SearchIcon } from 'tdesign-icons-vue-next';
import type { TableSort, TdBaseTableProps } from 'tdesign-vue-next';
import { computed, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import { getContainers } from '@/modules/container/api/container';
import ContainerResourceTable from '@/modules/container/components/ContainerResourceTable.vue';
import { CONTAINER_BOOTSTRAP_ROUTE } from '@/modules/container/contract/bootstrap';
import {
  buildContainerResourceColumnSettingOptions,
  CONTAINER_RESOURCE_ALL_COLUMN_KEYS,
  CONTAINER_RESOURCE_ALWAYS_VISIBLE_COLUMNS,
  CONTAINER_RESOURCE_DEFAULT_VISIBLE_COLUMNS,
  type ContainerResourceRowAction,
  displayContainerName,
} from '@/modules/container/shared/resource-table';
import type { ContainerSummaryRecord } from '@/modules/container/types/container';
import {
  createActionColumn,
  createMainTextColumn,
  ManagementPagedTable,
  TableActionMenu,
  TableViewToolbar,
} from '@/shared/components/management';
import { AdvancedQueryColumnDrawer } from '@/shared/components/query-list';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';

import {
  type ImportInspectNetworkRow,
  type ImportInspectResourceKey,
  type ImportInspectVolumeRow,
  normalizeImportInspectNetworkRows,
  normalizeImportInspectVolumeRows,
} from '../shared/import-inspect-resources';
import { appendResolvedTab, buildDetailTitleWithFallback } from '../shared/navigation';
import { useProjectPageContext } from '../shared/page-context';
import type { ProjectImportInspectResponse } from '../types/import';
import ProjectImportSectionHeading from './ProjectImportSectionHeading.vue';

defineOptions({
  name: 'ProjectImportInspectResources',
});

type PaginationState = {
  current: number;
  pageSize: number;
};

const DEFAULT_PAGE_SIZE = 10;
const CONTAINER_COLUMN_STORAGE_KEY = 'graft.project.import.resources.containers.columns.v1';
const NETWORK_COLUMN_STORAGE_KEY = 'graft.project.import.resources.networks.columns.v1';
const VOLUME_COLUMN_STORAGE_KEY = 'graft.project.import.resources.volumes.columns.v1';
const NETWORK_ALL_COLUMN_KEYS = ['name', 'driver', 'scope', 'internal', 'containers', 'operation'];
const NETWORK_DEFAULT_COLUMN_KEYS = ['name', 'driver', 'scope', 'internal', 'containers', 'operation'];
const NETWORK_LOCKED_COLUMN_KEYS = ['name', 'operation'];
const VOLUME_ALL_COLUMN_KEYS = ['name', 'driver', 'anonymous', 'mounted_by', 'operation'];
const VOLUME_DEFAULT_COLUMN_KEYS = ['name', 'driver', 'anonymous', 'mounted_by', 'operation'];
const VOLUME_LOCKED_COLUMN_KEYS = ['name', 'operation'];

const props = defineProps<{
  inspectLoading?: boolean;
  result: ProjectImportInspectResponse | null;
}>();

const emit = defineEmits<{
  (e: 'refresh-requested'): void;
}>();

const { t } = useI18n();
const { router, tabsRouterStore } = useProjectPageContext();

const activeResource = ref<ImportInspectResourceKey>('containers');
const resourceSearchKeyword = ref('');
const columnDrawerVisible = ref(false);
const detailDialogVisible = ref(false);
const detailDialogType = ref<'network' | 'volume' | ''>('');
const activeNetworkDetail = ref<ImportInspectNetworkRow | null>(null);
const activeVolumeDetail = ref<ImportInspectVolumeRow | null>(null);
const containerRows = ref<ContainerSummaryRecord[]>([]);
const containerLoading = ref(false);
const containerError = ref('');
const latestContainerRequestId = ref(0);
const visibleColumnKeys = reactive<Record<ImportInspectResourceKey, string[]>>({
  containers: loadVisibleColumnKeys(
    CONTAINER_COLUMN_STORAGE_KEY,
    CONTAINER_RESOURCE_DEFAULT_VISIBLE_COLUMNS,
    CONTAINER_RESOURCE_ALL_COLUMN_KEYS,
    CONTAINER_RESOURCE_ALWAYS_VISIBLE_COLUMNS,
  ),
  networks: loadVisibleColumnKeys(
    NETWORK_COLUMN_STORAGE_KEY,
    NETWORK_DEFAULT_COLUMN_KEYS,
    NETWORK_ALL_COLUMN_KEYS,
    NETWORK_LOCKED_COLUMN_KEYS,
  ),
  volumes: loadVisibleColumnKeys(
    VOLUME_COLUMN_STORAGE_KEY,
    VOLUME_DEFAULT_COLUMN_KEYS,
    VOLUME_ALL_COLUMN_KEYS,
    VOLUME_LOCKED_COLUMN_KEYS,
  ),
});
const resourcePagination = reactive<Record<ImportInspectResourceKey, PaginationState>>({
  containers: { current: 1, pageSize: DEFAULT_PAGE_SIZE },
  networks: { current: 1, pageSize: DEFAULT_PAGE_SIZE },
  volumes: { current: 1, pageSize: DEFAULT_PAGE_SIZE },
});
const resourceSort = reactive<Record<ImportInspectResourceKey, TableSort | undefined>>({
  containers: undefined,
  networks: undefined,
  volumes: undefined,
});

const normalizedKeyword = computed(() => resourceSearchKeyword.value.trim().toLowerCase());
const networkRows = computed(() => normalizeImportInspectNetworkRows(props.result));
const volumeRows = computed(() => normalizeImportInspectVolumeRows(props.result));
const resourceViews = computed(() => [
  {
    value: 'containers' as const,
    label: t('project.import.preview.resources.tabs.containers'),
    count: filteredContainerRows.value.length,
  },
  {
    value: 'networks' as const,
    label: t('project.import.preview.resources.tabs.networks'),
    count: filteredNetworkRows.value.length,
  },
  {
    value: 'volumes' as const,
    label: t('project.import.preview.resources.tabs.volumes'),
    count: filteredVolumeRows.value.length,
  },
]);

const networkColumns = computed<TdBaseTableProps['columns']>(() => [
  createMainTextColumn(t('project.import.preview.resources.columns.name'), 'name', 220),
  { title: t('project.import.preview.resources.columns.driver'), colKey: 'driver', width: 120, sorter: true },
  { title: t('project.import.preview.resources.columns.scope'), colKey: 'scope', width: 120, sorter: true },
  { title: t('project.import.preview.resources.columns.internal'), colKey: 'internal', width: 120, sorter: true },
  { title: t('project.import.preview.resources.columns.containers'), colKey: 'containers', width: 120, sorter: true },
  createActionColumn(t('components.commonTable.operation'), 120),
]);
const volumeColumns = computed<TdBaseTableProps['columns']>(() => [
  createMainTextColumn(t('project.import.preview.resources.columns.name'), 'name', 220),
  { title: t('project.import.preview.resources.columns.driver'), colKey: 'driver', width: 120, sorter: true },
  { title: t('project.import.preview.resources.columns.anonymous'), colKey: 'anonymous', width: 120, sorter: true },
  { title: t('project.import.preview.resources.columns.mountedBy'), colKey: 'mounted_by', minWidth: 220, sorter: true },
  createActionColumn(t('components.commonTable.operation'), 120),
]);
const networkColumnSettingOptions = computed(() =>
  NETWORK_ALL_COLUMN_KEYS.map((value) => ({
    label: t(`project.import.preview.resources.columns.${columnOptionLabel(value)}`),
    value,
  })),
);
const volumeColumnSettingOptions = computed(() =>
  VOLUME_ALL_COLUMN_KEYS.map((value) => ({
    label: t(`project.import.preview.resources.columns.${columnOptionLabel(value)}`),
    value,
  })),
);
const visibleNetworkColumns = computed(() =>
  filterVisibleColumns(networkColumns.value ?? [], visibleColumnKeys.networks, NETWORK_LOCKED_COLUMN_KEYS),
);
const visibleVolumeColumns = computed(() =>
  filterVisibleColumns(volumeColumns.value ?? [], visibleColumnKeys.volumes, VOLUME_LOCKED_COLUMN_KEYS),
);

const filteredContainerRows = computed(() =>
  containerRows.value.filter((row) =>
    matchesKeyword(normalizedKeyword.value, [
      row.id,
      row.name,
      ...(row.names ?? []),
      row.image,
      row.status,
      row.runtime,
      row.compose_project,
      row.compose_service,
      row.network_summary,
    ]),
  ),
);
const filteredNetworkRows = computed(() =>
  networkRows.value.filter((row) =>
    matchesKeyword(normalizedKeyword.value, [row.name, row.driver, row.scope, ...row.services, ...row.containers]),
  ),
);
const filteredVolumeRows = computed(() =>
  volumeRows.value.filter((row) =>
    matchesKeyword(normalizedKeyword.value, [
      row.name,
      row.driver,
      row.mountTarget,
      ...row.mountedBy,
      ...row.containers,
    ]),
  ),
);
const sortedContainerRows = computed(() =>
  sortRows(filteredContainerRows.value, resourceSort.containers, readContainerSortValue),
);
const sortedNetworkRows = computed(() =>
  sortRows(filteredNetworkRows.value, resourceSort.networks, readNetworkSortValue),
);
const sortedVolumeRows = computed(() => sortRows(filteredVolumeRows.value, resourceSort.volumes, readVolumeSortValue));
const pagedContainerRows = computed(() =>
  paginateRows(
    sortedContainerRows.value,
    resourcePagination.containers.current,
    resourcePagination.containers.pageSize,
  ),
);
const pagedNetworkRows = computed(() =>
  paginateRows(sortedNetworkRows.value, resourcePagination.networks.current, resourcePagination.networks.pageSize),
);
const pagedVolumeRows = computed(() =>
  paginateRows(sortedVolumeRows.value, resourcePagination.volumes.current, resourcePagination.volumes.pageSize),
);

const activeVisibleColumnKeys = computed({
  get: () => visibleColumnKeys[activeResource.value],
  set: (keys: string[]) => {
    visibleColumnKeys[activeResource.value] = normalizeVisibleColumnKeys(
      keys,
      activeAllColumnKeys.value,
      activeLockedColumnKeys.value,
    );
  },
});
const activeAllColumnKeys = computed(() => {
  if (activeResource.value === 'containers') {
    return CONTAINER_RESOURCE_ALL_COLUMN_KEYS;
  }
  if (activeResource.value === 'networks') {
    return NETWORK_ALL_COLUMN_KEYS;
  }
  return VOLUME_ALL_COLUMN_KEYS;
});
const activeLockedColumnKeys = computed(() => {
  if (activeResource.value === 'containers') {
    return CONTAINER_RESOURCE_ALWAYS_VISIBLE_COLUMNS;
  }
  if (activeResource.value === 'networks') {
    return NETWORK_LOCKED_COLUMN_KEYS;
  }
  return VOLUME_LOCKED_COLUMN_KEYS;
});
const activeDefaultColumnKeys = computed(() => {
  if (activeResource.value === 'containers') {
    return CONTAINER_RESOURCE_DEFAULT_VISIBLE_COLUMNS;
  }
  if (activeResource.value === 'networks') {
    return NETWORK_DEFAULT_COLUMN_KEYS;
  }
  return VOLUME_DEFAULT_COLUMN_KEYS;
});
const activeColumnSettingOptions = computed(() => {
  if (activeResource.value === 'containers') {
    return buildContainerResourceColumnSettingOptions(t);
  }
  if (activeResource.value === 'networks') {
    return networkColumnSettingOptions.value;
  }
  return volumeColumnSettingOptions.value;
});
const activeResourceDescription = computed(() => {
  if (activeResource.value === 'containers') {
    return t('project.import.preview.resources.descriptions.containers');
  }
  if (activeResource.value === 'networks') {
    return t('project.import.preview.resources.descriptions.networks');
  }
  return t('project.import.preview.resources.descriptions.volumes');
});
const activeResourceSummary = computed(() =>
  t('project.import.preview.resources.summary', {
    count: activeTotalCount.value,
    resource: t(`project.import.preview.resources.tabs.${activeResource.value}`),
  }),
);
const activeFooterSummary = computed(() =>
  t(
    'project.import.preview.resources.paginationSummary',
    paginationRange(activeTotalCount.value, resourcePagination[activeResource.value]),
  ),
);
const activeTotalCount = computed(() => {
  if (activeResource.value === 'containers') {
    return filteredContainerRows.value.length;
  }
  if (activeResource.value === 'networks') {
    return filteredNetworkRows.value.length;
  }
  return filteredVolumeRows.value.length;
});
const containerEmptyDescription = computed(() =>
  normalizedKeyword.value
    ? t('project.import.preview.resources.empty.filteredDescription')
    : t('project.import.preview.resources.empty.containers.description'),
);
const detailDialogTitle = computed(() => {
  if (detailDialogType.value === 'network' && activeNetworkDetail.value) {
    return activeNetworkDetail.value.name;
  }
  if (detailDialogType.value === 'volume' && activeVolumeDetail.value) {
    return activeVolumeDetail.value.name;
  }
  return '';
});

watch(
  () => props.result?.inspection_id,
  () => {
    void loadContainers();
    resetResourceState();
  },
  { immediate: true },
);

watch(normalizedKeyword, () => {
  resourcePagination[activeResource.value].current = 1;
});

watch(
  () => activeResource.value,
  () => {
    columnDrawerVisible.value = false;
  },
);

watch(
  () => filteredContainerRows.value.length,
  (total) => clampPagination(total, resourcePagination.containers),
);
watch(
  () => filteredNetworkRows.value.length,
  (total) => clampPagination(total, resourcePagination.networks),
);
watch(
  () => filteredVolumeRows.value.length,
  (total) => clampPagination(total, resourcePagination.volumes),
);

watch(
  () => visibleColumnKeys.containers,
  (keys) => persistVisibleColumnKeys(CONTAINER_COLUMN_STORAGE_KEY, keys),
  { deep: true },
);
watch(
  () => visibleColumnKeys.networks,
  (keys) => persistVisibleColumnKeys(NETWORK_COLUMN_STORAGE_KEY, keys),
  { deep: true },
);
watch(
  () => visibleColumnKeys.volumes,
  (keys) => persistVisibleColumnKeys(VOLUME_COLUMN_STORAGE_KEY, keys),
  { deep: true },
);

function activateResource(nextResource: ImportInspectResourceKey) {
  activeResource.value = nextResource;
}

function resourceTabId(resource: ImportInspectResourceKey) {
  return `project-import-resource-tab-${resource}`;
}

function resourcePanelId(resource: ImportInspectResourceKey) {
  return `project-import-resource-panel-${resource}`;
}

async function refreshActiveResource() {
  if (activeResource.value === 'containers') {
    await loadContainers();
    return;
  }

  emit('refresh-requested');
}

async function loadContainers() {
  const projectName = props.result?.canonical_project_name?.trim();
  if (!projectName) {
    latestContainerRequestId.value += 1;
    containerRows.value = [];
    containerLoading.value = false;
    containerError.value = '';
    return;
  }

  const requestId = latestContainerRequestId.value + 1;
  latestContainerRequestId.value = requestId;
  containerLoading.value = true;
  containerError.value = '';

  try {
    const rows: ContainerSummaryRecord[] = [];
    const pageSize = 100;
    let offset = 0;
    let total = 0;

    do {
      const payload = await getContainers({
        limit: pageSize,
        offset,
        orchestrator: 'compose',
        source_scope_kind: 'compose_project',
        source_scope: projectName,
      });
      if (requestId !== latestContainerRequestId.value) {
        return;
      }

      rows.push(...payload.items);
      total = payload.total;
      offset += payload.items.length;
      if (!payload.items.length) {
        break;
      }
    } while (rows.length < total);

    containerRows.value = rows;
  } catch (error) {
    if (requestId !== latestContainerRequestId.value) {
      return;
    }

    containerRows.value = [];
    containerError.value = resolveLocalizedErrorMessage(t, error, t('project.import.preview.resources.loadFailed'));
  } finally {
    if (requestId === latestContainerRequestId.value) {
      containerLoading.value = false;
    }
  }
}

function buildContainerRowActions(_row: ContainerSummaryRecord): ContainerResourceRowAction[] {
  return [
    {
      fallbackLabel: t('container.list.actions.detail'),
      label: 'container.list.actions.detail',
      value: 'detail',
    },
  ];
}

function handleContainerAction(payload: { action: string; row: ContainerSummaryRecord }) {
  if (payload.action !== 'detail') {
    return;
  }

  const target = {
    name: CONTAINER_BOOTSTRAP_ROUTE.DETAIL.pageRouteName,
    params: { id: payload.row.id },
    query: { name: displayContainerName(payload.row) },
  };
  const resolved = router.resolve(target);
  appendResolvedTab(
    tabsRouterStore,
    resolved,
    buildDetailTitleWithFallback('container.route.detail.title', displayContainerName(payload.row)),
  );
  void router.push(target);
}

function handleNetworkAction(action: string, row: ImportInspectNetworkRow) {
  if (action !== 'detail') {
    return;
  }

  activeNetworkDetail.value = row;
  detailDialogType.value = 'network';
  detailDialogVisible.value = true;
}

function handleVolumeAction(action: string, row: ImportInspectVolumeRow) {
  if (action !== 'detail') {
    return;
  }

  activeVolumeDetail.value = row;
  detailDialogType.value = 'volume';
  detailDialogVisible.value = true;
}

function handleSortChange(resource: ImportInspectResourceKey, sort: TableSort) {
  resourceSort[resource] = sort;
}

function resetResourceState() {
  resourcePagination.containers.current = 1;
  resourcePagination.networks.current = 1;
  resourcePagination.volumes.current = 1;
  resourceSort.containers = undefined;
  resourceSort.networks = undefined;
  resourceSort.volumes = undefined;
}

function paginationRange(total: number, pagination: PaginationState) {
  if (!total) {
    return { start: 0, end: 0, total: 0 };
  }

  const start = (pagination.current - 1) * pagination.pageSize + 1;
  const end = Math.min(pagination.current * pagination.pageSize, total);
  return { start, end, total };
}

function clampPagination(total: number, pagination: PaginationState) {
  const maxPage = Math.max(1, Math.ceil(total / pagination.pageSize));
  if (pagination.current > maxPage) {
    pagination.current = maxPage;
  }
}

function paginateRows<T>(rows: T[], current: number, pageSize: number) {
  const start = (current - 1) * pageSize;
  return rows.slice(start, start + pageSize);
}

function sortRows<T>(
  rows: T[],
  sort: TableSort | undefined,
  readValue: (row: T, sortBy: string) => string | number | boolean | null | undefined,
) {
  if (!sort || Array.isArray(sort) || !sort.sortBy) {
    return rows;
  }

  const nextRows = [...rows];
  nextRows.sort((left, right) => {
    const leftValue = normalizeSortValue(readValue(left, sort.sortBy));
    const rightValue = normalizeSortValue(readValue(right, sort.sortBy));
    if (leftValue === rightValue) {
      return 0;
    }
    if (leftValue < rightValue) {
      return sort.descending ? 1 : -1;
    }
    return sort.descending ? -1 : 1;
  });
  return nextRows;
}

function normalizeSortValue(value: string | number | boolean | null | undefined) {
  if (typeof value === 'number') {
    return value;
  }
  if (typeof value === 'boolean') {
    return value ? 1 : 0;
  }
  return String(value ?? '').toLowerCase();
}

function readContainerSortValue(row: ContainerSummaryRecord, sortBy: string) {
  switch (sortBy) {
    case 'state':
      return row.state;
    case 'name':
      return displayContainerName(row);
    case 'image':
      return row.image;
    case 'source':
      return row.compose_project || row.orchestrator?.group_value || '';
    case 'cpu':
      return row.resource?.cpu_percent ?? -1;
    case 'memory':
      return row.resource?.memory_usage_bytes ?? -1;
    case 'ports':
      return row.ports.length;
    case 'network':
      return row.network_summary || row.primary_ip || '';
    case 'runtime_status':
      return row.status;
    case 'created_at':
      return row.created_at;
    case 'started_at':
      return row.started_at;
    case 'restart_policy':
      return row.restart_policy;
    case 'image_id':
      return row.image_id;
    case 'labels':
      return Object.keys(row.labels ?? {})
        .sort()
        .join(',');
    case 'resource':
      return row.resource?.memory_usage_bytes ?? row.resource?.cpu_percent ?? -1;
    default:
      return '';
  }
}

function readNetworkSortValue(row: ImportInspectNetworkRow, sortBy: string) {
  switch (sortBy) {
    case 'name':
      return row.name;
    case 'driver':
      return row.driver;
    case 'scope':
      return row.scope;
    case 'internal':
      return row.internal;
    case 'containers':
      return row.containerCount;
    default:
      return '';
  }
}

function readVolumeSortValue(row: ImportInspectVolumeRow, sortBy: string) {
  switch (sortBy) {
    case 'name':
      return row.name;
    case 'driver':
      return row.driver;
    case 'anonymous':
      return row.anonymous;
    case 'mounted_by':
      return row.mountedBy.join(',');
    default:
      return '';
  }
}

function matchesKeyword(keyword: string, values: Array<string | null | undefined>) {
  if (!keyword) {
    return true;
  }

  return values.some((value) => value?.toLowerCase().includes(keyword));
}

function joinList(values: string[]) {
  return values.length ? values.join(', ') : '-';
}

function filterVisibleColumns(
  columns: NonNullable<TdBaseTableProps['columns']>,
  selectedKeys: string[],
  lockedKeys: string[],
) {
  const keySet = new Set([...selectedKeys, ...lockedKeys]);
  return columns.filter((column) => keySet.has(String(column.colKey)));
}

function columnOptionLabel(value: string) {
  if (value === 'mounted_by') {
    return 'mountedBy';
  }
  return value;
}

function loadVisibleColumnKeys(
  storageKey: string,
  defaultKeys: string[],
  allKeys: string[],
  alwaysVisibleKeys: string[],
) {
  if (typeof window === 'undefined') {
    return [...defaultKeys];
  }

  try {
    const stored = window.localStorage.getItem(storageKey);
    if (!stored) {
      return [...defaultKeys];
    }

    const parsed = JSON.parse(stored);
    if (!Array.isArray(parsed)) {
      return [...defaultKeys];
    }

    return normalizeVisibleColumnKeys(parsed, allKeys, alwaysVisibleKeys);
  } catch {
    return [...defaultKeys];
  }
}

function persistVisibleColumnKeys(storageKey: string, keys: string[]) {
  if (typeof window === 'undefined') {
    return;
  }

  try {
    window.localStorage.setItem(storageKey, JSON.stringify(keys));
  } catch {
    // Column settings are optional UI preferences.
  }
}

function normalizeVisibleColumnKeys(keys: unknown[], allKeys: string[], alwaysVisibleKeys: string[]) {
  const availableKeySet = new Set(allKeys);
  const nextKeys = new Set<string>();

  for (const key of keys) {
    if (typeof key === 'string' && availableKeySet.has(key)) {
      nextKeys.add(key);
    }
  }

  for (const key of alwaysVisibleKeys) {
    nextKeys.add(key);
  }

  return allKeys.filter((key) => nextKeys.has(key));
}
</script>
<style scoped lang="less">
.project-import-resources,
.project-import-resources__surface,
.project-import-resources__table,
.project-import-resource-detail,
.project-import-resource-detail__list {
  display: grid;
  gap: var(--graft-density-gap-16);
}

.project-import-resources__toolbar-copy {
  color: var(--td-text-color-secondary);
  margin: 0;
}

.project-import-resource-switcher {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12);
}

.project-import-resources__toolbar {
  align-items: flex-start;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
}

.project-import-resources__toolbar :deep(.management-list-search) {
  flex: 1 1 280px;
  min-width: 220px;
}
</style>
