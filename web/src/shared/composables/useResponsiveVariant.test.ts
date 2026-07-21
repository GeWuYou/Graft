import { afterEach, describe, expect, it, vi } from 'vitest';
import { effectScope, nextTick, ref } from 'vue';

import { useResponsiveVariant } from './useResponsiveVariant';

class ResizeObserverMock {
  static instances: ResizeObserverMock[] = [];

  callback: ResizeObserverCallback;
  disconnect = vi.fn();
  observe = vi.fn();

  constructor(callback: ResizeObserverCallback) {
    this.callback = callback;
    ResizeObserverMock.instances.push(this);
  }

  emit(width: number) {
    this.callback([{ contentRect: { height: 0, width } } as ResizeObserverEntry], this as unknown as ResizeObserver);
  }
}

describe('useResponsiveVariant', () => {
  afterEach(() => {
    ResizeObserverMock.instances = [];
    vi.unstubAllGlobals();
  });

  it('derives semantic variants from the observed container', async () => {
    vi.stubGlobal('ResizeObserver', ResizeObserverMock);
    const target = ref<HTMLElement | null>(null);
    const scope = effectScope();
    const variant = scope.run(() => useResponsiveVariant(target, { interaction: 'readonly', layout: 'stack' }));

    target.value = document.createElement('section');
    await nextTick();
    ResizeObserverMock.instances[0]?.emit(1000);

    expect(variant?.value).toEqual({
      density: 'spacious',
      interaction: 'readonly',
      layout: 'stack',
      presentation: 'data',
      surface: 'page',
    });
    scope.stop();
  });
});
