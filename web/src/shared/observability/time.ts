// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { Ref } from 'vue';

function resolveLocale(locale?: string | Ref<string | undefined> | null) {
  if (!locale) {
    return undefined;
  }

  if (typeof locale === 'string') {
    return locale || undefined;
  }

  return locale.value || undefined;
}

export function formatLocaleDateTime(
  value?: string | null,
  locale?: string | Ref<string | undefined> | null,
  options: Intl.DateTimeFormatOptions = {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: 'numeric',
    minute: '2-digit',
    second: '2-digit',
  },
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
