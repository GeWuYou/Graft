import type { ComposerTranslation } from 'vue-i18n';

import { PERMISSION_COPY_BY_CODE } from '../contract/permission-copy';
import type { PermissionListItem } from '../types/permission';

type PermissionLocaleKeyFields = {
  description_key?: string | null;
  display_key?: string | null;
};

function isLikelyLocaleKey(value?: string | null) {
  if (!value) return false;
  return /^[a-z][A-Za-z0-9]*(?:[.-][A-Za-z0-9_]+)+$/.test(value.trim());
}

function containsCjk(value?: string | null) {
  return Boolean(value && /[\u3400-\u9fff]/u.test(value));
}

function localizedMessage(
  t: ComposerTranslation,
  messageKey: string,
  fallback: string | null | undefined,
  locale: string,
) {
  const translated = t(messageKey);
  if (translated !== messageKey) {
    return translated;
  }

  if (isLikelyLocaleKey(fallback)) {
    return '';
  }

  return locale === 'zh-CN' || !containsCjk(fallback) ? fallback?.trim() || '' : '';
}

export function localizedPermissionDisplay(
  t: ComposerTranslation,
  permission: Pick<PermissionListItem, 'code' | 'display'> & PermissionLocaleKeyFields,
  locale = 'zh-CN',
) {
  if (permission.display_key) {
    const localized = localizedMessage(t, permission.display_key, permission.display, locale);
    if (localized) {
      return localized;
    }
  }

  const copyEntry = PERMISSION_COPY_BY_CODE[permission.code];
  if (!copyEntry) {
    return isLikelyLocaleKey(permission.display) || (locale !== 'zh-CN' && containsCjk(permission.display))
      ? permission.code
      : permission.display;
  }

  return localizedMessage(t, copyEntry.displayKey, permission.display, locale) || permission.code;
}

export function localizedPermissionDescription(
  t: ComposerTranslation,
  permission: Pick<PermissionListItem, 'code' | 'description'> & PermissionLocaleKeyFields,
  emptyDescriptionKey: string,
  locale = 'zh-CN',
) {
  if (permission.description_key) {
    const localized = localizedMessage(t, permission.description_key, permission.description, locale);
    if (localized) {
      return localized;
    }
  }

  const copyEntry = PERMISSION_COPY_BY_CODE[permission.code];
  if (copyEntry) {
    const localized = localizedMessage(t, copyEntry.descriptionKey, permission.description, locale);
    if (localized) {
      return localized;
    }
  }

  if (isLikelyLocaleKey(permission.description) || (locale !== 'zh-CN' && containsCjk(permission.description))) {
    return t(emptyDescriptionKey);
  }

  return locale === 'zh-CN' || !containsCjk(permission.description)
    ? permission.description?.trim() || t(emptyDescriptionKey)
    : t(emptyDescriptionKey);
}
