<template>
  <div class="docker-resources-page" data-page-type="list-form-detail">
    <management-page-header
      :title="t('container.resources.title')"
      :description="t('container.resources.description')"
    />
    <t-tabs v-model:value="active" theme="normal">
      <t-tab-panel value="containers" :label="t('container.resources.tabs.containers')"
        ><t-button theme="primary" variant="text" @click="router.push(CONTAINER_ROUTE_PATH.LIST)">{{
          t('container.resources.openContainers')
        }}</t-button></t-tab-panel
      >
      <t-tab-panel value="images" :label="t('container.resources.tabs.images')"
        ><t-table row-key="id" :data="images" :columns="imageColumns" :loading="loading"
      /></t-tab-panel>
      <t-tab-panel value="networks" :label="t('container.resources.tabs.networks')"
        ><t-table row-key="id" :data="networks" :columns="networkColumns" :loading="loading"
      /></t-tab-panel>
      <t-tab-panel value="volumes" :label="t('container.resources.tabs.volumes')"
        ><t-table row-key="name" :data="volumes" :columns="volumeColumns" :loading="loading"
      /></t-tab-panel>
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
import type { TableProps } from 'tdesign-vue-next';
import { computed, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { ManagementPageHeader } from '@/shared/components/management';

import { getDockerImages, getDockerNetworks, getDockerSystem, getDockerVolumes } from '../../api/container';
import { CONTAINER_ROUTE_PATH } from '../../contract/paths';
const { t } = useI18n();
const router = useRouter();
const active = ref('containers');
const loading = ref(false);
const images = ref<any[]>([]),
  networks = ref<any[]>([]),
  volumes = ref<any[]>([]),
  system = ref<Record<string, unknown>>({});
const imageColumns: TableProps['columns'] = [
  { colKey: 'id', title: 'ID' },
  { colKey: 'repository_tags', title: t('container.resources.columns.tags') },
  { colKey: 'size_bytes', title: t('container.resources.columns.size') },
];
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
async function load() {
  loading.value = true;
  try {
    if (active.value === 'images') images.value = (await getDockerImages()).items ?? [];
    if (active.value === 'networks') networks.value = (await getDockerNetworks()).items ?? [];
    if (active.value === 'volumes') volumes.value = (await getDockerVolumes()).items ?? [];
    if (active.value === 'system') system.value = await getDockerSystem();
  } finally {
    loading.value = false;
  }
}
watch(active, load);
onMounted(load);
</script>
