<template>
  <section class="build-jobs-page">
    <header class="build-jobs-page__header">
      <div>
        <h1>{{ t('build.jobs.title') }}</h1>
        <p>{{ t('build.jobs.description') }}</p>
      </div>
      <t-space
        ><t-button variant="outline" :loading="loading" @click="load">{{ t('build.jobs.refresh') }}</t-button
        ><t-button theme="primary" @click="router.push(BUILD_ROUTE_PATH.CREATE)">{{
          t('build.jobs.create.title')
        }}</t-button></t-space
      >
    </header>
    <t-alert v-if="errorMessage" theme="error" :message="errorMessage" />
    <t-form class="build-jobs-page__filters" layout="inline" @submit="applyFilters">
      <t-form-item :label="t('build.jobs.filters.applicationId')">
        <t-input-number v-model="filters.application_id" :min="1" :placeholder="t('build.jobs.filters.all')" />
      </t-form-item>
      <t-form-item :label="t('build.jobs.create.repository')">
        <t-input v-model="filters.image_repository" :placeholder="t('build.jobs.filters.all')" clearable />
      </t-form-item>
      <t-form-item :label="t('build.jobs.create.tag')">
        <t-input v-model="filters.image_tag" :placeholder="t('build.jobs.filters.all')" clearable />
      </t-form-item>
      <t-form-item :label="t('build.jobs.filters.createdAfter')">
        <t-input v-model="filters.created_after" :placeholder="t('build.jobs.filters.rfc3339')" clearable />
      </t-form-item>
      <t-form-item :label="t('build.jobs.filters.createdBefore')">
        <t-input v-model="filters.created_before" :placeholder="t('build.jobs.filters.rfc3339')" clearable />
      </t-form-item>
      <t-form-item
        ><t-button theme="primary" type="submit">{{ t('build.jobs.filters.apply') }}</t-button></t-form-item
      >
      <t-form-item
        ><t-button variant="outline" @click="resetFilters">{{ t('build.jobs.filters.reset') }}</t-button></t-form-item
      >
    </t-form>
    <t-table
      row-key="build_id"
      :columns="columns"
      :data="items"
      :loading="loading"
      :empty="t('build.jobs.empty')"
      @row-click="({ row }) => openDetail((row as BuildJobSummary).build_id)"
    >
      <template #image_repository="{ row }"
        >{{ (row as BuildJobSummary).image_repository }}:{{ (row as BuildJobSummary).image_tag }}</template
      >
      <template #artifact="{ row }"
        ><t-tag v-if="(row as BuildJobSummary).artifact" theme="success" variant="light-outline">{{
          (row as BuildJobSummary).artifact?.image_id
        }}</t-tag
        ><span v-else>-</span></template
      >
      <template #created_at="{ row }">{{ formatLocaleDateTime((row as BuildJobSummary).created_at, locale) }}</template>
    </t-table>
    <t-pagination
      v-if="total > 0"
      :current="currentPage"
      :page-size="pageSize"
      :total="total"
      :page-size-options="[20, 50, 100]"
      show-page-size
      @change="changePage"
    />
    <t-drawer
      :visible="detailVisible"
      size="large"
      :header="t('build.jobs.detail.title')"
      @close="detailVisible = false"
    >
      <t-loading :loading="detailLoading"
        ><template v-if="detail"
          ><t-descriptions bordered :column="2" size="small" :title="t('build.jobs.detail.summary')"
            ><t-descriptions-item :label="t('build.jobs.columns.build')">{{ detail.build_id }}</t-descriptions-item
            ><t-descriptions-item :label="t('build.jobs.columns.application')">{{
              detail.application_name
            }}</t-descriptions-item
            ><t-descriptions-item :label="t('build.jobs.create.contextPath')">{{
              detail.context_path
            }}</t-descriptions-item
            ><t-descriptions-item :label="t('build.jobs.create.dockerfilePath')">{{
              detail.dockerfile_path
            }}</t-descriptions-item
            ><t-descriptions-item :label="t('build.jobs.create.repository')">{{
              detail.image_repository
            }}</t-descriptions-item
            ><t-descriptions-item :label="t('build.jobs.create.tag')">{{
              detail.image_tag
            }}</t-descriptions-item></t-descriptions
          ><t-descriptions
            v-if="detail.artifact"
            bordered
            :column="1"
            size="small"
            :title="t('build.jobs.detail.artifact')"
            ><t-descriptions-item :label="t('build.jobs.columns.artifact')">{{
              detail.artifact.image_id
            }}</t-descriptions-item></t-descriptions
          ><t-empty v-else :title="t('build.jobs.detail.noArtifact')" /></template
      ></t-loading>
    </t-drawer>
  </section>
</template>
<script setup lang="ts">
// Build Jobs page consumes only the Build read projection; task state and logs keep their Task-owned presentation.
import type { TableProps } from 'tdesign-vue-next';
import { computed, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { formatLocaleDateTime } from '@/shared/observability';

import { getBuildJob, getBuildJobs } from '../../api/build';
import { BUILD_ROUTE_PATH } from '../../contract/paths';
import type { BuildJobDetail, BuildJobSummary } from '../../types/build';

type JobFilters = {
  application_id?: number;
  image_repository: string;
  image_tag: string;
  created_after: string;
  created_before: string;
};

const { locale, t } = useI18n();
const router = useRouter();
const items = ref<BuildJobSummary[]>([]);
const total = ref(0);
const currentPage = ref(1);
const pageSize = ref(20);
const loading = ref(false);
const errorMessage = ref('');
const detail = ref<BuildJobDetail>();
const detailLoading = ref(false);
const detailVisible = ref(false);
const filters = reactive<JobFilters>({ image_repository: '', image_tag: '', created_after: '', created_before: '' });
const columns = computed<NonNullable<TableProps['columns']>>(() => [
  { colKey: 'build_id', title: t('build.jobs.columns.build'), ellipsis: true },
  { colKey: 'application_name', title: t('build.jobs.columns.application'), ellipsis: true },
  {
    colKey: 'image_repository',
    title: t('build.jobs.columns.image'),
    cell: 'image_repository',
    ellipsis: true,
  },
  { colKey: 'artifact', title: t('build.jobs.columns.artifact'), cell: 'artifact', ellipsis: true },
  {
    colKey: 'created_at',
    title: t('build.jobs.columns.createdAt'),
    cell: 'created_at',
    width: 188,
  },
]);
function listQuery() {
  return {
    limit: pageSize.value,
    offset: (currentPage.value - 1) * pageSize.value,
    ...(filters.application_id ? { application_id: filters.application_id } : {}),
    ...(filters.image_repository.trim() ? { image_repository: filters.image_repository.trim() } : {}),
    ...(filters.image_tag.trim() ? { image_tag: filters.image_tag.trim() } : {}),
    ...(filters.created_after.trim() ? { created_after: filters.created_after.trim() } : {}),
    ...(filters.created_before.trim() ? { created_before: filters.created_before.trim() } : {}),
  };
}
async function load() {
  loading.value = true;
  errorMessage.value = '';
  try {
    const page = await getBuildJobs(listQuery());
    items.value = page.items;
    total.value = page.total;
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('build.jobs.loadFailed'));
  } finally {
    loading.value = false;
  }
}
function applyFilters() {
  currentPage.value = 1;
  void load();
}
function resetFilters() {
  filters.application_id = undefined;
  filters.image_repository = '';
  filters.image_tag = '';
  filters.created_after = '';
  filters.created_before = '';
  currentPage.value = 1;
  void load();
}
function changePage(pageInfo: { current: number; pageSize: number }) {
  currentPage.value = pageInfo.current;
  pageSize.value = pageInfo.pageSize;
  void load();
}
async function openDetail(buildId: string) {
  detailVisible.value = true;
  detailLoading.value = true;
  try {
    detail.value = await getBuildJob(buildId);
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('build.jobs.loadFailed'));
  } finally {
    detailLoading.value = false;
  }
}
void load();
</script>
<style scoped lang="less">
.build-jobs-page {
  display: grid;
  gap: var(--graft-density-gap-16);
  min-width: 0;
}

.build-jobs-page__header {
  align-items: flex-start;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-16);
  justify-content: space-between;
}

.build-jobs-page__filters {
  align-items: end;
}

.build-jobs-page h1,
.build-jobs-page p {
  margin: 0;
}

.build-jobs-page p {
  color: var(--td-text-color-secondary);
  margin-top: var(--graft-density-gap-4);
}
</style>
