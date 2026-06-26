const DEFAULT_FLUSH_INTERVAL_MS = 100;
const DEFAULT_MAX_BATCH_SIZE = 32;

/**
 * 验证值是否为正整数。
 *
 * @param value - 要检查的数值
 * @param label - 用于错误消息的字段名称
 * @throws {RangeError} 当值不是正整数时抛出
 */
function assertPositiveInteger(value: number, label: string) {
  if (!Number.isInteger(value) || value <= 0) {
    throw new RangeError(`${label} must be a positive integer`);
  }
}

export type LogBatchBufferOptions<T> = Readonly<{
  flushIntervalMs?: number;
  maxBatchSize?: number;
  onFlush: (batch: readonly T[]) => void;
}>;

/**
 * 在非响应式队列中收集高频日志，按时间窗或计数阈值批量下沉。
 */
export class LogBatchBuffer<T> {
  readonly #flushIntervalMs: number;
  readonly #maxBatchSize: number;
  readonly #onFlush: (batch: readonly T[]) => void;
  #pending: T[] = [];
  #flushTimer: ReturnType<typeof setTimeout> | null = null;
  #destroyed = false;

  constructor(options: LogBatchBufferOptions<T>) {
    assertPositiveInteger(options.flushIntervalMs ?? DEFAULT_FLUSH_INTERVAL_MS, 'LogBatchBuffer flushIntervalMs');
    assertPositiveInteger(options.maxBatchSize ?? DEFAULT_MAX_BATCH_SIZE, 'LogBatchBuffer maxBatchSize');
    this.#flushIntervalMs = options.flushIntervalMs ?? DEFAULT_FLUSH_INTERVAL_MS;
    this.#maxBatchSize = options.maxBatchSize ?? DEFAULT_MAX_BATCH_SIZE;
    this.#onFlush = options.onFlush;
  }

  append(item: T) {
    this.appendMany([item]);
  }

  appendMany(items: readonly T[]) {
    if (this.#destroyed || items.length === 0) {
      return this.#pending.length;
    }

    for (const item of items) {
      this.#pending.push(item);
      if (this.#pending.length >= this.#maxBatchSize) {
        this.flush();
      }
    }

    if (this.#pending.length > 0) {
      this.#scheduleFlush();
    }

    return this.#pending.length;
  }

  flush() {
    if (this.#destroyed) {
      return [] as readonly T[];
    }

    this.#clearFlushTimer();
    if (this.#pending.length === 0) {
      return [] as readonly T[];
    }

    const batch = this.#pending;
    this.#pending = [];
    this.#onFlush(batch);
    return batch;
  }

  clear() {
    this.#clearFlushTimer();
    this.#pending = [];
  }

  destroy() {
    if (this.#destroyed) {
      return;
    }

    this.clear();
    this.#destroyed = true;
  }

  pendingSize() {
    return this.#pending.length;
  }

  #scheduleFlush() {
    if (this.#flushTimer !== null) {
      return;
    }

    this.#flushTimer = setTimeout(() => {
      this.#flushTimer = null;
      this.flush();
    }, this.#flushIntervalMs);
  }

  #clearFlushTimer() {
    if (this.#flushTimer === null) {
      return;
    }

    clearTimeout(this.#flushTimer);
    this.#flushTimer = null;
  }
}
