<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="cron-expression-field scheduled-task-cron-field" :class="{ 'cron-expression-field--disabled': disabled }">
    <div class="scheduled-task-cron-row" data-testid="cron-expression-row">
      <div class="scheduled-task-cron-input">
        <t-input
          v-model="rawExpression"
          data-testid="cron-expression-input"
          clearable
          :disabled="disabled"
          :placeholder="t('scheduledTask.cronExpressionField.placeholder')"
          :status="inputStatus"
          @change="handleRawInput"
          @blur="handleInputBlur"
        />
      </div>
      <t-button
        class="scheduled-task-cron-configure"
        data-testid="cron-config-button"
        theme="primary"
        variant="outline"
        :disabled="disabled"
        @click="dialogVisible = true"
      >
        {{ t('scheduledTask.cronExpressionField.configure') }}
      </t-button>
    </div>

    <div
      v-if="invalidMessage"
      class="scheduled-task-cron-message scheduled-task-cron-message--error"
      data-testid="cron-expression-error"
    >
      {{ invalidMessage }}
    </div>

    <div
      v-else-if="displayValid"
      class="scheduled-task-cron-message scheduled-task-cron-message--success"
      data-testid="cron-expression-meta"
    >
      <t-tag size="small" theme="success" variant="light">
        {{ t('scheduledTask.cronExpressionField.validStatus') }}
      </t-tag>
      <span>{{ descriptionText }}</span>
    </div>

    <cron-schedule-dialog
      v-model:visible="dialogVisible"
      :model-value="rawExpression"
      :disabled="disabled"
      @confirm="handleDialogConfirm"
    />
  </div>
</template>
<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import {
  type CronValidationResult,
  describeCronExpression,
  normalizeCronExpression,
  validateCronExpression,
} from '../utils/cron';
import { translateCronDescription, translateCronValidation } from '../utils/cron-i18n';
import CronScheduleDialog from './CronScheduleDialog.vue';

const props = withDefaults(
  defineProps<{
    modelValue: string;
    disabled?: boolean;
    error?: string;
  }>(),
  {
    disabled: false,
    error: '',
  },
);

const emit = defineEmits<{
  'update:modelValue': [value: string];
  validate: [result: CronValidationResult & { normalizedExpression: string }];
}>();

const { t } = useI18n();
const rawExpression = ref(props.modelValue);
const dialogVisible = ref(false);

const normalizedExpression = computed(() => normalizeCronExpression(rawExpression.value));
const validation = computed(() => validateCronExpression(rawExpression.value));
const description = computed(() => describeCronExpression(rawExpression.value));
const invalidMessage = computed(() => props.error || cronValidationMessageText(validation.value));
const displayValid = computed(() => validation.value.valid && !props.error);
const inputStatus = computed(() => (invalidMessage.value ? 'error' : 'default'));
const descriptionText = computed(() => translateCronDescription(description.value, t));

watch(
  () => props.modelValue,
  (value) => {
    if (value !== rawExpression.value) {
      rawExpression.value = value;
    }
  },
);

watch(
  () =>
    [
      validation.value.valid,
      validation.value.messageKey,
      validation.value.messageParams,
      normalizedExpression.value,
    ] as const,
  () => {
    emitValidation();
  },
  { immediate: true },
);

function handleRawInput(value: string | number) {
  applyExpression(String(value));
}

function handleInputBlur(value: string | number) {
  handleRawInput(value);
}

function handleDialogConfirm(expression: string) {
  applyExpression(expression);
}

function applyExpression(value: string) {
  const nextExpression = normalizeCronExpression(value);
  rawExpression.value = nextExpression;
  emit('update:modelValue', nextExpression);
  emitValidation();
}

function emitValidation() {
  emit('validate', {
    ...validation.value,
    normalizedExpression: normalizedExpression.value,
  });
}

function cronValidationMessageText(result: CronValidationResult) {
  return translateCronValidation(result, t);
}
</script>
<style scoped lang="less">
.cron-expression-field,
.scheduled-task-cron-field {
  display: flex;
  flex-direction: column;
  min-width: 0;
  width: 100%;
}

.scheduled-task-cron-row {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-10);
  min-width: 0;
  width: 100%;
}

.scheduled-task-cron-input {
  flex: 1 1 0;
  min-width: 0;
}

.scheduled-task-cron-input :deep(.t-input__wrap) {
  min-width: 0;
  width: 100%;
}

.scheduled-task-cron-configure {
  flex: 0 0 auto;
  min-width: var(--td-comp-size-xxxl);
}

.scheduled-task-cron-message {
  font: var(--td-font-body-small);
  line-height: var(--td-line-height-body-small);
  margin-top: var(--td-comp-margin-xs);
  min-width: 0;
  overflow-wrap: anywhere;
}

.scheduled-task-cron-message--error {
  color: var(--td-error-color);
}

.scheduled-task-cron-message--success {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: flex;
  gap: var(--td-comp-margin-xs);
}

.cron-expression-field--disabled {
  opacity: 0.72;
}
</style>
