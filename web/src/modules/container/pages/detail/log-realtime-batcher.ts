import { LogBatchBuffer, LogRingBuffer, type LogRingBufferView } from '@/shared/observability';

import type { ContainerLogResponse } from '../../types/container';

const DEFAULT_LOG_BATCH_FLUSH_INTERVAL_MS = 100;
const DEFAULT_LOG_BATCH_MAX_SIZE = 32;

type ContainerLogRealtimeBatcherOptions = Readonly<{
  lineLimit: number;
  flushIntervalMs?: number;
  maxBatchSize?: number;
  onCommit: (snapshot: ContainerLogRealtimeBatcherSnapshot) => void;
}>;

type ContainerLogBase = Omit<ContainerLogResponse, 'lines' | 'tail' | 'truncated'>;

export type ContainerLogRealtimeBatcherSnapshot = Readonly<
  ContainerLogBase & {
    lineView: LogRingBufferView<string>;
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
function normalizeLines(lines: readonly string[]) {
  return lines.filter((line) => typeof line === 'string' && line.length > 0);
}

export class ContainerLogRealtimeBatcher {
  readonly #flushIntervalMs: number;
  readonly #maxBatchSize: number;
  readonly #onCommit: (snapshot: ContainerLogRealtimeBatcherSnapshot) => void;
  #lineLimit: number;
  #base: ContainerLogBase | null = null;
  #truncated = false;
  #lineBuffer: LogRingBuffer<string>;
  #batchBuffer: LogBatchBuffer<string>;

  constructor(options: ContainerLogRealtimeBatcherOptions) {
    this.#lineLimit = options.lineLimit;
    this.#flushIntervalMs = options.flushIntervalMs ?? DEFAULT_LOG_BATCH_FLUSH_INTERVAL_MS;
    this.#maxBatchSize = options.maxBatchSize ?? DEFAULT_LOG_BATCH_MAX_SIZE;
    this.#onCommit = options.onCommit;
    this.#lineBuffer = new LogRingBuffer<string>(this.#lineLimit);
    this.#batchBuffer = this.#createBatchBuffer();
  }

  seed(nextLogs: ContainerLogResponse) {
    this.#batchBuffer.clear();
    this.#lineBuffer = new LogRingBuffer<string>(this.#lineLimit);
    this.#base = {
      id: nextLogs.id,
      runtime: nextLogs.runtime,
      stderr: nextLogs.stderr,
      stdout: nextLogs.stdout,
      timestamps: nextLogs.timestamps,
    };
    this.#truncated = Boolean(nextLogs.truncated);
    this.#appendDirect(normalizeLines(nextLogs.lines));
    this.#emit();
  }

  enqueue(lines: readonly string[]) {
    const nextLines = normalizeLines(lines);
    if (!nextLines.length) {
      return;
    }

    this.#batchBuffer.appendMany(nextLines);
  }

  flush() {
    this.#batchBuffer.flush();
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
    this.#lineBuffer = new LogRingBuffer<string>(this.#lineLimit);
  }

  #createBatchBuffer() {
    return new LogBatchBuffer<string>({
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

  #appendDirect(lines: readonly string[]) {
    for (const line of lines) {
      const result = this.#lineBuffer.append(line);
      if (result.overwritten !== undefined) {
        this.#truncated = true;
      }
    }
  }

  #emit() {
    if (!this.#base) {
      return;
    }

    const lineView = this.#lineBuffer.snapshot();

    this.#onCommit(
      Object.freeze({
        ...this.#base,
        lineView,
        tail: this.#lineLimit,
        truncated: this.#truncated,
        version: lineView.version,
      }),
    );
  }
}
