<template>
  <div class="runtime-target-page" data-page-type="list-form-detail">
    <management-page-header :title="t('runtimeTarget.list.title')" :description="t('runtimeTarget.list.description')" />
    <management-toolbar
      ><template #filters
        ><t-button theme="default" :loading="loading" @click="load">{{
          t('runtimeTarget.list.reload')
        }}</t-button></template
      ></management-toolbar
    >
    <t-table row-key="id" :data="items" :columns="columns" :loading="loading" @row-click="openDetail" />
    <t-drawer
      v-model:visible="drawerVisible"
      :header="selected?.displayName ?? t('runtimeTarget.detail.title')"
      size="480px"
    >
      <t-descriptions v-if="selected" bordered :column="1">
        <t-descriptions-item :label="t('runtimeTarget.columns.provider')">{{ selected.provider }}</t-descriptions-item>
        <t-descriptions-item :label="t('runtimeTarget.columns.endpoint')">{{
          selected.endpointLabel
        }}</t-descriptions-item>
        <t-descriptions-item :label="t('runtimeTarget.columns.capabilities')">{{
          selected.capabilities.join(', ') || '-'
        }}</t-descriptions-item>
        <t-descriptions-item :label="t('runtimeTarget.columns.status')">{{
          availabilityText(selected.availability)
        }}</t-descriptions-item>
        <t-descriptions-item :label="t('runtimeTarget.columns.checkedAt')">{{
          selected.lastCheckedAt || '-'
        }}</t-descriptions-item>
        <t-descriptions-item :label="t('runtimeTarget.columns.error')">{{
          selected.lastError || '-'
        }}</t-descriptions-item>
      </t-descriptions>
      <template #footer
        ><t-button theme="primary" :loading="refreshing" @click="refreshSelected">{{
          t('runtimeTarget.list.refresh')
        }}</t-button></template
      >
    </t-drawer>
  </div>
</template>
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import {
  getRuntimeTarget,
  listRuntimeTargets,
  refreshRuntimeTarget,
  type RuntimeTarget,
} from '../../api/runtime-target';
const { t } = useI18n();
const loading = ref(false);
const refreshing = ref(false);
const items = ref<RuntimeTarget[]>([]);
const selected = ref<RuntimeTarget | null>(null);
const drawerVisible = ref(false);
const columns = computed(() => [
  { colKey: 'displayName', title: t('runtimeTarget.columns.name') },
  { colKey: 'provider', title: t('runtimeTarget.columns.provider') },
  { colKey: 'endpointLabel', title: t('runtimeTarget.columns.endpoint') },
  { colKey: 'connectionKind', title: t('runtimeTarget.columns.connectionKind') },
  { colKey: 'availability', title: t('runtimeTarget.columns.status') },
  { colKey: 'lastCheckedAt', title: t('runtimeTarget.columns.checkedAt') },
]);
function availabilityText(value: boolean) {
  return value ? t('runtimeTarget.status.available') : t('runtimeTarget.status.unavailable');
}
async function load() {
  loading.value = true;
  try {
    items.value = await listRuntimeTargets();
  } finally {
    loading.value = false;
  }
}
async function openDetail(context: { row: unknown }) {
  const row = context.row as RuntimeTarget;
  selected.value = (await getRuntimeTarget(row.id)) ?? row;
  drawerVisible.value = true;
}
async function refreshSelected() {
  if (!selected.value) return;
  refreshing.value = true;
  try {
    const refreshed = await refreshRuntimeTarget(selected.value.id);
    if (refreshed) {
      selected.value = refreshed;
      await load();
    }
  } finally {
    refreshing.value = false;
  }
}
onMounted(load);
</script>
