const DEFAULT_REFRESH_INTERVAL_MS = 250;

type TaskRealtimeRefreshSchedulerOptions = Readonly<{
  intervalMs?: number;
  onRefresh: () => Promise<void> | void;
}>;

/**
 * 合并实时通知，在通知突发时避免产生并发的持久化读取。
 */
export class TaskRealtimeRefreshScheduler {
  readonly #intervalMs: number;
  readonly #onRefresh: () => Promise<void> | void;
  #disposed = false;
  #pending = false;
  #refreshing = false;
  #timer: ReturnType<typeof setTimeout> | null = null;
  #inFlight: Promise<void> | null = null;

  constructor(options: TaskRealtimeRefreshSchedulerOptions) {
    this.#intervalMs = options.intervalMs ?? DEFAULT_REFRESH_INTERVAL_MS;
    this.#onRefresh = options.onRefresh;
  }

  request(immediate = false) {
    if (this.#disposed) return Promise.resolve();
    this.#pending = true;
    if (immediate) {
      this.#clearTimer();
      return this.#run();
    }
    this.#schedule();
    return this.#inFlight ?? Promise.resolve();
  }

  cancel() {
    this.#pending = false;
    this.#clearTimer();
  }

  destroy() {
    this.cancel();
    this.#disposed = true;
  }

  #schedule() {
    if (this.#timer || this.#refreshing || this.#disposed) return;
    this.#timer = setTimeout(() => {
      this.#timer = null;
      void this.#run();
    }, this.#intervalMs);
  }

  #run() {
    if (this.#disposed || !this.#pending) return this.#inFlight ?? Promise.resolve();
    if (this.#refreshing) return this.#inFlight ?? Promise.resolve();

    this.#pending = false;
    this.#refreshing = true;
    this.#inFlight = Promise.resolve(this.#onRefresh()).finally(() => {
      this.#refreshing = false;
      this.#inFlight = null;
      if (this.#pending) this.#schedule();
    });
    return this.#inFlight;
  }

  #clearTimer() {
    if (this.#timer === null) return;
    clearTimeout(this.#timer);
    this.#timer = null;
  }
}
