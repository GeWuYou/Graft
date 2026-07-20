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
        <t-select
          v-model="filters.usage"
          class="management-toolbar__select"
          :placeholder="t('container.volume.filters.usage')"
        >
          <t-option value="all" :label="t('container.volume.filters.allUsage')" />
          <t-option value="used" :label="t('container.volume.usage.used')" />
          <t-option value="unused" :label="t('container.volume.usage.unused')" />
        </t-select>
        <t-button theme="primary" @click="applyFilters">{{ t('container.volume.filters.query') }}</t-button>
        <t-button variant="text" @click="resetFilters">{{ t('container.volume.filters.reset') }}</t-button>
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
      <template #batch>
        <management-batch-bar
          v-if="selectedRowKeys.length"
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
      <template #size="{ row }">{{ formatBytes(row.size_bytes, t('container.volume.unavailable')) }}</template>
      <template #created_at="{ row }">{{ formatTime(row.created_at) }}</template>
      <template #labels="{ row }">{{ Object.keys(row.labels || {}).length }}</template>
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
    <t-drawer
      v-model:visible="detailDrawerVisible"
      :header="selectedVolume?.name || t('container.volume.detail.title')"
      size="520px"
    >
      <t-loading :loading="detailLoading">
        <t-alert v-if="detailError" theme="error" :message="detailError" />
        <t-descriptions v-else-if="selectedVolume" bordered :column="2">
          <t-descriptions-item :label="t('container.volume.columns.name')">{{
            selectedVolume.name
          }}</t-descriptions-item>
          <t-descriptions-item :label="t('container.volume.columns.driver')">{{
            selectedVolume.driver
          }}</t-descriptions-item>
          <t-descriptions-item :label="t('container.volume.columns.scope')">{{
            selectedVolume.scope
          }}</t-descriptions-item>
          <t-descriptions-item :label="t('container.volume.columns.usage')">{{
            usageLabel(selectedVolume)
          }}</t-descriptions-item>
          <t-descriptions-item :label="t('container.volume.columns.size')">{{
            formatBytes(selectedVolume.size_bytes, t('container.volume.unavailable'))
          }}</t-descriptions-item>
          <t-descriptions-item :label="t('container.volume.columns.createdAt')">{{
            formatTime(selectedVolume.created_at)
          }}</t-descriptions-item>
          <t-descriptions-item :label="t('container.volume.detail.labels')" :span="2">
            <t-space break-line>
              <t-tag v-for="(value, key) in selectedVolume.labels || {}" :key="key" variant="light-outline"
                >{{ key }}={{ value }}</t-tag
              >
              <span v-if="!Object.keys(selectedVolume.labels || {}).length">{{
                t('container.volume.detail.noLabels')
              }}</span>
            </t-space>
          </t-descriptions-item>
          <t-descriptions-item :label="t('container.volume.detail.references')" :span="2">
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
            <t-tag
              v-else-if="selectedVolume.reference_count === null || selectedVolume.reference_count === undefined"
              theme="warning"
              variant="light-outline"
            >
              {{ t('container.volume.usage.unknown') }}
            </t-tag>
            <span v-else>{{ t('container.volume.usage.unused') }}</span>
          </t-descriptions-item>
        </t-descriptions>
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
import { CONTAINER_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import { type CleanupBatchOutcome, useDockerCleanup } from '../../shared/cleanup/use-docker-cleanup';

type VolumeRow = Awaited<ReturnType<typeof listDockerVolumes>>['items'][number];
type CleanupVolume = Omit<VolumeRow, 'size_bytes'> & { id: string; size_bytes: number };
type UsageFilter = 'all' | 'used' | 'unused';
const { locale, t } = useI18n();
const router = useRouter();
const permissionStore = usePermissionStore();
const rows = ref<VolumeRow[]>([]);
const loading = ref(false);
const error = ref('');
const filters = reactive({ keyword: '', driver: '', scope: '', usage: 'all' as UsageFilter });
const applied = ref({ ...filters });
const pagination = reactive({ current: 1, pageSize: 20, total: 0 });
const canRemove = computed(() => permissionStore.hasPermission(CONTAINER_PERMISSION_CODE.VOLUME_REMOVE));
const hasActiveFilters = computed(() =>
  Boolean(applied.value.keyword || applied.value.driver || applied.value.scope || applied.value.usage !== 'all'),
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
  { colKey: 'name', title: t('container.volume.columns.name'), width: 280 },
  { colKey: 'driver', title: t('container.volume.columns.driver'), width: 140, ellipsis: true },
  { colKey: 'scope', title: t('container.volume.columns.scope'), width: 120, ellipsis: true },
  { colKey: 'references', title: t('container.volume.columns.references'), minWidth: 220 },
  { colKey: 'size', title: t('container.volume.columns.size'), width: 130 },
  { colKey: 'created_at', title: t('container.volume.columns.createdAt'), width: 180 },
  { colKey: 'labels', title: t('container.volume.columns.labels'), width: 100 },
  { colKey: 'actions', title: t('container.volume.columns.actions'), width: 150, fixed: 'right' },
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
  };
  pagination.current = 1;
  if (previousPage === 1) void refresh();
}
function resetFilters() {
  filters.keyword = '';
  filters.driver = '';
  filters.scope = '';
  filters.usage = 'all';
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
function usageLabel(row: VolumeRow) {
  if (row.reference_count === null || row.reference_count === undefined) return t('container.volume.usage.unknown');
  return row.reference_count > 0
    ? t('container.volume.usage.used', { count: row.reference_count })
    : t('container.volume.usage.unused');
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
