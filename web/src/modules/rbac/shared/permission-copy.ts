import type { ComposerTranslation } from 'vue-i18n';

import { PERMISSION_COPY_BY_CODE } from '../contract/permission-copy';
import type { PermissionListItem } from '../types/permission';

type PermissionLocaleKeyFields = {
  description_key?: string | null;
  display_key?: string | null;
};

function localizedMessage(t: ComposerTranslation, messageKey: string, fallback?: string | null) {
  const translated = t(messageKey);
  if (translated !== messageKey) {
    return translated;
  }

  return fallback?.trim() || '';
}

export function localizedPermissionDisplay(
  t: ComposerTranslation,
  permission: Pick<PermissionListItem, 'code' | 'display'> & PermissionLocaleKeyFields,
) {
  if (permission.display_key) {
    const localized = localizedMessage(t, permission.display_key, permission.display);
    if (localized) {
      return localized;
    }
  }

  const copyEntry = PERMISSION_COPY_BY_CODE[permission.code];
  if (!copyEntry) {
    return permission.display;
  }

  return localizedMessage(t, copyEntry.displayKey, permission.display) || permission.display;
}

export function localizedPermissionDescription(
  t: ComposerTranslation,
  permission: Pick<PermissionListItem, 'code' | 'description'> & PermissionLocaleKeyFields,
  emptyDescriptionKey: string,
) {
  if (permission.description_key) {
    const localized = localizedMessage(t, permission.description_key, permission.description);
    if (localized) {
      return localized;
    }
  }

  const copyEntry = PERMISSION_COPY_BY_CODE[permission.code];
  if (copyEntry) {
    const localized = localizedMessage(t, copyEntry.descriptionKey, permission.description);
    if (localized) {
      return localized;
    }
  }

  return permission.description?.trim() || t(emptyDescriptionKey);
}
