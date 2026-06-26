import { describe, expect, it, vi } from 'vitest';

import type { ContainerLogResponse } from '../../types/container';
import { ContainerLogRealtimeBatcher } from './log-realtime-batcher';

function createSeed(overrides?: Partial<ContainerLogResponse>): ContainerLogResponse {
  return {
    id: 'container-1',
    lines: ['seed-1', 'seed-2'],
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
          lines: snapshot.lineView.toArray(),
          truncated: snapshot.truncated,
          version: snapshot.version,
        });
      },
    });

    batcher.seed(createSeed());
    batcher.enqueue(['line-3']);
    batcher.enqueue(['line-4']);

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
          lines: snapshot.lineView.toArray(),
          truncated: snapshot.truncated,
        });
      },
    });

    batcher.seed(createSeed({ lines: ['seed-1'] }));
    batcher.enqueue(['', 'line-2', 'line-3']);
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
          lines: snapshot.lineView.toArray(),
          truncated: snapshot.truncated,
        });
      },
    });

    batcher.seed(createSeed({ lines: ['seed-1'] }));
    batcher.enqueue(['line-2']);
    batcher.clear();
    vi.runOnlyPendingTimers();

    expect(commits).toHaveLength(1);

    batcher.seed(createSeed({ id: 'container-2', lines: ['api-started'], tail: 2, truncated: true }));

    expect(commits).toHaveLength(2);
    expect(commits[1]?.id).toBe('container-2');
    expect(commits[1]?.lines).toEqual(['api-started']);
    expect(commits[1]?.truncated).toBe(true);

    vi.useRealTimers();
  });

  it('emits an immutable snapshot for paused consumers', () => {
    const snapshots: Array<{ lines: readonly string[]; version: number }> = [];
    const batcher = new ContainerLogRealtimeBatcher({
      lineLimit: 4,
      onCommit: (snapshot) => {
        snapshots.push({
          lines: snapshot.lineView.toArray(),
          version: snapshot.version,
        });
      },
    });

    batcher.seed(createSeed({ lines: ['seed-1'] }));
    batcher.enqueue(['line-2']);
    batcher.flush();

    expect(snapshots[0]?.lines).toEqual(['seed-1']);
    expect(snapshots[1]?.lines).toEqual(['seed-1', 'line-2']);
    expect(snapshots[0]?.version).toBe(1);
  });
});
