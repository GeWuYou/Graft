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
          <template v-if="status.latest && !status.cache_stale && !status.check_error">
            <div class="update-center__version">
              <strong>{{ status.latest.version }}</strong>
              <t-tag size="small" theme="success" variant="light">{{ channelLabel(status.latest.channel) }}</t-tag>
            </div>
            <p>{{ t('update.center.latest.available', { date: formatDate(status.latest.published_at) }) }}</p>
          </template>
          <template v-else-if="status.cache_stale || status.check_error">
            <strong class="update-center__up-to-date">{{ t('update.center.latest.unavailable') }}</strong>
            <p>{{ t('update.center.latest.unavailableDescription') }}</p>
          </template>
          <template v-else>
            <strong class="update-center__up-to-date">{{ t('update.center.latest.upToDate') }}</strong>
            <p>{{ t('update.center.latest.upToDateDescription') }}</p>
          </template>
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
                :disabled="!canStartUpgrade"
                :title="canStartUpgrade ? '' : upgradeUnavailableReason"
                data-testid="update-center-upgrade"
                @click="openConfirmation"
              >
                {{ t('update.center.release.upgrade') }}
              </t-button>
            </div>
            <div v-if="isDockerDiscovery" class="update-center__compose-root">
              <t-alert
                v-if="!composeCandidates.length"
                theme="warning"
                :message="t('update.center.composeRoot.noCandidates')"
              />
              <template v-else>
                <p class="update-center__compose-root-title">
                  {{ t('update.center.composeRoot.title') }}
                </p>
                <p class="update-center__card-description">
                  {{ t('update.center.composeRoot.description') }}
                </p>
                <t-radio-group v-model="selectedCandidateKey" direction="vertical">
                  <t-radio v-for="candidate in composeCandidates" :key="candidate.key" :value="candidate.key">
                    <span class="update-center__candidate">
                      <strong>{{ candidate.host_path }}</strong>
                      <small>
                        {{ candidate.compose_files.join(', ') }}
                        <template v-if="candidate.project_name"> · {{ candidate.project_name }} </template>
                      </small>
                    </span>
                  </t-radio>
                </t-radio-group>
              </template>
            </div>
            <t-alert v-if="!canStartUpgrade" theme="info" :message="upgradeUnavailableReason" />
            <div class="update-center__notes graft-scrollbar">
              <markdown-viewer :source="releaseNotes" />
            </div>
            <t-alert v-if="status.latest.upgrade_notes" theme="info" :message="status.latest.upgrade_notes" />
            <ol v-if="status.installation_profile.manual_steps?.length" class="update-center__manual-steps">
              <li v-for="step in status.installation_profile.manual_steps" :key="step.key">
                {{ t(`update.center.manualSteps.${step.key}`, step.params ?? {}) }}
              </li>
            </ol>
            <div class="update-center__release-links">
              <t-link theme="primary" :href="status.latest.manifest_url" target="_blank">
                {{ t('update.center.release.manifest') }}
              </t-link>
              <t-link v-if="status.latest.notes_url" theme="primary" :href="status.latest.notes_url" target="_blank">
                {{ t('update.center.release.releaseNotes') }}
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

        <t-card :title="t('update.center.advanced.title')" bordered>
          <t-collapse borderless>
            <t-collapse-panel :header="t('update.center.advanced.installation')" value="installation">
              <div class="update-center__profile">
                <span>{{ t('update.center.installation.declared') }}</span
                ><strong>{{ deploymentModeLabel(status.installation_profile.declared_mode) }}</strong>
                <span>{{ t('update.center.installation.detected') }}</span
                ><strong>{{ deploymentModeLabel(status.installation_profile.detected_mode) }}</strong>
              </div>
              <p class="update-center__card-description">{{ status.installation_profile.guidance }}</p>
            </t-collapse-panel>
            <t-collapse-panel :header="t('update.center.capabilities.title')" value="capabilities">
              <p class="update-center__card-description">{{ t('update.center.capabilities.description') }}</p>
              <t-table :data="capabilityRows" row-key="key" :columns="capabilityColumns" size="small" />
              <t-alert
                v-if="status.installation_profile.detected_mode === 'binary'"
                class="update-center__binary-guidance"
                theme="info"
                :message="t('update.center.binaryGuidance')"
              />
            </t-collapse-panel>
          </t-collapse>
        </t-card>
      </div>

      <p v-if="status.checked_at" class="update-center__checked-at">
        {{ t('update.center.checkedAt', { date: formatDate(status.checked_at) }) }}
      </p>

      <t-card :title="t('update.center.history.title')" bordered>
        <t-alert v-if="historyError" theme="warning" :message="historyError" />
        <t-table v-else :data="operations" row-key="operation_id" :columns="operationColumns" size="small" />
      </t-card>
    </template>

    <t-dialog
      v-model:visible="confirmationVisible"
      :header="t('update.center.confirmation.title', { version: status?.latest?.version })"
      :confirm-btn="{
        content: t('update.center.confirmation.confirm'),
        theme: 'danger',
        loading: submitting,
        disabled: !isExactConfirmation,
      }"
      :cancel-btn="{ content: t('update.center.confirmation.cancel') }"
      @confirm="submitUpgrade"
    >
      <p>{{ t('update.center.confirmation.description', { version: status?.latest?.version }) }}</p>
      <t-input v-model="confirmation" :placeholder="status?.latest?.version" autofocus />
      <t-alert
        v-if="operationError"
        class="update-center__confirmation-error"
        theme="error"
        :message="operationError"
      />
    </t-dialog>
  </div>
</template>
<script setup lang="ts">
// 更新管理页复用壳层 discovery snapshot，仅为历史和精确版本确认保留自身的 Update API 调用。
import type { PrimaryTableCol } from 'tdesign-vue-next';
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';

import { ManagementEmptyState } from '@/shared/components/management';
import { MarkdownViewer } from '@/shared/components/markdown';
import { formatLocaleDateTime } from '@/shared/observability';
import { usePermissionStore } from '@/store';

import { createUpdateOperation, getUpdateOperations } from '../../api/update';
import { isUpgradeEligible } from '../../composables/updateEligibility';
import { UPDATE_PERMISSION_CODE } from '../../contract/permissions';
import { useUpdateDiscoveryStore } from '../../store/discovery';
import type { UpdateChannel, UpdateOperation, UpdateStatus } from '../../types/update';

const { locale, t } = useI18n();
const route = useRoute();
const permissionStore = usePermissionStore();
const discoveryStore = useUpdateDiscoveryStore();
const status = computed<UpdateStatus | null>(() => discoveryStore.status);
const loading = computed(() => discoveryStore.phase === 'loading');
const checking = ref(false);
const loadError = ref('');
const historyError = ref('');
const operations = ref<UpdateOperation[]>([]);
const confirmationVisible = ref(false);
const confirmation = ref('');
const submitting = ref(false);
const operationError = ref('');
const selectedCandidateKey = ref('');
const canCheck = computed(() => permissionStore.hasPermission(UPDATE_PERMISSION_CODE.CHECK));
const canManage = computed(() => permissionStore.hasPermission(UPDATE_PERMISSION_CODE.MANAGE));
const isDockerDiscovery = computed(
  () => status.value?.installation_profile.compose_root_source === 'docker_discovered',
);
const composeCandidates = computed(() => status.value?.installation_profile.compose_candidates ?? []);
const hasSelectedCandidate = computed(
  () => !isDockerDiscovery.value || composeCandidates.value.some(({ key }) => key === selectedCandidateKey.value),
);
const canStartUpgrade = computed(() => isUpgradeEligible(status.value, canManage.value) && hasSelectedCandidate.value);
const isExactConfirmation = computed(
  () => confirmation.value === status.value?.latest?.version && hasSelectedCandidate.value,
);
const releaseNotes = computed(() => status.value?.latest?.notes || t('update.center.release.notesEmpty'));

const capabilityColumns = computed<PrimaryTableCol[]>(() => [
  { colKey: 'capability', title: t('update.center.capabilities.columns.capability'), width: 136 },
  { colKey: 'compose', title: t('update.center.capabilities.columns.compose'), width: 104 },
  { colKey: 'binary', title: t('update.center.capabilities.columns.binary') },
]);

const capabilityRows = computed(() => [
  capabilityRow('check', 'supported', 'supported'),
  capabilityRow('notes', 'supported', 'supported'),
  capabilityRow('verify', 'supported', 'supported'),
  capabilityRow('upgrade', 'supported', 'manual'),
  capabilityRow('backup', 'supported', 'manual'),
  capabilityRow('migration', 'supported', 'manual'),
]);

const operationColumns = computed<PrimaryTableCol[]>(() => [
  { colKey: 'target_version', title: t('update.center.history.target'), width: 140 },
  { colKey: 'status', title: t('update.center.history.status'), width: 160 },
  { colKey: 'failure_code', title: t('update.center.history.result'), ellipsis: true },
  {
    colKey: 'created_at',
    title: t('update.center.history.started'),
    width: 190,
    cell: (_h, { row }) => formatDate((row as UpdateOperation).created_at),
  },
]);

const upgradeUnavailableReason = computed(() => {
  if (!status.value) {
    return t('update.center.release.executionUnavailable');
  }
  if (status.value.cache_stale || status.value.check_error) {
    return t('update.center.release.catalogStale');
  }
  if (status.value.installation_profile.capability !== 'compose_upgrade_available') {
    return t('update.center.release.manualOnly');
  }
  if (!canManage.value) {
    return t('update.center.release.managePermissionRequired');
  }
  if (isDockerDiscovery.value && !composeCandidates.value.length) {
    return t('update.center.composeRoot.noCandidates');
  }
  if (isDockerDiscovery.value && !hasSelectedCandidate.value) {
    return t('update.center.composeRoot.selectionRequired');
  }
  return '';
});

onMounted(async () => {
  await loadStatus();
  if (route.query.upgrade === '1' && canStartUpgrade.value) {
    openConfirmation();
  }
});

async function loadStatus() {
  loadError.value = '';
  await discoveryStore.ensureSnapshot();
  syncCandidateSelection();
  if (discoveryStore.phase === 'error') {
    loadError.value = t('update.center.loadFailed');
  }
  await loadHistory();
}

async function refreshStatus() {
  checking.value = true;
  loadError.value = '';
  try {
    await discoveryStore.refreshSnapshot();
    syncCandidateSelection();
    await loadHistory();
  } catch {
    loadError.value = t('update.center.checkRequestFailed');
  } finally {
    checking.value = false;
  }
}

async function loadHistory() {
  historyError.value = '';
  try {
    operations.value = await getUpdateOperations();
  } catch {
    historyError.value = t('update.center.history.loadFailed');
  }
}

function openConfirmation() {
  if (!canStartUpgrade.value) {
    return;
  }
  confirmation.value = '';
  operationError.value = '';
  confirmationVisible.value = true;
}

async function submitUpgrade() {
  if (!status.value?.latest || !isExactConfirmation.value) {
    return;
  }
  submitting.value = true;
  operationError.value = '';
  try {
    await createUpdateOperation({
      target_version: status.value.latest.version,
      confirmation: confirmation.value,
      ...(isDockerDiscovery.value ? { compose_candidate_key: selectedCandidateKey.value } : {}),
    });
    confirmationVisible.value = false;
    await loadHistory();
  } catch {
    operationError.value = t('update.center.confirmation.submitFailed');
  } finally {
    submitting.value = false;
  }
}

function syncCandidateSelection() {
  const candidates = status.value?.installation_profile.compose_candidates ?? [];
  if (status.value?.installation_profile.compose_root_source !== 'docker_discovered') {
    selectedCandidateKey.value = '';
    return;
  }
  if (!candidates.some(({ key }) => key === selectedCandidateKey.value)) {
    selectedCandidateKey.value = candidates.length === 1 ? candidates[0].key : '';
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
  grid-template-columns: repeat(2, minmax(0, 1fr));
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

.update-center__compose-root {
  border-top: 1px solid var(--td-component-border);
  margin-top: var(--td-comp-margin-l);
  padding-top: var(--td-comp-paddingTB-l);
}

.update-center__compose-root-title {
  color: var(--td-text-color-primary);
  font-weight: 600;
  margin: 0;
}

.update-center__candidate {
  display: inline-flex;
  flex-direction: column;
  gap: var(--td-comp-margin-xs);
  margin-left: var(--td-comp-margin-xs);
  vertical-align: top;
}

.update-center__candidate small {
  color: var(--td-text-color-secondary);
  font-size: var(--td-font-size-body-small);
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
}

.update-center__release-links {
  justify-content: flex-end;
}

.update-center__manual-steps {
  margin: var(--td-comp-margin-m) 0;
  padding-inline-start: var(--td-comp-margin-xxl);
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

.update-center__confirmation-error {
  margin-top: var(--td-comp-margin-l);
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
