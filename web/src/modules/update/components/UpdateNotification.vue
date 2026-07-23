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

import { usePermissionStore } from '@/store';

import { useUpdatePreviewActions } from '../composables/useUpdatePreviewActions';
import { UPDATE_PERMISSION_CODE } from '../contract/permissions';
import { useUpdateDiscoveryStore } from '../store/discovery';
import UpdatePreviewDialog from './UpdatePreviewDialog.vue';

const { t } = useI18n();
const permissionStore = usePermissionStore();
const discoveryStore = useUpdateDiscoveryStore();
const visible = ref(false);
const canRead = computed(() => permissionStore.hasPermission(UPDATE_PERMISSION_CODE.READ));
const { canStartUpgrade, openManagement, startUpgrade } = useUpdatePreviewActions(visible);
const tooltip = computed(() =>
  discoveryStore.hasUpdate
    ? t('update.notification.available', { version: discoveryStore.status?.latest?.version })
    : t('update.notification.open'),
);
</script>
