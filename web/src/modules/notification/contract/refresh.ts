export const NOTIFICATION_HEADER_REFRESH_EVENT = 'graft:notification-header-refresh';

/** 广播通知状态变化，顶栏入口据此刷新摘要而不依赖页面实例互相引用。 */
export function requestNotificationHeaderRefresh() {
  if (typeof window === 'undefined') return;
  window.dispatchEvent(new CustomEvent(NOTIFICATION_HEADER_REFRESH_EVENT));
}
