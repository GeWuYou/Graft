<template>
  <section class="build-workspaces-page" data-page-type="list-form-detail">
    <management-page-header
      compact
      title-key="build.workspaces.title"
      :title="t('build.workspaces.title')"
      description-key="build.workspaces.description"
      :description="t('build.workspaces.description')"
      :source="{ labelKey: 'build.workspaces.eyebrow', fallback: t('build.workspaces.eyebrow') }"
    />

    <management-toolbar>
      <template #actions>
        <t-button variant="outline" :loading="loading" @click="load">
          <template #icon><refresh-icon /></template>
          {{ t('build.workspaces.refresh') }}
        </t-button>
        <t-button v-permission="BUILD_PERMISSION_CODE.CREATE" theme="primary" @click="openCreate">
          <template #icon><add-icon /></template>
          {{ t('build.workspaces.create.title') }}
        </t-button>
      </template>
    </management-toolbar>

    <management-table-card v-if="errorMessage" class="build-workspaces-page__table-card">
      <t-space class="build-workspaces-page__error" direction="vertical" size="large" align="start">
        <t-alert theme="error" :message="errorMessage" />
        <t-button variant="outline" @click="load">{{ t('build.workspaces.retry') }}</t-button>
      </t-space>
    </management-table-card>

    <management-paged-table
      v-else
      v-model:current="currentPage"
      v-model:page-size="pageSize"
      class="build-workspaces-page__table-card"
      :columns="columns"
      density-scope="viewport"
      :empty-description="t('build.workspaces.emptyDescription')"
      :empty-title="t('build.workspaces.empty')"
      :footer-summary="t('build.workspaces.summary', { count: items.length })"
      :loading="loading"
      :rows="items"
      :total="items.length"
      row-key="workspace_id"
      :pagination-visible="false"
      :cell-slot-names="['source_kind', 'source_reference', 'created_at', 'updated_at']"
    >
      <template #source_kind="{ row }">
        {{ t(`build.workspaces.sourceKinds.${(row as BuildWorkspace).source_kind}`) }}
      </template>
      <template #source_reference="{ row }">
        <span :title="(row as BuildWorkspace).source_reference">{{ applicationLabel(row as BuildWorkspace) }}</span>
      </template>
      <template #created_at="{ row }">{{ formatLocaleDateTime((row as BuildWorkspace).created_at, locale) }}</template>
      <template #updated_at="{ row }">{{ formatLocaleDateTime((row as BuildWorkspace).updated_at, locale) }}</template>
      <template #feedback>
        <t-alert v-if="applicationError" theme="warning" :message="applicationError" />
      </template>
    </management-paged-table>
  </section>
</template>
<script setup lang="ts">
// 工作区列表以 Build 服务端投影为准，应用名称仅作为来源引用的可读补充，缺失时保留原始 ID。
import { AddIcon, RefreshIcon } from 'tdesign-icons-vue-next';
import type { TableProps } from 'tdesign-vue-next';
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { getApplications } from '@/modules/project/api/project';
import { ManagementPageHeader, ManagementTableCard, ManagementToolbar } from '@/shared/components/management';
import ManagementPagedTable from '@/shared/components/management/ManagementPagedTable.vue';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { formatLocaleDateTime } from '@/shared/observability';

import { getBuildWorkspaces } from '../../api/build';
import { BUILD_ROUTE_PATH } from '../../contract/paths';
import { BUILD_PERMISSION_CODE } from '../../contract/permissions';
import type { BuildWorkspace } from '../../types/build';

const { locale, t } = useI18n();
const router = useRouter();
const items = ref<BuildWorkspace[]>([]);
const applicationLabels = ref(new Map<string, string>());
const loading = ref(false);
const errorMessage = ref('');
const applicationError = ref('');
const currentPage = ref(1);
const pageSize = ref(20);

const columns = computed<NonNullable<TableProps['columns']>>(() => [
  { colKey: 'name', title: t('build.workspaces.columns.name'), minWidth: 220, ellipsis: true },
  { colKey: 'source_kind', title: t('build.workspaces.columns.sourceKind'), cell: 'source_kind', width: 180 },
  {
    colKey: 'source_reference',
    title: t('build.workspaces.columns.sourceApplication'),
    cell: 'source_reference',
    minWidth: 240,
    ellipsis: true,
  },
  { colKey: 'retention_policy', title: t('build.workspaces.columns.retentionPolicy'), minWidth: 160 },
  { colKey: 'created_at', title: t('build.workspaces.columns.createdAt'), cell: 'created_at', width: 180 },
  { colKey: 'updated_at', title: t('build.workspaces.columns.updatedAt'), cell: 'updated_at', width: 180 },
]);

function applicationLabel(workspace: BuildWorkspace) {
  return (
    applicationLabels.value.get(workspace.source_reference) ??
    t('build.workspaces.applicationUnavailable', { id: workspace.source_reference })
  );
}

function openCreate() {
  void router.push(BUILD_ROUTE_PATH.CREATE_WORKSPACE);
}

async function load() {
  loading.value = true;
  errorMessage.value = '';
  applicationError.value = '';
  const [workspaceResult, applicationResult] = await Promise.allSettled([
    getBuildWorkspaces(),
    getApplications({ limit: 100, offset: 0 }),
  ]);
  if (workspaceResult.status === 'fulfilled') {
    items.value = workspaceResult.value.items ?? [];
  } else {
    items.value = [];
    errorMessage.value = resolveLocalizedErrorMessage(t, workspaceResult.reason, t('build.workspaces.loadFailed'));
  }
  if (applicationResult.status === 'fulfilled') {
    applicationLabels.value = new Map(
      (applicationResult.value.items ?? []).map((application) => [
        application.application_id,
        application.display_name,
      ]),
    );
  } else {
    applicationLabels.value = new Map();
    applicationError.value = resolveLocalizedErrorMessage(
      t,
      applicationResult.reason,
      t('build.workspaces.applicationLoadFailed'),
    );
  }
  loading.value = false;
}

onMounted(() => void load());
</script>
<style scoped lang="less">
.build-workspaces-page {
  display: grid;
  gap: var(--graft-density-gap-16);
  min-width: 0;
}

.build-workspaces-page__error {
  display: flex;
  padding: var(--graft-density-gap-16) var(--graft-density-gap-20);
  width: 100%;
}
</style>
