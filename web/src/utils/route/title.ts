// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { LocalizedTitle, SupportedLocale } from '@/contracts/i18n/locales';
import { supportedLocales } from '@/contracts/i18n/locales';
import { i18n } from '@/locales';

function resolveLocaleMessage(locale: SupportedLocale, titleKey: string): string | undefined {
  const messageTree = i18n.global.getLocaleMessage(locale) as Record<string, unknown>;
  const resolved = titleKey.split('.').reduce<unknown>((current, segment) => {
    if (!current || typeof current !== 'object') {
      return undefined;
    }

    return (current as Record<string, unknown>)[segment];
  }, messageTree);

  return typeof resolved === 'string' && resolved.length > 0 ? resolved : undefined;
}

function resolveTitleForLocale(locale: SupportedLocale, fallbackTitle: string, titleKey?: string): string {
  if (titleKey) {
    const translated = resolveLocaleMessage(locale, titleKey);
    if (translated) {
      return translated;
    }
  }

  return fallbackTitle;
}

export function localizeRouteTitle(fallbackTitle: string, titleKey?: string): LocalizedTitle {
  return supportedLocales.reduce<LocalizedTitle>((titles, locale) => {
    titles[locale] = resolveTitleForLocale(locale, fallbackTitle, titleKey);
    return titles;
  }, {} as LocalizedTitle);
}
