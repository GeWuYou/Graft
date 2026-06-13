// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { components } from '@/contracts/openapi/generated/schema';

export type AnnouncementItem = components['schemas']['announcement-item'];
export type AnnouncementListResponse = components['schemas']['announcement-list-response'];
export type AnnouncementReadAllResponse = components['schemas']['announcement-read-all-response'];
export type AnnouncementUnreadCountResponse = components['schemas']['announcement-unread-count-response'];
export type AnnouncementStatus = components['schemas']['announcement-status'];
export type AnnouncementLevel = components['schemas']['announcement-level'];
export type AnnouncementDeliveryMode = components['schemas']['announcement-delivery-mode'];
export type CreateAnnouncementRequest = components['schemas']['create-announcement-request'];
export type UpdateAnnouncementRequest = components['schemas']['update-announcement-request'];
export type PublishAnnouncementRequest = components['schemas']['publish-announcement-request'];
export type AnnouncementSort = 'updated_desc' | 'publish_desc' | 'pinned_publish_desc';
export type AnnouncementStatusFilter = '' | AnnouncementStatus;
export type AnnouncementLevelFilter = '' | AnnouncementLevel;
export type AnnouncementPinnedFilter = '' | 'true' | 'false';
export type AnnouncementVisibilityState = 'draft' | 'scheduled' | 'visible' | 'expired' | 'archived';

export type AnnouncementListQuery = {
  keyword?: string;
  level?: AnnouncementLevel;
  page?: number;
  page_size?: number;
  pinned?: boolean;
  sort?: AnnouncementSort;
  status?: AnnouncementStatus;
};

export type MyAnnouncementListQuery = {
  page?: number;
  page_size?: number;
  unread_only?: boolean;
};

export type AnnouncementFilterState = {
  keyword: string;
  level: AnnouncementLevelFilter;
  pinned: AnnouncementPinnedFilter;
  sort: AnnouncementSort;
  status: AnnouncementStatusFilter;
};

export type AnnouncementFormState = {
  content: string;
  delivery_mode: AnnouncementDeliveryMode;
  expire_at: string;
  level: AnnouncementLevel;
  pinned: boolean;
  publish_at: string;
  title: string;
};
