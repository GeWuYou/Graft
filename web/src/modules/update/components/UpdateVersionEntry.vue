<template>
  <t-popup
    v-if="canRead && versionLabel"
    v-model:visible="visible"
    destroy-on-close
    placement="bottom-right"
    trigger="click"
    :overlay-inner-style="{ padding: '0' }"
  >
    <button class="update-version-entry" data-testid="update-version-entry" type="button" :aria-label="tooltip">
      {{ versionLabel }}
    </button>
    <template #content>
      <section class="update-version-preview">
        <header class="update-version-preview__header">
          <span>{{ t('update.preview.current') }}</span>
          <t-tooltip :content="t('update.preview.checkNow')" placement="top">
            <t-button
              shape="square"
              size="small"
              theme="default"
              variant="text"
              :disabled="!canCheck"
              :loading="checking"
              @click="refreshStatus"
            >
              <refresh-icon />
            </t-button>
          </t-tooltip>
        </header>
        <div class="update-version-preview__current">
          <strong>{{ discoveryStore.status?.current_version }}</strong>
          <p v-if="releaseAvailable" class="update-version-preview__up-to-date">{{ t('update.preview.upToDate') }}</p>
          <p v-else class="update-version-preview__unavailable">{{ t('update.preview.releaseUnavailable') }}</p>
        </div>
        <template v-if="discoveryStore.status?.latest">
          <p class="update-version-preview__available">
            {{ t('update.preview.available') }} {{ discoveryStore.status.latest.version }}
          </p>
          <p class="update-version-preview__summary">{{ summary }}</p>
        </template>
        <footer class="update-version-preview__actions">
          <t-button size="small" variant="text" :disabled="!canViewRelease" @click="openManagement">
            {{ t('update.preview.viewRelease') }}
          </t-button>
          <t-button v-if="canStartUpgrade" size="small" theme="primary" @click="startUpgrade">
            {{ t('update.preview.startUpgrade') }}
          </t-button>
        </footer>
      </section>
    </template>
  </t-popup>
</template>
<script setup lang="ts">
// 品牌区复用壳层 discovery snapshot，并把版本 Badge 作为锚定的轻量更新入口。
import { RefreshIcon } from 'tdesign-icons-vue-next';
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { usePermissionStore } from '@/store';

import { checkForUpdates } from '../api/update';
import { useUpdatePreviewActions } from '../composables/useUpdatePreviewActions';
import { UPDATE_PERMISSION_CODE } from '../contract/permissions';
import { useUpdateDiscoveryStore } from '../store/discovery';

const { t } = useI18n();
const permissionStore = usePermissionStore();
const discoveryStore = useUpdateDiscoveryStore();
const visible = ref(false);
const checking = ref(false);
const canRead = computed(() => permissionStore.hasPermission(UPDATE_PERMISSION_CODE.READ));
const canCheck = computed(() => permissionStore.hasPermission(UPDATE_PERMISSION_CODE.CHECK));
const versionLabel = computed(() => discoveryStore.status?.current_version ?? '');
const { canStartUpgrade, openManagement, startUpgrade } = useUpdatePreviewActions(visible);
const releaseAvailable = computed(
  () =>
    Boolean(discoveryStore.status?.latest) &&
    !discoveryStore.status?.cache_stale &&
    !discoveryStore.status?.check_error,
);
const canViewRelease = computed(() => releaseAvailable.value);
const tooltip = computed(() =>
  discoveryStore.hasUpdate
    ? t('update.versionEntry.updateAvailable', { version: discoveryStore.status?.latest?.version })
    : t('update.versionEntry.openCenter', { version: versionLabel.value }),
);
const summary = computed(
  () =>
    discoveryStore.status?.latest?.upgrade_notes ||
    discoveryStore.status?.latest?.notes ||
    t('update.preview.summaryEmpty'),
);

async function refreshStatus() {
  if (!canCheck.value) {
    return;
  }
  checking.value = true;
  try {
    discoveryStore.replaceSnapshot(await checkForUpdates());
  } finally {
    checking.value = false;
  }
}
</script>
<style scoped lang="less">
.update-version-entry {
  align-items: center;
  appearance: none;
  background: var(--td-bg-color-secondarycontainer);
  border: 1px solid var(--td-component-stroke);
  border-radius: 4px;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  display: inline-flex;
  font: var(--td-font-body-small);
  font-variant-numeric: tabular-nums;
  font-weight: 600;
  line-height: 16px;
  margin-right: var(--td-comp-paddingLR-m);
  min-height: 18px;
  padding: 0 var(--td-comp-paddingLR-xs);
  white-space: nowrap;
}

.update-version-preview {
  display: grid;
  gap: var(--td-comp-margin-s);
  min-width: 336px;
  padding: var(--td-comp-paddingTB-l) var(--td-comp-paddingLR-l);
}

.update-version-preview__header {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: flex;
  font: var(--td-font-body-small);
  justify-content: space-between;
}

.update-version-preview__current {
  display: grid;
  padding: var(--td-comp-paddingTB-l) 0;
  place-items: center center;
}

.update-version-preview__current strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-display-medium);
  font-variant-numeric: tabular-nums;
}

.update-version-preview__available {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  margin: 0;
}

.update-version-preview__summary {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0;
  white-space: pre-line;
}

.update-version-preview__actions {
  border-top: 1px solid var(--td-component-stroke);
  display: flex;
  gap: var(--td-comp-margin-s);
  justify-content: flex-end;
  padding-top: var(--td-comp-paddingTB-s);
}

.update-version-preview__up-to-date {
  color: var(--td-success-color);
  font: var(--td-font-body-small);
  margin: 0;
}

.update-version-preview__unavailable {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0;
}
</style>
