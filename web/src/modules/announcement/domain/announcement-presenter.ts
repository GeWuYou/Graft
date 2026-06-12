// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { ComposerTranslation } from 'vue-i18n';

import { formatCompactDateTime } from '@/shared/components/management';
import { markdownToPlainTextSummary } from '@/shared/components/markdown';

import {
  ANNOUNCEMENT_DELIVERY_MODE_LABEL_KEY,
  ANNOUNCEMENT_LEVEL_LABEL_KEY,
  ANNOUNCEMENT_PINNED_LABEL_KEY,
  ANNOUNCEMENT_STATUS_LABEL_KEY,
} from '../contract/presentation';
import type {
  AnnouncementDeliveryMode,
  AnnouncementItem,
  AnnouncementLevel,
  AnnouncementStatus,
} from '../types/announcement';

export type AnnouncementTagTheme = 'default' | 'primary' | 'success' | 'warning' | 'danger';

export type AnnouncementViewModel = {
  content: string;
  createdAtLabel: string;
  deliveryMode: AnnouncementDeliveryMode;
  deliveryModeLabel: string;
  expireAtLabel: string;
  id: number;
  level: AnnouncementLevel;
  levelLabel: string;
  levelTheme: AnnouncementTagTheme;
  pinned: boolean;
  pinnedLabel: string;
  publishAtLabel: string;
  readAtLabel: string;
  summary: string;
  status: AnnouncementStatus;
  statusLabel: string;
  statusTheme: AnnouncementTagTheme;
  title: string;
  unread: boolean;
  unreadLabel: string;
  updatedAtLabel: string;
};

export function presentAnnouncement(
  item: AnnouncementItem,
  t: ComposerTranslation,
  locale: string,
): AnnouncementViewModel {
  return {
    content: item.content,
    createdAtLabel: formatAnnouncementDate(item.created_at, locale, t),
    deliveryMode: item.delivery_mode,
    deliveryModeLabel: resolveAnnouncementDeliveryModeLabel(item.delivery_mode, t),
    expireAtLabel: formatAnnouncementDate(item.expire_at, locale, t),
    id: item.id,
    level: item.level,
    levelLabel: resolveAnnouncementLevelLabel(item.level, t),
    levelTheme: announcementLevelTheme(item.level),
    pinned: item.pinned,
    pinnedLabel: resolvePinnedLabel(item.pinned, t),
    publishAtLabel: formatAnnouncementDate(item.publish_at, locale, t),
    readAtLabel: formatAnnouncementDate(item.read_at, locale, t),
    summary: markdownToPlainTextSummary(item.content),
    status: item.status,
    statusLabel: resolveAnnouncementStatusLabel(item.status, t),
    statusTheme: announcementStatusTheme(item.status),
    title: item.title,
    unread: resolveAnnouncementUnread(item),
    unreadLabel: resolveAnnouncementUnreadLabel(item, t),
    updatedAtLabel: formatAnnouncementDate(item.updated_at, locale, t),
  };
}

function resolveAnnouncementStatusLabel(status: AnnouncementStatus, t: ComposerTranslation) {
  return t(ANNOUNCEMENT_STATUS_LABEL_KEY[status]);
}

function resolveAnnouncementLevelLabel(level: AnnouncementLevel, t: ComposerTranslation) {
  return t(ANNOUNCEMENT_LEVEL_LABEL_KEY[level]);
}

function resolveAnnouncementDeliveryModeLabel(deliveryMode: AnnouncementDeliveryMode, t: ComposerTranslation) {
  return t(ANNOUNCEMENT_DELIVERY_MODE_LABEL_KEY[deliveryMode]);
}

export function resolvePinnedLabel(pinned: boolean, t: ComposerTranslation) {
  return t(pinned ? ANNOUNCEMENT_PINNED_LABEL_KEY.true : ANNOUNCEMENT_PINNED_LABEL_KEY.false);
}

export function resolveAnnouncementUnread(item: AnnouncementItem) {
  return typeof item.unread === 'boolean' ? item.unread : !item.read_at;
}

function resolveAnnouncementUnreadLabel(item: AnnouncementItem, t: ComposerTranslation) {
  return t(resolveAnnouncementUnread(item) ? 'announcement.readState.unread' : 'announcement.readState.read');
}

export function announcementStatusTheme(status: AnnouncementStatus): AnnouncementTagTheme {
  switch (status) {
    case 'published':
      return 'success';
    case 'archived':
      return 'default';
    case 'draft':
    default:
      return 'primary';
  }
}

export function announcementLevelTheme(level: AnnouncementLevel): AnnouncementTagTheme {
  switch (level) {
    case 'success':
      return 'success';
    case 'warning':
      return 'warning';
    case 'error':
      return 'danger';
    case 'info':
    default:
      return 'primary';
  }
}

function formatAnnouncementDate(value: string | null | undefined, locale: string, t: ComposerTranslation) {
  return value ? formatCompactDateTime(value, locale) : t('announcement.value.notSet');
}
