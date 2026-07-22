<template>
  <t-tooltip v-if="canRead && versionLabel" placement="bottom" :content="tooltip">
    <span class="update-version-entry" :aria-label="tooltip">
      {{ versionLabel }}
      <span v-if="discoveryStore.hasUpdate" class="update-version-entry__indicator" aria-hidden="true" />
    </span>
  </t-tooltip>
</template>
<script setup lang="ts">
// 品牌区只展示壳层已加载的当前版本，避免在导航中重复请求更新发现接口。
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import { usePermissionStore } from '@/store';

import { UPDATE_PERMISSION_CODE } from '../contract/permissions';
import { useUpdateDiscoveryStore } from '../store/discovery';

const { t } = useI18n();
const permissionStore = usePermissionStore();
const discoveryStore = useUpdateDiscoveryStore();
const canRead = computed(() => permissionStore.hasPermission(UPDATE_PERMISSION_CODE.READ));
const versionLabel = computed(() => discoveryStore.status?.current_version ?? '');
const tooltip = computed(() =>
  discoveryStore.hasUpdate
    ? t('update.versionEntry.updateAvailable', { version: discoveryStore.status?.latest?.version })
    : t('update.versionEntry.current', { version: versionLabel.value }),
);
</script>
<style scoped lang="less">
.update-version-entry {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: inline-flex;
  font: var(--td-font-body-small);
  font-variant-numeric: tabular-nums;
  gap: var(--td-comp-margin-xs);
  margin-left: 0;
  white-space: nowrap;
}

.update-version-entry__indicator {
  background: var(--td-success-color);
  border-radius: 50%;
  height: 6px;
  width: 6px;
}
</style>
