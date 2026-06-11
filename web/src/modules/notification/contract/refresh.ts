// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export const NOTIFICATION_HEADER_REFRESH_EVENT = 'graft:notification-header-refresh';

export function requestNotificationHeaderRefresh() {
  if (typeof window === 'undefined') return;
  window.dispatchEvent(new CustomEvent(NOTIFICATION_HEADER_REFRESH_EVENT));
}
