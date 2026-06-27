import { describe, expect, it } from 'vitest';

import { LogRingBuffer, type StructuredLogEntry } from '@/shared/observability';

import type { ContainerLogRealtimeBatcherSnapshot } from './log-realtime-batcher';
import { createContainerDetailLogViewStore } from './log-view-store';

function createSnapshot(lines: readonly string[], truncated = false): ContainerLogRealtimeBatcherSnapshot {
  const buffer = new LogRingBuffer<StructuredLogEntry>(Math.max(lines.length, 1));
  for (const line of lines) {
    buffer.append({
      line,
      occurredAt: '2026-06-26T03:00:00Z',
      stream: 'stdout',
    });
  }
  const entryView = buffer.snapshot();

  return Object.freeze({
    id: 'container-1',
    runtime: 'docker',
    stderr: true,
    stdout: true,
    timestamps: false,
    entryView,
    tail: buffer.capacity(),
    truncated,
    version: entryView.version,
  });
}

describe('createContainerDetailLogViewStore', () => {
  it('derives immutable log responses from committed snapshots', () => {
    const store = createContainerDetailLogViewStore();

    expect(store.hasSnapshot.value).toBe(false);
    store.commit(createSnapshot(['seed-1', 'seed-2']));

    expect(store.hasSnapshot.value).toBe(true);
    expect(store.version.value).toBe(2);
    expect(store.entries.value.map((entry) => entry.line)).toEqual(['seed-1', 'seed-2']);
    expect(store.logs.value?.entries.map((entry) => entry.line)).toEqual(['seed-1', 'seed-2']);
    expect(store.logs.value?.entries.map((entry) => entry.occurred_at)).toEqual([
      '2026-06-26T03:00:00Z',
      '2026-06-26T03:00:00Z',
    ]);
    expect(store.truncated.value).toBe(false);
  });

  it('keeps loading and error outside the snapshot and clears them on reset', () => {
    const store = createContainerDetailLogViewStore();

    store.setLoading(true);
    store.setError('load failed');
    store.commit(createSnapshot(['line-1'], true));

    expect(store.state.value.loading).toBe(true);
    expect(store.state.value.error).toBe('');
    expect(store.truncated.value).toBe(true);

    store.reset();

    expect(store.hasSnapshot.value).toBe(false);
    expect(store.version.value).toBe(0);
    expect(store.logs.value).toBeNull();
    expect(store.entries.value).toEqual([]);
    expect(store.state.value.loading).toBe(false);
    expect(store.state.value.error).toBe('');
  });

  it('suppresses UI publication while paused and publishes the latest snapshot once on resume', () => {
    const store = createContainerDetailLogViewStore();

    store.commit(createSnapshot(['seed-1']));
    store.pause();
    store.commit(createSnapshot(['seed-1', 'line-2']));
    store.commit(createSnapshot(['seed-1', 'line-2', 'line-3']));

    expect(store.paused.value).toBe(true);
    expect(store.version.value).toBe(1);
    expect(store.entries.value.map((entry) => entry.line)).toEqual(['seed-1']);

    store.resume();

    expect(store.paused.value).toBe(false);
    expect(store.version.value).toBe(3);
    expect(store.entries.value.map((entry) => entry.line)).toEqual(['seed-1', 'line-2', 'line-3']);
  });

  it('keeps paused snapshots stable after newer commits arrive', () => {
    const store = createContainerDetailLogViewStore();

    const initial = createSnapshot(['seed-1']);
    store.commit(initial);
    store.pause();

    const pending = createSnapshot(['seed-1', 'line-2']);
    store.commit(pending);

    expect(store.entries.value.map((entry) => entry.line)).toEqual(['seed-1']);
    expect(initial.entryView.toArray().map((entry) => entry.line)).toEqual(['seed-1']);
    expect(pending.entryView.toArray().map((entry) => entry.line)).toEqual(['seed-1', 'line-2']);
  });

  it('applies an empty clear snapshot immediately while paused', () => {
    const store = createContainerDetailLogViewStore();

    store.commit(createSnapshot(['seed-1', 'seed-2']));
    store.pause();
    store.commit(createSnapshot([]));

    expect(store.paused.value).toBe(true);
    expect(store.entries.value).toEqual([]);
    expect(store.logs.value?.entries).toEqual([]);
  });
});
