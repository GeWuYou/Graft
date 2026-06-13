// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

const ANNOUNCEMENT_CHANGED_EVENT = 'graft:announcement-changed';

export function emitAnnouncementChanged() {
  window.dispatchEvent(new CustomEvent(ANNOUNCEMENT_CHANGED_EVENT));
}

export function onAnnouncementChanged(handler: EventListener) {
  window.addEventListener(ANNOUNCEMENT_CHANGED_EVENT, handler);

  return () => {
    window.removeEventListener(ANNOUNCEMENT_CHANGED_EVENT, handler);
  };
}
