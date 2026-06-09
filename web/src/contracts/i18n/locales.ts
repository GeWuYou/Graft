// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import en_US from 'tdesign-vue-next/es/locale/en_US';
import zh_CN from 'tdesign-vue-next/es/locale/zh_CN';

export const LOCALE = {
  ZH_CN: 'zh-CN',
  EN_US: 'en-US',
} as const;

export const supportedLocales = [LOCALE.ZH_CN, LOCALE.EN_US] as const;
export type SupportedLocale = (typeof supportedLocales)[number];

export type LocalizedTitle = Record<SupportedLocale, string>;

const legacyLocaleAliases: Record<string, SupportedLocale> = {
  en_US: LOCALE.EN_US,
  'en-US': LOCALE.EN_US,
  zh_CN: LOCALE.ZH_CN,
  'zh-CN': LOCALE.ZH_CN,
};

const tdesignLocaleMap: Record<SupportedLocale, typeof zh_CN | typeof en_US> = {
  [LOCALE.ZH_CN]: zh_CN,
  [LOCALE.EN_US]: en_US,
};

export function normalizeLocale(input: string | null | undefined): SupportedLocale | null {
  if (!input) {
    return null;
  }

  return legacyLocaleAliases[input] ?? null;
}

export function getDefaultLocale(): SupportedLocale {
  return LOCALE.ZH_CN;
}

export function toTDesignLocale(locale: SupportedLocale) {
  return tdesignLocaleMap[locale];
}
