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
        <p class="update-version-preview__eyebrow">{{ t('update.preview.current') }}</p>
        <strong>{{ discoveryStore.status?.current_version }}</strong>
        <template v-if="discoveryStore.status?.latest">
          <p class="update-version-preview__eyebrow">{{ t('update.preview.available') }}</p>
          <strong>{{ discoveryStore.status.latest.version }}</strong>
          <p class="update-version-preview__summary">{{ summary }}</p>
          <div class="update-version-preview__actions">
            <t-button size="small" variant="text" @click="openManagement">
              {{ t('update.preview.viewManagement') }}
            </t-button>
            <t-button v-if="canStartUpgrade" size="small" theme="primary" @click="startUpgrade">
              {{ t('update.preview.startUpgrade') }}
            </t-button>
          </div>
        </template>
        <p v-else class="update-version-preview__up-to-date">{{ t('update.preview.upToDate') }}</p>
      </section>
    </template>
  </t-popup>
</template>
<script setup lang="ts">
// 品牌区复用壳层 discovery snapshot，并把版本 Badge 作为锚定的轻量更新入口。
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { usePermissionStore } from '@/store';

import { UPDATE_ROUTE_PATH } from '../contract/paths';
import { UPDATE_PERMISSION_CODE } from '../contract/permissions';
import { useUpdateDiscoveryStore } from '../store/discovery';

const { t } = useI18n();
const router = useRouter();
const permissionStore = usePermissionStore();
const discoveryStore = useUpdateDiscoveryStore();
const visible = ref(false);
const canRead = computed(() => permissionStore.hasPermission(UPDATE_PERMISSION_CODE.READ));
const versionLabel = computed(() => discoveryStore.status?.current_version ?? '');
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
    ? t('update.versionEntry.updateAvailable', { version: discoveryStore.status?.latest?.version })
    : t('update.versionEntry.current', { version: versionLabel.value }),
);
const summary = computed(
  () =>
    discoveryStore.status?.latest?.upgrade_notes ||
    discoveryStore.status?.latest?.notes ||
    t('update.preview.summaryEmpty'),
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
  font-size: 11px;
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
  min-width: 280px;
  padding: var(--td-comp-paddingTB-l) var(--td-comp-paddingLR-l);
}

.update-version-preview__eyebrow {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0;
}

.update-version-preview strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-large);
  font-variant-numeric: tabular-nums;
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
</style>
