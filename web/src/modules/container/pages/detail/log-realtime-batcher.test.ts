import { describe, expect, it, vi } from 'vitest';

import type { ContainerLogResponse } from '../../types/container';
import { ContainerLogRealtimeBatcher } from './log-realtime-batcher';

function createEntry(line: string, stream: 'stdout' | 'stderr' = 'stdout', occurredAt = '2026-06-26T03:00:00Z') {
  return {
    line,
    occurred_at: occurredAt,
    stream,
  } as const;
}

function createSeed(overrides?: Partial<ContainerLogResponse>): ContainerLogResponse {
  return {
    id: 'container-1',
    entries: [createEntry('seed-1'), createEntry('seed-2', 'stderr', '2026-06-26T03:00:01Z')],
    runtime: 'docker',
    stderr: true,
    stdout: true,
    tail: 3,
    timestamps: false,
    truncated: false,
    ...overrides,
  };
}

describe('ContainerLogRealtimeBatcher', () => {
  it('seeds current history and emits batched realtime lines in a single commit', () => {
    vi.useFakeTimers();
    const commits: Array<{ lines: readonly string[]; truncated: boolean; version: number }> = [];
    const batcher = new ContainerLogRealtimeBatcher({
      lineLimit: 3,
      onCommit: (snapshot) => {
        commits.push({
          lines: snapshot.entryView.toArray().map((entry) => entry.line),
          truncated: snapshot.truncated,
          version: snapshot.version,
        });
      },
    });

    batcher.seed(createSeed());
    batcher.enqueue([createEntry('line-3')]);
    batcher.enqueue([createEntry('line-4', 'stderr', '2026-06-26T03:00:04Z')]);

    expect(commits).toHaveLength(1);
    vi.advanceTimersByTime(100);

    expect(commits).toHaveLength(2);
    expect(commits[0]?.lines).toEqual(['seed-1', 'seed-2']);
    expect(commits[1]?.lines).toEqual(['seed-2', 'line-3', 'line-4']);
    expect(commits[1]?.truncated).toBe(true);
    expect(commits[1]?.version).toBeGreaterThan(commits[0]?.version ?? 0);

    vi.useRealTimers();
  });

  it('supports manual flush and ignores empty or invalid lines', () => {
    const commits: Array<{ lines: readonly string[]; truncated: boolean }> = [];
    const batcher = new ContainerLogRealtimeBatcher({
      lineLimit: 4,
      onCommit: (snapshot) => {
        commits.push({
          lines: snapshot.entryView.toArray().map((entry) => entry.line),
          truncated: snapshot.truncated,
        });
      },
    });

    batcher.seed(createSeed({ entries: [createEntry('seed-1')] }));
    batcher.enqueue([createEntry('', 'stdout'), createEntry('line-2'), createEntry('line-3')]);
    batcher.flush();

    expect(commits).toHaveLength(2);
    expect(commits[1]?.lines).toEqual(['seed-1', 'line-2', 'line-3']);
    expect(commits[1]?.truncated).toBe(false);
  });

  it('drops pending lines on clear and rebuilds after a fresh seed', () => {
    vi.useFakeTimers();
    const commits: Array<{ id: string; lines: readonly string[]; truncated: boolean }> = [];
    const batcher = new ContainerLogRealtimeBatcher({
      lineLimit: 2,
      onCommit: (snapshot) => {
        commits.push({
          id: snapshot.id,
          lines: snapshot.entryView.toArray().map((entry) => entry.line),
          truncated: snapshot.truncated,
        });
      },
    });

    batcher.seed(createSeed({ entries: [createEntry('seed-1')] }));
    batcher.enqueue([createEntry('line-2')]);
    batcher.clear();
    vi.runOnlyPendingTimers();

    expect(commits).toHaveLength(1);

    batcher.seed(createSeed({ id: 'container-2', entries: [createEntry('api-started')], tail: 2, truncated: true }));

    expect(commits).toHaveLength(2);
    expect(commits[1]?.id).toBe('container-2');
    expect(commits[1]?.lines).toEqual(['api-started']);
    expect(commits[1]?.truncated).toBe(true);

    vi.useRealTimers();
  });

  it('emits an immutable snapshot for paused consumers', () => {
    const snapshots: Array<{
      entryView: { readonly version: number; toArray(): readonly { line: string }[] };
      version: number;
    }> = [];
    const batcher = new ContainerLogRealtimeBatcher({
      lineLimit: 4,
      onCommit: (snapshot) => {
        snapshots.push({
          entryView: snapshot.entryView,
          version: snapshot.version,
        });
      },
    });

    batcher.seed(createSeed({ entries: [createEntry('seed-1')] }));
    batcher.enqueue([createEntry('line-2')]);
    batcher.flush();

    expect(snapshots[0]?.entryView.toArray().map((entry) => entry.line)).toEqual(['seed-1']);
    expect(snapshots[1]?.entryView.toArray().map((entry) => entry.line)).toEqual(['seed-1', 'line-2']);
    expect(snapshots[0]?.entryView.version).toBe(1);
    expect(snapshots[0]?.version).toBe(1);
  });
});
