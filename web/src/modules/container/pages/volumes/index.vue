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
    <t-table row-key="name" :data="rows" :columns="columns" :loading="loading" :disable-data-page="true">
      <template #name="{ row }"
        ><t-link theme="primary" @click="openDetail(row)">{{ row.name }}</t-link></template
      >
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
    </t-table>
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
import { DialogPlugin, MessagePlugin } from 'tdesign-vue-next';
import { computed, h, onMounted, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import {
  ManagementPageHeader,
  ManagementTablePagination,
  ManagementToolbar,
  TableViewToolbar,
} from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { formatBytes, formatLocaleDateTime } from '@/shared/observability';
import { usePermissionStore } from '@/store';

import { type DockerVolumeListQuery, listDockerVolumes, removeDockerVolume } from '../../api/container';
import { CONTAINER_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import { CONTAINER_PERMISSION_CODE } from '../../contract/permissions';

type VolumeRow = Awaited<ReturnType<typeof listDockerVolumes>>['items'][number];
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
const columns: TableProps['columns'] = [
  { colKey: 'name', title: t('container.volume.columns.name'), ellipsis: true },
  { colKey: 'driver', title: t('container.volume.columns.driver'), ellipsis: true },
  { colKey: 'scope', title: t('container.volume.columns.scope'), ellipsis: true },
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
  applied.value = {
    keyword: filters.keyword.trim(),
    driver: filters.driver.trim(),
    scope: filters.scope.trim(),
    usage: filters.usage,
  };
  pagination.current = 1;
  if (pagination.current === 1) void refresh();
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
function openDetail(row: VolumeRow) {
  void router.push({ name: CONTAINER_BOOTSTRAP_ROUTE.VOLUME_DETAIL.pageRouteName, params: { id: row.name } });
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
        h('input', {
          class: 't-input__inner',
          placeholder: row.name,
          onInput: (event: Event) => {
            typedName = (event.target as HTMLInputElement).value;
          },
        }),
        mustForce
          ? h('label', { class: 'docker-volume-remove-confirm__force' }, [
              h('input', {
                type: 'checkbox',
                onChange: (event: Event) => {
                  force = (event.target as HTMLInputElement).checked;
                },
              }),
              h('span', t('container.volume.actions.force')),
            ])
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
