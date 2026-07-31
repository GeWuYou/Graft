<template>
  <t-popup
    v-if="canRead && versionLabel"
    v-model:visible="visible"
    destroy-on-close
    placement="bottom-right"
    trigger="click"
    :overlay-inner-style="{ padding: '0' }"
  >
    <t-tooltip
      v-if="discoveryStore.hasUpdate"
      data-testid="update-version-tooltip"
      :content="updateTooltip"
      placement="bottom"
    >
      <button
        :class="['update-version-entry', { 'update-version-entry--available': discoveryStore.hasUpdate }]"
        data-testid="update-version-entry"
        type="button"
        :aria-label="versionEntryAriaLabel"
      >
        {{ versionLabel }}
      </button>
    </t-tooltip>
    <button
      v-else
      :class="['update-version-entry', { 'update-version-entry--available': discoveryStore.hasUpdate }]"
      data-testid="update-version-entry"
      type="button"
      :aria-label="versionEntryAriaLabel"
    >
      {{ versionLabel }}
    </button>
    <template #content>
      <section class="update-version-preview">
        <header class="update-version-preview__header">
          <span>{{ t('update.preview.current') }}</span>
          <t-tooltip :content="t('update.preview.checkNow')" placement="top">
            <t-button
              data-testid="update-preview-refresh"
              shape="square"
              size="small"
              theme="default"
              variant="text"
              :disabled="!canCheck || checking"
              :loading="checking"
              @click.stop="refreshStatus"
            >
              <template #icon>
                <refresh-icon />
              </template>
            </t-button>
          </t-tooltip>
        </header>
        <div class="update-version-preview__current">
          <strong>{{ discoveryStore.status?.current_version }}</strong>
          <p v-if="hasAvailableRelease" class="update-version-preview__available-status">
            {{ t('update.preview.availableVersion', { version: availableRelease?.version }) }}
          </p>
          <p v-else-if="checking" class="update-version-preview__checking">
            {{ t('update.preview.checking') }}
          </p>
          <p v-else-if="statusUnavailable" class="update-version-preview__unavailable">
            {{ t('update.preview.unavailable') }}
          </p>
          <p v-else-if="previewOpened" class="update-version-preview__up-to-date">
            {{ t('update.preview.upToDate') }}
          </p>
        </div>
        <template v-if="hasAvailableRelease">
          <t-button
            v-if="canStartUpgrade"
            class="update-version-preview__upgrade"
            data-testid="update-preview-upgrade"
            theme="primary"
            @click.stop="startUpgrade"
          >
            {{ t('update.preview.startUpgrade') }}
          </t-button>
        </template>
        <footer class="update-version-preview__actions">
          <t-button
            v-if="hasAvailableRelease"
            data-testid="update-preview-detail"
            size="small"
            variant="text"
            @click.stop="openManagement"
          >
            {{ t('update.preview.detail.open') }}
          </t-button>
          <t-button
            data-testid="update-preview-release"
            size="small"
            variant="text"
            :tag="canViewRelease ? 'a' : 'button'"
            :href="canViewRelease ? releaseUrl : undefined"
            target="_blank"
            rel="noopener noreferrer"
            :disabled="!canViewRelease"
          >
            {{ t('update.preview.viewRelease') }}
          </t-button>
        </footer>
      </section>
    </template>
  </t-popup>
</template>
<script setup lang="ts">
// 品牌区复用壳层 discovery snapshot，并把版本 Badge 作为锚定的轻量更新入口。
import { RefreshIcon } from 'tdesign-icons-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import { usePermissionStore } from '@/store';

import { getAvailableUpdateRelease, hasAvailableUpdate } from '../composables/releaseSelection';
import { useUpdatePreviewActions } from '../composables/useUpdatePreviewActions';
import { UPDATE_ROUTE_PATH } from '../contract/paths';
import { UPDATE_PERMISSION_CODE } from '../contract/permissions';
import { useUpdateDiscoveryStore } from '../store/discovery';

const props = withDefaults(
  defineProps<{
    centerPath?: string;
  }>(),
  { centerPath: UPDATE_ROUTE_PATH.CENTER },
);

const { t } = useI18n();
const permissionStore = usePermissionStore();
const discoveryStore = useUpdateDiscoveryStore();
const visible = ref(false);
const checking = ref(false);
const previewOpened = ref(false);
const previewCheckFailed = ref(false);
const canRead = computed(() => permissionStore.hasPermission(UPDATE_PERMISSION_CODE.READ));
const canCheck = computed(() => permissionStore.hasPermission(UPDATE_PERMISSION_CODE.CHECK));
const versionLabel = computed(() => discoveryStore.status?.current_version ?? '');
const { canStartUpgrade, openManagement, startUpgrade } = useUpdatePreviewActions(visible, props.centerPath);
const availableRelease = computed(() => getAvailableUpdateRelease(discoveryStore.status));
const hasAvailableRelease = computed(() => hasAvailableUpdate(discoveryStore.status));
const statusUnavailable = computed(
  () => previewCheckFailed.value || Boolean(discoveryStore.status?.cache_stale || discoveryStore.status?.check_error),
);
const releaseUrl = computed(() => availableRelease.value?.notes_url?.trim() ?? '');
const canViewRelease = computed(() => hasAvailableRelease.value && /^https:\/\//i.test(releaseUrl.value));
const updateTooltip = computed(() =>
  t('update.versionEntry.updateAvailable', { version: availableRelease.value?.version }),
);
const versionEntryAriaLabel = computed(() =>
  discoveryStore.hasUpdate ? updateTooltip.value : t('update.versionEntry.current', { version: versionLabel.value }),
);
watch(
  visible,
  (isVisible) => {
    // 使用 sync 确保弹窗打开时立即复用预览缓存或发起检查；手动刷新仍由 refreshStatus 负责强制更新。
    if (!isVisible || !canCheck.value) {
      return;
    }
    previewOpened.value = true;
    previewCheckFailed.value = false;
    checking.value = true;
    void discoveryStore
      .refreshPreviewSnapshot()
      .catch(() => {
        previewCheckFailed.value = true;
      })
      .finally(() => {
        checking.value = false;
      });
  },
  { flush: 'sync' },
);

async function refreshStatus() {
  if (!canCheck.value) {
    return;
  }
  previewOpened.value = true;
  previewCheckFailed.value = false;
  checking.value = true;
  try {
    await discoveryStore.refreshSnapshot();
  } catch {
    previewCheckFailed.value = true;
    MessagePlugin.error(t('update.preview.checkFailed'));
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

.update-version-entry--available {
  background: var(--td-warning-color-1);
  border-color: var(--td-warning-color-5);
  color: var(--td-warning-color-8);
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
  font: var(--td-font-title-large);
  font-variant-numeric: tabular-nums;
  max-width: 100%;
  overflow-wrap: anywhere;
  text-align: center;
}

.update-version-preview__actions {
  border-top: 1px solid var(--td-component-stroke);
  display: flex;
  gap: var(--td-comp-margin-s);
  justify-content: flex-end;
  padding-top: var(--td-comp-paddingTB-s);
}

.update-version-preview__upgrade {
  justify-self: center;
  min-height: 36px;
  width: min(184px, 100%);
}

.update-version-preview__up-to-date {
  color: var(--td-success-color);
  font: var(--td-font-body-small);
  margin: 0;
}

.update-version-preview__checking {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0;
}

.update-version-preview__available-status {
  color: var(--td-brand-color);
  font: var(--td-font-body-small);
  margin: 0;
}

.update-version-preview__unavailable {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0;
}
</style>
