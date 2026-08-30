import {
  computed,
  type ComputedRef,
  type MaybeRefOrGetter,
  type Ref,
  ref,
  shallowRef,
  toValue,
  watchEffect,
} from 'vue';

import { emitScrollEdgeDebug } from '@/shared/debug/scroll-edge-actions-investigation';

/** 滚动控制器对明确目标视口读取的尺寸、位置和边界快照。 */
export interface ScrollEdgeMetrics {
  target: HTMLElement | null;
  scrollTop: number;
  clientHeight: number;
  scrollHeight: number;
  maxScrollTop: number;
  isScrollable: boolean;
  atTop: boolean;
  atBottom: boolean;
}

/** 配置滚动目标、边界阈值以及可选的页面自定义行为；阈值只用于边界吸附，不改变溢出判定。 */
export interface ScrollEdgeActionsOptions {
  target?: MaybeRefOrGetter<HTMLElement | null | undefined>;
  threshold?: number;
  behavior?: ScrollBehavior;
  onScrollToTop?: (metrics: ScrollEdgeMetrics) => void;
  onScrollToBottom?: (metrics: ScrollEdgeMetrics) => void;
  isTopVisible?: (metrics: ScrollEdgeMetrics) => boolean;
  isBottomVisible?: (metrics: ScrollEdgeMetrics) => boolean;
}

/** 供壳层或页面注册并驱动一个滚动视口的稳定控制接口。 */
export interface ScrollEdgeActionsController {
  readonly metrics: Readonly<Ref<ScrollEdgeMetrics>>;
  readonly topVisible: ComputedRef<boolean>;
  readonly bottomVisible: ComputedRef<boolean>;
  readonly hasScrollableContent: ComputedRef<boolean>;
  setTarget(target: MaybeRefOrGetter<HTMLElement | null | undefined>): void;
  registerTarget(target: MaybeRefOrGetter<HTMLElement | null | undefined>): () => void;
  refresh(): void;
  scrollToTop(): void;
  scrollToBottom(): void;
}

const DEFAULT_THRESHOLD = 4;
let controllerSequence = 0;

function emptyMetrics(): ScrollEdgeMetrics {
  return {
    target: null,
    scrollTop: 0,
    clientHeight: 0,
    scrollHeight: 0,
    maxScrollTop: 0,
    isScrollable: false,
    atTop: true,
    atBottom: true,
  };
}

function prefersReducedMotion(): boolean {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return false;
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}

/**
 * 为一个明确的滚动视口提供顶部/底部动作状态；不会扫描页面中的嵌套滚动容器。
 */
export function useScrollEdgeActions(options: ScrollEdgeActionsOptions = {}): ScrollEdgeActionsController {
  const controllerId = ++controllerSequence;
  const threshold = Math.max(0, options.threshold ?? DEFAULT_THRESHOLD);
  // 保持目标来源本身的引用身份，避免 ref 自动解包导致 registerTarget 无法可靠恢复。
  const targetSource = shallowRef<MaybeRefOrGetter<HTMLElement | null | undefined>>(options.target) as Ref<
    MaybeRefOrGetter<HTMLElement | null | undefined>
  >;
  const metrics = ref<ScrollEdgeMetrics>(emptyMetrics());

  let boundTarget: HTMLElement | null = null;
  let targetResizeObserver: ResizeObserver | null = null;
  let targetMutationObserver: MutationObserver | null = null;
  let lastDebugMetricsSignature = '';

  function readMetrics(target: HTMLElement | null): ScrollEdgeMetrics {
    if (!target) return emptyMetrics();

    const scrollTop = Math.max(0, target.scrollTop);
    const clientHeight = Math.max(0, target.clientHeight);
    const scrollHeight = Math.max(0, target.scrollHeight);
    const maxScrollTop = Math.max(0, scrollHeight - clientHeight);
    const isScrollable = maxScrollTop > 0;

    return {
      target,
      scrollTop,
      clientHeight,
      scrollHeight,
      maxScrollTop,
      isScrollable,
      atTop: scrollTop <= threshold,
      atBottom: maxScrollTop - scrollTop <= threshold,
    };
  }

  function refresh() {
    const nextMetrics = readMetrics(boundTarget);
    metrics.value = nextMetrics;
    const debugSignature = [
      nextMetrics.target ? 'attached' : 'detached',
      nextMetrics.scrollTop,
      nextMetrics.clientHeight,
      nextMetrics.scrollHeight,
      nextMetrics.maxScrollTop,
      nextMetrics.isScrollable,
      nextMetrics.atTop,
      nextMetrics.atBottom,
    ].join('|');
    if (debugSignature === lastDebugMetricsSignature) {
      return;
    }
    lastDebugMetricsSignature = debugSignature;
    emitScrollEdgeDebug('STATE_CHANGE', 'controller-metrics', {
      controllerId,
      targetAttached: Boolean(nextMetrics.target),
      scrollTop: nextMetrics.scrollTop,
      clientHeight: nextMetrics.clientHeight,
      scrollHeight: nextMetrics.scrollHeight,
      maxScrollTop: nextMetrics.maxScrollTop,
      isScrollable: nextMetrics.isScrollable,
      atTop: nextMetrics.atTop,
      atBottom: nextMetrics.atBottom,
    });
  }

  function unbindTarget() {
    if (boundTarget) boundTarget.removeEventListener('scroll', refresh);
    targetResizeObserver?.disconnect();
    targetResizeObserver = null;
    targetMutationObserver?.disconnect();
    targetMutationObserver = null;
    boundTarget = null;
    refresh();
  }

  function bindTarget(target: HTMLElement | null) {
    if (target === boundTarget) {
      refresh();
      return;
    }

    unbindTarget();
    boundTarget = target;
    emitScrollEdgeDebug('LIFECYCLE', 'target-bind', {
      controllerId,
      targetAttached: Boolean(boundTarget),
    });
    if (!boundTarget) return;

    boundTarget.addEventListener('scroll', refresh, { passive: true });
    if (typeof ResizeObserver !== 'undefined') {
      targetResizeObserver = new ResizeObserver(refresh);
      targetResizeObserver.observe(boundTarget);
    }
    if (typeof MutationObserver !== 'undefined') {
      targetMutationObserver = new MutationObserver(refresh);
      targetMutationObserver.observe(boundTarget, {
        attributes: true,
        characterData: true,
        childList: true,
        subtree: true,
      });
    }
    refresh();
  }

  watchEffect((onCleanup) => {
    bindTarget(toValue(targetSource.value) ?? null);
    if (typeof window === 'undefined') return;
    window.addEventListener('resize', refresh, { passive: true });
    onCleanup(() => {
      window.removeEventListener('resize', refresh);
      unbindTarget();
    });
  });

  const topVisible = computed(() => {
    const current = metrics.value;
    return options.isTopVisible ? options.isTopVisible(current) : current.isScrollable && !current.atTop;
  });
  const bottomVisible = computed(() => {
    const current = metrics.value;
    return options.isBottomVisible ? options.isBottomVisible(current) : current.isScrollable && !current.atBottom;
  });

  function setTarget(target: MaybeRefOrGetter<HTMLElement | null | undefined>) {
    targetSource.value = target;
  }

  function registerTarget(target: MaybeRefOrGetter<HTMLElement | null | undefined>) {
    const previousTarget = targetSource.value;
    emitScrollEdgeDebug('LIFECYCLE', 'target-register', {
      controllerId,
      registration: 'override',
      targetAttached: Boolean(toValue(target)),
    });
    setTarget(target);
    return () => {
      if (targetSource.value !== target) return;
      emitScrollEdgeDebug('LIFECYCLE', 'target-register-cleanup', {
        controllerId,
        registration: 'restore',
        targetAttached: Boolean(toValue(previousTarget)),
      });
      setTarget(previousTarget);
    };
  }

  function scrollTo(position: number, callback: ((current: ScrollEdgeMetrics) => void) | undefined) {
    const current = metrics.value;
    if (callback) {
      callback(current);
      refresh();
      return;
    }
    if (!current.target) return;

    const behavior = prefersReducedMotion() ? 'auto' : (options.behavior ?? 'smooth');
    if (typeof current.target.scrollTo === 'function') {
      current.target.scrollTo({ top: position, behavior });
    } else {
      current.target.scrollTop = position;
    }
    refresh();
  }

  function scrollToTop() {
    scrollTo(0, options.onScrollToTop);
  }

  function scrollToBottom() {
    scrollTo(metrics.value.maxScrollTop, options.onScrollToBottom);
  }

  return {
    metrics,
    topVisible,
    bottomVisible,
    hasScrollableContent: computed(() => metrics.value.isScrollable),
    setTarget,
    registerTarget,
    refresh,
    scrollToTop,
    scrollToBottom,
  };
}
