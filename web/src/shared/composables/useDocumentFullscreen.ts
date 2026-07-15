import type { MaybeRefOrGetter } from 'vue';
import { onBeforeUnmount, onMounted, ref, toValue } from 'vue';

export interface UseDocumentFullscreenOptions {
  target?: MaybeRefOrGetter<HTMLElement | null>;
}

function getDocument(): Document | null {
  return typeof document === 'undefined' ? null : document;
}

/**
 * Provides lifecycle-safe access to the browser Document Fullscreen API.
 */
export function useDocumentFullscreen(options: UseDocumentFullscreenOptions = {}) {
  const isFullscreen = ref(false);
  const isSupported = ref(false);

  const syncState = () => {
    const currentDocument = getDocument();
    isFullscreen.value = Boolean(currentDocument?.fullscreenElement);
    isSupported.value = Boolean(
      currentDocument &&
      typeof currentDocument.exitFullscreen === 'function' &&
      typeof currentDocument.documentElement?.requestFullscreen === 'function',
    );
  };

  const resolveTarget = () => {
    const currentDocument = getDocument();
    return options.target === undefined ? (currentDocument?.documentElement ?? null) : toValue(options.target);
  };

  const enter = async (): Promise<boolean> => {
    const target = resolveTarget();
    if (!target || typeof target.requestFullscreen !== 'function') {
      syncState();
      return false;
    }

    try {
      await target.requestFullscreen();
      syncState();
      return true;
    } catch {
      syncState();
      return false;
    }
  };

  const exit = async (): Promise<boolean> => {
    const currentDocument = getDocument();
    if (!currentDocument || typeof currentDocument.exitFullscreen !== 'function') {
      syncState();
      return false;
    }

    try {
      await currentDocument.exitFullscreen();
      syncState();
      return true;
    } catch {
      syncState();
      return false;
    }
  };

  const toggle = async (): Promise<boolean> => {
    syncState();
    return isFullscreen.value ? exit() : enter();
  };

  onMounted(() => {
    const currentDocument = getDocument();
    syncState();
    currentDocument?.addEventListener('fullscreenchange', syncState);
  });

  onBeforeUnmount(() => {
    getDocument()?.removeEventListener('fullscreenchange', syncState);
  });

  return {
    enter,
    exit,
    isFullscreen,
    isSupported,
    toggle,
  };
}
