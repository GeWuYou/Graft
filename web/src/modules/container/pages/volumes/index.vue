<template>
  <div class="docker-volume-page" data-page-type="list-form-detail">
    <management-page-header
      action-layout="inline"
      compact
      :title="t('container.volume.list.title')"
      :description="t('container.volume.list.description')"
      :source="{ labelKey: 'container.list.eyebrow', fallback: t('container.list.eyebrow') }"
    >
      <template #actions>
        <t-button v-if="canRemove" size="small" variant="outline" :loading="cleanup.loading.value" @click="openCleanup">
          {{ t('container.volume.actions.cleanup') }}
        </t-button>
      </template>
    </management-page-header>

    <management-statistics-bar
      :items="volumeStatistics"
      layout="inline"
      :label="t('container.volume.list.total', { count: volumeSummary?.total ?? 0 })"
      aria-live="polite"
    />

    <management-toolbar class="docker-volume-page__toolbar">
      <template #filters>
        <t-input
          v-model="filters.keyword"
          class="management-list-search"
          clearable
          :placeholder="t('container.volume.filters.keyword')"
          @enter="applyFilters"
        />
        <t-select
          v-model="filters.usage"
          class="management-toolbar__select"
          :placeholder="t('container.volume.filters.usage')"
        >
          <t-option value="all" :label="t('container.volume.filters.allUsage')" />
          <t-option value="used" :label="t('container.volume.status.inUse')" />
          <t-option value="unused" :label="t('container.volume.status.unused')" />
          <t-option value="abnormal" :label="t('container.volume.filters.abnormal')" />
        </t-select>
        <t-button variant="outline" @click="advancedFiltersVisible = !advancedFiltersVisible">
          {{ t('container.resourceContext.moreFilters') }}
        </t-button>
        <t-button theme="primary" @click="applyFilters">{{ t('container.volume.filters.query') }}</t-button>
        <t-button variant="text" @click="resetFilters">{{ t('container.volume.filters.reset') }}</t-button>
        <template v-if="advancedFiltersVisible">
          <docker-resource-context-filters
            v-model:compose-project="filters.compose_project"
            v-model:source="filters.source"
            @apply="applyFilters"
          />
          <t-input
            v-model="filters.driver"
            class="management-toolbar__select"
            clearable
            :placeholder="t('container.volume.filters.driver')"
            @enter="applyFilters"
          />
          <t-input
            v-model="filters.scope"
            class="management-toolbar__select"
            clearable
            :placeholder="t('container.volume.filters.scope')"
            @enter="applyFilters"
          />
        </template>
      </template>
      <template #actions>
        <table-view-toolbar
          :refresh-label="t('container.list.refresh')"
          :refresh-loading="loading"
          @refresh="refresh"
        />
      </template>
    </management-toolbar>

    <management-paged-table
      v-model:current="pagination.current"
      v-model:page-size="pagination.pageSize"
      :columns="columns"
      cards-visible
      density-scope="viewport"
      entity-card-layout="compact"
      :empty-description="
        hasActiveFilters ? t('container.volume.filters.reset') : t('container.volume.list.description')
      "
      :empty-title="t('container.volume.pagination.empty')"
      :footer-summary="paginationSummary"
      :loading="loading"
      preserve-inactive
      row-key="name"
      :rows="rows"
      :selected-row-keys="selectedRowKeys"
      :sort="sort"
      :total="pagination.total"
      @page-change="handlePageChange"
      @select-change="handleSelectChange"
      @sort-change="handleSortChange"
    >
      <template v-if="selectedRowKeys.length" #batch>
        <management-batch-bar
          :selected-label="t('container.volume.batch.selected', { count: selectedRowKeys.length })"
          :clear-label="t('container.volume.batch.cancelSelection')"
          clear-test-id="docker-volume-batch-clear"
          @clear="clearSelection"
        >
          <t-button v-if="canRemove" size="small" theme="danger" variant="outline" @click="handleBatchRemove">
            {{ t('container.volume.batch.remove') }}
          </t-button>
        </management-batch-bar>
      </template>

      <template #feedback>
        <t-alert v-if="error" class="docker-volume-page__alert" theme="error" :message="error" />
        <t-alert
          v-else-if="abnormalCount"
          class="docker-volume-page__alert"
          theme="error"
          :message="t('container.volume.metrics.abnormalNotice', { count: abnormalCount })"
        />
      </template>
      <template #cards>
        <t-empty
          v-if="!rows.length && !loading"
          :title="hasActiveFilters ? t('container.volume.pagination.empty') : t('container.volume.empty.title')"
          :description="
            hasActiveFilters ? t('container.volume.filters.reset') : t('container.volume.empty.description')
          "
        >
          <template v-if="hasActiveFilters" #action>
            <t-button variant="outline" @click="resetFilters">{{ t('container.volume.filters.reset') }}</t-button>
          </template>
        </t-empty>
        <div v-else class="docker-volume-page__cards">
          <article v-for="row in rows" :key="row.name" class="docker-volume-page__card">
            <header class="docker-volume-page__card-header">
              <t-tag :theme="relationshipPresentation(row.relationship_status).theme" size="small" variant="light">
                {{ relationshipPresentation(row.relationship_status).label }}
              </t-tag>
              <t-tooltip :content="row.name">
                <strong>{{ middleEllipsis(row.name, 31) }}</strong>
              </t-tooltip>
            </header>
            <dl class="docker-volume-page__card-primary">
              <div>
                <dt>{{ t('container.volume.columns.size') }}</dt>
                <dd>{{ formatBytes(row.size_bytes, t('container.volume.notCollected')) }}</dd>
              </div>
              <div>
                <dt>{{ t('container.volume.columns.mountedContainers') }}</dt>
                <dd>
                  {{ t('container.volume.card.containerCount', { count: row.container_references?.length ?? 0 }) }}
                </dd>
              </div>
            </dl>
            <dl class="docker-volume-page__card-secondary">
              <div>
                <dt>{{ t('container.volume.columns.driver') }}</dt>
                <dd>{{ row.driver }}</dd>
              </div>
              <div>
                <dt>{{ t('container.volume.columns.createdAt') }}</dt>
                <dd>{{ formatCardDate(row.created_at) }}</dd>
              </div>
            </dl>
            <docker-resource-card-actions
              :detail-label="t('container.volume.actions.detail')"
              :more-actions="canRemove ? volumeCardOverflowOptions() : []"
              :more-label="t('container.list.actions.more')"
              @detail="openDetailPage(row)"
              @action="handleVolumeRowAction($event, row)"
            />
          </article>
        </div>
      </template>
      <template #name="{ row }">
        <div class="docker-volume-page__identity">
          <t-tooltip :content="row.name">
            <t-link class="docker-volume-page__name" theme="primary" @click="openDetail(row)">{{
              middleEllipsis(row.name, 31)
            }}</t-link>
          </t-tooltip>
          <span class="docker-volume-page__hint">{{ volumeType(row) }}</span>
        </div>
      </template>
      <template #references="{ row }">
        <div v-if="row.container_references?.length" class="docker-volume-page__container-list">
          <container-reference-list
            :references="row.container_references"
            :title="t('container.volume.columns.mountedContainers')"
            @open="openContainerReference"
          />
        </div>
        <span v-else class="docker-volume-page__muted">—</span>
      </template>
      <template #size="{ row }">{{ formatBytes(row.size_bytes, t('container.volume.notCollected')) }}</template>
      <template #created_at="{ row }">{{ formatTime(row.created_at) }}</template>
      <template #status="{ row }">
        <t-tag :theme="relationshipPresentation(row.relationship_status).theme" size="small" variant="light">
          {{ relationshipPresentation(row.relationship_status).label }}
        </t-tag>
      </template>
      <template #actions="{ row }">
        <table-action-menu
          :actions="volumeRowActions(row)"
          :more-label="t('container.list.actions.more')"
          @action="handleVolumeRowAction($event, row)"
        />
      </template>
      <template #empty>
        <t-empty :title="t('container.volume.pagination.empty')" :description="t('container.volume.list.description')">
          <template #action>
            <t-button v-if="hasActiveFilters" variant="outline" @click="resetFilters">
              {{ t('container.volume.filters.reset') }}
            </t-button>
          </template>
        </t-empty>
      </template>
    </management-paged-table>
    <t-dialog
      v-model:visible="cleanup.visible.value"
      :header="t('container.volume.cleanup.title')"
      width="760px"
      @confirm="confirmCleanupRemoval"
    >
      <docker-cleanup-loading-host :loading="cleanup.loading.value">
        <t-card v-if="cleanup.items.value.length" :bordered="false">
          <div class="docker-volume-cleanup-summary">
            <span>{{ t('container.volume.cleanup.candidateCount', { count: cleanup.items.value.length }) }}</span>
            <strong>{{ formatBytes(cleanup.totalSize.value, t('container.volume.notCollected')) }}</strong>
          </div>
        </t-card>
        <t-alert
          v-if="cleanup.items.value.length"
          class="docker-volume-cleanup-warning"
          theme="warning"
          :message="t('container.volume.cleanup.warning')"
        />
        <section v-if="cleanup.items.value.length" class="docker-volume-cleanup-preview">
          <div class="docker-volume-cleanup-section-head">
            <strong>{{
              t('container.volume.cleanup.selectedCount', { count: cleanup.selectedIds.value.length })
            }}</strong>
            <t-button
              v-if="cleanup.selectedIds.value.length"
              size="small"
              variant="text"
              @click="cleanup.clearSelection"
            >
              {{ t('container.volume.cleanup.clearSelection') }}
            </t-button>
          </div>
          <t-table
            :columns="cleanupColumns"
            :data="cleanup.previewItems.value"
            row-key="name"
            size="small"
            table-layout="fixed"
            :selected-row-keys="cleanup.selectedIds.value"
            @select-change="cleanup.select"
          >
            <template #name="{ row }">
              <t-tooltip :content="row.name"
                ><span>{{ middleEllipsis(row.name, 31) }}</span></t-tooltip
              >
            </template>
            <template #size="{ row }">{{ formatBytes(row.size_bytes, t('container.volume.notCollected')) }}</template>
          </t-table>
          <div v-if="cleanup.pageCount.value > 1" class="docker-volume-cleanup-pager">
            <t-button
              size="small"
              variant="text"
              :disabled="cleanup.previewPage.value === 1"
              @click="cleanup.previousPage"
            >
              {{ t('container.volume.cleanup.previousPage') }}
            </t-button>
            <span>{{ cleanup.previewPage.value }} / {{ cleanup.pageCount.value }}</span>
            <t-button
              size="small"
              variant="text"
              :disabled="cleanup.previewPage.value === cleanup.pageCount.value"
              @click="cleanup.nextPage"
            >
              {{ t('container.volume.cleanup.nextPage') }}
            </t-button>
          </div>
        </section>
      </docker-cleanup-loading-host>
      <t-empty
        v-if="!cleanup.loading.value && !cleanup.items.value.length"
        :title="t('container.volume.cleanup.empty')"
      />
      <template #footer>
        <t-space>
          <t-button variant="outline" @click="cleanup.visible.value = false">{{
            t('container.volume.cleanup.cancel')
          }}</t-button>
          <t-button theme="danger" :disabled="!cleanup.selectedIds.value.length" @click="confirmCleanupRemoval">
            {{ t('container.volume.cleanup.removeSelected', { count: cleanup.selectedIds.value.length }) }}
          </t-button>
        </t-space>
      </template>
    </t-dialog>
    <resource-detail-layout
      v-model:visible="detailDrawerVisible"
      :title="selectedVolume?.name || t('container.volume.detail.title')"
      :back-label="t('container.detail.back')"
      size="compact"
    >
      <template v-if="selectedVolume" #actions>
        <t-tag :theme="relationshipPresentation(selectedVolume.relationship_status).theme" size="small" variant="light">
          {{ relationshipPresentation(selectedVolume.relationship_status).label }}
        </t-tag>
      </template>
      <div class="docker-volume-page__detail-loading-host">
        <div v-if="detailLoading" class="docker-volume-page__detail-loading-host__indicator">
          <t-loading :loading="true" size="large" />
        </div>
        <t-alert v-else-if="detailError" theme="error" :message="detailError" />
        <volume-detail-content
          v-else-if="selectedVolume"
          :can-remove="canRemove"
          surface="drawer"
          :volume="selectedVolume"
          @open-container="openContainerReference"
          @remove="confirmRemove(selectedVolume)"
        />
        <div v-else-if="!detailLoading" class="docker-volume-page__detail-state">
          <t-empty
            size="small"
            :title="t('container.volume.detail.emptyTitle')"
            :description="t('container.volume.detail.emptyDescription')"
          />
        </div>
      </div>
    </resource-detail-layout>
  </div>
</template>
<script setup lang="ts">
// 数据卷页负责 Docker 数据卷查询与操作，清理流程通过现有批量删除契约执行未使用候选。
import type { TableProps, TableSort } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next';
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { CONTAINER_PERMISSION_CODE } from '@/contracts/generated/modules/container';
import type { components } from '@/contracts/openapi/generated/schema';
import {
  ManagementBatchBar,
  ManagementPageHeader,
  type ManagementStatisticItem,
  ManagementStatisticsBar,
  ManagementToolbar,
  TableActionMenu,
  TableViewToolbar,
} from '@/shared/components/management';
import ManagementPagedTable from '@/shared/components/management/ManagementPagedTable.vue';
import ResourceDetailLayout from '@/shared/components/responsive/ResourceDetailLayout.vue';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { formatBytes, formatLocaleDateOnly, formatLocaleDateTime } from '@/shared/observability';
import { usePermissionStore } from '@/store';

import {
  batchRemoveDockerVolumes,
  type DockerVolumeDetail,
  type DockerVolumeListQuery,
  getDockerVolume,
  listDockerVolumes,
  removeDockerVolume,
} from '../../api/container';
import DockerResourceCardActions from '../../components/DockerResourceCardActions.vue';
import DockerResourceContextFilters from '../../components/DockerResourceContextFilters.vue';
import VolumeDetailContent from '../../components/VolumeDetailContent.vue';
import { CONTAINER_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import DockerCleanupLoadingHost from '../../shared/cleanup/DockerCleanupLoadingHost.vue';
import { type CleanupBatchOutcome, useDockerCleanup } from '../../shared/cleanup/use-docker-cleanup';
import ContainerReferenceList from '../../shared/ContainerReferenceList.vue';
import { getDockerVolumeStatusPresentation } from '../../shared/volume-presentation';
import { openVolumeRemovalConfirmation } from '../../shared/volume-removal';

type VolumeRow = Awaited<ReturnType<typeof listDockerVolumes>>['items'][number];
type CleanupVolume = Omit<VolumeRow, 'size_bytes'> & { id: string; size_bytes: number };
type UsageFilter = 'all' | 'used' | 'unused' | 'abnormal';
type DockerResourceSource = components['schemas']['docker-resource-source'];
const { locale, t } = useI18n();
const router = useRouter();
const permissionStore = usePermissionStore();
const rows = ref<VolumeRow[]>([]);
const loading = ref(false);
const error = ref('');
const filters = reactive<{
  keyword: string;
  driver: string;
  scope: string;
  usage: UsageFilter;
  source: DockerResourceSource | '';
  compose_project: string;
}>({
  keyword: '',
  driver: '',
  scope: '',
  usage: 'all',
  source: '',
  compose_project: '',
});
const applied = ref({ ...filters });
const advancedFiltersVisible = ref(false);
const pagination = reactive({ current: 1, pageSize: 20, total: 0 });
const canRemove = computed(() => permissionStore.hasPermission(CONTAINER_PERMISSION_CODE.VOLUME_REMOVE));
const hasActiveFilters = computed(() =>
  Boolean(
    applied.value.keyword ||
    applied.value.driver ||
    applied.value.scope ||
    applied.value.source ||
    applied.value.compose_project ||
    applied.value.usage !== 'all',
  ),
);
const selectedRowKeys = ref<string[]>([]);
const selectedVolume = ref<DockerVolumeDetail | null>(null);
const detailDrawerVisible = ref(false);
const detailLoading = ref(false);
const detailError = ref('');
const sort = ref<TableSort>({ sortBy: 'size', descending: true });
const volumeSummary = ref<Awaited<ReturnType<typeof listDockerVolumes>>['summary'] | null>(null);
const abnormalCount = computed(() => volumeSummary.value?.reference_unknown ?? 0);
const volumeStatistics = computed<ManagementStatisticItem[]>(() => [
  { label: t('container.volume.metrics.total'), value: volumeSummary.value?.total ?? '--' },
  { label: t('container.volume.metrics.inUse'), value: volumeSummary.value?.in_use ?? '--' },
  { label: t('container.volume.metrics.unused'), value: volumeSummary.value?.unused ?? '--' },
  {
    label: t('container.volume.metrics.size'),
    value:
      volumeSummary.value?.size_bytes === null || volumeSummary.value?.size_bytes === undefined
        ? t('container.volume.notCollected')
        : formatBytes(volumeSummary.value.size_bytes, t('container.volume.notCollected')),
  },
  { label: t('container.volume.metrics.abnormal'), value: abnormalCount.value },
]);
const columns = computed<TableProps['columns']>(() => [
  { colKey: 'row-select', type: 'multiple' as const, width: 48 },
  { colKey: 'name', title: t('container.volume.columns.name'), minWidth: 280 },
  { colKey: 'status', title: t('container.volume.columns.status'), width: 120 },
  { colKey: 'size', title: t('container.volume.columns.size'), width: 120, align: 'right' as const, sorter: true },
  { colKey: 'references', title: t('container.volume.columns.mountedContainers'), minWidth: 260 },
  { colKey: 'driver', title: t('container.volume.columns.driver'), width: 120 },
  { colKey: 'created_at', title: t('container.volume.columns.createdAt'), width: 180 },
  { colKey: 'actions', title: t('container.volume.columns.actions'), width: 144, fixed: 'right' },
]);
const cleanup = useDockerCleanup<CleanupVolume>({
  fetchCandidates: fetchCleanupCandidates,
  execute: removeCleanupVolumes,
  onOutcome: handleCleanupOutcome,
});
const cleanupColumns = computed<TableProps['columns']>(() => [
  { colKey: 'row-select', type: 'multiple' as const, width: 48 },
  { colKey: 'name', title: t('container.volume.columns.name'), minWidth: 280 },
  { colKey: 'size', title: t('container.volume.columns.size'), width: 140, align: 'right' as const },
]);
const paginationSummary = computed(() => {
  if (!pagination.total || !rows.value.length) return t('container.volume.pagination.empty');
  return t('container.volume.pagination.summary', {
    start: (pagination.current - 1) * pagination.pageSize + 1,
    end: Math.min(pagination.current * pagination.pageSize, pagination.total),
    total: pagination.total,
  });
});
onMounted(() => void refresh());
watch(
  () => [pagination.current, pagination.pageSize],
  () => void refresh(),
);
function buildQuery(): DockerVolumeListQuery {
  return {
    limit: pagination.pageSize,
    offset: (pagination.current - 1) * pagination.pageSize,
    keyword: applied.value.keyword || undefined,
    driver: applied.value.driver || undefined,
    scope: applied.value.scope || undefined,
    usage: applied.value.usage === 'all' ? undefined : applied.value.usage,
    source: applied.value.source || undefined,
    compose_project: applied.value.compose_project || undefined,
    sort_by: 'size_bytes',
    sort_order: (Array.isArray(sort.value) ? sort.value[0]?.descending : sort.value.descending) ? 'desc' : 'asc',
  };
}
async function refresh() {
  loading.value = true;
  error.value = '';
  try {
    const response = await listDockerVolumes(buildQuery());
    rows.value = response.items;
    pagination.total = response.total;
    pagination.pageSize = response.limit;
    volumeSummary.value = response.summary;
  } catch (cause) {
    rows.value = [];
    pagination.total = 0;
    volumeSummary.value = null;
    error.value = resolveLocalizedErrorMessage(t, cause, t('container.volume.list.loadFailed'));
  } finally {
    loading.value = false;
  }
}
function applyFilters() {
  const previousPage = pagination.current;
  applied.value = {
    keyword: filters.keyword.trim(),
    driver: filters.driver.trim(),
    scope: filters.scope.trim(),
    usage: filters.usage,
    source: filters.source,
    compose_project: filters.compose_project.trim(),
  };
  pagination.current = 1;
  if (previousPage === 1) void refresh();
}
function resetFilters() {
  filters.keyword = '';
  filters.driver = '';
  filters.scope = '';
  filters.usage = 'all';
  filters.source = '';
  filters.compose_project = '';
  applyFilters();
}
function handlePageChange(page: { current: number; pageSize: number }) {
  pagination.current = page.current;
  pagination.pageSize = page.pageSize;
}
function handleSortChange(nextSort: TableSort) {
  sort.value = nextSort;
  const previousPage = pagination.current;
  pagination.current = 1;
  if (previousPage === 1) void refresh();
}
function volumeRowActions(_row: VolumeRow) {
  return [
    {
      fallbackLabel: t('container.volume.actions.detail'),
      label: 'container.volume.actions.detail',
      value: 'detail',
    },
    ...(canRemove.value
      ? [
          {
            danger: true,
            fallbackLabel: t('container.volume.actions.remove'),
            label: 'container.volume.actions.remove',
            value: 'remove',
          },
        ]
      : []),
  ];
}
function volumeCardOverflowOptions() {
  return [{ danger: true, label: t('container.volume.actions.remove'), value: 'remove' }];
}
function handleVolumeRowAction(action: string, row: VolumeRow) {
  if (action === 'detail') {
    void openDetail(row);
    return;
  }
  if (action === 'remove') confirmRemove(row);
}
function middleEllipsis(value: string, maxLength = 31) {
  if (value.length <= maxLength) return value;
  const edge = Math.floor((maxLength - 3) / 2);
  return `${value.slice(0, edge)}...${value.slice(-edge)}`;
}
function handleSelectChange(keys: Array<string | number>) {
  selectedRowKeys.value = keys.map(String);
}
async function openCleanup() {
  try {
    await cleanup.open();
  } catch (cause) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, cause, t('container.volume.cleanup.loadFailed')));
  }
}
function confirmCleanupRemoval() {
  if (!canRemove.value || !cleanup.selectedIds.value.length) return;
  const selected = new Set(cleanup.selectedIds.value);
  const candidates = cleanup.items.value
    .filter((item) => selected.has(item.id))
    .map((item) => ({ containerNames: [], name: item.name, sizeBytes: item.size_bytes }));
  openVolumeRemovalConfirmation({
    candidates,
    confirmLabel: t('container.volume.cleanup.removeSelected', { count: candidates.length }),
    header: t('container.volume.cleanup.confirmTitle'),
    t,
    onConfirm: async () => {
      await cleanup.submit();
      return true;
    },
  });
}
async function fetchCleanupCandidates(): Promise<CleanupVolume[]> {
  const firstPage = await listDockerVolumes({ limit: 100, offset: 0, usage: 'unused' });
  const all = [...firstPage.items];
  while (all.length < firstPage.total) {
    const page = await listDockerVolumes({ limit: 100, offset: all.length, usage: 'unused' });
    if (!page.items.length) break;
    all.push(...page.items);
  }
  return all.map((row) => ({ ...row, id: row.name, size_bytes: row.size_bytes ?? 0 }));
}
async function removeCleanupVolumes(ids: string[]): Promise<CleanupBatchOutcome> {
  const items: CleanupBatchOutcome['items'] = [];
  let requestError: unknown;
  for (let index = 0; index < ids.length; index += 50) {
    const chunk = ids.slice(index, index + 50);
    try {
      const response = await batchRemoveDockerVolumes({ names: chunk, force: false });
      items.push(
        ...response.items.map((item) => ({
          id: item.name,
          success: item.success,
          error_code: item.error_code ?? undefined,
        })),
      );
    } catch (cause) {
      requestError = cause;
      items.push(...chunk.map((id) => ({ id, success: false, error_code: 'UNKNOWN' })));
      break;
    }
  }
  return { items, unknownResponseIds: [], requestError };
}
async function handleCleanupOutcome(outcome: CleanupBatchOutcome) {
  const successCount = outcome.items.filter((item) => item.success).length;
  const failedCount = outcome.items.length - successCount;
  if (outcome.requestError) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, outcome.requestError, t('container.volume.cleanup.failed')));
  } else if (!failedCount) {
    MessagePlugin.success(t('container.volume.cleanup.success', { count: successCount }));
    cleanup.visible.value = false;
  } else if (successCount) {
    MessagePlugin.warning(t('container.volume.cleanup.partial', { success: successCount, failed: failedCount }));
  } else {
    MessagePlugin.error(t('container.volume.cleanup.failed'));
  }
  await refresh();
}
function clearSelection() {
  selectedRowKeys.value = [];
}
function handleBatchRemove() {
  if (!selectedRowKeys.value.length || !canRemove.value) return;
  const selected = new Set(selectedRowKeys.value);
  const mustForce = rows.value.some(
    (row) =>
      selected.has(row.name) &&
      (row.reference_count === null || row.reference_count === undefined || row.reference_count > 0),
  );
  const candidates = selectedRowKeys.value.map((name) => {
    const row = rows.value.find((item) => item.name === name);
    return {
      containerNames: row?.container_references.map((reference) => reference.name || reference.id) ?? [],
      name,
      sizeBytes: row?.size_bytes,
    };
  });
  openVolumeRemovalConfirmation({
    candidates,
    header: t('container.volume.batch.confirmTitle'),
    confirmLabel: t('container.volume.batch.remove'),
    forceRequired: mustForce,
    t,
    onConfirm: async (force) => {
      try {
        const response = await batchRemoveDockerVolumes({ names: selectedRowKeys.value, force });
        const successful = response.items.filter((item) => item.success).map((item) => item.name);
        const failed = response.items.length - successful.length;
        selectedRowKeys.value = selectedRowKeys.value.filter((name) => !successful.includes(name));
        if (!failed) MessagePlugin.success(t('container.volume.batch.success', { count: successful.length }));
        else MessagePlugin.warning(t('container.volume.batch.partial', { success: successful.length, failed }));
        await refresh();
        return true;
      } catch (cause) {
        MessagePlugin.error(resolveLocalizedErrorMessage(t, cause, t('container.volume.batch.failed')));
        return false;
      }
    },
  });
}
async function openDetail(row: VolumeRow) {
  selectedVolume.value = null;
  detailDrawerVisible.value = true;
  detailLoading.value = true;
  detailError.value = '';
  try {
    selectedVolume.value = await getDockerVolume(row.name);
  } catch (cause) {
    selectedVolume.value = null;
    detailError.value = resolveLocalizedErrorMessage(t, cause, t('container.volume.detail.loadFailed'));
  } finally {
    detailLoading.value = false;
  }
}
function openDetailPage(row: VolumeRow) {
  void router.push({ name: CONTAINER_BOOTSTRAP_ROUTE.VOLUME_DETAIL.pageRouteName, params: { name: row.name } });
}
function volumeType(row: VolumeRow) {
  return row.anonymous ? t('container.volume.types.anonymous') : t('container.volume.types.named');
}
const relationshipPresentation = (status: VolumeRow['relationship_status']) =>
  getDockerVolumeStatusPresentation(t, status);
function openContainerReference(containerId: string) {
  void router.push({
    name: CONTAINER_BOOTSTRAP_ROUTE.DETAIL.pageRouteName,
    params: { id: containerId },
    query: { tab: 'storage' },
  });
}
function formatTime(value?: string) {
  return value ? formatLocaleDateTime(value, locale) : t('container.volume.notCollected');
}
function formatCardDate(value?: string) {
  return value ? formatLocaleDateOnly(value, locale) : t('container.volume.notCollected');
}
function confirmRemove(row: VolumeRow) {
  openVolumeRemovalConfirmation({
    candidates: [
      {
        containerNames: row.container_references.map((reference) => reference.name || reference.id),
        name: row.name,
        sizeBytes: row.size_bytes,
      },
    ],
    confirmationName: row.name,
    forceRequired: row.relationship_status !== 'unused',
    header: t('container.volume.actions.confirmTitle'),
    confirmLabel: t('container.volume.actions.remove'),
    t,
    onConfirm: async (force) => {
      try {
        await removeDockerVolume(row.name, { force });
        MessagePlugin.success(t('container.volume.actions.removeSuccess'));
        detailDrawerVisible.value = false;
        await refresh();
        return true;
      } catch (cause) {
        MessagePlugin.error(resolveLocalizedErrorMessage(t, cause, t('container.volume.actions.removeFailed')));
        return false;
      }
    },
  });
}
</script>
<style scoped lang="less">
.docker-volume-page {
  container-type: inline-size;
  display: grid;
  gap: var(--graft-density-gap-16);
}

.docker-volume-page__alert {
  margin-bottom: var(--td-comp-margin-l);
}

.docker-volume-page__name {
  display: inline-block;
  max-width: 252px;
  overflow: hidden;
  white-space: nowrap;
}

.docker-volume-page__identity {
  display: grid;
  gap: var(--td-comp-margin-xs);
}

.docker-volume-page__cards {
  display: grid;
  gap: var(--graft-density-gap-10);
}

.docker-volume-page__card {
  background: var(--td-bg-color-container);
  block-size: 10rem;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-small);
  display: grid;
  gap: var(--graft-density-gap-4);
  grid-template-rows: 1.5rem 2.25rem 1.5rem 1fr;
  min-width: 0;
  overflow: hidden;
  padding: var(--graft-density-gap-8) var(--graft-density-gap-10);
}

.docker-volume-page__card-header {
  align-items: center;
  display: grid;
  gap: var(--graft-density-gap-8);
  grid-template-columns: auto minmax(0, 1fr);
}

.docker-volume-page__card-header strong {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.docker-volume-page__card-primary,
.docker-volume-page__card-secondary {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin: 0;
}

.docker-volume-page__card-primary {
  gap: var(--graft-density-gap-12);
}

.docker-volume-page__card-secondary {
  gap: var(--graft-density-gap-8);
}

.docker-volume-page__card dt {
  color: var(--td-text-color-secondary);
  font-size: var(--td-font-size-body-small);
}

.docker-volume-page__card dd {
  font-variant-numeric: tabular-nums;
  margin: var(--graft-density-gap-2) 0 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.docker-volume-page__card-secondary > div {
  align-items: baseline;
  display: flex;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.docker-volume-page__card-primary > div,
.docker-volume-page__card-secondary > div {
  min-width: 0;
}

@container (width < 768px) {
  .docker-volume-page {
    gap: var(--graft-density-gap-10);
  }

  .docker-volume-page :deep(.management-page-header),
  .docker-volume-page :deep(.management-toolbar),
  .docker-volume-page :deep(.management-table-card) {
    border-radius: 0;
    box-shadow: none;
  }

  .docker-volume-page :deep(.management-page-header) {
    background: transparent;
    border: 0;
    padding: var(--graft-density-gap-4) 0;
  }

  .docker-volume-page :deep(.management-toolbar) {
    border-left: 0;
    border-right: 0;
    min-height: auto;
    padding: var(--graft-density-gap-10) 0;
  }

  .docker-volume-page :deep(.management-table-card) {
    background: transparent;
    border: 0;
    overflow: visible;
  }

  .docker-volume-page :deep(.management-table-card__body) {
    overflow: visible;
    padding: 0;
  }

  .docker-volume-page :deep(.management-table-card__footer) {
    background: var(--td-bg-color-container);
    border: 1px solid var(--td-component-stroke);
    border-radius: var(--td-radius-small);
    margin-top: var(--graft-density-gap-10);
    padding: var(--graft-density-gap-10);
  }

  .docker-volume-page__alert {
    margin-bottom: var(--graft-density-gap-10);
  }
}

.docker-volume-page__hint,
.docker-volume-page__muted {
  color: var(--td-text-color-placeholder);
  font-size: var(--td-font-size-body-small);
}

.docker-volume-page__detail-state {
  align-items: center;
  display: flex;
  justify-content: center;
  min-height: 240px;
  padding: var(--graft-density-gap-24) var(--graft-density-gap-16);
}

.docker-volume-page__detail-loading-host {
  min-height: 240px;
}

.docker-volume-page__detail-loading-host__indicator {
  display: grid;
  min-height: inherit;
  place-items: center;
}

@media (prefers-reduced-motion: no-preference) {
  .docker-volume-page__detail-loading-host__indicator :deep(.t-icon-loading) {
    animation: docker-volume-detail-loading-spin 1s linear infinite;
  }
}

@keyframes docker-volume-detail-loading-spin {
  to {
    transform: rotate(360deg);
  }
}

.docker-volume-cleanup-summary,
.docker-volume-cleanup-section-head,
.docker-volume-cleanup-pager {
  align-items: center;
  display: flex;
  justify-content: space-between;
}

.docker-volume-cleanup-warning,
.docker-volume-cleanup-preview {
  margin-top: var(--td-comp-margin-l);
}

.docker-volume-cleanup-pager {
  gap: var(--td-comp-margin-m);
  justify-content: center;
  margin-top: var(--td-comp-margin-m);
}
</style>
