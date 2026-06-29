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

export type CronNextRunFormatOptions = {
  locale?: string;
  now?: Date;
};

export type CronDescriptionFormatOptions = {
  advancedExpressionText?: string;
  translate?: CronDescriptionTranslate;
};
