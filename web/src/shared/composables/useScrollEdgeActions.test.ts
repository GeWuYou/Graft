import { describe, expect, it, vi } from 'vitest';
import { effectScope, nextTick, ref } from 'vue';

import { type ScrollEdgeActionsController, useScrollEdgeActions } from './useScrollEdgeActions';

function createTarget({ scrollTop = 0, clientHeight = 100, scrollHeight = 300 } = {}) {
  const target = document.createElement('div');
  Object.defineProperties(target, {
    scrollTop: { configurable: true, writable: true, value: scrollTop },
    clientHeight: { configurable: true, value: clientHeight },
    scrollHeight: { configurable: true, value: scrollHeight },
  });
  return target;
}

function mountController(options: Parameters<typeof useScrollEdgeActions>[0] = {}) {
  const scope = effectScope();
  let controller!: ScrollEdgeActionsController;
  scope.run(() => {
    controller = useScrollEdgeActions(options);
  });
  return { controller, stop: () => scope.stop() };
}

describe('useScrollEdgeActions', () => {
  it('shows only the bottom action at the top and only the top action at the bottom', async () => {
    const target = createTarget();
    const { controller, stop } = mountController({ target });
    await nextTick();

    expect(controller.hasScrollableContent.value).toBe(true);
    expect(controller.topVisible.value).toBe(false);
    expect(controller.bottomVisible.value).toBe(true);

    target.scrollTop = 100;
    target.dispatchEvent(new Event('scroll'));
    expect(controller.topVisible.value).toBe(true);
    expect(controller.bottomVisible.value).toBe(true);

    target.scrollTop = 200;
    target.dispatchEvent(new Event('scroll'));
    expect(controller.topVisible.value).toBe(true);
    expect(controller.bottomVisible.value).toBe(false);
    stop();
  });

  it('hides both actions when the target has no overflow', async () => {
    const target = createTarget({ clientHeight: 300, scrollHeight: 300 });
    const { controller, stop } = mountController({ target });
    await nextTick();

    expect(controller.hasScrollableContent.value).toBe(false);
    expect(controller.topVisible.value).toBe(false);
    expect(controller.bottomVisible.value).toBe(false);
    stop();
  });

  it('keeps a real sub-threshold overflow marked as scrollable', async () => {
    const target = createTarget({ clientHeight: 100, scrollHeight: 101 });
    const { controller, stop } = mountController({ target, threshold: 4 });
    await nextTick();

    expect(controller.metrics.value.maxScrollTop).toBe(1);
    expect(controller.hasScrollableContent.value).toBe(true);
    stop();
  });

  it('supports target registration and restores the previous target on cleanup', async () => {
    const first = createTarget();
    const second = createTarget({ scrollTop: 200 });
    const { controller, stop } = mountController({ target: first });
    await nextTick();

    const unregister = controller.registerTarget(second);
    await nextTick();
    expect(controller.metrics.value.target).toBe(second);
    expect(controller.topVisible.value).toBe(true);

    unregister();
    await nextTick();
    expect(controller.metrics.value.target).toBe(first);
    stop();
  });

  it('uses custom callbacks and visibility predicates without scanning other elements', async () => {
    const target = createTarget();
    const onScrollToTop = vi.fn();
    const onScrollToBottom = vi.fn();
    const { controller, stop } = mountController({
      target,
      isTopVisible: () => true,
      isBottomVisible: () => false,
      onScrollToTop,
      onScrollToBottom,
    });
    await nextTick();

    controller.scrollToTop();
    controller.scrollToBottom();
    expect(onScrollToTop).toHaveBeenCalledOnce();
    expect(onScrollToBottom).toHaveBeenCalledOnce();
    expect(controller.topVisible.value).toBe(true);
    expect(controller.bottomVisible.value).toBe(false);
    stop();
  });

  it('reacts to a ref target changing and cleans up listeners', async () => {
    const first = createTarget();
    const second = createTarget({ scrollTop: 200 });
    const target = ref<HTMLElement | null>(first);
    const { controller, stop } = mountController({ target });
    await nextTick();
    expect(controller.metrics.value.target).toBe(first);

    target.value = second;
    await nextTick();
    expect(controller.metrics.value.target).toBe(second);
    stop();
    target.value = first;
    await nextTick();
    expect(controller.metrics.value.target).toBe(null);
  });

  it('restores the original ref target identity after temporary registration', async () => {
    const first = createTarget();
    const second = createTarget({ scrollTop: 200 });
    const initialTarget = ref<HTMLElement | null>(first);
    const { controller, stop } = mountController({ target: initialTarget });
    await nextTick();

    const unregister = controller.registerTarget(second);
    await nextTick();
    unregister();
    await nextTick();

    const third = createTarget({ scrollTop: 100 });
    initialTarget.value = third;
    await nextTick();
    expect(controller.metrics.value.target).toBe(third);
    stop();
  });

  it('uses instant scrolling when reduced motion is preferred', async () => {
    const target = createTarget();
    const scrollTo = vi.fn();
    Object.defineProperty(target, 'scrollTo', { configurable: true, value: scrollTo });
    vi.stubGlobal('matchMedia', () => ({ matches: true }));
    const { controller, stop } = mountController({ target });
    await nextTick();

    controller.scrollToBottom();
    expect(scrollTo).toHaveBeenCalledWith({ top: 200, behavior: 'auto' });
    stop();
    vi.unstubAllGlobals();
  });

  it('disconnects ResizeObserver when the controller scope is stopped', async () => {
    const disconnect = vi.fn();
    const observe = vi.fn();
    class ResizeObserverStub {
      observe = observe;
      disconnect = disconnect;
    }
    vi.stubGlobal('ResizeObserver', ResizeObserverStub);

    const { controller, stop } = mountController({ target: createTarget() });
    await nextTick();
    expect(observe).toHaveBeenCalledOnce();
    expect(controller.metrics.value.target).not.toBeNull();

    stop();
    expect(disconnect).toHaveBeenCalledOnce();
    vi.unstubAllGlobals();
  });

  it('refreshes metrics after target mutations and disconnects the observer on cleanup', async () => {
    let observerCallback!: MutationCallback;
    const observe = vi.fn();
    const disconnect = vi.fn();
    class MutationObserverStub {
      constructor(callback: MutationCallback) {
        observerCallback = callback;
      }
      observe = observe;
      disconnect = disconnect;
    }
    vi.stubGlobal('MutationObserver', MutationObserverStub);

    const target = createTarget();
    const { controller, stop } = mountController({ target });
    await nextTick();
    expect(observe).toHaveBeenCalledWith(target, expect.objectContaining({ childList: true, subtree: true }));

    Object.defineProperty(target, 'scrollHeight', { configurable: true, value: 500 });
    observerCallback([], {} as MutationObserver);
    expect(controller.metrics.value.scrollHeight).toBe(500);

    stop();
    expect(disconnect).toHaveBeenCalledOnce();
    vi.unstubAllGlobals();
  });
});
