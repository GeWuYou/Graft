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
                :disabled="!canOpenUpgradeFlow"
                :title="canOpenUpgradeFlow ? '' : upgradeUnavailableReason"
                data-testid="update-center-upgrade"
                @click="openConfirmation"
              >
                {{ t('update.center.release.upgrade') }}
              </t-button>
            </div>
            <t-alert v-if="!canOpenUpgradeFlow" theme="info" :message="upgradeUnavailableReason" />
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
        <t-table v-else :data="operations" row-key="operation_id" :columns="operationColumns" size="small">
          <template #status="{ row }">
            <t-tag size="small" :theme="operationStatusTheme(row.status)" variant="light-outline">
              {{ t(`update.center.history.statuses.${row.status}`) }}
            </t-tag>
          </template>
          <template #failure_code="{ row }">
            <t-button
              v-if="hasFailureDiagnostic(row) && !dataSource"
              size="small"
              variant="text"
              @click="showHistoryCause(row)"
            >
              {{ t('update.center.history.viewCause') }}
            </t-button>
            <span v-else>{{ row.failure_code || '-' }}</span>
          </template>
        </t-table>
      </t-card>
    </template>

    <t-dialog
      v-model:visible="confirmationVisible"
      :header="t('update.center.confirmation.title', { version: status?.latest?.version })"
      :confirm-btn="{
        content: t('update.center.confirmation.confirm'),
        theme: 'danger',
        loading: submitting,
        disabled: !canSubmitUpgrade,
      }"
      :cancel-btn="{ content: t('update.center.confirmation.cancel') }"
      @confirm="submitUpgrade"
    >
      <p>{{ t('update.center.confirmation.description', { version: status?.latest?.version }) }}</p>
      <template v-if="isDockerDiscovery">
        <p class="update-center__compose-root-title">{{ t('update.center.composeRoot.title') }}</p>
        <p class="update-center__card-description">{{ t('update.center.composeRoot.description') }}</p>
        <section v-if="resolvedCandidate" class="update-center__resolved-candidate" data-testid="resolved-compose-root">
          <strong>{{ resolvedCandidate.host_path }}</strong>
          <small>{{ resolvedCandidate.compose_files.join(', ') }}</small>
        </section>
        <t-radio-group v-else v-model="selectedCandidateKey" direction="vertical">
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
        <t-alert
          v-if="!hasSelectedCandidate"
          class="update-center__candidate-selection"
          theme="warning"
          :message="t('update.center.composeRoot.selectionRequired')"
        />
      </template>
      <t-alert
        v-if="operationError"
        class="update-center__confirmation-error"
        theme="error"
        :message="operationError"
      />
      <p
        v-if="operationRequestId"
        class="update-center__confirmation-request-id"
        data-testid="update-operation-request-id"
      >
        {{ t('update.center.confirmation.requestId', { requestId: operationRequestId }) }}
      </p>
      <t-alert
        v-if="diagnosticUnavailable"
        class="update-center__diagnostic-unavailable"
        theme="warning"
        :message="t('update.center.confirmation.diagnosticUnavailable')"
      />
      <section v-if="operationDiagnostic" class="update-center__diagnostic" data-testid="update-operation-diagnostic">
        <div class="update-center__diagnostic-heading">
          <strong>{{ t('update.center.confirmation.diagnosticTitle') }}</strong>
          <t-button size="small" variant="text" @click="openAppLogs">
            {{ t('update.center.confirmation.viewAppLogs') }}
          </t-button>
        </div>
        <dl>
          <div>
            <dt>{{ t('update.center.confirmation.diagnosticCode') }}</dt>
            <dd>{{ operationDiagnostic.failure_code }}</dd>
          </div>
          <div>
            <dt>{{ t('update.center.confirmation.diagnosticStage') }}</dt>
            <dd>{{ operationDiagnostic.failure_stage }}</dd>
          </div>
        </dl>
        <pre>{{ operationDiagnostic.detail }}</pre>
      </section>
    </t-dialog>
  </div>
</template>
<script setup lang="ts">
// 更新管理页复用壳层 discovery snapshot，仅为历史和受控升级提交保留自身的 Update API 调用。
import type { PrimaryTableCol } from 'tdesign-vue-next';
import { computed, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { buildAppLogLocation } from '@/modules/app-log/contract/deep-link';
import { ManagementEmptyState } from '@/shared/components/management';
import { MarkdownViewer } from '@/shared/components/markdown';
import { formatLocaleDateTime } from '@/shared/observability';
import { usePermissionStore } from '@/store';
import { isApiRequestError } from '@/utils/request';

import { createUpdateOperation, getUpdateFailureDiagnostic, getUpdateOperations } from '../../api/update';
import { isUpgradeEligible } from '../../composables/updateEligibility';
import { isUpdateOperationFailureCode, UPDATE_OPERATION_FAILURE_MESSAGE_KEY } from '../../contract/failure-codes';
import { UPDATE_PERMISSION_CODE } from '../../contract/permissions';
import { useUpdateDiscoveryStore } from '../../store/discovery';
import { useUpdateProgressStore } from '../../store/progress';
import type { UpdateCenterDataSource } from '../../types/preview';
import type { UpdateChannel, UpdateFailureDiagnostic, UpdateOperation, UpdateStatus } from '../../types/update';

const props = defineProps<{
  dataSource?: UpdateCenterDataSource;
}>();

const { locale, t } = useI18n();
const route = useRoute();
const router = useRouter();
const permissionStore = usePermissionStore();
const discoveryStore = useUpdateDiscoveryStore();
const progressStore = useUpdateProgressStore();
const previewStatus = ref<UpdateStatus | null>(null);
const status = computed<UpdateStatus | null>(() => (props.dataSource ? previewStatus.value : discoveryStore.status));
const loading = computed(() => (props.dataSource ? !previewStatus.value : discoveryStore.phase === 'loading'));
const checking = ref(false);
const loadError = ref('');
const historyError = ref('');
const operations = ref<UpdateOperation[]>([]);
const confirmationVisible = ref(false);
const submitting = ref(false);
const operationError = ref('');
const operationRequestId = ref('');
const operationDiagnostic = ref<UpdateFailureDiagnostic | null>(null);
const diagnosticUnavailable = ref(false);
const selectedCandidateKey = ref('');
const canCheck = computed(() =>
  props.dataSource ? props.dataSource.permissions.check : permissionStore.hasPermission(UPDATE_PERMISSION_CODE.CHECK),
);
const canManage = computed(() =>
  props.dataSource ? props.dataSource.permissions.manage : permissionStore.hasPermission(UPDATE_PERMISSION_CODE.MANAGE),
);
const isDockerDiscovery = computed(
  () => status.value?.installation_profile.compose_root_source === 'docker_discovered',
);
const composeCandidates = computed(() => status.value?.installation_profile.compose_candidates ?? []);
const composeRootConfirmationRequired = computed(
  () => status.value?.installation_profile.compose_root_confirmation_required === true,
);
const resolvedCandidate = computed(() => {
  if (composeRootConfirmationRequired.value) return null;
  const candidates = composeCandidates.value;
  const highConfidence = candidates.filter(({ confidence }) => confidence === 'high');
  return highConfidence[0] ?? candidates[0] ?? null;
});
const hasSelectedCandidate = computed(
  () =>
    !isDockerDiscovery.value ||
    Boolean(resolvedCandidate.value) ||
    composeCandidates.value.some(({ key }) => key === selectedCandidateKey.value),
);
const canOpenUpgradeFlow = computed(
  () =>
    isUpgradeEligible(status.value, canManage.value) &&
    (!isDockerDiscovery.value || composeCandidates.value.length > 0),
);
const canSubmitUpgrade = computed(() => canOpenUpgradeFlow.value && hasSelectedCandidate.value);
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
  return '';
});

onMounted(async () => {
  await loadStatus();
});

watch(
  [() => route.query.upgrade, canOpenUpgradeFlow],
  ([upgradeRequested, eligible]) => {
    if (upgradeRequested === '1' && eligible && !confirmationVisible.value) {
      openConfirmation();
    }
  },
  { immediate: true },
);

async function loadStatus() {
  loadError.value = '';
  if (props.dataSource) {
    try {
      previewStatus.value = await props.dataSource.getStatus();
      syncCandidateSelection();
    } catch {
      loadError.value = t('update.center.loadFailed');
    }
    await loadHistory();
    return;
  }
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
    if (props.dataSource) {
      previewStatus.value = await props.dataSource.checkForUpdates();
    } else {
      await discoveryStore.refreshSnapshot();
    }
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
    operations.value = props.dataSource ? await props.dataSource.getOperations() : await getUpdateOperations();
  } catch {
    historyError.value = t('update.center.history.loadFailed');
  }
}

function openConfirmation() {
  if (!canOpenUpgradeFlow.value) {
    return;
  }
  operationError.value = '';
  operationRequestId.value = '';
  operationDiagnostic.value = null;
  diagnosticUnavailable.value = false;
  confirmationVisible.value = true;
}

async function submitUpgrade() {
  if (!status.value?.latest || !canSubmitUpgrade.value) {
    return;
  }
  submitting.value = true;
  operationError.value = '';
  operationRequestId.value = '';
  operationDiagnostic.value = null;
  diagnosticUnavailable.value = false;
  try {
    const payload = {
      target_version: status.value.latest.version,
      // 唯一高置信候选已由服务端解析，避免把选择键回传成第二份客户端事实。
      ...(isDockerDiscovery.value && composeRootConfirmationRequired.value
        ? { compose_candidate_key: selectedCandidateKey.value }
        : {}),
    };
    let operation: UpdateOperation;
    if (props.dataSource) {
      operation = await props.dataSource.createOperation(payload);
    } else {
      operation = await createUpdateOperation(payload);
    }
    confirmationVisible.value = false;
    if (!props.dataSource) {
      progressStore.begin(operation);
    }
    await loadHistory();
  } catch (error) {
    operationError.value = resolveOperationErrorMessage(error);
    operationRequestId.value =
      isApiRequestError(error) && isUpdateOperationFailureCode(error.code) ? error.traceId.trim() : '';
    if (operationRequestId.value) {
      try {
        operationDiagnostic.value = props.dataSource
          ? await props.dataSource.getFailureDiagnostic(operationRequestId.value)
          : await getUpdateFailureDiagnostic(operationRequestId.value);
      } catch {
        diagnosticUnavailable.value = true;
      }
    }
  } finally {
    submitting.value = false;
  }
}

function openAppLogs() {
  if (!operationRequestId.value) {
    return;
  }
  void router.push(buildAppLogLocation({ request_id: operationRequestId.value }));
}

function hasFailureDiagnostic(operation: UpdateOperation) {
  return Boolean(
    operation.failure_diagnostic_available && (operation.status === 'FAILED' || operation.status === 'NEEDS_ATTENTION'),
  );
}

function showHistoryCause(operation: UpdateOperation) {
  if (props.dataSource) {
    return;
  }
  progressStore.begin(operation);
}

function operationStatusTheme(status: UpdateOperation['status']) {
  if (status === 'FAILED' || status === 'NEEDS_ATTENTION') return 'danger';
  if (status === 'SUCCESS' || status === 'RECOVERED') return 'success';
  if (status === 'RECREATING' || status === 'VERIFYING') return 'warning';
  return 'primary';
}

function syncCandidateSelection() {
  const candidates = status.value?.installation_profile.compose_candidates ?? [];
  if (status.value?.installation_profile.compose_root_source !== 'docker_discovered') {
    selectedCandidateKey.value = '';
    return;
  }
  if (resolvedCandidate.value) {
    selectedCandidateKey.value = '';
    return;
  }
  if (!candidates.some(({ key }) => key === selectedCandidateKey.value)) {
    const highConfidenceCandidates = candidates.filter(({ confidence }) => confidence === 'high');
    selectedCandidateKey.value = highConfidenceCandidates.length === 1 ? highConfidenceCandidates[0].key : '';
  }
}

function resolveOperationErrorMessage(error: unknown) {
  if (!isApiRequestError(error)) {
    return t('update.center.confirmation.failure.generic');
  }

  return isUpdateOperationFailureCode(error.code)
    ? t(UPDATE_OPERATION_FAILURE_MESSAGE_KEY[error.code])
    : t('update.center.confirmation.failure.generic');
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

.update-center__resolved-candidate {
  background: var(--td-bg-color-secondarycontainer);
  border: 1px solid var(--td-component-border);
  display: grid;
  gap: var(--td-comp-margin-xs);
  margin-top: var(--td-comp-margin-m);
  padding: var(--td-comp-paddingTB-s) var(--td-comp-paddingLR-s);
}

.update-center__resolved-candidate small {
  color: var(--td-text-color-secondary);
  overflow-wrap: anywhere;
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

.update-center__diagnostic-unavailable {
  margin-top: var(--td-comp-margin-l);
}

.update-center__diagnostic {
  border-top: 1px solid var(--td-component-border);
  display: grid;
  gap: var(--td-comp-margin-s);
  margin-top: var(--td-comp-margin-l);
  padding-top: var(--td-comp-paddingTB-l);

  dl {
    display: grid;
    gap: var(--td-comp-margin-l);
    grid-template-columns: repeat(2, minmax(0, 1fr));
    margin: 0;
  }

  dt {
    color: var(--td-text-color-secondary);
    font-size: var(--td-font-size-s);
  }

  dd {
    margin: var(--td-comp-margin-xs) 0 0;
    overflow-wrap: anywhere;
  }

  pre {
    background: var(--td-bg-color-container-hover);
    color: var(--td-text-color-primary);
    font-family: var(--td-font-family-mono);
    font-size: var(--td-font-size-s);
    line-height: 1.5;
    margin: 0;
    max-height: 240px;
    overflow: auto;
    overflow-wrap: anywhere;
    padding: var(--td-comp-paddingTB-s) var(--td-comp-paddingLR-s);
    white-space: pre-wrap;
  }
}

.update-center__diagnostic-heading {
  align-items: center;
  display: flex;
  gap: var(--td-comp-margin-s);
  justify-content: space-between;
}

.update-center__candidate-selection {
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
