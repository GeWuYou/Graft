import { afterEach, describe, expect, it, vi } from 'vitest';
import { effectScope, nextTick, ref } from 'vue';

import { useContainerSize } from './useContainerSize';

class ResizeObserverMock {
  static instances: ResizeObserverMock[] = [];

  callback: ResizeObserverCallback;
  disconnect = vi.fn();
  observe = vi.fn();

  constructor(callback: ResizeObserverCallback) {
    this.callback = callback;
    ResizeObserverMock.instances.push(this);
  }

  emit(width: number, height: number) {
    this.callback([{ contentRect: { height, width } } as ResizeObserverEntry], this as unknown as ResizeObserver);
  }
}

describe('useContainerSize', () => {
  afterEach(() => {
    ResizeObserverMock.instances = [];
    vi.unstubAllGlobals();
  });

  it('observes element content size and disconnects when its scope ends', async () => {
    vi.stubGlobal('ResizeObserver', ResizeObserverMock);
    const target = ref<HTMLElement | null>(null);
    const scope = effectScope();
    const size = scope.run(() => useContainerSize(target));

    target.value = document.createElement('section');
    Object.defineProperties(target.value, {
      clientHeight: { configurable: true, value: 320 },
      clientWidth: { configurable: true, value: 480 },
    });
    await nextTick();

    const observer = ResizeObserverMock.instances[0];
    expect(observer?.observe).toHaveBeenCalledWith(target.value);
    expect(size?.value).toEqual({ height: 320, width: 480 });
    observer?.emit(480, 320);
    expect(size?.value).toEqual({ height: 320, width: 480 });

    scope.stop();
    expect(observer?.disconnect).toHaveBeenCalled();
  });

  it('uses an empty SSR-safe fallback when ResizeObserver is unavailable', async () => {
    vi.stubGlobal('ResizeObserver', undefined);
    const target = ref<HTMLElement | null>(document.createElement('section'));
    const scope = effectScope();
    const size = scope.run(() => useContainerSize(target));
    await nextTick();

    expect(size?.value).toEqual({ height: 0, width: 0 });
    scope.stop();
  });

  it('reads the mounted container size before the first observer notification', async () => {
    vi.stubGlobal('ResizeObserver', ResizeObserverMock);
    const target = ref<HTMLElement | null>(document.createElement('section'));
    Object.defineProperties(target.value, {
      clientHeight: { configurable: true, value: 640 },
      clientWidth: { configurable: true, value: 480 },
    });
    const scope = effectScope();
    const size = scope.run(() => useContainerSize(target));
    await nextTick();

    expect(size?.value).toEqual({ height: 640, width: 480 });
    scope.stop();
  });
});
