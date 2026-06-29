import type { CronScheduleValue, CronValidationResult, ParsedCronExpression } from './cron-types';

export type CronFieldRule = {
  name: string;
  min: number;
  max: number;
  allowStep: boolean;
};

export type NormalizedCronFields = [string, string, string, string, string, string];

export type DailySchedule = {
  hour: number;
  minute: number;
};

export type WeeklySchedule = DailySchedule & {
  dayOfWeek: number;
};

export type MonthlySchedule = DailySchedule & {
  dayOfMonth: number;
};

export type ExpandedCronFieldValues = {
  seconds: number[];
  minutes: number[];
  hours: number[];
  dayOfMonths: number[];
  months: number[];
  dayOfWeeks: number[];
};

type CronFieldName = 'second' | 'minute' | 'hour' | 'dayOfMonth' | 'month' | 'dayOfWeek';

type CronFieldNumberPattern = {
  min: number;
  max: number;
  normalize?: (value: number) => number;
};

type CronFieldPattern = string | CronFieldNumberPattern;
type CronFieldPatternMap = Partial<Record<CronFieldName, CronFieldPattern>>;
type ParsedCronFieldNumbers = Partial<Record<CronFieldName, number>>;

export const FIELD_COUNT_UNIX = 5;
export const FIELD_COUNT_SECONDS = 6;

const CRON_FIELD_NAMES = [
  'second',
  'minute',
  'hour',
  'dayOfMonth',
  'month',
  'dayOfWeek',
] as const satisfies ReadonlyArray<CronFieldName>;
const CRON_MINUTE_PATTERN = { min: 0, max: 59 } as const satisfies CronFieldNumberPattern;
const CRON_HOUR_PATTERN = { min: 0, max: 23 } as const satisfies CronFieldNumberPattern;
const CRON_DAY_OF_MONTH_PATTERN = { min: 1, max: 31 } as const satisfies CronFieldNumberPattern;
const CRON_DAY_OF_WEEK_PATTERN = {
  min: 0,
  max: 7,
  normalize: normalizeCronWeekday,
} as const satisfies CronFieldNumberPattern;
const INTERVAL_MINUTE_PATTERN = {
  second: '0',
  hour: '*',
  dayOfMonth: '*',
  month: '*',
  dayOfWeek: '*',
} as const satisfies CronFieldPatternMap;
const HOURLY_MINUTE_PATTERN = {
  second: '0',
  minute: CRON_MINUTE_PATTERN,
  hour: '*',
  dayOfMonth: '*',
  month: '*',
  dayOfWeek: '*',
} as const satisfies CronFieldPatternMap;
const DAILY_SCHEDULE_PATTERN = {
  second: '0',
  minute: CRON_MINUTE_PATTERN,
  hour: CRON_HOUR_PATTERN,
  dayOfMonth: '*',
  month: '*',
  dayOfWeek: '*',
} as const satisfies CronFieldPatternMap;
const WEEKLY_SCHEDULE_PATTERN = {
  second: '0',
  minute: CRON_MINUTE_PATTERN,
  hour: CRON_HOUR_PATTERN,
  dayOfMonth: '*',
  month: '*',
  dayOfWeek: CRON_DAY_OF_WEEK_PATTERN,
} as const satisfies CronFieldPatternMap;
const MONTHLY_SCHEDULE_PATTERN = {
  second: '0',
  minute: CRON_MINUTE_PATTERN,
  hour: CRON_HOUR_PATTERN,
  dayOfMonth: CRON_DAY_OF_MONTH_PATTERN,
  month: '*',
  dayOfWeek: '*',
} as const satisfies CronFieldPatternMap;

export const CRON_FIELD_RULES: CronFieldRule[] = [
  { name: 'seconds', min: 0, max: 59, allowStep: true },
  { name: 'minutes', min: 0, max: 59, allowStep: true },
  { name: 'hours', min: 0, max: 23, allowStep: false },
  { name: 'day-of-month', min: 1, max: 31, allowStep: false },
  { name: 'month', min: 1, max: 12, allowStep: false },
  { name: 'day-of-week', min: 0, max: 7, allowStep: false },
];

export const DEFAULT_PARSED_CRON_VALUE: CronScheduleValue = {
  dayOfMonth: 1,
  hour: 17,
  intervalMinutes: 5,
  minute: 0,
  weekday: 1,
};

export function splitCronFields(expression: string): string[] {
  return expression.trim().split(/\s+/).filter(Boolean);
}

export function toNormalizedCronFields(expression: string): NormalizedCronFields | null {
  const fields = splitCronFields(expression);

  if (fields.length === FIELD_COUNT_UNIX) {
    return ['0', ...fields] as NormalizedCronFields;
  }

  if (fields.length === FIELD_COUNT_SECONDS) {
    return fields as NormalizedCronFields;
  }

  return null;
}

export function isSupportedCronFieldCount(expression: string): boolean {
  return toNormalizedCronFields(expression) !== null;
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

export function validateCronField(field: string, rule: CronFieldRule): CronValidationResult {
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

export function readIntervalMinuteMode([second, minute, hour, dayOfMonth, month, dayOfWeek]: NormalizedCronFields) {
  if (!readCronPatternValues([second, minute, hour, dayOfMonth, month, dayOfWeek], INTERVAL_MINUTE_PATTERN)) {
    return null;
  }

  const intervalMinutes = parseStepValue(minute);
  if (intervalMinutes === null) {
    return null;
  }

  return intervalMinutes;
}

export function readHourlyMinute(fields: NormalizedCronFields) {
  return readCronPatternResult(fields, HOURLY_MINUTE_PATTERN, ({ minute }) => minute ?? null);
}

export function readDailySchedule(fields: NormalizedCronFields): DailySchedule | null {
  return readCronPatternResult(fields, DAILY_SCHEDULE_PATTERN, ({ hour, minute }) =>
    hour === undefined || minute === undefined ? null : { hour, minute },
  );
}

export function readWeeklySchedule(fields: NormalizedCronFields): WeeklySchedule | null {
  return readCronPatternResult(fields, WEEKLY_SCHEDULE_PATTERN, ({ dayOfWeek, hour, minute }) =>
    dayOfWeek === undefined || hour === undefined || minute === undefined ? null : { dayOfWeek, hour, minute },
  );
}

export function readMonthlySchedule(fields: NormalizedCronFields): MonthlySchedule | null {
  return readCronPatternResult(fields, MONTHLY_SCHEDULE_PATTERN, ({ dayOfMonth, hour, minute }) =>
    dayOfMonth === undefined || hour === undefined || minute === undefined ? null : { dayOfMonth, hour, minute },
  );
}

export function readMonthlyDescriptionSchedule(fields: NormalizedCronFields): MonthlySchedule | null {
  const monthlySchedule = readMonthlySchedule(fields);

  if (!monthlySchedule || monthlySchedule.minute !== 0) {
    return null;
  }

  return monthlySchedule;
}

export function buildExpandedCronFieldValues([
  second,
  minute,
  hour,
  dayOfMonth,
  month,
  dayOfWeek,
]: NormalizedCronFields) {
  return {
    seconds: expandCronField(second, 0, 59),
    minutes: expandCronField(minute, 0, 59),
    hours: expandCronField(hour, 0, 23),
    dayOfMonths: expandCronField(dayOfMonth, 1, 31),
    months: expandCronField(month, 1, 12),
    dayOfWeeks: expandCronField(dayOfWeek, 0, 7).map(normalizeCronWeekday),
  } satisfies ExpandedCronFieldValues;
}

export function collectNextRuns(fieldValues: ExpandedCronFieldValues, from: Date, count: number): Date[] {
  const nextRuns: Date[] = [];
  const fromTime = from.getTime();
  const dayCursor = new Date(from.getFullYear(), from.getMonth(), from.getDate());
  const maxLookaheadDays = 366;

  for (let dayOffset = 0; dayOffset <= maxLookaheadDays && nextRuns.length < count; dayOffset += 1) {
    const candidateDay = new Date(dayCursor);
    candidateDay.setDate(dayCursor.getDate() + dayOffset);

    if (!matchesCandidateDay(candidateDay, fieldValues)) {
      continue;
    }

    const remaining = count - nextRuns.length;
    const dayRuns = buildDayRunCandidates(candidateDay, fieldValues)
      .filter((candidate) => candidate.getTime() > fromTime)
      .slice(0, remaining);

    nextRuns.push(...dayRuns);
  }

  return nextRuns;
}

export function createAdvancedParsedCronExpression(expression: string): ParsedCronExpression {
  return {
    expression,
    mode: 'advanced',
    value: DEFAULT_PARSED_CRON_VALUE,
  };
}

export function formatCronClockTime(hour: number, minute: number): string {
  return `${hour.toString().padStart(2, '0')}:${minute.toString().padStart(2, '0')}`;
}

export function clampInteger(value: number | string, min: number, max: number): number {
  const numericValue = Number(value);
  if (!Number.isFinite(numericValue)) {
    return min;
  }

  return Math.min(Math.max(Math.trunc(numericValue), min), max);
}

function matchesCandidateDay(candidateDay: Date, fieldValues: ExpandedCronFieldValues): boolean {
  return (
    fieldValues.months.includes(candidateDay.getMonth() + 1) &&
    fieldValues.dayOfMonths.includes(candidateDay.getDate()) &&
    fieldValues.dayOfWeeks.includes(candidateDay.getDay())
  );
}

function buildDayRunCandidates(candidateDay: Date, fieldValues: ExpandedCronFieldValues): Date[] {
  return fieldValues.hours.flatMap((candidateHour) =>
    fieldValues.minutes.flatMap((candidateMinute) =>
      fieldValues.seconds.map(
        (candidateSecond) =>
          new Date(
            candidateDay.getFullYear(),
            candidateDay.getMonth(),
            candidateDay.getDate(),
            candidateHour,
            candidateMinute,
            candidateSecond,
            0,
          ),
      ),
    ),
  );
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

function range(min: number, max: number): number[] {
  return Array.from({ length: max - min + 1 }, (_item, index) => min + index);
}

function isPositiveInteger(value: string): boolean {
  return /^[1-9]\d*$/.test(value);
}

function toNormalizedCronFieldRecord([
  second,
  minute,
  hour,
  dayOfMonth,
  month,
  dayOfWeek,
]: NormalizedCronFields): Record<CronFieldName, string> {
  return {
    second,
    minute,
    hour,
    dayOfMonth,
    month,
    dayOfWeek,
  };
}

function readCronPatternResult<TResult>(
  fields: NormalizedCronFields,
  pattern: CronFieldPatternMap,
  buildResult: (values: ParsedCronFieldNumbers) => TResult | null,
): TResult | null {
  const values = readCronPatternValues(fields, pattern);
  return values ? buildResult(values) : null;
}

function readCronPatternValues(
  fields: NormalizedCronFields,
  pattern: CronFieldPatternMap,
): ParsedCronFieldNumbers | null {
  const fieldRecord = toNormalizedCronFieldRecord(fields);
  const parsedValues: ParsedCronFieldNumbers = {};

  for (const fieldName of CRON_FIELD_NAMES) {
    const expectedValue = pattern[fieldName];

    if (expectedValue === undefined) {
      continue;
    }

    const fieldValue = fieldRecord[fieldName];
    if (typeof expectedValue === 'string') {
      if (fieldValue !== expectedValue) {
        return null;
      }
      continue;
    }

    const numericValue = readCronNumber(fieldValue, expectedValue.min, expectedValue.max);
    if (numericValue === null) {
      return null;
    }

    parsedValues[fieldName] = expectedValue.normalize ? expectedValue.normalize(numericValue) : numericValue;
  }

  return parsedValues;
}

function readCronNumber(value: string, min: number, max: number): number | null {
  if (!isCronNumberInRange(value, min, max)) {
    return null;
  }

  return Number(value);
}

function normalizeCronWeekday(value: number): number {
  return value === 7 ? 0 : value;
}
