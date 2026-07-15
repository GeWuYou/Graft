import type { MaybeRefOrGetter } from 'vue';
import { onBeforeUnmount, onMounted, ref, toValue } from 'vue';

export interface UseDocumentFullscreenOptions {
  target?: MaybeRefOrGetter<HTMLElement | null>;
}

/**
 * 获取当前运行环境中的文档对象。
 *
 * @returns 可用的全局 `Document` 对象；在文档对象不可用时返回 `null`
 */
function getDocument(): Document | null {
  return typeof document === 'undefined' ? null : document;
}

/**
 * 提供与文档全屏状态同步的浏览器 Fullscreen API 组合式封装。
 *
 * @param options - 配置全屏目标元素；未指定时使用文档根元素
 * @returns 包含进入、退出和切换全屏的方法，以及全屏状态和 API 支持状态
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
