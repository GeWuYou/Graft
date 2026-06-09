// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { CronDescriptionResult, CronValidationResult } from './cron';

type CronTranslate = (key: string, params?: Record<string, string | number>) => string;

export function translateCronDescription(result: CronDescriptionResult | string, t: CronTranslate): string {
  if (typeof result === 'string') {
    return result;
  }

  switch (result.key) {
    case 'scheduledTask.cronDescription.everyMinute':
      return t('scheduledTask.cronDescription.everyMinute', result.params ?? {});
    case 'scheduledTask.cronDescription.everyNMinutes':
      return t('scheduledTask.cronDescription.everyNMinutes', result.params ?? {});
    case 'scheduledTask.cronDescription.hourly':
      return t('scheduledTask.cronDescription.hourly', result.params ?? {});
    case 'scheduledTask.cronDescription.daily':
      return t('scheduledTask.cronDescription.daily', result.params ?? {});
    case 'scheduledTask.cronDescription.weekly':
      return t('scheduledTask.cronDescription.weekly', result.params ?? {});
    case 'scheduledTask.cronDescription.monthly':
      return t('scheduledTask.cronDescription.monthly', result.params ?? {});
    case 'scheduledTask.cronDescription.custom':
      return t('scheduledTask.cronDescription.custom', result.params ?? {});
    case 'scheduledTask.cronDescription.invalid':
    default:
      return t('scheduledTask.cronDescription.invalid', result.params ?? {});
  }
}

export function translateCronValidation(result: CronValidationResult, t: CronTranslate): string {
  switch (result.messageKey) {
    case 'scheduledTask.cronValidation.required':
      return t('scheduledTask.cronValidation.required', result.messageParams ?? {});
    case 'scheduledTask.cronValidation.fieldCount':
      return t('scheduledTask.cronValidation.fieldCount', result.messageParams ?? {});
    case 'scheduledTask.cronValidation.stepRange':
      return t('scheduledTask.cronValidation.stepRange', result.messageParams ?? {});
    case 'scheduledTask.cronValidation.fieldRange':
      return t('scheduledTask.cronValidation.fieldRange', result.messageParams ?? {});
    default:
      return '';
  }
}
