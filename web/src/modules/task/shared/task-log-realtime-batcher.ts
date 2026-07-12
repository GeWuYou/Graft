import { LogRingBuffer, normalizeStructuredLogEntry, type StructuredLogEntry } from '@/shared/observability';

import type { TaskLogEntry, TaskLogResponse } from '../types/task';

const TASK_LOG_VIEW_CAPACITY = 1000;

export type TaskLogRealtimeSnapshot = Readonly<{
  contentVersion: number;
  entries: readonly StructuredLogEntry[];
  nextAfterSequence: number;
  truncated: boolean;
}>;

type TaskLogRealtimeBatcherOptions = Readonly<{
  capacity?: number;
  onCommit: (snapshot: TaskLogRealtimeSnapshot) => void;
}>;

/**
 * Keeps the task log view bounded and commits only after one durable replay page.
 */
export class TaskLogRealtimeBatcher {
  readonly #capacity: number;
  readonly #onCommit: (snapshot: TaskLogRealtimeSnapshot) => void;
  #entries: LogRingBuffer<StructuredLogEntry>;
  #nextAfterSequence = 0;
  #truncated = false;

  constructor(options: TaskLogRealtimeBatcherOptions) {
    this.#capacity = options.capacity ?? TASK_LOG_VIEW_CAPACITY;
    this.#entries = new LogRingBuffer<StructuredLogEntry>(this.#capacity);
    this.#onCommit = options.onCommit;
  }

  seed(response: TaskLogResponse) {
    this.#entries = new LogRingBuffer<StructuredLogEntry>(this.#capacity);
    this.#nextAfterSequence = 0;
    this.#truncated = false;
    this.#append(response.items);
    this.#nextAfterSequence = response.next_after_sequence;
    this.#emit();
  }

  append(response: TaskLogResponse) {
    this.#append(response.items);
    this.#nextAfterSequence = Math.max(this.#nextAfterSequence, response.next_after_sequence);
    this.#emit();
  }

  nextAfterSequence() {
    return this.#nextAfterSequence;
  }

  clear() {
    this.#entries.clear();
    this.#nextAfterSequence = 0;
    this.#truncated = false;
  }

  #append(entries: readonly TaskLogEntry[]) {
    for (const entry of entries) {
      const normalized = normalizeStructuredLogEntry(entry);
      if (!normalized) continue;
      if (this.#entries.append(normalized).overwritten !== undefined) this.#truncated = true;
    }
  }

  #emit() {
    const entryView = this.#entries.snapshot();
    this.#onCommit({
      contentVersion: entryView.version,
      entries: entryView.toArray(),
      nextAfterSequence: this.#nextAfterSequence,
      truncated: this.#truncated,
    });
  }
}
