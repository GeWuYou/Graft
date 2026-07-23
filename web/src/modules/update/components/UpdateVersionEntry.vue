<template>
  <t-tooltip v-if="canRead" placement="right" :content="tooltip">
    <t-button
      class="update-version-entry"
      variant="text"
      size="small"
      :loading="loading"
      :aria-label="tooltip"
      @click.stop="openCenter"
    >
      <span class="update-version-entry__label">{{ versionLabel }}</span>
      <span v-if="hasUpdate" class="update-version-entry__indicator" aria-hidden="true" />
    </t-button>
  </t-tooltip>
</template>
<script setup lang="ts">
// 侧栏版本入口只读取已授权的发现快照，避免在壳层复制更新模块的状态或权限判断。
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { usePermissionStore } from '@/store';

import { getUpdateStatus } from '../api/update';
import { UPDATE_ROUTE_PATH } from '../contract/paths';
import { UPDATE_PERMISSION_CODE } from '../contract/permissions';
import type { UpdateStatus } from '../types/update';

const { t } = useI18n();
const router = useRouter();
const permissionStore = usePermissionStore();
const loading = ref(false);
const status = ref<UpdateStatus | null>(null);
const hasLoaded = ref(false);
const canRead = computed(() => permissionStore.hasPermission(UPDATE_PERMISSION_CODE.READ));
const hasUpdate = computed(() => Boolean(status.value?.latest));
const versionLabel = computed(() => status.value?.current_version || t('update.versionEntry.unavailable'));
const tooltip = computed(() =>
  hasUpdate.value
    ? t('update.versionEntry.updateAvailable', { version: status.value?.latest?.version })
    : t('update.versionEntry.openCenter', { version: versionLabel.value }),
);

// 权限快照在壳层异步恢复后才可用；只在首次取得读取权限时加载，避免重复 hydration 产生并发请求。
watch(
  canRead,
  (allowed) => {
    if (allowed) {
      void loadStatus();
    }
  },
  { immediate: true },
);

async function loadStatus() {
  if (hasLoaded.value || loading.value) {
    return;
  }

  hasLoaded.value = true;
  loading.value = true;
  try {
    status.value = await getUpdateStatus();
  } catch {
    status.value = null;
  } finally {
    loading.value = false;
  }
}

function openCenter() {
  void router.push(UPDATE_ROUTE_PATH.CENTER);
}
</script>
<style scoped lang="less">
.update-version-entry {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: inline-flex;
  font-variant-numeric: tabular-nums;
  justify-content: center;
  margin-left: calc(var(--td-comp-size-l) + var(--td-comp-margin-s));
  min-height: 24px;
  padding: 0 var(--td-comp-paddingLR-s);
  width: calc(100% - var(--td-comp-size-l) - var(--td-comp-margin-l));
}

.update-version-entry__label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.update-version-entry__indicator {
  background: var(--td-success-color);
  border-radius: 50%;
  display: inline-block;
  height: 6px;
  margin-left: var(--td-comp-margin-xs);
  width: 6px;
}
</style>
