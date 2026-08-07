<template>
  <t-dialog
    :visible="progress.visible"
    :header="t('update.center.progress.title')"
    :cancel-btn="null"
    :confirm-btn="null"
    :close-btn="terminal"
    :close-on-esc-keydown="terminal"
    :close-on-overlay-click="terminal"
    :prevent-scroll-through="true"
    :footer="false"
    width="480px"
    @close="closeTerminalDialog"
  >
    <section class="update-progress" data-testid="update-progress-dialog">
      <t-progress
        v-if="showProgress"
        data-testid="update-progress-overall"
        :percentage="overallPercentage"
        :label="true"
      />
      <t-alert
        :theme="
          progress.phase === 'failed' || progress.phase === 'unavailable'
            ? 'error'
            : progress.phase === 'success'
              ? 'success'
              : 'info'
        "
        :message="phaseMessage"
      />
      <t-steps v-if="showRunnerStage" :current="currentStep" :options="stepOptions" layout="vertical" readonly />
      <section v-if="showRunnerStage" class="update-progress__current-stage">
        <div class="update-progress__current-stage-heading">
          <span>{{ t('update.center.progress.currentStage') }}</span>
          <strong>{{ currentStageLabel }}</strong>
        </div>
      </section>
      <section v-if="progress.phase === 'failed'" class="update-progress__failure">
        <template v-if="progress.failureDiagnostic">
          <strong data-testid="update-progress-diagnostic-summary">{{ progress.failureDiagnostic.summary }}</strong>
          <p data-testid="update-progress-diagnostic-detail">{{ progress.failureDiagnostic.detail }}</p>
        </template>
        <p v-else-if="progress.failureDiagnosticLoading">{{ t('update.center.progress.diagnosticLoading') }}</p>
        <p v-else-if="progress.failureDiagnosticError">{{ t('update.center.progress.diagnosticUnavailable') }}</p>
        <p v-else>{{ progress.operation?.message || t('update.center.progress.diagnosticUnavailable') }}</p>
      </section>
      <section v-if="canRecoverTerminatedRunner" class="update-progress__recovery">
        <p>{{ t('update.center.progress.recovery.description') }}</p>
        <t-alert v-if="progress.recoveryError" theme="error" :message="t('update.center.progress.recovery.failed')" />
        <t-button
          data-testid="update-progress-recovery"
          theme="primary"
          :loading="progress.recoveryLoading"
          @click="recoverTerminatedRunner"
        >
          {{ t('update.center.progress.recovery.action') }}
        </t-button>
      </section>
      <section v-if="progress.phase === 'unavailable'" class="update-progress__failure">
        <p>{{ t('update.center.progress.sourceUnavailable') }}</p>
      </section>
      <section v-if="progress.events.length" class="update-progress__events" data-testid="update-progress-events">
        <h3>{{ t('update.center.progress.events.title') }}</h3>
        <ol class="graft-scrollbar">
          <li v-for="event in progress.events" :key="event.revision">
            <strong>{{ t(`update.center.history.phases.${event.phase}`) }}</strong>
            <span>{{ eventMessage(event.message) }}</span>
          </li>
        </ol>
      </section>
      <t-button v-if="terminal" class="update-progress__close" @click="closeTerminalDialog">
        {{ t('update.center.progress.close') }}
      </t-button>
    </section>
  </t-dialog>
</template>
<script setup lang="ts">
import { computed, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';

import { usePermissionStore } from '@/store';

import { UPDATE_PERMISSION_CODE } from '../contract/permissions';
import { useUpdateProgressStore } from '../store/progress';
import type { UpdateOperationPhase } from '../types/update';

// 后台壳只呈现服务端验证的操作投影；服务重建中的连接状态不能覆盖其报告的真实进度。
const { t } = useI18n();
const progress = useUpdateProgressStore();
const permissionStore = usePermissionStore();
const steps = [
  'ready',
  'preflight',
  'backup',
  'pullImages',
  'applyUpdate',
  'migration',
  'startServices',
  'healthCheck',
] as const;
const stepOptions = computed(() =>
  steps.map((step) => ({
    title: t(`update.center.progress.steps.${step}`),
  })),
);
const stageIndex: Record<UpdateOperationPhase, number> = {
  READY: 0,
  PREFLIGHT: 1,
  BACKUP: 2,
  PULL_IMAGES: 3,
  STOP_SERVICES: 4,
  APPLY_UPDATE: 4,
  MIGRATION: 5,
  START_SERVICES: 6,
  HEALTH_CHECK: 7,
  SUCCESS: 7,
  FAILED: 7,
  ROLLBACK: 7,
};
const stageStatus = computed(() => {
  if (progress.lastActivePhase) return progress.lastActivePhase;
  return progress.operation?.phase ?? 'READY';
});
const runnerDisconnected = computed(
  () =>
    progress.operation?.state_source === 'runner_lost' || progress.operation?.state_source === 'runner_state_corrupt',
);
const canRecoverTerminatedRunner = computed(
  () =>
    (progress.operation?.state_source === 'runner_lost' ||
      progress.operation?.state_source === 'runner_state_corrupt') &&
    !progress.recoveryPending &&
    permissionStore.hasPermission(UPDATE_PERMISSION_CODE.MANAGE),
);
const showProgress = computed(() => !runnerDisconnected.value);
const showRunnerStage = computed(() => Boolean(progress.operation) && !runnerDisconnected.value);
const currentStep = computed(() => stageIndex[stageStatus.value]);
const overallPercentage = computed(() => progress.operation?.progress ?? 0);
const currentStageLabel = computed(() => {
  return t(`update.center.history.phases.${stageStatus.value}`);
});
const phaseMessage = computed(() => t(`update.center.progress.phase.${progress.phase}`));
const terminal = computed(() => progress.isTerminal());

const eventMessageKeys = new Set([
  'runner_starting',
  'runner_accepted',
  'checking_environment',
  'creating_backup',
  'pulling_images',
  'verifying_images',
  'stopping_services',
  'applying_update',
  'running_migrations',
  'starting_services',
  'checking_health',
  'update_completed',
  'update_failed',
  'rollback_completed',
]);

onMounted(() => progress.resume());

function closeTerminalDialog() {
  if (terminal.value) progress.reset();
}

function recoverTerminatedRunner() {
  void progress.recoverTerminatedRunner();
}

function eventMessage(message: string) {
  return eventMessageKeys.has(message)
    ? t(`update.center.history.messages.${message}`)
    : t('update.center.progress.events.recorded');
}
</script>
<style scoped lang="less">
.update-progress {
  display: grid;
  gap: var(--td-comp-margin-l);
}

.update-progress__failure {
  color: var(--td-text-color-secondary);
  display: grid;
  gap: var(--td-comp-margin-s);
}

.update-progress__failure p {
  margin: 0;
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.update-progress__recovery {
  display: grid;
  gap: var(--td-comp-margin-s);
}

.update-progress__recovery p {
  color: var(--td-text-color-secondary);
  margin: 0;
}

.update-progress__current-stage {
  display: grid;
  gap: var(--td-comp-margin-s);
}

.update-progress__current-stage-heading {
  color: var(--td-text-color-secondary);
  display: flex;
  gap: var(--td-comp-margin-s);
  justify-content: space-between;
}

.update-progress__current-stage-heading strong {
  color: var(--td-text-color-primary);
  font-weight: var(--td-font-weight-medium);
}

.update-progress__events {
  border-top: 1px solid var(--td-component-border);
  display: grid;
  gap: var(--td-comp-margin-s);
  padding-top: var(--td-comp-paddingTB-m);
}

.update-progress__events h3 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  margin: 0;
}

.update-progress__events ol {
  display: grid;
  gap: var(--td-comp-margin-xs);
  list-style: none;
  margin: 0;
  max-height: 12rem;
  overflow: auto;
  padding: 0;
}

.update-progress__events li {
  color: var(--td-text-color-secondary);
  display: grid;
  gap: var(--td-comp-margin-xs);
  grid-template-columns: minmax(7rem, auto) minmax(0, 1fr);
  overflow-wrap: anywhere;
}

.update-progress__events strong {
  color: var(--td-text-color-primary);
  font-weight: var(--td-font-weight-medium);
}
</style>
