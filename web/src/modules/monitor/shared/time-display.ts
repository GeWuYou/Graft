import type { Ref } from 'vue';

import { formatLocaleDateOnly, formatLocaleDateTime, formatLocaleTimeOnly } from '@/shared/observability';

export function formatTimeOnly(value?: string | null, locale?: string | Ref<string | undefined> | null) {
  if (!value || Number.isNaN(new Date(value).getTime())) {
    return '--';
  }

  const formatted = formatLocaleTimeOnly(value, locale);
  return formatted === '-' ? '--' : formatted;
}

export function formatDateOnly(value?: string | null, locale?: string | Ref<string | undefined> | null) {
  if (!value || Number.isNaN(new Date(value).getTime())) {
    return '';
  }

  return formatLocaleDateOnly(value, locale);
}

export function formatChartTimeOnly(value?: string | null, locale?: string | Ref<string | undefined> | null) {
  if (!value || Number.isNaN(new Date(value).getTime())) {
    return value ?? '';
  }

  return formatLocaleDateTime(value, locale, {
    hour: '2-digit',
    minute: '2-digit',
  });
}
