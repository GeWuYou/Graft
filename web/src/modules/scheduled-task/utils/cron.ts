// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import 'cronstrue/locales/zh_CN';

import { CronExpressionParser } from 'cron-parser';
import cronstrue from 'cronstrue';

export type CronValidationResult = {
  valid: boolean;
  messageKey?: CronValidationMessageKey;
  messageParams?: Record<string, string | number>;
};

export type CronValidationMessageKey =
  | 'scheduledTask.cronValidation.required'
  | 'scheduledTask.cronValidation.fieldCount'
  | 'scheduledTask.cronValidation.stepRange'
  | 'scheduledTask.cronValidation.fieldRange';

export type CronDescriptionKey =
  | 'scheduledTask.cronDescription.everyMinute'
  | 'scheduledTask.cronDescription.everyNMinutes'
  | 'scheduledTask.cronDescription.hourly'
  | 'scheduledTask.cronDescription.daily'
  | 'scheduledTask.cronDescription.weekly'
  | 'scheduledTask.cronDescription.monthly'
  | 'scheduledTask.cronDescription.advanced'
  | 'scheduledTask.cronDescription.custom'
  | 'scheduledTask.cronDescription.invalid';

export type CronDescriptionResult = {
  key: CronDescriptionKey;
  params?: Record<string, string | number>;
  normalizedExpression: string;
  valid: boolean;
};

export type CronDescriptionTranslate = (key: CronDescriptionKey, params?: Record<string, string | number>) => string;

export type CronExecutionPreview = {
  intervalMs?: number;
  nextRuns: Date[];
  normalizedExpression: string;
  valid: boolean;
};

export type CronMode = 'intervalMinutes' | 'hourly' | 'daily' | 'weekly' | 'monthly' | 'advanced';

export type CronScheduleValue = {
  dayOfMonth: number;
  hour: number;
  intervalMinutes: number;
  minute: number;
  weekday: number;
};

export type ParsedCronExpression = {
  expression: string;
  mode: CronMode;
  value: CronScheduleValue;
};

const FIELD_COUNT_UNIX = 5;
const FIELD_COUNT_SECONDS = 6;

export type CronNextRunFormatOptions = {
  locale?: string;
  now?: Date;
};

export type CronDescriptionFormatOptions = {
  advancedExpressionText?: string;
  translate?: CronDescriptionTranslate;
};

type SimpleDailyCronTime = {
  hour: number;
  minute: number;
};

type CronFieldRule = {
  name: string;
  min: number;
  max: number;
  allowStep: boolean;
};

const CRON_FIELD_RULES: CronFieldRule[] = [
  { name: 'seconds', min: 0, max: 59, allowStep: true },
  { name: 'minutes', min: 0, max: 59, allowStep: true },
  { name: 'hours', min: 0, max: 23, allowStep: false },
  { name: 'day-of-month', min: 1, max: 31, allowStep: false },
  { name: 'month', min: 1, max: 12, allowStep: false },
  { name: 'day-of-week', min: 0, max: 7, allowStep: false },
];

export function normalizeCronExpression(expression: string): string {
  const normalized = splitCronFields(expression).join(' ');
  const fields = splitCronFields(normalized);

  if (fields.length === FIELD_COUNT_UNIX) {
    return ['0', ...fields].join(' ');
  }

  return normalized;
}

export function formatCronExpression(expression: string): string {
  return splitCronFields(expression).join(' ');
}

export function getNextRunText(expression: string, timezone?: string, options: CronNextRunFormatOptions = {}): string {
  const normalizedExpression = formatCronExpression(expression);
  if (!isSupportedCronFieldCount(normalizedExpression)) {
    return '';
  }

  try {
    const interval = CronExpressionParser.parse(normalizedExpression, {
      currentDate: options.now ?? new Date(),
      tz: timezone,
    });
    const nextRun = interval.next().toDate();
    return formatCronDateTime(nextRun, options.locale, timezone);
  } catch {
    return '';
  }
}

export function getCronDescription(
  expression: string,
  locale?: string,
  options: CronDescriptionFormatOptions = {},
): string {
  const normalizedExpression = formatCronExpression(expression);
  const cronstrueLocale = toCronstrueLocale(locale);
  if (!isSupportedCronFieldCount(normalizedExpression)) {
    return (
      options.advancedExpressionText ||
      options.translate?.('scheduledTask.cronDescription.advanced') ||
      'scheduledTask.cronDescription.advanced'
    );
  }

  try {
    const simpleDescription = describeNormalizedCronExpression(normalizeCronExpression(normalizedExpression));
    if (simpleDescription.valid && simpleDescription.key !== 'scheduledTask.cronDescription.custom') {
      return options.translate?.(simpleDescription.key, simpleDescription.params) ?? simpleDescription.key;
    }

    const description = cronstrue.toString(normalizedExpression, {
      locale: cronstrueLocale,
      throwExceptionOnParseError: true,
      use24HourTimeFormat: true,
      verbose: true,
    });
    return polishCronDescription(description, normalizedExpression, cronstrueLocale);
  } catch {
    return (
      options.advancedExpressionText ||
      options.translate?.('scheduledTask.cronDescription.advanced') ||
      'scheduledTask.cronDescription.advanced'
    );
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
  const fields = splitCronFields(unixExpression);
  const defaultValue: CronScheduleValue = {
    dayOfMonth: 1,
    hour: 17,
    intervalMinutes: 5,
    minute: 0,
    weekday: 1,
  };

  if (fields.length !== FIELD_COUNT_UNIX || !validateCronExpression(unixExpression).valid) {
    return {
      expression: unixExpression,
      mode: 'advanced',
      value: defaultValue,
    };
  }

  const [minute, hour, dayOfMonth, month, dayOfWeek] = fields;
  const minuteInterval = parseStepValue(minute);

  if (minuteInterval !== null && hour === '*' && dayOfMonth === '*' && month === '*' && dayOfWeek === '*') {
    return {
      expression: unixExpression,
      mode: 'intervalMinutes',
      value: { ...defaultValue, intervalMinutes: minuteInterval },
    };
  }

  if (isCronNumberInRange(minute, 0, 59) && hour === '*' && dayOfMonth === '*' && month === '*' && dayOfWeek === '*') {
    return {
      expression: unixExpression,
      mode: 'hourly',
      value: { ...defaultValue, minute: Number(minute) },
    };
  }

  if (isCronNumberInRange(minute, 0, 59) && isCronNumberInRange(hour, 0, 23) && dayOfMonth === '*' && month === '*') {
    if (dayOfWeek === '*') {
      return {
        expression: unixExpression,
        mode: 'daily',
        value: { ...defaultValue, hour: Number(hour), minute: Number(minute) },
      };
    }

    if (isCronNumberInRange(dayOfWeek, 0, 7)) {
      return {
        expression: unixExpression,
        mode: 'weekly',
        value: {
          ...defaultValue,
          hour: Number(hour),
          minute: Number(minute),
          weekday: Number(dayOfWeek) === 7 ? 0 : Number(dayOfWeek),
        },
      };
    }
  }

  if (
    isCronNumberInRange(minute, 0, 59) &&
    isCronNumberInRange(hour, 0, 23) &&
    isCronNumberInRange(dayOfMonth, 1, 31) &&
    month === '*' &&
    dayOfWeek === '*'
  ) {
    return {
      expression: unixExpression,
      mode: 'monthly',
      value: {
        ...defaultValue,
        dayOfMonth: Number(dayOfMonth),
        hour: Number(hour),
        minute: Number(minute),
      },
    };
  }

  return {
    expression: unixExpression,
    mode: 'advanced',
    value: defaultValue,
  };
}

export function validateCronExpression(expression: string): CronValidationResult {
  const fields = splitCronFields(expression);

  if (fields.length === 0) {
    return {
      valid: false,
      messageKey: 'scheduledTask.cronValidation.required',
    };
  }

  if (fields.length !== FIELD_COUNT_UNIX && fields.length !== FIELD_COUNT_SECONDS) {
    return {
      valid: false,
      messageKey: 'scheduledTask.cronValidation.fieldCount',
      messageParams: { unixFields: FIELD_COUNT_UNIX, secondsFields: FIELD_COUNT_SECONDS },
    };
  }

  const normalizedFields = normalizeCronExpression(expression).split(' ');

  for (const [index, field] of normalizedFields.entries()) {
    const rule = CRON_FIELD_RULES[index];
    const result = validateCronField(field, rule);
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

  if (translate) {
    return translate(description.key, description.params);
  }

  return description;
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

  const [second, minute, hour, dayOfMonth, month, dayOfWeek] = normalizedExpression.split(' ');
  const fieldValues = {
    seconds: expandCronField(second, 0, 59),
    minutes: expandCronField(minute, 0, 59),
    hours: expandCronField(hour, 0, 23),
    dayOfMonths: expandCronField(dayOfMonth, 1, 31),
    months: expandCronField(month, 1, 12),
    dayOfWeeks: expandCronField(dayOfWeek, 0, 7).map((value) => (value === 7 ? 0 : value)),
  };

  const nextRuns: Date[] = [];
  const fromTime = from.getTime();
  const dayCursor = new Date(from.getFullYear(), from.getMonth(), from.getDate());
  const maxLookaheadDays = 366;

  for (let dayOffset = 0; dayOffset <= maxLookaheadDays && nextRuns.length < count; dayOffset += 1) {
    const candidateDay = new Date(dayCursor);
    candidateDay.setDate(dayCursor.getDate() + dayOffset);

    if (!fieldValues.months.includes(candidateDay.getMonth() + 1)) {
      continue;
    }

    if (!fieldValues.dayOfMonths.includes(candidateDay.getDate())) {
      continue;
    }

    if (!fieldValues.dayOfWeeks.includes(candidateDay.getDay())) {
      continue;
    }

    for (const candidateHour of fieldValues.hours) {
      for (const candidateMinute of fieldValues.minutes) {
        for (const candidateSecond of fieldValues.seconds) {
          const candidate = new Date(
            candidateDay.getFullYear(),
            candidateDay.getMonth(),
            candidateDay.getDate(),
            candidateHour,
            candidateMinute,
            candidateSecond,
            0,
          );
          if (candidate.getTime() <= fromTime) {
            continue;
          }

          nextRuns.push(candidate);
          if (nextRuns.length >= count) {
            break;
          }
        }
        if (nextRuns.length >= count) {
          break;
        }
      }
      if (nextRuns.length >= count) {
        break;
      }
    }
  }

  return {
    intervalMs: nextRuns.length >= 2 ? nextRuns[1].getTime() - nextRuns[0].getTime() : undefined,
    nextRuns,
    normalizedExpression,
    valid: true,
  };
}

export function getNextRuns(expression: string, count: number, from = new Date()): Date[] {
  return previewCronExecutions(expression, from, count).nextRuns;
}

function describeNormalizedCronExpression(normalizedExpression: string): CronDescriptionResult {
  const fields = normalizedExpression.split(' ');
  const [second, minute, hour, dayOfMonth, month, dayOfWeek] = fields;

  if (second === '0' && minute === '*' && hour === '*' && dayOfMonth === '*' && month === '*' && dayOfWeek === '*') {
    return {
      key: 'scheduledTask.cronDescription.everyMinute',
      normalizedExpression,
      valid: true,
    };
  }

  const minuteInterval = parseStepValue(minute);
  if (
    second === '0' &&
    minuteInterval !== null &&
    hour === '*' &&
    dayOfMonth === '*' &&
    month === '*' &&
    dayOfWeek === '*'
  ) {
    return {
      key: 'scheduledTask.cronDescription.everyNMinutes',
      params: { interval: minuteInterval },
      normalizedExpression,
      valid: true,
    };
  }

  if (second === '0' && minute === '0' && hour === '*' && dayOfMonth === '*' && month === '*' && dayOfWeek === '*') {
    return {
      key: 'scheduledTask.cronDescription.hourly',
      normalizedExpression,
      valid: true,
    };
  }

  if (
    second === '0' &&
    isCronNumberInRange(minute, 0, 59) &&
    isCronNumberInRange(hour, 0, 23) &&
    dayOfMonth === '*' &&
    month === '*'
  ) {
    if (dayOfWeek === '*') {
      return {
        key: 'scheduledTask.cronDescription.daily',
        params: {
          hour: Number(hour),
          minute: Number(minute),
          time: formatCronClockTime(Number(hour), Number(minute)),
        },
        normalizedExpression,
        valid: true,
      };
    }

    if (isCronNumberInRange(dayOfWeek, 0, 7)) {
      return {
        key: 'scheduledTask.cronDescription.weekly',
        params: {
          hour: Number(hour),
          minute: Number(minute),
          time: formatCronClockTime(Number(hour), Number(minute)),
          dayOfWeek: Number(dayOfWeek),
        },
        normalizedExpression,
        valid: true,
      };
    }
  }

  if (
    second === '0' &&
    minute === '0' &&
    isCronNumberInRange(hour, 0, 23) &&
    isCronNumberInRange(dayOfMonth, 1, 31) &&
    month === '*' &&
    dayOfWeek === '*'
  ) {
    return {
      key: 'scheduledTask.cronDescription.monthly',
      params: { hour: Number(hour), dayOfMonth: Number(dayOfMonth) },
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

function splitCronFields(expression: string): string[] {
  return expression.trim().split(/\s+/).filter(Boolean);
}

function isSupportedCronFieldCount(expression: string): boolean {
  const fields = splitCronFields(expression);
  return fields.length === FIELD_COUNT_UNIX || fields.length === FIELD_COUNT_SECONDS;
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

function toCronstrueLocale(locale?: string): string {
  if (locale?.toLowerCase().startsWith('zh')) {
    return 'zh_CN';
  }

  return 'en';
}

function polishCronDescription(description: string, expression: string, locale: string): string {
  if (locale !== 'zh_CN') {
    return description;
  }

  const dailyTime = parseSimpleDailyCronTime(expression);
  if (dailyTime && /每天/.test(description) && /在\s*\d{1,2}:\d{2}/.test(description)) {
    return description
      .replace(/\d{1,2}:\d{2}/, formatCronClockTime(dailyTime.hour, dailyTime.minute))
      .replace(/,\s*/g, '，');
  }

  return description.replace(/,\s*/g, '，');
}

function parseSimpleDailyCronTime(expression: string): SimpleDailyCronTime | null {
  const fields = splitCronFields(expression);
  const normalizedFields = fields.length === FIELD_COUNT_UNIX ? ['0', ...fields] : fields;
  if (normalizedFields.length !== FIELD_COUNT_SECONDS) {
    return null;
  }

  const [second, minute, hour, dayOfMonth, month, dayOfWeek] = normalizedFields;
  if (
    second !== '0' ||
    !isCronNumberInRange(minute, 0, 59) ||
    !isCronNumberInRange(hour, 0, 23) ||
    dayOfMonth !== '*' ||
    month !== '*' ||
    dayOfWeek !== '*'
  ) {
    return null;
  }

  return {
    hour: Number(hour),
    minute: Number(minute),
  };
}

function formatCronClockTime(hour: number, minute: number): string {
  return `${hour.toString().padStart(2, '0')}:${minute.toString().padStart(2, '0')}`;
}

function validateCronField(field: string, rule: CronFieldRule): CronValidationResult {
  if (field === '*') {
    return { valid: true };
  }

  if (rule.allowStep && field.startsWith('*/')) {
    const step = field.slice(2);
    if (isCronNumberInRange(step, 1, rule.max)) {
      return { valid: true };
    }

    return {
      valid: false,
      messageKey: 'scheduledTask.cronValidation.stepRange',
      messageParams: { field: rule.name, min: 1, max: rule.max },
    };
  }

  if (isCronNumberInRange(field, rule.min, rule.max)) {
    return { valid: true };
  }

  return {
    valid: false,
    messageKey: 'scheduledTask.cronValidation.fieldRange',
    messageParams: { field: rule.name, min: rule.min, max: rule.max },
  };
}

function expandCronField(field: string, min: number, max: number): number[] {
  if (field === '*') {
    return range(min, max);
  }

  const stepValue = parseStepValue(field);
  if (stepValue !== null) {
    return range(min, max).filter((value) => value % stepValue === 0);
  }

  return [Number(field)];
}

function parseStepValue(field: string): number | null {
  if (!field.startsWith('*/')) {
    return null;
  }

  const step = field.slice(2);
  return isPositiveInteger(step) ? Number(step) : null;
}

function isCronNumberInRange(value: string, min: number, max: number): boolean {
  if (!/^\d+$/.test(value)) {
    return false;
  }

  const numericValue = Number(value);
  return numericValue >= min && numericValue <= max;
}

function range(min: number, max: number): number[] {
  return Array.from({ length: max - min + 1 }, (_item, index) => min + index);
}

function isPositiveInteger(value: string): boolean {
  return /^[1-9]\d*$/.test(value);
}

function clampInteger(value: number | string, min: number, max: number): number {
  const numericValue = Number(value);
  if (!Number.isFinite(numericValue)) {
    return min;
  }

  return Math.min(Math.max(Math.trunc(numericValue), min), max);
}
