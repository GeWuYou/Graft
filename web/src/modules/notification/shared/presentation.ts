// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { ComposerTranslation } from 'vue-i18n';

import type { NotificationItem } from '../types/notification';

export function notificationSeverityTheme(severity: NotificationItem['severity']) {
  switch (severity) {
    case 'critical':
    case 'error':
      return 'danger';
    case 'warning':
      return 'warning';
    case 'info':
      return 'primary';
    default:
      return 'default';
  }
}

export function notificationStatusTheme(status: NotificationItem['status']) {
  return status === 'unread' ? 'primary' : 'default';
}

export function notificationSourceLabel(sourceModule: string, t: ComposerTranslation) {
  const normalized = sourceModule.trim();
  if (!normalized) {
    return t('notification.values.unknownSource');
  }

  const key = `notification.source.${normalized}`;
  const translated = t(key);
  return translated === key ? normalized : translated;
}

export function notificationTitle(item: NotificationItem, t: ComposerTranslation) {
  if (item.title_key) {
    const translated = t(item.title_key);
    if (translated !== item.title_key) {
      return translated;
    }
  }
  return item.title;
}

export function notificationMessage(item: NotificationItem, t: ComposerTranslation) {
  if (item.message_key) {
    const translated = t(item.message_key);
    if (translated !== item.message_key) {
      return translated;
    }
  }
  return item.message;
}
