import type { ComposerTranslation } from 'vue-i18n';

import { getMyAnnouncements } from '../api/announcement';
import { type AnnouncementViewModel, presentAnnouncement } from '../domain/announcement-presenter';
import type { AnnouncementItem } from '../types/announcement';

export type AnnouncementCandidateFilter = (item: AnnouncementItem) => boolean;

export async function loadUnreadAnnouncementCandidate(options: {
  filter?: AnnouncementCandidateFilter;
  locale: string;
  onError?: (error: unknown) => void;
  pageSize: number;
  t: ComposerTranslation;
}): Promise<AnnouncementViewModel | null> {
  try {
    const page = await getMyAnnouncements({
      page: 1,
      page_size: options.pageSize,
      unread_only: true,
    });
    const item = page.items.find(options.filter ?? (() => true));

    return item ? presentAnnouncement(item, options.t, options.locale) : null;
  } catch (error) {
    options.onError?.(error);
    return null;
  }
}
