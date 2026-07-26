import { normalizeStructuredLogEntry, type StructuredLogEntry } from '@/shared/observability';

import type { TaskLogEntry, TaskLogResponse } from '../types/task';

const TASK_LOG_VIEW_CAPACITY = 10_000;

export type TaskLogRealtimeSnapshot = Readonly<{
  contentVersion: number;
  entries: readonly StructuredLogEntry[];
  oldestSequence: number;
  nextAfterSequence: number;
  truncated: boolean;
}>;

type TaskLogRealtimeBatcherOptions = Readonly<{
  capacity?: number;
  onCommit: (snapshot: TaskLogRealtimeSnapshot) => void;
}>;

/**
 * 按持久化 sequence 合并任务日志分页，让向前读取历史和向后追加实时日志共用一个有界视图。
 */
export class TaskLogRealtimeBatcher {
  readonly #capacity: number;
  readonly #onCommit: (snapshot: TaskLogRealtimeSnapshot) => void;
  #entries = new Map<number, StructuredLogEntry>();
  #contentVersion = 0;
  #nextAfterSequence = 0;
  #truncated = false;

  constructor(options: TaskLogRealtimeBatcherOptions) {
    this.#capacity = options.capacity ?? TASK_LOG_VIEW_CAPACITY;
    this.#onCommit = options.onCommit;
  }

  seed(response: TaskLogResponse) {
    this.#entries.clear();
    this.#contentVersion = 0;
    this.#nextAfterSequence = 0;
    this.#truncated = false;
    this.#append(response.items);
    this.#nextAfterSequence = response.next_after_sequence;
    this.#emit();
  }

  append(response: TaskLogResponse) {
    const entryVersion = this.#entries.size;
    const currentCursor = this.#nextAfterSequence;
    this.#append(response.items);
    this.#nextAfterSequence = Math.max(currentCursor, response.next_after_sequence);

    // 空轮询且游标未推进时不创建新的响应式快照，避免日志视图无意义重渲染。
    if (this.#entries.size === entryVersion && this.#nextAfterSequence === currentCursor) {
      return;
    }

    this.#emit();
  }

  prepend(response: TaskLogResponse) {
    const entryVersion = this.#entries.size;
    this.#append(response.items);
    if (this.#entries.size === entryVersion) return;
    this.#emit();
  }

  nextAfterSequence() {
    return this.#nextAfterSequence;
  }

  oldestSequence() {
    if (!this.#entries.size) return 0;
    return Math.min(...this.#entries.keys());
  }

  clear(resetCursor = true) {
    this.#entries.clear();
    this.#contentVersion += 1;
    if (resetCursor) this.#nextAfterSequence = 0;
    this.#truncated = false;
    this.#emit();
  }

  #append(entries: readonly TaskLogEntry[]) {
    for (const entry of entries) {
      const normalized = normalizeStructuredLogEntry(entry);
      if (!normalized) continue;
      if (!this.#entries.has(entry.sequence)) {
        this.#entries.set(entry.sequence, normalized);
        this.#contentVersion += 1;
      }
    }
    this.#trim();
  }

  #emit() {
    const entries = [...this.#entries.entries()].sort(([left], [right]) => left - right);
    this.#onCommit({
      contentVersion: this.#contentVersion,
      entries: entries.map(([, entry]) => entry),
      oldestSequence: entries[0]?.[0] ?? 0,
      nextAfterSequence: this.#nextAfterSequence,
      truncated: this.#truncated,
    });
  }

  #trim() {
    while (this.#entries.size > this.#capacity) {
      const oldestSequence = Math.min(...this.#entries.keys());
      this.#entries.delete(oldestSequence);
      this.#contentVersion += 1;
      this.#truncated = true;
    }
  }
}
