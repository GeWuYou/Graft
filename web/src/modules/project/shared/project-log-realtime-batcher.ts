import { LogBatchBuffer } from '@/shared/observability';

import type { ApplicationLogEntry, ApplicationLogResponse } from '../types/project';
import { emitApplicationLogDebug } from './project-log-debug';

const DEFAULT_LOG_BATCH_FLUSH_INTERVAL_MS = 100;
const DEFAULT_LOG_BATCH_MAX_SIZE = 32;

type ApplicationLogRealtimeBatcherOptions = Readonly<{
  lineLimit: number;
  flushIntervalMs?: number;
  maxBatchSize?: number;
  onCommit: (snapshot: ApplicationLogResponse) => void;
}>;

/**
 * ApplicationLogRealtimeBatcher 在限制日志行数的同时，将高频实时日志合并为稳定的时间序列快照。
 *
 * HTTP 快照加载期间到达的实时日志会先暂存，待快照作为基线写入后再合并，避免刷新覆盖已经收到的日志。
 */
export class ApplicationLogRealtimeBatcher {
  readonly #onCommit: (snapshot: ApplicationLogResponse) => void;
  #base: Omit<ApplicationLogResponse, 'entries' | 'tail' | 'truncated'> | null = null;
  #entries = new Map<string, ApplicationLogEntry>();
  #lineLimit: number;
  #pendingSnapshotEntries = new Map<string, ApplicationLogEntry>();
  #pendingSnapshotTruncated = false;
  #snapshotPending = false;
  #truncated = false;
  #batchBuffer: LogBatchBuffer<ApplicationLogEntry>;

  constructor(options: ApplicationLogRealtimeBatcherOptions) {
    this.#lineLimit = options.lineLimit;
    this.#onCommit = options.onCommit;
    this.#batchBuffer = new LogBatchBuffer<ApplicationLogEntry>({
      flushIntervalMs: options.flushIntervalMs ?? DEFAULT_LOG_BATCH_FLUSH_INTERVAL_MS,
      maxBatchSize: options.maxBatchSize ?? DEFAULT_LOG_BATCH_MAX_SIZE,
      onFlush: (entries) => this.#flushEntries(entries),
    });
  }

  beginSnapshot(lineLimit: number) {
    this.#lineLimit = normalizeLineLimit(lineLimit);
    this.#snapshotPending = true;
    emitApplicationLogDebug('batcher-snapshot-begin', {
      lineLimit: this.#lineLimit,
      retainedCount: this.#entries.size,
      bufferedCount: this.#pendingSnapshotEntries.size,
    });
    this.#batchBuffer.flush();
  }

  seed(snapshot: ApplicationLogResponse) {
    this.#lineLimit = normalizeLineLimit(snapshot.tail);
    this.#base = {
      compose_project_name: snapshot.compose_project_name,
      application_id: snapshot.application_id,
      stderr: snapshot.stderr,
      stdout: snapshot.stdout,
      timestamps: snapshot.timestamps,
      ...(snapshot.since === undefined ? {} : { since: snapshot.since }),
    };
    this.#entries.clear();
    this.#truncated = Boolean(snapshot.truncated) || this.#pendingSnapshotTruncated;
    this.#merge(snapshot.entries);
    this.#merge(this.#pendingSnapshotEntries.values());
    const bufferedCount = this.#pendingSnapshotEntries.size;
    this.#pendingSnapshotEntries.clear();
    this.#pendingSnapshotTruncated = false;
    this.#snapshotPending = false;
    emitApplicationLogDebug('batcher-snapshot-seeded', {
      responseCount: snapshot.entries.length,
      retainedCount: this.#entries.size,
      bufferedCount,
      responseTruncated: Boolean(snapshot.truncated),
      truncated: this.#truncated,
      lineLimit: this.#lineLimit,
    });
    this.#emit();
  }

  enqueue(entry: ApplicationLogEntry) {
    this.#batchBuffer.append(entry);
  }

  flush() {
    this.#batchBuffer.flush();
  }

  clearView() {
    this.#batchBuffer.clear();
    this.#entries.clear();
    this.#pendingSnapshotEntries.clear();
    this.#pendingSnapshotTruncated = false;
    this.#truncated = false;
    this.#emit();
  }

  clear() {
    this.#batchBuffer.clear();
    this.#base = null;
    this.#entries.clear();
    this.#pendingSnapshotEntries.clear();
    this.#pendingSnapshotTruncated = false;
    this.#snapshotPending = false;
    this.#truncated = false;
  }

  destroy() {
    this.#batchBuffer.destroy();
    this.clear();
  }

  #flushEntries(entries: readonly ApplicationLogEntry[]) {
    // 快照建立前不能提交局部结果，否则 HTTP 响应会覆盖这段时间内已经到达的实时日志。
    if (this.#snapshotPending) {
      const trimmed = this.#mergeInto(this.#pendingSnapshotEntries, entries);
      this.#pendingSnapshotTruncated ||= trimmed;
      emitApplicationLogDebug('batcher-buffered-before-snapshot', {
        flushedCount: entries.length,
        bufferedCount: this.#pendingSnapshotEntries.size,
        truncated: this.#pendingSnapshotTruncated,
      });
      return;
    }
    if (!this.#base) {
      const trimmed = this.#mergeInto(this.#pendingSnapshotEntries, entries);
      this.#pendingSnapshotTruncated ||= trimmed;
      emitApplicationLogDebug('batcher-buffered-without-base', {
        flushedCount: entries.length,
        bufferedCount: this.#pendingSnapshotEntries.size,
        truncated: this.#pendingSnapshotTruncated,
      });
      return;
    }
    const previousCount = this.#entries.size;
    this.#merge(entries);
    emitApplicationLogDebug('batcher-flushed', {
      flushedCount: entries.length,
      retainedCount: this.#entries.size,
      deduplicatedCount: Math.max(0, previousCount + entries.length - this.#entries.size),
      truncated: this.#truncated,
      lineLimit: this.#lineLimit,
    });
    this.#emit();
  }

  #merge(entries: Iterable<ApplicationLogEntry>) {
    const trimmed = this.#mergeInto(this.#entries, entries);
    this.#truncated ||= trimmed;
  }

  #mergeInto(target: Map<string, ApplicationLogEntry>, entries: Iterable<ApplicationLogEntry>) {
    for (const entry of entries) {
      target.set(projectLogEntryKey(entry), entry);
    }
    const ordered = orderApplicationLogEntries(target.values());
    if (ordered.length > this.#lineLimit) {
      target.clear();
      for (const entry of ordered.slice(-this.#lineLimit)) {
        target.set(projectLogEntryKey(entry), entry);
      }
      return true;
    }
    return false;
  }

  #emit() {
    if (!this.#base) {
      return;
    }
    emitApplicationLogDebug('batcher-committed', {
      retainedCount: this.#entries.size,
      truncated: this.#truncated,
      lineLimit: this.#lineLimit,
    });
    this.#onCommit({
      ...this.#base,
      entries: orderApplicationLogEntries(this.#entries.values()),
      tail: this.#lineLimit,
      truncated: this.#truncated,
    });
  }
}

/**
 * 将行数限制规范化为有效的正整数。
 *
 * @param value - 待规范化的行数限制
 * @returns 有效的正整数限制；输入无效时返回 `200`
 */
function normalizeLineLimit(value: number) {
  return Number.isInteger(value) && value > 0 ? value : 200;
}

/**
 * 生成项目日志条目的稳定去重键。
 *
 * @param entry - 要生成键的项目日志条目
 * @returns 由容器、服务、流、发生时间和日志内容组成的去重键
 */
function projectLogEntryKey(entry: ApplicationLogEntry) {
  return [entry.container_id, entry.service_name, entry.stream, entry.occurred_at, entry.line].join('::');
}

/**
 * 按发生时间及日志属性对项目日志条目进行稳定排序。
 *
 * @param entries - 待排序的项目日志条目
 * @returns 按发生时间、服务名、容器名、流和日志行依次排序的条目数组
 */
function orderApplicationLogEntries(entries: Iterable<ApplicationLogEntry>) {
  return [...entries].sort((left, right) => {
    const timeDiff = compareText(left.occurred_at, right.occurred_at);
    if (timeDiff !== 0) return timeDiff;
    const serviceDiff = compareText(left.service_name, right.service_name);
    if (serviceDiff !== 0) return serviceDiff;
    const containerDiff = compareText(left.container_name, right.container_name);
    if (containerDiff !== 0) return containerDiff;
    const streamDiff = compareText(left.stream, right.stream);
    if (streamDiff !== 0) return streamDiff;
    return compareText(left.line, right.line);
  });
}

/**
 * 比较两个文本值的字典序。
 *
 * @param left - 第一个文本值
 * @param right - 第二个文本值
 * @returns `left` 小于 `right` 时为 `-1`，大于时为 `1`，相等时为 `0`
 */
function compareText(left: string, right: string) {
  return left < right ? -1 : left > right ? 1 : 0;
}
