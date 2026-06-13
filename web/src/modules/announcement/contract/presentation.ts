// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type {
  AnnouncementDeliveryMode,
  AnnouncementLevel,
  AnnouncementStatus,
  AnnouncementVisibilityState,
} from '../types/announcement';

export const ANNOUNCEMENT_STATUS_LABEL_KEY: Record<AnnouncementStatus, string> = {
  archived: 'announcement.status.archived',
  draft: 'announcement.status.draft',
  published: 'announcement.status.published',
};

export const ANNOUNCEMENT_LEVEL_LABEL_KEY: Record<AnnouncementLevel, string> = {
  error: 'announcement.level.error',
  info: 'announcement.level.info',
  success: 'announcement.level.success',
  warning: 'announcement.level.warning',
};

export const ANNOUNCEMENT_DELIVERY_MODE_LABEL_KEY: Record<AnnouncementDeliveryMode, string> = {
  popup: 'announcement.deliveryMode.popup',
  silent: 'announcement.deliveryMode.silent',
};

export const ANNOUNCEMENT_PINNED_LABEL_KEY = {
  false: 'announcement.pinned.no',
  true: 'announcement.pinned.yes',
} as const;

export const ANNOUNCEMENT_VISIBILITY_LABEL_KEY: Record<AnnouncementVisibilityState, string> = {
  archived: 'announcement.visibility.archived',
  draft: 'announcement.visibility.draft',
  expired: 'announcement.visibility.expired',
  scheduled: 'announcement.visibility.scheduled',
  visible: 'announcement.visibility.visible',
};
