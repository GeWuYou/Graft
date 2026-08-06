<template>
  <section class="build-artifacts-page" data-page-type="list-form-detail">
    <management-page-header
      compact
      title-key="build.artifacts.title"
      :title="t('build.artifacts.title')"
      description-key="build.artifacts.description"
      :description="t('build.artifacts.description')"
      :source="{ labelKey: 'build.artifacts.eyebrow', fallback: t('build.artifacts.eyebrow') }"
    />

    <management-toolbar>
      <template #actions>
        <t-button variant="outline" :loading="loading" @click="load">
          <template #icon><refresh-icon /></template>
          {{ t('build.artifacts.refresh') }}
        </t-button>
      </template>
    </management-toolbar>

    <management-table-card v-if="errorMessage" class="build-artifacts-page__table-card">
      <t-space class="build-artifacts-page__error" direction="vertical" size="large" align="start">
        <t-alert theme="error" :message="errorMessage" />
        <t-button variant="outline" @click="load">{{ t('build.artifacts.retry') }}</t-button>
      </t-space>
    </management-table-card>

    <management-paged-table
      v-else
      v-model:current="currentPage"
      v-model:page-size="pageSize"
      class="build-artifacts-page__table-card"
      :columns="columns"
      density-scope="viewport"
      :empty-description="t('build.artifacts.emptyDescription')"
      :empty-title="t('build.artifacts.empty')"
      :footer-summary="t('build.artifacts.summary', { count: total })"
      :loading="loading"
      :rows="items"
      :total="total"
      row-key="artifact_id"
      :pagination-props="{ showPageSize: true }"
      :page-size-options="[20, 50, 100]"
      :cell-slot-names="['digest', 'platforms', 'size_bytes', 'created_at']"
      @page-change="changePage"
    >
      <template #digest="{ row }">
        <code class="build-artifacts-page__digest" :title="(row as BuildArtifact).digest">{{
          (row as BuildArtifact).digest
        }}</code>
      </template>
      <template #platforms="{ row }">
        <t-space break-line size="small">
          <t-tag v-for="platform in (row as BuildArtifact).platforms" :key="platform" variant="light-outline">
            {{ platform }}
          </t-tag>
        </t-space>
      </template>
      <template #size_bytes="{ row }">{{ formatBytes((row as BuildArtifact).size_bytes) }}</template>
      <template #created_at="{ row }">{{ formatLocaleDateTime((row as BuildArtifact).created_at, locale) }}</template>
    </management-paged-table>
  </section>
</template>
<script setup lang="ts">
// 构建产物页只呈现不可变 Artifact 投影，不以构建任务、标签或仓库引用替代产物身份。
import { RefreshIcon } from 'tdesign-icons-vue-next';
import type { TableProps } from 'tdesign-vue-next';
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { ManagementPageHeader, ManagementTableCard, ManagementToolbar } from '@/shared/components/management';
import ManagementPagedTable from '@/shared/components/management/ManagementPagedTable.vue';
import { formatBytes, formatLocaleDateTime } from '@/shared/observability';

import { getBuildArtifacts } from '../../api/build';
import type { BuildArtifact } from '../../types/build';

const { locale, t } = useI18n();
const items = ref<BuildArtifact[]>([]);
const total = ref(0);
const currentPage = ref(1);
const pageSize = ref(20);
const loading = ref(false);
const errorMessage = ref('');
let requestSequence = 0;

const columns = computed<NonNullable<TableProps['columns']>>(() => [
  { colKey: 'digest', title: t('build.artifacts.columns.digest'), cell: 'digest', ellipsis: true, minWidth: 250 },
  { colKey: 'media_type', title: t('build.artifacts.columns.mediaType'), ellipsis: true, minWidth: 180 },
  { colKey: 'platforms', title: t('build.artifacts.columns.platforms'), cell: 'platforms', minWidth: 180 },
  { colKey: 'size_bytes', title: t('build.artifacts.columns.size'), cell: 'size_bytes', width: 120 },
  { colKey: 'created_at', title: t('build.artifacts.columns.createdAt'), cell: 'created_at', width: 180 },
]);

async function load() {
  const sequence = ++requestSequence;
  loading.value = true;
  errorMessage.value = '';
  try {
    const page = await getBuildArtifacts({
      limit: pageSize.value,
      offset: (currentPage.value - 1) * pageSize.value,
    });
    if (sequence !== requestSequence) return;
    items.value = page.items;
    total.value = page.total;
  } catch {
    if (sequence === requestSequence) errorMessage.value = t('build.artifacts.loadFailed');
  } finally {
    if (sequence === requestSequence) loading.value = false;
  }
}
function changePage(info: { current: number; pageSize: number }) {
  currentPage.value = info.current;
  pageSize.value = info.pageSize;
  void load();
}

void load();
</script>
<style scoped lang="less">
.build-artifacts-page {
  display: grid;
  gap: var(--graft-density-gap-16);
  min-width: 0;
}

.build-artifacts-page__error {
  display: flex;
  padding: var(--graft-density-gap-16) var(--graft-density-gap-20);
  width: 100%;
}

.build-artifacts-page__digest {
  display: block;
  font-family: var(--td-font-family);
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
