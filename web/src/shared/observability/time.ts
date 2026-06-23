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

export function formatLocaleDateOnly(value?: string | null, locale?: string | Ref<string | undefined> | null) {
  const formatted = formatLocaleDateTime(value, locale, DATE_ONLY_FORMAT_OPTIONS);
  return formatted === '-' ? '' : formatted;
}
