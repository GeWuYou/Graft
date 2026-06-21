export const ANNOUNCEMENT_ROUTE_PATH = {
  MANAGEMENT: '/server/announcements',
  USER_LIST: '/announcements',
} as const;

export const ANNOUNCEMENT_API_PATH = {
  LIST: '/api/announcements',
  DETAIL: '/api/announcements/{id}',
  PUBLISH: '/api/announcements/{id}/publish',
  ARCHIVE: '/api/announcements/{id}/archive',
  MY_LIST: '/api/my/announcements',
  MY_READ: '/api/my/announcements/{id}/read',
  MY_READ_ALL: '/api/my/announcements/read-all',
  MY_UNREAD_COUNT: '/api/my/announcements/unread-count',
} as const;

export function buildAnnouncementDetailApiPath(id: number) {
  return ANNOUNCEMENT_API_PATH.DETAIL.replace('{id}', encodeURIComponent(String(id)));
}

export function buildAnnouncementPublishApiPath(id: number) {
  return ANNOUNCEMENT_API_PATH.PUBLISH.replace('{id}', encodeURIComponent(String(id)));
}

export function buildAnnouncementArchiveApiPath(id: number) {
  return ANNOUNCEMENT_API_PATH.ARCHIVE.replace('{id}', encodeURIComponent(String(id)));
}

export function buildMyAnnouncementReadApiPath(id: number) {
  return ANNOUNCEMENT_API_PATH.MY_READ.replace('{id}', encodeURIComponent(String(id)));
}
