<template>
  <div class="docker-resources-page" data-page-type="list-form-detail">
    <management-page-header
      :title="t('container.resources.title')"
      :description="t('container.resources.description')"
    />
    <management-toolbar>
      <template #actions>
        <t-button theme="default" variant="outline" :loading="loading" @click="refreshActiveResource">
          <template #icon><refresh-icon /></template>
          {{ t('container.images.actions.refresh') }}
        </t-button>
      </template>
    </management-toolbar>
    <t-tabs v-model:value="active" theme="normal">
      <t-tab-panel value="containers" :label="t('container.resources.tabs.containers')"
        ><t-button theme="primary" variant="text" @click="router.push(CONTAINER_ROUTE_PATH.LIST)">{{
          t('container.resources.openContainers')
        }}</t-button></t-tab-panel
      >
      <t-tab-panel value="networks" :label="t('container.resources.tabs.networks')">
        <t-table row-key="id" :data="networks" :columns="networkColumns" :loading="loading">
          <template #empty>
            <t-empty :description="t('container.resources.description')" />
          </template>
        </t-table>
      </t-tab-panel>
      <t-tab-panel value="volumes" :label="t('container.resources.tabs.volumes')">
        <t-table row-key="name" :data="volumes" :columns="volumeColumns" :loading="loading">
          <template #empty>
            <t-empty :description="t('container.resources.description')" />
          </template>
        </t-table>
      </t-tab-panel>
      <t-tab-panel value="system" :label="t('container.resources.tabs.system')"
        ><t-descriptions bordered :column="2"
          ><t-descriptions-item v-for="item in systemItems" :key="item.label" :label="item.label">{{
            item.value
          }}</t-descriptions-item></t-descriptions
        ></t-tab-panel
      >
    </t-tabs>
  </div>
</template>
<script setup lang="ts">
import { RefreshIcon } from 'tdesign-icons-vue-next';
import type { TableProps } from 'tdesign-vue-next';
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { ManagementPageHeader, ManagementToolbar } from '@/shared/components/management';

import { CONTAINER_ROUTE_PATH } from '../../contract/paths';
import { type DockerResourceTab, useDockerResourceQueries } from '../../shared/container-resource-queries';

const { t } = useI18n();
const router = useRouter();

/** 此页面仅展示按 tab 激活的 Docker 静态资源快照，不承载容器实时运行状态。 */
const active = ref<DockerResourceTab>('containers');
const { networks: networksQuery, system: systemQuery, volumes: volumesQuery } = useDockerResourceQueries(active);

const networks = computed(() => networksQuery.data.value?.items ?? []);
const volumes = computed(() => volumesQuery.data.value?.items ?? []);
const system = computed(() => systemQuery.data.value ?? {});
const loading = computed(() => {
  switch (active.value) {
    case 'networks':
      return networksQuery.isFetching.value;
    case 'volumes':
      return volumesQuery.isFetching.value;
    case 'system':
      return systemQuery.isFetching.value;
    default:
      return false;
  }
});

function refreshActiveResource() {
  switch (active.value) {
    case 'networks':
      void networksQuery.refetch();
      break;
    case 'volumes':
      void volumesQuery.refetch();
      break;
    case 'system':
      void systemQuery.refetch();
      break;
  }
}

const networkColumns: TableProps['columns'] = [
  { colKey: 'name', title: t('container.resources.columns.name') },
  { colKey: 'driver', title: t('container.resources.columns.driver') },
  { colKey: 'scope', title: t('container.resources.columns.scope') },
];
const volumeColumns: TableProps['columns'] = [
  { colKey: 'name', title: t('container.resources.columns.name') },
  { colKey: 'driver', title: t('container.resources.columns.driver') },
  { colKey: 'scope', title: t('container.resources.columns.scope') },
];
const systemItems = computed(() =>
  Object.entries(system.value).map(([label, value]) => ({
    label,
    value: Array.isArray(value) ? value.join(', ') : String(value ?? '-'),
  })),
);
</script>
