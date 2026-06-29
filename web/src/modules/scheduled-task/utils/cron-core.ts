import 'cronstrue/locales/zh_CN';

import { CronExpressionParser } from 'cron-parser';
import cronstrue from 'cronstrue';

import {
  describeNormalizedCronExpression,
  getAdvancedCronDescription,
  polishCronDescription,
  toCronstrueLocale,
  translateCronDescription,
} from './cron-description';
import {
  buildExpandedCronFieldValues,
  clampInteger,
  collectNextRuns,
  createAdvancedParsedCronExpression,
  CRON_FIELD_RULES,
  DEFAULT_PARSED_CRON_VALUE,
  FIELD_COUNT_SECONDS,
  FIELD_COUNT_UNIX,
  isSupportedCronFieldCount,
  readDailySchedule,
  readHourlyMinute,
  readIntervalMinuteMode,
  readMonthlySchedule,
  readWeeklySchedule,
  splitCronFields,
  toNormalizedCronFields,
  validateCronField,
} from './cron-internal';
import type {
  CronDescriptionFormatOptions,
  CronDescriptionResult,
  CronDescriptionTranslate,
  CronExecutionPreview,
  CronMode,
  CronNextRunFormatOptions,
  CronScheduleValue,
  CronValidationResult,
  ParsedCronExpression,
} from './cron-types';

export type {
  CronDescriptionFormatOptions,
  CronDescriptionKey,
  CronDescriptionResult,
  CronDescriptionTranslate,
  CronExecutionPreview,
  CronMode,
  CronNextRunFormatOptions,
  CronScheduleValue,
  CronValidationMessageKey,
  CronValidationResult,
  ParsedCronExpression,
} from './cron-types';

export function normalizeCronExpression(expression: string): string {
  const fields = splitCronFields(expression);

  if (fields.length === FIELD_COUNT_UNIX) {
    return ['0', ...fields].join(' ');
  }

  return fields.join(' ');
}

export function formatCronExpression(expression: string): string {
  return splitCronFields(expression).join(' ');
}

export function getNextRunText(expression: string, timezone?: string, options: CronNextRunFormatOptions = {}): string {
  const formattedExpression = formatCronExpression(expression);
  if (!isSupportedCronFieldCount(formattedExpression)) {
    return '';
  }

  try {
    const interval = CronExpressionParser.parse(formattedExpression, {
      currentDate: options.now ?? new Date(),
      tz: timezone,
    });

    return formatCronDateTime(interval.next().toDate(), options.locale, timezone);
  } catch {
    return '';
  }
}

export function getCronDescription(
  expression: string,
  locale?: string,
  options: CronDescriptionFormatOptions = {},
): string {
  const formattedExpression = formatCronExpression(expression);
  if (!isSupportedCronFieldCount(formattedExpression)) {
    return getAdvancedCronDescription(options);
  }

  try {
    const normalizedExpression = normalizeCronExpression(formattedExpression);
    const simpleDescription = describeNormalizedCronExpression(normalizedExpression);

    if (simpleDescription.valid && simpleDescription.key !== 'scheduledTask.cronDescription.custom') {
      return translateCronDescription(simpleDescription, options.translate);
    }

    const cronstrueLocale = toCronstrueLocale(locale);
    const description = cronstrue.toString(formattedExpression, {
      locale: cronstrueLocale,
      throwExceptionOnParseError: true,
      use24HourTimeFormat: true,
      verbose: true,
    });

    return polishCronDescription(description, normalizedExpression, cronstrueLocale);
  } catch {
    return getAdvancedCronDescription(options);
  }
}

export function toUnixCronExpression(expression: string): string {
  const fields = splitCronFields(expression);

  if (fields.length === FIELD_COUNT_SECONDS && fields[0] === '0') {
    return fields.slice(1).join(' ');
  }

  return fields.join(' ');
}

export function buildCronExpression(mode: CronMode, value: CronScheduleValue): string {
  const minute = clampInteger(value.minute, 0, 59);
  const hour = clampInteger(value.hour, 0, 23);

  switch (mode) {
    case 'intervalMinutes':
      return `*/${clampInteger(value.intervalMinutes, 1, 59)} * * * *`;
    case 'hourly':
      return `${minute} * * * *`;
    case 'daily':
      return `${minute} ${hour} * * *`;
    case 'weekly':
      return `${minute} ${hour} * * ${clampInteger(value.weekday, 0, 6)}`;
    case 'monthly':
      return `${minute} ${hour} ${clampInteger(value.dayOfMonth, 1, 31)} * *`;
    case 'advanced':
    default:
      return `${minute} ${hour} * * *`;
  }
}

export function parseCronExpression(expression: string): ParsedCronExpression {
  const unixExpression = toUnixCronExpression(expression || '0 17 * * *');
  const normalizedFields = toNormalizedCronFields(unixExpression);

  if (!normalizedFields || !validateCronExpression(unixExpression).valid) {
    return createAdvancedParsedCronExpression(unixExpression);
  }

  const intervalMinutes = readIntervalMinuteMode(normalizedFields);
  if (intervalMinutes !== null) {
    return {
      expression: unixExpression,
      mode: 'intervalMinutes',
      value: { ...DEFAULT_PARSED_CRON_VALUE, intervalMinutes },
    };
  }

  const hourlyMinute = readHourlyMinute(normalizedFields);
  if (hourlyMinute !== null) {
    return {
      expression: unixExpression,
      mode: 'hourly',
      value: { ...DEFAULT_PARSED_CRON_VALUE, minute: hourlyMinute },
    };
  }

  const dailySchedule = readDailySchedule(normalizedFields);
  if (dailySchedule) {
    return {
      expression: unixExpression,
      mode: 'daily',
      value: { ...DEFAULT_PARSED_CRON_VALUE, ...dailySchedule },
    };
  }

  const weeklySchedule = readWeeklySchedule(normalizedFields);
  if (weeklySchedule) {
    return {
      expression: unixExpression,
      mode: 'weekly',
      value: { ...DEFAULT_PARSED_CRON_VALUE, ...weeklySchedule, weekday: weeklySchedule.dayOfWeek },
    };
  }

  const monthlySchedule = readMonthlySchedule(normalizedFields);
  if (monthlySchedule) {
    return {
      expression: unixExpression,
      mode: 'monthly',
      value: { ...DEFAULT_PARSED_CRON_VALUE, ...monthlySchedule },
    };
  }

  return createAdvancedParsedCronExpression(unixExpression);
}

export function validateCronExpression(expression: string): CronValidationResult {
  const fields = splitCronFields(expression);

  if (fields.length === 0) {
    return {
      valid: false,
      messageKey: 'scheduledTask.cronValidation.required',
    };
  }

  const normalizedFields = toNormalizedCronFields(expression);
  if (!normalizedFields) {
    return {
      valid: false,
      messageKey: 'scheduledTask.cronValidation.fieldCount',
      messageParams: { unixFields: FIELD_COUNT_UNIX, secondsFields: FIELD_COUNT_SECONDS },
    };
  }

  for (const [index, field] of normalizedFields.entries()) {
    const result = validateCronField(field, CRON_FIELD_RULES[index]!);
    if (!result.valid) {
      return result;
    }
  }

  return { valid: true };
}

export function describeCronExpression(
  expression: string,
  translate?: CronDescriptionTranslate,
): CronDescriptionResult | string {
  const validation = validateCronExpression(expression);
  const normalizedExpression = normalizeCronExpression(expression);
  const description = validation.valid
    ? describeNormalizedCronExpression(normalizedExpression)
    : ({
        key: 'scheduledTask.cronDescription.invalid',
        params: { expression: normalizedExpression },
        normalizedExpression,
        valid: false,
      } satisfies CronDescriptionResult);

  return translate ? translate(description.key, description.params) : description;
}

export function previewCronExecutions(expression: string, from = new Date(), count = 4): CronExecutionPreview {
  const validation = validateCronExpression(expression);
  const normalizedExpression = normalizeCronExpression(expression);

  if (!validation.valid) {
    return {
      nextRuns: [],
      normalizedExpression,
      valid: false,
    };
  }

  const normalizedFields = toNormalizedCronFields(normalizedExpression);
  if (!normalizedFields) {
    return {
      nextRuns: [],
      normalizedExpression,
      valid: false,
    };
  }

  const nextRuns = collectNextRuns(buildExpandedCronFieldValues(normalizedFields), from, count);

  return {
    intervalMs: nextRuns.length >= 2 ? nextRuns[1]!.getTime() - nextRuns[0]!.getTime() : undefined,
    nextRuns,
    normalizedExpression,
    valid: true,
  };
}

export function getNextRuns(expression: string, count: number, from = new Date()): Date[] {
  return previewCronExecutions(expression, from, count).nextRuns;
}

function formatCronDateTime(date: Date, locale?: string, timezone?: string): string {
  const parts = new Intl.DateTimeFormat(locale, {
    day: '2-digit',
    hour: '2-digit',
    hourCycle: 'h23',
    minute: '2-digit',
    month: '2-digit',
    timeZone: timezone,
    year: 'numeric',
  }).formatToParts(date);

  const valueByType = Object.fromEntries(parts.map((part) => [part.type, part.value]));
  return `${valueByType.year}-${valueByType.month}-${valueByType.day} ${valueByType.hour}:${valueByType.minute}`;
}
