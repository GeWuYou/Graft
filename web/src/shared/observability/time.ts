import type { Ref } from 'vue';

import { getDefaultLocale, normalizeLocale } from '@/contracts/i18n/locales';

const DEFAULT_DATE_TIME_FORMAT_OPTIONS = {
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: 'numeric',
  minute: '2-digit',
  second: '2-digit',
} satisfies Intl.DateTimeFormatOptions;

export const MEDIUM_DATE_TIME_FORMAT_OPTIONS = {
  dateStyle: 'medium',
  timeStyle: 'short',
} satisfies Intl.DateTimeFormatOptions;

export const MEDIUM_DATE_TIME_WITH_SECONDS_FORMAT_OPTIONS = {
  dateStyle: 'medium',
  timeStyle: 'medium',
} satisfies Intl.DateTimeFormatOptions;

const MONTH_DAY_TIME_FORMAT_OPTIONS = {
  month: 'short',
  day: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hour12: false,
} satisfies Intl.DateTimeFormatOptions;

const YEAR_MONTH_DAY_TIME_FORMAT_OPTIONS = {
  year: 'numeric',
  month: 'numeric',
  day: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hour12: false,
} satisfies Intl.DateTimeFormatOptions;

const DATE_ONLY_FORMAT_OPTIONS = {
  year: 'numeric',
  month: 'numeric',
  day: 'numeric',
} satisfies Intl.DateTimeFormatOptions;

const TIME_ONLY_FORMAT_OPTIONS = {
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hour12: false,
} satisfies Intl.DateTimeFormatOptions;

function resolveLocale(locale?: string | Ref<string | undefined> | null) {
  const fallbackLocale = getDefaultLocale();

  if (!locale) {
    return fallbackLocale;
  }

  if (typeof locale === 'string') {
    return normalizeLocale(locale) ?? fallbackLocale;
  }

  return normalizeLocale(locale.value) ?? fallbackLocale;
}

export function formatLocaleDateTime(
  value?: string | null,
  locale?: string | Ref<string | undefined> | null,
  options: Intl.DateTimeFormatOptions = DEFAULT_DATE_TIME_FORMAT_OPTIONS,
) {
  if (!value) {
    return '-';
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat(resolveLocale(locale), options).format(date);
}

export function formatLocaleTimeOnly(value?: string | null, locale?: string | Ref<string | undefined> | null) {
  return formatLocaleDateTime(value, locale, TIME_ONLY_FORMAT_OPTIONS);
}

/**
 * 格式化仅包含日期的本地化时间字符串。
 *
 * @returns 格式化后的日期字符串；当输入为空时返回空字符串。
 */
export function formatLocaleDateOnly(value?: string | null, locale?: string | Ref<string | undefined> | null) {
  const formatted = formatLocaleDateTime(value, locale, DATE_ONLY_FORMAT_OPTIONS);
  return formatted === '-' ? '' : formatted;
}

/**
 * 格式化日志查看器中的时间戳。
 *
 * @param value - 要格式化的时间字符串
 * @param locale - 用于格式化的区域设置
 * @returns 格式化后的时间字符串；当 `value` 为空时返回空字符串，当 `value` 不是有效日期时返回原始值
 */
export function formatLogViewerTimestamp(value?: string | null, locale?: string | Ref<string | undefined> | null) {
  if (!value) {
    return '';
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  const currentLocale = resolveLocale(locale);
  const now = new Date();

  if (isSameLocalDay(date, now)) {
    return new Intl.DateTimeFormat(currentLocale, TIME_ONLY_FORMAT_OPTIONS).format(date);
  }

  const yesterday = new Date(now);
  yesterday.setDate(now.getDate() - 1);
  if (isSameLocalDay(date, yesterday)) {
    return new Intl.DateTimeFormat(currentLocale, TIME_ONLY_FORMAT_OPTIONS).format(date);
  }

  if (date.getFullYear() === now.getFullYear()) {
    return new Intl.DateTimeFormat(currentLocale, MONTH_DAY_TIME_FORMAT_OPTIONS).format(date);
  }

  return new Intl.DateTimeFormat(currentLocale, YEAR_MONTH_DAY_TIME_FORMAT_OPTIONS).format(date);
}

/**
 * 判断两个日期是否属于同一天。
 *
 * @param left - 左侧日期
 * @param right - 右侧日期
 * @returns `true` if `left` 和 `right` 在本地时区的年、月、日相同，`false` otherwise.
 */
function isSameLocalDay(left: Date, right: Date) {
  return (
    left.getFullYear() === right.getFullYear() &&
    left.getMonth() === right.getMonth() &&
    left.getDate() === right.getDate()
  );
}
