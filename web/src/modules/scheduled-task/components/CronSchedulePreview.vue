<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="cron-schedule-preview" data-testid="cron-preview">
    <div class="cron-schedule-preview__row">
      <span>{{ t('scheduledTask.cronScheduleDialog.preview.expression') }}</span>
      <code data-testid="cron-preview-expression">{{ displayExpression || emptyExpressionText }}</code>
    </div>
    <div class="cron-schedule-preview__row">
      <span>{{ t('scheduledTask.cronScheduleDialog.preview.status') }}</span>
      <t-tag :theme="validation.valid ? 'success' : 'danger'" variant="light">
        {{ statusText }}
      </t-tag>
    </div>
    <div class="cron-schedule-preview__row">
      <span>{{ t('scheduledTask.cronScheduleDialog.preview.description') }}</span>
      <p data-testid="cron-preview-description">{{ descriptionText }}</p>
    </div>
    <div class="cron-schedule-preview__row">
      <span>{{ t('scheduledTask.cronScheduleDialog.preview.nextRun') }}</span>
      <p data-testid="cron-preview-next-run">{{ nextRunText }}</p>
    </div>
    <div class="cron-schedule-preview__row">
      <span>{{ t('scheduledTask.cronScheduleDialog.preview.interval') }}</span>
      <p data-testid="cron-preview-interval">{{ intervalText }}</p>
    </div>
    <div class="cron-schedule-preview__row cron-schedule-preview__row--runs">
      <span>{{ t('scheduledTask.cronScheduleDialog.preview.upcomingRuns') }}</span>
      <div class="cron-schedule-preview__run-list" data-testid="cron-preview-upcoming-runs">
        <p v-for="run in upcomingRunTexts" :key="run">{{ run }}</p>
        <p v-if="upcomingRunTexts.length === 0">{{ t('scheduledTask.cronScheduleDialog.preview.noPreview') }}</p>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import {
  type CronValidationResult,
  describeCronExpression,
  getNextRuns,
  toUnixCronExpression,
  validateCronExpression,
} from '../utils/cron';
import { translateCronDescription, translateCronValidation } from '../utils/cron-i18n';

const props = defineProps<{
  expression: string;
}>();

const { locale, t } = useI18n();

const validation = computed(() => validateCronExpression(props.expression));
const displayExpression = computed(() => toUnixCronExpression(props.expression));
const emptyExpressionText = computed(() => t('scheduledTask.cronScheduleDialog.preview.emptyExpression'));
const descriptionText = computed(() => translateCronDescription(describeCronExpression(props.expression), t));
const nextRuns = computed(() => getNextRuns(props.expression, 4));
const statusText = computed(() =>
  validation.value.valid
    ? t('scheduledTask.cronScheduleDialog.preview.valid')
    : cronValidationMessageText(validation.value) || t('scheduledTask.cronScheduleDialog.preview.invalid'),
);
const nextRunText = computed(() => {
  const nextRun = nextRuns.value[0];
  return nextRun ? formatPreviewDate(nextRun, true) : t('scheduledTask.cronScheduleDialog.preview.noPreview');
});
const intervalText = computed(() => {
  if (nextRuns.value.length < 2) {
    return t('scheduledTask.cronScheduleDialog.preview.noPreview');
  }

  return formatInterval(nextRuns.value[1].getTime() - nextRuns.value[0].getTime());
});
const upcomingRunTexts = computed(() => nextRuns.value.slice(0, 3).map((run) => formatPreviewDate(run, false)));

function cronValidationMessageText(result: CronValidationResult) {
  return translateCronValidation(result, t);
}

function formatPreviewDate(value: Date, withSeconds: boolean) {
  return new Intl.DateTimeFormat(locale.value, {
    dateStyle: 'short',
    timeStyle: withSeconds ? 'medium' : 'short',
  }).format(value);
}

function formatInterval(intervalMs: number) {
  const totalSeconds = Math.round(intervalMs / 1000);
  if (totalSeconds < 60) {
    return t('scheduledTask.cronScheduleDialog.preview.intervalSeconds', { count: totalSeconds });
  }

  const totalMinutes = Math.round(totalSeconds / 60);
  if (totalMinutes < 60) {
    return t('scheduledTask.cronScheduleDialog.preview.intervalMinutes', { count: totalMinutes });
  }

  const totalHours = Math.round(totalMinutes / 60);
  if (totalHours < 24) {
    return t('scheduledTask.cronScheduleDialog.preview.intervalHours', { count: totalHours });
  }

  const totalDays = Math.round(totalHours / 24);
  return t('scheduledTask.cronScheduleDialog.preview.intervalDays', { count: totalDays });
}
</script>
<style scoped lang="less">
.cron-schedule-preview {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
  display: grid;
  gap: var(--graft-density-gap-8);
  padding: var(--graft-density-gap-12);
}

.cron-schedule-preview__row {
  align-items: center;
  display: grid;
  gap: var(--graft-density-gap-8);
  grid-template-columns: 76px minmax(0, 1fr);
  min-width: 0;
}

.cron-schedule-preview__row > span {
  color: var(--td-text-color-secondary);
}

.cron-schedule-preview__row code {
  background: var(--td-bg-color-secondarycontainer);
  border-radius: var(--td-radius-default);
  color: var(--td-text-color-primary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace;
  overflow-wrap: anywhere;
  padding: var(--td-comp-paddingTB-xs) var(--td-comp-paddingLR-s);
  width: fit-content;
}

.cron-schedule-preview__row p {
  color: var(--td-text-color-primary);
  margin: 0;
  min-width: 0;
  overflow-wrap: anywhere;
}

.cron-schedule-preview__row--runs {
  align-items: start;
}

.cron-schedule-preview__run-list {
  display: grid;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}
</style>
