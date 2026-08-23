<template>
  <section class="build-jobs-page" data-page-type="list-form-detail">
    <management-page-header
      compact
      title-key="build.jobs.title"
      :title="t('build.jobs.title')"
      description-key="build.jobs.description"
      :description="t('build.jobs.description')"
      :source="{ labelKey: 'build.jobs.eyebrow', fallback: t('build.jobs.eyebrow') }"
    />

    <management-toolbar>
      <template #filters>
        <t-input
          v-model="search"
          class="build-jobs-page__search"
          clearable
          :placeholder="t('build.jobs.search')"
          @enter="applySearch"
          @clear="applySearch"
        >
          <template #prefix-icon><search-icon /></template>
        </t-input>
        <t-button variant="outline" @click="filterVisible = true">
          <template #icon><filter-icon /></template>
          {{ t('build.jobs.filter') }}<span v-if="activeFilterCount"> ({{ activeFilterCount }})</span>
        </t-button>
      </template>
      <template #actions>
        <t-button variant="outline" :loading="loading" @click="load">
          <template #icon><refresh-icon /></template>{{ t('build.jobs.refresh') }}
        </t-button>
        <t-button theme="primary" @click="router.push(BUILD_ROUTE_PATH.CREATE)">
          <template #icon><add-icon /></template>{{ t('build.jobs.create.title') }}
        </t-button>
      </template>
    </management-toolbar>

    <div v-if="appliedFilters.length" class="build-jobs-page__chips">
      <t-tag v-for="filter in appliedFilters" :key="filter.key" closable @close="clearFilter(filter.key)">
        {{ filter.label }}
      </t-tag>
    </div>

    <management-table-card v-if="errorMessage" class="build-jobs-page__table-card">
      <div class="build-jobs-page__inline-error">
        <t-alert theme="error" :title="t('build.jobs.error.title')" :message="errorMessage" />
        <t-space>
          <t-button variant="outline" @click="load">{{ t('build.jobs.error.retry') }}</t-button>
          <t-button variant="text" @click="errorDetailsVisible = true">{{ t('build.jobs.error.details') }}</t-button>
          <t-button theme="primary" variant="text" @click="router.push(RUNTIME_TARGET_ROUTE_PATH.LIST)">
            {{ t('build.jobs.error.configure') }}
          </t-button>
        </t-space>
      </div>
    </management-table-card>

    <management-paged-table
      v-else
      v-model:current="currentPage"
      v-model:page-size="pageSize"
      class="build-jobs-page__table-card"
      :columns="columns"
      density-scope="viewport"
      :empty-description="
        hasAppliedFilters ? t('build.jobs.emptyFilteredDescription') : t('build.jobs.emptyDescription')
      "
      :empty-title="hasAppliedFilters ? t('build.jobs.emptyFiltered') : t('build.jobs.empty')"
      :footer-summary="t('build.jobs.summary', { count: total })"
      :loading="loading"
      :rows="items"
      :total="total"
      row-key="build_id"
      :pagination-props="{ showPageSize: true }"
      :page-size-options="[20, 50, 100]"
      :cell-slot-names="[
        'build_id',
        'snapshot',
        'repository',
        'image',
        'status',
        'progress',
        'created_at',
        'duration',
        'builder',
        'actions',
      ]"
      @page-change="changePage"
    >
      <template #snapshot="{ row }">
        <div class="build-jobs-page__snapshot">
          <span class="build-jobs-page__ellipsis">{{ (row as BuildJobSummary).input_snapshot_digest }}</span>
          <small>{{ (row as BuildJobSummary).source_kind }}</small>
        </div>
      </template>
      <template #repository="{ row }"
        ><span class="build-jobs-page__ellipsis">{{ (row as BuildJobSummary).image_repository }}</span></template
      >
      <template #image="{ row }"
        ><span class="build-jobs-page__ellipsis">{{ imageReference(row as BuildJobSummary) }}</span></template
      >
      <template #status="{ row }"
        ><t-tag :theme="statusTheme((row as BuildJobSummary).execution?.status)" variant="light-outline">{{
          statusLabel((row as BuildJobSummary).execution?.status)
        }}</t-tag></template
      >
      <template #progress="{ row }">
        <div class="build-jobs-page__progress">
          <t-progress :percentage="progressPercent(row as BuildJobSummary)" size="small" :label="false" /><span>{{
            progressLabel(row as BuildJobSummary)
          }}</span>
        </div>
      </template>
      <template #created_at="{ row }">{{ formatLocaleDateTime((row as BuildJobSummary).created_at, locale) }}</template>
      <template #duration="{ row }">{{ durationLabel(row as BuildJobSummary) }}</template>
      <template #builder="{ row }"
        ><span class="build-jobs-page__ellipsis" :title="(row as BuildJobSummary).builder?.name">{{
          (row as BuildJobSummary).builder?.name || '-'
        }}</span></template
      >
      <template #actions="{ row }">
        <div class="build-jobs-page__row-actions">
          <t-button
            v-if="(row as BuildJobSummary).execution?.capabilities?.retry"
            variant="text"
            shape="square"
            size="small"
            :aria-label="t('build.jobs.actions.retry')"
            @click.stop="openTask((row as BuildJobSummary).task_id)"
            ><template #icon><rotate-icon /></template
          ></t-button>
          <t-button
            v-if="(row as BuildJobSummary).execution?.capabilities?.cancel"
            variant="text"
            shape="square"
            size="small"
            :aria-label="t('build.jobs.actions.cancel')"
            @click.stop="openTask((row as BuildJobSummary).task_id)"
            ><template #icon><stop-circle-icon /></template
          ></t-button>
          <t-button
            variant="text"
            shape="square"
            size="small"
            :aria-label="t('build.jobs.actions.logs')"
            @click.stop="openTask((row as BuildJobSummary).task_id)"
            ><template #icon><file-search-icon /></template
          ></t-button>
          <t-button
            variant="text"
            shape="square"
            size="small"
            :aria-label="t('build.jobs.actions.details')"
            @click.stop="openDetail((row as BuildJobSummary).build_id)"
            ><template #icon><view-list-icon /></template
          ></t-button>
        </div>
      </template>
      <template #empty-action>
        <t-button v-if="!hasAppliedFilters" theme="primary" @click="router.push(BUILD_ROUTE_PATH.CREATE)">{{
          t('build.jobs.create.title')
        }}</t-button>
        <t-button v-else variant="outline" @click="resetAllQueries">{{ t('build.jobs.filters.reset') }}</t-button>
      </template>
    </management-paged-table>

    <t-drawer
      v-model:visible="filterVisible"
      :header="t('build.jobs.filter')"
      :footer="true"
      size="min(520px, 92vw)"
      @confirm="applyFilters"
      @cancel="filterVisible = false"
    >
      <t-form layout="vertical">
        <t-form-item :label="t('build.jobs.filters.repository')"
          ><t-input v-model="filters.image_repository" clearable
        /></t-form-item>
        <t-form-item :label="t('build.jobs.filters.imageTag')"
          ><t-input v-model="filters.image_tag" clearable
        /></t-form-item>
        <t-form-item :label="t('build.jobs.filters.status')"
          ><t-select v-model="filters.status" clearable :options="statusOptions"
        /></t-form-item>
        <t-form-item :label="t('build.jobs.filters.builder')"
          ><t-input-number v-model="filters.builder_id" clearable :min="1"
        /></t-form-item>
        <t-form-item :label="t('build.jobs.filters.createdTime')"
          ><t-date-range-picker v-model="createdRange" clearable enable-time-picker
        /></t-form-item>
      </t-form>
      <template #footer
        ><t-space
          ><t-button variant="outline" @click="resetFilters">{{ t('build.jobs.filters.reset') }}</t-button
          ><t-button theme="primary" @click="applyFilters">{{ t('build.jobs.filters.apply') }}</t-button></t-space
        ></template
      >
    </t-drawer>

    <task-detail-drawer v-model:visible="taskVisible" :task-id="taskId" />
    <t-drawer
      v-model:visible="detailVisible"
      :header="t('build.jobs.detail.title')"
      :footer="false"
      size="min(680px, 92vw)"
    >
      <t-loading :loading="detailLoading"
        ><t-descriptions v-if="detail" bordered :column="2" size="small">
          <t-descriptions-item :label="t('build.jobs.columns.build')">{{ detail.build_id }}</t-descriptions-item
          ><t-descriptions-item :label="t('build.jobs.columns.snapshot')">{{
            detail.input_snapshot_digest
          }}</t-descriptions-item>
          <t-descriptions-item :label="t('build.jobs.columns.repository')">{{
            detail.image_repository
          }}</t-descriptions-item
          ><t-descriptions-item :label="t('build.jobs.columns.imageTag')">{{ detail.image_tag }}</t-descriptions-item>
          <t-descriptions-item :label="t('build.jobs.columns.status')">{{
            statusLabel(detail.execution?.status)
          }}</t-descriptions-item
          ><t-descriptions-item :label="t('build.jobs.columns.builder')">{{
            detail.builder?.name || '-'
          }}</t-descriptions-item> </t-descriptions
        ><t-alert v-else-if="detailError" theme="error" :message="detailError"
      /></t-loading>
    </t-drawer>
    <t-drawer v-model:visible="errorDetailsVisible" :header="t('build.jobs.error.details')" :footer="false"
      ><p>{{ t('build.jobs.error.detailsDescription') }}</p></t-drawer
    >
  </section>
</template>
<script setup lang="ts">
// 构建列表只展示 Build 投影；Task 详情、日志及操作继续交给 Task Runtime 组件。
import {
  AddIcon,
  FileSearchIcon,
  FilterIcon,
  RefreshIcon,
  RotateIcon,
  SearchIcon,
  StopCircleIcon,
  ViewListIcon,
} from 'tdesign-icons-vue-next';
import type { TableProps } from 'tdesign-vue-next';
import { computed, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { RUNTIME_TARGET_ROUTE_PATH } from '@/modules/runtime-target/contract/paths';
import { TaskDetailDrawer } from '@/modules/task/contract/task-ui';
import { ManagementPageHeader, ManagementTableCard, ManagementToolbar } from '@/shared/components/management';
import ManagementPagedTable from '@/shared/components/management/ManagementPagedTable.vue';
import { formatLocaleDateTime } from '@/shared/observability';

import { getBuildJob, getBuildJobs } from '../../api/build';
import { BUILD_ROUTE_PATH } from '../../contract/paths';
import type { BuildExecutionProjection, BuildJobDetail, BuildJobSummary, BuildStatusFilter } from '../../types/build';

type TaskStatus = NonNullable<BuildExecutionProjection['status']>;
const { locale, t } = useI18n();
const router = useRouter();
const items = ref<BuildJobSummary[]>([]);
const total = ref(0);
const currentPage = ref(1);
const pageSize = ref(20);
const loading = ref(false);
const errorMessage = ref('');
const search = ref('');
const filterVisible = ref(false);
const taskVisible = ref(false);
const taskId = ref<number | null>(null);
const detailVisible = ref(false);
const detailLoading = ref(false);
const detailError = ref('');
const detail = ref<BuildJobDetail>();
const errorDetailsVisible = ref(false);
const createdRange = ref<string[]>([]);
const filters = reactive({
  image_repository: '',
  image_tag: '',
  status: '' as BuildStatusFilter | '',
  builder_id: undefined as number | undefined,
});
let requestSequence = 0;
let detailRequestSequence = 0;
const statusOptions = computed(() =>
  [
    ['queued', 'queued'],
    ['running', 'running'],
    ['success', 'success'],
    ['failed', 'failed'],
    ['cancelled', 'cancelled'],
  ].map(([value, label]) => ({ label: t(`build.jobs.status.${label}`), value: value as BuildStatusFilter })),
);
const hasAppliedFilters = computed(() =>
  Boolean(search.value.trim() || Object.values(filters).some(Boolean) || createdRange.value.length),
);
const activeFilterCount = computed(
  () =>
    [filters.image_repository, filters.image_tag, filters.status, filters.builder_id, ...createdRange.value].filter(
      Boolean,
    ).length,
);
const appliedFilters = computed(() => [
  ...Object.entries(filters)
    .filter(([, value]) => value)
    .map(([key, value]) => ({
      key,
      label: `${filterLabel(key)}: ${key === 'status' ? t(`build.jobs.status.${value}`) : value}`,
    })),
  ...(createdRange.value.length
    ? [{ key: 'created_time', label: `${t('build.jobs.filters.createdTime')}: ${createdRange.value.join(' - ')}` }]
    : []),
]);
function filterLabel(key: string) {
  if (key === 'image_repository') return t('build.jobs.filters.repository');
  if (key === 'image_tag') return t('build.jobs.filters.imageTag');
  if (key === 'builder_id') return t('build.jobs.filters.builder');
  return t('build.jobs.filters.status');
}
const columns = computed<NonNullable<TableProps['columns']>>(() => [
  { colKey: 'build_id', title: t('build.jobs.columns.build'), ellipsis: true, width: 145 },
  { colKey: 'snapshot', title: t('build.jobs.columns.snapshot'), cell: 'snapshot', ellipsis: true, width: 220 },
  {
    colKey: 'repository',
    title: t('build.jobs.columns.repository'),
    cell: 'repository',
    ellipsis: true,
    width: 190,
  },
  { colKey: 'image', title: t('build.jobs.columns.imageTag'), cell: 'image', ellipsis: true, width: 160 },
  { colKey: 'status', title: t('build.jobs.columns.status'), cell: 'status', width: 112 },
  { colKey: 'progress', title: t('build.jobs.columns.progress'), cell: 'progress', width: 130 },
  { colKey: 'created_at', title: t('build.jobs.columns.createdAt'), cell: 'created_at', width: 170 },
  { colKey: 'duration', title: t('build.jobs.columns.duration'), cell: 'duration', width: 100 },
  { colKey: 'builder', title: t('build.jobs.columns.builder'), cell: 'builder', width: 130 },
  { colKey: 'actions', title: t('build.jobs.columns.actions'), cell: 'actions', width: 150 },
]);
function listQuery() {
  const range = createdRange.value;
  return {
    limit: pageSize.value,
    offset: (currentPage.value - 1) * pageSize.value,
    ...(search.value.trim() ? { search: search.value.trim() } : {}),
    ...(filters.image_repository ? { image_repository: filters.image_repository } : {}),
    ...(filters.image_tag ? { image_tag: filters.image_tag } : {}),
    ...(filters.status ? { build_status: filters.status } : {}),
    ...(filters.builder_id ? { builder_id: filters.builder_id } : {}),
    ...(range[0] ? { created_after: range[0] } : {}),
    ...(range[1] ? { created_before: range[1] } : {}),
  };
}
async function load() {
  const sequence = ++requestSequence;
  loading.value = true;
  errorMessage.value = '';
  try {
    const page = await getBuildJobs(listQuery());
    if (sequence !== requestSequence) return;
    items.value = page.items as BuildJobSummary[];
    total.value = page.total;
  } catch (error) {
    if (sequence === requestSequence) errorMessage.value = buildServiceErrorMessage(error);
  } finally {
    if (sequence === requestSequence) loading.value = false;
  }
}
function applySearch() {
  currentPage.value = 1;
  void load();
}
function applyFilters() {
  filterVisible.value = false;
  currentPage.value = 1;
  void load();
}
function resetFilters() {
  Object.assign(filters, {
    image_repository: '',
    image_tag: '',
    status: '',
    builder_id: undefined,
  });
  createdRange.value = [];
  filterVisible.value = false;
  currentPage.value = 1;
  void load();
}
function resetAllQueries() {
  search.value = '';
  resetFilters();
}
function clearFilter(key: string) {
  if (key === 'created_time') {
    createdRange.value = [];
    applyFilters();
    return;
  }
  if (key === 'builder_id') {
    filters.builder_id = undefined;
  } else if (key === 'status') {
    filters.status = '';
  } else if (key === 'image_repository') {
    filters.image_repository = '';
  } else if (key === 'image_tag') {
    filters.image_tag = '';
  }
  applyFilters();
}
function changePage(info: { current: number; pageSize: number }) {
  currentPage.value = info.current;
  pageSize.value = info.pageSize;
  void load();
}
function openTask(id: number) {
  taskId.value = id;
  taskVisible.value = true;
}
async function openDetail(id: string) {
  const sequence = ++detailRequestSequence;
  detailVisible.value = true;
  detailLoading.value = true;
  detailError.value = '';
  try {
    const nextDetail = await getBuildJob(id);
    if (sequence !== detailRequestSequence) return;
    detail.value = nextDetail;
  } catch (error) {
    if (sequence === detailRequestSequence) detailError.value = buildServiceErrorMessage(error);
  } finally {
    if (sequence === detailRequestSequence) detailLoading.value = false;
  }
}
function imageReference(row: BuildJobSummary) {
  return `${row.image_repository}:${row.image_tag}`;
}
function statusLabel(status?: TaskStatus) {
  return t(`build.jobs.status.${productStatus(status)}`);
}
function productStatus(status?: TaskStatus) {
  if (status === 'pending' || status === 'ready' || status === 'scheduled') return 'queued';
  if (status === 'failed' || status === 'needs_attention') return 'failed';
  return status || 'unknown';
}
function statusTheme(status?: TaskStatus) {
  const product = productStatus(status);
  return product === 'success'
    ? 'success'
    : product === 'failed'
      ? 'danger'
      : product === 'running'
        ? 'primary'
        : product === 'cancelled'
          ? 'warning'
          : 'default';
}
function progressPercent(row: BuildJobSummary) {
  const execution = row.execution;
  return execution?.stage_count ? Math.round((execution.completed_stage_count / execution.stage_count) * 100) : 0;
}
function progressLabel(row: BuildJobSummary) {
  const execution = row.execution;
  return execution?.stage_count ? `${execution.completed_stage_count}/${execution.stage_count}` : '-';
}
function durationLabel(row: BuildJobSummary) {
  const ms = row.execution?.duration_ms;
  return ms ? `${Math.max(1, Math.round(ms / 1000))}s` : '-';
}
function buildServiceErrorMessage(_error: unknown) {
  // 传输细节仅在受控诊断面展示，避免将 Axios 或 HTTP 实现泄露给操作员。
  return t('build.jobs.error.unavailable');
}
void load();
</script>
<style scoped lang="less">
.build-jobs-page {
  display: grid;
  gap: var(--graft-density-gap-16);
  min-width: 0;
}

.build-jobs-page__search {
  flex: 0 1 clamp(240px, 32vw, 360px);
  min-width: min(100%, 220px);
}

.build-jobs-page__chips {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
}

.build-jobs-page__inline-error {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  padding: var(--graft-density-gap-16) var(--graft-density-gap-20);
}

.build-jobs-page__inline-error :deep(.t-alert) {
  flex: 1 1 auto;
  min-width: 0;
}

.build-jobs-page__ellipsis {
  display: block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.build-jobs-page__snapshot {
  display: grid;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.build-jobs-page__snapshot small {
  color: var(--td-text-color-secondary);
}

.build-jobs-page__progress {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-8);
  min-width: 100px;
}

.build-jobs-page__progress :deep(.t-progress) {
  flex: 1;
}

.build-jobs-page__row-actions {
  display: inline-flex;
  gap: var(--graft-density-gap-4);
}

@media (width <= 768px) {
  .build-jobs-page__inline-error {
    align-items: stretch;
    flex-direction: column;
    padding: var(--graft-density-gap-16);
  }

  .build-jobs-page__search {
    flex-basis: 100%;
    width: 100%;
  }
}
</style>
