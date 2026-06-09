// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { t } from '@/locales';

const DEFAULT_DASHBOARD_TEXT = '-';

function hasText(value: string | undefined): value is string {
  return Boolean(value?.trim());
}

export function hasDashboardTranslation(key?: string) {
  if (!hasText(key)) {
    return false;
  }

  const translated = t(key);
  return translated !== key;
}

export function resolveDashboardText(key?: string, fallback?: string, defaultText = DEFAULT_DASHBOARD_TEXT) {
  if (hasText(key)) {
    const translated = t(key);
    if (translated !== key) {
      return translated;
    }
  }

  if (hasText(fallback)) {
    return fallback;
  }

  return defaultText;
}

export function resolveDashboardRelatedText(
  baseKey: string | undefined,
  relatedName: string,
  fallback?: string,
  defaultText = '',
) {
  if (hasText(baseKey)) {
    const segments = baseKey.split('.');
    segments[segments.length - 1] = relatedName;
    const relatedKey = segments.join('.');
    if (hasDashboardTranslation(relatedKey)) {
      return resolveDashboardText(relatedKey, fallback, defaultText);
    }
  }

  return resolveDashboardText(undefined, fallback, defaultText);
}
