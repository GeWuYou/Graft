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
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { formatLocaleDateTime } from '@/shared/observability';

import { getBuildJob, getBuildJobs } from '../../api/build';
import { BUILD_ROUTE_PATH } from '../../contract/paths';
import type { BuildJobDetail, BuildJobSummary } from '../../types/build';
const { locale, t } = useI18n();
const router = useRouter();
const items = ref<BuildJobSummary[]>([]);
const loading = ref(false);
const errorMessage = ref('');
const detail = ref<BuildJobDetail>();
const detailLoading = ref(false);
const detailVisible = ref(false);
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
async function load() {
  loading.value = true;
  errorMessage.value = '';
  try {
    items.value = (await getBuildJobs({ limit: 50, offset: 0 })).items;
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('build.jobs.loadFailed'));
  } finally {
    loading.value = false;
  }
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

.build-jobs-page h1,
.build-jobs-page p {
  margin: 0;
}

.build-jobs-page p {
  color: var(--td-text-color-secondary);
  margin-top: var(--graft-density-gap-4);
}
</style>
