<template>
  <div class="docker-volume-page" data-page-type="list-form-detail">
    <management-page-header
      :title="t('container.volume.list.title')"
      :description="t('container.volume.list.description')"
      :source="{ labelKey: 'container.list.eyebrow', fallback: t('container.list.eyebrow') }"
    >
      <template #actions>
        <t-button v-if="canRemove" variant="outline" :loading="cleanup.loading.value" @click="openCleanup">
          {{ t('container.volume.actions.cleanup') }}
        </t-button>
      </template>
    </management-page-header>

    <management-statistics-bar
      :items="volumeStatistics"
      :label="t('container.volume.list.total', { count: volumeSummary?.total ?? 0 })"
      aria-live="polite"
    />

    <management-toolbar>
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
          <t-option value="used" :label="t('container.volume.usage.used')" />
          <t-option value="unused" :label="t('container.volume.usage.unused')" />
        </t-select>
        <t-button variant="outline" @click="advancedFiltersVisible = !advancedFiltersVisible">
          {{ t('container.resourceContext.moreFilters') }}
        </t-button>
        <t-button theme="primary" @click="applyFilters">{{ t('container.volume.filters.query') }}</t-button>
        <t-button variant="text" @click="resetFilters">{{ t('container.volume.filters.reset') }}</t-button>
        <template v-if="advancedFiltersVisible">
          <t-select
            v-model="filters.source"
            class="management-toolbar__select"
            clearable
            :placeholder="t('container.resourceContext.source')"
          >
            <t-option v-for="source in resourceSources" :key="source" :value="source" :label="sourceLabel(source)" />
          </t-select>
          <t-input
            v-model="filters.compose_project"
            class="management-toolbar__select"
            clearable
            :placeholder="t('container.resourceContext.project')"
            @enter="applyFilters"
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
    </management-toolbar>

    <management-toolbar>
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
      :empty-description="
        hasActiveFilters ? t('container.volume.filters.reset') : t('container.volume.list.description')
      "
      :empty-title="t('container.volume.pagination.empty')"
      :footer-summary="paginationSummary"
      :loading="loading"
      row-key="name"
      :rows="rows"
      :selected-row-keys="selectedRowKeys"
      :total="pagination.total"
      @page-change="handlePageChange"
      @select-change="handleSelectChange"
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
          v-else-if="volumeSummary?.reference_unknown"
          class="docker-volume-page__alert"
          theme="warning"
          :message="t('container.volume.metrics.referenceUnknown', { count: volumeSummary.reference_unknown })"
        />
      </template>
      <template #name="{ row }">
        <t-tooltip :content="row.name">
          <t-link class="docker-volume-page__name" theme="primary" @click="openDetail(row)">{{
            middleEllipsis(row.name, 31)
          }}</t-link>
        </t-tooltip>
      </template>
      <template #context="{ row }">
        <div class="docker-volume-page__context">
          <t-tag size="small" variant="light-outline">{{ sourceLabel(row.context.source) }}</t-tag>
          <span v-if="row.context.compose_project">{{ row.context.compose_project }}</span>
          <span v-else-if="row.context.compose_resource">{{ row.context.compose_resource }}</span>
        </div>
      </template>
      <template #references="{ row }">
        <t-space v-if="row.container_references?.length" size="small" break-line>
          <t-link
            v-for="reference in row.container_references.slice(0, 2)"
            :key="reference.id"
            theme="primary"
            @click="openContainerReference(reference.id)"
          >
            <t-tooltip :content="reference.id">{{ reference.name || reference.id }}</t-tooltip>
          </t-link>
          <t-tag v-if="row.container_references.length > 2" size="small" variant="light-outline">
            +{{ row.container_references.length - 2 }}
          </t-tag>
        </t-space>
        <t-tag
          v-else-if="row.reference_count === null || row.reference_count === undefined"
          theme="warning"
          variant="light-outline"
        >
          {{ t('container.volume.usage.unknown') }}
        </t-tag>
        <t-tag v-else size="small" variant="light-outline">{{ t('container.volume.usage.unused') }}</t-tag>
      </template>
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
      @confirm="submitCleanup"
    >
      <t-loading :loading="cleanup.loading.value">
        <t-card v-if="cleanup.items.value.length" :bordered="false">
          <div class="docker-volume-cleanup-summary">
            <span>{{ t('container.volume.cleanup.candidateCount', { count: cleanup.items.value.length }) }}</span>
            <strong>{{ formatBytes(cleanup.totalSize.value, t('container.volume.unavailable')) }}</strong>
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
            <template #size="{ row }">{{ formatBytes(row.size_bytes, t('container.volume.unavailable')) }}</template>
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
      </t-loading>
      <t-empty
        v-if="!cleanup.loading.value && !cleanup.items.value.length"
        :title="t('container.volume.cleanup.empty')"
      />
      <template #footer>
        <t-space>
          <t-button variant="outline" @click="cleanup.visible.value = false">{{
            t('container.volume.cleanup.cancel')
          }}</t-button>
          <t-button theme="danger" :disabled="!cleanup.selectedIds.value.length" @click="submitCleanup">
            {{ t('container.volume.cleanup.removeSelected', { count: cleanup.selectedIds.value.length }) }}
          </t-button>
        </t-space>
      </template>
    </t-dialog>
    <t-drawer
      v-model:visible="detailDrawerVisible"
      :header="selectedVolume?.name || t('container.volume.detail.title')"
      size="520px"
      :footer="false"
    >
      <t-loading :loading="detailLoading">
        <t-alert v-if="detailError" theme="error" :message="detailError" />
        <template v-else-if="selectedVolume">
          <section class="docker-volume-page__section">
            <h3>{{ t('container.resourceContext.overview') }}</h3>
            <t-space break-line size="small">
              <t-tag size="small" variant="light-outline">{{ selectedVolume.driver }}</t-tag>
              <t-tag
                :theme="relationshipPresentation(selectedVolume.relationship_status).theme"
                size="small"
                variant="light"
                >{{ relationshipPresentation(selectedVolume.relationship_status).label }}</t-tag
              >
              <span>{{ formatBytes(selectedVolume.size_bytes, t('container.volume.unavailable')) }}</span>
              <span>{{ formatTime(selectedVolume.created_at) }}</span>
            </t-space>
          </section>
          <docker-resource-context-card :context="selectedVolume.context" resource-kind="volume" />
          <section class="docker-volume-page__section">
            <h3>{{ t('container.resourceContext.relations') }}</h3>
            <t-space v-if="selectedVolume.container_references?.length" size="small" break-line>
              <t-link
                v-for="reference in selectedVolume.container_references"
                :key="reference.id"
                theme="primary"
                @click="openContainerReference(reference.id)"
              >
                <t-tooltip :content="reference.id">{{ reference.name || reference.id }}</t-tooltip>
              </t-link>
            </t-space>
            <span v-else class="docker-volume-page__muted">{{
              relationEmptyLabel(selectedVolume.relationship_status)
            }}</span>
          </section>
          <section class="docker-volume-page__section">
            <h3>{{ t('container.resourceContext.configuration') }}</h3>
            <t-descriptions :column="2">
              <t-descriptions-item :label="t('container.volume.columns.driver')">{{
                selectedVolume.driver
              }}</t-descriptions-item>
              <t-descriptions-item :label="t('container.volume.columns.scope')">{{
                selectedVolume.scope
              }}</t-descriptions-item>
            </t-descriptions>
          </section>
          <t-collapse v-if="Object.keys(selectedVolume.labels || {}).length" class="docker-volume-page__section">
            <t-collapse-panel :header="t('container.resourceContext.metadata')" value="metadata">
              <t-space break-line size="small"
                ><t-tag v-for="(value, key) in selectedVolume.labels" :key="key" variant="light-outline"
                  >{{ key }}={{ value }}</t-tag
                ></t-space
              >
            </t-collapse-panel>
          </t-collapse>
          <section v-if="canRemove" class="docker-volume-page__danger-zone">
            <h3>{{ t('container.resourceContext.dangerZone') }}</h3>
            <t-button theme="danger" variant="outline" @click="confirmRemove(selectedVolume)">{{
              t('container.volume.actions.remove')
            }}</t-button>
          </section>
        </template>
      </t-loading>
    </t-drawer>
  </div>
</template>
<script setup lang="ts">
// 数据卷页负责 Docker 数据卷查询与操作，清理流程通过现有批量删除契约执行未使用候选。
import type { TableProps } from 'tdesign-vue-next';
import { Checkbox, DialogPlugin, Input, MessagePlugin } from 'tdesign-vue-next';
import { computed, h, onMounted, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { CONTAINER_PERMISSION_CODE } from '@/contracts/generated/modules/container';
import type { components } from '@/contracts/openapi/generated/schema';
import {
  ManagementBatchBar,
  ManagementPagedTable,
  ManagementPageHeader,
  type ManagementStatisticItem,
  ManagementStatisticsBar,
  ManagementToolbar,
  TableActionMenu,
  TableViewToolbar,
} from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { formatBytes, formatLocaleDateTime } from '@/shared/observability';
import { usePermissionStore } from '@/store';

import {
  batchRemoveDockerVolumes,
  type DockerVolumeDetail,
  type DockerVolumeListQuery,
  getDockerVolume,
  listDockerVolumes,
  removeDockerVolume,
} from '../../api/container';
import DockerResourceContextCard from '../../components/DockerResourceContextCard.vue';
import { CONTAINER_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import { type CleanupBatchOutcome, useDockerCleanup } from '../../shared/cleanup/use-docker-cleanup';

type VolumeRow = Awaited<ReturnType<typeof listDockerVolumes>>['items'][number];
type CleanupVolume = Omit<VolumeRow, 'size_bytes'> & { id: string; size_bytes: number };
type UsageFilter = 'all' | 'used' | 'unused';
type DockerResourceSource = components['schemas']['docker-resource-source'];
type RelationshipStatus = VolumeRow['relationship_status'];
const resourceSources = ['compose', 'docker_default', 'docker', 'managed', 'imported', 'unknown'] as const;
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
const volumeSummary = ref<Awaited<ReturnType<typeof listDockerVolumes>>['summary'] | null>(null);
const volumeStatistics = computed<ManagementStatisticItem[]>(() => [
  { label: t('container.volume.metrics.total'), value: volumeSummary.value?.total ?? '--' },
  { label: t('container.volume.metrics.inUse'), value: volumeSummary.value?.in_use ?? '--' },
  { label: t('container.volume.metrics.unused'), value: volumeSummary.value?.unused ?? '--' },
  {
    label: t('container.volume.metrics.size'),
    value:
      volumeSummary.value?.size_bytes === null || volumeSummary.value?.size_bytes === undefined
        ? t('container.volume.unavailable')
        : formatBytes(volumeSummary.value.size_bytes, t('container.volume.unavailable')),
  },
]);
const columns = computed<TableProps['columns']>(() => [
  { colKey: 'row-select', type: 'multiple' as const, width: 48 },
  { colKey: 'name', title: t('container.volume.columns.name'), minWidth: 280 },
  { colKey: 'context', title: t('container.resourceContext.context'), minWidth: 210 },
  { colKey: 'references', title: t('container.volume.columns.references'), minWidth: 220 },
  { colKey: 'status', title: t('container.volume.columns.usage'), width: 120 },
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
            fallbackLabel: t('container.volume.actions.remove'),
            label: 'container.volume.actions.remove',
            value: 'remove',
          },
        ]
      : []),
  ];
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
async function submitCleanup() {
  if (!canRemove.value) return;
  await cleanup.submit();
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
function renderForceCheckbox(isChecked: () => boolean, onChange: (checked: boolean) => void) {
  return h(
    Checkbox,
    {
      class: 'docker-volume-remove-confirm__force',
      defaultChecked: isChecked(),
      onChange,
    },
    { default: () => t('container.volume.actions.force') },
  );
}
function handleBatchRemove() {
  if (!selectedRowKeys.value.length || !canRemove.value) return;
  let force = false;
  const selected = new Set(selectedRowKeys.value);
  const mustForce = rows.value.some(
    (row) =>
      selected.has(row.name) &&
      (row.reference_count === null || row.reference_count === undefined || row.reference_count > 0),
  );
  const dialog = DialogPlugin.confirm({
    header: t('container.volume.batch.confirmTitle'),
    theme: 'danger',
    confirmBtn: t('container.volume.batch.remove'),
    cancelBtn: t('container.volume.actions.cancel'),
    body: () =>
      h('div', { class: 'docker-volume-remove-confirm' }, [
        h('p', t('container.volume.batch.confirm', { count: selectedRowKeys.value.length })),
        h('div', { class: 'docker-volume-remove-confirm__names' }, selectedRowKeys.value.join(', ')),
        mustForce
          ? renderForceCheckbox(
              () => force,
              (checked) => (force = checked),
            )
          : null,
      ]),
    onConfirm: async () => {
      if (mustForce && !force) {
        MessagePlugin.warning(t('container.volume.actions.forceRequired'));
        return;
      }
      try {
        const response = await batchRemoveDockerVolumes({ names: selectedRowKeys.value, force });
        const successful = response.items.filter((item) => item.success).map((item) => item.name);
        const failed = response.items.length - successful.length;
        selectedRowKeys.value = selectedRowKeys.value.filter((name) => !successful.includes(name));
        dialog.hide();
        if (!failed) MessagePlugin.success(t('container.volume.batch.success', { count: successful.length }));
        else MessagePlugin.warning(t('container.volume.batch.partial', { success: successful.length, failed }));
        await refresh();
      } catch (cause) {
        MessagePlugin.error(resolveLocalizedErrorMessage(t, cause, t('container.volume.batch.failed')));
      }
    },
  });
}
async function openDetail(row: VolumeRow) {
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
function sourceLabel(source: DockerResourceSource) {
  return t(`container.resourceContext.sourceValues.${source}`);
}
function relationshipPresentation(status: RelationshipStatus) {
  const theme =
    status === 'used'
      ? ('success' as const)
      : status === 'unused'
        ? ('default' as const)
        : status === 'unknown'
          ? ('warning' as const)
          : ('danger' as const);
  return { theme, label: t(`container.resourceContext.relationship.${status}`) };
}
function relationEmptyLabel(status: RelationshipStatus) {
  return status === 'unknown' || status === 'exception'
    ? relationshipPresentation(status).label
    : t('container.resourceContext.noRelations');
}
function openContainerReference(containerId: string) {
  void router.push({
    name: CONTAINER_BOOTSTRAP_ROUTE.DETAIL.pageRouteName,
    params: { id: containerId },
    query: { tab: 'storage' },
  });
}
function formatTime(value?: string) {
  return value ? formatLocaleDateTime(value, locale) : t('container.volume.unavailable');
}
function confirmRemove(row: VolumeRow) {
  let typedName = '';
  let force = false;
  const mustForce = row.reference_count === null || row.reference_count === undefined || row.reference_count > 0;
  const dialog = DialogPlugin.confirm({
    header: t('container.volume.actions.confirmTitle'),
    theme: 'danger',
    confirmBtn: t('container.volume.actions.remove'),
    cancelBtn: t('container.volume.actions.cancel'),
    body: () =>
      h('div', { class: 'docker-volume-remove-confirm' }, [
        h('p', t('container.volume.actions.confirm', { name: row.name })),
        h(Input, {
          defaultValue: typedName,
          placeholder: row.name,
          onChange: (value: string | number) => (typedName = String(value)),
        }),
        mustForce
          ? renderForceCheckbox(
              () => force,
              (checked) => (force = checked),
            )
          : null,
      ]),
    onConfirm: async () => {
      if (typedName !== row.name || (mustForce && !force)) {
        MessagePlugin.warning(
          t(mustForce && !force ? 'container.volume.actions.forceRequired' : 'container.volume.actions.nameRequired'),
        );
        return;
      }
      try {
        await removeDockerVolume(row.name, { force });
        MessagePlugin.success(t('container.volume.actions.removeSuccess'));
        dialog.hide();
        await refresh();
      } catch (cause) {
        MessagePlugin.error(resolveLocalizedErrorMessage(t, cause, t('container.volume.actions.removeFailed')));
      }
    },
  });
}
</script>
<style scoped lang="less">
.docker-volume-page {
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

.docker-volume-page__context {
  display: grid;
  gap: var(--td-comp-margin-xs);
}

.docker-volume-page__section,
.docker-volume-page__danger-zone {
  margin-top: var(--td-comp-margin-xl);
}

.docker-volume-page__section h3,
.docker-volume-page__danger-zone h3 {
  font-size: var(--td-font-size-body-large);
  margin: 0 0 var(--td-comp-margin-m);
}

.docker-volume-page__muted {
  color: var(--td-text-color-placeholder);
}

.docker-volume-page__danger-zone {
  border-top: 1px solid var(--td-component-stroke);
  padding-top: var(--td-comp-paddingTB-l);
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

.docker-volume-remove-confirm {
  display: grid;
  gap: var(--td-comp-margin-l);
}

.docker-volume-remove-confirm__force {
  align-items: center;
  display: flex;
  gap: var(--td-comp-margin-s);
}
</style>
