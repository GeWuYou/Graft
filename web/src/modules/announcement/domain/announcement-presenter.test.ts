// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';

import type { AnnouncementItem } from '../types/announcement';
import {
  announcementLevelTheme,
  announcementStatusTheme,
  presentAnnouncement,
  resolveAnnouncementUnread,
  resolvePinnedLabel,
} from './announcement-presenter';

const labels: Record<string, string> = {
  'announcement.level.error': 'Error',
  'announcement.level.info': 'Info',
  'announcement.level.success': 'Success',
  'announcement.level.warning': 'Warning',
  'announcement.deliveryMode.popup': 'Popup',
  'announcement.deliveryMode.silent': 'Silent',
  'announcement.pinned.no': 'Normal',
  'announcement.pinned.yes': 'Pinned',
  'announcement.readState.read': 'Read',
  'announcement.readState.unread': 'Unread',
  'announcement.status.archived': 'Archived',
  'announcement.status.draft': 'Draft',
  'announcement.status.published': 'Published',
  'announcement.value.notSet': 'Not Set',
};

const translate = (key: string) => labels[key] ?? key;

describe('announcement presenter', () => {
  it('maps status and level values to TDesign tag themes', () => {
    expect(announcementStatusTheme('draft')).toBe('primary');
    expect(announcementStatusTheme('published')).toBe('success');
    expect(announcementStatusTheme('archived')).toBe('default');
    expect(announcementLevelTheme('info')).toBe('primary');
    expect(announcementLevelTheme('success')).toBe('success');
    expect(announcementLevelTheme('warning')).toBe('warning');
    expect(announcementLevelTheme('error')).toBe('danger');
  });

  it('presents list and detail display fields from key-first labels', () => {
    const view = presentAnnouncement(
      {
        content: 'Body',
        created_at: '2026-06-12T00:00:00Z',
        delivery_mode: 'popup',
        id: 12,
        level: 'warning',
        pinned: true,
        status: 'published',
        title: 'Maintenance',
        updated_at: '2026-06-12T01:00:00Z',
      } satisfies AnnouncementItem,
      translate,
      'en-US',
    );

    expect(view.statusLabel).toBe('Published');
    expect(view.levelLabel).toBe('Warning');
    expect(view.deliveryModeLabel).toBe('Popup');
    expect(view.pinnedLabel).toBe('Pinned');
    expect(view.publishAtLabel).toBe('Not Set');
    expect(view.unread).toBe(true);
    expect(view.unreadLabel).toBe('Unread');
  });

  it('resolves pinned labels without template branching', () => {
    expect(resolvePinnedLabel(true, translate)).toBe('Pinned');
    expect(resolvePinnedLabel(false, translate)).toBe('Normal');
  });

  it('prefers explicit unread state and falls back to read_at when needed', () => {
    expect(resolveAnnouncementUnread({ unread: false, read_at: null } as AnnouncementItem)).toBe(false);
    expect(resolveAnnouncementUnread({ read_at: '2026-06-12T01:00:00Z' } as AnnouncementItem)).toBe(false);
    expect(resolveAnnouncementUnread({ read_at: null } as AnnouncementItem)).toBe(true);
  });
});
