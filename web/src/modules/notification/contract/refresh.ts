export const NOTIFICATION_HEADER_REFRESH_EVENT = 'graft:notification-header-refresh';

export function requestNotificationHeaderRefresh() {
  if (typeof window === 'undefined') return;
  window.dispatchEvent(new CustomEvent(NOTIFICATION_HEADER_REFRESH_EVENT));
}
