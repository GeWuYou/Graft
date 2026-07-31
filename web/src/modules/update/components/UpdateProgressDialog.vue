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
        :theme="progress.phase === 'failed' ? 'error' : progress.phase === 'success' ? 'success' : 'info'"
        :message="phaseMessage"
      />
      <t-steps v-if="progress.operation" :current="currentStep" layout="vertical" readonly>
        <t-step v-for="step in steps" :key="step" :title="t(`update.center.progress.steps.${step}`)" />
      </t-steps>
      <section v-if="progress.operation && progress.phase !== 'success'" class="update-progress__current-stage">
        <div class="update-progress__current-stage-heading">
          <span>{{ t('update.center.progress.currentStage') }}</span>
          <strong>{{ currentStageLabel }}</strong>
        </div>
        <t-progress data-testid="update-progress-stage" :percentage="currentStagePercentage" :label="false" />
      </section>
      <section v-if="progress.phase === 'failed'" class="update-progress__failure">
        <p>{{ progress.diagnostic?.detail || t('update.center.progress.diagnosticUnavailable') }}</p>
        <t-button v-if="requestId" variant="text" @click="openAppLogs">{{
          t('update.center.progress.viewAppLogs')
        }}</t-button>
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
import { useRouter } from 'vue-router';

import { buildAppLogLocation } from '@/modules/app-log/contract/deep-link';

import { useUpdateProgressStore } from '../store/progress';

// 后台壳唯一挂载的升级会话表面：恢复同标签页会话，并在运行中阻止误关弹窗。
const { t } = useI18n();
const router = useRouter();
const progress = useUpdateProgressStore();
const steps = ['planning', 'backingUp', 'pulling', 'migrating', 'recreating', 'verifying'] as const;
const stageIndex: Record<string, number> = {
  PLANNING: 0,
  BACKING_UP: 1,
  PULLING: 2,
  MIGRATING: 3,
  RECREATING: 4,
  VERIFYING: 5,
  SUCCESS: 5,
  RECOVERED: 5,
};
const stagePercentage: Record<string, number> = {
  PLANNING: 0,
  BACKING_UP: 10,
  PULLING: 30,
  MIGRATING: 55,
  RECREATING: 75,
  VERIFYING: 90,
  SUCCESS: 100,
  RECOVERED: 100,
};
const stageStatus = computed(() => {
  if (progress.lastActiveStatus) return progress.lastActiveStatus;
  if (progress.phase === 'success' || progress.phase === 'failed') return 'VERIFYING';
  return progress.operation?.status ?? 'PLANNING';
});
const currentStep = computed(() => stageIndex[stageStatus.value] ?? 0);
const overallPercentage = computed(() => stagePercentage[stageStatus.value] ?? 0);
const currentStagePercentage = computed(() => overallPercentage.value);
const currentStageLabel = computed(() => {
  return t(`update.center.history.status.${stageStatus.value}`);
});
const requestId = computed(() => progress.diagnostic?.request_id?.trim() ?? '');
const phaseMessage = computed(() => t(`update.center.progress.phase.${progress.phase}`));
const terminal = computed(() => progress.phase === 'success' || progress.phase === 'failed');

onMounted(() => progress.resume());

function closeTerminalDialog() {
  if (terminal.value) progress.reset();
}

function openAppLogs() {
  const operationRequestId = requestId.value;
  if (!operationRequestId) return;
  progress.reset();
  void router.push(buildAppLogLocation({ request_id: operationRequestId }));
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
</style>
