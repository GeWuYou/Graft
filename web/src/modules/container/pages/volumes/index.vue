<template>
  <div class="docker-volume-page" data-page-type="list-form-detail">
    <management-page-header
      :title="t('container.volume.list.title')"
      :description="t('container.volume.list.description')"
      :source="{ labelKey: 'container.list.eyebrow', fallback: t('container.list.eyebrow') }"
    >
      <template #meta>
        <t-tag theme="default" variant="light-outline">{{
          t('container.volume.list.total', { count: pagination.total })
        }}</t-tag>
      </template>
    </management-page-header>

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
      <template #actions>
        <table-view-toolbar
          :refresh-label="t('container.list.refresh')"
          :refresh-loading="loading"
          @refresh="refresh"
        />
      </template>
    </management-toolbar>

    <t-alert v-if="error" class="docker-volume-page__alert" theme="error" :message="error" />
    <div v-if="selectedRowKeys.length" class="docker-volume-page__batch-bar">
      <span>{{ t('container.volume.batch.selected', { count: selectedRowKeys.length }) }}</span>
      <t-button v-if="canRemove" size="small" theme="danger" variant="outline" @click="handleBatchRemove">
        {{ t('container.volume.batch.remove') }}
      </t-button>
      <t-button size="small" variant="text" @click="clearSelection">
        {{ t('container.volume.batch.cancelSelection') }}
      </t-button>
    </div>
    <t-table
      row-key="name"
      :data="rows"
      :columns="columns"
      :loading="loading"
      :selected-row-keys="selectedRowKeys"
      :disable-data-page="true"
      table-layout="fixed"
      @select-change="handleSelectChange"
    >
      <template #name="{ row }">
        <t-tooltip :content="row.name">
          <t-link class="docker-volume-page__name" theme="primary" @click="openDetail(row)">{{
            middleEllipsis(row.name)
          }}</t-link>
        </t-tooltip>
      </template>
      <template #usage="{ row }"
        ><t-tag :theme="usageTheme(row)" variant="light-outline">{{ usageLabel(row) }}</t-tag></template
      >
      <template #size="{ row }">{{ formatBytes(row.size_bytes, t('container.volume.unavailable')) }}</template>
      <template #created_at="{ row }">{{ formatTime(row.created_at) }}</template>
      <template #labels="{ row }">{{ Object.keys(row.labels || {}).length }}</template>
      <template #actions="{ row }">
        <t-space size="small">
          <t-button variant="text" size="small" @click="openDetail(row)">{{
            t('container.volume.actions.detail')
          }}</t-button>
          <t-button v-if="canRemove" theme="danger" variant="text" size="small" @click="confirmRemove(row)">{{
            t('container.volume.actions.remove')
          }}</t-button>
        </t-space>
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
    </t-table>
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
        </t-descriptions>
      </t-loading>
    </t-drawer>
    <management-table-pagination :summary="paginationSummary">
      <t-pagination
        v-model:current="pagination.current"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        show-page-size
        @change="handlePageChange"
      />
    </management-table-pagination>
  </div>
</template>
<script setup lang="ts">
import type { TableProps } from 'tdesign-vue-next';
import { Checkbox, DialogPlugin, Input, MessagePlugin } from 'tdesign-vue-next';
import { computed, h, onMounted, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import {
  ManagementPageHeader,
  ManagementTablePagination,
  ManagementToolbar,
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
import { CONTAINER_PERMISSION_CODE } from '../../contract/permissions';

type VolumeRow = Awaited<ReturnType<typeof listDockerVolumes>>['items'][number];
type UsageFilter = 'all' | 'used' | 'unused';
const { locale, t } = useI18n();
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
const columns: TableProps['columns'] = [
  { colKey: 'row-select', type: 'multiple' as const, width: 48 },
  { colKey: 'name', title: t('container.volume.columns.name'), width: 280 },
  { colKey: 'driver', title: t('container.volume.columns.driver'), width: 140, ellipsis: true },
  { colKey: 'scope', title: t('container.volume.columns.scope'), width: 120, ellipsis: true },
  { colKey: 'usage', title: t('container.volume.columns.usage'), width: 130 },
  { colKey: 'size', title: t('container.volume.columns.size'), width: 130 },
  { colKey: 'created_at', title: t('container.volume.columns.createdAt'), width: 180 },
  { colKey: 'labels', title: t('container.volume.columns.labels'), width: 100 },
  { colKey: 'actions', title: t('container.volume.columns.actions'), width: 150, fixed: 'right' },
];
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
  } catch (cause) {
    rows.value = [];
    pagination.total = 0;
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
function middleEllipsis(value: string, maxLength = 42) {
  if (value.length <= maxLength) return value;
  const edge = Math.floor((maxLength - 3) / 2);
  return `${value.slice(0, edge)}...${value.slice(-edge)}`;
}
function handleSelectChange(keys: Array<string | number>) {
  selectedRowKeys.value = keys.map(String);
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
function usageTheme(row: VolumeRow) {
  return row.reference_count === null || row.reference_count === undefined
    ? 'warning'
    : row.reference_count > 0
      ? 'primary'
      : 'default';
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
  gap: var(--td-comp-margin-xl);
}

.docker-volume-page__alert {
  margin-bottom: var(--td-comp-margin-l);
}

.docker-volume-page__batch-bar {
  align-items: center;
  display: flex;
  gap: var(--td-comp-margin-m);
}

.docker-volume-page__name {
  display: inline-block;
  max-width: 252px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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
