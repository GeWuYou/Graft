import { normalizeStructuredLogEntry, type StructuredLogEntry } from '@/shared/observability';

import type { TaskLogEntry, TaskLogResponse } from '../types/task';

const TASK_LOG_COMMIT_INTERVAL_MS = 100;
const TASK_LOG_NORMALIZATION_CHUNK_SIZE = 25;

export type TaskLogRealtimeSnapshot = Readonly<{
  contentVersion: number;
  entries: readonly StructuredLogEntry[];
  nextAfterSequence: number;
  truncated: boolean;
}>;

type TaskLogRealtimeBatcherOptions = Readonly<{
  capacity?: number;
  commitIntervalMs?: number;
  onCommit: (snapshot: TaskLogRealtimeSnapshot) => void;
}>;

/**
 * 通过分片合并持久化回放页；默认保留全部已加载日志，由调用方按序游标决定何时继续加载。
 */
export class TaskLogRealtimeBatcher {
  readonly #capacity: number | null;
  readonly #commitIntervalMs: number;
  readonly #onCommit: (snapshot: TaskLogRealtimeSnapshot) => void;
  #entries: StructuredLogEntry[] = [];
  #contentVersion = 0;
  #nextAfterSequence = 0;
  #truncated = false;
  #commitTimer: ReturnType<typeof setTimeout> | null = null;
  #generation = 0;

  constructor(options: TaskLogRealtimeBatcherOptions) {
    this.#capacity = options.capacity ?? null;
    this.#commitIntervalMs = options.commitIntervalMs ?? TASK_LOG_COMMIT_INTERVAL_MS;
    this.#onCommit = options.onCommit;
  }

  seed(response: TaskLogResponse) {
    this.#generation += 1;
    this.#entries = [];
    this.#contentVersion = 0;
    this.#nextAfterSequence = 0;
    this.#truncated = false;
    this.#append(response.items);
    this.#nextAfterSequence = response.next_after_sequence;
    this.#emitImmediately();
  }

  append(response: TaskLogResponse) {
    const entryVersion = this.#contentVersion;
    const currentCursor = this.#nextAfterSequence;
    this.#append(response.items);
    this.#nextAfterSequence = Math.max(currentCursor, response.next_after_sequence);

    // 空轮询且游标未推进时不创建新的响应式快照，避免日志视图无意义重渲染。
    if (this.#contentVersion === entryVersion && this.#nextAfterSequence === currentCursor) {
      return;
    }

    this.#scheduleEmit();
  }

  /**
   * 分片处理首屏日志，避免大任务的历史回放阻塞抽屉打开或关闭动画。
   */
  async seedDeferred(response: TaskLogResponse) {
    const generation = ++this.#generation;
    this.#entries = [];
    this.#contentVersion = 0;
    this.#nextAfterSequence = 0;
    this.#truncated = false;
    await this.#appendDeferred(response.items, generation);
    if (generation !== this.#generation) return false;
    this.#nextAfterSequence = response.next_after_sequence;
    this.#emitImmediately();
    return true;
  }

  /**
   * 分片合并实时增量，确保高频日志不会长时间占用主线程。
   */
  async appendDeferred(response: TaskLogResponse) {
    const generation = this.#generation;
    const entryVersion = this.#contentVersion;
    const currentCursor = this.#nextAfterSequence;
    await this.#appendDeferred(response.items, generation);
    if (generation !== this.#generation) return false;
    this.#nextAfterSequence = Math.max(currentCursor, response.next_after_sequence);

    if (this.#contentVersion === entryVersion && this.#nextAfterSequence === currentCursor) {
      return true;
    }

    this.#scheduleEmit();
    return true;
  }

  nextAfterSequence() {
    return this.#nextAfterSequence;
  }

  clear() {
    this.#generation += 1;
    this.#clearCommitTimer();
    this.#entries = [];
    this.#contentVersion += 1;
    this.#nextAfterSequence = 0;
    this.#truncated = false;
  }

  destroy() {
    this.clear();
  }

  #append(entries: readonly TaskLogEntry[]) {
    for (const entry of entries) {
      const normalized = normalizeStructuredLogEntry(entry);
      if (!normalized) continue;
      this.#appendNormalized(normalized);
    }
  }

  async #appendDeferred(entries: readonly TaskLogEntry[], generation: number) {
    for (let index = 0; index < entries.length; index += 1) {
      if (generation !== this.#generation) return;
      const normalized = normalizeStructuredLogEntry(entries[index]);
      if (normalized) this.#appendNormalized(normalized);
      if ((index + 1) % TASK_LOG_NORMALIZATION_CHUNK_SIZE === 0) await yieldToBrowser();
    }
  }

  #scheduleEmit() {
    if (this.#commitTimer !== null) return;
    this.#commitTimer = setTimeout(() => {
      this.#commitTimer = null;
      this.#emitImmediately();
    }, this.#commitIntervalMs);
  }

  #clearCommitTimer() {
    if (this.#commitTimer === null) return;
    clearTimeout(this.#commitTimer);
    this.#commitTimer = null;
  }

  #emitImmediately() {
    this.#clearCommitTimer();
    this.#onCommit({
      contentVersion: this.#contentVersion,
      entries: this.#entries.slice(),
      nextAfterSequence: this.#nextAfterSequence,
      truncated: this.#truncated,
    });
  }

  #appendNormalized(entry: StructuredLogEntry) {
    this.#entries.push(entry);
    this.#contentVersion += 1;
    if (this.#capacity === null || this.#entries.length <= this.#capacity) return;
    this.#entries.splice(0, this.#entries.length - this.#capacity);
    this.#truncated = true;
  }
}

function yieldToBrowser() {
  return new Promise<void>((resolve) => setTimeout(resolve, 0));
}
