import { t } from '@/locales';

const DEFAULT_DASHBOARD_TEXT = '-';

function hasText(value: string | undefined): value is string {
  return Boolean(value?.trim());
}

export function hasDashboardTranslation(key?: string) {
  return hasText(key) && t(key) !== key;
}

/** 优先使用稳定 i18n key，同时保留服务端展示值作为受控回退。 */
export function resolveDashboardText(key?: string, fallback?: string, defaultText = DEFAULT_DASHBOARD_TEXT) {
  if (hasText(key) && hasDashboardTranslation(key)) {
    return t(key);
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
