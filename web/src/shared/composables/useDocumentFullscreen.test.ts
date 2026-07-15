import { mount } from '@vue/test-utils';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { defineComponent } from 'vue';

import { useDocumentFullscreen } from './useDocumentFullscreen';

type FullscreenState = ReturnType<typeof useDocumentFullscreen>;

const originalFullscreenElement = Object.getOwnPropertyDescriptor(document, 'fullscreenElement');
const originalExitFullscreen = Object.getOwnPropertyDescriptor(document, 'exitFullscreen');
const originalRequestFullscreen = Object.getOwnPropertyDescriptor(HTMLElement.prototype, 'requestFullscreen');

let fullscreenElement: Element | null = null;

function restoreFullscreenApi() {
  fullscreenElement = null;

  if (originalFullscreenElement) {
    Object.defineProperty(document, 'fullscreenElement', originalFullscreenElement);
  } else {
    Reflect.deleteProperty(document, 'fullscreenElement');
  }

  if (originalExitFullscreen) {
    Object.defineProperty(document, 'exitFullscreen', originalExitFullscreen);
  } else {
    Reflect.deleteProperty(document, 'exitFullscreen');
  }

  if (originalRequestFullscreen) {
    Object.defineProperty(HTMLElement.prototype, 'requestFullscreen', originalRequestFullscreen);
  } else {
    Reflect.deleteProperty(HTMLElement.prototype, 'requestFullscreen');
  }
}

function installFullscreenApi() {
  Object.defineProperty(document, 'fullscreenElement', {
    configurable: true,
    get: () => fullscreenElement,
  });
  Object.defineProperty(document, 'exitFullscreen', {
    configurable: true,
    value: vi.fn(async () => {
      fullscreenElement = null;
      document.dispatchEvent(new Event('fullscreenchange'));
    }),
  });
  Object.defineProperty(HTMLElement.prototype, 'requestFullscreen', {
    configurable: true,
    value: vi.fn(async () => {
      fullscreenElement = document.documentElement;
      document.dispatchEvent(new Event('fullscreenchange'));
    }),
  });
}

function mountComposable(options?: Parameters<typeof useDocumentFullscreen>[0]) {
  let state: FullscreenState | undefined;
  const Harness = defineComponent({
    setup() {
      state = useDocumentFullscreen(options);
      return () => null;
    },
  });
  const wrapper = mount(Harness);

  return {
    get state() {
      if (!state) {
        throw new Error('Fullscreen composable was not initialized.');
      }
      return state;
    },
    wrapper,
  };
}

afterEach(() => {
  restoreFullscreenApi();
});

describe('useDocumentFullscreen', () => {
  it('uses documentElement by default and reflects enter/exit operations', async () => {
    installFullscreenApi();
    const { state, wrapper } = mountComposable();

    expect(state.isSupported.value).toBe(true);
    await expect(state.enter()).resolves.toBe(true);
    expect(fullscreenElement).toBe(document.documentElement);
    expect(state.isFullscreen.value).toBe(true);

    await expect(state.toggle()).resolves.toBe(true);
    expect(state.isFullscreen.value).toBe(false);
    wrapper.unmount();
  });

  it('uses an optional target and follows browser-driven fullscreen exits', async () => {
    installFullscreenApi();
    const target = document.createElement('section');
    Object.defineProperty(target, 'requestFullscreen', {
      configurable: true,
      value: vi.fn(async () => {
        fullscreenElement = target;
        document.dispatchEvent(new Event('fullscreenchange'));
      }),
    });
    const { state, wrapper } = mountComposable({ target });

    await state.enter();
    expect(fullscreenElement).toBe(target);
    expect(state.isFullscreen.value).toBe(true);

    fullscreenElement = null;
    document.dispatchEvent(new Event('fullscreenchange'));
    expect(state.isFullscreen.value).toBe(false);
    wrapper.unmount();
  });

  it('returns false for rejected calls and removes its event listener on unmount', async () => {
    installFullscreenApi();
    const requestFullscreen = vi.fn(async () => Promise.reject(new Error('Fullscreen denied')));
    Object.defineProperty(HTMLElement.prototype, 'requestFullscreen', {
      configurable: true,
      value: requestFullscreen,
    });
    const { state, wrapper } = mountComposable();

    await expect(state.enter()).resolves.toBe(false);
    expect(state.isFullscreen.value).toBe(false);

    wrapper.unmount();
    fullscreenElement = document.documentElement;
    document.dispatchEvent(new Event('fullscreenchange'));
    expect(state.isFullscreen.value).toBe(false);
  });
});
