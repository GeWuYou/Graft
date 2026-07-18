import type { LocalizedTitle, SupportedLocale } from '@/contracts/i18n/locales';
import { supportedLocales } from '@/contracts/i18n/locales';
import { i18n } from '@/locales';

const ROUTE_TITLE_KEY_PATTERN = /^[a-z][a-zA-Z0-9]*(\.[a-zA-Z0-9_-]+)+$/;

function resolveLocaleMessage(locale: SupportedLocale, titleKey: string): string | undefined {
  const messageTree = i18n.global.getLocaleMessage(locale) as Record<string, unknown>;
  const directMessage = messageTree[titleKey];
  if (typeof directMessage === 'string' && directMessage.length > 0) {
    return directMessage;
  }

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

export function localizeRouteTitleKey(titleKey: string): LocalizedTitle {
  return localizeRouteTitle(titleKey, titleKey);
}

/**
 * 判断标题片段是否符合前端消息 key 的稳定格式。
 */
export function isRouteTitleKey(value: string) {
  return ROUTE_TITLE_KEY_PATTERN.test(value.trim());
}

/**
 * 判断本地化标题是否仍包含未解析的消息 key，包括导航层级组成的复合标题。
 */
export function hasUnresolvedRouteTitleKey(title?: LocalizedTitle) {
  return Object.values(title ?? {}).some((value) =>
    value.split('/').some((segment) => isRouteTitleKey(segment.trim())),
  );
}
