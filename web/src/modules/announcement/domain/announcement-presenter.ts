import type { ComposerTranslation } from 'vue-i18n';

import { markdownToPlainTextSummary } from '@/shared/components/markdown';
import { formatLocaleDateTime } from '@/shared/observability';

import {
  ANNOUNCEMENT_DELIVERY_MODE_LABEL_KEY,
  ANNOUNCEMENT_LEVEL_LABEL_KEY,
  ANNOUNCEMENT_PINNED_LABEL_KEY,
  ANNOUNCEMENT_STATUS_LABEL_KEY,
  ANNOUNCEMENT_VISIBILITY_LABEL_KEY,
} from '../contract/presentation';
import type {
  AnnouncementDeliveryMode,
  AnnouncementItem,
  AnnouncementLevel,
  AnnouncementStatus,
  AnnouncementVisibilityState,
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
  publishedAtLabel: string;
  publishedByLabel: string;
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
  archivedAtLabel: string;
  visibility: AnnouncementVisibilityState;
  visibilityLabel: string;
  visibilityTheme: AnnouncementTagTheme;
};

export function presentAnnouncement(
  item: AnnouncementItem,
  t: ComposerTranslation,
  locale: string,
  now = new Date(),
): AnnouncementViewModel {
  const visibility = resolveAnnouncementVisibility(item, now);
  return {
    archivedAtLabel: formatAnnouncementDate(item.archived_at, locale, t),
    content: item.content,
    createdAtLabel: formatAnnouncementDate(item.created_at, locale, t),
    deliveryMode: item.delivery_mode,
    deliveryModeLabel: resolveAnnouncementDeliveryModeLabel(item.delivery_mode, t),
    expireAtLabel: formatExpireAtLabel(item.expire_at, locale, t),
    id: item.id,
    level: item.level,
    levelLabel: resolveAnnouncementLevelLabel(item.level, t),
    levelTheme: announcementLevelTheme(item.level),
    pinned: item.pinned,
    pinnedLabel: resolvePinnedLabel(item.pinned, t),
    publishedAtLabel: formatAnnouncementDate(item.published_at, locale, t),
    publishedByLabel: item.published_by ? String(item.published_by) : t('announcement.value.notSet'),
    publishAtLabel: formatPublishAtLabel(item.publish_at, locale, t),
    readAtLabel: formatAnnouncementDate(item.read_at, locale, t),
    summary: markdownToPlainTextSummary(item.content),
    status: item.status,
    statusLabel: resolveAnnouncementStatusLabel(item.status, t),
    statusTheme: announcementStatusTheme(item.status),
    title: item.title,
    unread: resolveAnnouncementUnread(item),
    unreadLabel: resolveAnnouncementUnreadLabel(item, t),
    updatedAtLabel: formatAnnouncementDate(item.updated_at, locale, t),
    visibility,
    visibilityLabel: resolveAnnouncementVisibilityLabel(visibility, t),
    visibilityTheme: announcementVisibilityTheme(visibility),
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

function resolveAnnouncementVisibilityLabel(visibility: AnnouncementVisibilityState, t: ComposerTranslation) {
  return t(ANNOUNCEMENT_VISIBILITY_LABEL_KEY[visibility]);
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

function announcementVisibilityTheme(visibility: AnnouncementVisibilityState): AnnouncementTagTheme {
  switch (visibility) {
    case 'visible':
      return 'success';
    case 'scheduled':
      return 'warning';
    case 'expired':
      return 'danger';
    case 'archived':
      return 'default';
    case 'draft':
    default:
      return 'primary';
  }
}

function resolveAnnouncementVisibility(
  item: Pick<AnnouncementItem, 'expire_at' | 'publish_at' | 'status'>,
  now = new Date(),
): AnnouncementVisibilityState {
  if (item.status === 'draft') {
    return 'draft';
  }
  if (item.status === 'archived') {
    return 'archived';
  }

  const nowTime = now.getTime();
  const expireAt = parseAnnouncementDate(item.expire_at);
  if (expireAt && expireAt.getTime() <= nowTime) {
    return 'expired';
  }

  const publishAt = parseAnnouncementDate(item.publish_at);
  if (publishAt && publishAt.getTime() > nowTime) {
    return 'scheduled';
  }

  return 'visible';
}

function formatPublishAtLabel(value: string | null | undefined, locale: string, t: ComposerTranslation) {
  return value ? formatLocaleDateTime(value, locale) : t('announcement.value.immediateEffective');
}

function formatExpireAtLabel(value: string | null | undefined, locale: string, t: ComposerTranslation) {
  return value ? formatLocaleDateTime(value, locale) : t('announcement.value.longTerm');
}

function formatAnnouncementDate(value: string | null | undefined, locale: string, t: ComposerTranslation) {
  return value ? formatLocaleDateTime(value, locale) : t('announcement.value.notSet');
}

function parseAnnouncementDate(value: string | null | undefined) {
  if (!value) {
    return null;
  }
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? null : date;
}
