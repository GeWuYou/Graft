const DEFAULT_FLUSH_INTERVAL_MS = 100;
const DEFAULT_INTERACTION_IDLE_MS = 120;

type FrameScheduler = (callback: FrameRequestCallback) => number;
type FrameCanceller = (handle: number) => void;

export type LogViewportCommitSchedulerOptions<T> = Readonly<{
  flushIntervalMs?: number;
  interactionIdleMs?: number;
  onCommit: (value: T) => void;
  requestFrame?: FrameScheduler;
  cancelFrame?: FrameCanceller;
}>;

/**
 * 限制日志视图更新频率，同时保留最新的完整快照；用户滚动期间延迟提交，避免视口状态被后台流量抢占。
 */
export class LogViewportCommitScheduler<T> {
  readonly #flushIntervalMs: number;
  readonly #interactionIdleMs: number;
  readonly #onCommit: (value: T) => void;
  readonly #requestFrame: FrameScheduler;
  readonly #cancelFrame: FrameCanceller;
  #latest: T | undefined;
  #hasLatest = false;
  #interactionLocked = false;
  #disposed = false;
  #flushTimer: ReturnType<typeof setTimeout> | null = null;
  #interactionTimer: ReturnType<typeof setTimeout> | null = null;
  #frameHandle: number | null = null;

  constructor(options: LogViewportCommitSchedulerOptions<T>) {
    this.#flushIntervalMs = options.flushIntervalMs ?? DEFAULT_FLUSH_INTERVAL_MS;
    this.#interactionIdleMs = options.interactionIdleMs ?? DEFAULT_INTERACTION_IDLE_MS;
    this.#onCommit = options.onCommit;
    this.#requestFrame = options.requestFrame ?? defaultRequestFrame;
    this.#cancelFrame = options.cancelFrame ?? defaultCancelFrame;
  }

  publish(value: T, immediate = false) {
    if (this.#disposed) return;
    this.#latest = value;
    this.#hasLatest = true;
    if (immediate && !this.#isInteractionActive()) {
      this.#clearScheduledFlush();
      this.#commit();
      return;
    }
    this.#scheduleFlush();
  }

  beginInteraction() {
    if (this.#disposed) return;
    this.#interactionLocked = true;
    this.#clearScheduledFlush();
    this.#clearInteractionTimer();
  }

  endInteraction() {
    if (this.#disposed) return;
    this.#interactionLocked = false;
    this.#deferUntilInteractionIdle();
  }

  noteUserScroll() {
    if (this.#disposed || this.#interactionLocked) return;
    this.#clearScheduledFlush();
    this.#deferUntilInteractionIdle();
  }

  destroy() {
    this.#disposed = true;
    this.#hasLatest = false;
    this.#latest = undefined;
    this.#clearScheduledFlush();
    this.#clearInteractionTimer();
  }

  #isInteractionActive() {
    return this.#interactionLocked || this.#interactionTimer !== null;
  }

  #deferUntilInteractionIdle() {
    this.#clearInteractionTimer();
    this.#interactionTimer = setTimeout(() => {
      this.#interactionTimer = null;
      this.#scheduleFlush();
    }, this.#interactionIdleMs);
  }

  #scheduleFlush() {
    if (
      this.#disposed ||
      !this.#hasLatest ||
      this.#isInteractionActive() ||
      this.#flushTimer ||
      this.#frameHandle !== null
    ) {
      return;
    }
    this.#flushTimer = setTimeout(() => {
      this.#flushTimer = null;
      if (this.#disposed || this.#isInteractionActive() || !this.#hasLatest) return;
      this.#frameHandle = this.#requestFrame(() => {
        this.#frameHandle = null;
        this.#commit();
      });
    }, this.#flushIntervalMs);
  }

  #commit() {
    if (this.#disposed || this.#isInteractionActive() || !this.#hasLatest) return;
    const value = this.#latest as T;
    this.#hasLatest = false;
    this.#latest = undefined;
    this.#onCommit(value);
  }

  #clearScheduledFlush() {
    if (this.#flushTimer !== null) {
      clearTimeout(this.#flushTimer);
      this.#flushTimer = null;
    }
    if (this.#frameHandle !== null) {
      this.#cancelFrame(this.#frameHandle);
      this.#frameHandle = null;
    }
  }

  #clearInteractionTimer() {
    if (this.#interactionTimer === null) return;
    clearTimeout(this.#interactionTimer);
    this.#interactionTimer = null;
  }
}

/**
 * 调度一个帧回调，并在当前环境不支持动画帧时使用定时器作为回退方案。
 *
 * @param callback - 帧调度时调用的回调函数
 * @returns 可用于取消已调度回调的句柄
 */
function defaultRequestFrame(callback: FrameRequestCallback) {
  if (typeof globalThis.requestAnimationFrame === 'function') {
    return globalThis.requestAnimationFrame(callback);
  }
  return globalThis.setTimeout(() => callback(Date.now()), 0) as unknown as number;
}

/**
 * 取消通过默认帧调度器安排的回调。
 *
 * @param handle - 待取消的帧或定时器句柄
 */
function defaultCancelFrame(handle: number) {
  if (typeof globalThis.cancelAnimationFrame === 'function') {
    globalThis.cancelAnimationFrame(handle);
    return;
  }
  globalThis.clearTimeout(handle);
}
