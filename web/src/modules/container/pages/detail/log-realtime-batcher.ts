import {
  cloneLogRingBufferView,
  LogBatchBuffer,
  LogRingBuffer,
  type LogRingBufferView,
  normalizeStructuredLogEntry,
} from '@/shared/observability';

import type { ContainerLogEntry, ContainerLogResponse } from '../../types/container';

const DEFAULT_LOG_BATCH_FLUSH_INTERVAL_MS = 100;
const DEFAULT_LOG_BATCH_MAX_SIZE = 32;

type ContainerLogRealtimeBatcherOptions = Readonly<{
  lineLimit: number;
  flushIntervalMs?: number;
  maxBatchSize?: number;
  onCommit: (snapshot: ContainerLogRealtimeBatcherSnapshot) => void;
}>;

type ContainerLogBase = Omit<ContainerLogResponse, 'entries' | 'tail' | 'truncated'>;

export type ContainerLogRealtimeBatcherSnapshot = Readonly<
  ContainerLogBase & {
    entryView: LogRingBufferView<ContainerLogEntry>;
    tail: number;
    truncated: boolean;
    version: number;
  }
>;

/**
 * 过滤并保留有效的日志行。
 *
 * @param lines - 待处理的行数组
 * @returns 仅包含非空字符串的数组
 */
function normalizeEntries(entries: readonly ContainerLogEntry[]) {
  return entries
    .map((entry) => normalizeStructuredLogEntry(entry))
    .filter((entry): entry is NonNullable<ReturnType<typeof normalizeStructuredLogEntry>> => entry !== null)
    .map((entry) => ({
      line: entry.line,
      occurred_at: entry.occurredAt,
      stream: entry.stream,
    }));
}

export class ContainerLogRealtimeBatcher {
  readonly #flushIntervalMs: number;
  readonly #maxBatchSize: number;
  readonly #onCommit: (snapshot: ContainerLogRealtimeBatcherSnapshot) => void;
  #lineLimit: number;
  #base: ContainerLogBase | null = null;
  #truncated = false;
  #lineBuffer: LogRingBuffer<ContainerLogEntry>;
  #batchBuffer: LogBatchBuffer<ContainerLogEntry>;

  constructor(options: ContainerLogRealtimeBatcherOptions) {
    this.#lineLimit = options.lineLimit;
    this.#flushIntervalMs = options.flushIntervalMs ?? DEFAULT_LOG_BATCH_FLUSH_INTERVAL_MS;
    this.#maxBatchSize = options.maxBatchSize ?? DEFAULT_LOG_BATCH_MAX_SIZE;
    this.#onCommit = options.onCommit;
    this.#lineBuffer = new LogRingBuffer<ContainerLogEntry>(this.#lineLimit);
    this.#batchBuffer = this.#createBatchBuffer();
  }

  seed(nextLogs: ContainerLogResponse) {
    this.#batchBuffer.clear();
    this.#lineBuffer = new LogRingBuffer<ContainerLogEntry>(this.#lineLimit);
    this.#base = {
      id: nextLogs.id,
      runtime: nextLogs.runtime,
      stderr: nextLogs.stderr,
      stdout: nextLogs.stdout,
      timestamps: nextLogs.timestamps,
    };
    this.#truncated = Boolean(nextLogs.truncated);
    this.#appendDirect(normalizeEntries(nextLogs.entries));
    this.#emit();
  }

  enqueue(entries: readonly ContainerLogEntry[]) {
    const nextEntries = normalizeEntries(entries);
    if (!nextEntries.length) {
      return;
    }

    this.#batchBuffer.appendMany(nextEntries);
  }

  flush() {
    this.#batchBuffer.flush();
  }

  clearView() {
    this.#batchBuffer.clear();
    this.#lineBuffer.clear();
    this.#truncated = false;
    this.#emit();
  }

  clear() {
    this.#batchBuffer.clear();
    this.#lineBuffer.clear();
    this.#base = null;
    this.#truncated = false;
  }

  destroy() {
    this.#batchBuffer.destroy();
    this.#lineBuffer.clear();
    this.#base = null;
    this.#truncated = false;
  }

  updateLineLimit(lineLimit: number) {
    this.#lineLimit = lineLimit;
    this.clear();
    this.#batchBuffer.destroy();
    this.#batchBuffer = this.#createBatchBuffer();
    this.#lineBuffer = new LogRingBuffer<ContainerLogEntry>(this.#lineLimit);
  }

  #createBatchBuffer() {
    return new LogBatchBuffer<ContainerLogEntry>({
      flushIntervalMs: this.#flushIntervalMs,
      maxBatchSize: this.#maxBatchSize,
      onFlush: (batch) => {
        if (!this.#base || batch.length === 0) {
          return;
        }

        this.#appendDirect(batch);
        this.#emit();
      },
    });
  }

  #appendDirect(entries: readonly ContainerLogEntry[]) {
    for (const entry of entries) {
      const result = this.#lineBuffer.append(entry);
      if (result.overwritten !== undefined) {
        this.#truncated = true;
      }
    }
  }

  #emit() {
    if (!this.#base) {
      return;
    }

    const entryView = cloneLogRingBufferView(this.#lineBuffer.snapshot());

    this.#onCommit(
      Object.freeze({
        ...this.#base,
        entryView,
        tail: this.#lineLimit,
        truncated: this.#truncated,
        version: entryView.version,
      }),
    );
  }
}
