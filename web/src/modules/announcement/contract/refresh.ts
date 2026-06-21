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
