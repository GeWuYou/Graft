import { describe, expect, it } from 'vitest';

import {
  type CronDescriptionResult,
  describeCronExpression,
  normalizeCronExpression,
  previewCronExecutions,
  validateCronExpression,
} from './cron';

describe('scheduled-task cron utility', () => {
  it('normalizes 5-field Unix cron expressions to backend seconds cron', () => {
    expect(normalizeCronExpression('*/5 * * * *')).toBe('0 */5 * * * *');
    expect(normalizeCronExpression('  0   0  *  *  *  ')).toBe('0 0 0 * * *');
  });

  it('keeps 6-field seconds cron expressions canonical', () => {
    expect(normalizeCronExpression('0 */10 * * * *')).toBe('0 */10 * * * *');
  });

  it('accepts common generated cron fields supported by the backend parser contract', () => {
    expect(validateCronExpression('*/30 */5 * * * *')).toEqual({ valid: true });
    expect(validateCronExpression('0 0 23 31 12 7')).toEqual({ valid: true });
    expect(validateCronExpression('00 05 09 * * *')).toEqual({ valid: true });
    expect(validateCronExpression('* * * * *')).toEqual({ valid: true });
  });

  it('rejects unsupported field counts', () => {
    expect(validateCronExpression('* * * *').valid).toBe(false);
    expect(validateCronExpression('0 * * * * * *').valid).toBe(false);
    expect(validateCronExpression('*/5 * * *')).toMatchObject({
      valid: false,
      messageKey: 'scheduledTask.cronValidation.fieldCount',
    });
  });

  it('rejects empty cron expressions as required input', () => {
    expect(validateCronExpression('')).toEqual({
      valid: false,
      messageKey: 'scheduledTask.cronValidation.required',
    });
  });

  it('rejects out-of-range simple fields', () => {
    expect(validateCronExpression('60 * * * * *')).toMatchObject({ valid: false });
    expect(validateCronExpression('0 60 * * * *')).toMatchObject({ valid: false });
    expect(validateCronExpression('0 0 24 * * *')).toMatchObject({ valid: false });
    expect(validateCronExpression('0 0 * 0 * *')).toMatchObject({ valid: false });
    expect(validateCronExpression('0 0 * * 13 *')).toMatchObject({ valid: false });
    expect(validateCronExpression('0 0 * * * 8')).toMatchObject({ valid: false });
  });

  it('rejects unsupported complex syntax in the local utility surface', () => {
    expect(validateCronExpression('0 0 9-17 * * *')).toMatchObject({ valid: false });
    expect(validateCronExpression('0 0 9,12 * * *')).toMatchObject({ valid: false });
    expect(validateCronExpression('0 0 9 ? * MON')).toMatchObject({ valid: false });
  });

  it('describes recognized simple schedules with i18n-safe keys', () => {
    expect(describeCronExpression('* * * * *')).toMatchObject({
      key: 'scheduledTask.cronDescription.everyMinute',
      normalizedExpression: '0 * * * * *',
      valid: true,
    });
    expect(describeCronExpression('*/5 * * * *')).toMatchObject({
      key: 'scheduledTask.cronDescription.everyNMinutes',
      params: { interval: 5 },
      normalizedExpression: '0 */5 * * * *',
      valid: true,
    });
    expect(describeCronExpression('0 0 * * *')).toMatchObject({
      key: 'scheduledTask.cronDescription.daily',
      params: { hour: 0 },
    });
    expect(describeCronExpression('0 17 * * *')).toMatchObject({
      key: 'scheduledTask.cronDescription.daily',
      normalizedExpression: '0 0 17 * * *',
      params: { hour: 17 },
      valid: true,
    });
    expect(describeCronExpression('0 0 17 * * *')).toMatchObject({
      key: 'scheduledTask.cronDescription.daily',
      normalizedExpression: '0 0 17 * * *',
      params: { hour: 17 },
      valid: true,
    });
  });

  it('uses custom fallback for valid but unrecognized simple schedules', () => {
    expect(describeCronExpression('15 30 8 * * *')).toMatchObject({
      key: 'scheduledTask.cronDescription.custom',
      params: { expression: '15 30 8 * * *' },
      valid: true,
    });
  });

  it('can return a translated description when a translator is provided', () => {
    const description = describeCronExpression('*/10 * * * *', (key, params) => `${key}:${params?.interval}`);

    expect(description).toBe('scheduledTask.cronDescription.everyNMinutes:10');
  });

  it('returns invalid description metadata for invalid expressions', () => {
    const description = describeCronExpression('0 0 24 * * *') as CronDescriptionResult;

    expect(description.valid).toBe(false);
    expect(description.key).toBe('scheduledTask.cronDescription.invalid');
  });

  it('previews upcoming execution times from the current local time', () => {
    const preview = previewCronExecutions('*/5 * * * *', new Date(2026, 5, 6, 10, 2, 0), 3);

    expect(preview.valid).toBe(true);
    expect(preview.normalizedExpression).toBe('0 */5 * * * *');
    expect(preview.intervalMs).toBe(5 * 60 * 1000);
    expect(preview.nextRuns.map((run) => [run.getHours(), run.getMinutes(), run.getSeconds()])).toEqual([
      [10, 5, 0],
      [10, 10, 0],
      [10, 15, 0],
    ]);
  });

  it('previews daily execution times across day boundaries', () => {
    const preview = previewCronExecutions('0 17 * * *', new Date(2026, 5, 6, 18, 0, 0), 2);

    expect(preview.nextRuns.map((run) => [run.getDate(), run.getHours(), run.getMinutes()])).toEqual([
      [7, 17, 0],
      [8, 17, 0],
    ]);
  });
});
