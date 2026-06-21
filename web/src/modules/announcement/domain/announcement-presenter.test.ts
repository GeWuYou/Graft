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
  'announcement.value.immediateEffective': 'After Publish',
  'announcement.value.longTerm': 'Long-Term',
  'announcement.value.notSet': 'Not Set',
  'announcement.visibility.archived': 'Archived',
  'announcement.visibility.draft': 'Draft',
  'announcement.visibility.expired': 'Expired',
  'announcement.visibility.scheduled': 'Not Started',
  'announcement.visibility.visible': 'Visible',
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
        published_at: '2026-06-12T00:30:00Z',
        published_by: 7,
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
    expect(view.publishAtLabel).toBe('After Publish');
    expect(view.expireAtLabel).toBe('Long-Term');
    expect(view.publishedAtLabel).not.toBe('Not Set');
    expect(view.publishedByLabel).toBe('7');
    expect(view.visibilityLabel).toBe('Visible');
    expect(view.unread).toBe(true);
    expect(view.unreadLabel).toBe('Unread');
  });

  it('formats visible dates with the provided locale', () => {
    const view = presentAnnouncement(
      {
        content: 'Body',
        created_at: '2026-06-12T00:00:00Z',
        delivery_mode: 'silent',
        id: 13,
        level: 'info',
        pinned: false,
        publish_at: '2026-06-12T01:00:00Z',
        status: 'published',
        title: 'Locale date',
        updated_at: '2026-06-12T02:00:00Z',
      } satisfies AnnouncementItem,
      translate,
      'en-US',
    );

    expect(view.createdAtLabel).toBe(
      new Intl.DateTimeFormat('en-US', {
        day: '2-digit',
        hour: 'numeric',
        minute: '2-digit',
        month: '2-digit',
        second: '2-digit',
        year: 'numeric',
      }).format(new Date('2026-06-12T00:00:00Z')),
    );
  });

  it('resolves pinned labels without template branching', () => {
    expect(resolvePinnedLabel(true, translate)).toBe('Pinned');
    expect(resolvePinnedLabel(false, translate)).toBe('Normal');
  });

  it('derives management visibility state from lifecycle and visibility window', () => {
    const now = new Date('2026-06-12T02:00:00Z');
    const base = {
      content: 'Body',
      created_at: '2026-06-12T00:00:00Z',
      delivery_mode: 'silent',
      id: 1,
      level: 'info',
      pinned: false,
      title: 'Visibility',
      updated_at: '2026-06-12T00:00:00Z',
    } satisfies Partial<AnnouncementItem>;

    expect(
      presentAnnouncement({ ...base, id: 1, status: 'draft' } as AnnouncementItem, translate, 'en-US', now).visibility,
    ).toBe('draft');
    expect(
      presentAnnouncement({ ...base, id: 2, status: 'archived' } as AnnouncementItem, translate, 'en-US', now)
        .visibility,
    ).toBe('archived');
    expect(
      presentAnnouncement(
        { ...base, id: 3, publish_at: '2026-06-12T03:00:00Z', status: 'published' } as AnnouncementItem,
        translate,
        'en-US',
        now,
      ).visibility,
    ).toBe('scheduled');
    expect(
      presentAnnouncement(
        { ...base, id: 4, publish_at: '2026-06-12T01:00:00Z', status: 'published' } as AnnouncementItem,
        translate,
        'en-US',
        now,
      ).visibility,
    ).toBe('visible');
    expect(
      presentAnnouncement(
        { ...base, expire_at: '2026-06-12T02:00:00Z', id: 5, status: 'published' } as AnnouncementItem,
        translate,
        'en-US',
        now,
      ).visibility,
    ).toBe('expired');
  });

  it('prefers explicit unread state and falls back to read_at when needed', () => {
    expect(resolveAnnouncementUnread({ unread: false, read_at: null } as AnnouncementItem)).toBe(false);
    expect(resolveAnnouncementUnread({ read_at: '2026-06-12T01:00:00Z' } as AnnouncementItem)).toBe(false);
    expect(resolveAnnouncementUnread({ read_at: null } as AnnouncementItem)).toBe(true);
  });
});
