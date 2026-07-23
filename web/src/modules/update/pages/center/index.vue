<template>
  <div class="update-center" data-page-type="workflow">
    <section class="update-center__header">
      <div>
        <p class="update-center__eyebrow">{{ t('update.center.navHint') }}</p>
        <h1>{{ t('update.center.title') }}</h1>
        <p>{{ t('update.center.description') }}</p>
      </div>
      <t-button v-if="canCheck" theme="primary" :loading="checking" @click="refreshStatus">
        {{ t('update.center.checkNow') }}
      </t-button>
    </section>

    <t-alert v-if="loadError" theme="error" :message="loadError" close @close="loadError = ''" />
    <t-alert
      v-else-if="status?.check_error"
      theme="warning"
      :message="t('update.center.checkFailed', { reason: status.check_error })"
    />

    <div v-if="loading && !status" class="update-center__loading">
      <t-loading />
    </div>

    <template v-else-if="status">
      <div class="update-center__summary-grid">
        <t-card :title="t('update.center.current.title')" bordered>
          <div class="update-center__version">
            <strong>{{ status.current_version }}</strong>
            <t-tag size="small" variant="light">{{ channelLabel(status.channel) }}</t-tag>
          </div>
          <p>{{ t('update.center.current.description') }}</p>
        </t-card>
        <t-card :title="t('update.center.latest.title')" bordered>
          <template v-if="status.latest">
            <div class="update-center__version">
              <strong>{{ status.latest.version }}</strong>
              <t-tag size="small" theme="success" variant="light">{{ channelLabel(status.latest.channel) }}</t-tag>
            </div>
            <p>{{ t('update.center.latest.available', { date: formatDate(status.latest.published_at) }) }}</p>
          </template>
          <template v-else>
            <strong class="update-center__up-to-date">{{ t('update.center.latest.upToDate') }}</strong>
            <p>{{ t('update.center.latest.upToDateDescription') }}</p>
          </template>
        </t-card>
        <t-card :title="t('update.center.installation.title')" bordered>
          <div class="update-center__profile">
            <span>{{ t('update.center.installation.declared') }}</span>
            <strong>{{ deploymentModeLabel(status.installation_profile.declared_mode) }}</strong>
            <span>{{ t('update.center.installation.detected') }}</span>
            <strong>{{ deploymentModeLabel(status.installation_profile.detected_mode) }}</strong>
          </div>
          <p>{{ status.installation_profile.guidance }}</p>
        </t-card>
      </div>

      <div class="update-center__content-grid">
        <t-card :title="t('update.center.release.title')" bordered>
          <template v-if="status.latest">
            <div class="update-center__release-heading">
              <div>
                <strong>{{ status.latest.version }}</strong>
                <p>{{ t('update.center.release.verified') }}</p>
              </div>
              <t-button
                theme="primary"
                :disabled="true"
                :title="upgradeUnavailableReason"
                data-testid="update-center-upgrade"
              >
                {{ t('update.center.release.upgrade') }}
              </t-button>
            </div>
            <t-alert theme="info" :message="upgradeUnavailableReason" />
            <pre class="update-center__notes graft-scrollbar">{{
              status.latest.notes || t('update.center.release.notesEmpty')
            }}</pre>
            <div class="update-center__release-links">
              <t-link theme="primary" :href="status.latest.manifest_url" target="_blank">
                {{ t('update.center.release.manifest') }}
              </t-link>
              <t-link
                v-if="status.latest.checksums_url"
                theme="primary"
                :href="status.latest.checksums_url"
                target="_blank"
              >
                {{ t('update.center.release.checksums') }}
              </t-link>
            </div>
          </template>
          <management-empty-state
            v-else
            :title="t('update.center.release.emptyTitle')"
            :description="t('update.center.release.emptyDescription')"
          />
        </t-card>

        <t-card :title="t('update.center.capabilities.title')" bordered>
          <p class="update-center__card-description">{{ t('update.center.capabilities.description') }}</p>
          <t-table :data="capabilityRows" row-key="key" :columns="capabilityColumns" size="small" />
          <t-alert
            v-if="status.installation_profile.detected_mode === 'binary'"
            class="update-center__binary-guidance"
            theme="info"
            :message="t('update.center.binaryGuidance')"
          />
        </t-card>
      </div>

      <p v-if="status.checked_at" class="update-center__checked-at">
        {{ t('update.center.checkedAt', { date: formatDate(status.checked_at) }) }}
      </p>
    </template>
  </div>
</template>
<script setup lang="ts">
// Update Center 只消费发现接口；执行入口必须等待后端 Compose 执行器提供受审计的任务 API。
import type { PrimaryTableCol } from 'tdesign-vue-next';
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { ManagementEmptyState } from '@/shared/components/management';
import { formatLocaleDateTime } from '@/shared/observability';
import { usePermissionStore } from '@/store';

import { checkForUpdates, getUpdateStatus } from '../../api/update';
import { UPDATE_PERMISSION_CODE } from '../../contract/permissions';
import type { UpdateChannel, UpdateStatus } from '../../types/update';

const { locale, t } = useI18n();
const permissionStore = usePermissionStore();
const status = ref<UpdateStatus | null>(null);
const loading = ref(false);
const checking = ref(false);
const loadError = ref('');
const canCheck = computed(() => permissionStore.hasPermission(UPDATE_PERMISSION_CODE.CHECK));
const canManage = computed(() => permissionStore.hasPermission(UPDATE_PERMISSION_CODE.MANAGE));

const capabilityColumns = computed<PrimaryTableCol[]>(() => [
  { colKey: 'capability', title: t('update.center.capabilities.columns.capability'), width: 136 },
  { colKey: 'compose', title: t('update.center.capabilities.columns.compose'), width: 104 },
  { colKey: 'binary', title: t('update.center.capabilities.columns.binary') },
]);

const capabilityRows = computed(() => [
  capabilityRow('check', 'supported', 'supported'),
  capabilityRow('notes', 'supported', 'supported'),
  capabilityRow('verify', 'supported', 'supported'),
  capabilityRow('upgrade', 'pending', 'manual'),
  capabilityRow('backup', 'pending', 'manual'),
  capabilityRow('migration', 'pending', 'manual'),
]);

const upgradeUnavailableReason = computed(() => {
  if (!status.value) {
    return t('update.center.release.executionUnavailable');
  }
  if (status.value.installation_profile.capability !== 'compose_upgrade_available') {
    return t('update.center.release.manualOnly');
  }
  if (!canManage.value) {
    return t('update.center.release.managePermissionRequired');
  }
  return t('update.center.release.executionUnavailable');
});

onMounted(() => {
  void loadStatus();
});

async function loadStatus() {
  loading.value = true;
  loadError.value = '';
  try {
    status.value = await getUpdateStatus();
  } catch {
    loadError.value = t('update.center.loadFailed');
  } finally {
    loading.value = false;
  }
}

async function refreshStatus() {
  checking.value = true;
  loadError.value = '';
  try {
    status.value = await checkForUpdates();
  } catch {
    loadError.value = t('update.center.checkRequestFailed');
  } finally {
    checking.value = false;
  }
}

function capabilityRow(key: string, compose: string, binary: string) {
  return {
    key,
    capability: t(`update.center.capabilities.rows.${key}`),
    compose: t(`update.center.capabilities.states.${compose}`),
    binary: t(`update.center.capabilities.states.${binary}`),
  };
}

function channelLabel(channel: UpdateChannel) {
  return t(`update.center.channels.${channel}`);
}

function deploymentModeLabel(mode: string) {
  return t(`update.center.installation.modes.${mode}`);
}

function formatDate(value: string) {
  return formatLocaleDateTime(value, locale.value);
}
</script>
<style scoped lang="less">
.update-center {
  display: grid;
  gap: var(--td-comp-margin-xxl);
}

.update-center__header {
  align-items: flex-start;
  display: flex;
  gap: var(--td-comp-margin-xl);
  justify-content: space-between;
}

.update-center__header h1,
.update-center__header p {
  margin: 0;
}

.update-center__header h1 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-large);
}

.update-center__header > div > p:last-child,
.update-center__card-description,
.update-center__summary-grid p,
.update-center__release-heading p {
  color: var(--td-text-color-secondary);
  margin-top: var(--td-comp-margin-s);
}

.update-center__eyebrow {
  color: var(--td-brand-color);
  font: var(--td-font-body-small);
  margin-bottom: var(--td-comp-margin-xs) !important;
}

.update-center__loading {
  display: grid;
  min-height: 240px;
  place-items: center;
}

.update-center__summary-grid,
.update-center__content-grid {
  display: grid;
  gap: var(--td-comp-margin-l);
}

.update-center__summary-grid {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.update-center__content-grid {
  grid-template-columns: minmax(0, 1.1fr) minmax(360px, 0.9fr);
}

.update-center__version,
.update-center__release-heading,
.update-center__release-links {
  align-items: center;
  display: flex;
  gap: var(--td-comp-margin-s);
}

.update-center__version strong,
.update-center__up-to-date,
.update-center__release-heading strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-large);
  font-variant-numeric: tabular-nums;
}

.update-center__up-to-date {
  color: var(--td-success-color);
}

.update-center__profile {
  display: grid;
  gap: var(--td-comp-margin-xs) var(--td-comp-margin-l);
  grid-template-columns: max-content 1fr;
}

.update-center__profile span {
  color: var(--td-text-color-secondary);
}

.update-center__release-heading {
  justify-content: space-between;
  margin-bottom: var(--td-comp-margin-l);
}

.update-center__release-heading p {
  margin-bottom: 0;
}

.update-center__notes {
  background: var(--td-bg-color-secondarycontainer);
  border: 1px solid var(--td-component-border);
  border-radius: var(--td-radius-medium);
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  margin: var(--td-comp-margin-l) 0;
  max-height: 360px;
  overflow: auto;
  padding: var(--td-comp-paddingLR-l);
  white-space: pre-wrap;
}

.update-center__release-links {
  justify-content: flex-end;
}

.update-center__binary-guidance {
  margin-top: var(--td-comp-margin-l);
}

.update-center__checked-at {
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
  margin: 0;
  text-align: right;
}

@media (width <= 900px) {
  .update-center__summary-grid,
  .update-center__content-grid {
    grid-template-columns: 1fr;
  }
}

@media (width <= 640px) {
  .update-center__header,
  .update-center__release-heading {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
