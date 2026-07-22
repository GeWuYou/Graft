<template>
  <t-badge v-if="canRead" :dot="discoveryStore.hasUpdate">
    <t-tooltip placement="bottom" :content="tooltip">
      <t-button
        data-testid="update-notification"
        theme="default"
        shape="square"
        variant="text"
        :aria-label="tooltip"
        @click="visible = true"
      >
        <cloud-download-icon />
      </t-button>
    </t-tooltip>
  </t-badge>
  <update-preview-dialog
    v-model:visible="visible"
    :status="discoveryStore.status"
    :can-start-upgrade="canStartUpgrade"
    @view-management="openManagement"
    @start-upgrade="startUpgrade"
  />
</template>
<script setup lang="ts">
// 顶部入口只消费 Provider 已加载的模块状态，不承担 discovery 请求职责。
import { CloudDownloadIcon } from 'tdesign-icons-vue-next';
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { usePermissionStore } from '@/store';

import { UPDATE_ROUTE_PATH } from '../contract/paths';
import { UPDATE_PERMISSION_CODE } from '../contract/permissions';
import { useUpdateDiscoveryStore } from '../store/discovery';
import UpdatePreviewDialog from './UpdatePreviewDialog.vue';

const { t } = useI18n();
const router = useRouter();
const permissionStore = usePermissionStore();
const discoveryStore = useUpdateDiscoveryStore();
const visible = ref(false);
const canRead = computed(() => permissionStore.hasPermission(UPDATE_PERMISSION_CODE.READ));
const canStartUpgrade = computed(
  () =>
    Boolean(discoveryStore.status?.latest) &&
    !discoveryStore.status?.cache_stale &&
    !discoveryStore.status?.check_error &&
    discoveryStore.status?.installation_profile.capability === 'compose_upgrade_available' &&
    permissionStore.hasPermission(UPDATE_PERMISSION_CODE.MANAGE),
);
const tooltip = computed(() =>
  discoveryStore.hasUpdate
    ? t('update.notification.available', { version: discoveryStore.status?.latest?.version })
    : t('update.notification.open'),
);
function openManagement() {
  visible.value = false;
  void router.push(UPDATE_ROUTE_PATH.CENTER);
}

function startUpgrade() {
  visible.value = false;
  void router.push({ path: UPDATE_ROUTE_PATH.CENTER, query: { upgrade: '1' } });
}
</script>
