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
      <t-progress data-testid="update-progress-overall" :percentage="overallPercentage" :label="true" />
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
      <t-steps v-if="progress.operation" :current="currentStep" layout="vertical" readonly>
        <t-step v-for="step in steps" :key="step" :title="t(`update.center.progress.steps.${step}`)" />
      </t-steps>
      <section v-if="progress.operation" class="update-progress__current-stage">
        <div class="update-progress__current-stage-heading">
          <span>{{ t('update.center.progress.currentStage') }}</span>
          <strong>{{ currentStageLabel }}</strong>
        </div>
      </section>
      <section v-if="progress.phase === 'failed'" class="update-progress__failure">
        <p>{{ progress.operation?.message || t('update.center.progress.diagnosticUnavailable') }}</p>
      </section>
      <section v-if="progress.phase === 'unavailable'" class="update-progress__failure">
        <p>{{ t('update.center.progress.sourceUnavailable') }}</p>
      </section>
      <section v-if="progress.events.length" class="update-progress__events" data-testid="update-progress-events">
        <h3>{{ t('update.center.progress.events.title') }}</h3>
        <ol>
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

import { useUpdateProgressStore } from '../store/progress';
import type { UpdateOperationPhase } from '../types/update';

// 后台壳只呈现 runner 快照；服务重建中的连接状态不能覆盖 runner 报告的真实进度。
const { t } = useI18n();
const progress = useUpdateProgressStore();
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
const currentStep = computed(() => stageIndex[stageStatus.value]);
const overallPercentage = computed(() => progress.operation?.progress ?? 0);
const currentStageLabel = computed(() => {
  return t(`update.center.history.phases.${stageStatus.value}`);
});
const phaseMessage = computed(() => t(`update.center.progress.phase.${progress.phase}`));
const terminal = computed(
  () => progress.phase === 'success' || progress.phase === 'failed' || progress.phase === 'unavailable',
);

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
