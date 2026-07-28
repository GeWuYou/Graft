<template>
  <t-dialog
    :visible="progress.visible"
    :header="t('update.center.progress.title')"
    :cancel-btn="null"
    :confirm-btn="null"
    :close-btn="false"
    :close-on-esc-keydown="false"
    :close-on-overlay-click="false"
    :prevent-scroll-through="true"
    :footer="false"
    width="480px"
  >
    <section class="update-progress" data-testid="update-progress-dialog">
      <t-progress v-if="progress.phase !== 'failed'" :percentage="100" :indeterminate="progress.phase !== 'success'" />
      <t-alert
        :theme="progress.phase === 'failed' ? 'error' : progress.phase === 'success' ? 'success' : 'info'"
        :message="phaseMessage"
      />
      <t-steps v-if="progress.operation" :current="currentStep" readonly>
        <t-step v-for="step in steps" :key="step" :title="t(`update.center.progress.steps.${step}`)" />
      </t-steps>
      <section v-if="progress.phase === 'failed'" class="update-progress__failure">
        <p>{{ progress.diagnostic?.detail || t('update.center.progress.diagnosticUnavailable') }}</p>
        <t-button v-if="requestId" variant="text" @click="openAppLogs">{{
          t('update.center.progress.viewAppLogs')
        }}</t-button>
      </section>
    </section>
  </t-dialog>
</template>
<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { buildAppLogLocation } from '@/modules/app-log/contract/deep-link';

import { useUpdateProgressStore } from '../store/progress';

const { t } = useI18n();
const router = useRouter();
const progress = useUpdateProgressStore();
const steps = ['planning', 'pulling', 'recreating', 'verifying'] as const;
const stageIndex: Record<string, number> = {
  PLANNING: 0,
  BACKING_UP: 0,
  PULLING: 1,
  MIGRATING: 1,
  RECREATING: 2,
  VERIFYING: 3,
  SUCCESS: 3,
  RECOVERED: 3,
};
const currentStep = computed(() => stageIndex[progress.operation?.status ?? 'PLANNING'] ?? 0);
const requestId = computed(() => progress.diagnostic?.request_id?.trim() ?? '');
const phaseMessage = computed(() => t(`update.center.progress.phase.${progress.phase}`));
function openAppLogs() {
  if (requestId.value) void router.push(buildAppLogLocation({ request_id: requestId.value }));
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
</style>
