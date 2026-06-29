import {
  formatCronClockTime,
  type NormalizedCronFields,
  readDailySchedule,
  readIntervalMinuteMode,
  readMonthlyDescriptionSchedule,
  readWeeklySchedule,
  toNormalizedCronFields,
} from './cron-internal';
import type { CronDescriptionFormatOptions, CronDescriptionResult, CronDescriptionTranslate } from './cron-types';

export function getAdvancedCronDescription(options: CronDescriptionFormatOptions): string {
  return (
    options.advancedExpressionText ||
    options.translate?.('scheduledTask.cronDescription.advanced') ||
    'scheduledTask.cronDescription.advanced'
  );
}

export function translateCronDescription(
  description: CronDescriptionResult,
  translate?: CronDescriptionTranslate,
): string {
  return translate?.(description.key, description.params) ?? description.key;
}

export function describeNormalizedCronExpression(normalizedExpression: string): CronDescriptionResult {
  const normalizedFields = toNormalizedCronFields(normalizedExpression);

  if (!normalizedFields) {
    return {
      key: 'scheduledTask.cronDescription.custom',
      params: { expression: normalizedExpression },
      normalizedExpression,
      valid: true,
    };
  }

  if (isEveryMinutePattern(normalizedFields)) {
    return {
      key: 'scheduledTask.cronDescription.everyMinute',
      normalizedExpression,
      valid: true,
    };
  }

  const intervalMinutes = readIntervalMinuteMode(normalizedFields);
  if (intervalMinutes !== null) {
    return {
      key: 'scheduledTask.cronDescription.everyNMinutes',
      params: { interval: intervalMinutes },
      normalizedExpression,
      valid: true,
    };
  }

  if (isHourlyPattern(normalizedFields)) {
    return {
      key: 'scheduledTask.cronDescription.hourly',
      normalizedExpression,
      valid: true,
    };
  }

  const dailySchedule = readDailySchedule(normalizedFields);
  if (dailySchedule) {
    return {
      key: 'scheduledTask.cronDescription.daily',
      params: {
        ...dailySchedule,
        time: formatCronClockTime(dailySchedule.hour, dailySchedule.minute),
      },
      normalizedExpression,
      valid: true,
    };
  }

  const weeklySchedule = readWeeklySchedule(normalizedFields);
  if (weeklySchedule) {
    return {
      key: 'scheduledTask.cronDescription.weekly',
      params: {
        ...weeklySchedule,
        time: formatCronClockTime(weeklySchedule.hour, weeklySchedule.minute),
      },
      normalizedExpression,
      valid: true,
    };
  }

  const monthlySchedule = readMonthlyDescriptionSchedule(normalizedFields);
  if (monthlySchedule) {
    return {
      key: 'scheduledTask.cronDescription.monthly',
      params: { hour: monthlySchedule.hour, dayOfMonth: monthlySchedule.dayOfMonth },
      normalizedExpression,
      valid: true,
    };
  }

  return {
    key: 'scheduledTask.cronDescription.custom',
    params: { expression: normalizedExpression },
    normalizedExpression,
    valid: true,
  };
}

export function polishCronDescription(description: string, expression: string, locale: string): string {
  if (locale !== 'zh_CN') {
    return description;
  }

  const dailySchedule = readDailyScheduleFromExpression(expression);
  if (dailySchedule && /每天/.test(description) && /在\s*\d{1,2}:\d{2}/.test(description)) {
    return description
      .replace(/\d{1,2}:\d{2}/, formatCronClockTime(dailySchedule.hour, dailySchedule.minute))
      .replace(/,\s*/g, '，');
  }

  return description.replace(/,\s*/g, '，');
}

export function toCronstrueLocale(locale?: string): string {
  return locale?.toLowerCase().startsWith('zh') ? 'zh_CN' : 'en';
}

function isEveryMinutePattern([second, minute, hour, dayOfMonth, month, dayOfWeek]: NormalizedCronFields) {
  return second === '0' && minute === '*' && hour === '*' && dayOfMonth === '*' && month === '*' && dayOfWeek === '*';
}

function isHourlyPattern([second, minute, hour, dayOfMonth, month, dayOfWeek]: NormalizedCronFields) {
  return second === '0' && minute === '0' && hour === '*' && dayOfMonth === '*' && month === '*' && dayOfWeek === '*';
}

function readDailyScheduleFromExpression(expression: string) {
  const normalizedFields = toNormalizedCronFields(expression);
  return normalizedFields ? readDailySchedule(normalizedFields) : null;
}
